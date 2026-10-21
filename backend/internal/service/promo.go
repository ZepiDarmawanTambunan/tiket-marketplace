package service

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
)

// ---------- Promo codes ----------

type PromoService struct{ deps *Deps }

func NewPromoService(deps *Deps) *PromoService { return &PromoService{deps: deps} }

// PromoInput carries create/update fields; nil pointers = unchanged on update.
type PromoInput struct {
	Code          string
	Type          string
	Value         float64
	MinSpendCents int64
	Quota         *int
	PerUserLimit  *int
	StartsAt      *time.Time
	EndsAt        *time.Time
	Scope         string
	EventID       *int64
	Active        *bool
}

func normalizeCode(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// validateInput checks the promo fields; eventID is the resolved target event (0 = none).
func (s *PromoService) validateInput(actorID int64, role string, in PromoInput, forUpdate bool) (int64, error) {
	if !forUpdate && in.Code == "" {
		return 0, fmt.Errorf("%w: code wajib diisi", ErrUnprocessable)
	}
	switch in.Type {
	case model.PromoTypePercent:
		if in.Value <= 0 || in.Value > 100 {
			return 0, fmt.Errorf("%w: value percent harus 0-100", ErrUnprocessable)
		}
	case model.PromoTypeFlat:
		if in.Value <= 0 {
			return 0, fmt.Errorf("%w: value flat harus > 0 (rupiah)", ErrUnprocessable)
		}
	default:
		return 0, fmt.Errorf("%w: type harus percent|flat", ErrUnprocessable)
	}
	if in.MinSpendCents < 0 {
		return 0, fmt.Errorf("%w: min_spend_cents tidak boleh negatif", ErrUnprocessable)
	}
	if in.Quota != nil && *in.Quota <= 0 {
		return 0, fmt.Errorf("%w: quota harus > 0", ErrUnprocessable)
	}
	if in.PerUserLimit != nil && *in.PerUserLimit <= 0 {
		return 0, fmt.Errorf("%w: per_user_limit harus > 0", ErrUnprocessable)
	}
	if in.StartsAt != nil && in.EndsAt != nil && !in.EndsAt.After(*in.StartsAt) {
		return 0, fmt.Errorf("%w: ends_at harus setelah starts_at", ErrUnprocessable)
	}
	scope := in.Scope
	if scope == "" {
		scope = model.PromoScopeGlobal
	}
	var eventID int64
	switch scope {
	case model.PromoScopeGlobal:
		if in.EventID != nil {
			return 0, fmt.Errorf("%w: scope global tidak memakai event_id", ErrUnprocessable)
		}
	case model.PromoScopeEvent:
		if in.EventID == nil || *in.EventID <= 0 {
			return 0, fmt.Errorf("%w: scope event wajib event_id", ErrUnprocessable)
		}
		eventID = *in.EventID
		e, err := s.deps.Events.FindByID(eventID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, fmt.Errorf("%w: event tidak ditemukan", ErrUnprocessable)
			}
			return 0, err
		}
		if role != model.RoleSuperAdmin && e.SellerID != actorID {
			return 0, fmt.Errorf("%w: event bukan milik Anda", ErrForbidden)
		}
	default:
		return 0, fmt.Errorf("%w: scope harus global|event", ErrUnprocessable)
	}
	if role != model.RoleSuperAdmin && scope != model.PromoScopeEvent {
		return 0, fmt.Errorf("%w: seller hanya boleh membuat promo scope event miliknya", ErrUnprocessable)
	}
	return eventID, nil
}

// ownedPromo loads a promo and enforces owner-or-admin access.
func (s *PromoService) ownedPromo(actorID int64, role string, id int64) (*model.PromoCode, error) {
	p, err := s.deps.Promos.FindByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if role != model.RoleSuperAdmin && p.SellerID != actorID {
		return nil, ErrForbidden
	}
	return p, nil
}

// Create makes a new promo code. Sellers may only create event-scoped promos
// for their own events; admins may create any scope.
func (s *PromoService) Create(actorID int64, role string, in PromoInput) (*model.PromoCode, error) {
	eventID, err := s.validateInput(actorID, role, in, false)
	if err != nil {
		return nil, err
	}
	code := normalizeCode(in.Code)
	if _, err := s.deps.Promos.FindByCode(code); err == nil {
		return nil, fmt.Errorf("%w: code sudah dipakai", ErrConflict)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	scope := in.Scope
	if scope == "" {
		scope = model.PromoScopeGlobal
	}
	p := &model.PromoCode{
		Code: code, Type: in.Type, Value: in.Value, MinSpendCents: in.MinSpendCents,
		Quota: in.Quota, PerUserLimit: in.PerUserLimit, StartsAt: in.StartsAt,
		EndsAt: in.EndsAt, Scope: scope, SellerID: actorID, Active: true,
	}
	if eventID > 0 {
		p.EventID = &eventID
	}
	if in.Active != nil {
		p.Active = *in.Active
	}
	created, err := s.deps.Promos.Create(p)
	if err != nil {
		return nil, err
	}
	_ = s.deps.Audit.Log(int64ptr(actorID), "promo.create",
		fmt.Sprintf("promo_id=%d code=%s scope=%s", created.ID, created.Code, created.Scope))
	return created, nil
}

// List returns promos: the caller's own for sellers, all for admins.
func (s *PromoService) List(actorID int64, role string, active *bool, eventID int64, page, perPage int) ([]model.PromoCode, int64, error) {
	sellerID := actorID
	if role == model.RoleSuperAdmin {
		sellerID = 0
	}
	return s.deps.Promos.List(sellerID, active, eventID, perPage, (page-1)*perPage)
}

// Update replaces a promo (full-replace semantics, like events); the code is immutable.
func (s *PromoService) Update(actorID int64, role string, id int64, in PromoInput) (*model.PromoCode, error) {
	p, err := s.ownedPromo(actorID, role, id)
	if err != nil {
		return nil, err
	}
	if in.Code != "" && normalizeCode(in.Code) != p.Code {
		return nil, fmt.Errorf("%w: code tidak bisa diubah", ErrUnprocessable)
	}
	if in.Type == "" {
		in.Type = p.Type
	}
	if in.Value == 0 {
		in.Value = p.Value
	}
	if in.Scope == "" {
		in.Scope = p.Scope
	}
	if in.EventID == nil {
		in.EventID = p.EventID
	}
	if in.Active == nil {
		in.Active = &p.Active
	}
	eventID, err := s.validateInput(actorID, role, in, true)
	if err != nil {
		return nil, err
	}
	p.Type = in.Type
	p.Value = in.Value
	p.MinSpendCents = in.MinSpendCents
	p.Quota = in.Quota
	p.PerUserLimit = in.PerUserLimit
	p.StartsAt = in.StartsAt
	p.EndsAt = in.EndsAt
	p.Scope = in.Scope
	p.EventID = nil
	if eventID > 0 {
		p.EventID = &eventID
	}
	p.Active = *in.Active
	if err := s.deps.Promos.Update(p); err != nil {
		return nil, err
	}
	_ = s.deps.Audit.Log(int64ptr(actorID), "promo.update",
		fmt.Sprintf("promo_id=%d code=%s", p.ID, p.Code))
	return p, nil
}

// Delete performs a soft delete (active=false).
func (s *PromoService) Delete(actorID int64, role string, id int64) error {
	p, err := s.ownedPromo(actorID, role, id)
	if err != nil {
		return err
	}
	if err := s.deps.Promos.Deactivate(p.ID); err != nil {
		return err
	}
	_ = s.deps.Audit.Log(int64ptr(actorID), "promo.deactivate",
		fmt.Sprintf("promo_id=%d code=%s", p.ID, p.Code))
	return nil
}

// PromoValidation is the result of validating a code against a cart.
type PromoValidation struct {
	Valid         bool   `json:"valid"`
	DiscountCents int64  `json:"discount_cents"`
	Message       string `json:"message"`
	Code          string `json:"code"`
}

// checkPromoTx validates a code against an event cart inside a transaction and
// returns the discount in rupiah.
func (s *PromoService) checkPromoTx(tx *sql.Tx, userID, eventID int64, code string, subtotal int64) (*model.PromoCode, int64, error) {
	p, err := s.deps.Promos.FindByCode(normalizeCode(code))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, fmt.Errorf("%w: kode promo tidak ditemukan", ErrUnprocessable)
		}
		return nil, 0, err
	}
	if !p.Active {
		return nil, 0, fmt.Errorf("%w: kode promo tidak aktif", ErrUnprocessable)
	}
	now := time.Now()
	if p.StartsAt != nil && now.Before(*p.StartsAt) {
		return nil, 0, fmt.Errorf("%w: kode promo belum berlaku", ErrUnprocessable)
	}
	if p.EndsAt != nil && now.After(*p.EndsAt) {
		return nil, 0, fmt.Errorf("%w: kode promo sudah kedaluwarsa", ErrUnprocessable)
	}
	if p.Scope == model.PromoScopeEvent && (p.EventID == nil || *p.EventID != eventID) {
		return nil, 0, fmt.Errorf("%w: kode promo tidak berlaku untuk event ini", ErrUnprocessable)
	}
	if subtotal < p.MinSpendCents {
		return nil, 0, fmt.Errorf("%w: minimal belanja Rp %d untuk kode ini",
			ErrUnprocessable, p.MinSpendCents)
	}
	if p.Quota != nil {
		n, err := s.deps.Promos.UsageCountTx(tx, p.ID)
		if err != nil {
			return nil, 0, err
		}
		if n >= *p.Quota {
			return nil, 0, fmt.Errorf("%w: kuota kode promo habis", ErrUnprocessable)
		}
	}
	if p.PerUserLimit != nil {
		n, err := s.deps.Promos.UserUsageCountTx(tx, p.ID, userID)
		if err != nil {
			return nil, 0, err
		}
		if n >= *p.PerUserLimit {
			return nil, 0, fmt.Errorf("%w: batas pakai kode ini untuk akun Anda tercapai",
				ErrUnprocessable)
		}
	}
	return p, discountFor(p, subtotal), nil
}

// discountFor computes the rupiah discount for a subtotal, capped at subtotal.
func discountFor(p *model.PromoCode, subtotal int64) int64 {
	var d int64
	switch p.Type {
	case model.PromoTypePercent:
		d = int64(math.Round(float64(subtotal) * p.Value / 100))
	default:
		d = int64(math.Round(p.Value))
	}
	if d > subtotal {
		d = subtotal
	}
	if d < 0 {
		d = 0
	}
	return d
}

// Validate checks a code against a cart without creating anything.
// It always returns 200-style data: valid=false carries the reason in message.
func (s *PromoService) Validate(userID, eventID int64, code string, items []model.OrderItem) (*PromoValidation, error) {
	code = normalizeCode(code)
	if code == "" {
		return &PromoValidation{Valid: false, Message: "code wajib diisi", Code: code}, nil
	}
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if e.Status != model.EventPublished {
		return &PromoValidation{Valid: false, Message: "event belum tayang", Code: code}, nil
	}
	var subtotal int64
	for _, it := range items {
		tt, err := s.deps.Tickets.FindByID(it.TicketTypeID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return &PromoValidation{Valid: false,
					Message: fmt.Sprintf("ticket_type %d tidak ditemukan", it.TicketTypeID), Code: code}, nil
			}
			return nil, err
		}
		if tt.EventID != eventID || it.Qty <= 0 {
			return &PromoValidation{Valid: false, Message: "item tidak valid untuk event ini", Code: code}, nil
		}
		subtotal += tt.PriceCents * int64(it.Qty)
	}
	tx, err := s.deps.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, discount, err := s.checkPromoTx(tx, userID, eventID, code, subtotal)
	if err != nil {
		if errors.Is(err, ErrUnprocessable) {
			return &PromoValidation{Valid: false, Message: trimSentinel(err), Code: code}, nil
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &PromoValidation{Valid: true, DiscountCents: discount,
		Message: fmt.Sprintf("kode %s valid: hemat Rp %d", p.Code, discount), Code: p.Code}, nil
}

// RecordUsageTx records one promo application inside an order-creation transaction.
func (s *PromoService) RecordUsageTx(tx *sql.Tx, promoID, userID, orderID int64) error {
	return s.deps.Promos.RecordUsageTx(tx, promoID, userID, orderID)
}

// trimSentinel strips the "unprocessable: " prefix for user-facing messages.
func trimSentinel(err error) string {
	s := err.Error()
	const p = "unprocessable: "
	if len(s) > len(p) && s[:len(p)] == p {
		return s[len(p):]
	}
	return s
}
