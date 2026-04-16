// Package storage opens SQLite (modernc driver, CGO-free), sets pragmas, and applies embedded SQL migrations.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	*sql.DB
}

// Open connects to SQLite; dsn examples: file:./app.db, file::memory:?cache=shared for tests.
func Open(ctx context.Context, dsn string) (*DB, error) {
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, p := range pragmas {
		if _, err := conn.ExecContext(ctx, p); err != nil {
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &DB{conn}, nil
}

// Migrate executes each file in migrations/ in lexical order; SQL uses IF NOT EXISTS for idempotency.
func (db *DB) Migrate(ctx context.Context) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Name(), err)
		}
		if _, err := db.ExecContext(ctx, string(b)); err != nil {
			return fmt.Errorf("apply %s: %w", e.Name(), err)
		}
	}
	return nil
}
