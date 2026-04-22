package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPricingDBPath     = "data/pricing.db"
	defaultPricingLockTTLEnv = "10m"
	defaultPricingCurrency   = "RUB"
)

type FuelPriceSnapshot struct {
	PriceVersionID     int64
	PriceVersionTag    string
	FuelType           string
	DisplayName        string
	Grade              string
	PricePerLiterMinor int64
	Currency           string
	EffectiveFrom      time.Time
}

type FuelPriceView struct {
	FuelType       string  `json:"fuelType"`
	Name           string  `json:"name"`
	Grade          string  `json:"grade"`
	PricePerLiter  float64 `json:"pricePerLiter"`
	Currency       string  `json:"currency"`
	PriceVersionID int64   `json:"priceVersionId"`
	VersionTag     string  `json:"versionTag"`
	EffectiveFrom  string  `json:"effectiveFrom"`
}

type PriceRepository interface {
	GetCurrentPrice(now time.Time, fuelType string) (FuelPriceSnapshot, error)
	ListCurrentPrices(now time.Time) ([]FuelPriceSnapshot, error)
}

type PriceService struct {
	repo PriceRepository
}

func NewPriceService(repo PriceRepository) *PriceService {
	return &PriceService{repo: repo}
}

func (s *PriceService) GetCurrentPrice(fuelType string) (FuelPriceSnapshot, error) {
	normalizedFuelType := strings.TrimSpace(fuelType)
	if normalizedFuelType == "" {
		return FuelPriceSnapshot{}, errors.New("fuel type is required")
	}
	price, err := s.repo.GetCurrentPrice(time.Now(), normalizedFuelType)
	if err != nil {
		return FuelPriceSnapshot{}, err
	}
	return price, nil
}

func (s *PriceService) ListCurrentPrices() ([]FuelPriceView, error) {
	rows, err := s.repo.ListCurrentPrices(time.Now())
	if err != nil {
		return nil, err
	}
	result := make([]FuelPriceView, 0, len(rows))
	for _, row := range rows {
		result = append(result, FuelPriceView{
			FuelType:       row.FuelType,
			Name:           row.DisplayName,
			Grade:          row.Grade,
			PricePerLiter:  float64(row.PricePerLiterMinor) / 100.0,
			Currency:       row.Currency,
			PriceVersionID: row.PriceVersionID,
			VersionTag:     row.PriceVersionTag,
			EffectiveFrom:  row.EffectiveFrom.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *PriceService) ApplySelectionPricing(tx *Transaction, lockTTL time.Duration) error {
	if tx == nil {
		return errors.New("transaction is required")
	}
	price, err := s.GetCurrentPrice(tx.FuelType)
	if err != nil {
		return err
	}
	amountMinor, err := ComputeAmountMinor(tx.OrderMode, tx.AmountRub, tx.Liters, tx.Preset, price.PricePerLiterMinor)
	if err != nil {
		return err
	}

	now := time.Now()
	tx.PriceVersionID = price.PriceVersionID
	tx.PriceVersionTag = price.PriceVersionTag
	tx.UnitPriceMinor = price.PricePerLiterMinor
	tx.ComputedAmountMinor = amountMinor
	tx.Currency = price.Currency
	tx.PricingSnapshotAt = now
	tx.PriceLockedUntil = now.Add(lockTTL)
	tx.PriceWasRepriced = false
	return nil
}

func (s *PriceService) RepriceIfNeeded(tx *Transaction, lockTTL time.Duration, now time.Time) (bool, error) {
	if tx == nil {
		return false, errors.New("transaction is required")
	}
	if !tx.PriceLockedUntil.IsZero() && now.Before(tx.PriceLockedUntil) {
		return false, nil
	}
	price, err := s.GetCurrentPrice(tx.FuelType)
	if err != nil {
		return false, err
	}
	amountMinor, err := ComputeAmountMinor(tx.OrderMode, tx.AmountRub, tx.Liters, tx.Preset, price.PricePerLiterMinor)
	if err != nil {
		return false, err
	}

	tx.PriceVersionID = price.PriceVersionID
	tx.PriceVersionTag = price.PriceVersionTag
	tx.UnitPriceMinor = price.PricePerLiterMinor
	tx.ComputedAmountMinor = amountMinor
	tx.Currency = price.Currency
	tx.PricingSnapshotAt = now
	tx.PriceLockedUntil = now.Add(lockTTL)
	tx.PriceWasRepriced = true
	return true, nil
}

func ComputeAmountMinor(orderMode string, amountRub int64, liters float64, preset string, pricePerLiterMinor int64) (int64, error) {
	switch orderMode {
	case "amount":
		if amountRub <= 0 {
			return 0, errors.New("amountRub must be > 0")
		}
		return amountRub * 100, nil
	case "liters":
		if liters <= 0 {
			return 0, errors.New("liters must be > 0")
		}
		if pricePerLiterMinor <= 0 {
			return 0, errors.New("price per liter must be > 0")
		}
		return int64(math.Round(liters * float64(pricePerLiterMinor))), nil
	case "preset":
		amountMinor, err := amountMinorFromPreset(preset, pricePerLiterMinor)
		if err != nil {
			return 0, err
		}
		return amountMinor, nil
	default:
		return 0, errors.New("invalid order mode")
	}
}

func amountMinorFromPreset(preset string, pricePerLiterMinor int64) (int64, error) {
	normalizedPreset := strings.TrimSpace(preset)
	if normalizedPreset == "" {
		return 0, errors.New("preset is required")
	}
	if strings.HasPrefix(normalizedPreset, "fast_") {
		rawAmount := strings.TrimPrefix(normalizedPreset, "fast_")
		amountRub, err := strconv.ParseInt(rawAmount, 10, 64)
		if err != nil || amountRub <= 0 {
			return 0, fmt.Errorf("invalid amount preset: %s", normalizedPreset)
		}
		return amountRub * 100, nil
	}
	if strings.HasPrefix(normalizedPreset, "liters_") {
		rawLiters := strings.TrimPrefix(normalizedPreset, "liters_")
		liters, err := strconv.ParseFloat(rawLiters, 64)
		if err != nil || liters <= 0 {
			return 0, fmt.Errorf("invalid liters preset: %s", normalizedPreset)
		}
		if pricePerLiterMinor <= 0 {
			return 0, errors.New("price per liter must be > 0")
		}
		return int64(math.Round(liters * float64(pricePerLiterMinor))), nil
	}
	return 0, fmt.Errorf("unknown preset: %s", normalizedPreset)
}
