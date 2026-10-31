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
	apiHandler := api.NewHandler(reservationService, equipmentRepository)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/templates/index.html")
	})
	mux.HandleFunc("/", http.NotFound)

	server := &http.Server{
		Addr:              serverAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Aplicación local escuchando en http://localhost:8080 (127.0.0.1:8080)")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
