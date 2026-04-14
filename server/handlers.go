package main

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

var transactionStore = NewTransactionStore()

// createTransactionRequest описывает данные для создания транзакции.
type createTransactionRequest struct {
	FuelType  string  `json:"fuelType"`
	OrderMode string  `json:"orderMode"`
	AmountRub int64   `json:"amountRub"`
	Liters    float64 `json:"liters"`
	Preset    string  `json:"preset"`
}

func createTransactionHandler(c *gin.Context) {
	var req createTransactionRequest
	// Читаем JSON. Пустое тело допускаем, остальные ошибки считаем некорректным запросом.
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Создаем новую транзакцию со стартовыми статусами.
	tx := &Transaction{
		FuelType:      req.FuelType,
		OrderMode:     req.OrderMode,
		AmountRub:     req.AmountRub,
		Liters:        req.Liters,
		Preset:        req.Preset,
		Status:        TransactionStatusSelection,
		PaymentStatus: PaymentStatusNone,
		FiscalStatus:  FiscalStatusNone,
		FuelingStatus: FuelingStatusNone,
	}

	// Сохраняем транзакцию и возвращаем ее клиенту.
	created := transactionStore.Create(tx)
	c.JSON(http.StatusCreated, created)
}
