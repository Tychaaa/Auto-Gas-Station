package fueling

import (
	"context"
	"fmt"
	"math"

	"AUTO-GAS-STATION/server/internal/adapter/azt"
)

type AZTSerialAdapter struct {
	config azt.SerialConfig
}

func NewAZTSerialAdapter(cfg azt.SerialConfig) (*AZTSerialAdapter, error) {
	if cfg.Port == "" {
		return nil, fmt.Errorf("fuel port is required")
	}
	if cfg.Address < 1 || cfg.Address > 15 {
		return nil, fmt.Errorf("fuel address must be in range 1..15")
	}
	return &AZTSerialAdapter{config: cfg}, nil
}

func (a *AZTSerialAdapter) StartFueling(ctx context.Context, input StartInput) (StartResult, error) {
	client, err := a.newClient()
	if err != nil {
		return StartResult{}, err
	}
	defer client.Close()

	status, err := client.GetStatus(ctx)
	if err != nil {
		return StartResult{}, err
	}
	if status.StatusCode != '0' && status.StatusCode != '1' {
		return StartResult{}, fmt.Errorf("fuel dispenser is in unexpected state %q", status.StatusCode)
	}

	priceMinor, err := calculatePriceMinor(input)
	if err != nil {
		return StartResult{}, err
	}
	if priceMinor > 0 {
		if err := client.SetPrice(ctx, priceMinor); err != nil {
			return StartResult{}, err
		}
	}

	switch input.OrderMode {
	case "amount":
		if input.AmountRub <= 0 {
			return StartResult{}, fmt.Errorf("amountRub must be > 0")
		}
		if err := client.SetAmountDose(ctx, input.AmountRub*100); err != nil {
			return StartResult{}, err
		}
	case "liters":
		if input.Liters <= 0 {
			return StartResult{}, fmt.Errorf("liters must be > 0")
		}
		if err := client.SetLitersDose(ctx, input.Liters); err != nil {
			return StartResult{}, err
		}
	default:
		return StartResult{}, fmt.Errorf("unsupported order mode %q", input.OrderMode)
	}

	if err := client.Authorize(ctx); err != nil {
		return StartResult{}, err
	}

	authorizedStatus, err := client.GetStatus(ctx)
	if err != nil {
		return StartResult{}, err
	}

	return StartResult{
		SessionID:      input.TransactionID,
		ProviderStatus: authorizedStatus.ProviderStatus,
	}, nil
}

func (a *AZTSerialAdapter) GetFuelingStatus(ctx context.Context, input StatusInput) (StatusResult, error) {
	client, err := a.newClient()
	if err != nil {
		return StatusResult{}, err
	}
	defer client.Close()

	status, err := client.GetStatus(ctx)
	if err != nil {
		return StatusResult{}, err
	}

	result := StatusResult{
		SessionID:       input.SessionID,
		ProviderStatus:  status.ProviderStatus,
		DispensedLiters: status.DispensedLiters,
		Completed:       status.Completed,
		Partial:         status.Partial,
	}

	switch status.StatusCode {
	case '3':
		liters, err := client.ReadCurrentVolume(ctx)
		if err != nil {
			return StatusResult{}, err
		}
		result.DispensedLiters = liters
	case '4':
		totals, err := client.ReadTotals(ctx)
		if err != nil {
			return StatusResult{}, err
		}
		result.DispensedLiters = totals.DispensedLiters
		if err := client.ConfirmTotals(ctx); err != nil {
			return StatusResult{}, err
		}
	}

	return result, nil
}

func (a *AZTSerialAdapter) newClient() (*azt.MasterClient, error) {
	transport, err := azt.NewWindowsSerialTransport(a.config)
	if err != nil {
		return nil, err
	}

	client, err := azt.NewMasterClient(transport, a.config.Address)
	if err != nil {
		transport.Close()
		return nil, err
	}
	return client, nil
}

func calculatePriceMinor(input StartInput) (int64, error) {
	if input.UnitPriceMinor > 0 {
		return input.UnitPriceMinor, nil
	}
	if input.OrderMode == "amount" && input.Liters > 0 {
		price := float64(input.AmountRub*100) / input.Liters
		return int64(math.Round(price)), nil
	}
	return 0, nil
}
