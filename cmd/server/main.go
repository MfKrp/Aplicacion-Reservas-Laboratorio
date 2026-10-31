package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"reservation-system/internal/api"
	"reservation-system/internal/inventory"
	"reservation-system/internal/reservation"
	"reservation-system/internal/storage"
)

const serverAddress = "127.0.0.1:8080"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	db, err := storage.Initialize(context.Background())
	if err != nil {
		return err
	}
	defer db.Close()

	reservationRepository := reservation.NewSQLiteRepository(db)
	equipmentRepository := inventory.NewRepository(db)
	reservationService := reservation.NewService(reservationRepository, equipmentRepository)
	handler := api.NewHandler(reservationService, equipmentRepository)

	server := &http.Server{
		Addr:              serverAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("API local escuchando en http://%s", serverAddress)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
