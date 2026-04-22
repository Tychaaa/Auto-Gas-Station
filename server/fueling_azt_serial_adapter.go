package main

import (
	"context"
	"fmt"
	"math"
)

type AZTSerialFuelingAdapter struct {
	config FuelSerialConfig
}

func NewAZTSerialFuelingAdapter(cfg FuelSerialConfig) (*AZTSerialFuelingAdapter, error) {
	if cfg.Port == "" {
		return nil, fmt.Errorf("fuel port is required")
	}
	if cfg.Address < 1 || cfg.Address > 15 {
		return nil, fmt.Errorf("fuel address must be in range 1..15")
	}
	return &AZTSerialFuelingAdapter{config: cfg}, nil
}

func (a *AZTSerialFuelingAdapter) StartFueling(ctx context.Context, input FuelingStartInput) (FuelingStartResult, error) {
	client, err := a.newClient()
	if err != nil {
		return FuelingStartResult{}, err
	}
	defer client.Close()

	status, err := client.GetStatus(ctx)
	if err != nil {
		return FuelingStartResult{}, err
	}
	if status.StatusCode != '0' && status.StatusCode != '1' {
		return FuelingStartResult{}, fmt.Errorf("fuel dispenser is in unexpected state %q", status.StatusCode)
	}

	priceMinor, err := calculatePriceMinor(input)
	if err != nil {
		return FuelingStartResult{}, err
	}
	if priceMinor > 0 {
		if err := client.SetPrice(ctx, priceMinor); err != nil {
			return FuelingStartResult{}, err
		}
	}

	switch input.OrderMode {
	case "amount":
		if input.AmountRub <= 0 {
			return FuelingStartResult{}, fmt.Errorf("amountRub must be > 0")
		}
		if err := client.SetAmountDose(ctx, input.AmountRub*100); err != nil {
			return FuelingStartResult{}, err
		}
	case "liters":
		if input.Liters <= 0 {
			return FuelingStartResult{}, fmt.Errorf("liters must be > 0")
		}
		if err := client.SetLitersDose(ctx, input.Liters); err != nil {
			return FuelingStartResult{}, err
		}
	default:
		return FuelingStartResult{}, fmt.Errorf("unsupported order mode %q", input.OrderMode)
	}

	if err := client.Authorize(ctx); err != nil {
		return FuelingStartResult{}, err
	}

	authorizedStatus, err := client.GetStatus(ctx)
	if err != nil {
		return FuelingStartResult{}, err
	}

	return FuelingStartResult{
		SessionID:       input.TransactionID,
		ProviderStatus:  authorizedStatus.ProviderStatus,
		DispensedLiters: 0,
	}, nil
}

func (a *AZTSerialFuelingAdapter) GetFuelingStatus(ctx context.Context, input FuelingStatusInput) (FuelingStatusResult, error) {
	client, err := a.newClient()
	if err != nil {
		return FuelingStatusResult{}, err
	}
	defer client.Close()

	status, err := client.GetStatus(ctx)
	if err != nil {
		return FuelingStatusResult{}, err
	}

	result := FuelingStatusResult{
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
			return FuelingStatusResult{}, err
		}
		result.DispensedLiters = liters
	case '4':
		totals, err := client.ReadTotals(ctx)
		if err != nil {
			return FuelingStatusResult{}, err
		}
		result.DispensedLiters = totals.DispensedLiters
		if err := client.ConfirmTotals(ctx); err != nil {
			return FuelingStatusResult{}, err
		}
	}

	return result, nil
}

func (a *AZTSerialFuelingAdapter) newClient() (*AZTMasterClient, error) {
	transport, err := NewWindowsSerialTransport(a.config)
	if err != nil {
		return nil, err
	}

	client, err := NewAZTMasterClient(transport, a.config.Address)
	if err != nil {
		transport.Close()
		return nil, err
	}
	return client, nil
}

func calculatePriceMinor(input FuelingStartInput) (int64, error) {
	if input.OrderMode == "amount" && input.Liters > 0 {
		price := float64(input.AmountRub*100) / input.Liters
		return int64(math.Round(price)), nil
	}
	return 0, nil
}
