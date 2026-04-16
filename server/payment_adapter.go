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
	GetPaymentStatus(ctx context.Context, input PaymentStatusInput) (PaymentStatusResult, error)
	ApprovePayment(ctx context.Context, input PaymentApproveInput) (PaymentApproveResult, error)
	DeclinePayment(ctx context.Context, input PaymentDeclineInput) (PaymentDeclineResult, error)
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

// Данные для запроса текущего статуса платежа
type PaymentStatusInput struct {
	SessionID string
}

// Текущее состояние платежной сессии
type PaymentStatusResult struct {
	SessionID string
	Status    string
	Error     string
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

// Данные для отклонения платежа
type PaymentDeclineInput struct {
	SessionID string
}

// Результат отклонения платежа
type PaymentDeclineResult struct {
	SessionID string
	Status    string
	Error     string
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
