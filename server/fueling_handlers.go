package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type fuelingStartRequest struct {
	PumpID   string `json:"pumpId"`
	NozzleID string `json:"nozzleId"`
	Scenario string `json:"scenario"`
}

type mockFuelingStartRequest struct {
	TransactionID string  `json:"transactionId"`
	PumpID        string  `json:"pumpId"`
	NozzleID      string  `json:"nozzleId"`
	OrderMode     string  `json:"orderMode"`
	AmountRub     int64   `json:"amountRub"`
	Liters        float64 `json:"liters"`
	Scenario      string  `json:"scenario"`
}

type mockFuelingStartResponse struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// fuelingStartHandler запускает отпуск топлива через mock API
func fuelingStartHandler(c *gin.Context) {
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

	var req fuelingStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Для старта должны быть известны колонка и рукав
	if strings.TrimSpace(req.PumpID) == "" || strings.TrimSpace(req.NozzleID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "pumpId and nozzleId are required",
		})
		return
	}

	// Проверяем, что у транзакции заполнены данные выбора
	if err := tx.ValidateSelection(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// TODO(Артём): после интеграции с payment контуром разрешать старт только из paid
	sessionID, providerStatus, err := startFuelingViaMock(tx, req)
	if err != nil {
		updated, updateErr := transactionStore.Update(id, func(tx *Transaction) error {
			tx.Status = TransactionStatusFailed
			tx.FuelingStatus = FuelingStatusFailed
			tx.FuelingError = err.Error()
			tx.FuelingSessionID = ""
			return nil
		})
		if updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": updateErr.Error(),
			})
			return
		}

		// TODO(Артём): записать событие ошибки старта отпуска топлива в журнал/БД
		c.JSON(http.StatusBadGateway, gin.H{
			"error":            err.Error(),
			"providerStatus":   providerStatus,
			"fuelingStarted":   false,
			"fuelingSessionId": "",
			"transaction":      updated,
		})
		return
	}

	updated, updateErr := transactionStore.Update(id, func(tx *Transaction) error {
		tx.Status = TransactionStatusFueling
		tx.FuelingStatus = FuelingStatusStarting
		tx.FuelingSessionID = sessionID
		tx.FuelingError = ""
		tx.DispensedLiters = 0
		tx.DispenseComplete = false
		tx.DispensePartial = false
		return nil
	})
	if updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": updateErr.Error(),
		})
		return
	}

	// TODO(Артём): записать событие успешного старта отпуска топлива в журнал/БД
	c.JSON(http.StatusOK, gin.H{
		"fuelingStarted":   true,
		"providerStatus":   providerStatus,
		"fuelingSessionId": sessionID,
		"transaction":      updated,
	})
}

// startFuelingViaMock вызывает внешний mock сервис отпуска топлива
func startFuelingViaMock(tx *Transaction, req fuelingStartRequest) (string, string, error) {
	baseURL := strings.TrimSpace(os.Getenv("FUEL_MOCK_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}

	payload := mockFuelingStartRequest{
		TransactionID: tx.ID,
		PumpID:        req.PumpID,
		NozzleID:      req.NozzleID,
		OrderMode:     tx.OrderMode,
		AmountRub:     tx.AmountRub,
		Liters:        tx.Liters,
		Scenario:      req.Scenario,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("marshal fueling start request: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}

	// Таймаут делаем коротким, чтобы быстро ловить недоступность mock сервиса
	resp, err := httpClient.Post(baseURL+"/api/v1/fueling/start", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("fuel mock request failed: %w", err)
	}
	defer resp.Body.Close()

	var parsed mockFuelingStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", "", fmt.Errorf("decode fuel mock response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Status == "" {
			parsed.Status = "error"
		}
		return "", parsed.Status, fmt.Errorf("fuel mock start rejected")
	}

	if parsed.SessionID == "" {
		return "", parsed.Status, fmt.Errorf("fuel mock returned empty session id")
	}

	if parsed.Status == "" {
		parsed.Status = "authorized"
	}

	return parsed.SessionID, parsed.Status, nil
}
