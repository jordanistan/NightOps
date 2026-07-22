// Package sqlite provides the local durable store.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial.sql
var migrationFS embed.FS

// Store owns a SQLite connection and its schema lifecycle.
type Store struct{ db *sql.DB }

// Open opens a SQLite database and applies all embedded migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) migrate(ctx context.Context) error {
	data, err := migrationFS.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, string(data)); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (1, ?)`, time.Now().UTC().Format(time.RFC3339))
	return err
}

// Ping verifies that the local database is available.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Close releases the database connection.
func (s *Store) Close() error { return s.db.Close() }
