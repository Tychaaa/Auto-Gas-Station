package dto

// InactivityTimeoutResponse сообщает клиенту, можно ли очистить локальное
// состояние и вернуться на главный экран, или транзакция не может быть прервана.
type InactivityTimeoutResponse struct {
	Cleared bool   `json:"cleared"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
}

type CreateTransactionRequest struct {
	FuelType  string  `json:"fuelType"`
	OrderMode string  `json:"orderMode"`
	AmountRub int64   `json:"amountRub"`
	Liters    float64 `json:"liters"`
	Preset    string  `json:"preset"`
}

type UpdateSelectionRequest struct {
	FuelType  string  `json:"fuelType"`
	OrderMode string  `json:"orderMode"`
	AmountRub int64   `json:"amountRub"`
	Liters    float64 `json:"liters"`
	Preset    string  `json:"preset"`
}
