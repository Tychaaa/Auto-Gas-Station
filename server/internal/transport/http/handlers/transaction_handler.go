package handlers

import (
	"errors"
	"io"
	nethttp "net/http"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	transactions *service.TransactionService
	prices       *service.PriceService
	dispensers   *service.DispenserService
}

func NewTransactionHandler(transactions *service.TransactionService, prices *service.PriceService, dispensers *service.DispenserService) *TransactionHandler {
	return &TransactionHandler{transactions: transactions, prices: prices, dispensers: dispensers}
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var req dto.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	created, err := h.transactions.Create(req)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, created)
}

func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}
	tx, err := h.transactions.Get(id)
	if err != nil {
		writeTransactionError(c, err, nil)
		return
	}
	c.JSON(nethttp.StatusOK, tx)
}

func (h *TransactionHandler) UpdateSelection(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}

	var req dto.UpdateSelectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	updated, err := h.transactions.UpdateSelection(id, req)
	if err != nil {
		writeTransactionError(c, err, service.ErrSelectionStateConflict)
		return
	}
	c.JSON(nethttp.StatusOK, updated)
}

// InactivityTimeout обрабатывает таймаут неактивности терминала.
// Безопасно завершает транзакцию, если она находится в состоянии selection.
// Транзакции в процессе оплаты, фискализации и налива не прерываются.
func (h *TransactionHandler) InactivityTimeout(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}

	result, err := h.transactions.InactivityTimeout(id)
	if err != nil {
		writeTransactionError(c, err, nil)
		return
	}

	c.JSON(nethttp.StatusOK, dto.InactivityTimeoutResponse{
		Cleared: result.Cleared,
		Status:  string(result.Status),
		Reason:  result.Reason,
	})
}

func (h *TransactionHandler) FuelPrices(c *gin.Context) {
	prices, err := h.prices.ListCurrentPrices()
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	dispenserList, err := h.dispensers.ListDispensers()
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	priceByFuel := make(map[string]model.FuelPriceView, len(prices))
	for _, p := range prices {
		priceByFuel[p.FuelType] = p
	}

	// Iterate dispensers in sort order, include only enabled ones with a known price.
	result := make([]model.FuelPriceView, 0, len(dispenserList))
	for _, d := range dispenserList {
		if !d.Enabled || d.FuelType == "" {
			continue
		}
		p, ok := priceByFuel[d.FuelType]
		if !ok {
			continue
		}
		p.DispenserID = d.ID
		p.DispenserLabel = d.Label
		result = append(result, p)
	}

	c.JSON(nethttp.StatusOK, gin.H{"items": result})
}

func writeTransactionError(c *gin.Context, err error, conflict error) {
	switch {
	case errors.Is(err, repository.ErrTransactionNotFound):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
	case conflict != nil && errors.Is(err, conflict):
		c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
