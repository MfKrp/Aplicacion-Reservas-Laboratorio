package reservation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Repository defines the persistence operations required by the service.
type Repository interface {
	Create(ctx context.Context, item Reservation) (Reservation, error)
	List(ctx context.Context) ([]Reservation, error)
	GetByID(ctx context.Context, id int64) (Reservation, error)
	Cancel(ctx context.Context, id int64, cancelledAt time.Time) (Reservation, error)
	HasActiveOverlap(ctx context.Context, equipmentID string, startsAt, endsAt time.Time) (bool, error)
}

// SQLiteRepository stores reservations using the shared SQLite connection.
type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, item Reservation) (Reservation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Reservation{}, fmt.Errorf("begin reservation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	overlap, err := hasActiveOverlap(ctx, tx, item.EquipmentID, item.StartsAt, item.EndsAt)
	if err != nil {
		return Reservation{}, err
	}
	if overlap {
		return Reservation{}, ErrReservationConflict
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO reservations
			(equipment_id, requester, starts_at, ends_at, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		item.EquipmentID,
		item.Requester,
		formatDatabaseTime(item.StartsAt),
		formatDatabaseTime(item.EndsAt),
		StatusActive,
		formatDatabaseTime(item.CreatedAt),
	)
	if err != nil {
		return Reservation{}, fmt.Errorf("insert reservation: %w", err)
	}

	item.ID, err = result.LastInsertId()
	if err != nil {
		return Reservation{}, fmt.Errorf("read new reservation ID: %w", err)
	}
	item.Status = StatusActive
	if err := tx.Commit(); err != nil {
		return Reservation{}, fmt.Errorf("commit reservation: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Reservation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, equipment_id, requester, starts_at, ends_at, status, created_at, canceled_at
		FROM reservations
		ORDER BY starts_at, id
	`)
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}
	defer rows.Close()

	items := make([]Reservation, 0)
	for rows.Next() {
		item, err := scanReservation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reservations: %w", err)
	}
	return items, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (Reservation, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, equipment_id, requester, starts_at, ends_at, status, created_at, canceled_at
		FROM reservations
		WHERE id = ?
	`, id)
	item, err := scanReservation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Reservation{}, ErrReservationNotFound
	}
	if err != nil {
		return Reservation{}, fmt.Errorf("get reservation %d: %w", id, err)
	}
	return item, nil
}

func (r *SQLiteRepository) Cancel(ctx context.Context, id int64, cancelledAt time.Time) (Reservation, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE reservations
		SET status = ?, canceled_at = ?
		WHERE id = ? AND status = ?
	`, StatusCancelled, formatDatabaseTime(cancelledAt), id, StatusActive)
	if err != nil {
		return Reservation{}, fmt.Errorf("cancel reservation %d: %w", id, err)
	}

	changed, err := result.RowsAffected()
	if err != nil {
		return Reservation{}, fmt.Errorf("check cancellation result: %w", err)
	}
	if changed == 0 {
		item, getErr := r.GetByID(ctx, id)
		if getErr != nil {
			return Reservation{}, getErr
		}
		if item.Status == StatusCancelled {
			return Reservation{}, ErrReservationAlreadyCancelled
		}
		return Reservation{}, ErrReservationNotActive
	}
	return r.GetByID(ctx, id)
}

func (r *SQLiteRepository) HasActiveOverlap(ctx context.Context, equipmentID string, startsAt, endsAt time.Time) (bool, error) {
	return hasActiveOverlap(ctx, r.db, equipmentID, startsAt, endsAt)
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func hasActiveOverlap(ctx context.Context, queryer queryRower, equipmentID string, startsAt, endsAt time.Time) (bool, error) {
	var overlap bool
	err := queryer.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM reservations
			WHERE equipment_id = ?
			  AND status = ?
			  AND starts_at < ?
			  AND ends_at > ?
		)
	`, equipmentID, StatusActive, formatDatabaseTime(endsAt), formatDatabaseTime(startsAt)).Scan(&overlap)
	if err != nil {
		return false, fmt.Errorf("check reservation overlap: %w", err)
	}
	return overlap, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReservation(row rowScanner) (Reservation, error) {
	var item Reservation
	var startsAt, endsAt, createdAt string
	var cancelledAt sql.NullString
	if err := row.Scan(
		&item.ID,
		&item.EquipmentID,
		&item.Requester,
		&startsAt,
		&endsAt,
		&item.Status,
		&createdAt,
		&cancelledAt,
	); err != nil {
		return Reservation{}, err
	}

	var err error
	if item.StartsAt, err = parseDatabaseTime(startsAt); err != nil {
		return Reservation{}, fmt.Errorf("parse reservation start time: %w", err)
	}
	if item.EndsAt, err = parseDatabaseTime(endsAt); err != nil {
		return Reservation{}, fmt.Errorf("parse reservation end time: %w", err)
	}
	if item.CreatedAt, err = parseDatabaseTime(createdAt); err != nil {
		return Reservation{}, fmt.Errorf("parse reservation creation time: %w", err)
	}
	if cancelledAt.Valid {
		parsed, err := parseDatabaseTime(cancelledAt.String)
		if err != nil {
			return Reservation{}, fmt.Errorf("parse reservation cancellation time: %w", err)
		}
		item.CancelledAt = &parsed
	}
	return item, nil
}
