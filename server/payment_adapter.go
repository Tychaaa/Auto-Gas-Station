package main

import (
	"context"
	"os"
	"strings"
	"time"
)

const defaultVendotekMockBaseURL = "http://localhost:8082"

var paymentAdapter PaymentAdapter

// Единый интерфейс для разных платежных интеграций
type PaymentAdapter interface {
	StartPayment(ctx context.Context, input PaymentStartInput) (PaymentStartResult, error)
	ApprovePayment(ctx context.Context, input PaymentApproveInput) (PaymentApproveResult, error)
}

// Данные для старта платежа
type PaymentStartInput struct {
	ExternalTransactionID string
	AmountMinor           int64
	Currency              string
}

// Результат запуска платежа
type PaymentStartResult struct {
	SessionID string
	Status    string
}

// Данные для подтверждения платежа
type PaymentApproveInput struct {
	SessionID string
}

// Результат подтверждения платежа
type PaymentApproveResult struct {
	SessionID string
	Status    string
}

func initPaymentAdapterFromEnv() {
	// Читаем адрес мока из переменной окружения
	baseURL := strings.TrimSpace(os.Getenv("VENDOTEK_MOCK_BASE_URL"))
	// Если адрес не задан используем значение по умолчанию
	if baseURL == "" {
		baseURL = defaultVendotekMockBaseURL
	}

	// Создаем адаптер с фиксированным таймаутом запросов
	paymentAdapter = NewVendotekMockAdapter(baseURL, 5*time.Second)
}
