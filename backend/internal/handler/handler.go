package handler

import (
	"net/http"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/middleware"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/service"
)

// ---------- Auth ----------

type AuthHandler struct{ svc *service.AuthService }

func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, errBadRequest("name, email, dan password wajib diisi"))
		return
	}
	res, err := h.svc.Register(req.Name, req.Email, req.Phone, req.Password, req.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, res)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	res, err := h.svc.Login(req.Email, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, res)
}

func (h *AuthHandler) Me(deps *service.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := deps.Users.FindByID(middleware.UserID(r))
		if err != nil {
			writeError(w, service.ErrNotFound)
			return
		}
		writeData(w, http.StatusOK, map[string]any{"user": u})
	}
}

// ---------- Events ----------

type EventHandler struct {
	svc *service.EventService
	tts *service.TicketTypeService
}

func NewEventHandler(svc *service.EventService, tts *service.TicketTypeService) *EventHandler {
	return &EventHandler{svc: svc, tts: tts}
}

// List is public; with ?mine=true it lists the seller's own events (auth).
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	q := r.URL.Query()
	var (
		events []model.Event
		total  int64
		err    error
	)
	if q.Get("mine") == "true" {
		uid := middleware.UserID(r)
		if uid == 0 {
			writeError(w, service.ErrUnauthorized)
			return
		}
		events, total, err = h.svc.ListMine(uid, page, perPage)
	} else {
		events, total, err = h.svc.ListPublic(q.Get("category"), q.Get("search"),
			q.Get("from"), q.Get("to"), page, perPage)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"events": events, "page": page, "per_page": perPage, "total": total,
	})
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	e, err := h.svc.GetDetail(id, middleware.UserID(r), middleware.Role(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"event": e})
}

type eventRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	Venue        string `json:"venue"`
	Address      string `json:"address"`
	MapsURL      string `json:"maps_url"`
	StartsAt     string `json:"starts_at"`
	EndsAt       string `json:"ends_at"`
	BannerURL    string `json:"banner_url"`
	RefundPolicy string `json:"refund_policy"`
}

func (h *EventHandler) toModel(req eventRequest) (*model.Event, error) {
	starts, err := parseTime(req.StartsAt)
	if err != nil {
		return nil, errBadRequest("starts_at harus RFC3339")
	}
	ends, err := parseTime(req.EndsAt)
	if err != nil {
		return nil, errBadRequest("ends_at harus RFC3339")
	}
	return &model.Event{
		Title: req.Title, Description: req.Description, Category: req.Category,
		Venue: req.Venue, Address: req.Address, MapsURL: req.MapsURL,
		StartsAt: starts, EndsAt: ends, BannerURL: req.BannerURL,
		RefundPolicy: req.RefundPolicy,
	}, nil
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.Title == "" {
		writeError(w, errBadRequest("title wajib diisi"))
		return
	}
	e, err := h.toModel(req)
	if err != nil {
		writeError(w, err)
		return
	}
	created, err := h.svc.CreateEvent(middleware.UserID(r), e)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, map[string]any{"event": created})
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req eventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	e, err := h.toModel(req)
	if err != nil {
		writeError(w, err)
		return
	}
	updated, err := h.svc.UpdateEvent(middleware.UserID(r), id, e)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"event": updated})
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	if err := h.svc.DeleteEvent(middleware.UserID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *EventHandler) Submit(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	e, err := h.svc.SubmitForReview(middleware.UserID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"event": e})
}

// ---------- Ticket types ----------

type ticketTypeRequest struct {
	Name         string `json:"name"`
	PriceCents   int64  `json:"price_cents"`
	Quota        int    `json:"quota"`
	SaleStartsAt string `json:"sale_starts_at"`
	SaleEndsAt   string `json:"sale_ends_at"`
}

func (h *EventHandler) ListTicketTypes(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	tts, err := h.tts.ListByEvent(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"ticket_types": tts})
}

func (h *EventHandler) CreateTicketType(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req ticketTypeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	starts, err := parseTime(req.SaleStartsAt)
	if err != nil {
		writeError(w, errBadRequest("sale_starts_at harus RFC3339"))
		return
	}
	ends, err := parseTime(req.SaleEndsAt)
	if err != nil {
		writeError(w, errBadRequest("sale_ends_at harus RFC3339"))
		return
	}
	tt, err := h.tts.Create(middleware.UserID(r), eventID, &model.TicketType{
		Name: req.Name, PriceCents: req.PriceCents, Quota: req.Quota,
		SaleStartsAt: starts, SaleEndsAt: ends,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, map[string]any{"ticket_type": tt})
}

func (h *EventHandler) UpdateTicketType(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req ticketTypeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	starts, _ := parseTime(req.SaleStartsAt)
	ends, _ := parseTime(req.SaleEndsAt)
	tt, err := h.tts.Update(middleware.UserID(r), id, &model.TicketType{
		Name: req.Name, PriceCents: req.PriceCents, Quota: req.Quota,
		SaleStartsAt: starts, SaleEndsAt: ends,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"ticket_type": tt})
}

func (h *EventHandler) DeleteTicketType(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	if err := h.tts.Delete(middleware.UserID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"deleted": true})
}

// ---------- Orders ----------

type OrderHandler struct{ svc *service.OrderService }

func NewOrderHandler(svc *service.OrderService) *OrderHandler { return &OrderHandler{svc: svc} }

type createOrderRequest struct {
	EventID   int64  `json:"event_id"`
	PromoCode string `json:"promo_code"`
	Items     []struct {
		TicketTypeID int64 `json:"ticket_type_id"`
		Qty          int   `json:"qty"`
	} `json:"items"`
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	items := make([]model.OrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, model.OrderItem{TicketTypeID: it.TicketTypeID, Qty: it.Qty})
	}
	res, err := h.svc.CreateOrder(middleware.UserID(r), req.EventID, items, req.PromoCode)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, res)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	o, err := h.svc.GetOrder(id, middleware.UserID(r), middleware.Role(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"order": o})
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	q := r.URL.Query().Get("status")
	uid := middleware.UserID(r)
	var (
		orders []model.Order
		total  int64
		err    error
	)
	switch middleware.Role(r) {
	case "seller":
		// Sellers see orders placed for their own events.
		orders, total, err = h.svc.ListForSeller(uid, q, page, perPage)
	case "super_admin":
		orders, total, err = h.svc.ListAll(q, page, perPage)
	default:
		orders, total, err = h.svc.ListMine(uid, q, page, perPage)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"orders": orders, "page": page, "per_page": perPage, "total": total,
	})
}

func (h *OrderHandler) Refund(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	o, err := h.svc.RequestRefund(id, middleware.UserID(r), req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"order": o})
}

// ---------- Webhook ----------

type WebhookHandler struct{ svc *service.OrderService }

func NewWebhookHandler(svc *service.OrderService) *WebhookHandler { return &WebhookHandler{svc: svc} }

func (h *WebhookHandler) Midtrans(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// Read the raw body: the signature is computed over the field values,
	// and we keep the exact bytes for the payment audit record.
	body := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err != nil {
			break
		}
		if len(body) > 1<<20 {
			writeError(w, errBadRequest("payload terlalu besar"))
			return
		}
	}
	res, err := h.svc.HandleNotification(body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, res)
}

// ---------- Tickets & check-in ----------

type TicketHandler struct{ svc *service.TicketService }

func NewTicketHandler(svc *service.TicketService) *TicketHandler { return &TicketHandler{svc: svc} }

func (h *TicketHandler) MyTickets(w http.ResponseWriter, r *http.Request) {
	tickets, err := h.svc.MyTickets(middleware.UserID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"tickets": tickets})
}

// Transfer moves a ticket to another registered user.
func (h *TicketHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TicketCode string `json:"ticket_code"`
		ToEmail    string `json:"to_email"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.TicketCode == "" || req.ToEmail == "" {
		writeError(w, errBadRequest("ticket_code dan to_email wajib diisi"))
		return
	}
	t, err := h.svc.TransferTicket(middleware.UserID(r), req.TicketCode, req.ToEmail)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"ticket_code": t.TicketCode, "holder_name": t.HolderName,
		"transferred_to_user_id": t.HolderUserID,
	})
}

// TransferHistory returns transfers involving the logged-in user.
func (h *TicketHandler) TransferHistory(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	transfers, total, err := h.svc.TransferHistory(middleware.UserID(r), page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	if transfers == nil {
		transfers = []model.TicketTransfer{}
	}
	writeData(w, http.StatusOK, map[string]any{
		"transfers": transfers, "page": page, "per_page": perPage, "total": total,
	})
}

func (h *TicketHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TicketCode string `json:"ticket_code"`
		QRPayLoad  string `json:"qr_payload"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	res, err := h.svc.CheckIn(middleware.UserID(r), middleware.Role(r), req.TicketCode, req.QRPayLoad)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, res)
}
