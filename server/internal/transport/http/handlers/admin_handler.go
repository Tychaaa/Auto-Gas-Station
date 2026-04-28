package handlers

import (
	nethttp "net/http"
	"time"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	prices *service.PriceService
}

func NewAdminHandler(prices *service.PriceService) *AdminHandler {
	return &AdminHandler{prices: prices}
}

func (h *AdminHandler) ListPriceVersions(c *gin.Context) {
	if h.prices == nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": "price service is not configured"})
		return
	}

	versions, err := h.prices.ListVersions(0)
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": versions})
}

func (h *AdminHandler) CreatePriceVersion(c *gin.Context) {
	if h.prices == nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": "price service is not configured"})
		return
	}

	var req dto.AdminCreatePriceVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	prices := make(map[string]float64, len(req.Items))
	for _, item := range req.Items {
		prices[item.FuelType] = item.PricePerLiter
	}

	version, err := h.prices.CreatePriceVersion(req.VersionTag, req.EffectiveFrom, prices)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, version)
}

func (h *AdminHandler) ListTransactions(c *gin.Context) {
	now := time.Now()
	examples := []dto.AdminTransactionView{
		{
			ID:            "demo-tx-0001",
			CreatedAt:     now.Add(-2 * time.Hour).Format(time.RFC3339),
			FuelType:      "АИ-95",
			Liters:        31.7,
			AmountRub:     2064.94,
			Status:        string(model.TransactionStatusCompleted),
			PaymentStatus: string(model.PaymentStatusApproved),
			FiscalStatus:  string(model.FiscalStatusDone),
			ReceiptNumber: "000123",
		},
		{
			ID:            "demo-tx-0002",
			CreatedAt:     now.Add(-45 * time.Minute).Format(time.RFC3339),
			FuelType:      "ДТ",
			Liters:        12.0,
			AmountRub:     943.32,
			Status:        string(model.TransactionStatusFailed),
			PaymentStatus: string(model.PaymentStatusDeclined),
			FiscalStatus:  string(model.FiscalStatusNone),
			ErrorMessage:  "Карта отклонена банком",
		},
		{
			ID:            "demo-tx-0003",
			CreatedAt:     now.Add(-5 * time.Minute).Format(time.RFC3339),
			FuelType:      "АИ-92",
			Liters:        0,
			AmountRub:     1000.00,
			Status:        string(model.TransactionStatusPaymentPending),
			PaymentStatus: string(model.PaymentStatusPending),
			FiscalStatus:  string(model.FiscalStatusNone),
		},
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": examples})
}
