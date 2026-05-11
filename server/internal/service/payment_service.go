package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/payment"
	"AUTO-GAS-STATION/server/internal/model"
)

var (
	ErrPaymentStartStateConflict  = errors.New("payment can only be started from selection")
	ErrPaymentStatusStateConflict = errors.New("payment status sync is only allowed from payment_pending")
)

type PaymentService struct {
	store        TransactionRepository
	prices       *PriceService
	payments     payment.Adapter
	fiscal       *FiscalService
	priceLockTTL time.Duration
}

func NewPaymentService(
	store TransactionRepository,
	prices *PriceService,
	payments payment.Adapter,
	fiscalService *FiscalService,
	priceLockTTL time.Duration,
) *PaymentService {
	return &PaymentService{
		store:        store,
		prices:       prices,
		payments:     payments,
		fiscal:       fiscalService,
		priceLockTTL: priceLockTTL,
	}
}

func (s *PaymentService) Start(ctx context.Context, id string) (*model.Transaction, error) {
	txSnapshot, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if txSnapshot.Status != model.TransactionStatusSelection {
		return nil, ErrPaymentStartStateConflict
	}

	pricingSnapshot, err := s.store.Update(id, func(tx *model.Transaction) error {
		if tx.Status != model.TransactionStatusSelection {
			return ErrPaymentStartStateConflict
		}
		if err := tx.ValidateSelection(); err != nil {
			return err
		}
		if tx.ComputedAmountMinor <= 0 || tx.UnitPriceMinor <= 0 || tx.Currency == "" {
			return s.prices.ApplySelectionPricing(tx, s.priceLockTTL)
		}
		_, err := s.prices.RepriceIfNeeded(tx, s.priceLockTTL, time.Now())
		return err
	})
	if err != nil {
		return nil, err
	}
	if pricingSnapshot.ComputedAmountMinor <= 0 {
		return nil, errors.New("computed amount must be > 0 to start payment")
	}

	currency := pricingSnapshot.Currency
	if strings.TrimSpace(currency) == "" {
		currency = DefaultPricingCurrency
	}
	startResult, err := s.payments.StartPayment(ctx, payment.StartInput{
		ExternalTransactionID: id,
		AmountMinor:           pricingSnapshot.ComputedAmountMinor,
		Currency:              currency,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.store.Update(id, func(tx *model.Transaction) error {
		if err := tx.MarkPaymentPending(); err != nil {
			return err
		}
		tx.PaymentProvider = model.PaymentProviderVendotekMock
		tx.PaymentSessionID = startResult.SessionID
		tx.PaymentError = ""
		return nil
	})
	if err != nil {
		return nil, err
	}

	sessionStatus, err := s.payments.GetPaymentStatus(ctx, payment.StatusInput{SessionID: startResult.SessionID})
	if err != nil {
		return nil, err
	}
	updated, err = s.store.Update(id, func(tx *model.Transaction) error {
		return applyPaymentStatusToTransaction(tx, sessionStatus)
	})
	if err != nil {
		return nil, err
	}
	return s.maybeFiscalizeAfterPayment(ctx, updated)
}

func (s *PaymentService) SyncStatus(ctx context.Context, id string) (*model.Transaction, error) {
	txSnapshot, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if txSnapshot.PaymentSessionID == "" {
		return nil, errors.New("payment session id is required")
	}
	if txSnapshot.Status != model.TransactionStatusPaymentPending {
		return txSnapshot, nil
	}

	sessionStatus, err := s.payments.GetPaymentStatus(ctx, payment.StatusInput{SessionID: txSnapshot.PaymentSessionID})
	if err != nil {
		return nil, err
	}
	updated, err := s.store.Update(id, func(tx *model.Transaction) error {
		return applyPaymentStatusToTransaction(tx, sessionStatus)
	})
	if err != nil {
		return nil, err
	}
	return s.maybeFiscalizeAfterPayment(ctx, updated)
}

// maybeFiscalizeAfterPayment запускает фискальный чек, если оплата только что
// получила статус approved и FiscalService подключён. Сетевой вызов к ККТ
// проходит вне репозиторного Update.
func (s *PaymentService) maybeFiscalizeAfterPayment(ctx context.Context, tx *model.Transaction) (*model.Transaction, error) {
	if tx == nil {
		return nil, nil
	}
	if s.fiscal == nil {
		return tx, nil
	}
	if tx.Status != model.TransactionStatusPaid {
		return tx, nil
	}
	if tx.FiscalStatus != model.FiscalStatusNone && tx.FiscalStatus != model.FiscalStatusFailed {
		return tx, nil
	}
	updated, err := s.fiscal.FiscalizePaid(ctx, tx.ID)
	if err != nil {
		if updated != nil {
			return updated, err
		}
		return tx, err
	}
	return updated, nil
}

func applyPaymentStatusToTransaction(tx *model.Transaction, status payment.StatusResult) error {
	if tx.Status != model.TransactionStatusPaymentPending {
		return ErrPaymentStatusStateConflict
	}

	switch strings.ToLower(strings.TrimSpace(status.Status)) {
	case "approved":
		if err := tx.MarkPaid(); err != nil {
			return err
		}
		tx.PaymentError = ""
	case "declined", "timeout", "cancelled":
		msg := strings.TrimSpace(status.Error)
		if msg == "" {
			msg = defaultPaymentFailureMessage(status.Status)
		}
		if err := tx.MarkPaymentFailed(msg); err != nil {
			return err
		}
	case "created", "pending", "processing":
	default:
	}
	return nil
}

func defaultPaymentFailureMessage(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "timeout":
		return "payment timeout"
	case "cancelled":
		return "payment cancelled"
	default:
		return "payment declined"
	}
}
