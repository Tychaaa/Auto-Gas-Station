package dto

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
