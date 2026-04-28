package handlers

import (
	"errors"
	"io"
	nethttp "net/http"
	"strings"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/payment"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

var (
	ErrSelectionStateConflict     = errors.New("transaction is not in selection status")
	ErrPaymentStartStateConflict  = errors.New("payment can only be started from selection")
	ErrPaymentStatusStateConflict = errors.New("payment status sync is only allowed from payment_pending")
)

type TransactionHandler struct {
	store        *repository.TransactionStore
	prices       *service.PriceService
	payments     payment.Adapter
	priceLockTTL time.Duration
}

func NewTransactionHandler(store *repository.TransactionStore, prices *service.PriceService, payments payment.Adapter, priceLockTTL time.Duration) *TransactionHandler {
	return &TransactionHandler{store: store, prices: prices, payments: payments, priceLockTTL: priceLockTTL}
}

type createTransactionRequest struct {
	FuelType  string  `json:"fuelType"`
	OrderMode string  `json:"orderMode"`
	AmountRub int64   `json:"amountRub"`
	Liters    float64 `json:"liters"`
	Preset    string  `json:"preset"`
}

type updateSelectionRequest struct {
	FuelType  string  `json:"fuelType"`
	OrderMode string  `json:"orderMode"`
	AmountRub int64   `json:"amountRub"`
	Liters    float64 `json:"liters"`
	Preset    string  `json:"preset"`
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var req createTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	tx := &model.Transaction{
		FuelType:      req.FuelType,
		OrderMode:     req.OrderMode,
		AmountRub:     req.AmountRub,
		Liters:        req.Liters,
		Preset:        req.Preset,
		Status:        model.TransactionStatusSelection,
		PaymentStatus: model.PaymentStatusNone,
		FiscalStatus:  model.FiscalStatusNone,
		FuelingStatus: model.FuelingStatusNone,
	}
	if err := tx.ValidateSelection(); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.prices.ApplySelectionPricing(tx, h.priceLockTTL); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created := h.store.Create(tx)
	c.JSON(nethttp.StatusCreated, created)
}

func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}
	tx, ok := h.store.Get(id)
	if !ok {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
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

	var req updateSelectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	updated, err := h.store.Update(id, func(tx *model.Transaction) error {
		if tx.Status != model.TransactionStatusSelection {
			return ErrSelectionStateConflict
		}
		tx.FuelType = req.FuelType
		tx.OrderMode = req.OrderMode
		tx.AmountRub = req.AmountRub
		tx.Liters = req.Liters
		tx.Preset = req.Preset
		if err := tx.ValidateSelection(); err != nil {
			return err
		}
		return h.prices.ApplySelectionPricing(tx, h.priceLockTTL)
	})
	if err != nil {
		h.writeStoreError(c, err, ErrSelectionStateConflict)
		return
	}
	c.JSON(nethttp.StatusOK, updated)
}

func (h *TransactionHandler) StartPayment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}

	txSnapshot, ok := h.store.Get(id)
	if !ok {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if txSnapshot.Status != model.TransactionStatusSelection {
		c.JSON(nethttp.StatusConflict, gin.H{"error": ErrPaymentStartStateConflict.Error()})
		return
	}

	pricingSnapshot, err := h.store.Update(id, func(tx *model.Transaction) error {
		if tx.Status != model.TransactionStatusSelection {
			return ErrPaymentStartStateConflict
		}
		if err := tx.ValidateSelection(); err != nil {
			return err
		}
		if tx.ComputedAmountMinor <= 0 || tx.UnitPriceMinor <= 0 || tx.Currency == "" {
			return h.prices.ApplySelectionPricing(tx, h.priceLockTTL)
		}
		_, err := h.prices.RepriceIfNeeded(tx, h.priceLockTTL, time.Now())
		return err
	})
	if err != nil {
		h.writeStoreError(c, err, ErrPaymentStartStateConflict)
		return
	}
	if pricingSnapshot.ComputedAmountMinor <= 0 {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "computed amount must be > 0 to start payment"})
		return
	}
	currency := pricingSnapshot.Currency
	if strings.TrimSpace(currency) == "" {
		currency = service.DefaultPricingCurrency
	}

	startResult, err := h.payments.StartPayment(c.Request.Context(), payment.StartInput{
		ExternalTransactionID: id,
		AmountMinor:           pricingSnapshot.ComputedAmountMinor,
		Currency:              currency,
	})
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.store.Update(id, func(tx *model.Transaction) error {
		if err := tx.MarkPaymentPending(); err != nil {
			return err
		}
		tx.PaymentProvider = "vendotek_mock"
		tx.PaymentSessionID = startResult.SessionID
		tx.PaymentError = ""
		return nil
	})
	if err != nil {
		h.writeStoreError(c, err, ErrPaymentStartStateConflict)
		return
	}

	sessionStatus, err := h.payments.GetPaymentStatus(c.Request.Context(), payment.StatusInput{SessionID: startResult.SessionID})
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	updated, err = h.store.Update(id, func(tx *model.Transaction) error {
		return applyPaymentStatusToTransaction(tx, sessionStatus)
	})
	if err != nil {
		h.writeStoreError(c, err, ErrPaymentStatusStateConflict)
		return
	}
	c.JSON(nethttp.StatusOK, updated)
}

func (h *TransactionHandler) PaymentStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}

	txSnapshot, ok := h.store.Get(id)
	if !ok {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if txSnapshot.PaymentSessionID == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "payment session id is required"})
		return
	}
	if txSnapshot.Status != model.TransactionStatusPaymentPending {
		c.JSON(nethttp.StatusOK, txSnapshot)
		return
	}

	sessionStatus, err := h.payments.GetPaymentStatus(c.Request.Context(), payment.StatusInput{SessionID: txSnapshot.PaymentSessionID})
	if err != nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.store.Update(id, func(tx *model.Transaction) error {
		return applyPaymentStatusToTransaction(tx, sessionStatus)
	})
	if err != nil {
		h.writeStoreError(c, err, ErrPaymentStatusStateConflict)
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

func (h *TransactionHandler) writeStoreError(c *gin.Context, err error, conflict error) {
	switch {
	case errors.Is(err, repository.ErrTransactionNotFound):
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
	case errors.Is(err, conflict):
		c.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func applyPaymentStatusToTransaction(tx *model.Transaction, status payment.StatusResult) error {
	if tx.Status != model.TransactionStatusPaymentPending {
		return ErrPaymentStatusStateConflict
	}

	switch strings.ToLower(strings.TrimSpace(status.Status)) {
	case "approved":
		if err := tx.MarkPaid(); err != nil {
			return err
		}
		tx.PaymentError = ""
	case "declined", "timeout", "cancelled":
		msg := strings.TrimSpace(status.Error)
		if msg == "" {
			msg = defaultPaymentFailureMessage(status.Status)
		}
		if err := tx.MarkPaymentFailed(msg); err != nil {
			return err
		}
	case "created", "pending", "processing":
	default:
	}
	return nil
}

func defaultPaymentFailureMessage(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "timeout":
		return "payment timeout"
	case "cancelled":
		return "payment cancelled"
	default:
		return "payment declined"
	}
}
