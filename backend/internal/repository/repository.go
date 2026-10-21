// Package repository provides MySQL-backed data access for the marketplace.
package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
)

// Open connects to MySQL and verifies the connection.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

// ---------- Users ----------

type UserRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

func scanUser(u *model.User, row interface{ Scan(...any) error }) error {
	var phone, bankName, bankAccount sql.NullString
	err := row.Scan(&u.ID, &u.Name, &u.Email, &phone, &u.PasswordHash, &u.Role,
		&u.SellerVerified, &bankName, &bankAccount, &u.CreatedAt)
	u.Phone = phone.String
	u.BankName = bankName.String
	u.BankAccount = bankAccount.String
	return err
}

func (r *UserRepository) Create(name, email, phone, passwordHash, role string) (*model.User, error) {
	res, err := r.db.Exec(`INSERT INTO users (name, email, phone, password_hash, role)
		VALUES (?,?,?,?,?)`, name, email, nullIfEmpty(phone), passwordHash, role)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

func (r *UserRepository) FindByID(id int64) (*model.User, error) {
	u := &model.User{}
	err := scanUser(u, r.db.QueryRow(`SELECT id,name,email,phone,password_hash,role,
		seller_verified,bank_name,bank_account,created_at FROM users WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	u := &model.User{}
	err := scanUser(u, r.db.QueryRow(`SELECT id,name,email,phone,password_hash,role,
		seller_verified,bank_name,bank_account,created_at FROM users WHERE email=?`, email))
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) List(role, search string, limit, offset int) ([]model.User, int64, error) {
	where, args := "WHERE 1=1", []any{}
	if role != "" {
		where += " AND role=?"
		args = append(args, role)
	}
	if search != "" {
		where += " AND (name LIKE ? OR email LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM users "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`SELECT id,name,email,phone,password_hash,role,
		seller_verified,bank_name,bank_account,created_at FROM users `+where+
		` ORDER BY id DESC LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.User
	for rows.Next() {
		u := model.User{}
		if err := scanUser(&u, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func (r *UserRepository) UpdateRole(id int64, role string) error {
	_, err := r.db.Exec(`UPDATE users SET role=? WHERE id=?`, role, id)
	return err
}

func (r *UserRepository) VerifySeller(id int64, verified bool, bankName, bankAccount string) error {
	_, err := r.db.Exec(`UPDATE users SET seller_verified=?, bank_name=?, bank_account=?
		WHERE id=?`, verified, nullIfEmpty(bankName), nullIfEmpty(bankAccount), id)
	return err
}

// ---------- Events ----------

type EventRepository struct{ db *sql.DB }

func NewEventRepository(db *sql.DB) *EventRepository { return &EventRepository{db: db} }

func scanEvent(e *model.Event, row interface{ Scan(...any) error }) error {
	var desc, category, venue, address, mapsURL, bannerURL, refundPolicy sql.NullString
	err := row.Scan(&e.ID, &e.SellerID, &e.Title, &desc, &category, &venue, &address,
		&mapsURL, &e.StartsAt, &e.EndsAt, &bannerURL, &e.Status, &refundPolicy, &e.CreatedAt)
	e.Description = desc.String
	e.Category = category.String
	e.Venue = venue.String
	e.Address = address.String
	e.MapsURL = mapsURL.String
	e.BannerURL = bannerURL.String
	e.RefundPolicy = refundPolicy.String
	return err
}

func (r *EventRepository) Create(e *model.Event) (*model.Event, error) {
	res, err := r.db.Exec(`INSERT INTO events (seller_id,title,description,category,venue,address,
		maps_url,starts_at,ends_at,banner_url,status,refund_policy)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.SellerID, e.Title, nullIfEmpty(e.Description), nullIfEmpty(e.Category),
		nullIfEmpty(e.Venue), nullIfEmpty(e.Address), nullIfEmpty(e.MapsURL),
		e.StartsAt, e.EndsAt, nullIfEmpty(e.BannerURL), model.EventDraft, nullIfEmpty(e.RefundPolicy))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

func (r *EventRepository) FindByID(id int64) (*model.Event, error) {
	e := &model.Event{}
	err := scanEvent(e, r.db.QueryRow(`SELECT id,seller_id,title,description,category,venue,address,
		maps_url,starts_at,ends_at,banner_url,status,refund_policy,created_at
		FROM events WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	return e, nil
}

// TitlesByIDs batch-loads event titles for many events in one query.
func (r *EventRepository) TitlesByIDs(ids []int64) (map[int64]string, error) {
	out := make(map[int64]string)
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := r.db.Query(`SELECT id,title FROM events WHERE id IN (`+
		strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		out[id] = title
	}
	return out, rows.Err()
}

type EventFilter struct {
	Category, Search, From, To string
	Status                     string // "" = any status
	SellerID                   int64  // 0 = any; -1 = ignore (public only published)
	OnlyPublished              bool
	Limit, Offset              int
}

func (r *EventRepository) List(f EventFilter) ([]model.Event, int64, error) {
	where, args := "WHERE 1=1", []any{}
	if f.OnlyPublished {
		where += " AND status='published'"
	}
	if f.Category != "" {
		where += " AND category=?"
		args = append(args, f.Category)
	}
	if f.Search != "" {
		where += " AND (title LIKE ? OR venue LIKE ? OR description LIKE ?)"
		like := "%" + f.Search + "%"
		args = append(args, like, like, like)
	}
	if f.From != "" {
		where += " AND starts_at >= ?"
		args = append(args, f.From)
	}
	if f.To != "" {
		where += " AND starts_at < DATE_ADD(?, INTERVAL 1 DAY)"
		args = append(args, f.To)
	}
	if f.SellerID > 0 {
		where += " AND seller_id=?"
		args = append(args, f.SellerID)
	}
	if f.Status != "" {
		where += " AND status=?"
		args = append(args, f.Status)
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM events "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`SELECT id,seller_id,title,description,category,venue,address,
		maps_url,starts_at,ends_at,banner_url,status,refund_policy,created_at
		FROM events `+where+` ORDER BY starts_at ASC LIMIT ? OFFSET ?`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Event
	for rows.Next() {
		e := model.Event{}
		if err := scanEvent(&e, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func (r *EventRepository) Update(e *model.Event) error {
	_, err := r.db.Exec(`UPDATE events SET title=?,description=?,category=?,venue=?,address=?,
		maps_url=?,starts_at=?,ends_at=?,banner_url=?,refund_policy=? WHERE id=?`,
		e.Title, nullIfEmpty(e.Description), nullIfEmpty(e.Category), nullIfEmpty(e.Venue),
		nullIfEmpty(e.Address), nullIfEmpty(e.MapsURL), e.StartsAt, e.EndsAt,
		nullIfEmpty(e.BannerURL), nullIfEmpty(e.RefundPolicy), e.ID)
	return err
}

func (r *EventRepository) UpdateStatus(id int64, status string) error {
	_, err := r.db.Exec(`UPDATE events SET status=? WHERE id=?`, status, id)
	return err
}

func (r *EventRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM events WHERE id=?`, id)
	return err
}

// ---------- Ticket types ----------

type TicketTypeRepository struct{ db *sql.DB }

func NewTicketTypeRepository(db *sql.DB) *TicketTypeRepository { return &TicketTypeRepository{db: db} }

func scanTicketType(t *model.TicketType, row interface{ Scan(...any) error }) error {
	var saleStarts, saleEnds sql.NullTime
	err := row.Scan(&t.ID, &t.EventID, &t.Name, &t.PriceCents, &t.Quota, &t.Sold, &saleStarts, &saleEnds)
	if saleStarts.Valid {
		t.SaleStartsAt = saleStarts.Time
	}
	if saleEnds.Valid {
		t.SaleEndsAt = saleEnds.Time
	}
	return err
}

func (r *TicketTypeRepository) Create(t *model.TicketType) (*model.TicketType, error) {
	res, err := r.db.Exec(`INSERT INTO ticket_types (event_id,name,price_cents,quota,sale_starts_at,sale_ends_at)
		VALUES (?,?,?,?,?,?)`, t.EventID, t.Name, t.PriceCents, t.Quota,
		nullTime(t.SaleStartsAt), nullTime(t.SaleEndsAt))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

func (r *TicketTypeRepository) FindByID(id int64) (*model.TicketType, error) {
	t := &model.TicketType{}
	err := scanTicketType(t, r.db.QueryRow(`SELECT id,event_id,name,price_cents,quota,sold,
		sale_starts_at,sale_ends_at FROM ticket_types WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TicketTypeRepository) ListByEvent(eventID int64) ([]model.TicketType, error) {
	rows, err := r.db.Query(`SELECT id,event_id,name,price_cents,quota,sold,
		sale_starts_at,sale_ends_at FROM ticket_types WHERE event_id=? ORDER BY price_cents ASC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.TicketType
	for rows.Next() {
		t := model.TicketType{}
		if err := scanTicketType(&t, rows); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListByEventIDs batch-loads ticket types for many events in one query,
// grouped by event ID (used by event list endpoints).
func (r *TicketTypeRepository) ListByEventIDs(ids []int64) (map[int64][]model.TicketType, error) {
	out := make(map[int64][]model.TicketType)
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := r.db.Query(`SELECT id,event_id,name,price_cents,quota,sold,
		sale_starts_at,sale_ends_at FROM ticket_types WHERE event_id IN (`+
		strings.Join(placeholders, ",")+`) ORDER BY price_cents ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		t := model.TicketType{}
		if err := scanTicketType(&t, rows); err != nil {
			return nil, err
		}
		out[t.EventID] = append(out[t.EventID], t)
	}
	return out, rows.Err()
}

func (r *TicketTypeRepository) Update(t *model.TicketType) error {
	_, err := r.db.Exec(`UPDATE ticket_types SET name=?,price_cents=?,quota=?,
		sale_starts_at=?,sale_ends_at=? WHERE id=?`, t.Name, t.PriceCents, t.Quota,
		nullTime(t.SaleStartsAt), nullTime(t.SaleEndsAt), t.ID)
	return err
}

func (r *TicketTypeRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM ticket_types WHERE id=?`, id)
	return err
}

// ---------- Orders ----------

type OrderRepository struct{ db *sql.DB }

func NewOrderRepository(db *sql.DB) *OrderRepository { return &OrderRepository{db: db} }

// CreateTx inserts the order header inside an existing transaction.
func (r *OrderRepository) CreateTx(tx *sql.Tx, o *model.Order) (int64, error) {
	res, err := tx.Exec(`INSERT INTO orders (order_code,user_id,event_id,subtotal_cents,
		discount_cents,promo_code,commission_cents,gateway_fee_cents,gateway_fee_bearer,
		gross_cents,status,snap_token)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.OrderCode, o.UserID, o.EventID, o.SubtotalCents, o.DiscountCents,
		nullIfEmpty(o.PromoCode), o.CommissionCents,
		o.GatewayFeeCents, o.GatewayFeeBearer, o.GrossCents, o.Status, nullIfEmpty(o.SnapToken))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func scanOrder(o *model.Order, row interface{ Scan(...any) error }) error {
	var snap, promo sql.NullString
	err := row.Scan(&o.ID, &o.OrderCode, &o.UserID, &o.EventID, &o.SubtotalCents,
		&o.DiscountCents, &promo, &o.CommissionCents, &o.GatewayFeeCents,
		&o.GatewayFeeBearer, &o.GrossCents,
		&o.Status, &snap, &o.CreatedAt)
	o.SnapToken = snap.String
	o.PromoCode = promo.String
	return err
}

const orderCols = `id,order_code,user_id,event_id,subtotal_cents,discount_cents,promo_code,
	commission_cents,gateway_fee_cents,gateway_fee_bearer,gross_cents,status,snap_token,created_at`

func (r *OrderRepository) FindByID(id int64) (*model.Order, error) {
	o := &model.Order{}
	err := scanOrder(o, r.db.QueryRow("SELECT "+orderCols+" FROM orders WHERE id=?", id))
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (r *OrderRepository) FindByCode(code string) (*model.Order, error) {
	o := &model.Order{}
	err := scanOrder(o, r.db.QueryRow("SELECT "+orderCols+" FROM orders WHERE order_code=?", code))
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (r *OrderRepository) ListByUser(userID int64, status string, limit, offset int) ([]model.Order, int64, error) {
	where, args := "WHERE user_id=?", []any{userID}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM orders "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query("SELECT "+orderCols+" FROM orders "+where+
		" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Order
	for rows.Next() {
		o := model.Order{}
		if err := scanOrder(&o, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *OrderRepository) ListBySeller(sellerID int64, status string, limit, offset int) ([]model.Order, int64, error) {
	where, args := "WHERE e.seller_id=?", []any{sellerID}
	if status != "" {
		where += " AND o.status=?"
		args = append(args, status)
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM orders o JOIN events e ON e.id=o.event_id "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`SELECT o.id,o.order_code,o.user_id,o.event_id,o.subtotal_cents,
		o.discount_cents,o.promo_code,o.commission_cents,o.gateway_fee_cents,
		o.gateway_fee_bearer,o.gross_cents,o.status,
		o.snap_token,o.created_at FROM orders o JOIN events e ON e.id=o.event_id `+where+
		" ORDER BY o.id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Order
	for rows.Next() {
		o := model.Order{}
		if err := scanOrder(&o, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *OrderRepository) ListAll(status, from, to string, limit, offset int) ([]model.Order, int64, error) {
	where, args := "WHERE 1=1", []any{}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if from != "" {
		where += " AND created_at >= ?"
		args = append(args, from)
	}
	if to != "" {
		where += " AND created_at < DATE_ADD(?, INTERVAL 1 DAY)"
		args = append(args, to)
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM orders "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query("SELECT "+orderCols+" FROM orders "+where+
		" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Order
	for rows.Next() {
		o := model.Order{}
		if err := scanOrder(&o, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, total, rows.Err()
}

func (r *OrderRepository) UpdateStatusTx(tx *sql.Tx, id int64, status string) error {
	_, err := tx.Exec(`UPDATE orders SET status=? WHERE id=?`, status, id)
	return err
}

func (r *OrderRepository) UpdateStatus(id int64, status string) error {
	_, err := r.db.Exec(`UPDATE orders SET status=? WHERE id=?`, status, id)
	return err
}

func (r *OrderRepository) UpdateSnapToken(id int64, token string) error {
	_, err := r.db.Exec(`UPDATE orders SET snap_token=? WHERE id=?`, nullIfEmpty(token), id)
	return err
}

func (r *OrderRepository) HasOrdersForEvent(eventID int64) (bool, error) {
	var n int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE event_id=?`, eventID).Scan(&n)
	return n > 0, err
}

// ---------- Order items / tickets ----------

type TicketRepository struct{ db *sql.DB }

func NewTicketRepository(db *sql.DB) *TicketRepository { return &TicketRepository{db: db} }

// InsertTx creates concrete ticket rows inside a transaction.
func (r *TicketRepository) InsertTx(tx *sql.Tx, orderID, ticketTypeID int64, ticketCode, qrPayload string, priceCents int64) error {
	_, err := tx.Exec(`INSERT INTO order_items (order_id,ticket_type_id,ticket_code,qr_payload,price_cents)
		VALUES (?,?,?,?,?)`, orderID, ticketTypeID, ticketCode, qrPayload, priceCents)
	return err
}

type TicketLookup struct {
	ItemID         int64
	TicketCode     string
	QRPayLoad      string
	UsedAt         sql.NullTime
	PriceCents     int64
	EventID        int64
	EventSellerID  int64
	EventTitle     string
	EventStartsAt  time.Time
	TicketTypeName string
	HolderUserID   int64
	HolderName     string
	OrderCode      string
	OrderStatus    string
}

const ticketLookupCols = `
		SELECT oi.id, oi.ticket_code, oi.qr_payload, oi.used_at, oi.price_cents,
		       o.event_id, e.seller_id, e.title, e.starts_at, tt.name,
		       COALESCE(oi.holder_user_id, o.user_id), h.name, o.order_code, o.status
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN events e ON e.id = o.event_id
		JOIN ticket_types tt ON tt.id = oi.ticket_type_id
		JOIN users h ON h.id = COALESCE(oi.holder_user_id, o.user_id)`

func (r *TicketRepository) FindByCode(code string) (*TicketLookup, error) {
	t := &TicketLookup{}
	err := r.db.QueryRow(ticketLookupCols+` WHERE oi.ticket_code=?`, code).Scan(
		&t.ItemID, &t.TicketCode, &t.QRPayLoad, &t.UsedAt, &t.PriceCents,
		&t.EventID, &t.EventSellerID, &t.EventTitle, &t.EventStartsAt, &t.TicketTypeName,
		&t.HolderUserID, &t.HolderName, &t.OrderCode, &t.OrderStatus)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TicketRepository) FindByQRPayload(payload string) (*TicketLookup, error) {
	t := &TicketLookup{}
	err := r.db.QueryRow(ticketLookupCols+` WHERE oi.qr_payload=?`, payload).Scan(
		&t.ItemID, &t.TicketCode, &t.QRPayLoad, &t.UsedAt, &t.PriceCents,
		&t.EventID, &t.EventSellerID, &t.EventTitle, &t.EventStartsAt, &t.TicketTypeName,
		&t.HolderUserID, &t.HolderName, &t.OrderCode, &t.OrderStatus)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// MarkUsedTx atomically marks a ticket used only if it is still unused.
// Returns true when the ticket was marked, false when it was already used.
func (r *TicketRepository) MarkUsedTx(tx *sql.Tx, itemID int64) (bool, error) {
	res, err := tx.Exec(`UPDATE order_items SET used_at=NOW() WHERE id=? AND used_at IS NULL`, itemID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r *TicketRepository) ListByUser(userID int64) ([]model.Ticket, error) {
	rows, err := r.db.Query(`
		SELECT oi.id, oi.ticket_code, oi.qr_payload, e.title, e.starts_at, tt.name,
		       oi.price_cents, o.order_code, h.id, h.name, oi.used_at
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN events e ON e.id = o.event_id
		JOIN ticket_types tt ON tt.id = oi.ticket_type_id
		JOIN users h ON h.id = COALESCE(oi.holder_user_id, o.user_id)
		WHERE COALESCE(oi.holder_user_id, o.user_id)=? AND o.status='paid'
		ORDER BY oi.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Ticket
	for rows.Next() {
		t := model.Ticket{}
		var used sql.NullTime
		var holderID int64
		if err := rows.Scan(&t.ID, &t.TicketCode, &t.QRPayLoad, &t.EventTitle, &t.EventStartsAt,
			&t.TicketTypeName, &t.PriceCents, &t.OrderCode, &holderID, &t.HolderName, &used); err != nil {
			return nil, err
		}
		t.HolderUserID = &holderID
		t.Status = "unused"
		if used.Valid {
			t.Status = "used"
			t.UsedAt = &used.Time
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TransferTx changes the ticket holder inside a transaction; the ticket must
// still be unused (checked by the caller under lock).
func (r *TicketRepository) TransferTx(tx *sql.Tx, itemID, toUserID int64) error {
	_, err := tx.Exec(`UPDATE order_items SET holder_user_id=? WHERE id=? AND used_at IS NULL`,
		toUserID, itemID)
	return err
}

// ---------- Payments ----------

type PaymentRepository struct{ db *sql.DB }

func NewPaymentRepository(db *sql.DB) *PaymentRepository { return &PaymentRepository{db: db} }

// InsertTx records a notification; returns false when it is a duplicate
// (idempotency on midtrans_order_id+transaction_id+transaction_status).
func (r *PaymentRepository) InsertTx(tx *sql.Tx, p *model.Payment, raw string) (bool, error) {
	res, err := tx.Exec(`INSERT IGNORE INTO payments (order_id,midtrans_order_id,transaction_id,
		payment_type,transaction_status,gross_amount,raw_notification)
		VALUES (?,?,?,?,?,?,?)`,
		p.OrderID, p.MidtransOrderID, p.TransactionID, nullIfEmpty(p.PaymentType),
		p.TransactionStatus, p.GrossAmount, nullIfEmpty(raw))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// ---------- Order lines (persisted checkout items) ----------

type OrderLineRepository struct{ db *sql.DB }

func NewOrderLineRepository(db *sql.DB) *OrderLineRepository { return &OrderLineRepository{db: db} }

// OrderLine is a persisted checkout line: how many of which ticket type.
type OrderLine struct {
	ID           int64
	OrderID      int64
	TicketTypeID int64
	Qty          int
	PriceCents   int64
}

func (r *OrderLineRepository) InsertTx(tx *sql.Tx, orderID, ticketTypeID int64, qty int, priceCents int64) error {
	_, err := tx.Exec(`INSERT INTO order_lines (order_id,ticket_type_id,qty,price_cents)
		VALUES (?,?,?,?)`, orderID, ticketTypeID, qty, priceCents)
	return err
}

func (r *OrderLineRepository) ListByOrderTx(tx *sql.Tx, orderID int64) ([]OrderLine, error) {
	rows, err := tx.Query(`SELECT id,order_id,ticket_type_id,qty,price_cents
		FROM order_lines WHERE order_id=?`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderLine
	for rows.Next() {
		var l OrderLine
		if err := rows.Scan(&l.ID, &l.OrderID, &l.TicketTypeID, &l.Qty, &l.PriceCents); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ListByOrderIDs batch-loads order lines for many orders in one query,
// grouped by order ID, with ticket type names joined in.
func (r *OrderLineRepository) ListByOrderIDs(ids []int64) (map[int64][]model.OrderItem, error) {
	out := make(map[int64][]model.OrderItem)
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := r.db.Query(`SELECT ol.order_id, ol.ticket_type_id, COALESCE(tt.name,''), ol.qty, ol.price_cents
		FROM order_lines ol
		LEFT JOIN ticket_types tt ON tt.id = ol.ticket_type_id
		WHERE ol.order_id IN (`+strings.Join(placeholders, ",")+`) ORDER BY ol.id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var orderID int64
		var it model.OrderItem
		if err := rows.Scan(&orderID, &it.TicketTypeID, &it.TicketTypeName, &it.Qty, &it.PriceCents); err != nil {
			return nil, err
		}
		out[orderID] = append(out[orderID], it)
	}
	return out, rows.Err()
}

// ---------- Refunds ----------

type RefundRepository struct{ db *sql.DB }

func NewRefundRepository(db *sql.DB) *RefundRepository { return &RefundRepository{db: db} }

func scanRefund(f *model.Refund, row interface{ Scan(...any) error }) error {
	var handled sql.NullInt64
	var midtransRefundID, rawResponse, note sql.NullString
	err := row.Scan(&f.ID, &f.OrderID, &f.OrderCode, &f.AmountCents, &f.Reason,
		&f.Status, &handled, &midtransRefundID, &rawResponse, &note, &f.CreatedAt)
	if handled.Valid {
		f.HandledBy = &handled.Int64
	}
	f.MidtransRefundID = midtransRefundID.String
	f.RawResponse = rawResponse.String
	f.Note = note.String
	return err
}

const refundCols = `r.id,r.order_id,o.order_code,r.amount_cents,r.reason,
	r.status,r.handled_by,r.midtrans_refund_id,r.raw_response,r.note,r.created_at`

func (r *RefundRepository) Create(orderID, amountCents int64, reason string) (*model.Refund, error) {
	res, err := r.db.Exec(`INSERT INTO refunds (order_id,amount_cents,reason,status)
		VALUES (?,?,?,'requested')`, orderID, amountCents, reason)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

func (r *RefundRepository) FindByID(id int64) (*model.Refund, error) {
	f := &model.Refund{}
	err := scanRefund(f, r.db.QueryRow(`SELECT `+refundCols+` FROM refunds r
		JOIN orders o ON o.id=r.order_id WHERE r.id=?`, id))
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (r *RefundRepository) FindPendingByOrder(orderID int64) (*model.Refund, error) {
	f := &model.Refund{}
	err := scanRefund(f, r.db.QueryRow(`SELECT `+refundCols+` FROM refunds r
		JOIN orders o ON o.id=r.order_id
		WHERE r.order_id=? AND r.status='requested' ORDER BY r.id DESC LIMIT 1`, orderID))
	if err != nil {
		return nil, err
	}
	return f, nil
}

// List returns refunds for the admin console, newest first.
func (r *RefundRepository) List(status string, limit, offset int) ([]model.Refund, int64, error) {
	where, args := "", []any{}
	if status != "" {
		where, args = "WHERE r.status=?", []any{status}
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM refunds r "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`SELECT `+refundCols+` FROM refunds r
		JOIN orders o ON o.id=r.order_id `+where+`
		ORDER BY r.id DESC LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []model.Refund{}
	for rows.Next() {
		f := model.Refund{}
		if err := scanRefund(&f, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, f)
	}
	return out, total, rows.Err()
}

func (r *RefundRepository) UpdateStatusTx(tx *sql.Tx, id int64, status string, handledBy int64) error {
	_, err := tx.Exec(`UPDATE refunds SET status=?, handled_by=? WHERE id=?`, status, handledBy, id)
	return err
}

// ApproveTx records an approved refund together with the Midtrans outcome.
func (r *RefundRepository) ApproveTx(tx *sql.Tx, id, handledBy int64, amountCents int64,
	reason, midtransRefundID, rawResponse, note string) error {
	_, err := tx.Exec(`UPDATE refunds SET status='approved', handled_by=?, amount_cents=?,
		reason=?, midtrans_refund_id=?, raw_response=?, note=? WHERE id=?`,
		handledBy, amountCents, nullIfEmpty(reason),
		nullIfEmpty(midtransRefundID), nullIfEmpty(rawResponse), nullIfEmpty(note), id)
	return err
}

// ---------- Payouts ----------

type PayoutRepository struct{ db *sql.DB }

func NewPayoutRepository(db *sql.DB) *PayoutRepository { return &PayoutRepository{db: db} }

const payoutCols = `id,seller_id,event_id,amount_cents,status,method,
	iris_reference_no,iris_status,proof_url,created_at`

func scanPayout(p *model.Payout, row interface{ Scan(...any) error }) error {
	var proof, irisRef, irisStatus sql.NullString
	err := row.Scan(&p.ID, &p.SellerID, &p.EventID, &p.AmountCents, &p.Status, &p.Method,
		&irisRef, &irisStatus, &proof, &p.CreatedAt)
	p.ProofURL = proof.String
	p.IrisReferenceNo = irisRef.String
	p.IrisStatus = irisStatus.String
	return err
}

func (r *PayoutRepository) Create(sellerID, eventID, amountCents int64, method string) (*model.Payout, error) {
	if method == "" {
		method = model.PayoutMethodManual
	}
	res, err := r.db.Exec(`INSERT INTO payouts (seller_id,event_id,amount_cents,status,method)
		VALUES (?,?,?,'pending',?)`, sellerID, eventID, amountCents, method)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

func (r *PayoutRepository) FindByID(id int64) (*model.Payout, error) {
	p := &model.Payout{}
	err := scanPayout(p, r.db.QueryRow("SELECT "+payoutCols+" FROM payouts WHERE id=?", id))
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PayoutRepository) List(status string, sellerID int64, limit, offset int) ([]model.Payout, int64, error) {
	where, args := "WHERE 1=1", []any{}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if sellerID > 0 {
		where += " AND seller_id=?"
		args = append(args, sellerID)
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM payouts "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query("SELECT "+payoutCols+" FROM payouts "+where+
		" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Payout
	for rows.Next() {
		p := model.Payout{}
		if err := scanPayout(&p, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// MarkPaid marks a pending/processing payout as paid with transfer proof.
func (r *PayoutRepository) MarkPaid(id int64, proofURL string) error {
	_, err := r.db.Exec(`UPDATE payouts SET status='paid', proof_url=?
		WHERE id=? AND status IN ('pending','processing')`, nullIfEmpty(proofURL), id)
	return err
}

// MarkIrisSent records an Iris disbursement and moves the payout to processing.
func (r *PayoutRepository) MarkIrisSent(id int64, referenceNo, irisStatus string) error {
	_, err := r.db.Exec(`UPDATE payouts SET method='iris', iris_reference_no=?,
		iris_status=?, status='processing' WHERE id=? AND status='pending'`,
		nullIfEmpty(referenceNo), nullIfEmpty(irisStatus), id)
	return err
}

// UpdateIrisStatus refreshes the Iris-side status and optionally the payout status.
func (r *PayoutRepository) UpdateIrisStatus(id int64, irisStatus, payoutStatus string) error {
	if payoutStatus == "" {
		_, err := r.db.Exec(`UPDATE payouts SET iris_status=? WHERE id=?`,
			nullIfEmpty(irisStatus), id)
		return err
	}
	_, err := r.db.Exec(`UPDATE payouts SET iris_status=?, status=? WHERE id=?`,
		nullIfEmpty(irisStatus), payoutStatus, id)
	return err
}

// ---------- Settings ----------

type SettingsRepository struct{ db *sql.DB }

func NewSettingsRepository(db *sql.DB) *SettingsRepository { return &SettingsRepository{db: db} }

func (r *SettingsRepository) Get(key string) (string, bool, error) {
	var v sql.NullString
	err := r.db.QueryRow(`SELECT value_encrypted FROM settings WHERE `+"`key`"+`=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v.String, true, nil
}

func (r *SettingsRepository) Set(key, encryptedValue string) error {
	_, err := r.db.Exec(`INSERT INTO settings (`+"`key`"+`, value_encrypted) VALUES (?,?)
		ON DUPLICATE KEY UPDATE value_encrypted=VALUES(value_encrypted)`, key, nullIfEmpty(encryptedValue))
	return err
}

func (r *SettingsRepository) All() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT ` + "`key`" + `, value_encrypted FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k string
		var v sql.NullString
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v.String
	}
	return out, rows.Err()
}

// ---------- Audit logs ----------

type AuditRepository struct{ db *sql.DB }

func NewAuditRepository(db *sql.DB) *AuditRepository { return &AuditRepository{db: db} }

func (r *AuditRepository) Log(actorID *int64, action, detail string) error {
	_, err := r.db.Exec(`INSERT INTO audit_logs (actor_id,action,detail) VALUES (?,?,?)`,
		actorID, action, detail)
	return err
}

func (r *AuditRepository) List(action string, actorID int64, limit, offset int) ([]model.AuditLog, int64, error) {
	where, args := "WHERE 1=1", []any{}
	if action != "" {
		where += " AND action=?"
		args = append(args, action)
	}
	if actorID > 0 {
		where += " AND actor_id=?"
		args = append(args, actorID)
	}
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM audit_logs "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`SELECT l.id,l.actor_id,COALESCE(u.name,''),l.action,l.detail,l.created_at
		FROM audit_logs l LEFT JOIN users u ON u.id=l.actor_id `+where+
		` ORDER BY l.id DESC LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.AuditLog
	for rows.Next() {
		l := model.AuditLog{}
		var actor sql.NullInt64
		if err := rows.Scan(&l.ID, &actor, &l.ActorName, &l.Action, &l.Detail, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		if actor.Valid {
			l.ActorID = &actor.Int64
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

// ---------- Reports ----------

type ReportRepository struct{ db *sql.DB }

func NewReportRepository(db *sql.DB) *ReportRepository { return &ReportRepository{db: db} }

func (r *ReportRepository) Summary(from, to string) (*model.ReportSummary, error) {
	s := &model.ReportSummary{From: from, To: to}
	where := "WHERE status='paid'"
	args := []any{}
	if from != "" {
		where += " AND created_at >= ?"
		args = append(args, from)
	}
	if to != "" {
		where += " AND created_at < DATE_ADD(?, INTERVAL 1 DAY)"
		args = append(args, to)
	}
	err := r.db.QueryRow(`SELECT COALESCE(SUM(gross_cents),0),
		COALESCE(SUM(commission_cents),0),
		COALESCE(SUM(CASE WHEN gateway_fee_bearer='buyer' THEN gateway_fee_cents ELSE 0 END),0),
		COUNT(*) FROM orders `+where, args...).
		Scan(&s.GMVCents, &s.CommissionCents, &s.GatewayFeeCollectedCents, &s.OrdersPaid)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow(`SELECT COALESCE(SUM(oi_cnt),0) FROM
		(SELECT COUNT(*) AS oi_cnt FROM order_items oi JOIN orders o ON o.id=oi.order_id `+
		where+` GROUP BY oi.order_id) t`, args...).Scan(&s.TicketsSold)
	if err != nil {
		return nil, err
	}
	refWhere := "WHERE r.status='approved'"
	refArgs := []any{}
	if from != "" {
		refWhere += " AND r.created_at >= ?"
		refArgs = append(refArgs, from)
	}
	if to != "" {
		refWhere += " AND r.created_at < DATE_ADD(?, INTERVAL 1 DAY)"
		refArgs = append(refArgs, to)
	}
	_ = r.db.QueryRow(`SELECT COALESCE(SUM(amount_cents),0) FROM refunds r `+refWhere, refArgs...).
		Scan(&s.RefundedCents)
	return s, nil
}

// ---------- helpers ----------

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
