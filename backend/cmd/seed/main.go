// Command seed creates the super admin and default settings.
// Usage: go run ./cmd/seed   (reads DB_DSN, SEED_ADMIN_PASSWORD from the environment)
package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/crypto"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/repository"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/service"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DB_DSN environment variable is required")
		os.Exit(1)
	}
	masterKeyEnv := os.Getenv("SETTINGS_MASTER_KEY")
	if masterKeyEnv == "" {
		fmt.Fprintln(os.Stderr, "SETTINGS_MASTER_KEY environment variable is required")
		os.Exit(1)
	}
	db, err := repository.Open(dsn)
	if err != nil {
		slog.Error("database error", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// 1. Super admin.
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	usingDefault := false
	if password == "" {
		password = "admin123"
		usingDefault = true
	}
	if err := ensureSuperAdmin(db, "admin@tiket.local", "Administrator", password); err != nil {
		slog.Error("seed admin", "err", err)
		os.Exit(1)
	}
	if usingDefault {
		slog.Warn("SEED_ADMIN_PASSWORD tidak di-set; memakai password default 'admin123'. WAJIB diganti setelah login pertama!")
	}

	// 2. Default settings (only fills keys that are missing).
	settings := service.NewSettingsService(
		repository.NewSettingsRepository(db), crypto.DeriveKey(masterKeyEnv))
	if err := settings.SeedDefaults(); err != nil {
		slog.Error("seed settings", "err", err)
		os.Exit(1)
	}

	// 3. Optional: override Midtrans keys from environment (useful for first setup).
	for envKey, settingKey := range map[string]string{
		"MIDTRANS_SERVER_KEY":  service.KeyMidtransServerKey,
		"MIDTRANS_CLIENT_KEY":  service.KeyMidtransClientKey,
		"MIDTRANS_MERCHANT_ID": service.KeyMidtransMerchantID,
		"MIDTRANS_ENVIRONMENT": service.KeyMidtransEnvironment,
	} {
		if v := os.Getenv(envKey); v != "" {
			if err := settings.Set(settingKey, v); err != nil {
				slog.Error("seed midtrans setting", "key", envKey, "err", err)
				os.Exit(1)
			}
			slog.Info("midtrans setting dari env", "key", envKey)
		}
	}

	slog.Info("seed selesai",
		"admin", "admin@tiket.local",
		"commission_default", "3",
		"require_seller_verification", "false",
		"gateway_fee_bearer", "buyer",
		"midtrans", "sandbox (dummy keys)")
}

func ensureSuperAdmin(db *sql.DB, email, name, password string) error {
	var id int64
	err := db.QueryRow(`SELECT id FROM users WHERE email=?`, email).Scan(&id)
	if err == nil {
		slog.Info("super admin sudah ada", "email", email)
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO users (name,email,password_hash,role,seller_verified)
		VALUES (?,?,?,'super_admin',1)`, name, email, string(hash))
	if err != nil {
		return err
	}
	slog.Info("super admin dibuat", "email", email)
	return nil
}
