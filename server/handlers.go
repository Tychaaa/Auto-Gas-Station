package main

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var transactionStore = NewTransactionStore()
var ErrSelectionStateConflict = errors.New("transaction is not in selection status")
var ErrPaymentStartStateConflict = errors.New("payment can only be started from selection")
var ErrPaymentApproveStateConflict = errors.New("paid is only allowed from payment_pending")
var ErrPaymentDeclineStateConflict = errors.New("payment failure is only allowed from payment_pending")
var ErrPaymentStatusStateConflict = errors.New("payment status sync is only allowed from payment_pending")

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

func paymentStartHandler(c *gin.Context) {
	// Берем id транзакции из адреса запроса
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transaction id is required",
		})
		return
	}

	// Проверяем что платежный адаптер подключен
	if paymentAdapter == nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "payment adapter is not configured",
		})
		return
	}

	// Ищем транзакцию и проверяем сумму для оплаты
	txSnapshot, ok := transactionStore.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}
	if txSnapshot.AmountRub <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "amountRub must be > 0 to start payment",
		})
		return
	}

	// Запускаем платеж во внешнем адаптере
	startResult, err := paymentAdapter.StartPayment(c.Request.Context(), PaymentStartInput{
		ExternalTransactionID: id,
		AmountMinor:           txSnapshot.AmountRub * 100,
		Currency:              "RUB",
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Сначала сохраняем факт старта оплаты и session id
	updated, err := transactionStore.Update(id, func(tx *Transaction) error {
		if err := tx.MarkPaymentPending(); err != nil {
			return err
		}
		tx.PaymentProvider = "vendotek_mock"
		tx.PaymentSessionID = startResult.SessionID
		tx.PaymentError = ""
		return nil
	})
	if err != nil {
		// Возвращаем понятную ошибку в зависимости от причины
		switch {
		case errors.Is(err, ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "transaction not found",
			})
		case err.Error() == ErrPaymentStartStateConflict.Error():
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

	// Делаем первый sync статуса сессии сразу после старта
	sessionStatus, err := paymentAdapter.GetPaymentStatus(c.Request.Context(), PaymentStatusInput{
		SessionID: startResult.SessionID,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	updated, err = transactionStore.Update(id, func(tx *Transaction) error {
		return applyVendotekStatusToTransaction(tx, sessionStatus)
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "transaction not found",
			})
		case err.Error() == ErrPaymentStatusStateConflict.Error():
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

	// Возвращаем транзакцию после первого status-sync
	c.JSON(http.StatusOK, updated)
}

func paymentStatusHandler(c *gin.Context) {
	// Берем id транзакции из адреса запроса
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transaction id is required",
		})
		return
	}

	// Проверяем что платежный адаптер подключен
	if paymentAdapter == nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "payment adapter is not configured",
		})
		return
	}

	// Ищем транзакцию и данные платежной сессии
	txSnapshot, ok := transactionStore.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}
	if txSnapshot.PaymentSessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "payment session id is required",
		})
		return
	}
	// После финализации платежа повторно статус не применяем
	if txSnapshot.Status != TransactionStatusPaymentPending {
		c.JSON(http.StatusOK, txSnapshot)
		return
	}

	// Получаем актуальный статус сессии из Vendotek
	sessionStatus, err := paymentAdapter.GetPaymentStatus(c.Request.Context(), PaymentStatusInput{
		SessionID: txSnapshot.PaymentSessionID,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	updated, err := transactionStore.Update(id, func(tx *Transaction) error {
		return applyVendotekStatusToTransaction(tx, sessionStatus)
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "transaction not found",
			})
		case err.Error() == ErrPaymentStatusStateConflict.Error():
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

	c.JSON(http.StatusOK, updated)
}

func applyVendotekStatusToTransaction(tx *Transaction, status PaymentStatusResult) error {
	if tx.Status != TransactionStatusPaymentPending {
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
		// Оставляем payment_pending без изменения
	default:
		// Неизвестный статус не финализирует оплату
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

func paymentApproveHandler(c *gin.Context) {
	// Берем id транзакции из адреса запроса
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transaction id is required",
		})
		return
	}

	// Проверяем что платежный адаптер подключен
	if paymentAdapter == nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "payment adapter is not configured",
		})
		return
	}

	// Ищем транзакцию и проверяем данные для подтверждения оплаты
	txSnapshot, ok := transactionStore.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}
	if txSnapshot.PaymentSessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "payment session id is required",
		})
		return
	}
	if txSnapshot.Status != TransactionStatusPaymentPending {
		c.JSON(http.StatusConflict, gin.H{
			"error": ErrPaymentApproveStateConflict.Error(),
		})
		return
	}

	// Подтверждаем платеж во внешнем адаптере
	_, err := paymentAdapter.ApprovePayment(c.Request.Context(), PaymentApproveInput{
		SessionID: txSnapshot.PaymentSessionID,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Обновляем локальное состояние транзакции после подтверждения оплаты
	updated, err := transactionStore.Update(id, func(tx *Transaction) error {
		if err := tx.MarkPaid(); err != nil {
			return err
		}
		tx.PaymentError = ""
		return nil
	})
	if err != nil {
		// Возвращаем понятную ошибку в зависимости от причины
		switch {
		case errors.Is(err, ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "transaction not found",
			})
		case err.Error() == ErrPaymentApproveStateConflict.Error():
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

	// Возвращаем обновленную транзакцию клиенту
	c.JSON(http.StatusOK, updated)
}

func paymentDeclineHandler(c *gin.Context) {
	// Берем id транзакции из адреса запроса
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transaction id is required",
		})
		return
	}

	// Проверяем что платежный адаптер подключен
	if paymentAdapter == nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "payment adapter is not configured",
		})
		return
	}

	// Ищем транзакцию и проверяем данные для отклонения оплаты
	txSnapshot, ok := transactionStore.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}
	if txSnapshot.PaymentSessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "payment session id is required",
		})
		return
	}
	if txSnapshot.Status != TransactionStatusPaymentPending {
		c.JSON(http.StatusConflict, gin.H{
			"error": ErrPaymentDeclineStateConflict.Error(),
		})
		return
	}

	// Отклоняем платеж во внешнем адаптере
	declineResult, err := paymentAdapter.DeclinePayment(c.Request.Context(), PaymentDeclineInput{
		SessionID: txSnapshot.PaymentSessionID,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": err.Error(),
		})
		return
	}

	declineMessage := strings.TrimSpace(declineResult.Error)
	if declineMessage == "" {
		declineMessage = "payment declined"
	}

	// Обновляем локальное состояние транзакции после отклонения оплаты
	updated, err := transactionStore.Update(id, func(tx *Transaction) error {
		return tx.MarkPaymentFailed(declineMessage)
	})
	if err != nil {
		// Возвращаем понятную ошибку в зависимости от причины
		switch {
		case errors.Is(err, ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "transaction not found",
			})
		case err.Error() == ErrPaymentDeclineStateConflict.Error():
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

	// Возвращаем обновленную транзакцию клиенту
	c.JSON(http.StatusOK, updated)
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
