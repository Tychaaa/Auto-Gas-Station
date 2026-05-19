package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"AUTO-GAS-STATION/server/internal/adapter/fiscal"
	"AUTO-GAS-STATION/server/internal/model"
)

// ErrFiscalizationNotApplicable - транзакция не в том состоянии, чтобы её фискализировать.
var ErrFiscalizationNotApplicable = errors.New("transaction is not eligible for fiscalization")

// FiscalService - оркестратор фискализации после успешной оплаты.
// Не делает сетевых вызовов к ККТ под mutex хранилища.
type FiscalService struct {
	store   TransactionRepository
	adapter fiscal.Adapter
}

// NewFiscalService возвращает сервис. Если adapter == nil, фискализация будет
// возвращать ErrFiscalizationAdapterUnavailable - удобно для прогонов без ККТ.
func NewFiscalService(store TransactionRepository, adapter fiscal.Adapter) *FiscalService {
	return &FiscalService{store: store, adapter: adapter}
}

// ErrFiscalizationAdapterUnavailable - адаптер ККТ не сконфигурирован.
var ErrFiscalizationAdapterUnavailable = errors.New("fiscal adapter is not configured")

// FiscalizePaid запускает чек по уже оплаченной транзакции. Возвращает финальный
// snapshot транзакции (paid + FiscalStatus=done при успехе или failed при ошибке)
// и ошибку, если фискализация не удалась.
//
// Сетевой вызов к ККТ выполняется ВНЕ Update-блока, чтобы не держать mutex.
func (s *FiscalService) FiscalizePaid(ctx context.Context, id string) (*model.Transaction, error) {
	if s.adapter == nil {
		return nil, ErrFiscalizationAdapterUnavailable
	}

	tx, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if !canStartFiscalization(tx) {
		return tx, ErrFiscalizationNotApplicable
	}

	input, err := buildReceiptInput(tx)
	if err != nil {
		failed, updateErr := s.store.Update(id, func(tx *model.Transaction) error {
			if tx.Status != model.TransactionStatusPaid {
				return ErrFiscalizationNotApplicable
			}
			tx.Status = model.TransactionStatusFailed
			tx.FiscalStatus = model.FiscalStatusFailed
			tx.FiscalError = err.Error()
			return nil
		})
		if updateErr != nil {
			return nil, updateErr
		}
		return failed, fmt.Errorf("build receipt input: %w", err)
	}

	moved, err := s.store.Update(id, func(tx *model.Transaction) error {
		if !canStartFiscalization(tx) {
			return ErrFiscalizationNotApplicable
		}
		return tx.BeginFiscalizationFromPaid()
	})
	if err != nil {
		return nil, err
	}
	_ = moved

	result, fiscalErr := s.adapter.Fiscalize(ctx, input)
	if fiscalErr != nil {
		failed, updateErr := s.store.Update(id, func(tx *model.Transaction) error {
			return tx.MarkFiscalFailed(fiscalErr.Error())
		})
		if updateErr != nil {
			return nil, updateErr
		}
		return failed, fiscalErr
	}

	receiptNumber := formatReceiptNumber(result)
	updated, err := s.store.Update(id, func(tx *model.Transaction) error {
		return tx.MarkPaidFiscalized(receiptNumber)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// canStartFiscalization - true если транзакция в paid и чек ещё не выпущен.
func canStartFiscalization(tx *model.Transaction) bool {
	if tx == nil {
		return false
	}
	if tx.Status != model.TransactionStatusPaid {
		return false
	}
	switch tx.FiscalStatus {
	case model.FiscalStatusNone, model.FiscalStatusFailed:
		return true
	default:
		return false
	}
}

// buildReceiptInput собирает ReceiptInput только из данных транзакции и снапшота цены.
func buildReceiptInput(tx *model.Transaction) (fiscal.ReceiptInput, error) {
	if tx == nil {
		return fiscal.ReceiptInput{}, errors.New("transaction is nil")
	}
	if tx.UnitPriceMinor <= 0 {
		return fiscal.ReceiptInput{}, errors.New("transaction has no unit price snapshot")
	}
	if tx.ComputedAmountMinor <= 0 {
		return fiscal.ReceiptInput{}, errors.New("transaction has no computed amount")
	}

	good := strings.TrimSpace(tx.FuelType)
	if good == "" {
		return fiscal.ReceiptInput{}, errors.New("transaction has empty fuel type")
	}

	quantityMicro, err := computeQuantityMicro(tx)
	if err != nil {
		return fiscal.ReceiptInput{}, err
	}

	paymentKind, err := paymentKindForTransaction(tx)
	if err != nil {
		return fiscal.ReceiptInput{}, err
	}

	return fiscal.ReceiptInput{
		TransactionID:  tx.ID,
		GoodName:       good,
		QuantityMicro:  quantityMicro,
		UnitPriceMinor: tx.UnitPriceMinor,
		TotalMinor:     tx.ComputedAmountMinor,
		PaymentKind:    paymentKind,
		RoundingMinor:  0,
	}, nil
}

// computeQuantityMicro считает литры * 1_000_000 для FF46h:
//   - если в заказе явно указаны литры - берём их;
//   - иначе восстанавливаем литры из (ComputedAmountMinor / UnitPriceMinor).
//
// Округление до 6-го знака идёт math.Round(...), чтобы не терять копейки.
func computeQuantityMicro(tx *model.Transaction) (int64, error) {
	if tx.Liters > 0 {
		q := int64(math.Round(tx.Liters * 1_000_000))
		if q <= 0 {
			return 0, errors.New("computed quantity is not positive")
		}
		return q, nil
	}
	if tx.UnitPriceMinor <= 0 {
		return 0, errors.New("cannot compute quantity without unit price")
	}
	q := int64(math.Round(float64(tx.ComputedAmountMinor) * 1_000_000.0 / float64(tx.UnitPriceMinor)))
	if q <= 0 {
		return 0, errors.New("computed quantity is not positive")
	}
	return q, nil
}

// paymentKindForTransaction маппит провайдера оплаты в kind для FF45h.
// В текущем сценарии терминал Vendotek - всегда безналичный платёж.
func paymentKindForTransaction(tx *model.Transaction) (fiscal.PaymentKind, error) {
	switch strings.TrimSpace(tx.PaymentProvider) {
	case model.PaymentProviderVendotekMock, model.PaymentProviderVendotekEzPOS:
		return fiscal.PaymentCashless, nil
	case "":
		return "", errors.New("transaction has no payment provider")
	default:
		// На будущее - здесь можно добавить наличный кассовый канал.
		return fiscal.PaymentCashless, nil
	}
}

// formatReceiptNumber - человекочитаемый номер чека для UI / логов.
func formatReceiptNumber(r fiscal.Result) string {
	return fmt.Sprintf("ФД %d / ФП %d (смена %d, чек %d)",
		r.FDNumber, r.FiscalSign, r.ShiftNumber, r.ReceiptNumber)
}
