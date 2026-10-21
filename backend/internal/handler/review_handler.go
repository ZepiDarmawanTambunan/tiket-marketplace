package handler

import (
	"net/http"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/middleware"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/service"
)

// ---------- Reviews ----------

type ReviewHandler struct{ svc *service.ReviewService }

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler { return &ReviewHandler{svc: svc} }

// List is public: reviews of an event + rating summary.
func (h *ReviewHandler) List(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	page, perPage := pageParams(r)
	res, err := h.svc.List(id, page, perPage)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, res)
}

// Create: authenticated buyer with a paid order for a finished event.
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req struct {
		Rating int    `json:"rating"`
		Text   string `json:"text"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	rv, err := h.svc.Create(middleware.UserID(r), id, req.Rating, req.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, map[string]any{"review": rv})
}

// Update: the caller's own review.
func (h *ReviewHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	var req struct {
		Rating int    `json:"rating"`
		Text   string `json:"text"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	rv, err := h.svc.Update(middleware.UserID(r), id, req.Rating, req.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"review": rv})
}

// DeleteByAdmin: admin moderation.
func (h *ReviewHandler) DeleteByAdmin(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, errBadRequest("id tidak valid"))
		return
	}
	if err := h.svc.DeleteByAdmin(middleware.UserID(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{"deleted": true})
}
