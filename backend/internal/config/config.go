// Package config loads service configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the ticket marketplace API.
type Config struct {
	Port               string
	DBDSN              string
	JWTSecret          string
	AccessTTL          time.Duration
	SettingsMasterKey  string // master key for AES-256-GCM encryption of settings
	QRSecret           string // HMAC key for ticket qr_payload
	SeedAdminPassword  string
	CORSAllowedOrigins []string // allowed origins for browser CORS, e.g. http://localhost:3000
}

// Load reads configuration from the environment and applies defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		DBDSN:              getEnv("DB_DSN", "tiket:tiket@tcp(127.0.0.1:3306)/tiket_db?parseTime=true"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		AccessTTL:          getDurationEnv("JWT_ACCESS_TTL", 24*time.Hour),
		SettingsMasterKey:  os.Getenv("SETTINGS_MASTER_KEY"),
		QRSecret:           os.Getenv("QR_SECRET"),
		SeedAdminPassword:  getEnv("SEED_ADMIN_PASSWORD", "admin123"),
		CORSAllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if cfg.SettingsMasterKey == "" {
		return nil, fmt.Errorf("SETTINGS_MASTER_KEY environment variable is required")
	}
	if cfg.QRSecret == "" {
		return nil, fmt.Errorf("QR_SECRET environment variable is required")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getEnvSlice reads a comma-separated env var into a slice.
func getEnvSlice(key string, def []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return def
}

func getDurationEnv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if secs, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return def
}
