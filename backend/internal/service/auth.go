// Package service holds the business logic of the ticket marketplace.
package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/notify"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/repository"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("conflict")
	ErrAlreadyUsed   = errors.New("already_used")
	ErrUnprocessable = errors.New("unprocessable")
	ErrUnauthorized  = errors.New("unauthorized")
)

// Deps bundles the shared dependencies injected into every service.
type Deps struct {
	DB         *sql.DB
	Users      *repository.UserRepository
	Events     *repository.EventRepository
	Tickets    *repository.TicketTypeRepository
	Orders     *repository.OrderRepository
	OrderLines *repository.OrderLineRepository
	Items      *repository.TicketRepository
	Payments   *repository.PaymentRepository
	Refunds    *repository.RefundRepository
	Payouts    *repository.PayoutRepository
	Promos     *repository.PromoRepository
	Reviews    *repository.ReviewRepository
	Transfers  *repository.TransferRepository
	Settings   *SettingsService
	Reports    *repository.ReportRepository
	Audit      *repository.AuditRepository
	Notify     *notify.Notifier
}

// NewDeps wires all repositories and the settings service.
func NewDeps(db *sql.DB, masterKey []byte) *Deps {
	settings := NewSettingsService(repository.NewSettingsRepository(db), masterKey)
	return &Deps{
		DB:         db,
		Users:      repository.NewUserRepository(db),
		Events:     repository.NewEventRepository(db),
		Tickets:    repository.NewTicketTypeRepository(db),
		Orders:     repository.NewOrderRepository(db),
		OrderLines: repository.NewOrderLineRepository(db),
		Items:      repository.NewTicketRepository(db),
		Payments:   repository.NewPaymentRepository(db),
		Refunds:    repository.NewRefundRepository(db),
		Payouts:    repository.NewPayoutRepository(db),
		Promos:     repository.NewPromoRepository(db),
		Reviews:    repository.NewReviewRepository(db),
		Transfers:  repository.NewTransferRepository(db),
		Settings:   settings,
		Reports:    repository.NewReportRepository(db),
		Audit:      repository.NewAuditRepository(db),
		Notify:     notify.NewNotifier(settings),
	}
}

// ---------- Auth ----------

type AuthService struct {
	deps      *Deps
	jwtSecret []byte
	accessTTL time.Duration
}

func NewAuthService(deps *Deps, jwtSecret string, accessTTL time.Duration) *AuthService {
	return &AuthService{deps: deps, jwtSecret: []byte(jwtSecret), accessTTL: accessTTL}
}

type AuthResult struct {
	User  *model.User `json:"user"`
	Token string      `json:"token"`
}

// Register creates a buyer (default) or seller account.
func (s *AuthService) Register(name, email, phone, password, role string) (*AuthResult, error) {
	if role == "" {
		role = model.RoleBuyer
	}
	if role != model.RoleBuyer && role != model.RoleSeller {
		return nil, fmt.Errorf("%w: role must be buyer or seller", ErrUnprocessable)
	}
	if len(password) < 6 {
		return nil, fmt.Errorf("%w: password minimal 6 karakter", ErrUnprocessable)
	}
	if _, err := s.deps.Users.FindByEmail(email); err == nil {
		return nil, fmt.Errorf("%w: email sudah terdaftar", ErrConflict)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u, err := s.deps.Users.Create(name, email, phone, string(hash), role)
	if err != nil {
		return nil, err
	}
	token, err := s.issueToken(u)
	if err != nil {
		return nil, err
	}
	return &AuthResult{User: u, Token: token}, nil
}

// Login verifies credentials and returns a token.
func (s *AuthService) Login(email, password string) (*AuthResult, error) {
	u, err := s.deps.Users.FindByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: email atau password salah", ErrUnauthorized)
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("%w: email atau password salah", ErrUnauthorized)
	}
	token, err := s.issueToken(u)
	if err != nil {
		return nil, err
	}
	return &AuthResult{User: u, Token: token}, nil
}

func (s *AuthService) issueToken(u *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": u.ID,
		"role":    u.Role,
		"exp":     time.Now().Add(s.accessTTL).Unix(),
		"iat":     time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}
