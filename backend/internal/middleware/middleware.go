// Package middleware provides HTTP middleware: JWT auth, RBAC, and basics.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	ctxUserID ctxKey = "user_id"
	ctxRole   ctxKey = "role"
)

// UserID returns the authenticated user's id from the request context.
func UserID(r *http.Request) int64 {
	if v, ok := r.Context().Value(ctxUserID).(int64); ok {
		return v
	}
	return 0
}

// Role returns the authenticated user's role from the request context.
func Role(r *http.Request) string {
	if v, ok := r.Context().Value(ctxRole).(string); ok {
		return v
	}
	return ""
}

// Auth verifies the Bearer JWT and stores user_id + role in the context.
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeUnauthorized(w, "missing bearer token")
				return
			}
			claims := jwt.MapClaims{}
			parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !parsed.Valid {
				writeUnauthorized(w, "invalid token")
				return
			}
			uid, _ := claims["user_id"].(float64)
			role, _ := claims["role"].(string)
			if uid == 0 || role == "" {
				writeUnauthorized(w, "invalid token claims")
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserID, int64(uid))
			ctx = context.WithValue(ctx, ctxRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth is like Auth but lets the request through without a token
// (user_id/role stay zero-value when unauthenticated).
func OptionalAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			claims := jwt.MapClaims{}
			parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			})
			if err == nil && parsed.Valid {
				if uid, ok := claims["user_id"].(float64); ok && uid != 0 {
					ctx := context.WithValue(r.Context(), ctxUserID, int64(uid))
					if role, ok := claims["role"].(string); ok {
						ctx = context.WithValue(ctx, ctxRole, role)
					}
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole rejects requests whose role is not in the allowed list.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	set := map[string]bool{}
	for _, r := range allowed {
		set[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !set[Role(r)] {
				writeJSON(w, http.StatusForbidden, map[string]any{
					"error": map[string]string{"code": "FORBIDDEN", "message": "akses ditolak untuk role ini"},
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusUnauthorized, map[string]any{
		"error": map[string]string{"code": "UNAUTHORIZED", "message": msg},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Best-effort JSON encoding for middleware error responses.
	switch t := v.(type) {
	case map[string]any:
		errObj, _ := t["error"].(map[string]string)
		w.Write([]byte(`{"error":{"code":"` + errObj["code"] + `","message":"` + errObj["message"] + `"}}`))
	default:
		w.Write([]byte(`{"error":{"code":"UNKNOWN","message":"error"}}`))
	}
}

// CORS allows browser requests from configured origins.
// Preflight (OPTIONS) requests are answered directly.
func CORS(allowed []string) func(http.Handler) http.Handler {
	set := map[string]bool{}
	for _, o := range allowed {
		set[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && set[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID attaches a simple request id header for tracing.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = time.Now().Format("20060102150405.000000")
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r)
	})
}

// Recoverer recovers panics and returns 500.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"error": map[string]string{"code": "INTERNAL", "message": "terjadi kesalahan server"},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
