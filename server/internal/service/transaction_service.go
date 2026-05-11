package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
)

var ErrSelectionStateConflict = errors.New("transaction is not in selection status")

// TransactionRepository определяет контракт хранилища транзакций.
type TransactionRepository interface {
	Create(tx *model.Transaction) (*model.Transaction, error)
	Get(id string) (*model.Transaction, error)
	Update(id string, apply func(*model.Transaction) error) (*model.Transaction, error)
	ListAll() ([]*model.Transaction, error)
}

type TransactionService struct {
	store        TransactionRepository
	prices       *PriceService
	priceLockTTL time.Duration
}

func NewTransactionService(store TransactionRepository, prices *PriceService, priceLockTTL time.Duration) *TransactionService {
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
	return s.store.Create(tx)
}

func (s *TransactionService) Get(id string) (*model.Transaction, error) {
	return s.store.Get(id)
}

// InactivityTimeoutResult возвращается клиенту в ответ на запрос таймаута неактивности.
type InactivityTimeoutResult struct {
	Cleared bool
	Status  model.TransactionStatus
	Reason  string
}

// InactivityTimeout проверяет состояние транзакции и безопасно завершает её,
// если это возможно. Вызывается клиентом при истечении таймаута неактивности.
func (s *TransactionService) InactivityTimeout(id string) (*InactivityTimeoutResult, error) {
	tx, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}

	switch tx.Status {
	case model.TransactionStatusSelection:
		if _, err := s.store.Update(id, func(t *model.Transaction) error {
			return t.Abandon("inactivity_timeout")
		}); err != nil {
			return nil, err
		}
		return &InactivityTimeoutResult{Cleared: true, Status: model.TransactionStatusAbandoned}, nil

	case model.TransactionStatusCompleted,
		model.TransactionStatusFailed,
		model.TransactionStatusAbandoned:
		return &InactivityTimeoutResult{Cleared: true, Status: tx.Status}, nil

	default:
		return &InactivityTimeoutResult{
			Cleared: false,
			Status:  tx.Status,
			Reason:  "transaction in progress, cannot be abandoned",
		}, nil
	}
}

// StartSweeper запускает фоновый горутин, который периодически помечает
// старые selection-транзакции как abandoned. Это fallback на случай, если
// клиент не успел отправить inactivity-timeout запрос (обрыв сети, краш браузера).
func (s *TransactionService) StartSweeper(ctx context.Context, selectionTTL time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sweepStaleSelections(selectionTTL)
			}
		}
	}()
}

func (s *TransactionService) sweepStaleSelections(ttl time.Duration) {
	threshold := time.Now().Add(-ttl)
	txs, err := s.store.ListAll()
	if err != nil {
		slog.Error("sweeper: failed to list transactions", "err", err)
		return
	}
	for _, tx := range txs {
		if tx.Status != model.TransactionStatusSelection || !tx.UpdatedAt.Before(threshold) {
			continue
		}
		if _, err := s.store.Update(tx.ID, func(t *model.Transaction) error {
			return t.Abandon("inactivity_timeout")
		}); err == nil {
			slog.Info("sweeper: abandoned stale selection transaction", "id", tx.ID)
		}
	}
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
