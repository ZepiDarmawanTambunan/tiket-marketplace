// Package midtrans is a minimal client for the Midtrans Snap API (sandbox/production).
package midtrans

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to Midtrans Snap + Core API.
type Client struct {
	ServerKey   string
	ClientKey   string
	MerchantID  string
	Environment string // sandbox | production
	http        *http.Client
}

// NewClient builds a Midtrans client for the given environment.
func NewClient(serverKey, clientKey, merchantID, environment string) *Client {
	if environment == "" {
		environment = "sandbox"
	}
	return &Client{
		ServerKey:   serverKey,
		ClientKey:   clientKey,
		MerchantID:  merchantID,
		Environment: environment,
		http:        &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) snapBase() string {
	if c.Environment == "production" {
		return "https://app.midtrans.com"
	}
	return "https://app.sandbox.midtrans.com"
}

func (c *Client) apiBase() string {
	if c.Environment == "production" {
		return "https://api.midtrans.com"
	}
	return "https://api.sandbox.midtrans.com"
}

// IsDummy reports whether the configured keys are clearly placeholders.
func (c *Client) IsDummy() bool {
	return c.ServerKey == "" || len(c.ServerKey) < 16 || hasDummyPrefix(c.ServerKey)
}

func hasDummyPrefix(s string) bool {
	for _, p := range []string{"DUMMY", "dummy", "SB-Mid-server-DUMMY", "xxx", "***"} {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

// SnapRequest is the payload sent to create a Snap transaction.
type SnapRequest struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
}

// SnapResponse is the subset of the Snap create-transaction response we use.
type SnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

// CreateSnapToken creates a Snap transaction and returns its token.
func (c *Client) CreateSnapToken(orderID string, grossAmount int64) (*SnapResponse, error) {
	payload, _ := json.Marshal(map[string]any{
		"transaction_details": map[string]any{
			"order_id":     orderID,
			"gross_amount": grossAmount,
		},
	})
	req, err := http.NewRequest(http.MethodPost, c.snapBase()+"/snap/v1/transactions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.ServerKey, "")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("midtrans snap request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("midtrans snap returned %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	var out SnapResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode snap response: %w", err)
	}
	if out.Token == "" {
		return nil, fmt.Errorf("midtrans snap returned empty token")
	}
	return &out, nil
}

// CheckStatus queries a transaction status (used by the admin connection test).
func (c *Client) CheckStatus(orderID string) (int, string, error) {
	req, err := http.NewRequest(http.MethodGet, c.apiBase()+"/v2/"+orderID+"/status", nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.ServerKey, "")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("midtrans status request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, truncate(string(body), 500), nil
}

// Refund issues a direct refund via the Core API:
// POST {apiBase}/v2/{order_id}/refund {refund_key, amount, reason}.
// Returns the Midtrans refund_id and the raw response body.
func (c *Client) Refund(orderID string, amount int64, reason string) (refundID, raw string, err error) {
	refundKey := "REF-" + randomToken(10)
	payload, _ := json.Marshal(map[string]any{
		"refund_key": refundKey,
		"amount":     amount,
		"reason":     reason,
	})
	req, err := http.NewRequest(http.MethodPost, c.apiBase()+"/v2/"+orderID+"/refund", bytes.NewReader(payload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.ServerKey, "")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("midtrans refund request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	raw = truncate(string(body), 2000)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", raw, fmt.Errorf("midtrans refund returned %d: %s", resp.StatusCode, truncate(raw, 300))
	}
	var out struct {
		RefundID string `json:"refund_id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", raw, fmt.Errorf("decode refund response: %w", err)
	}
	if out.RefundID == "" {
		return "", raw, fmt.Errorf("midtrans refund returned no refund_id")
	}
	return out.RefundID, raw, nil
}

// VerifySignature checks the Midtrans signature_key:
// SHA512(order_id + status_code + gross_amount + serverKey).
func VerifySignature(serverKey, orderID, statusCode, grossAmount, signature string) bool {
	h := sha512.New()
	h.Write([]byte(orderID + statusCode + grossAmount + serverKey))
	sum := h.Sum(nil)
	expected, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return len(expected) == len(sum) && subtleEqual(sum, expected)
}

func subtleEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func randomToken(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

// Notification is the webhook payload sent by Midtrans.
type Notification struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	PaymentType       string `json:"payment_type"`
	FraudStatus       string `json:"fraud_status"`
}
