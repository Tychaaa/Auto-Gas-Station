package model

const PaymentProviderVendotekMock = "vendotek_mock"

type PaymentSession struct {
	Provider  string
	SessionID string
	Status    PaymentStatus
	Error     string
}
