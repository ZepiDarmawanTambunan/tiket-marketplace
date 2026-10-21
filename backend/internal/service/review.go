package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
)

// ---------- Reviews ----------

type ReviewService struct{ deps *Deps }

func NewReviewService(deps *Deps) *ReviewService { return &ReviewService{deps: deps} }

// Create lets a buyer rate an event: requires a paid order AND the event to
// have ended (ends_at in the past). One review per user per event.
func (s *ReviewService) Create(userID, eventID int64, rating int, text string) (*model.Review, error) {
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("%w: rating harus 1-5", ErrUnprocessable)
	}
	e, err := s.deps.Events.FindByID(eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !time.Now().After(e.EndsAt) {
		return nil, fmt.Errorf("%w: ulasan hanya bisa diberikan setelah event selesai", ErrUnprocessable)
	}
	has, err := s.deps.Reviews.HasPaidOrder(userID, eventID)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("%w: hanya pembeli dengan order paid yang bisa mengulas", ErrUnprocessable)
	}
	if _, err := s.deps.Reviews.FindByEventUser(eventID, userID); err == nil {
		return nil, fmt.Errorf("%w: Anda sudah mengulas event ini", ErrConflict)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	rv, err := s.deps.Reviews.Create(eventID, userID, rating, text)
	if err != nil {
		return nil, err
	}
	_ = s.deps.Audit.Log(int64ptr(userID), "review.create",
		fmt.Sprintf("review_id=%d event_id=%d rating=%d", rv.ID, eventID, rating))
	return rv, nil
}

// Update edits the caller's own review for an event.
func (s *ReviewService) Update(userID, eventID int64, rating int, text string) (*model.Review, error) {
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("%w: rating harus 1-5", ErrUnprocessable)
	}
	rv, err := s.deps.Reviews.FindByEventUser(eventID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.deps.Reviews.Update(rv.ID, rating, text); err != nil {
		return nil, err
	}
	rv.Rating = rating
	rv.Text = text
	return rv, nil
}

// DeleteByAdmin removes a review (moderation).
func (s *ReviewService) DeleteByAdmin(adminID, reviewID int64) error {
	rv, err := s.deps.Reviews.FindByID(reviewID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if err := s.deps.Reviews.Delete(reviewID); err != nil {
		return err
	}
	_ = s.deps.Audit.Log(int64ptr(adminID), "review.delete",
		fmt.Sprintf("review_id=%d event_id=%d user_id=%d", rv.ID, rv.EventID, rv.UserID))
	return nil
}

type ReviewListResult struct {
	Reviews []model.Review      `json:"reviews"`
	Summary model.ReviewSummary `json:"summary"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
	Total   int64               `json:"total"`
}

// List returns public reviews of an event plus the rating summary.
func (s *ReviewService) List(eventID int64, page, perPage int) (*ReviewListResult, error) {
	if _, err := s.deps.Events.FindByID(eventID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	reviews, total, err := s.deps.Reviews.List(eventID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, err
	}
	avg, count, err := s.deps.Reviews.Summary(eventID)
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []model.Review{}
	}
	return &ReviewListResult{
		Reviews: reviews,
		Summary: model.ReviewSummary{Avg: round1(avg), Count: count},
		Page:    page, PerPage: perPage, Total: total,
	}, nil
}

// EventRating returns the avg/count for embedding in event detail.
func (s *ReviewService) EventRating(eventID int64) (float64, int64) {
	avg, count, err := s.deps.Reviews.Summary(eventID)
	if err != nil {
		return 0, 0
	}
	return round1(avg), count
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
