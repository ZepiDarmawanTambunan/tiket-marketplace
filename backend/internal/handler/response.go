// Package handler exposes the HTTP API of the ticket marketplace.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/service"
)

// ---------- response helpers ----------

func writeData(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

// badRequestError is a 400-class validation error.
type badRequestError struct{ msg string }

func (e badRequestError) Error() string { return e.msg }

func errBadRequest(msg string) error { return badRequestError{msg} }

func writeError(w http.ResponseWriter, err error) {
	code, msg := "INTERNAL", "terjadi kesalahan server"
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		status, code, msg = http.StatusUnauthorized, "UNAUTHORIZED", errMsg(err, "tidak terautentikasi")
	case errors.Is(err, service.ErrForbidden):
		status, code, msg = http.StatusForbidden, "FORBIDDEN", errMsg(err, "akses ditolak")
	case errors.Is(err, service.ErrNotFound):
		status, code, msg = http.StatusNotFound, "NOT_FOUND", errMsg(err, "tidak ditemukan")
	case errors.Is(err, service.ErrConflict):
		status, code, msg = http.StatusConflict, "CONFLICT", errMsg(err, "konflik")
	case errors.Is(err, service.ErrAlreadyUsed):
		status, code, msg = http.StatusConflict, "ALREADY_USED", "tiket sudah dipakai"
	case errors.Is(err, service.ErrUnprocessable):
		status, code, msg = http.StatusUnprocessableEntity, "UNPROCESSABLE", errMsg(err, "tidak dapat diproses")
	default:
		var bre badRequestError
		if errors.As(err, &bre) {
			status, code, msg = http.StatusBadRequest, "BAD_REQUEST", bre.msg
		} else if je := jsonSyntaxError(err); je != "" {
			status, code, msg = http.StatusBadRequest, "BAD_REQUEST", je
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}

// errMsg returns the wrapped message when the error carries one, else the fallback.
func errMsg(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	// Bare sentinel without context -> use the generic fallback message.
	for _, sentinel := range []error{service.ErrUnauthorized, service.ErrForbidden,
		service.ErrNotFound, service.ErrConflict, service.ErrUnprocessable} {
		if err == sentinel {
			return fallback
		}
	}
	s := err.Error()
	// Strip the sentinel prefix ("conflict: ...") for a clean message.
	for _, p := range []string{"unauthorized: ", "forbidden: ", "not found: ", "conflict: ", "unprocessable: "} {
		if len(s) > len(p) && s[:len(p)] == p {
			return s[len(p):]
		}
	}
	return s
}

func jsonSyntaxError(err error) string {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return "body JSON tidak valid"
	}
	return ""
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// ---------- request helpers ----------

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, name), 10, 64)
}

func queryInt(r *http.Request, key string, def, max int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func pageParams(r *http.Request) (page, perPage int) {
	return queryInt(r, "page", 1, 10000), queryInt(r, "per_page", 20, 100)
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

func paged(data any, page, perPage int, total int64) map[string]any {
	return map[string]any{"page": page, "per_page": perPage, "total": total, "items": data}
}
