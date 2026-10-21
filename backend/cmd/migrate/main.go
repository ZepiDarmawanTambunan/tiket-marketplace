// Command migrate applies SQL migrations in backend/migrations/.
// Usage: go run ./cmd/migrate up   (reads DB_DSN from the environment)
package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "up" {
		fmt.Fprintln(os.Stderr, "usage: migrate up")
		os.Exit(1)
	}
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DB_DSN environment variable is required")
		os.Exit(1)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		slog.Error("ping db", "err", err)
		os.Exit(1)
	}

	dir := migrationDir()
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		slog.Error("glob migrations", "err", err)
		os.Exit(1)
	}
	sort.Strings(files)

	for _, f := range files {
		version := strings.TrimSuffix(filepath.Base(f), ".up.sql")
		if applied(db, version) {
			slog.Info("migration already applied", "version", version)
			continue
		}
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			slog.Error("read migration", "file", f, "err", err)
			os.Exit(1)
		}
		if err := applyFile(db, string(sqlBytes)); err != nil {
			slog.Error("apply migration failed", "version", version, "err", err)
			os.Exit(1)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			slog.Error("record migration", "version", version, "err", err)
			os.Exit(1)
		}
		slog.Info("migration applied", "version", version)
	}
	slog.Info("migrations up to date")
}

func migrationDir() string {
	// Prefer backend/migrations relative to the module root (works with go run ./cmd/migrate).
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	// Fallback: next to the source file (go run with absolute path).
	exe, _ := os.Executable()
	cand := filepath.Join(filepath.Dir(exe), "..", "..", "migrations")
	if _, err := os.Stat(cand); err == nil {
		return cand
	}
	return "migrations"
}

func applied(db *sql.DB, version string) bool {
	// schema_migrations is created by the first migration itself.
	var dummy int
	err := db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version=?`, version).Scan(&dummy)
	if err != nil {
		return false
	}
	return true
}

// applyFile executes a migration file statement by statement.
func applyFile(db *sql.DB, content string) error {
	for _, stmt := range splitStatements(content) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", truncate(stmt, 80), err)
		}
	}
	return nil
}

// splitStatements splits on semicolons at line ends (good enough for our DDL).
func splitStatements(content string) []string {
	lines := strings.Split(content, "\n")
	var stmts []string
	var cur strings.Builder
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		cur.WriteString(ln + "\n")
		if strings.HasSuffix(trimmed, ";") {
			stmts = append(stmts, cur.String())
			cur.Reset()
		}
	}
	if rest := strings.TrimSpace(cur.String()); rest != "" {
		stmts = append(stmts, rest)
	}
	return stmts
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
