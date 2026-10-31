package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"reservation-system/internal/inventory"

	_ "modernc.org/sqlite"
)

const databasePath = "data/reservations.db"

// Initialize opens the local SQLite database, creates its schema, and loads
// the initial inventory without replacing equipment already stored there.
func Initialize(ctx context.Context) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	dsn := filepath.ToSlash(databasePath) + "?_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}

	// Keep the local SQLite connection pool bounded. The DSN applies foreign
	// key enforcement to every physical connection opened by database/sql.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to SQLite database: %w", err)
	}

	if err := initializeSchemaAndInventory(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func initializeSchemaAndInventory(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin database initialization: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS equipment (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS reservations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			equipment_id TEXT NOT NULL,
			starts_at TEXT NOT NULL,
			ends_at TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			canceled_at TEXT,
			FOREIGN KEY (equipment_id) REFERENCES equipment(id)
		)`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("create database schema: %w", err)
		}
	}

	for _, item := range inventory.InitialEquipment() {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO equipment (id, name) VALUES (?, ?) ON CONFLICT (id) DO NOTHING`,
			item.ID,
			item.Name,
		); err != nil {
			return fmt.Errorf("seed equipment %s: %w", item.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit database initialization: %w", err)
	}
	return nil
}
