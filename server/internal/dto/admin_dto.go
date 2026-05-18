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
	UpdatedAt     string  `json:"updatedAt"`
	FuelType      string  `json:"fuelType"`
	OrderMode     string  `json:"orderMode"`
	Liters        float64 `json:"liters"`
	AmountRub     float64 `json:"amountRub"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	PaymentStatus string  `json:"paymentStatus"`
	FiscalStatus  string  `json:"fiscalStatus"`
	FuelingStatus string  `json:"fuelingStatus"`
	ReceiptNumber string  `json:"receiptNumber"`
	ErrorMessage  string  `json:"errorMessage"`
}

type TransactionEventDTO struct {
	EventType  string `json:"eventType"`
	OccurredAt string `json:"occurredAt"`
	Detail     string `json:"detail,omitempty"`
}

type AdminTransactionDetailsView struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	// Заказ
	FuelType  string  `json:"fuelType"`
	OrderMode string  `json:"orderMode"`
	AmountRub int64   `json:"amountRub"`
	Liters    float64 `json:"liters"`
	Preset    string  `json:"preset"`
	// Snapshot цены
	PriceVersionTag   string  `json:"priceVersionTag"`
	UnitPriceRub      float64 `json:"unitPriceRub"`
	ComputedAmountRub float64 `json:"computedAmountRub"`
	Currency          string  `json:"currency"`
	PricingSnapshotAt string  `json:"pricingSnapshotAt"`
	PriceLockedUntil  string  `json:"priceLockedUntil"`
	PriceWasRepriced  bool    `json:"priceWasRepriced"`
	// Статусы
	Status        string `json:"status"`
	PaymentStatus string `json:"paymentStatus"`
	FiscalStatus  string `json:"fiscalStatus"`
	FuelingStatus string `json:"fuelingStatus"`
	// Оплата
	PaymentProvider  string `json:"paymentProvider"`
	PaymentSessionID string `json:"paymentSessionId"`
	PaymentError     string `json:"paymentError"`
	// Фискализация
	ReceiptNumber string `json:"receiptNumber"`
	FiscalError   string `json:"fiscalError"`
	// Налив
	FuelingSessionID string  `json:"fuelingSessionId"`
	DispensedLiters  float64 `json:"dispensedLiters"`
	DispenseComplete bool    `json:"dispenseComplete"`
	DispensePartial  bool    `json:"dispensePartial"`
	FuelingError     string  `json:"fuelingError"`
	// Прочее
	AbandonReason string `json:"abandonReason"`
	// Журнал событий жизненного цикла
	Events []TransactionEventDTO `json:"events"`
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

type EquipmentDispenserCheckView struct {
	Online          bool      `json:"online"`
	StatusCode      string    `json:"statusCode,omitempty"`
	ReasonCode      string    `json:"reasonCode,omitempty"`
	ProviderStatus  string    `json:"providerStatus,omitempty"`
	DispensedLiters float64   `json:"dispensedLiters,omitempty"`
	Completed       bool      `json:"completed,omitempty"`
	Error           string    `json:"error,omitempty"`
	CheckedAt       time.Time `json:"checkedAt"`
}

type EquipmentKKTCheckView struct {
	Online        bool      `json:"online"`
	Mode          uint8     `json:"mode"`
	Submode       uint8     `json:"submode"`
	IsShiftOpen   bool      `json:"isShiftOpen"`
	IsReceiptOpen bool      `json:"isReceiptOpen"`
	Error         string    `json:"error,omitempty"`
	CheckedAt     time.Time `json:"checkedAt"`
}
