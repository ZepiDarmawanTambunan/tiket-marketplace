package service

import (
	"fmt"
	"strconv"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/crypto"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/notify"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/repository"
)

// Setting keys stored in the settings table.
const (
	KeyCommissionDefault         = "commission_default"          // plain, percent, default "3"
	KeyRequireSellerVerification = "require_seller_verification" // plain bool, default "false"
	KeyGatewayFeeBearer          = "gateway_fee_bearer"          // plain: buyer|seller|platform
	KeyGatewayFeePercent         = "gateway_fee_percent"         // plain, default "2.9"
	KeyGatewayFeeFlatCents       = "gateway_fee_flat_cents"      // plain rupiah, default "2000"
	KeyMidtransEnvironment       = "midtrans_environment"        // plain: sandbox|production
	KeyMidtransServerKey         = "midtrans_server_key"         // encrypted
	KeyMidtransClientKey         = "midtrans_client_key"         // encrypted
	KeyMidtransMerchantID        = "midtrans_merchant_id"        // plain
	KeyIrisAPIKey                = "iris_api_key"                // encrypted (creator key)
	KeyIrisFromAccount           = "iris_from_account"           // plain (facilitator account label)
	KeyWAProvider                = notify.KeyWAProvider          // plain, default "fonnte"
	KeyWAFonnteToken             = notify.KeyWAFonnteToken       // encrypted
	KeyWAEnabled                 = notify.KeyWAEnabled           // plain bool, default "false"
)

// sensitiveKeys are encrypted at rest with AES-256-GCM.
var sensitiveKeys = map[string]bool{
	KeyMidtransServerKey: true,
	KeyMidtransClientKey: true,
	KeyIrisAPIKey:        true,
	KeyWAFonnteToken:     true,
}

// SettingsService reads/writes platform settings, decrypting sensitive values.
type SettingsService struct {
	repo      *repository.SettingsRepository
	masterKey []byte
}

func NewSettingsService(repo *repository.SettingsRepository, masterKey []byte) *SettingsService {
	return &SettingsService{repo: repo, masterKey: masterKey}
}

// defaults applied when a key was never stored.
var settingDefaults = map[string]string{
	KeyCommissionDefault:         "3",
	KeyRequireSellerVerification: "false",
	KeyGatewayFeeBearer:          "buyer",
	KeyGatewayFeePercent:         "2.9",
	KeyGatewayFeeFlatCents:       "2000",
	KeyMidtransEnvironment:       "sandbox",
	KeyMidtransServerKey:         "DUMMY-SB-Mid-server-xxxxxxxxxxxxxxxx",
	KeyMidtransClientKey:         "DUMMY-SB-Mid-client-xxxxxxxxxxxxxxxx",
	KeyMidtransMerchantID:        "DUMMY-MERCHANT",
	KeyIrisAPIKey:                "",
	KeyIrisFromAccount:           "",
	KeyWAProvider:                "fonnte",
	KeyWAFonnteToken:             "",
	KeyWAEnabled:                 "false",
}

// Get returns the decrypted value of a setting, falling back to its default.
func (s *SettingsService) Get(key string) (string, error) {
	enc, ok, err := s.repo.Get(key)
	if err != nil {
		return "", err
	}
	if !ok {
		return settingDefaults[key], nil
	}
	if sensitiveKeys[key] {
		if enc == "" {
			return "", nil
		}
		return crypto.DecryptAESGCM(s.masterKey, enc)
	}
	if enc == "" {
		return settingDefaults[key], nil
	}
	return enc, nil
}

// Set stores a setting, encrypting it when the key is sensitive.
func (s *SettingsService) Set(key, value string) error {
	if sensitiveKeys[key] {
		enc, err := crypto.EncryptAESGCM(s.masterKey, value)
		if err != nil {
			return fmt.Errorf("encrypt setting %s: %w", key, err)
		}
		return s.repo.Set(key, enc)
	}
	return s.repo.Set(key, value)
}

// SeedDefaults stores the defaults for keys that are not set yet.
func (s *SettingsService) SeedDefaults() error {
	for key, def := range settingDefaults {
		if _, ok, err := s.repo.Get(key); err != nil {
			return err
		} else if !ok {
			if err := s.Set(key, def); err != nil {
				return err
			}
		}
	}
	return nil
}

// CommissionDefault returns the platform commission percent.
func (s *SettingsService) CommissionDefault() (float64, error) {
	v, err := s.Get(KeyCommissionDefault)
	if err != nil {
		return 3, err
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 {
		return 3, nil
	}
	return f, nil
}

// RequireSellerVerification reports whether sellers must be verified before selling.
func (s *SettingsService) RequireSellerVerification() (bool, error) {
	v, err := s.Get(KeyRequireSellerVerification)
	if err != nil {
		return false, err
	}
	return v == "true" || v == "1", nil
}

// GatewayFeeBearer returns who pays the gateway fee: buyer|seller|platform.
func (s *SettingsService) GatewayFeeBearer() (string, error) {
	v, err := s.Get(KeyGatewayFeeBearer)
	if err != nil {
		return "buyer", err
	}
	switch v {
	case "seller", "platform":
		return v, nil
	default:
		return "buyer", nil
	}
}

// GatewayFeeParams returns percent + flat rupiah used to estimate the gateway fee.
func (s *SettingsService) GatewayFeeParams() (percent float64, flat int64, err error) {
	pv, err := s.Get(KeyGatewayFeePercent)
	if err != nil {
		return 2.9, 2000, err
	}
	percent, err = strconv.ParseFloat(pv, 64)
	if err != nil || percent < 0 {
		percent = 2.9
	}
	fv, err := s.Get(KeyGatewayFeeFlatCents)
	if err != nil {
		return percent, 2000, err
	}
	flat, err = strconv.ParseInt(fv, 10, 64)
	if err != nil || flat < 0 {
		flat = 2000
	}
	return percent, flat, nil
}

// MidtransEnv returns sandbox|production.
func (s *SettingsService) MidtransEnv() (string, error) {
	v, err := s.Get(KeyMidtransEnvironment)
	if err != nil {
		return "sandbox", err
	}
	if v == "production" {
		return "production", nil
	}
	return "sandbox", nil
}

// MidtransKeys returns the decrypted server key, client key, and merchant ID.
func (s *SettingsService) MidtransKeys() (serverKey, clientKey, merchantID string, err error) {
	if serverKey, err = s.Get(KeyMidtransServerKey); err != nil {
		return "", "", "", err
	}
	if clientKey, err = s.Get(KeyMidtransClientKey); err != nil {
		return "", "", "", err
	}
	if merchantID, err = s.Get(KeyMidtransMerchantID); err != nil {
		return "", "", "", err
	}
	return serverKey, clientKey, merchantID, nil
}

// IrisAPIKey returns the decrypted Iris creator API key ("" when not configured).
func (s *SettingsService) IrisAPIKey() (string, error) { return s.Get(KeyIrisAPIKey) }

// IrisFromAccount returns the configured Iris source account label.
func (s *SettingsService) IrisFromAccount() (string, error) { return s.Get(KeyIrisFromAccount) }

// WANotifications returns the WhatsApp notification settings.
func (s *SettingsService) WANotifications() (provider string, enabled bool, err error) {
	provider, err = s.Get(KeyWAProvider)
	if err != nil {
		return "", false, err
	}
	if provider == "" {
		provider = "fonnte"
	}
	v, err := s.Get(KeyWAEnabled)
	if err != nil {
		return provider, false, err
	}
	return provider, v == "true" || v == "1", nil
}
