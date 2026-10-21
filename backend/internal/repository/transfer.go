package repository

import (
	"database/sql"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
)

// ---------- Ticket transfers ----------

type TransferRepository struct{ db *sql.DB }

func NewTransferRepository(db *sql.DB) *TransferRepository { return &TransferRepository{db: db} }

// RecordTx logs a ticket transfer inside an existing transaction.
func (r *TransferRepository) RecordTx(tx *sql.Tx, ticketCode string, fromUserID, toUserID int64) error {
	_, err := tx.Exec(`INSERT INTO ticket_transfers (ticket_code,from_user_id,to_user_id)
		VALUES (?,?,?)`, ticketCode, fromUserID, toUserID)
	return err
}

// ListByUser returns the transfer history involving the user (as sender or receiver).
func (r *TransferRepository) ListByUser(userID int64, limit, offset int) ([]model.TicketTransfer, int64, error) {
	where := `WHERE tt.from_user_id=? OR tt.to_user_id=?`
	args := []any{userID, userID}
	var total int64
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM ticket_transfers tt `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(`SELECT tt.id,tt.ticket_code,tt.from_user_id,uf.name,
		tt.to_user_id,ut.name,tt.created_at FROM ticket_transfers tt
		JOIN users uf ON uf.id=tt.from_user_id
		JOIN users ut ON ut.id=tt.to_user_id `+where+`
		ORDER BY tt.id DESC LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.TicketTransfer
	for rows.Next() {
		t := model.TicketTransfer{}
		if err := rows.Scan(&t.ID, &t.TicketCode, &t.FromUserID, &t.FromUserName,
			&t.ToUserID, &t.ToUserName, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}
