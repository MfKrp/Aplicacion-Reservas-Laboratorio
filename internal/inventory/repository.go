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
