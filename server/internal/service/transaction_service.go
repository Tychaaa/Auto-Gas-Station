package service

import (
	"errors"
	"time"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
)

var ErrSelectionStateConflict = errors.New("transaction is not in selection status")

type TransactionService struct {
	store        *repository.TransactionStore
	prices       *PriceService
	priceLockTTL time.Duration
}

func NewTransactionService(store *repository.TransactionStore, prices *PriceService, priceLockTTL time.Duration) *TransactionService {
	return &TransactionService{store: store, prices: prices, priceLockTTL: priceLockTTL}
}

func (s *TransactionService) Create(req dto.CreateTransactionRequest) (*model.Transaction, error) {
	tx := &model.Transaction{
		FuelType:      req.FuelType,
		OrderMode:     req.OrderMode,
		AmountRub:     req.AmountRub,
		Liters:        req.Liters,
		Preset:        req.Preset,
		Status:        model.TransactionStatusSelection,
		PaymentStatus: model.PaymentStatusNone,
		FiscalStatus:  model.FiscalStatusNone,
		FuelingStatus: model.FuelingStatusNone,
	}
	if err := tx.ValidateSelection(); err != nil {
		return nil, err
	}
	if err := s.prices.ApplySelectionPricing(tx, s.priceLockTTL); err != nil {
		return nil, err
	}
	return s.store.Create(tx), nil
}

func (s *TransactionService) Get(id string) (*model.Transaction, error) {
	tx, ok := s.store.Get(id)
	if !ok {
		return nil, repository.ErrTransactionNotFound
	}
	return tx, nil
}

func (s *TransactionService) UpdateSelection(id string, req dto.UpdateSelectionRequest) (*model.Transaction, error) {
	return s.store.Update(id, func(tx *model.Transaction) error {
		if tx.Status != model.TransactionStatusSelection {
			return ErrSelectionStateConflict
		}
		tx.FuelType = req.FuelType
		tx.OrderMode = req.OrderMode
		tx.AmountRub = req.AmountRub
		tx.Liters = req.Liters
		tx.Preset = req.Preset
		if err := tx.ValidateSelection(); err != nil {
			return err
		}
		return s.prices.ApplySelectionPricing(tx, s.priceLockTTL)
	})
}
