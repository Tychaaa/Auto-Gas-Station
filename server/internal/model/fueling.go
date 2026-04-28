package model

type FuelingStatus string

const (
	FuelingStatusNone                   FuelingStatus = "none"
	FuelingStatusStarting               FuelingStatus = "starting"
	FuelingStatusDispensing             FuelingStatus = "dispensing"
	FuelingStatusCompletedWaitingFiscal FuelingStatus = "completed_waiting_fiscal"
	FuelingStatusFailed                 FuelingStatus = "failed"
)
