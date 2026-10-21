package service

import (
	"fmt"
	"strings"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/crypto"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/notify"
)

// ---------- WhatsApp notification settings ----------

// NotificationSettingsView is the masked notification settings view.
type NotificationSettingsView struct {
	WAProvider          string `json:"wa_provider"`
	WAFonnteTokenMasked string `json:"wa_fonnte_token_masked"`
	WAEnabled           bool   `json:"wa_enabled"`
}

// GetNotificationSettings returns the masked notification settings.
func (s *AdminService) GetNotificationSettings() (*NotificationSettingsView, error) {
	provider, enabled, err := s.deps.Settings.WANotifications()
	if err != nil {
		return nil, err
	}
	token, err := s.deps.Settings.Get(KeyWAFonnteToken)
	if err != nil {
		return nil, err
	}
	return &NotificationSettingsView{
		WAProvider:          provider,
		WAFonnteTokenMasked: crypto.MaskSecret(token),
		WAEnabled:           enabled,
	}, nil
}

// UpdateNotificationSettings patches the WhatsApp settings. Sensitive values
// are encrypted at rest; every change is audit-logged.
func (s *AdminService) UpdateNotificationSettings(adminID int64, provider, token *string, enabled *bool) (*NotificationSettingsView, error) {
	changed := []string{}
	if provider != nil {
		p := strings.ToLower(strings.TrimSpace(*provider))
		if p != "fonnte" {
			return nil, fmt.Errorf("%w: wa_provider yang didukung: fonnte", ErrUnprocessable)
		}
		if err := s.deps.Settings.Set(KeyWAProvider, p); err != nil {
			return nil, err
		}
		changed = append(changed, KeyWAProvider)
	}
	if token != nil {
		if err := s.deps.Settings.Set(notify.KeyWAFonnteToken, *token); err != nil {
			return nil, err
		}
		changed = append(changed, "wa_fonnte_token")
	}
	if enabled != nil {
		v := "false"
		if *enabled {
			v = "true"
		}
		if err := s.deps.Settings.Set(KeyWAEnabled, v); err != nil {
			return nil, err
		}
		changed = append(changed, KeyWAEnabled)
	}
	_ = s.deps.Audit.Log(int64ptr(adminID), "settings.notifications.update",
		fmt.Sprintf("changed=%v", changed))
	return s.GetNotificationSettings()
}

// NotificationTestResult is the outcome of the admin test message.
type NotificationTestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// TestNotification sends a synchronous test message so the result is reportable.
func (s *AdminService) TestNotification(to string) (*NotificationTestResult, error) {
	if strings.TrimSpace(to) == "" {
		return nil, fmt.Errorf("%w: nomor tujuan wajib diisi", ErrUnprocessable)
	}
	provider, enabled, err := s.deps.Settings.WANotifications()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return &NotificationTestResult{OK: false,
			Message: "wa_enabled=false: aktifkan dulu via PUT /api/admin/settings/notifications"}, nil
	}
	if err := s.deps.Notify.SendTest(to,
		"Tes notifikasi WhatsApp marketplace tiket ✅ (provider: "+provider+")"); err != nil {
		return &NotificationTestResult{OK: false, Message: err.Error()}, nil
	}
	return &NotificationTestResult{OK: true, Message: "pesan tes terkirim ke " + to}, nil
}
