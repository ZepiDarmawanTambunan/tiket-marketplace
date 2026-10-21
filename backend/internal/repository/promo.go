package repository

import (
	"database/sql"
	"time"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
)

// ---------- Promo codes ----------

type PromoRepository struct{ db *sql.DB }

func NewPromoRepository(db *sql.DB) *PromoRepository { return &PromoRepository{db: db} }

const promoCols = `id,code,type,value,min_spend_cents,quota,per_user_limit,
	starts_at,ends_at,scope,event_id,seller_id,active,created_at`

func scanPromo(p *model.PromoCode, row interface{ Scan(...any) error }) error {
	var quota, perUserLimit sql.NullInt64
	var startsAt, endsAt sql.NullTime
	var eventID sql.NullInt64
	var active int
	err := row.Scan(&p.ID, &p.Code, &p.Type, &p.Value, &p.MinSpendCents,
		&quota, &perUserLimit, &startsAt, &endsAt, &p.Scope, &eventID,
		&p.SellerID, &active, &p.CreatedAt)
	if quota.Valid {
		q := int(quota.Int64)
		p.Quota = &q
	}
	if perUserLimit.Valid {
		l := int(perUserLimit.Int64)
		p.PerUserLimit = &l
	}
	if startsAt.Valid {
		p.StartsAt = &startsAt.Time
	}
	if endsAt.Valid {
		p.EndsAt = &endsAt.Time
	}
	if eventID.Valid {
		e := eventID.Int64
		p.EventID = &e
	}
	p.Active = active != 0
	return err
}

func nullInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func (r *PromoRepository) Create(p *model.PromoCode) (*model.PromoCode, error) {
	res, err := r.db.Exec(`INSERT INTO promo_codes (code,type,value,min_spend_cents,quota,
		per_user_limit,starts_at,ends_at,scope,event_id,seller_id,active)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.Code, p.Type, p.Value, p.MinSpendCents, nullInt(p.Quota),
		nullInt(p.PerUserLimit), nullTimePtr(p.StartsAt), nullTimePtr(p.EndsAt),
		p.Scope, nullInt64(p.EventID), p.SellerID, boolToInt(p.Active))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

func (r *PromoRepository) FindByID(id int64) (*model.PromoCode, error) {
	p := &model.PromoCode{}
	if err := scanPromo(p, r.db.QueryRow("SELECT "+promoCols+" FROM promo_codes WHERE id=?", id)); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PromoRepository) FindByCode(code string) (*model.PromoCode, error) {
	p := &model.PromoCode{}
	if err := scanPromo(p, r.db.QueryRow("SELECT "+promoCols+" FROM promo_codes WHERE code=?", code)); err != nil {
		return nil, err
	}
	return p, nil
}

// List returns promos owned by a seller (sellerID > 0) or all promos (admin,
// sellerID <= 0), with optional active/event filters.
func (r *PromoRepository) List(sellerID int64, active *bool, eventID int64, limit, offset int) ([]model.PromoCode, int64, error) {
	where, args := "WHERE 1=1", []any{}
	if sellerID > 0 {
		where += " AND seller_id=?"
		args = append(args, sellerID)
	}
	if active != nil {
		where += " AND active=?"
		args = append(args, boolToInt(*active))
	}
	if eventID > 0 {
		where += " AND event_id=?"
		args = append(args, eventID)
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM promo_codes "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query("SELECT "+promoCols+" FROM promo_codes "+where+
		" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.PromoCode
	for rows.Next() {
		p := model.PromoCode{}
		if err := scanPromo(&p, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// Update edits a promo; the code itself is immutable (callers must not change it).
func (r *PromoRepository) Update(p *model.PromoCode) error {
	_, err := r.db.Exec(`UPDATE promo_codes SET type=?,value=?,min_spend_cents=?,quota=?,
		per_user_limit=?,starts_at=?,ends_at=?,scope=?,event_id=?,active=?
		WHERE id=?`,
		p.Type, p.Value, p.MinSpendCents, nullInt(p.Quota),
		nullInt(p.PerUserLimit), nullTimePtr(p.StartsAt), nullTimePtr(p.EndsAt),
		p.Scope, nullInt64(p.EventID), boolToInt(p.Active), p.ID)
	return err
}

// Deactivate performs a soft delete.
func (r *PromoRepository) Deactivate(id int64) error {
	_, err := r.db.Exec(`UPDATE promo_codes SET active=0 WHERE id=?`, id)
	return err
}

// UsageCountTx counts total applications of a promo.
func (r *PromoRepository) UsageCountTx(tx *sql.Tx, promoID int64) (int, error) {
	var n int
	err := tx.QueryRow(`SELECT COUNT(*) FROM promo_usages WHERE promo_id=?`, promoID).Scan(&n)
	return n, err
}

// UserUsageCountTx counts applications of a promo by one user.
func (r *PromoRepository) UserUsageCountTx(tx *sql.Tx, promoID, userID int64) (int, error) {
	var n int
	err := tx.QueryRow(`SELECT COUNT(*) FROM promo_usages WHERE promo_id=? AND user_id=?`,
		promoID, userID).Scan(&n)
	return n, err
}

// RecordUsageTx records one promo application.
func (r *PromoRepository) RecordUsageTx(tx *sql.Tx, promoID, userID, orderID int64) error {
	_, err := tx.Exec(`INSERT INTO promo_usages (promo_id,user_id,order_id) VALUES (?,?,?)`,
		promoID, userID, orderID)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullTimeOrNil(t *time.Time) any {
	return nullTimePtr(t)
}
