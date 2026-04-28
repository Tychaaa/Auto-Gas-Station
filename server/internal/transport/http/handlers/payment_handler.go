package handlers

import (
	"errors"
	nethttp "net/http"

	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	payments *service.PaymentService
}

func NewPaymentHandler(payments *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

func (h *PaymentHandler) Start(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}

	updated, err := h.payments.Start(c.Request.Context(), id)
	if err != nil {
		writePaymentError(c, err, service.ErrPaymentStartStateConflict)
		return
	}
	c.JSON(nethttp.StatusOK, updated)
}

func (h *PaymentHandler) Status(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}

	updated, err := h.payments.SyncStatus(c.Request.Context(), id)
	if err != nil {
		writePaymentError(c, err, service.ErrPaymentStatusStateConflict)
		return
	}
	c.JSON(nethttp.StatusOK, updated)
}

func writePaymentError(c *gin.Context, err error, conflict error) {
	switch {
	case errors.Is(err, repository.ErrTransactionNotFound):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
	case errors.Is(err, conflict):
		c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
