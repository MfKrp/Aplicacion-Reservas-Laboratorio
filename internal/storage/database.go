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
	return InitializeAt(ctx, databasePath)
}

// InitializeAt initializes the same schema and inventory at a caller-selected
// SQLite file. It is useful for isolated local tools such as manual checks.
func InitializeAt(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("database path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	dsn := filepath.ToSlash(path) + "?_pragma=foreign_keys(1)"
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
			requester TEXT NOT NULL DEFAULT '',
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

	if err := ensureRequesterColumn(ctx, tx); err != nil {
		return err
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

func ensureRequesterColumn(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `PRAGMA table_info(reservations)`)
	if err != nil {
		return fmt.Errorf("inspect reservation schema: %w", err)
	}
	found := false
	for rows.Next() {
		var columnID, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&columnID, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read reservation schema: %w", err)
		}
		if name == "requester" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate reservation schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close reservation schema query: %w", err)
	}
	if found {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE reservations ADD COLUMN requester TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add reservation requester column: %w", err)
	}
	return nil
}
