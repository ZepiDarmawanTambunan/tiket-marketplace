package handler

import (
	"net/http"
	"strconv"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/middleware"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/service"
)

// ---------- Promo codes ----------

type PromoHandler struct{ svc *service.PromoService }

func NewPromoHandler(svc *service.PromoService) *PromoHandler { return &PromoHandler{svc: svc} }

type promoRequest struct {
	Code          string  `json:"code"`
	Type          string  `json:"type"`
	Value         float64 `json:"value"`
	MinSpendCents int64   `json:"min_spend_cents"`
	Quota         *int    `json:"quota"`
	PerUserLimit  *int    `json:"per_user_limit"`
	StartsAt      string  `json:"starts_at"`
	EndsAt        string  `json:"ends_at"`
	Scope         string  `json:"scope"`
	EventID       *int64  `json:"event_id"`
	Active        *bool   `json:"active"`
}

func toPromoInput(req promoRequest) (service.PromoInput, error) {
	in := service.PromoInput{
		Code: req.Code, Type: req.Type, Value: req.Value,
		MinSpendCents: req.MinSpendCents, Quota: req.Quota,
		PerUserLimit: req.PerUserLimit, Scope: req.Scope,
		EventID: req.EventID, Active: req.Active,
	}
	if req.StartsAt != "" {
		t, err := parseTime(req.StartsAt)
		if err != nil {
			return in, errBadRequest("starts_at harus RFC3339")
		}
		in.StartsAt = &t
	}
	if req.EndsAt != "" {
		t, err := parseTime(req.EndsAt)
		if err != nil {
			return in, errBadRequest("ends_at harus RFC3339")
		}
		in.EndsAt = &t
	}
	return in, nil
}

// Create: seller (own event promos) or admin.
func (h *PromoHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req promoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	in, err := toPromoInput(req)
	if err != nil {
		writeError(w, err)
		return
	}
	p, err := h.svc.Create(middleware.UserID(r), middleware.Role(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, map[string]any{"promo": p})
}

// List: seller sees own promos; admin sees all. Query: active, event_id.
func (h *PromoHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	q := r.URL.Query()
	var active *bool
	if v := q.Get("active"); v != "" {
		b := v == "true" || v == "1"
		active = &b
	}
	var eventID int64
	if v := q.Get("event_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			eventID = n
		}
	}
	promos, total, err := h.svc.List(middleware.UserID(r), middleware.Role(r), active, eventID, page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	if promos == nil {
		promos = []model.PromoCode{}
	}
	writeData(w, http.StatusOK, map[string]any{
		"promos": promos, "page": page, "per_page": perPage, "total": total,
	})
}

// Update: full replace; code immutable.
func (h *PromoHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req promoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	in, err := toPromoInput(req)
	if err != nil {
		writeError(w, err)
		return
	}
	p, err := h.svc.Update(middleware.UserID(r), middleware.Role(r), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"promo": p})
}

// Delete: soft delete (active=false).
func (h *PromoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	if err := h.svc.Delete(middleware.UserID(r), middleware.Role(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"deleted": true})
}

// Validate: any authenticated user checks a code against a cart.
func (h *PromoHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code    string `json:"code"`
		EventID int64  `json:"event_id"`
		Items   []struct {
			TicketTypeID int64 `json:"ticket_type_id"`
			Qty          int   `json:"qty"`
		} `json:"items"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	items := make([]model.OrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, model.OrderItem{TicketTypeID: it.TicketTypeID, Qty: it.Qty})
	}
	res, err := h.svc.Validate(middleware.UserID(r), req.EventID, req.Code, items)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, res)
}
