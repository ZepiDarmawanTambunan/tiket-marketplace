package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/crypto"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/midtrans"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
)

// ---------- Orders ----------

type OrderService struct {
	deps     *Deps
	qrSecret string
	promos   *PromoService
}

func NewOrderService(deps *Deps, qrSecret string, promos *PromoService) *OrderService {
	return &OrderService{deps: deps, qrSecret: qrSecret, promos: promos}
}

type CreateOrderResult struct {
	Order       *model.Order `json:"order"`
	SnapToken   string       `json:"snap_token"`
	ClientKey   string       `json:"client_key"`
	RedirectURL *string      `json:"redirect_url"`
	PaymentNote *string      `json:"payment_note,omitempty"`
}

// CreateOrder validates the checkout, computes commission + gateway fee,
// persists the order, and obtains a Midtrans Snap token (dummy fallback in dev).
// promoCode is optional; when set it is validated server-side and the discount
// cuts the subtotal BEFORE commission and gateway fee are computed.
func (s *OrderService) CreateOrder(userID, eventID int64, items []model.OrderItem, promoCode string) (*CreateOrderResult, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: items kosong", ErrUnprocessable)
	}
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if e.Status != model.EventPublished {
		return nil, fmt.Errorf("%w: event belum tayang", ErrUnprocessable)
	}

	commissionPct, err := s.deps.Settings.CommissionDefault()
	if err != nil {
		return nil, err
	}
	bearer, err := s.deps.Settings.GatewayFeeBearer()
	if err != nil {
		return nil, err
	}
	feePct, feeFlat, err := s.deps.Settings.GatewayFeeParams()
	if err != nil {
		return nil, err
	}

	tx, err := s.deps.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var subtotal int64
	now := time.Now()
	for i := range items {
		if items[i].Qty <= 0 {
			return nil, fmt.Errorf("%w: qty harus > 0", ErrUnprocessable)
		}
		// Lock the ticket type row to prevent overselling.
		var tt model.TicketType
		var saleStarts, saleEnds sql.NullTime
		err := tx.QueryRow(`SELECT id,event_id,name,price_cents,quota,sold,sale_starts_at,sale_ends_at
			FROM ticket_types WHERE id=? FOR UPDATE`, items[i].TicketTypeID).
			Scan(&tt.ID, &tt.EventID, &tt.Name, &tt.PriceCents, &tt.Quota, &tt.Sold,
				&saleStarts, &saleEnds)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("%w: ticket_type %d tidak ditemukan", ErrUnprocessable, items[i].TicketTypeID)
			}
			return nil, err
		}
		if tt.EventID != eventID {
			return nil, fmt.Errorf("%w: ticket_type %d bukan milik event ini", ErrUnprocessable, items[i].TicketTypeID)
		}
		if !tt.SaleStartsAt.IsZero() && now.Before(tt.SaleStartsAt) {
			return nil, fmt.Errorf("%w: penjualan %q belum dibuka", ErrUnprocessable, tt.Name)
		}
		if !tt.SaleEndsAt.IsZero() && now.After(tt.SaleEndsAt) {
			return nil, fmt.Errorf("%w: penjualan %q sudah ditutup", ErrUnprocessable, tt.Name)
		}
		if tt.Quota-tt.Sold < items[i].Qty {
			return nil, fmt.Errorf("%w: kuota %q tersisa %d", ErrUnprocessable, tt.Name, tt.Quota-tt.Sold)
		}
		if saleStarts.Valid {
			tt.SaleStartsAt = saleStarts.Time
		}
		if saleEnds.Valid {
			tt.SaleEndsAt = saleEnds.Time
		}
		items[i].PriceCents = tt.PriceCents
		subtotal += tt.PriceCents * int64(items[i].Qty)
	}

	// Optional promo: validated server-side; the discount cuts the subtotal
	// BEFORE commission and gateway fee are computed.
	var discount int64
	var appliedPromo *model.PromoCode
	if strings.TrimSpace(promoCode) != "" {
		p, d, err := s.promos.checkPromoTx(tx, userID, eventID, promoCode, subtotal)
		if err != nil {
			return nil, err
		}
		appliedPromo, discount = p, d
	}
	netSubtotal := subtotal - discount
	commission := int64(math.Round(float64(netSubtotal) * commissionPct / 100))
	gatewayFee := int64(math.Ceil(float64(netSubtotal)*feePct/100)) + feeFlat
	gross := netSubtotal + commission
	if bearer == model.FeeBearerBuyer {
		gross += gatewayFee
	}

	orderCode := s.uniqueOrderCode(tx)
	order := &model.Order{
		OrderCode:        orderCode,
		UserID:           userID,
		EventID:          eventID,
		SubtotalCents:    subtotal,
		DiscountCents:    discount,
		CommissionCents:  commission,
		GatewayFeeCents:  gatewayFee,
		GatewayFeeBearer: bearer,
		GrossCents:       gross,
		Status:           model.OrderPending,
		Items:            items,
	}
	if appliedPromo != nil {
		order.PromoCode = appliedPromo.Code
	}
	orderID, err := s.deps.Orders.CreateTx(tx, order)
	if err != nil {
		return nil, err
	}
	// Persist the checkout lines so tickets can be generated when payment succeeds.
	for i := range items {
		if err := s.deps.OrderLines.InsertTx(tx, orderID, items[i].TicketTypeID, items[i].Qty, items[i].PriceCents); err != nil {
			return nil, err
		}
	}
	if appliedPromo != nil {
		if err := s.promos.RecordUsageTx(tx, appliedPromo.ID, userID, orderID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	order.ID = orderID

	// Obtain a Snap token. In development (dummy keys / unreachable Midtrans)
	// the order is still created and the token falls back to "DUMMY".
	serverKey, clientKey, _, err := s.deps.Settings.MidtransKeys()
	if err != nil {
		return nil, err
	}
	env, _ := s.deps.Settings.MidtransEnv()
	mc := midtrans.NewClient(serverKey, clientKey, "", env)

	result := &CreateOrderResult{Order: order, ClientKey: clientKey}
	if mc.IsDummy() {
		note := "mode development: snap dummy (isi Midtrans key asli via admin settings)"
		result.SnapToken = "DUMMY"
		result.PaymentNote = &note
		_ = s.deps.Orders.UpdateSnapToken(orderID, "DUMMY")
		order.SnapToken = "DUMMY"
		slog.Info("order created with dummy snap token", "order_code", orderCode)
		return result, nil
	}
	snap, err := mc.CreateSnapToken(orderCode, gross)
	if err != nil {
		note := "midtrans tidak terjangkau: " + err.Error()
		result.SnapToken = "DUMMY"
		result.PaymentNote = &note
		_ = s.deps.Orders.UpdateSnapToken(orderID, "DUMMY")
		order.SnapToken = "DUMMY"
		slog.Warn("midtrans snap failed, using dummy token", "order_code", orderCode, "err", err)
		return result, nil
	}
	result.SnapToken = snap.Token
	result.RedirectURL = &snap.RedirectURL
	_ = s.deps.Orders.UpdateSnapToken(orderID, snap.Token)
	order.SnapToken = snap.Token
	return result, nil
}

func (s *OrderService) uniqueOrderCode(tx *sql.Tx) string {
	for i := 0; i < 10; i++ {
		code := fmt.Sprintf("TIX-%s-%s", time.Now().Format("20060102"), randomAlnum(4))
		var n int
		_ = tx.QueryRow(`SELECT COUNT(*) FROM orders WHERE order_code=?`, code).Scan(&n)
		if n == 0 {
			return code
		}
	}
	// Extremely unlikely fallback: include nanoseconds.
	return fmt.Sprintf("TIX-%s-%s%d", time.Now().Format("20060102"), randomAlnum(4), time.Now().UnixNano()%1000)
}

// GetOrder returns an order if the viewer owns it or is admin.
func (s *OrderService) GetOrder(orderID, viewerID int64, viewerRole string) (*model.Order, error) {
	o, err := s.deps.Orders.FindByID(orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if viewerRole != model.RoleSuperAdmin && o.UserID != viewerID {
		return nil, ErrForbidden
	}
	return o, nil
}

// attachOrderDetails fills Items (order lines) and EventTitle on a page of
// orders with two batch queries, so list views render line items and event
// names without N+1 queries. Orders always carry a non-nil Items slice.
func attachOrderDetails(deps *Deps, orders []model.Order) error {
	if len(orders) == 0 {
		return nil
	}
	orderIDs := make([]int64, len(orders))
	eventIDs := make([]int64, 0, len(orders))
	seenEvent := make(map[int64]bool)
	for i, o := range orders {
		orderIDs[i] = o.ID
		if !seenEvent[o.EventID] {
			seenEvent[o.EventID] = true
			eventIDs = append(eventIDs, o.EventID)
		}
	}
	lines, err := deps.OrderLines.ListByOrderIDs(orderIDs)
	if err != nil {
		return err
	}
	titles, err := deps.Events.TitlesByIDs(eventIDs)
	if err != nil {
		return err
	}
	for i := range orders {
		if items, ok := lines[orders[i].ID]; ok {
			orders[i].Items = items
		} else {
			orders[i].Items = []model.OrderItem{}
		}
		orders[i].EventTitle = titles[orders[i].EventID]
	}
	return nil
}

// ListMine returns the viewer's order history.
func (s *OrderService) ListMine(userID int64, status string, page, perPage int) ([]model.Order, int64, error) {
	orders, total, err := s.deps.Orders.ListByUser(userID, status, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	if err := attachOrderDetails(s.deps, orders); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListForSeller returns orders placed for events owned by the seller.
func (s *OrderService) ListForSeller(sellerID int64, status string, page, perPage int) ([]model.Order, int64, error) {
	orders, total, err := s.deps.Orders.ListBySeller(sellerID, status, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	if err := attachOrderDetails(s.deps, orders); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListAll returns every order (used for super_admin convenience).
func (s *OrderService) ListAll(status string, page, perPage int) ([]model.Order, int64, error) {
	orders, total, err := s.deps.Orders.ListAll(status, "", "", perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	if err := attachOrderDetails(s.deps, orders); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// RequestRefund moves a paid order to refund_requested.
func (s *OrderService) RequestRefund(orderID, userID int64, reason string) (*model.Order, error) {
	o, err := s.deps.Orders.FindByID(orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if o.UserID != userID {
		return nil, ErrForbidden
	}
	if o.Status != model.OrderPaid {
		return nil, fmt.Errorf("%w: hanya order paid yang bisa di-refund", ErrUnprocessable)
	}
	if _, err := s.deps.Refunds.FindPendingByOrder(orderID); err == nil {
		return nil, fmt.Errorf("%w: sudah ada pengajuan refund pending", ErrConflict)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if _, err := s.deps.Refunds.Create(orderID, o.GrossCents, reason); err != nil {
		return nil, err
	}
	if err := s.deps.Orders.UpdateStatus(orderID, model.OrderRefundRequested); err != nil {
		return nil, err
	}
	o.Status = model.OrderRefundRequested
	return o, nil
}

// ---------- Webhook ----------

type WebhookResult struct {
	Status string `json:"status"` // ok | duplicate_ignored
}

// HandleNotification processes a Midtrans webhook: verifies the signature,
// ignores duplicates (idempotency), maps the transaction status, and on
// payment success generates the concrete tickets.
func (s *OrderService) HandleNotification(raw []byte) (*WebhookResult, error) {
	var n midtrans.Notification
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, fmt.Errorf("%w: payload bukan JSON valid", ErrUnprocessable)
	}
	if n.OrderID == "" || n.SignatureKey == "" || n.TransactionID == "" {
		return nil, fmt.Errorf("%w: field webhook tidak lengkap", ErrUnprocessable)
	}
	serverKey, _, _, err := s.deps.Settings.MidtransKeys()
	if err != nil {
		return nil, err
	}
	if !midtrans.VerifySignature(serverKey, n.OrderID, n.StatusCode, n.GrossAmount, n.SignatureKey) {
		return nil, ErrUnauthorized // caller maps to 401
	}

	order, err := s.deps.Orders.FindByCode(n.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: order %s tidak dikenal", ErrNotFound, n.OrderID)
		}
		return nil, err
	}

	var gross int64
	fmt.Sscanf(n.GrossAmount, "%d", &gross)

	tx, err := s.deps.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	inserted, err := s.deps.Payments.InsertTx(tx, &model.Payment{
		OrderID:           order.ID,
		MidtransOrderID:   n.OrderID,
		TransactionID:     n.TransactionID,
		PaymentType:       n.PaymentType,
		TransactionStatus: n.TransactionStatus,
		GrossAmount:       gross,
	}, string(raw))
	if err != nil {
		return nil, err
	}
	if !inserted {
		// Duplicate notification: same order_id + transaction_id + status.
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &WebhookResult{Status: "duplicate_ignored"}, nil
	}

	newStatus := mapTransactionStatus(n.TransactionStatus, n.FraudStatus)
	becamePaid := newStatus == model.OrderPaid && order.Status != model.OrderPaid
	if becamePaid {
		if err := s.fulfillOrderTx(tx, order); err != nil {
			return nil, err
		}
	} else if newStatus != "" && newStatus != order.Status {
		if err := s.deps.Orders.UpdateStatusTx(tx, order.ID, newStatus); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	slog.Info("webhook processed", "order", n.OrderID, "status", n.TransactionStatus)
	if becamePaid {
		s.notifyTicketsIssued(order)
	}
	return &WebhookResult{Status: "ok"}, nil
}

// notifyTicketsIssued sends a WhatsApp notification to the buyer when tickets
// are issued. Fire-and-forget: failures are only logged.
func (s *OrderService) notifyTicketsIssued(order *model.Order) {
	u, err := s.deps.Users.FindByID(order.UserID)
	if err != nil {
		return
	}
	e, err := s.deps.Events.FindByID(order.EventID)
	if err != nil {
		return
	}
	msg := fmt.Sprintf("Tiket kamu untuk *%s* sudah terbit! 🎟️\nKode order: %s\nLihat di menu Tiket Saya.",
		e.Title, order.OrderCode)
	s.deps.Notify.Notify(u.Phone, msg)
}

// mapTransactionStatus maps Midtrans statuses to order statuses.
func mapTransactionStatus(ts, fraud string) string {
	switch ts {
	case "settlement":
		return model.OrderPaid
	case "capture":
		if fraud == "challenge" {
			return model.OrderPending
		}
		return model.OrderPaid
	case "pending":
		return model.OrderPending
	case "expire":
		return model.OrderCancelled
	case "cancel":
		return model.OrderCancelled
	case "deny":
		return model.OrderFailed
	default:
		return ""
	}
}

// fulfillOrderTx generates concrete tickets for a paid order and bumps sold counters.
// Quota is re-checked under lock; if stock ran out, the order is marked failed
// instead of silently overselling.
func (s *OrderService) fulfillOrderTx(tx *sql.Tx, order *model.Order) error {
	lines, err := s.deps.OrderLines.ListByOrderTx(tx, order.ID)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		return fmt.Errorf("order %s has no order lines", order.OrderCode)
	}
	for _, ln := range lines {
		var quota, sold int
		if err := tx.QueryRow(`SELECT quota, sold FROM ticket_types WHERE id=? FOR UPDATE`,
			ln.TicketTypeID).Scan(&quota, &sold); err != nil {
			return err
		}
		if quota-sold < ln.Qty {
			// Oversold between order and payment: fail the order loudly.
			if err := s.deps.Orders.UpdateStatusTx(tx, order.ID, model.OrderFailed); err != nil {
				return err
			}
			slog.Error("oversold at payment time", "order", order.OrderCode,
				"ticket_type", ln.TicketTypeID, "need", ln.Qty, "left", quota-sold)
			return nil
		}
		for q := 0; q < ln.Qty; q++ {
			code := s.uniqueTicketCode(tx)
			if err := s.deps.Items.InsertTx(tx, order.ID, ln.TicketTypeID, code,
				s.qrPayload(code), ln.PriceCents); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`UPDATE ticket_types SET sold = sold + ? WHERE id=?`,
			ln.Qty, ln.TicketTypeID); err != nil {
			return err
		}
	}
	return s.deps.Orders.UpdateStatusTx(tx, order.ID, model.OrderPaid)
}

// uniqueTicketCode generates a unique per-ticket code.
func (s *OrderService) uniqueTicketCode(tx *sql.Tx) string {
	for i := 0; i < 10; i++ {
		code := "TIX-" + randomAlnum(8)
		var n int
		_ = tx.QueryRow(`SELECT COUNT(*) FROM order_items WHERE ticket_code=?`, code).Scan(&n)
		if n == 0 {
			return code
		}
	}
	return fmt.Sprintf("TIX-%s%d", randomAlnum(8), time.Now().UnixNano()%100000)
}

// qrPayload builds the verifiable QR payload: "<ticket_code>.<hmac>".
func (s *OrderService) qrPayload(ticketCode string) string {
	return ticketCode + "." + crypto.HMACSHA256Hex(s.qrSecret, ticketCode)
}

func randomAlnum(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}
