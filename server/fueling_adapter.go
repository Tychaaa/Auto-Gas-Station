package main

import (
	"context"
	"os"
	"strconv"
	"strings"
)

const (
	defaultFuelPort     = "COM1"
	defaultFuelBaud     = 4800
	defaultFuelDataBits = 7
	defaultFuelStopBits = 2
	defaultFuelParity   = "even"
	defaultFuelAddress  = 1
)

var fuelingAdapter FuelingAdapter

// FuelingAdapter описывает единый контракт запуска и контроля отпуска топлива
type FuelingAdapter interface {
	StartFueling(ctx context.Context, input FuelingStartInput) (FuelingStartResult, error)
	GetFuelingStatus(ctx context.Context, input FuelingStatusInput) (FuelingStatusResult, error)
}

type FuelingStartInput struct {
	TransactionID string
	PumpID        string
	NozzleID      string
	OrderMode     string
	AmountRub     int64
	Liters        float64
	Scenario      string
}

type FuelingStartResult struct {
	SessionID      string
	ProviderStatus string
	DispensedLiters float64
}

type FuelingStatusInput struct {
	SessionID string
	PumpID    string
	NozzleID  string
}

type FuelingStatusResult struct {
	SessionID       string
	ProviderStatus  string
	DispensedLiters float64
	Completed       bool
	Partial         bool
	Error           string
}

type FuelSerialConfig struct {
	Port     string
	Baud     int
	DataBits int
	StopBits int
	Parity   string
	Address  int
}

func initFuelingAdapterFromEnv() error {
	cfg := FuelSerialConfig{
		Port:     envStringOrDefault("FUEL_PORT", defaultFuelPort),
		Baud:     envIntOrDefault("FUEL_BAUD", defaultFuelBaud),
		DataBits: envIntOrDefault("FUEL_DATABITS", defaultFuelDataBits),
		StopBits: envIntOrDefault("FUEL_STOPBITS", defaultFuelStopBits),
		Parity:   envStringOrDefault("FUEL_PARITY", defaultFuelParity),
		Address:  envIntOrDefault("FUEL_ADDRESS", defaultFuelAddress),
	}

	adapter, err := NewAZTSerialFuelingAdapter(cfg)
	if err != nil {
		return err
	}
	fuelingAdapter = adapter
	return nil
}

func envStringOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envIntOrDefault(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
