package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// adminListPriceVersionsHandler отдает историю версий цен админу
// Используется на странице "Цены" для отображения истории изменений
func adminListPriceVersionsHandler(c *gin.Context) {
	if priceService == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "price service is not configured"})
		return
	}

	versions, err := priceService.ListVersions(0)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": versions})
}

type adminCreatePriceVersionItem struct {
	FuelType      string  `json:"fuelType" binding:"required"`
	PricePerLiter float64 `json:"pricePerLiter" binding:"required"`
}

type adminCreatePriceVersionRequest struct {
	VersionTag    string                        `json:"versionTag"`
	EffectiveFrom time.Time                     `json:"effectiveFrom" binding:"required"`
	Items         []adminCreatePriceVersionItem `json:"items" binding:"required,min=1"`
}

// adminCreatePriceVersionHandler создает новую версию цен по запросу от UI
// UI присылает только fuelType + pricePerLiter для каждого топлива
// displayName и grade сервер берет из defaultFuelCatalog чтобы не дублировать справочник
func adminCreatePriceVersionHandler(c *gin.Context) {
	if priceService == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "price service is not configured"})
		return
	}

	var req adminCreatePriceVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	prices := make(map[string]float64, len(req.Items))
	for _, item := range req.Items {
		prices[item.FuelType] = item.PricePerLiter
	}

	version, err := priceService.CreatePriceVersion(req.VersionTag, req.EffectiveFrom, prices)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, version)
}

// adminTransactionView упрощенный DTO транзакции для таблицы в админке
// Формат намеренно плоский и стабильный: когда появится персистентный журнал
// транзакций в SQLite, контракт этой ручки не изменится
type adminTransactionView struct {
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

// adminListTransactionsHandler пока возвращает захардкоженные примеры транзакций
// TODO: заменить на чтение из персистентного журнала транзакций
// (см. план по SQLite-таблице transactions: пока что transactionStore
// хранит только активные транзакции в памяти и не годится для журнала).
func adminListTransactionsHandler(c *gin.Context) {
	now := time.Now()
	examples := []adminTransactionView{
		{
			ID:            "demo-tx-0001",
			CreatedAt:     now.Add(-2 * time.Hour).Format(time.RFC3339),
			FuelType:      "АИ-95",
			Liters:        31.7,
			AmountRub:     2064.94,
			Status:        string(TransactionStatusCompleted),
			PaymentStatus: string(PaymentStatusApproved),
			FiscalStatus:  string(FiscalStatusDone),
			ReceiptNumber: "000123",
		},
		{
			ID:            "demo-tx-0002",
			CreatedAt:     now.Add(-45 * time.Minute).Format(time.RFC3339),
			FuelType:      "ДТ",
			Liters:        12.0,
			AmountRub:     943.32,
			Status:        string(TransactionStatusFailed),
			PaymentStatus: string(PaymentStatusDeclined),
			FiscalStatus:  string(FiscalStatusNone),
			ErrorMessage:  "Карта отклонена банком",
		},
		{
			ID:            "demo-tx-0003",
			CreatedAt:     now.Add(-5 * time.Minute).Format(time.RFC3339),
			FuelType:      "АИ-92",
			Liters:        0,
			AmountRub:     1000.00,
			Status:        string(TransactionStatusPaymentPending),
			PaymentStatus: string(PaymentStatusPending),
			FiscalStatus:  string(FiscalStatusNone),
		},
	}
	c.JSON(http.StatusOK, gin.H{"items": examples})
}
