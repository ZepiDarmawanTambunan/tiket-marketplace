package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/crypto"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/repository"
)

// ---------- Tickets & check-in ----------

type TicketService struct {
	deps     *Deps
	qrSecret string
}

func NewTicketService(deps *Deps, qrSecret string) *TicketService {
	return &TicketService{deps: deps, qrSecret: qrSecret}
}

// MyTickets returns the viewer's tickets from paid orders.
// A ticket belongs to the viewer when they are its current holder
// (holder_user_id, falling back to the order owner when never transferred).
func (s *TicketService) MyTickets(userID int64) ([]model.Ticket, error) {
	return s.deps.Items.ListByUser(userID)
}

// TransferTicket moves a ticket to another registered user. Conditions:
// the caller is the current holder, the ticket is unused, its order is paid
// (a refunded order's tickets are rejected), and the event hasn't started.
// The recipient gets a WhatsApp notification (fire-and-forget).
func (s *TicketService) TransferTicket(fromUserID int64, ticketCode, toEmail string) (*repository.TicketLookup, error) {
	t, err := s.deps.Items.FindByCode(ticketCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if t.HolderUserID != fromUserID {
		return nil, fmt.Errorf("%w: tiket bukan milik Anda", ErrForbidden)
	}
	if t.UsedAt.Valid {
		return nil, fmt.Errorf("%w: tiket sudah dipakai", ErrAlreadyUsed)
	}
	if t.OrderStatus == model.OrderRefunded {
		return nil, fmt.Errorf("%w: tiket dari order yang sudah di-refund", ErrUnprocessable)
	}
	if t.OrderStatus != model.OrderPaid {
		return nil, fmt.Errorf("%w: order belum dibayar", ErrUnprocessable)
	}
	if !time.Now().Before(t.EventStartsAt) {
		return nil, fmt.Errorf("%w: event sudah dimulai, tiket tidak bisa ditransfer", ErrUnprocessable)
	}
	to, err := s.deps.Users.FindByEmail(toEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: email penerima belum terdaftar", ErrUnprocessable)
		}
		return nil, err
	}
	if to.ID == fromUserID {
		return nil, fmt.Errorf("%w: tidak bisa transfer ke diri sendiri", ErrUnprocessable)
	}

	tx, err := s.deps.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.deps.Items.TransferTx(tx, t.ItemID, to.ID); err != nil {
		return nil, err
	}
	if err := s.deps.Transfers.RecordTx(tx, t.TicketCode, fromUserID, to.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	from, _ := s.deps.Users.FindByID(fromUserID)
	fromName := ""
	if from != nil {
		fromName = from.Name
	}
	msg := fmt.Sprintf("Kamu menerima tiket *%s* (%s) dari %s.\nKode tiket: %s\nLihat di menu Tiket Saya.",
		t.EventTitle, t.TicketTypeName, fromName, t.TicketCode)
	s.deps.Notify.Notify(to.Phone, msg)

	t.HolderUserID = to.ID
	t.HolderName = to.Name
	_ = s.deps.Audit.Log(int64ptr(fromUserID), "ticket.transfer",
		fmt.Sprintf("ticket=%s from=%d to=%d", t.TicketCode, fromUserID, to.ID))
	return t, nil
}

// TransferHistory returns transfers involving the user (sent or received).
func (s *TicketService) TransferHistory(userID int64, page, perPage int) ([]model.TicketTransfer, int64, error) {
	return s.deps.Transfers.ListByUser(userID, perPage, (page-1)*perPage)
}

type CheckInResult struct {
	TicketCode     string    `json:"ticket_code"`
	EventTitle     string    `json:"event_title"`
	TicketTypeName string    `json:"ticket_type_name"`
	HolderName     string    `json:"holder_name"`
	CheckedInAt    time.Time `json:"checked_in_at"`
}

// CheckIn marks a ticket as used. The actor must be the event's seller or a
// super admin. Double scans are rejected with ErrConflict (mapped to 409).
func (s *TicketService) CheckIn(actorID int64, actorRole, ticketCode, qrPayload string) (*CheckInResult, error) {
	var (
		t   *repository.TicketLookup
		err error
	)
	if qrPayload != "" {
		// Verify the HMAC before trusting the payload (anti fake QR).
		code, mac, ok := strings.Cut(qrPayload, ".")
		if !ok || code == "" || mac == "" {
			return nil, fmt.Errorf("%w: format qr_payload tidak valid", ErrUnprocessable)
		}
		if !crypto.VerifyHMAC(s.qrSecret, code, mac) {
			return nil, fmt.Errorf("%w: qr_payload tidak valid", ErrUnauthorized)
		}
		t, err = s.deps.Items.FindByQRPayload(qrPayload)
	} else if ticketCode != "" {
		t, err = s.deps.Items.FindByCode(ticketCode)
	} else {
		return nil, fmt.Errorf("%w: ticket_code atau qr_payload wajib diisi", ErrUnprocessable)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if t.OrderStatus == model.OrderRefunded {
		return nil, fmt.Errorf("%w: tiket dari order yang sudah di-refund", ErrUnprocessable)
	}
	if t.OrderStatus != model.OrderPaid {
		return nil, fmt.Errorf("%w: order belum dibayar", ErrUnprocessable)
	}
	if actorRole != model.RoleSuperAdmin && t.EventSellerID != actorID {
		return nil, fmt.Errorf("%w: tiket bukan dari event Anda", ErrForbidden)
	}

	tx, err := s.deps.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	marked, err := s.deps.Items.MarkUsedTx(tx, t.ItemID)
	if err != nil {
		return nil, err
	}
	if !marked {
		return nil, fmt.Errorf("%w: tiket sudah dipakai", ErrAlreadyUsed)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	now := time.Now()
	return &CheckInResult{
		TicketCode:     t.TicketCode,
		EventTitle:     t.EventTitle,
		TicketTypeName: t.TicketTypeName,
		HolderName:     t.HolderName,
		CheckedInAt:    now,
	}, nil
}
