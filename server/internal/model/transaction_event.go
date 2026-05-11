package model

import "time"

type TransactionEventType string

const (
	TxEventCreated            TransactionEventType = "created"
	TxEventSelectionUpdated   TransactionEventType = "selection_updated"
	TxEventPaymentStarted     TransactionEventType = "payment_started"
	TxEventPaymentApproved    TransactionEventType = "payment_approved"
	TxEventPaymentDeclined    TransactionEventType = "payment_declined"
	TxEventFiscalizingStarted TransactionEventType = "fiscalizing_started"
	TxEventReceiptIssued      TransactionEventType = "receipt_issued"
	TxEventFiscalFailed       TransactionEventType = "fiscal_failed"
	TxEventFuelingStarted     TransactionEventType = "fueling_started"
	TxEventFuelingDispensing  TransactionEventType = "fueling_dispensing"
	TxEventFuelingCompleted   TransactionEventType = "fueling_completed"
	TxEventFuelingFailed      TransactionEventType = "fueling_failed"
	TxEventCompleted          TransactionEventType = "completed"
	TxEventFailed             TransactionEventType = "failed"
	TxEventAbandoned          TransactionEventType = "abandoned"
)

type TransactionEvent struct {
	ID            int64
	TransactionID string
	EventType     TransactionEventType
	OccurredAt    time.Time
	Detail        string
}
