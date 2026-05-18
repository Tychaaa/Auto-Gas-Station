package service

import (
	"errors"
	"fmt"

	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
)

type DispenserRepository interface {
	List() ([]*model.Dispenser, error)
	GetByFuelType(fuelType string) (*model.Dispenser, error)
	GetByID(id int) (*model.Dispenser, error)
	SetFuelType(id int, fuelType string) (*model.Dispenser, error)
}

type DispenserService struct {
	repo DispenserRepository
}

func NewDispenserService(repo DispenserRepository) *DispenserService {
	return &DispenserService{repo: repo}
}

func (s *DispenserService) ListDispensers() ([]*model.Dispenser, error) {
	return s.repo.List()
}

func (s *DispenserService) FindByFuelType(fuelType string) (*model.Dispenser, error) {
	return s.repo.GetByFuelType(fuelType)
}

func (s *DispenserService) GetDispenser(id int) (*model.Dispenser, error) {
	return s.repo.GetByID(id)
}

func (s *DispenserService) AssignFuelType(id int, fuelType string) (*model.Dispenser, error) {
	if fuelType != "" {
		existing, err := s.repo.GetByFuelType(fuelType)
		if err != nil && !errors.Is(err, repository.ErrDispenserNotFound) {
			return nil, fmt.Errorf("check fuel type conflict: %w", err)
		}
		if existing != nil && existing.ID != id {
			return nil, fmt.Errorf("fuel type %q is already assigned to dispenser %d", fuelType, existing.ID)
		}
	}
	return s.repo.SetFuelType(id, fuelType)
}
