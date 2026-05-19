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
	Update(id int, fuelType string, enabled bool) (*model.Dispenser, error)
	Add() (*model.Dispenser, error)
	Delete(id int) error
	Count() (int, error)
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

func (s *DispenserService) UpdateDispenser(id int, fuelType string, enabled bool) (*model.Dispenser, error) {
	return s.repo.Update(id, fuelType, enabled)
}

func (s *DispenserService) AddDispenser(maxCount int) (*model.Dispenser, error) {
	count, err := s.repo.Count()
	if err != nil {
		return nil, fmt.Errorf("count dispensers: %w", err)
	}
	if count >= maxCount {
		return nil, fmt.Errorf("maximum number of dispensers (%d) reached", maxCount)
	}
	return s.repo.Add()
}

func (s *DispenserService) DeleteDispenser(id int) error {
	err := s.repo.Delete(id)
	if errors.Is(err, repository.ErrDispenserNotFound) {
		return repository.ErrDispenserNotFound
	}
	return err
}
