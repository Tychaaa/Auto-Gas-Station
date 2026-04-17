package main

import (
	"errors"
	"time"
)

// Этап жизненного цикла транзакции на терминале
type TransactionStatus string

const (
	TransactionStatusSelection      TransactionStatus = "selection"
	TransactionStatusPaymentPending TransactionStatus = "payment_pending"
	TransactionStatusPaid           TransactionStatus = "paid"
	TransactionStatusFueling        TransactionStatus = "fueling"
	TransactionStatusFiscalizing    TransactionStatus = "fiscalizing"
	TransactionStatusCompleted      TransactionStatus = "completed"
	TransactionStatusFailed         TransactionStatus = "failed"
)

// Результат и стадия оплаты отдельно от этапа транзакции
type PaymentStatus string

const (
	PaymentStatusNone     PaymentStatus = "none"
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusApproved PaymentStatus = "approved"
	PaymentStatusDeclined PaymentStatus = "declined"
)

// Стадия фискализации и печати чека
type FiscalStatus string

const (
	FiscalStatusNone    FiscalStatus = "none"
	FiscalStatusPending FiscalStatus = "pending"
	FiscalStatusDone    FiscalStatus = "done"
	FiscalStatusFailed  FiscalStatus = "failed"
)

// Подэтап отпуска топлива при статусе TransactionStatusFueling
type FuelingStatus string

const (
	FuelingStatusNone                   FuelingStatus = "none"
	FuelingStatusStarting               FuelingStatus = "starting"
	FuelingStatusDispensing             FuelingStatus = "dispensing"
	FuelingStatusCompletedWaitingFiscal FuelingStatus = "completed_waiting_fiscal"
	FuelingStatusFailed                 FuelingStatus = "failed"
)

// Данные заказа и текущие статусы проведения
type Transaction struct {
	ID               string
	FuelType         string
	OrderMode        string // Способ заказа amount liters или preset
	AmountRub        int64
	Liters           float64
	Preset           string
	Status           TransactionStatus
	PaymentStatus    PaymentStatus
	FiscalStatus     FiscalStatus
	PaymentProvider  string
	PaymentSessionID string
	PaymentError     string
	FiscalError      string
	ReceiptNumber    string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	// Поля топливного контура после оплаты и до MarkFiscalizing
	FuelingStatus    FuelingStatus
	FuelingError     string
	FuelingSessionID string
	DispensedLiters  float64
	DispenseComplete bool
	DispensePartial  bool
}

// Проверяет топливо и ровно один вариант заказа сумма литры или пресет
func (t *Transaction) ValidateSelection() error {
	if t.FuelType == "" {
		return errors.New("fuel type is required")
	}

	n := 0
	if t.AmountRub > 0 {
		n++
	}
	if t.Liters > 0 {
		n++
	}
	if t.Preset != "" {
		n++
	}
	if n == 0 {
		return errors.New("exactly one order option is required")
	}
	if n > 1 {
		return errors.New("only one order option may be set")
	}

	switch t.OrderMode {
	case "amount", "liters", "preset":
	default:
		return errors.New("invalid order mode")
	}

	switch {
	case t.AmountRub > 0:
		if t.OrderMode != "amount" {
			return errors.New("order mode must be amount when amount is set")
		}
	case t.Liters > 0:
		if t.OrderMode != "liters" {
			return errors.New("order mode must be liters when liters are set")
		}
	case t.Preset != "":
		if t.OrderMode != "preset" {
			return errors.New("order mode must be preset when preset is set")
		}
	}

	return nil
}

// Начало оплаты только из selection с проверкой заказа
func (t *Transaction) MarkPaymentPending() error {
	if t.Status != TransactionStatusSelection {
		return errors.New("payment can only be started from selection")
	}
	if err := t.ValidateSelection(); err != nil {
		return err
	}
	t.Status = TransactionStatusPaymentPending
	t.PaymentStatus = PaymentStatusPending
	t.UpdatedAt = time.Now()
	return nil
}

// Успешная оплата только из payment_pending
func (t *Transaction) MarkPaid() error {
	if t.Status != TransactionStatusPaymentPending {
		return errors.New("paid is only allowed from payment_pending")
	}
	t.Status = TransactionStatusPaid
	t.PaymentStatus = PaymentStatusApproved
	t.UpdatedAt = time.Now()
	return nil
}

// Отказ или ошибка оплаты только из payment_pending текст в PaymentError
func (t *Transaction) MarkPaymentFailed(msg string) error {
	if t.Status != TransactionStatusPaymentPending {
		return errors.New("payment failure is only allowed from payment_pending")
	}
	t.Status = TransactionStatusFailed
	t.PaymentStatus = PaymentStatusDeclined
	t.PaymentError = msg
	t.UpdatedAt = time.Now()
	return nil
}

// Переход из paid в fueling sessionID выдает API отпуска или заглушка
func (t *Transaction) BeginFueling(sessionID string) error {
	if t.Status != TransactionStatusPaid {
		return errors.New("fueling can only be started from paid")
	}
	if sessionID == "" {
		return errors.New("fueling session id is required")
	}
	t.Status = TransactionStatusFueling
	t.FuelingStatus = FuelingStatusStarting
	t.FuelingSessionID = sessionID
	t.FuelingError = ""
	t.DispensedLiters = 0
	t.DispenseComplete = false
	t.DispensePartial = false
	t.UpdatedAt = time.Now()
	return nil
}

// После подтверждения старта от API переводит starting в dispensing
func (t *Transaction) MarkFuelingDispensing() error {
	if t.Status != TransactionStatusFueling {
		return errors.New("dispensing is only allowed during fueling")
	}
	if t.FuelingStatus != FuelingStatusStarting {
		return errors.New("dispensing can only start from fueling starting state")
	}
	t.FuelingStatus = FuelingStatusDispensing
	t.UpdatedAt = time.Now()
	return nil
}

// Обновляет текущий накопленный объем налива и не меняет TransactionStatus
func (t *Transaction) UpdateDispensedLiters(liters float64) error {
	if t.Status != TransactionStatusFueling {
		return errors.New("dispensed liters update is only allowed during fueling")
	}
	if t.FuelingStatus != FuelingStatusDispensing {
		return errors.New("dispensed liters can only be updated while dispensing")
	}
	if liters < 0 {
		return errors.New("dispensed liters cannot be negative")
	}
	t.DispensedLiters = liters
	t.UpdatedAt = time.Now()
	return nil
}

// Фиксирует факт отпуска и оставляет Status равным fueling до MarkFiscalizing
func (t *Transaction) CompleteFuelingDispense(actualLiters float64, partial bool) error {
	if t.Status != TransactionStatusFueling {
		return errors.New("complete dispense is only allowed during fueling")
	}
	if t.FuelingStatus != FuelingStatusDispensing {
		return errors.New("complete dispense requires active dispensing state")
	}
	if actualLiters < 0 {
		return errors.New("actual liters cannot be negative")
	}
	t.DispensedLiters = actualLiters
	t.DispenseComplete = true
	t.DispensePartial = partial
	t.FuelingStatus = FuelingStatusCompletedWaitingFiscal
	t.UpdatedAt = time.Now()
	return nil
}

// Ошибка налива или API переводит fueling в failed
func (t *Transaction) MarkFuelingFailed(msg string) error {
	if t.Status != TransactionStatusFueling {
		return errors.New("fueling failure is only allowed from fueling")
	}
	t.Status = TransactionStatusFailed
	t.FuelingStatus = FuelingStatusFailed
	t.FuelingError = msg
	t.UpdatedAt = time.Now()
	return nil
}

// Оплата прошла но отпуск нельзя начать например если API недоступен
func (t *Transaction) AbortFuelingFromPaid(msg string) error {
	if t.Status != TransactionStatusPaid {
		return errors.New("abort fueling from paid is only allowed from paid")
	}
	t.Status = TransactionStatusFailed
	t.FuelingStatus = FuelingStatusFailed
	t.FuelingError = msg
	t.UpdatedAt = time.Now()
	return nil
}

// Старт фискализации после этапа fueling
func (t *Transaction) MarkFiscalizing() error {
	if t.Status != TransactionStatusFueling {
		return errors.New("fiscalizing is only allowed from fueling")
	}
	t.Status = TransactionStatusFiscalizing
	t.FiscalStatus = FiscalStatusPending
	t.UpdatedAt = time.Now()
	return nil
}

// Успешный чек записывает номер в ReceiptNumber и переводит в completed
func (t *Transaction) MarkFiscalized(receipt string) error {
	if t.Status != TransactionStatusFiscalizing {
		return errors.New("fiscalized is only allowed from fiscalizing")
	}
	if receipt == "" {
		return errors.New("receipt number is required")
	}
	t.Status = TransactionStatusCompleted
	t.FiscalStatus = FiscalStatusDone
	t.ReceiptNumber = receipt
	t.UpdatedAt = time.Now()
	return nil
}

// Ошибка ККТ или чека сохраняет текст в FiscalError и переводит в failed
func (t *Transaction) MarkFiscalFailed(msg string) error {
	if t.Status != TransactionStatusFiscalizing {
		return errors.New("fiscal failure is only allowed from fiscalizing")
	}
	t.Status = TransactionStatusFailed
	t.FiscalStatus = FiscalStatusFailed
	t.FiscalError = msg
	t.UpdatedAt = time.Now()
	return nil
}
