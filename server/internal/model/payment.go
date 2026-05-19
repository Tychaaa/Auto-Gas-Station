package model

const (
	PaymentProviderVendotekMock  = "vendotek_mock"
	PaymentProviderVendotekEzPOS = "vendotek_ezpos"
)

type PaymentSession struct {
	Provider  string
	SessionID string
	Status    PaymentStatus
	Error     string
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
