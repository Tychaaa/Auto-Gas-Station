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

type VendotekEzPOSAdapter struct {
	baseURL  string
	opPrefix string
	client   *http.Client
}

func NewVendotekEzPOSAdapter(baseURL string, timeout time.Duration, opPrefix string) *VendotekEzPOSAdapter {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	transport := &http.Transport{
		MaxConnsPerHost:     1,
		MaxIdleConnsPerHost: 1,
		IdleConnTimeout:     30 * time.Second,
	}
	return &VendotekEzPOSAdapter{
		baseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		opPrefix: strings.TrimSpace(opPrefix),
		client:   &http.Client{Timeout: timeout, Transport: transport},
	}
}

func (a *VendotekEzPOSAdapter) StartPayment(ctx context.Context, input StartInput) (StartResult, error) {
	if a == nil || a.baseURL == "" {
		return StartResult{}, ErrAdapterUnavailable
	}
	if input.AmountMinor <= 0 {
		return StartResult{}, errors.New("amountMinor must be > 0")
	}

	opID := a.buildOpID(input.ExternalTransactionID)

	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "RUB"
	}

	req := ezposSaleRequest{
		ID:       opID,
		Sum:      input.AmountMinor,
		Currency: currency,
	}
	var resp ezposSaleResponse
	if err := a.postJSON(ctx, "/async/cashless/sale", req, &resp); err != nil {
		return StartResult{}, err
	}

	return StartResult{SessionID: opID, Status: mapEzPOSStatus(resp.Status)}, nil
}

func (a *VendotekEzPOSAdapter) GetPaymentStatus(ctx context.Context, input StatusInput) (StatusResult, error) {
	if a == nil || a.baseURL == "" {
		return StatusResult{}, ErrAdapterUnavailable
	}
	if strings.TrimSpace(input.SessionID) == "" {
		return StatusResult{}, errors.New("payment session id is required")
	}

	url := fmt.Sprintf("%s/sale?id=%s", a.baseURL, input.SessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return StatusResult{}, fmt.Errorf("build vendotek status request: %w", err)
	}

	var resp ezposSaleResponse
	if err := a.doJSON(req, &resp); err != nil {
		return StatusResult{}, err
	}

	result := StatusResult{
		SessionID: input.SessionID,
		Status:    mapEzPOSStatus(resp.Status),
		Error:     resp.Info,
	}
	if resp.Slip != nil {
		result.Slip = &PaymentSlip{
			PAN:          resp.Slip.PAN,
			RRN:          resp.Slip.RRN,
			ApprovalCode: resp.Slip.ApprovalCode,
			Amount:       resp.Slip.Amount,
			Date:         resp.Slip.Date,
			POSEntryMode: resp.Slip.POSEntryMode,
			AppLabel:     resp.Slip.AppLabel,
		}
	}
	return result, nil
}

func (a *VendotekEzPOSAdapter) CancelPayment(ctx context.Context, sessionID string) error {
	if a == nil || a.baseURL == "" {
		return ErrAdapterUnavailable
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("session id is required for cancel")
	}

	url := fmt.Sprintf("%s/async/cashless/sale/cancel?id=%s", a.baseURL, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("build vendotek cancel request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return a.doJSON(req, nil)
}

func (a *VendotekEzPOSAdapter) CheckVendotek(ctx context.Context) VendotekCheckResult {
	if a == nil || a.baseURL == "" {
		return VendotekCheckResult{Error: "adapter not configured"}
	}

	url := a.baseURL + "/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return VendotekCheckResult{Error: fmt.Sprintf("build request: %v", err)}
	}

	var raw ezposStatusResponse
	if err := a.doJSON(req, &raw); err != nil {
		return VendotekCheckResult{Error: err.Error()}
	}

	return VendotekCheckResult{
		Online:       raw.Status == "ok" || raw.Status == "busy",
		Status:       raw.Status,
		SerialNumber: raw.SerialNumber,
		LastOpID:     raw.LastOpID,
		Info:         raw.Info,
	}
}

func (a *VendotekEzPOSAdapter) buildOpID(externalID string) string {
	stripped := strings.ReplaceAll(externalID, "-", "")
	return a.opPrefix + stripped
}

func (a *VendotekEzPOSAdapter) postJSON(ctx context.Context, path string, reqBody any, respBody any) error {
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

func (a *VendotekEzPOSAdapter) doJSON(req *http.Request, respBody any) error {
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("call vendotek: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr vendotekEzPOSErrorResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&apiErr); decodeErr == nil && apiErr.Error != "" {
			return fmt.Errorf("vendotek %s: %s", resp.Status, apiErr.Error)
		}
		return fmt.Errorf("vendotek returned %s", resp.Status)
	}
	if respBody == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return fmt.Errorf("decode vendotek response: %w", err)
	}
	return nil
}

// mapEzPOSStatus переводит статусы EzPOS в словарь payment_service.
func mapEzPOSStatus(ezposStatus string) string {
	switch strings.ToLower(strings.TrimSpace(ezposStatus)) {
	case "created":
		return "created"
	case "wait_for_card":
		return "pending"
	case "in_progress":
		return "processing"
	case "completed", "fiscalization", "fiscalized":
		return "approved"
	case "reverted":
		return "cancelled"
	case "fail":
		return "declined"
	default:
		return ezposStatus
	}
}

type ezposSaleRequest struct {
	ID       string `json:"id"`
	Sum      int64  `json:"sum"`
	Currency string `json:"currency"`
}

type ezposSaleResponse struct {
	ID     string    `json:"id"`
	Status string    `json:"status"`
	Info   string    `json:"info,omitempty"`
	Slip   *ezposSlip `json:"slip,omitempty"`
}

type ezposSlip struct {
	PAN          string `json:"pan"`
	RRN          string `json:"rrn"`
	ApprovalCode string `json:"approval_code"`
	Amount       int64  `json:"amount"`
	Date         string `json:"date"`
	POSEntryMode string `json:"pos_entry_mode"`
	AppLabel     string `json:"app_label"`
}

type ezposStatusResponse struct {
	Status       string `json:"status"`
	SerialNumber string `json:"S/N"`
	Info         string `json:"info,omitempty"`
	LastOpID     string `json:"last_op_id,omitempty"`
}

type vendotekEzPOSErrorResponse struct {
	Error string `json:"error"`
}
