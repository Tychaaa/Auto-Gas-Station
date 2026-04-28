package dto

import "time"

type AdminCreatePriceVersionItem struct {
	FuelType      string  `json:"fuelType" binding:"required"`
	PricePerLiter float64 `json:"pricePerLiter" binding:"required"`
}

type AdminCreatePriceVersionRequest struct {
	VersionTag    string                        `json:"versionTag"`
	EffectiveFrom time.Time                     `json:"effectiveFrom" binding:"required"`
	Items         []AdminCreatePriceVersionItem `json:"items" binding:"required,min=1"`
}

type AdminTransactionView struct {
	ID            string  `json:"id"`
	CreatedAt     string  `json:"createdAt"`
	FuelType      string  `json:"fuelType"`
	Liters        float64 `json:"liters"`
	AmountRub     float64 `json:"amountRub"`
	Status        string  `json:"status"`
	PaymentStatus string  `json:"paymentStatus"`
	FiscalStatus  string  `json:"fiscalStatus"`
	ReceiptNumber string  `json:"receiptNumber"`
	ErrorMessage  string  `json:"errorMessage"`
}
