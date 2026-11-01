package reservation_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"reservation-system/internal/inventory"
	"reservation-system/internal/reservation"
	"reservation-system/internal/storage"
)

type reservationFixture struct {
	db      *sql.DB
	service *reservation.Service
}

func newReservationFixture(t *testing.T) *reservationFixture {
	t.Helper()
	db, err := storage.InitializeAt(context.Background(), filepath.Join(t.TempDir(), "reservations.db"))
	if err != nil {
		t.Fatalf("initialize isolated SQLite database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close SQLite database: %v", err)
		}
	})
	return &reservationFixture{db: db, service: newReservationService(db)}
}

func newReservationService(db *sql.DB) *reservation.Service {
	equipmentRepository := inventory.NewRepository(db)
	return reservation.NewService(
		reservation.NewSQLiteRepository(db),
		inventory.NewService(equipmentRepository),
	)
}

func TestCreateValidReservation(t *testing.T) {
	fixture := newReservationFixture(t)
	input := reservationInput(t, "MIC-01", "Ana Pérez", 9, 11)

	created, err := fixture.service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("Create() ID = %d, want a positive generated ID", created.ID)
	}
	if created.Status != reservation.StatusActive {
		t.Errorf("Create() status = %q, want %q", created.Status, reservation.StatusActive)
	}
	if created.EquipmentID != input.EquipmentID || created.Requester != input.Requester {
		t.Errorf("Create() identity fields = (%q, %q), want (%q, %q)", created.EquipmentID, created.Requester, input.EquipmentID, input.Requester)
	}
	wantStart, wantEnd := parsedInputTimes(t, input)
	if !created.StartsAt.Equal(wantStart) || !created.EndsAt.Equal(wantEnd) {
		t.Errorf("Create() interval = (%s, %s), want (%s, %s)", created.StartsAt, created.EndsAt, wantStart, wantEnd)
	}
	if created.CreatedAt.IsZero() {
		t.Error("Create() CreatedAt is zero")
	}

	stored, err := fixture.service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.ID != created.ID || stored.EquipmentID != input.EquipmentID || stored.Requester != input.Requester || stored.Status != reservation.StatusActive || !stored.StartsAt.Equal(wantStart) || !stored.EndsAt.Equal(wantEnd) {
		t.Errorf("stored reservation = %#v, does not match submitted reservation", stored)
	}
}

func TestAcceptsUnicodeRequesterNames(t *testing.T) {
	for _, name := range []string{"Juan Pérez", "María José", "José Núñez", "Jose\u0301 Núñez"} {
		t.Run(name, func(t *testing.T) {
			fixture := newReservationFixture(t)
			created, err := fixture.service.Create(context.Background(), reservationInput(t, "MIC-01", name, 9, 11))
			if err != nil {
				t.Fatalf("Create() rejected valid requester %q: %v", name, err)
			}
			if created.Requester != name {
				t.Errorf("stored requester = %q, want %q", created.Requester, name)
			}
		})
	}
}

func TestRejectsInvalidRequesterNames(t *testing.T) {
	for _, name := range []string{"Juan123", "Juan@Pérez", "12345", "!@#$%", "   "} {
		t.Run(name, func(t *testing.T) {
			fixture := newReservationFixture(t)
			input := reservationInput(t, "MIC-01", name, 9, 11)
			if _, err := fixture.service.Create(context.Background(), input); !errors.Is(err, reservation.ErrInvalidInput) {
				t.Fatalf("Create() error = %v, want ErrInvalidInput for requester %q", err, name)
			}
			items, err := fixture.service.List(context.Background())
			if err != nil {
				t.Fatalf("List() after rejected requester: %v", err)
			}
			if len(items) != 0 {
				t.Errorf("invalid requester persisted %d reservations, want 0", len(items))
			}
		})
	}
}

func TestRejectsOverlappingActiveReservation(t *testing.T) {
	fixture := newReservationFixture(t)
	first := reservationInput(t, "MIC-01", "Ana", 9, 11)
	if _, err := fixture.service.Create(context.Background(), first); err != nil {
		t.Fatalf("create first reservation: %v", err)
	}
	second := reservationInput(t, "MIC-01", "Bruno", 10, 12)
	if _, err := fixture.service.Create(context.Background(), second); !errors.Is(err, reservation.ErrReservationConflict) {
		t.Fatalf("second Create() error = %v, want ErrReservationConflict", err)
	}
	items, err := fixture.service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("List() returned %d reservations, want 1", len(items))
	}
}

func TestAllowsConsecutiveReservations(t *testing.T) {
	fixture := newReservationFixture(t)
	for _, input := range []reservation.CreateInput{
		reservationInput(t, "MIC-01", "Ana", 9, 11),
		reservationInput(t, "MIC-01", "Bruno", 11, 13),
	} {
		if _, err := fixture.service.Create(context.Background(), input); err != nil {
			t.Fatalf("Create() for %02d:00-%02d:00 returned error: %v", mustHour(t, input.StartsAt), mustHour(t, input.EndsAt), err)
		}
	}
	items, err := fixture.service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d reservations, want 2", len(items))
	}
}

func TestAllowsReservationOnMaximumDateInclusive(t *testing.T) {
	fixture := newReservationFixture(t)
	maximumDate := testMaximumReservationDate(time.Now())
	start := time.Date(maximumDate.Year(), maximumDate.Month(), maximumDate.Day(), 9, 0, 0, 0, time.Local)
	end := time.Date(maximumDate.Year(), maximumDate.Month(), maximumDate.Day(), 10, 0, 0, 0, time.Local)
	_, err := fixture.service.Create(context.Background(), reservation.CreateInput{
		EquipmentID: "MIC-01",
		Requester:   "Ana",
		StartsAt:    start.Format(time.RFC3339),
		EndsAt:      end.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("Create() on inclusive maximum date %s returned error: %v", maximumDate.Format("2006-01-02"), err)
	}
}

func TestAllowsSimultaneousReservationsForDifferentEquipment(t *testing.T) {
	fixture := newReservationFixture(t)
	intervals := []reservation.CreateInput{
		reservationInput(t, "MIC-01", "Ana", 9, 11),
		reservationInput(t, "OSC-01", "Bruno", 9, 11),
	}
	for _, input := range intervals {
		if _, err := fixture.service.Create(context.Background(), input); err != nil {
			t.Fatalf("Create() for %s returned error: %v", input.EquipmentID, err)
		}
	}
	available, err := fixture.service.IsAvailable(context.Background(), "OSC-01", intervals[0].StartsAt, intervals[0].EndsAt)
	if err != nil {
		t.Fatalf("IsAvailable() for OSC-01: %v", err)
	}
	if available {
		t.Error("OSC-01 should be unavailable after its reservation")
	}
	items, err := fixture.service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d reservations, want 2", len(items))
	}
}

func TestRejectsInvalidReservationInputs(t *testing.T) {
	base := reservationInput(t, "MIC-01", "Solicitante", 9, 11)
	maximumDate := testMaximumReservationDate(time.Now())
	afterLimit := maximumDate.AddDate(0, 0, 1).Format("2006-01-02")
	maximumDateString := maximumDate.Format("2006-01-02")
	cases := []struct {
		name  string
		input reservation.CreateInput
		want  error
	}{
		{name: "empty equipment ID", input: reservation.CreateInput{Requester: base.Requester, StartsAt: base.StartsAt, EndsAt: base.EndsAt}, want: reservation.ErrInvalidInput},
		{name: "empty requester", input: reservation.CreateInput{EquipmentID: base.EquipmentID, StartsAt: base.StartsAt, EndsAt: base.EndsAt}, want: reservation.ErrInvalidInput},
		{name: "unknown equipment", input: reservation.CreateInput{EquipmentID: "MISSING-01", Requester: base.Requester, StartsAt: base.StartsAt, EndsAt: base.EndsAt}, want: reservation.ErrEquipmentNotFound},
		{name: "invalid start format", input: reservation.CreateInput{EquipmentID: base.EquipmentID, Requester: base.Requester, StartsAt: "not-a-date", EndsAt: base.EndsAt}, want: reservation.ErrInvalidInput},
		{name: "invalid end format", input: reservation.CreateInput{EquipmentID: base.EquipmentID, Requester: base.Requester, StartsAt: base.StartsAt, EndsAt: "not-a-date"}, want: reservation.ErrInvalidInput},
		{name: "start after end", input: reservationInput(t, "MIC-01", "Solicitante", 11, 9), want: reservation.ErrInvalidInput},
		{name: "equal dates", input: reservation.CreateInput{EquipmentID: base.EquipmentID, Requester: base.Requester, StartsAt: base.StartsAt, EndsAt: base.StartsAt}, want: reservation.ErrInvalidInput},
		{name: "start beyond six-month horizon", input: reservation.CreateInput{EquipmentID: base.EquipmentID, Requester: base.Requester, StartsAt: afterLimit + "T09:00:00-03:00", EndsAt: afterLimit + "T10:00:00-03:00"}, want: reservation.ErrDateBeyondLimit},
		{name: "end beyond six-month horizon", input: reservation.CreateInput{EquipmentID: base.EquipmentID, Requester: base.Requester, StartsAt: maximumDateString + "T09:00:00-03:00", EndsAt: afterLimit + "T10:00:00-03:00"}, want: reservation.ErrDateBeyondLimit},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fixture := newReservationFixture(t)
			if _, err := fixture.service.Create(context.Background(), test.input); !errors.Is(err, test.want) {
				t.Fatalf("Create() error = %v, want %v", err, test.want)
			}
			items, err := fixture.service.List(context.Background())
			if err != nil {
				t.Fatalf("List() after rejected input: %v", err)
			}
			if len(items) != 0 {
				t.Errorf("invalid input persisted %d reservations, want 0", len(items))
			}
		})
	}
}

func TestCancellationRetainsReservationAndReleasesInterval(t *testing.T) {
	fixture := newReservationFixture(t)
	input := reservationInput(t, "MIC-01", "Ana", 9, 11)
	created, err := fixture.service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("create reservation: %v", err)
	}

	cancelled, err := fixture.service.Cancel(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if cancelled.Status != reservation.StatusCancelled {
		t.Errorf("Cancel() status = %q, want %q", cancelled.Status, reservation.StatusCancelled)
	}
	if cancelled.CancelledAt == nil || cancelled.CancelledAt.IsZero() {
		t.Fatal("Cancel() did not record a cancellation timestamp")
	}

	stored, err := fixture.service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() after cancellation: %v", err)
	}
	if stored.Status != reservation.StatusCancelled || stored.CancelledAt == nil || !stored.CancelledAt.Equal(*cancelled.CancelledAt) {
		t.Errorf("stored cancellation = %#v, want cancelled status and timestamp", stored)
	}

	if _, err := fixture.service.Create(context.Background(), input); err != nil {
		t.Fatalf("create reservation in released interval: %v", err)
	}
	items, err := fixture.service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 || items[0].Status != reservation.StatusCancelled || items[1].Status != reservation.StatusActive {
		t.Errorf("history after rebooking = %#v, want cancelled history and a new active reservation", items)
	}
}

func TestReservationStatesPersistAfterSQLiteReopen(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "persistent.db")
	db, err := storage.InitializeAt(ctx, databasePath)
	if err != nil {
		t.Fatalf("initialize SQLite: %v", err)
	}
	service := newReservationService(db)
	activeInput := reservationInput(t, "MIC-01", "Activa", 9, 11)
	active, err := service.Create(ctx, activeInput)
	if err != nil {
		t.Fatalf("create active reservation: %v", err)
	}
	cancelledInput := reservationInput(t, "OSC-01", "Cancelada", 9, 11)
	cancelled, err := service.Create(ctx, cancelledInput)
	if err != nil {
		t.Fatalf("create reservation to cancel: %v", err)
	}
	cancelled, err = service.Cancel(ctx, cancelled.ID)
	if err != nil {
		t.Fatalf("cancel reservation: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close SQLite before reopen: %v", err)
	}

	reopened, err := storage.InitializeAt(ctx, databasePath)
	if err != nil {
		t.Fatalf("reopen same SQLite database: %v", err)
	}
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("close reopened SQLite database: %v", err)
		}
	}()
	reopenedService := newReservationService(reopened)

	storedActive, err := reopenedService.GetByID(ctx, active.ID)
	if err != nil {
		t.Fatalf("read active reservation after reopen: %v", err)
	}
	if storedActive.Status != reservation.StatusActive {
		t.Errorf("active reservation status after reopen = %q, want %q", storedActive.Status, reservation.StatusActive)
	}
	storedCancelled, err := reopenedService.GetByID(ctx, cancelled.ID)
	if err != nil {
		t.Fatalf("read cancelled reservation after reopen: %v", err)
	}
	if storedCancelled.Status != reservation.StatusCancelled {
		t.Errorf("cancelled reservation status after reopen = %q, want %q", storedCancelled.Status, reservation.StatusCancelled)
	}
	if storedCancelled.CancelledAt == nil || !storedCancelled.CancelledAt.Equal(*cancelled.CancelledAt) {
		t.Errorf("cancellation timestamp after reopen = %v, want %v", storedCancelled.CancelledAt, cancelled.CancelledAt)
	}
}

func reservationInput(t *testing.T, equipmentID, requester string, startHour, endHour int) reservation.CreateInput {
	t.Helper()
	date := time.Now().In(time.Local).AddDate(0, 0, 1)
	start := time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0, time.Local)
	end := time.Date(date.Year(), date.Month(), date.Day(), endHour, 0, 0, 0, time.Local)
	return reservation.CreateInput{
		EquipmentID: equipmentID,
		Requester:   requester,
		StartsAt:    start.Format(time.RFC3339),
		EndsAt:      end.Format(time.RFC3339),
	}
}

func parsedInputTimes(t *testing.T, input reservation.CreateInput) (time.Time, time.Time) {
	t.Helper()
	start, err := time.Parse(time.RFC3339, input.StartsAt)
	if err != nil {
		t.Fatalf("parse test start time: %v", err)
	}
	end, err := time.Parse(time.RFC3339, input.EndsAt)
	if err != nil {
		t.Fatalf("parse test end time: %v", err)
	}
	return start.UTC(), end.UTC()
}

func mustHour(t *testing.T, value string) int {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse test interval time: %v", err)
	}
	return parsed.Hour()
}

func testMaximumReservationDate(now time.Time) time.Time {
	year, month, day := now.In(time.Local).Date()
	monthIndex := int(month) - 1 + 6
	targetYear := year + monthIndex/12
	targetMonth := time.Month(monthIndex%12 + 1)
	lastDay := time.Date(targetYear, targetMonth+1, 0, 0, 0, 0, 0, time.Local).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetYear, targetMonth, day, 0, 0, 0, 0, time.Local)
}
