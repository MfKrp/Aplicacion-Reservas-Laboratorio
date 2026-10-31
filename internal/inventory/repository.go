package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"reservation-system/internal/reservation"
)

// Repository reads equipment records needed by the reservation service.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]reservation.Equipment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM equipment ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list equipment: %w", err)
	}
	defer rows.Close()

	items := make([]reservation.Equipment, 0)
	for rows.Next() {
		var item reservation.Equipment
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan equipment: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate equipment: %w", err)
	}
	return items, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (reservation.Equipment, error) {
	var item reservation.Equipment
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name
		FROM equipment
		WHERE id = ?
	`, id).Scan(&item.ID, &item.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return reservation.Equipment{}, fmt.Errorf("equipment %q: %w", id, ErrNotFound)
	}
	if err != nil {
		return reservation.Equipment{}, fmt.Errorf("get equipment %q: %w", id, err)
	}
	return item, nil
}

var ErrNotFound = reservation.ErrEquipmentNotFound
