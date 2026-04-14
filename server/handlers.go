package main

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

var transactionStore = NewTransactionStore()
var ErrSelectionStateConflict = errors.New("transaction is not in selection status")

// Данные запроса на создание транзакции
type createTransactionRequest struct {
	FuelType  string  `json:"fuelType"`
	OrderMode string  `json:"orderMode"`
	AmountRub int64   `json:"amountRub"`
	Liters    float64 `json:"liters"`
	Preset    string  `json:"preset"`
}

// Данные запроса на изменение параметров выбора
type updateSelectionRequest struct {
	FuelType  string  `json:"fuelType"`
	OrderMode string  `json:"orderMode"`
	AmountRub int64   `json:"amountRub"`
	Liters    float64 `json:"liters"`
	Preset    string  `json:"preset"`
}

func createTransactionHandler(c *gin.Context) {
	var req createTransactionRequest
	// Читаем JSON из тела запроса
	// Пустое тело допускаем, остальные ошибки считаем некорректным запросом
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Создаем новую транзакцию со стартовыми статусами
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

	// Сохраняем транзакцию и возвращаем ее клиенту
	created := transactionStore.Create(tx)
	c.JSON(http.StatusCreated, created)
}

func getTransactionHandler(c *gin.Context) {
	// Берем id транзакции из адреса запроса
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transaction id is required",
		})
		return
	}

	tx, ok := transactionStore.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}

	// Возвращаем найденную транзакцию
	c.JSON(http.StatusOK, tx)
}

func updateSelectionHandler(c *gin.Context) {
	// Берем id транзакции из адреса запроса
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transaction id is required",
		})
		return
	}

	// Читаем новые параметры выбора из тела запроса
	var req updateSelectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Обновляем только транзакцию в статусе выбора
	updated, err := transactionStore.Update(id, func(tx *Transaction) error {
		if tx.Status != TransactionStatusSelection {
			return ErrSelectionStateConflict
		}

		// Перезаписываем выбранные пользователем значения
		tx.FuelType = req.FuelType
		tx.OrderMode = req.OrderMode
		tx.AmountRub = req.AmountRub
		tx.Liters = req.Liters
		tx.Preset = req.Preset

		// Проверяем что новые данные валидны
		return tx.ValidateSelection()
	})
	if err != nil {
		// Возвращаем код ошибки в зависимости от причины
		switch {
		case errors.Is(err, ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "transaction not found",
			})
		case errors.Is(err, ErrSelectionStateConflict):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	// Отправляем обновленную транзакцию
	c.JSON(http.StatusOK, updated)
}
