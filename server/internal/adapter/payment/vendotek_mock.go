package payment

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

var ErrAdapterUnavailable = errors.New("payment adapter is unavailable")

type VendotekMockAdapter struct {
	baseURL string
	client  *http.Client
}

func NewVendotekMockAdapter(baseURL string, timeout time.Duration) *VendotekMockAdapter {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &VendotekMockAdapter{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (a *VendotekMockAdapter) StartPayment(ctx context.Context, input StartInput) (StartResult, error) {
	if a == nil || a.baseURL == "" {
		return StartResult{}, ErrAdapterUnavailable
	}
	if input.AmountMinor <= 0 {
		return StartResult{}, errors.New("amountMinor must be > 0")
	}

	createReq := vendotekCreateSessionRequest{
		ExternalTransactionID: input.ExternalTransactionID,
		AmountMinor:           input.AmountMinor,
		Currency:              input.Currency,
	}
	createResp := vendotekSessionResponse{}
	if err := a.postJSON(ctx, "/sessions", createReq, &createResp); err != nil {
		return StartResult{}, err
	}
	if createResp.SessionID == "" {
		return StartResult{}, errors.New("vendotek session id is empty")
	}

	startResp := vendotekSessionResponse{}
	startPath := fmt.Sprintf("/sessions/%s/start", createResp.SessionID)
	if err := a.postJSON(ctx, startPath, struct{}{}, &startResp); err != nil {
		return StartResult{}, err
	}

	return StartResult{SessionID: createResp.SessionID, Status: startResp.Status}, nil
}

func (a *VendotekMockAdapter) GetPaymentStatus(ctx context.Context, input StatusInput) (StatusResult, error) {
	if a == nil || a.baseURL == "" {
		return StatusResult{}, ErrAdapterUnavailable
	}
	if strings.TrimSpace(input.SessionID) == "" {
		return StatusResult{}, errors.New("payment session id is required")
	}

	statusResp := vendotekSessionResponse{}
	statusPath := fmt.Sprintf("/sessions/%s", input.SessionID)
	if err := a.getJSON(ctx, statusPath, &statusResp); err != nil {
		return StatusResult{}, err
	}

	return StatusResult{SessionID: input.SessionID, Status: statusResp.Status, Error: statusResp.Error}, nil
}

func (a *VendotekMockAdapter) postJSON(ctx context.Context, path string, reqBody any, respBody any) error {
	rawBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal vendotek request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, bytes.NewReader(rawBody))
	if err != nil {
		return fmt.Errorf("build vendotek request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return a.doJSON(req, respBody)
}

func (a *VendotekMockAdapter) getJSON(ctx context.Context, path string, respBody any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build vendotek request: %w", err)
	}
	return a.doJSON(req, respBody)
}

func (a *VendotekMockAdapter) doJSON(req *http.Request, respBody any) error {
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("call vendotek mock: %w", err)
	}
	defer resp.Body.Close()

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
