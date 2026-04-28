package handlers

import (
	"errors"
	"io"
	nethttp "net/http"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	transactions *service.TransactionService
	prices       *service.PriceService
}

func NewTransactionHandler(transactions *service.TransactionService, prices *service.PriceService) *TransactionHandler {
	return &TransactionHandler{transactions: transactions, prices: prices}
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

func (h *TransactionHandler) FuelPrices(c *gin.Context) {
	prices, err := h.prices.ListCurrentPrices()
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"items": prices})
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
