package repository

import (
	"database/sql"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
)

// ---------- Reviews ----------

type ReviewRepository struct{ db *sql.DB }

func NewReviewRepository(db *sql.DB) *ReviewRepository { return &ReviewRepository{db: db} }

func (r *ReviewRepository) Create(eventID, userID int64, rating int, text string) (*model.Review, error) {
	res, err := r.db.Exec(`INSERT INTO reviews (event_id,user_id,rating,text)
		VALUES (?,?,?,?)`, eventID, userID, rating, nullIfEmpty(text))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

func scanReview(rv *model.Review, row interface{ Scan(...any) error }) error {
	var text sql.NullString
	var userName sql.NullString
	err := row.Scan(&rv.ID, &rv.EventID, &rv.UserID, &userName, &rv.Rating, &text,
		&rv.CreatedAt, &rv.UpdatedAt)
	rv.Text = text.String
	rv.UserName = userName.String
	return err
}

func (r *ReviewRepository) FindByID(id int64) (*model.Review, error) {
	rv := &model.Review{}
	err := scanReview(rv, r.db.QueryRow(`SELECT rv.id,rv.event_id,rv.user_id,u.name,rv.rating,
		rv.text,rv.created_at,rv.updated_at FROM reviews rv
		JOIN users u ON u.id=rv.user_id WHERE rv.id=?`, id))
	if err != nil {
		return nil, err
	}
	return rv, nil
}

// FindByEventUser returns the review a user left for an event, or sql.ErrNoRows.
func (r *ReviewRepository) FindByEventUser(eventID, userID int64) (*model.Review, error) {
	rv := &model.Review{}
	err := scanReview(rv, r.db.QueryRow(`SELECT rv.id,rv.event_id,rv.user_id,u.name,rv.rating,
		rv.text,rv.created_at,rv.updated_at FROM reviews rv
		JOIN users u ON u.id=rv.user_id WHERE rv.event_id=? AND rv.user_id=?`, eventID, userID))
	if err != nil {
		return nil, err
	}
	return rv, nil
}

// List returns one page of reviews for an event, newest first.
func (r *ReviewRepository) List(eventID int64, limit, offset int) ([]model.Review, int64, error) {
	var total int64
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE event_id=?`, eventID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`SELECT rv.id,rv.event_id,rv.user_id,u.name,rv.rating,
		rv.text,rv.created_at,rv.updated_at FROM reviews rv
		JOIN users u ON u.id=rv.user_id WHERE rv.event_id=?
		ORDER BY rv.id DESC LIMIT ? OFFSET ?`, eventID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Review
	for rows.Next() {
		rv := model.Review{}
		if err := scanReview(&rv, rows); err != nil {
			return nil, 0, err
		}
		out = append(out, rv)
	}
	return out, total, rows.Err()
}

// Summary returns the average rating and review count for an event.
func (r *ReviewRepository) Summary(eventID int64) (avg float64, count int64, err error) {
	err = r.db.QueryRow(`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM reviews WHERE event_id=?`,
		eventID).Scan(&avg, &count)
	return avg, count, err
}

// Update edits a review's rating and text.
func (r *ReviewRepository) Update(id int64, rating int, text string) error {
	_, err := r.db.Exec(`UPDATE reviews SET rating=?, text=? WHERE id=?`,
		rating, nullIfEmpty(text), id)
	return err
}

func (r *ReviewRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM reviews WHERE id=?`, id)
	return err
}

// HasPaidOrder reports whether the user has a paid order for the event.
func (r *ReviewRepository) HasPaidOrder(userID, eventID int64) (bool, error) {
	var n int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM orders
		WHERE user_id=? AND event_id=? AND status='paid'`, userID, eventID).Scan(&n)
	return n > 0, err
}
