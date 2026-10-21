package handler

import (
	"net/http"
	"strconv"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/middleware"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/service"
)

// ---------- Admin ----------

type AdminHandler struct{ svc *service.AdminService }

func NewAdminHandler(svc *service.AdminService) *AdminHandler { return &AdminHandler{svc: svc} }

// --- users ---

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	q := r.URL.Query()
	users, total, err := h.svc.ListUsers(q.Get("role"), q.Get("search"), page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"users": users, "page": page, "per_page": perPage, "total": total,
	})
}

func (h *AdminHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	u, err := h.svc.UpdateRole(middleware.UserID(r), id, req.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"user": u})
}

func (h *AdminHandler) VerifySeller(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req struct {
		Verified    bool   `json:"verified"`
		BankName    string `json:"bank_name"`
		BankAccount string `json:"bank_account"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	u, err := h.svc.VerifySeller(middleware.UserID(r), id, req.Verified, req.BankName, req.BankAccount)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"user": u})
}

// --- events (admin approve/reject injected via EventService) ---

type AdminEventHandler struct{ svc *service.EventService }

func NewAdminEventHandler(svc *service.EventService) *AdminEventHandler {
	return &AdminEventHandler{svc: svc}
}

func (h *AdminEventHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	e, err := h.svc.Approve(id, middleware.UserID(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"event": e})
}

func (h *AdminEventHandler) Reject(w http.ResponseWriter, r *http.Request) {
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
	e, err := h.svc.Reject(id, middleware.UserID(r), req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"event": e})
}

// List returns all events for moderation, optionally filtered by status
// (?status=pending_review|published|draft|rejected|finished).
func (h *AdminEventHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	events, total, err := h.svc.ListAll(r.URL.Query().Get("status"), page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"events": events, "page": page, "per_page": perPage, "total": total,
	})
}

// --- transactions ---

func (h *AdminHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	q := r.URL.Query()
	orders, total, err := h.svc.ListTransactions(q.Get("status"), q.Get("from"), q.Get("to"), page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"orders": orders, "page": page, "per_page": perPage, "total": total,
	})
}

// --- refunds ---

// ListRefunds lists refund requests for moderation.
func (h *AdminHandler) ListRefunds(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	refunds, total, err := h.svc.ListRefunds(r.URL.Query().Get("status"), page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	if refunds == nil {
		refunds = []model.Refund{}
	}
	writeData(w, http.StatusOK, map[string]any{
		"refunds": refunds, "page": page, "per_page": perPage, "total": total,
	})
}

func (h *AdminHandler) ApproveRefund(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req struct {
		AmountCents *int64  `json:"amount_cents"`
		Reason      *string `json:"reason"`
	}
	// Body is optional; an empty body means "full gross, keep reason".
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
	}
	ref, err := h.svc.ApproveRefund(middleware.UserID(r), id, req.AmountCents, req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"refund": ref})
}

func (h *AdminHandler) RejectRefund(w http.ResponseWriter, r *http.Request) {
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
	ref, err := h.svc.RejectRefund(middleware.UserID(r), id, req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"refund": ref})
}

// --- payouts ---

// SellerPayouts lists payouts belonging to the signed-in seller
// (registered under the seller route group, not the admin one).
func (h *AdminHandler) SellerPayouts(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	payouts, total, err := h.svc.ListPayouts(r.URL.Query().Get("status"), middleware.UserID(r), page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"payouts": payouts, "page": page, "per_page": perPage, "total": total,
	})
}

func (h *AdminHandler) ListPayouts(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	q := r.URL.Query()
	var sellerID int64
	if v := q.Get("seller_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			sellerID = n
		}
	}
	payouts, total, err := h.svc.ListPayouts(q.Get("status"), sellerID, page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"payouts": payouts, "page": page, "per_page": perPage, "total": total,
	})
}

func (h *AdminHandler) CreatePayout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SellerID    int64  `json:"seller_id"`
		EventID     int64  `json:"event_id"`
		AmountCents int64  `json:"amount_cents"`
		Method      string `json:"method"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	p, err := h.svc.CreatePayout(middleware.UserID(r), req.SellerID, req.EventID, req.AmountCents, req.Method)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, map[string]any{"payout": p})
}

// SendIrisPayout disburses a pending payout through Midtrans Iris.
func (h *AdminHandler) SendIrisPayout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	p, err := h.svc.SendIrisPayout(middleware.UserID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"payout": p})
}

// IrisStatus refreshes a payout's status from Iris.
func (h *AdminHandler) IrisStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	p, err := h.svc.RefreshIrisStatus(middleware.UserID(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"payout": p})
}

// --- notification settings ---

func (h *AdminHandler) GetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.GetNotificationSettings()
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, view)
}

func (h *AdminHandler) UpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WAProvider    *string `json:"wa_provider"`
		WAFonnteToken *string `json:"wa_fonnte_token"`
		WAEnabled     *bool   `json:"wa_enabled"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	view, err := h.svc.UpdateNotificationSettings(middleware.UserID(r), req.WAProvider, req.WAFonnteToken, req.WAEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, view)
}

func (h *AdminHandler) TestNotification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		To string `json:"to"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	res, err := h.svc.TestNotification(req.To)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, res)
}

func (h *AdminHandler) PayPayout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req struct {
		ProofURL string `json:"proof_url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	p, err := h.svc.MarkPayoutPaid(middleware.UserID(r), id, req.ProofURL)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"payout": p})
}

// --- reports & audit ---

func (h *AdminHandler) ReportSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sum, err := h.svc.ReportSummary(q.Get("from"), q.Get("to"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, sum)
}

func (h *AdminHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	q := r.URL.Query()
	var actorID int64
	if v := q.Get("actor_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			actorID = n
		}
	}
	logs, total, err := h.svc.ListAuditLogs(q.Get("action"), actorID, page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"audit_logs": logs, "page": page, "per_page": perPage, "total": total,
	})
}

// --- midtrans settings ---

func (h *AdminHandler) GetMidtransSettings(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.GetMidtransSettings()
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, view)
}

func (h *AdminHandler) UpdateMidtransSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServerKey                 *string  `json:"server_key"`
		ClientKey                 *string  `json:"client_key"`
		MerchantID                *string  `json:"merchant_id"`
		Environment               *string  `json:"environment"`
		CommissionDefault         *float64 `json:"commission_default"`
		GatewayFeeBearer          *string  `json:"gateway_fee_bearer"`
		GatewayFeePercent         *float64 `json:"gateway_fee_percent"`
		GatewayFeeFlatCents       *int64   `json:"gateway_fee_flat_cents"`
		RequireSellerVerification *bool    `json:"require_seller_verification"`
		IrisAPIKey                *string  `json:"iris_api_key"`
		IrisFromAccount           *string  `json:"iris_from_account"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	view, err := h.svc.UpdateMidtransSettings(middleware.UserID(r), service.MidtransSettingsPatch{
		ServerKey: req.ServerKey, ClientKey: req.ClientKey, MerchantID: req.MerchantID,
		Environment: req.Environment, CommissionDefault: req.CommissionDefault,
		GatewayFeeBearer: req.GatewayFeeBearer, GatewayFeePercent: req.GatewayFeePercent,
		GatewayFeeFlatCents:       req.GatewayFeeFlatCents,
		RequireSellerVerification: req.RequireSellerVerification,
		IrisAPIKey:                req.IrisAPIKey,
		IrisFromAccount:           req.IrisFromAccount,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, view)
}

func (h *AdminHandler) TestMidtrans(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.TestMidtransConnection()
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, res)
}

// MidtransClientKey is public: the client key is meant for the browser (snap.js).
func (h *AdminHandler) MidtransClientKey(w http.ResponseWriter, r *http.Request) {
	clientKey, env, err := h.svc.MidtransClientKey()
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"client_key": clientKey, "environment": env,
	})
}

// PublicConfig is public: non-sensitive platform display settings (commission
// and gateway-fee policy) so public pages render live values, not hardcodes.
func (h *AdminHandler) PublicConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.PublicConfig()
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, cfg)
}
