package reservation

import (
	"errors"
	"time"
)

// Equipment represents one physical item that can be reserved.
type Equipment struct {
	ID   string
	Name string
}

// Status is the persisted state of a reservation.
type Status string

const (
	StatusActive    Status = "ACTIVE"
	StatusCancelled Status = "CANCELLED"
)

// Reservation records one requested interval for a piece of equipment.
type Reservation struct {
	ID          int64
	EquipmentID string
	Requester   string
	StartsAt    time.Time
	EndsAt      time.Time
	Status      Status
	CreatedAt   time.Time
	CancelledAt *time.Time
}

// CreateInput contains the values supplied when creating a reservation.
type CreateInput struct {
	EquipmentID string
	Requester   string
	StartsAt    string
	EndsAt      string
}

var (
	ErrInvalidInput                = errors.New("invalid reservation input")
	ErrDateBeyondLimit             = errors.New("reservation date exceeds the allowed range")
	ErrEquipmentNotFound           = errors.New("equipment not found")
	ErrReservationNotFound         = errors.New("reservation not found")
	ErrReservationConflict         = errors.New("reservation overlaps an active reservation")
	ErrReservationAlreadyCancelled = errors.New("reservation is already cancelled")
	ErrReservationNotActive        = errors.New("reservation is not active")
)

const databaseTimeLayout = "2006-01-02T15:04:05.000000000Z"

func formatDatabaseTime(value time.Time) string {
	return value.UTC().Format(databaseTimeLayout)
}

func parseDatabaseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(databaseTimeLayout, value)
	if err == nil {
		return parsed.UTC(), nil
	}

	// Accept the legacy SQLite CURRENT_TIMESTAMP representation for rows
	// created before the reservation repository began writing UTC timestamps.
	parsed, legacyErr := time.Parse("2006-01-02 15:04:05", value)
	if legacyErr == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, err
}
