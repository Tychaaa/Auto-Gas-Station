package handlers

import (
	"errors"
	nethttp "net/http"
	"time"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	prices *service.PriceService
	txRepo service.TransactionRepository
}

func NewAdminHandler(prices *service.PriceService, txRepo service.TransactionRepository) *AdminHandler {
	return &AdminHandler{prices: prices, txRepo: txRepo}
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
	txs, err := h.txRepo.ListAll()
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]dto.AdminTransactionView, 0, len(txs))
	for _, tx := range txs {
		items = append(items, toAdminTransactionView(tx))
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": items})
}

func (h *AdminHandler) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	tx, err := h.txRepo.Get(id)
	if errors.Is(err, repository.ErrTransactionNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	events, err := h.txRepo.GetEvents(id)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, toAdminTransactionDetailsView(tx, events))
}

func toAdminTransactionView(tx *model.Transaction) dto.AdminTransactionView {
	return dto.AdminTransactionView{
		ID:            tx.ID,
		CreatedAt:     tx.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     tx.UpdatedAt.Format(time.RFC3339),
		FuelType:      tx.FuelType,
		OrderMode:     tx.OrderMode,
		Liters:        txDisplayLiters(tx),
		AmountRub:     txDisplayAmountRub(tx),
		Currency:      tx.Currency,
		Status:        string(tx.Status),
		PaymentStatus: string(tx.PaymentStatus),
		FiscalStatus:  string(tx.FiscalStatus),
		FuelingStatus: string(tx.FuelingStatus),
		ReceiptNumber: tx.ReceiptNumber,
		ErrorMessage:  txErrorMessage(tx),
	}
}

func toAdminTransactionDetailsView(tx *model.Transaction, events []model.TransactionEvent) dto.AdminTransactionDetailsView {
	eventDTOs := make([]dto.TransactionEventDTO, 0, len(events))
	for _, ev := range events {
		eventDTOs = append(eventDTOs, dto.TransactionEventDTO{
			EventType:  string(ev.EventType),
			OccurredAt: ev.OccurredAt.Format(time.RFC3339),
			Detail:     ev.Detail,
		})
	}

	return dto.AdminTransactionDetailsView{
		ID:                tx.ID,
		CreatedAt:         tx.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         tx.UpdatedAt.Format(time.RFC3339),
		FuelType:          tx.FuelType,
		OrderMode:         tx.OrderMode,
		AmountRub:         tx.AmountRub,
		Liters:            tx.Liters,
		Preset:            tx.Preset,
		PriceVersionTag:   tx.PriceVersionTag,
		UnitPriceRub:      float64(tx.UnitPriceMinor) / 100.0,
		ComputedAmountRub: float64(tx.ComputedAmountMinor) / 100.0,
		Currency:          tx.Currency,
		PricingSnapshotAt: formatOptionalTime(tx.PricingSnapshotAt),
		PriceLockedUntil:  formatOptionalTime(tx.PriceLockedUntil),
		PriceWasRepriced:  tx.PriceWasRepriced,
		Status:            string(tx.Status),
		PaymentStatus:     string(tx.PaymentStatus),
		FiscalStatus:      string(tx.FiscalStatus),
		FuelingStatus:     string(tx.FuelingStatus),
		PaymentProvider:   tx.PaymentProvider,
		PaymentSessionID:  tx.PaymentSessionID,
		PaymentError:      tx.PaymentError,
		ReceiptNumber:     tx.ReceiptNumber,
		FiscalError:       tx.FiscalError,
		FuelingSessionID:  tx.FuelingSessionID,
		DispensedLiters:   tx.DispensedLiters,
		DispenseComplete:  tx.DispenseComplete,
		DispensePartial:   tx.DispensePartial,
		FuelingError:      tx.FuelingError,
		AbandonReason:     tx.AbandonReason,
		Events:            eventDTOs,
	}
}

func txDisplayLiters(tx *model.Transaction) float64 {
	if tx.DispensedLiters > 0 {
		return tx.DispensedLiters
	}
	return tx.Liters
}

func txDisplayAmountRub(tx *model.Transaction) float64 {
	if tx.ComputedAmountMinor > 0 {
		return float64(tx.ComputedAmountMinor) / 100.0
	}
	return float64(tx.AmountRub)
}

func txErrorMessage(tx *model.Transaction) string {
	if tx.PaymentError != "" {
		return tx.PaymentError
	}
	if tx.FiscalError != "" {
		return tx.FiscalError
	}
	return tx.FuelingError
}

func formatOptionalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
