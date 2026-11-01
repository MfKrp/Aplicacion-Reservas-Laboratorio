package reservation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
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
	if !validRequesterName(input.Requester) {
		return Reservation{}, fmt.Errorf("%w: requester may contain only letters and spaces", ErrInvalidInput)
	}

	startsAt, endsAt, err := parseInterval(input.StartsAt, input.EndsAt)
	if err != nil {
		return Reservation{}, err
	}
	if err := validateReservationDateRange(startsAt, endsAt, time.Now()); err != nil {
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

func validRequesterName(value string) bool {
	hasLetter := false
	previousWasLetterOrMark := false
	for _, character := range value {
		switch {
		case unicode.IsLetter(character):
			hasLetter = true
			previousWasLetterOrMark = true
		case unicode.IsMark(character) && previousWasLetterOrMark:
			// Combining accents are valid only as part of a preceding letter.
		case character == ' ':
			previousWasLetterOrMark = false
		default:
			return false
		}
	}
	return hasLetter
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
	if err := validateReservationDateRange(startsAt, endsAt, time.Now()); err != nil {
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

func validateReservationDateRange(startsAt, endsAt, now time.Time) error {
	location := now.Location()
	maximumDate := maximumReservationDate(now, 6)
	if calendarDate(startsAt, location).After(maximumDate) || calendarDate(endsAt, location).After(maximumDate) {
		return fmt.Errorf("%w: latest allowed date is %s", ErrDateBeyondLimit, maximumDate.Format("2006-01-02"))
	}
	return nil
}

func maximumReservationDate(now time.Time, months int) time.Time {
	location := now.Location()
	year, month, day := now.In(location).Date()
	targetMonthIndex := int(month) - 1 + months
	targetYear := year + targetMonthIndex/12
	targetMonth := time.Month(targetMonthIndex%12 + 1)
	lastDay := time.Date(targetYear, targetMonth+1, 0, 0, 0, 0, 0, location).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetYear, targetMonth, day, 0, 0, 0, 0, location)
}

func calendarDate(value time.Time, location *time.Location) time.Time {
	year, month, day := value.In(location).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}
