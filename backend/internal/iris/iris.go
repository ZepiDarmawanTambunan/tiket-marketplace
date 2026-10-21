// Package iris is a minimal client for the Midtrans Payouts API
// (formerly Iris) — sandbox and production. Auth is HTTP Basic with the
// Iris creator API key as username and an empty password, per
// https://iris-docs.midtrans.com.
package iris

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to the Midtrans Payouts (Iris) API.
type Client struct {
	APIKey      string
	Environment string // sandbox | production
	http        *http.Client
}

// NewClient builds an Iris client for the given environment.
func NewClient(apiKey, environment string) *Client {
	if environment == "" {
		environment = "sandbox"
	}
	return &Client{
		APIKey:      apiKey,
		Environment: environment,
		http:        &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) base() string {
	if c.Environment == "production" {
		return "https://app.midtrans.com/iris/api/v1"
	}
	return "https://app.sandbox.midtrans.com/iris/api/v1"
}

// IsConfigured reports whether a usable API key is set.
func (c *Client) IsConfigured() bool { return len(c.APIKey) >= 8 }

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.APIKey, "")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("iris request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// PayoutRequest is a single disbursement to a bank account.
type PayoutRequest struct {
	BeneficiaryName    string
	BeneficiaryAccount string
	BeneficiaryBank    string
	BeneficiaryEmail   string
	AmountCents        int64 // rupiah
	Notes              string
}

// CreatePayout creates one payout and returns its reference_no and initial status.
func (c *Client) CreatePayout(p PayoutRequest) (referenceNo, status, raw string, err error) {
	payload, _ := json.Marshal(map[string]any{
		"payouts": []any{map[string]any{
			"beneficiary_name":    p.BeneficiaryName,
			"beneficiary_account": p.BeneficiaryAccount,
			"beneficiary_bank":    p.BeneficiaryBank,
			"beneficiary_email":   p.BeneficiaryEmail,
			"amount":              fmt.Sprintf("%d.00", p.AmountCents),
			"notes":               p.Notes,
		}},
	})
	req, err := http.NewRequest(http.MethodPost, c.base()+"/payouts", bytes.NewReader(payload))
	if err != nil {
		return "", "", "", err
	}
	body, code, err := c.do(req)
	if err != nil {
		return "", "", "", err
	}
	raw = truncate(string(body), 2000)
	if code < 200 || code >= 300 {
		return "", "", raw, fmt.Errorf("iris create payout returned %d: %s", code, truncate(raw, 300))
	}
	var out struct {
		Payouts []struct {
			Status      string `json:"status"`
			ReferenceNo string `json:"reference_no"`
		} `json:"payouts"`
		ErrorMessage string   `json:"error_message"`
		Errors       []string `json:"errors"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", raw, fmt.Errorf("iris decode: %w", err)
	}
	if len(out.Payouts) == 0 {
		return "", "", raw, fmt.Errorf("iris: empty payouts in response (err=%s %v)",
			out.ErrorMessage, out.Errors)
	}
	return out.Payouts[0].ReferenceNo, out.Payouts[0].Status, raw, nil
}

// GetPayoutStatus returns the current status of a payout by reference_no.
func (c *Client) GetPayoutStatus(referenceNo string) (status, raw string, err error) {
	req, err := http.NewRequest(http.MethodGet, c.base()+"/payouts/"+referenceNo, nil)
	if err != nil {
		return "", "", err
	}
	body, code, err := c.do(req)
	if err != nil {
		return "", "", err
	}
	raw = truncate(string(body), 2000)
	if code < 200 || code >= 300 {
		return "", raw, fmt.Errorf("iris payout status returned %d: %s", code, truncate(raw, 300))
	}
	var out struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", raw, fmt.Errorf("iris decode: %w", err)
	}
	return out.Status, raw, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
