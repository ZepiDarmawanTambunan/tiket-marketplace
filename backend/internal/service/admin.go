package service

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/crypto"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/iris"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/midtrans"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/model"
)

// ---------- Admin ----------

type AdminService struct{ deps *Deps }

func NewAdminService(deps *Deps) *AdminService { return &AdminService{deps: deps} }

// ListUsers returns users with optional role/search filters.
func (s *AdminService) ListUsers(role, search string, page, perPage int) ([]model.User, int64, error) {
	return s.deps.Users.List(role, search, perPage, (page-1)*perPage)
}

// UpdateRole changes a user's role.
func (s *AdminService) UpdateRole(adminID, userID int64, role string) (*model.User, error) {
	switch role {
	case model.RoleBuyer, model.RoleSeller, model.RoleSuperAdmin:
	default:
		return nil, fmt.Errorf("%w: role tidak valid", ErrUnprocessable)
	}
	u, err := s.deps.Users.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.deps.Users.UpdateRole(userID, role); err != nil {
		return nil, err
	}
	u.Role = role
	_ = s.deps.Audit.Log(int64ptr(adminID), "user.role",
		fmt.Sprintf("user_id=%d role=%s", userID, role))
	return u, nil
}

// VerifySeller toggles the seller_verified badge and stores bank details.
func (s *AdminService) VerifySeller(adminID, userID int64, verified bool, bankName, bankAccount string) (*model.User, error) {
	u, err := s.deps.Users.FindByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := s.deps.Users.VerifySeller(userID, verified, bankName, bankAccount); err != nil {
		return nil, err
	}
	u.SellerVerified = verified
	u.BankName = bankName
	u.BankAccount = bankAccount
	_ = s.deps.Audit.Log(int64ptr(adminID), "user.verify_seller",
		fmt.Sprintf("user_id=%d verified=%v", userID, verified))
	return u, nil
}

// ListTransactions returns orders with Midtrans payment info.
func (s *AdminService) ListTransactions(status, from, to string, page, perPage int) ([]model.Order, int64, error) {
	orders, total, err := s.deps.Orders.ListAll(status, from, to, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	if err := attachOrderDetails(s.deps, orders); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ApproveRefund approves a refund request and marks the order refunded.
// amountCents/reason override the requested values when provided (default:
// the full gross). It attempts a direct refund via the Midtrans Core API;
// when Midtrans is unreachable or rejects the call, it falls back to the
// phase-1 behavior: the refund is still approved with note "manual"
// (the request never fails because of this).
// ListRefunds returns refund requests for the admin console.
func (s *AdminService) ListRefunds(status string, page, perPage int) ([]model.Refund, int64, error) {
	return s.deps.Refunds.List(status, perPage, (page-1)*perPage)
}

func (s *AdminService) ApproveRefund(adminID, refundID int64, amountCents *int64, reason *string) (*model.Refund, error) {
	r, err := s.deps.Refunds.FindByID(refundID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if r.Status != model.RefundRequested {
		return nil, fmt.Errorf("%w: refund sudah diproses", ErrConflict)
	}
	amt := r.AmountCents
	if amountCents != nil && *amountCents > 0 {
		amt = *amountCents
	}
	reasonStr := r.Reason
	if reason != nil && *reason != "" {
		reasonStr = *reason
	}
	order, err := s.deps.Orders.FindByID(r.OrderID)
	if err != nil {
		return nil, err
	}

	// Try the Midtrans direct refund; fall back to "manual" on any failure.
	var midtransRefundID, rawResponse, note string
	serverKey, _, _, err := s.deps.Settings.MidtransKeys()
	if err != nil {
		return nil, err
	}
	env, _ := s.deps.Settings.MidtransEnv()
	mc := midtrans.NewClient(serverKey, "", "", env)
	if mc.IsDummy() {
		note = "manual"
		slog.Warn("midtrans refund skipped: dummy server key, manual fallback",
			"refund_id", refundID)
	} else if id, raw, rerr := mc.Refund(order.OrderCode, amt, reasonStr); rerr != nil {
		note = "manual"
		rawResponse = raw
		slog.Warn("midtrans refund failed, manual fallback", "refund_id", refundID, "err", rerr)
	} else {
		midtransRefundID, rawResponse = id, raw
	}

	tx, err := s.deps.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.deps.Refunds.ApproveTx(tx, refundID, adminID, amt, reasonStr,
		midtransRefundID, rawResponse, note); err != nil {
		return nil, err
	}
	if err := s.deps.Orders.UpdateStatusTx(tx, r.OrderID, model.OrderRefunded); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	r.Status = model.RefundApproved
	r.HandledBy = int64ptr(adminID)
	r.AmountCents = amt
	r.Reason = reasonStr
	r.MidtransRefundID = midtransRefundID
	r.RawResponse = rawResponse
	r.Note = note
	_ = s.deps.Audit.Log(int64ptr(adminID), "refund.approve",
		fmt.Sprintf("refund_id=%d order=%s amount=%d midtrans_refund=%s note=%s",
			refundID, r.OrderCode, amt, midtransRefundID, note))

	// Notify the buyer (fire-and-forget).
	if u, err := s.deps.Users.FindByID(order.UserID); err == nil {
		s.deps.Notify.Notify(u.Phone, fmt.Sprintf(
			"Refund order %s disetujui: Rp %d.%s",
			r.OrderCode, amt, manualSuffix(note)))
	}
	return r, nil
}

func manualSuffix(note string) string {
	if note == "manual" {
		return "\nDana dikembalikan manual oleh admin (cek mutasi rekening)."
	}
	return ""
}

// RejectRefund rejects a refund request and returns the order to paid.
func (s *AdminService) RejectRefund(adminID, refundID int64, reason string) (*model.Refund, error) {
	r, err := s.deps.Refunds.FindByID(refundID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if r.Status != model.RefundRequested {
		return nil, fmt.Errorf("%w: refund sudah diproses", ErrConflict)
	}
	tx, err := s.deps.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.deps.Refunds.UpdateStatusTx(tx, refundID, model.RefundRejected, adminID); err != nil {
		return nil, err
	}
	if err := s.deps.Orders.UpdateStatusTx(tx, r.OrderID, model.OrderPaid); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	r.Status = model.RefundRejected
	r.HandledBy = int64ptr(adminID)
	_ = s.deps.Audit.Log(int64ptr(adminID), "refund.reject",
		fmt.Sprintf("refund_id=%d order=%s reason=%q", refundID, r.OrderCode, reason))
	return r, nil
}

// ListPayouts returns payouts with filters.
func (s *AdminService) ListPayouts(status string, sellerID int64, page, perPage int) ([]model.Payout, int64, error) {
	return s.deps.Payouts.List(status, sellerID, perPage, (page-1)*perPage)
}

// CreatePayout records a payout to a seller. method is "manual" (default) or "iris".
func (s *AdminService) CreatePayout(adminID, sellerID, eventID, amountCents int64, method string) (*model.Payout, error) {
	if method == "" {
		method = model.PayoutMethodManual
	}
	if method != model.PayoutMethodManual && method != model.PayoutMethodIris {
		return nil, fmt.Errorf("%w: method harus manual|iris", ErrUnprocessable)
	}
	if _, err := s.deps.Users.FindByID(sellerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: seller tidak ditemukan", ErrUnprocessable)
		}
		return nil, err
	}
	if _, err := s.deps.Events.FindByID(eventID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: event tidak ditemukan", ErrUnprocessable)
		}
		return nil, err
	}
	p, err := s.deps.Payouts.Create(sellerID, eventID, amountCents, method)
	if err != nil {
		return nil, err
	}
	_ = s.deps.Audit.Log(int64ptr(adminID), "payout.create",
		fmt.Sprintf("payout_id=%d seller=%d amount=%d method=%s", p.ID, sellerID, amountCents, method))
	return p, nil
}

// MarkPayoutPaid marks a payout as paid with transfer proof (manual fallback).
func (s *AdminService) MarkPayoutPaid(adminID, payoutID int64, proofURL string) (*model.Payout, error) {
	p, err := s.deps.Payouts.FindByID(payoutID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if p.Status != model.PayoutPending && p.Status != model.PayoutProcessing {
		return nil, fmt.Errorf("%w: payout sudah diproses", ErrConflict)
	}
	if err := s.deps.Payouts.MarkPaid(payoutID, proofURL); err != nil {
		return nil, err
	}
	p.Status = model.PayoutPaid
	p.ProofURL = proofURL
	_ = s.deps.Audit.Log(int64ptr(adminID), "payout.pay",
		fmt.Sprintf("payout_id=%d proof=%s", payoutID, proofURL))
	if u, err := s.deps.Users.FindByID(p.SellerID); err == nil {
		s.deps.Notify.Notify(u.Phone, fmt.Sprintf(
			"Payout Rp %d untuk event #%d sudah dibayar. Bukti: %s",
			p.AmountCents, p.EventID, proofURL))
	}
	return p, nil
}

// SendIrisPayout disburses a pending payout through Midtrans Iris.
// The beneficiary bank details come from the seller's profile.
func (s *AdminService) SendIrisPayout(adminID, payoutID int64) (*model.Payout, error) {
	p, err := s.deps.Payouts.FindByID(payoutID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if p.Status != model.PayoutPending {
		return nil, fmt.Errorf("%w: hanya payout pending yang bisa dikirim via Iris", ErrConflict)
	}
	apiKey, err := s.deps.Settings.IrisAPIKey()
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, fmt.Errorf("%w: Iris belum dikonfigurasi (isi iris_api_key di admin settings)",
			ErrUnprocessable)
	}
	seller, err := s.deps.Users.FindByID(p.SellerID)
	if err != nil {
		return nil, err
	}
	if seller.BankName == "" || seller.BankAccount == "" {
		return nil, fmt.Errorf("%w: data bank seller belum lengkap (bank_name/bank_account)",
			ErrUnprocessable)
	}
	env, _ := s.deps.Settings.MidtransEnv()
	client := iris.NewClient(apiKey, env)
	refNo, irisStatus, _, err := client.CreatePayout(iris.PayoutRequest{
		BeneficiaryName:    seller.Name,
		BeneficiaryAccount: seller.BankAccount,
		BeneficiaryBank:    seller.BankName,
		BeneficiaryEmail:   seller.Email,
		AmountCents:        p.AmountCents,
		Notes:              fmt.Sprintf("Payout tiket event #%d", p.EventID),
	})
	if err != nil {
		return nil, fmt.Errorf("iris: %w", err)
	}
	if err := s.deps.Payouts.MarkIrisSent(payoutID, refNo, irisStatus); err != nil {
		return nil, err
	}
	p.Method = model.PayoutMethodIris
	p.IrisReferenceNo = refNo
	p.IrisStatus = irisStatus
	p.Status = model.PayoutProcessing
	_ = s.deps.Audit.Log(int64ptr(adminID), "payout.iris_send",
		fmt.Sprintf("payout_id=%d ref=%s status=%s", payoutID, refNo, irisStatus))
	s.deps.Notify.Notify(seller.Phone, fmt.Sprintf(
		"Payout Rp %d dikirim via Midtrans Iris.\nRef: %s\nStatus: %s",
		p.AmountCents, refNo, irisStatus))
	return p, nil
}

// RefreshIrisStatus pulls the latest payout status from Iris and updates the payout.
func (s *AdminService) RefreshIrisStatus(adminID, payoutID int64) (*model.Payout, error) {
	p, err := s.deps.Payouts.FindByID(payoutID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if p.IrisReferenceNo == "" {
		return nil, fmt.Errorf("%w: payout ini tidak dikirim via Iris", ErrUnprocessable)
	}
	apiKey, err := s.deps.Settings.IrisAPIKey()
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, fmt.Errorf("%w: Iris belum dikonfigurasi (isi iris_api_key di admin settings)",
			ErrUnprocessable)
	}
	env, _ := s.deps.Settings.MidtransEnv()
	client := iris.NewClient(apiKey, env)
	irisStatus, _, err := client.GetPayoutStatus(p.IrisReferenceNo)
	if err != nil {
		return nil, fmt.Errorf("iris: %w", err)
	}
	var payoutStatus string
	switch irisStatus {
	case "completed":
		payoutStatus = model.PayoutPaid
	case "failed", "rejected":
		// Return to pending so the admin can retry or pay manually.
		payoutStatus = model.PayoutPending
	}
	if err := s.deps.Payouts.UpdateIrisStatus(payoutID, irisStatus, payoutStatus); err != nil {
		return nil, err
	}
	p.IrisStatus = irisStatus
	if payoutStatus != "" {
		p.Status = payoutStatus
	}
	_ = s.deps.Audit.Log(int64ptr(adminID), "payout.iris_status",
		fmt.Sprintf("payout_id=%d ref=%s iris_status=%s", payoutID, p.IrisReferenceNo, irisStatus))
	if payoutStatus == model.PayoutPaid {
		if u, err := s.deps.Users.FindByID(p.SellerID); err == nil {
			s.deps.Notify.Notify(u.Phone, fmt.Sprintf(
				"Payout Rp %d via Iris selesai (ref %s).", p.AmountCents, p.IrisReferenceNo))
		}
	}
	return p, nil
}

// ReportSummary aggregates GMV, commission, and tickets for a period.
func (s *AdminService) ReportSummary(from, to string) (*model.ReportSummary, error) {
	return s.deps.Reports.Summary(from, to)
}

// ListAuditLogs returns audit logs with filters.
func (s *AdminService) ListAuditLogs(action string, actorID int64, page, perPage int) ([]model.AuditLog, int64, error) {
	return s.deps.Audit.List(action, actorID, perPage, (page-1)*perPage)
}

// MidtransSettingsView is the masked settings view returned to admins.
type MidtransSettingsView struct {
	Environment               string  `json:"environment"`
	ServerKeyMasked           string  `json:"server_key_masked"`
	ClientKeyMasked           string  `json:"client_key_masked"`
	MerchantID                string  `json:"merchant_id"`
	CommissionDefault         float64 `json:"commission_default"`
	GatewayFeeBearer          string  `json:"gateway_fee_bearer"`
	GatewayFeePercent         float64 `json:"gateway_fee_percent"`
	GatewayFeeFlatCents       int64   `json:"gateway_fee_flat_cents"`
	RequireSellerVerification bool    `json:"require_seller_verification"`
	IrisAPIKeyMasked          string  `json:"iris_api_key_masked"`
	IrisFromAccount           string  `json:"iris_from_account"`
	UpdatedAt                 string  `json:"updated_at,omitempty"`
}

// GetMidtransSettings returns the masked settings view (never plaintext keys).
func (s *AdminService) GetMidtransSettings() (*MidtransSettingsView, error) {
	serverKey, clientKey, merchantID, err := s.deps.Settings.MidtransKeys()
	if err != nil {
		return nil, err
	}
	env, _ := s.deps.Settings.MidtransEnv()
	comm, _ := s.deps.Settings.CommissionDefault()
	bearer, _ := s.deps.Settings.GatewayFeeBearer()
	feePct, feeFlat, _ := s.deps.Settings.GatewayFeeParams()
	reqVerif, _ := s.deps.Settings.RequireSellerVerification()
	irisKey, _ := s.deps.Settings.IrisAPIKey()
	irisFrom, _ := s.deps.Settings.IrisFromAccount()
	return &MidtransSettingsView{
		Environment:               env,
		ServerKeyMasked:           crypto.MaskSecret(serverKey),
		ClientKeyMasked:           crypto.MaskSecret(clientKey),
		MerchantID:                merchantID,
		CommissionDefault:         comm,
		GatewayFeeBearer:          bearer,
		GatewayFeePercent:         feePct,
		GatewayFeeFlatCents:       feeFlat,
		RequireSellerVerification: reqVerif,
		IrisAPIKeyMasked:          crypto.MaskSecret(irisKey),
		IrisFromAccount:           irisFrom,
	}, nil
}

// MidtransSettingsPatch holds optional settings updates; nil = unchanged.
type MidtransSettingsPatch struct {
	ServerKey                 *string
	ClientKey                 *string
	MerchantID                *string
	Environment               *string
	CommissionDefault         *float64
	GatewayFeeBearer          *string
	GatewayFeePercent         *float64
	GatewayFeeFlatCents       *int64
	RequireSellerVerification *bool
	IrisAPIKey                *string
	IrisFromAccount           *string
}

// UpdateMidtransSettings applies a patch, encrypts sensitive values, and audit-logs.
func (s *AdminService) UpdateMidtransSettings(adminID int64, p MidtransSettingsPatch) (*MidtransSettingsView, error) {
	changed := []string{}
	set := func(key string, v *string) error {
		if v == nil {
			return nil
		}
		if err := s.deps.Settings.Set(key, *v); err != nil {
			return err
		}
		changed = append(changed, key)
		return nil
	}
	if err := set(KeyMidtransServerKey, p.ServerKey); err != nil {
		return nil, err
	}
	if err := set(KeyMidtransClientKey, p.ClientKey); err != nil {
		return nil, err
	}
	if err := set(KeyMidtransMerchantID, p.MerchantID); err != nil {
		return nil, err
	}
	if p.Environment != nil {
		if *p.Environment != "sandbox" && *p.Environment != "production" {
			return nil, fmt.Errorf("%w: environment harus sandbox|production", ErrUnprocessable)
		}
		if err := s.deps.Settings.Set(KeyMidtransEnvironment, *p.Environment); err != nil {
			return nil, err
		}
		changed = append(changed, KeyMidtransEnvironment)
	}
	if p.CommissionDefault != nil {
		if *p.CommissionDefault < 0 || *p.CommissionDefault > 100 {
			return nil, fmt.Errorf("%w: komisi 0-100", ErrUnprocessable)
		}
		if err := s.deps.Settings.Set(KeyCommissionDefault, strconv.FormatFloat(*p.CommissionDefault, 'f', -1, 64)); err != nil {
			return nil, err
		}
		changed = append(changed, KeyCommissionDefault)
	}
	if p.GatewayFeeBearer != nil {
		switch *p.GatewayFeeBearer {
		case model.FeeBearerBuyer, model.FeeBearerSeller, model.FeeBearerPlatform:
		default:
			return nil, fmt.Errorf("%w: gateway_fee_bearer harus buyer|seller|platform", ErrUnprocessable)
		}
		if err := s.deps.Settings.Set(KeyGatewayFeeBearer, *p.GatewayFeeBearer); err != nil {
			return nil, err
		}
		changed = append(changed, KeyGatewayFeeBearer)
	}
	if p.GatewayFeePercent != nil {
		if err := s.deps.Settings.Set(KeyGatewayFeePercent, strconv.FormatFloat(*p.GatewayFeePercent, 'f', -1, 64)); err != nil {
			return nil, err
		}
		changed = append(changed, KeyGatewayFeePercent)
	}
	if p.GatewayFeeFlatCents != nil {
		if err := s.deps.Settings.Set(KeyGatewayFeeFlatCents, strconv.FormatInt(*p.GatewayFeeFlatCents, 10)); err != nil {
			return nil, err
		}
		changed = append(changed, KeyGatewayFeeFlatCents)
	}
	if err := set(KeyIrisAPIKey, p.IrisAPIKey); err != nil {
		return nil, err
	}
	if p.IrisFromAccount != nil {
		if err := s.deps.Settings.Set(KeyIrisFromAccount, *p.IrisFromAccount); err != nil {
			return nil, err
		}
		changed = append(changed, KeyIrisFromAccount)
	}
	if p.RequireSellerVerification != nil {
		v := "false"
		if *p.RequireSellerVerification {
			v = "true"
		}
		if err := s.deps.Settings.Set(KeyRequireSellerVerification, v); err != nil {
			return nil, err
		}
		changed = append(changed, KeyRequireSellerVerification)
	}
	_ = s.deps.Audit.Log(int64ptr(adminID), "settings.midtrans.update",
		fmt.Sprintf("changed=%v", changed))
	return s.GetMidtransSettings()
}

// MidtransClientKey returns the plaintext client key + environment for snap.js.
func (s *AdminService) MidtransClientKey() (clientKey, environment string, err error) {
	_, clientKey, _, err = s.deps.Settings.MidtransKeys()
	if err != nil {
		return "", "", err
	}
	environment, _ = s.deps.Settings.MidtransEnv()
	return clientKey, environment, nil
}

// PublicConfig exposes non-sensitive platform settings for public pages.
type PublicConfig struct {
	CommissionPercent float64 `json:"commission_percent"`
	GatewayFeeBearer  string  `json:"gateway_fee_bearer"`
	GatewayFeePercent float64 `json:"gateway_fee_percent"`
	GatewayFeeFlat    int64   `json:"gateway_fee_flat_cents"`
}

func (s *AdminService) PublicConfig() (*PublicConfig, error) {
	comm, err := s.deps.Settings.CommissionDefault()
	if err != nil {
		return nil, err
	}
	bearer, _ := s.deps.Settings.GatewayFeeBearer()
	feePct, feeFlat, _ := s.deps.Settings.GatewayFeeParams()
	return &PublicConfig{
		CommissionPercent: comm,
		GatewayFeeBearer:  bearer,
		GatewayFeePercent: feePct,
		GatewayFeeFlat:    feeFlat,
	}, nil
}

type MidtransTestResult struct {
	OK         bool   `json:"ok"`
	HTTPStatus int    `json:"http_status"`
	Message    string `json:"message"`
	Note       string `json:"note,omitempty"`
}

// TestMidtransConnection calls the Midtrans status API with a dummy order id
// to verify connectivity and key validity. Tolerant of dummy keys.
func (s *AdminService) TestMidtransConnection() (*MidtransTestResult, error) {
	serverKey, _, _, err := s.deps.Settings.MidtransKeys()
	if err != nil {
		return nil, err
	}
	env, _ := s.deps.Settings.MidtransEnv()
	mc := midtrans.NewClient(serverKey, "", "", env)
	if mc.IsDummy() {
		return &MidtransTestResult{OK: false, Message: "server key masih dummy",
			Note: "isi server key asli via PUT /api/admin/settings/midtrans"}, nil
	}
	status, body, err := mc.CheckStatus("TEST-DUMMY-ORDER-0001")
	if err != nil {
		return &MidtransTestResult{OK: false, Message: err.Error(),
			Note: "midtrans tidak terjangkau"}, nil
	}
	switch {
	case status == 200:
		return &MidtransTestResult{OK: true, HTTPStatus: status, Message: body}, nil
	case status == 404:
		// 404 "transaction doesn't exist" proves auth + connectivity work.
		return &MidtransTestResult{OK: true, HTTPStatus: status,
			Message: "koneksi OK (order dummy memang tidak ada)", Note: body}, nil
	case status == 401:
		return &MidtransTestResult{OK: false, HTTPStatus: status,
			Message: "auth gagal: server key salah/tidak valid", Note: body}, nil
	default:
		return &MidtransTestResult{OK: false, HTTPStatus: status, Message: body}, nil
	}
}
