package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrPaymentAdapterUnavailable = errors.New("payment adapter is unavailable")

// Адаптер для работы с моком Vendotek по HTTP
type VendotekMockAdapter struct {
	baseURL string
	client  *http.Client
}

func NewVendotekMockAdapter(baseURL string, timeout time.Duration) *VendotekMockAdapter {
	// Если таймаут не задан используем безопасное значение по умолчанию
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	// Нормализуем адрес и создаем HTTP клиент
	return &VendotekMockAdapter{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (a *VendotekMockAdapter) StartPayment(ctx context.Context, input PaymentStartInput) (PaymentStartResult, error) {
	// Проверяем что адаптер и базовый адрес доступны
	if a == nil || a.baseURL == "" {
		return PaymentStartResult{}, ErrPaymentAdapterUnavailable
	}
	// Проверяем корректность суммы в копейках
	if input.AmountMinor <= 0 {
		return PaymentStartResult{}, errors.New("amountMinor must be > 0")
	}

	// Сначала создаем платежную сессию
	createReq := vendotekCreateSessionRequest{
		ExternalTransactionID: input.ExternalTransactionID,
		AmountMinor:           input.AmountMinor,
		Currency:              input.Currency,
	}
	createResp := vendotekSessionResponse{}
	if err := a.postJSON(ctx, "/sessions", createReq, &createResp); err != nil {
		return PaymentStartResult{}, err
	}
	if createResp.SessionID == "" {
		return PaymentStartResult{}, errors.New("vendotek session id is empty")
	}

	// Затем запускаем созданную сессию
	startResp := vendotekSessionResponse{}
	startPath := fmt.Sprintf("/sessions/%s/start", createResp.SessionID)
	if err := a.postJSON(ctx, startPath, struct{}{}, &startResp); err != nil {
		return PaymentStartResult{}, err
	}

	// Возвращаем id сессии и текущий статус из ответа мока
	return PaymentStartResult{
		SessionID: createResp.SessionID,
		Status:    startResp.Status,
	}, nil
}

func (a *VendotekMockAdapter) GetPaymentStatus(ctx context.Context, input PaymentStatusInput) (PaymentStatusResult, error) {
	// Проверяем что адаптер и базовый адрес доступны
	if a == nil || a.baseURL == "" {
		return PaymentStatusResult{}, ErrPaymentAdapterUnavailable
	}
	// Проверяем корректность идентификатора сессии
	if strings.TrimSpace(input.SessionID) == "" {
		return PaymentStatusResult{}, errors.New("payment session id is required")
	}

	// Читаем текущее состояние платежной сессии
	statusResp := vendotekSessionResponse{}
	statusPath := fmt.Sprintf("/sessions/%s", input.SessionID)
	if err := a.getJSON(ctx, statusPath, &statusResp); err != nil {
		return PaymentStatusResult{}, err
	}

	return PaymentStatusResult{
		SessionID: input.SessionID,
		Status:    statusResp.Status,
		Error:     statusResp.Error,
	}, nil
}

func (a *VendotekMockAdapter) ApprovePayment(ctx context.Context, input PaymentApproveInput) (PaymentApproveResult, error) {
	// Проверяем что адаптер и базовый адрес доступны
	if a == nil || a.baseURL == "" {
		return PaymentApproveResult{}, ErrPaymentAdapterUnavailable
	}
	// Проверяем корректность идентификатора сессии
	if strings.TrimSpace(input.SessionID) == "" {
		return PaymentApproveResult{}, errors.New("payment session id is required")
	}

	// Подтверждаем платежную сессию
	approveResp := vendotekSessionResponse{}
	approvePath := fmt.Sprintf("/sessions/%s/approve", input.SessionID)
	if err := a.postJSON(ctx, approvePath, struct{}{}, &approveResp); err != nil {
		return PaymentApproveResult{}, err
	}

	return PaymentApproveResult{
		SessionID: input.SessionID,
		Status:    approveResp.Status,
	}, nil
}

func (a *VendotekMockAdapter) DeclinePayment(ctx context.Context, input PaymentDeclineInput) (PaymentDeclineResult, error) {
	// Проверяем что адаптер и базовый адрес доступны
	if a == nil || a.baseURL == "" {
		return PaymentDeclineResult{}, ErrPaymentAdapterUnavailable
	}
	// Проверяем корректность идентификатора сессии
	if strings.TrimSpace(input.SessionID) == "" {
		return PaymentDeclineResult{}, errors.New("payment session id is required")
	}

	// Отклоняем платежную сессию
	declineResp := vendotekSessionResponse{}
	declinePath := fmt.Sprintf("/sessions/%s/decline", input.SessionID)
	if err := a.postJSON(ctx, declinePath, struct{}{}, &declineResp); err != nil {
		return PaymentDeclineResult{}, err
	}

	return PaymentDeclineResult{
		SessionID: input.SessionID,
		Status:    declineResp.Status,
		Error:     declineResp.Error,
	}, nil
}

func (a *VendotekMockAdapter) postJSON(ctx context.Context, path string, reqBody any, respBody any) error {
	// Сериализуем тело запроса в JSON
	rawBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal vendotek request: %w", err)
	}

	// Формируем POST запрос к нужному пути
	url := a.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return fmt.Errorf("build vendotek request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос и закрываем тело ответа
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("call vendotek mock: %w", err)
	}
	defer resp.Body.Close()

	// Для неуспешного кода пробуем достать текст ошибки из API
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr vendotekErrorResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&apiErr); decodeErr == nil && apiErr.Error != "" {
			return fmt.Errorf("vendotek mock %s: %s", resp.Status, apiErr.Error)
		}
		return fmt.Errorf("vendotek mock returned %s", resp.Status)
	}

	// Если тело ответа не нужно завершаем без декодирования
	if respBody == nil {
		return nil
	}
	// Читаем JSON ответ в целевую структуру
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return fmt.Errorf("decode vendotek response: %w", err)
	}
	return nil
}

func (a *VendotekMockAdapter) getJSON(ctx context.Context, path string, respBody any) error {
	// Формируем GET запрос к нужному пути
	url := a.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build vendotek request: %w", err)
	}

	// Отправляем запрос и закрываем тело ответа
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("call vendotek mock: %w", err)
	}
	defer resp.Body.Close()

	// Для неуспешного кода пробуем достать текст ошибки из API
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr vendotekErrorResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&apiErr); decodeErr == nil && apiErr.Error != "" {
			return fmt.Errorf("vendotek mock %s: %s", resp.Status, apiErr.Error)
		}
		return fmt.Errorf("vendotek mock returned %s", resp.Status)
	}

	if respBody == nil {
		return nil
	}
	// Читаем JSON ответ в целевую структуру
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return fmt.Errorf("decode vendotek response: %w", err)
	}
	return nil
}

type vendotekCreateSessionRequest struct {
	ExternalTransactionID string `json:"externalTransactionId"`
	AmountMinor           int64  `json:"amountMinor"`
	Currency              string `json:"currency"`
}

type vendotekSessionResponse struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

type vendotekErrorResponse struct {
	Error string `json:"error"`
}
