package reservation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// EquipmentReader lets the service validate equipment without depending on
// the inventory persistence implementation.
type EquipmentReader interface {
	GetByID(ctx context.Context, id string) (Equipment, error)
}

type Service struct {
	reservations Repository
	equipment    EquipmentReader
}

func NewService(reservations Repository, equipment EquipmentReader) *Service {
	return &Service{reservations: reservations, equipment: equipment}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Reservation, error) {
	equipmentID := strings.TrimSpace(input.EquipmentID)
	requester := strings.TrimSpace(input.Requester)
	if equipmentID == "" {
		return Reservation{}, fmt.Errorf("%w: equipment ID is required", ErrInvalidInput)
	}
	if requester == "" {
		return Reservation{}, fmt.Errorf("%w: requester is required", ErrInvalidInput)
	}

	startsAt, endsAt, err := parseInterval(input.StartsAt, input.EndsAt)
	if err != nil {
		return Reservation{}, err
	}
	if _, err := s.findEquipment(ctx, equipmentID); err != nil {
		return Reservation{}, err
	}

	item := Reservation{
		EquipmentID: equipmentID,
		Requester:   requester,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		Status:      StatusActive,
		CreatedAt:   time.Now().UTC(),
	}
	return s.reservations.Create(ctx, item)
}

func (s *Service) List(ctx context.Context) ([]Reservation, error) {
	return s.reservations.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id int64) (Reservation, error) {
	if id <= 0 {
		return Reservation{}, fmt.Errorf("%w: reservation ID must be positive", ErrInvalidInput)
	}
	return s.reservations.GetByID(ctx, id)
}

func (s *Service) Cancel(ctx context.Context, id int64) (Reservation, error) {
	if id <= 0 {
		return Reservation{}, fmt.Errorf("%w: reservation ID must be positive", ErrInvalidInput)
	}
	return s.reservations.Cancel(ctx, id, time.Now().UTC())
}

func (s *Service) IsAvailable(ctx context.Context, equipmentID, startsAtInput, endsAtInput string) (bool, error) {
	equipmentID = strings.TrimSpace(equipmentID)
	if equipmentID == "" {
		return false, fmt.Errorf("%w: equipment ID is required", ErrInvalidInput)
	}
	startsAt, endsAt, err := parseInterval(startsAtInput, endsAtInput)
	if err != nil {
		return false, err
	}
	if _, err := s.findEquipment(ctx, equipmentID); err != nil {
		return false, err
	}
	overlap, err := s.reservations.HasActiveOverlap(ctx, equipmentID, startsAt, endsAt)
	if err != nil {
		return false, err
	}
	return !overlap, nil
}

func (s *Service) findEquipment(ctx context.Context, id string) (Equipment, error) {
	item, err := s.equipment.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrEquipmentNotFound) {
			return Equipment{}, fmt.Errorf("%w: %s", ErrEquipmentNotFound, id)
		}
		return Equipment{}, fmt.Errorf("lookup equipment %s: %w", id, err)
	}
	return item, nil
}

func parseInterval(startsAtInput, endsAtInput string) (time.Time, time.Time, error) {
	startsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(startsAtInput))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: start must be RFC3339: %v", ErrInvalidInput, err)
	}
	endsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(endsAtInput))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: end must be RFC3339: %v", ErrInvalidInput, err)
	}
	startsAt = startsAt.UTC()
	endsAt = endsAt.UTC()
	if !startsAt.Before(endsAt) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: start must be before end", ErrInvalidInput)
	}
	return startsAt, endsAt, nil
}
