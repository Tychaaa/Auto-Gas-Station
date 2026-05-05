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

// WatchdogStatusView — представление состояния ESP32 watchdog для админки.
// Поле Mode принимает значения "serial" или "disabled". Online=false при
// потере связи с ESP32 (ESP32 перезагрузился, не подключён, не питается).
type WatchdogStatusView struct {
	Mode               string `json:"mode"`
	Online             bool   `json:"online"`
	LastHeartbeatAt    string `json:"lastHeartbeatAt"`
	LastHeartbeatAgoMs int64  `json:"lastHeartbeatAgoMs"`
	EspUptimeMs        int64  `json:"espUptimeMs"`
	LastError          string `json:"lastError"`
}

type AdminSystemRebootRequest struct {
	Method string `json:"method" binding:"required,oneof=soft hard"`
}
