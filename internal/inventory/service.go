package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"reservation-system/internal/reservation"
)

var (
	ErrInvalidInput  = errors.New("invalid equipment input")
	ErrAlreadyExists = errors.New("equipment already exists")
)

type RepositoryPort interface {
	List(context.Context) ([]reservation.Equipment, error)
	GetByID(context.Context, string) (reservation.Equipment, error)
	Create(context.Context, reservation.Equipment) (reservation.Equipment, error)
}

type Service struct {
	repository RepositoryPort
}

func NewService(repository RepositoryPort) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context) ([]reservation.Equipment, error) {
	return s.repository.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (reservation.Equipment, error) {
	return s.repository.GetByID(ctx, strings.TrimSpace(id))
}

func (s *Service) Create(ctx context.Context, id, name string) (reservation.Equipment, error) {
	item := reservation.Equipment{ID: strings.TrimSpace(id), Name: strings.TrimSpace(name)}
	if item.ID == "" || item.Name == "" {
		return reservation.Equipment{}, fmt.Errorf("%w: ID and name are required", ErrInvalidInput)
	}
	return s.repository.Create(ctx, item)
}
