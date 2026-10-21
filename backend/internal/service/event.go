package service

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/repository"
)

// ---------- Events ----------

type EventService struct{ deps *Deps }

func NewEventService(deps *Deps) *EventService { return &EventService{deps: deps} }

// CreateEvent creates a new event in draft status. Sellers may sell immediately
// unless require_seller_verification is enabled (default false).
func (s *EventService) CreateEvent(sellerID int64, e *model.Event) (*model.Event, error) {
	seller, err := s.deps.Users.FindByID(sellerID)
	if err != nil {
		return nil, err
	}
	if seller.Role != model.RoleSeller && seller.Role != model.RoleSuperAdmin {
		return nil, fmt.Errorf("%w: hanya seller yang bisa membuat event", ErrForbidden)
	}
	if req, err := s.deps.Settings.RequireSellerVerification(); err != nil {
		return nil, err
	} else if req && !seller.SellerVerified && seller.Role != model.RoleSuperAdmin {
		return nil, fmt.Errorf("%w: akun seller belum terverifikasi", ErrForbidden)
	}
	if e.StartsAt.IsZero() || e.EndsAt.IsZero() || !e.EndsAt.After(e.StartsAt) {
		return nil, fmt.Errorf("%w: starts_at/ends_at tidak valid", ErrUnprocessable)
	}
	e.SellerID = sellerID
	return s.deps.Events.Create(e)
}

// UpdateEvent edits an event owned by the seller; only draft/rejected events are editable.
func (s *EventService) UpdateEvent(sellerID, eventID int64, e *model.Event) (*model.Event, error) {
	cur, err := s.ownedEvent(sellerID, eventID)
	if err != nil {
		return nil, err
	}
	if cur.Status != model.EventDraft && cur.Status != model.EventRejected {
		return nil, fmt.Errorf("%w: event yang sudah tayang tidak bisa diubah", ErrConflict)
	}
	if e.StartsAt.IsZero() || e.EndsAt.IsZero() || !e.EndsAt.After(e.StartsAt) {
		return nil, fmt.Errorf("%w: starts_at/ends_at tidak valid", ErrUnprocessable)
	}
	cur.Title = e.Title
	cur.Description = e.Description
	cur.Category = e.Category
	cur.Venue = e.Venue
	cur.Address = e.Address
	cur.MapsURL = e.MapsURL
	cur.StartsAt = e.StartsAt
	cur.EndsAt = e.EndsAt
	cur.BannerURL = e.BannerURL
	cur.RefundPolicy = e.RefundPolicy
	if err := s.deps.Events.Update(cur); err != nil {
		return nil, err
	}
	return cur, nil
}

// DeleteEvent removes a draft/rejected event with no orders.
func (s *EventService) DeleteEvent(sellerID, eventID int64) error {
	cur, err := s.ownedEvent(sellerID, eventID)
	if err != nil {
		return err
	}
	if cur.Status != model.EventDraft && cur.Status != model.EventRejected {
		return fmt.Errorf("%w: hanya draft/rejected yang bisa dihapus", ErrConflict)
	}
	if has, err := s.deps.Orders.HasOrdersForEvent(eventID); err != nil {
		return err
	} else if has {
		return fmt.Errorf("%w: event sudah memiliki order", ErrConflict)
	}
	return s.deps.Events.Delete(eventID)
}

// SubmitForReview moves a draft/rejected event to pending_review.
func (s *EventService) SubmitForReview(sellerID, eventID int64) (*model.Event, error) {
	cur, err := s.ownedEvent(sellerID, eventID)
	if err != nil {
		return nil, err
	}
	if cur.Status != model.EventDraft && cur.Status != model.EventRejected {
		return nil, fmt.Errorf("%w: hanya draft/rejected yang bisa diajukan", ErrConflict)
	}
	tts, err := s.deps.Tickets.ListByEvent(eventID)
	if err != nil {
		return nil, err
	}
	if len(tts) == 0 {
		return nil, fmt.Errorf("%w: event butuh minimal 1 tipe tiket", ErrUnprocessable)
	}
	if err := s.deps.Events.UpdateStatus(eventID, model.EventPendingReview); err != nil {
		return nil, err
	}
	cur.Status = model.EventPendingReview
	return cur, nil
}

// Approve publishes a pending event (admin only).
func (s *EventService) Approve(eventID int64, actorID int64) (*model.Event, error) {
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if e.Status != model.EventPendingReview {
		return nil, fmt.Errorf("%w: hanya pending_review yang bisa di-approve", ErrConflict)
	}
	if err := s.deps.Events.UpdateStatus(eventID, model.EventPublished); err != nil {
		return nil, err
	}
	e.Status = model.EventPublished
	_ = s.deps.Audit.Log(int64ptr(actorID), "event.approve", fmt.Sprintf("event_id=%d title=%q", e.ID, e.Title))
	return e, nil
}

// Reject sends an event back with a reason (admin only).
func (s *EventService) Reject(eventID int64, actorID int64, reason string) (*model.Event, error) {
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if e.Status != model.EventPendingReview {
		return nil, fmt.Errorf("%w: hanya pending_review yang bisa di-reject", ErrConflict)
	}
	if err := s.deps.Events.UpdateStatus(eventID, model.EventRejected); err != nil {
		return nil, err
	}
	e.Status = model.EventRejected
	_ = s.deps.Audit.Log(int64ptr(actorID), "event.reject",
		fmt.Sprintf("event_id=%d reason=%q", e.ID, reason))
	return e, nil
}

// attachTicketTypes batch-loads ticket types for a page of events so list
// responses include them (event cards render the cheapest price).
func (s *EventService) attachTicketTypes(events []model.Event) ([]model.Event, error) {
	if len(events) == 0 {
		return events, nil
	}
	ids := make([]int64, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	byEvent, err := s.deps.Tickets.ListByEventIDs(ids)
	if err != nil {
		return nil, err
	}
	for i := range events {
		if tts, ok := byEvent[events[i].ID]; ok {
			events[i].TicketTypes = tts
		} else {
			// Tetap kirim array kosong agar kontrak JSON konsisten
			// (field tidak hilang karena omitempty).
			events[i].TicketTypes = []model.TicketType{}
		}
	}
	return events, nil
}

// ListPublic returns published events with filters.
func (s *EventService) ListPublic(category, search, from, to string, page, perPage int) ([]model.Event, int64, error) {
	f := repository.EventFilter{
		Category: category, Search: search, From: from, To: to,
		OnlyPublished: true, Limit: perPage, Offset: (page - 1) * perPage,
	}
	events, total, err := s.deps.Events.List(f)
	if err != nil {
		return nil, 0, err
	}
	events, err = s.attachTicketTypes(events)
	return events, total, err
}

// ListAll returns all events (admin), optionally filtered by status.
func (s *EventService) ListAll(status string, page, perPage int) ([]model.Event, int64, error) {
	f := repository.EventFilter{
		Status: status, Limit: perPage, Offset: (page - 1) * perPage,
	}
	events, total, err := s.deps.Events.List(f)
	if err != nil {
		return nil, 0, err
	}
	events, err = s.attachTicketTypes(events)
	return events, total, err
}

// ListMine returns events owned by a seller.
func (s *EventService) ListMine(sellerID int64, page, perPage int) ([]model.Event, int64, error) {
	f := repository.EventFilter{
		SellerID: sellerID, Limit: perPage, Offset: (page - 1) * perPage,
	}
	events, total, err := s.deps.Events.List(f)
	if err != nil {
		return nil, 0, err
	}
	events, err = s.attachTicketTypes(events)
	return events, total, err
}

// GetDetail returns an event with its ticket types. Unpublished events are only
// visible to the owner or an admin.
func (s *EventService) GetDetail(eventID int64, viewerID int64, viewerRole string) (*model.Event, error) {
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if e.Status != model.EventPublished &&
		viewerRole != model.RoleSuperAdmin && e.SellerID != viewerID {
		return nil, ErrNotFound
	}
	tts, err := s.deps.Tickets.ListByEvent(eventID)
	if err != nil {
		return nil, err
	}
	if tts == nil {
		tts = []model.TicketType{}
	}
	e.TicketTypes = tts
	if avg, count, err := s.deps.Reviews.Summary(eventID); err == nil {
		e.RatingAvg = round1(avg)
		e.RatingCount = count
	}
	return e, nil
}

func (s *EventService) ownedEvent(sellerID, eventID int64) (*model.Event, error) {
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if e.SellerID != sellerID {
		return nil, ErrForbidden
	}
	return e, nil
}

// FinishExpired marks published events whose ends_at passed as finished.
// Intended to be called by a scheduler or admin; safe to run repeatedly.
func (s *EventService) FinishExpired() (int64, error) {
	res, err := s.deps.DB.Exec(`UPDATE events SET status='finished'
		WHERE status='published' AND ends_at < NOW()`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ---------- Ticket types ----------

type TicketTypeService struct{ deps *Deps }

func NewTicketTypeService(deps *Deps) *TicketTypeService { return &TicketTypeService{deps: deps} }

// Create adds a ticket type to an event owned by the seller.
func (s *TicketTypeService) Create(sellerID, eventID int64, t *model.TicketType) (*model.TicketType, error) {
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if e.SellerID != sellerID {
		return nil, ErrForbidden
	}
	if e.Status != model.EventDraft && e.Status != model.EventRejected {
		return nil, fmt.Errorf("%w: tipe tiket hanya bisa ditambah saat draft/rejected", ErrConflict)
	}
	if t.PriceCents < 0 || t.Quota <= 0 {
		return nil, fmt.Errorf("%w: harga/kuota tidak valid", ErrUnprocessable)
	}
	t.EventID = eventID
	return s.deps.Tickets.Create(t)
}

// Update edits a ticket type; price/quota changes are blocked once sold > 0.
func (s *TicketTypeService) Update(sellerID, id int64, t *model.TicketType) (*model.TicketType, error) {
	cur, err := s.ownedTicketType(sellerID, id)
	if err != nil {
		return nil, err
	}
	if cur.Sold > 0 && (t.PriceCents != cur.PriceCents || t.Quota != cur.Quota) {
		return nil, fmt.Errorf("%w: harga/kuota tidak bisa diubah setelah ada penjualan", ErrConflict)
	}
	cur.Name = t.Name
	cur.PriceCents = t.PriceCents
	cur.Quota = t.Quota
	cur.SaleStartsAt = t.SaleStartsAt
	cur.SaleEndsAt = t.SaleEndsAt
	if err := s.deps.Tickets.Update(cur); err != nil {
		return nil, err
	}
	return cur, nil
}

// Delete removes a ticket type with no sales yet.
func (s *TicketTypeService) Delete(sellerID, id int64) error {
	cur, err := s.ownedTicketType(sellerID, id)
	if err != nil {
		return err
	}
	if cur.Sold > 0 {
		return fmt.Errorf("%w: tipe tiket sudah terjual", ErrConflict)
	}
	return s.deps.Tickets.Delete(id)
}

func (s *TicketTypeService) ownedTicketType(sellerID, id int64) (*model.TicketType, error) {
	t, err := s.deps.Tickets.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	e, err := s.deps.Events.FindByID(t.EventID)
	if err != nil {
		return nil, err
	}
	if e.SellerID != sellerID {
		return nil, ErrForbidden
	}
	return t, nil
}

// ListByEvent is public; draft events' types are visible to owner/admin via GetDetail.
func (s *TicketTypeService) ListByEvent(eventID int64) ([]model.TicketType, error) {
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if e.Status != model.EventPublished {
		return nil, ErrNotFound
	}
	return s.deps.Tickets.ListByEvent(eventID)
}

func int64ptr(v int64) *int64 { return &v }
