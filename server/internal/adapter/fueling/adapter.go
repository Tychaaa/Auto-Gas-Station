package fueling

import "context"

type Adapter interface {
	StartFueling(ctx context.Context, input StartInput) (StartResult, error)
	GetFuelingStatus(ctx context.Context, input StatusInput) (StatusResult, error)
	Check(ctx context.Context) (CheckResult, error)
}

type CheckResult struct {
	StatusCode      string
	ReasonCode      string
	ProviderStatus  string
	DispensedLiters float64
	Completed       bool
}

type StartInput struct {
	TransactionID  string
	PumpID         string
	NozzleID       string
	OrderMode      string
	AmountRub      int64
	Liters         float64
	UnitPriceMinor int64
	Scenario       string
}

type StartResult struct {
	SessionID       string
	ProviderStatus  string
	DispensedLiters float64
}

type StatusInput struct {
	SessionID string
	PumpID    string
	NozzleID  string
}

type StatusResult struct {
	SessionID       string
	ProviderStatus  string
	DispensedLiters float64
	Completed       bool
	Partial         bool
	Error           string
}
