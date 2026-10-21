// Package notify sends WhatsApp notifications (fire-and-forget) through a
// pluggable Sender. The default provider is Fonnte.
package notify

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Setting keys read from the platform settings store.
const (
	KeyWAProvider    = "wa_provider"     // plain, default "fonnte"
	KeyWAFonnteToken = "wa_fonnte_token" // encrypted
	KeyWAEnabled     = "wa_enabled"      // plain bool, default "false"
)

// Sender delivers one message to a WhatsApp number.
type Sender interface {
	Send(to, message string) error
}

// SettingsGetter is satisfied by the platform settings service.
type SettingsGetter interface {
	Get(key string) (string, error)
}

// Notifier routes messages to the configured provider. Delivery happens in a
// background goroutine: a failed send is logged and never fails the caller.
type Notifier struct{ settings SettingsGetter }

func NewNotifier(s SettingsGetter) *Notifier { return &Notifier{settings: s} }

// senderFor builds the Sender for the currently configured provider.
func (n *Notifier) senderFor() (Sender, error) {
	provider, _ := n.settings.Get(KeyWAProvider)
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "fonnte":
		token, _ := n.settings.Get(KeyWAFonnteToken)
		if token == "" {
			return nil, errors.New("token Fonnte belum diisi di admin settings")
		}
		return NewFonnte(token), nil
	default:
		return nil, fmt.Errorf("provider WhatsApp %q tidak didukung", provider)
	}
}

// send performs a single synchronous send, honoring the wa_enabled flag.
func (n *Notifier) send(to, message string) error {
	enabled, _ := n.settings.Get(KeyWAEnabled)
	if enabled != "true" && enabled != "1" {
		return nil
	}
	if strings.TrimSpace(to) == "" {
		return nil
	}
	sender, err := n.senderFor()
	if err != nil {
		return err
	}
	return sender.Send(to, message)
}

// Notify sends a message without blocking the caller. Any failure is logged
// only — notification must never break the main flow.
func (n *Notifier) Notify(to, message string) {
	go func() {
		if err := n.send(to, message); err != nil {
			slog.Warn("whatsapp notify failed", "to", maskPhone(to), "err", err)
		}
	}()
}

// SendTest sends synchronously so the admin test endpoint can report the result.
func (n *Notifier) SendTest(to, message string) error { return n.send(to, message) }

func maskPhone(p string) string {
	if len(p) <= 4 {
		return "***"
	}
	return "***" + p[len(p)-4:]
}

// ---------- Fonnte ----------

// Fonnte is the Fonnte (api.fonnte.com) Sender implementation.
type Fonnte struct {
	token string
	http  *http.Client
}

func NewFonnte(token string) *Fonnte {
	return &Fonnte{token: token, http: &http.Client{Timeout: 60 * time.Second}}
}

type fonnteResponse struct {
	Status bool   `json:"status"`
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

// Send posts a form-encoded message to Fonnte.
func (f *Fonnte) Send(to, message string) error {
	form := url.Values{"target": {to}, "message": {message}}
	req, err := http.NewRequest(http.MethodPost, "https://api.fonnte.com/send",
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", f.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.http.Do(req)
	if err != nil {
		return fmt.Errorf("fonnte request: %w", err)
	}
	defer resp.Body.Close()
	var out fonnteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("fonnte decode: %w", err)
	}
	if !out.Status {
		return fmt.Errorf("fonnte rejected: %s", firstNonEmpty(out.Reason, out.Detail))
	}
	return nil
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
