package payment

import "context"

type Adapter interface {
	StartPayment(ctx context.Context, input StartInput) (StartResult, error)
	GetPaymentStatus(ctx context.Context, input StatusInput) (StatusResult, error)
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
}
