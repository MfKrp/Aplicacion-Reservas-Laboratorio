package main

import (
	"context"
	"fmt"
	"log"

	"reservation-system/internal/storage"
)

func main() {
	db, err := storage.Initialize(context.Background())
	if err != nil {
		log.Fatalf("No se pudo inicializar la base de datos: %v", err)
	}
	defer db.Close()

	fmt.Println("Sistema de reservas de laboratorio en desarrollo")
}
