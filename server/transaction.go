package main

import (
	"errors"
	"time"
)

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

type PaymentStatus string

const (
	PaymentStatusNone     PaymentStatus = "none"
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusApproved PaymentStatus = "approved"
	PaymentStatusDeclined PaymentStatus = "declined"
)

type FiscalStatus string

const (
	FiscalStatusNone    FiscalStatus = "none"
	FiscalStatusPending FiscalStatus = "pending"
	FiscalStatusDone    FiscalStatus = "done"
	FiscalStatusFailed  FiscalStatus = "failed"
)

type Transaction struct {
	ID            string
	FuelType      string
	OrderMode     string // amount | liters | preset
	AmountRub     int64
	Liters        float64
	Preset        string
	Status        TransactionStatus
	PaymentStatus PaymentStatus
	FiscalStatus  FiscalStatus
	PaymentError  string
	FiscalError   string
	ReceiptNumber string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

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

func (t *Transaction) MarkPaid() error {
	if t.Status != TransactionStatusPaymentPending {
		return errors.New("paid is only allowed from payment_pending")
	}
	t.Status = TransactionStatusPaid
	t.PaymentStatus = PaymentStatusApproved
	t.UpdatedAt = time.Now()
	return nil
}

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

// TODO(Артём): отпуск топлива, состояние fueling, фактический объём.

func (t *Transaction) MarkFiscalizing() error {
	if t.Status != TransactionStatusFueling {
		return errors.New("fiscalizing is only allowed from fueling")
	}
	t.Status = TransactionStatusFiscalizing
	t.FiscalStatus = FiscalStatusPending
	t.UpdatedAt = time.Now()
	return nil
}

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
