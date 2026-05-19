package payment

import (
	"context"
	"errors"
)

var ErrAdapterUnavailable = errors.New("payment adapter is unavailable")

type Adapter interface {
	StartPayment(ctx context.Context, input StartInput) (StartResult, error)
	GetPaymentStatus(ctx context.Context, input StatusInput) (StatusResult, error)
	CancelPayment(ctx context.Context, sessionID string) error
}

type VendotekChecker interface {
	CheckVendotek(ctx context.Context) VendotekCheckResult
}

type StartInput struct {
	ExternalTransactionID string
	AmountMinor           int64
	Currency              string
}

type StartResult struct {
	SessionID string
	Status    string
}

type StatusInput struct {
	SessionID string
}

type StatusResult struct {
	SessionID string
	Status    string
	Error     string
	Slip      *PaymentSlip
}

type PaymentSlip struct {
	PAN          string `json:"pan"`
	RRN          string `json:"rrn"`
	ApprovalCode string `json:"approval_code"`
	Amount       int64  `json:"amount"`
	Date         string `json:"date"`
	POSEntryMode string `json:"pos_entry_mode"`
	AppLabel     string `json:"app_label"`
}

type VendotekCheckResult struct {
	Online       bool
	Status       string
	SerialNumber string
	LastOpID     string
	Info         string
	Error        string
}
