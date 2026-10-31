package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"reservation-system/internal/inventory"
	"reservation-system/internal/reservation"
	"reservation-system/internal/storage"
)

const manualDatabasePath = "data/manual-check.db"

const (
	start0900 = "2030-06-15T09:00:00-03:00"
	start1000 = "2030-06-15T10:00:00-03:00"
	start1100 = "2030-06-15T11:00:00-03:00"
	end1100   = "2030-06-15T11:00:00-03:00"
	end1200   = "2030-06-15T12:00:00-03:00"
	end1300   = "2030-06-15T13:00:00-03:00"
)

type summary struct {
	passed int
	failed int
}

func main() {
	os.Exit(run())
}

func run() int {
	if err := prepareCleanDatabase(manualDatabasePath); err != nil {
		fmt.Fprintf(os.Stderr, "No se pudo preparar la base manual: %v\n", err)
		return 1
	}

	ctx := context.Background()
	db, err := storage.InitializeAt(ctx, manualDatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "No se pudo inicializar la base manual: %v\n", err)
		return 1
	}
	defer db.Close()

	service := reservation.NewService(
		reservation.NewSQLiteRepository(db),
		inventory.NewRepository(db),
	)
	results := &summary{}
	var firstReservationID int64

	created, err := service.Create(ctx, reservation.CreateInput{
		EquipmentID: "MIC-01",
		Requester:   "Verificación manual",
		StartsAt:    start0900,
		EndsAt:      end1100,
	})
	if err == nil {
		firstReservationID = created.ID
		report(results, "1. Creación válida", "Crear MIC-01 de 09:00 a 11:00", "creación exitosa", fmt.Sprintf("creada con ID %d y estado %s", created.ID, created.Status), true)
	} else {
		report(results, "1. Creación válida", "Crear MIC-01 de 09:00 a 11:00", "creación exitosa", "error: "+err.Error(), false)
	}

	overlap, err := service.Create(ctx, reservation.CreateInput{
		EquipmentID: "MIC-01",
		Requester:   "Verificación manual",
		StartsAt:    start1000,
		EndsAt:      end1200,
	})
	overlapCorrect := errors.Is(err, reservation.ErrReservationConflict)
	overlapActual := "rechazada por conflicto de horario"
	if err == nil {
		overlapActual = fmt.Sprintf("aceptada inesperadamente con ID %d", overlap.ID)
	} else if !overlapCorrect {
		overlapActual = "error distinto del conflicto esperado: " + err.Error()
	}
	report(results, "2. Superposición", "Crear MIC-01 de 10:00 a 12:00", "rechazo por conflicto", overlapActual, overlapCorrect)

	consecutive, err := service.Create(ctx, reservation.CreateInput{
		EquipmentID: "MIC-01",
		Requester:   "Verificación manual",
		StartsAt:    start1100,
		EndsAt:      end1300,
	})
	reportCreate(results, "3. Reservas consecutivas", "Crear MIC-01 de 11:00 a 13:00", consecutive, err)

	differentEquipment, err := service.Create(ctx, reservation.CreateInput{
		EquipmentID: "OSC-01",
		Requester:   "Verificación manual",
		StartsAt:    start0900,
		EndsAt:      end1100,
	})
	reportCreate(results, "4. Equipos diferentes", "Crear OSC-01 de 09:00 a 11:00", differentEquipment, err)

	if firstReservationID == 0 {
		report(results, "5. Cancelación", "Cancelar la primera reserva y conservarla con fecha", "estado CANCELLED, fecha registrada y fila persistida", "no se ejecutó: el escenario 1 no creó una reserva", false)
	} else {
		cancelled, cancelErr := service.Cancel(ctx, firstReservationID)
		if cancelErr != nil {
			report(results, "5. Cancelación", "Cancelar la primera reserva y conservarla con fecha", "estado CANCELLED, fecha registrada y fila persistida", "error: "+cancelErr.Error(), false)
		} else {
			stored, getErr := service.GetByID(ctx, firstReservationID)
			persisted := getErr == nil && stored.ID == firstReservationID
			validStatus := cancelled.Status == reservation.StatusCancelled && persisted && stored.Status == reservation.StatusCancelled
			persistedCancellationTime := persisted && cancelled.CancelledAt != nil && stored.CancelledAt != nil && stored.CancelledAt.Equal(*cancelled.CancelledAt)
			validCancellationTime := cancelled.CancelledAt != nil && !cancelled.CancelledAt.IsZero() && persistedCancellationTime
			actual := fmt.Sprintf("estado=%s; fecha registrada=%t; fila persistida=%t", cancelled.Status, validCancellationTime, persisted)
			report(results, "5. Cancelación", "Cancelar la primera reserva y conservarla con fecha", "estado CANCELLED, fecha registrada y fila persistida", actual, validStatus && validCancellationTime)
		}
	}

	released, err := service.Create(ctx, reservation.CreateInput{
		EquipmentID: "MIC-01",
		Requester:   "Verificación manual",
		StartsAt:    start0900,
		EndsAt:      end1100,
	})
	reportCreate(results, "6. Liberación del intervalo", "Volver a reservar MIC-01 de 09:00 a 11:00", released, err)

	expectRejected(results, service, "7a. Validación: solicitante vacío", reservation.CreateInput{
		EquipmentID: "MIC-02",
		Requester:   "   ",
		StartsAt:    start0900,
		EndsAt:      end1100,
	}, reservation.ErrInvalidInput)
	expectRejected(results, service, "7b. Validación: inicio posterior al fin", reservation.CreateInput{
		EquipmentID: "MIC-02",
		Requester:   "Verificación manual",
		StartsAt:    end1300,
		EndsAt:      start1100,
	}, reservation.ErrInvalidInput)
	expectRejected(results, service, "7c. Validación: fechas iguales", reservation.CreateInput{
		EquipmentID: "MIC-02",
		Requester:   "Verificación manual",
		StartsAt:    start0900,
		EndsAt:      start0900,
	}, reservation.ErrInvalidInput)
	expectRejected(results, service, "7d. Validación: equipo inexistente", reservation.CreateInput{
		EquipmentID: "NO-EXISTE",
		Requester:   "Verificación manual",
		StartsAt:    start0900,
		EndsAt:      end1100,
	}, reservation.ErrEquipmentNotFound)

	fmt.Printf("\nResumen: %d verificaciones correctas, %d fallidas\n", results.passed, results.failed)
	if results.failed > 0 {
		return 1
	}
	return 0
}

func prepareCleanDatabase(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	for _, suffix := range []string{"", "-journal", "-wal", "-shm"} {
		if err := os.Remove(path + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("eliminar %s: %w", path+suffix, err)
		}
	}
	return nil
}

func reportCreate(results *summary, title, description string, created reservation.Reservation, err error) {
	if err != nil {
		report(results, title, description, "creación exitosa", "error: "+err.Error(), false)
		return
	}
	report(results, title, description, "creación exitosa", fmt.Sprintf("creada con ID %d y estado %s", created.ID, created.Status), created.ID > 0 && created.Status == reservation.StatusActive)
}

func expectRejected(results *summary, service *reservation.Service, title string, input reservation.CreateInput, expectedError error) {
	_, err := service.Create(context.Background(), input)
	if err == nil {
		report(results, title, "Enviar datos inválidos", "rechazo de la operación", "operación aceptada inesperadamente", false)
		return
	}
	correct := errors.Is(err, expectedError)
	actual := "rechazada: " + err.Error()
	if !correct {
		actual = "rechazada por un motivo distinto: " + err.Error()
	}
	report(results, title, "Enviar datos inválidos", "rechazo de la operación", actual, correct)
}

func report(results *summary, title, description, expected, actual string, correct bool) {
	state := "ERROR"
	if correct {
		state = "CORRECTO"
		results.passed++
	} else {
		results.failed++
	}
	fmt.Printf("\n%s\n  Descripción: %s\n  Esperado: %s\n  Obtenido: %s\n  Estado: %s\n", title, description, expected, actual, state)
}
