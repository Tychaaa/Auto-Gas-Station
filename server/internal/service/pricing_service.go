package service

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"AUTO-GAS-STATION/server/internal/model"
)

const (
	DefaultPricingDBPath     = "data/pricing.db"
	DefaultPricingLockTTLEnv = "10m"
	DefaultPricingCurrency   = "RUB"
)

var DefaultFuelCatalog = []model.SeededFuelPrice{
	{FuelType: "\u0410\u0418-92", DisplayName: "\u0410\u0418-92", Grade: "\u0420\u0435\u0433\u0443\u043b\u044f\u0440\u043d\u044b\u0439", PriceMinor: 6153},
	{FuelType: "\u0410\u0418-95", DisplayName: "\u0410\u0418-95", Grade: "\u0423\u043b\u0443\u0447\u0448\u0435\u043d\u043d\u044b\u0439", PriceMinor: 6514},
	{FuelType: "\u0410\u0418-100", DisplayName: "\u0410\u0418-100", Grade: "\u041f\u0440\u0435\u043c\u0438\u0443\u043c", PriceMinor: 8780},
	{FuelType: "\u0414\u0422", DisplayName: "\u0414\u0422", Grade: "\u0414\u0438\u0437\u0435\u043b\u044c", PriceMinor: 7861},
}

type PriceRepository interface {
	GetCurrentPrice(now time.Time, fuelType string) (model.FuelPriceSnapshot, error)
	ListCurrentPrices(now time.Time) ([]model.FuelPriceSnapshot, error)
	ListVersions(limit int) ([]model.PriceVersion, error)
	CreatePriceVersion(versionTag string, effectiveFrom time.Time, items []model.SeededFuelPrice) (model.PriceVersion, error)
}

type PriceService struct {
	repo PriceRepository
}

func NewPriceService(repo PriceRepository) *PriceService {
	return &PriceService{repo: repo}
}

func (s *PriceService) GetCurrentPrice(fuelType string) (model.FuelPriceSnapshot, error) {
	normalizedFuelType := strings.TrimSpace(fuelType)
	if normalizedFuelType == "" {
		return model.FuelPriceSnapshot{}, errors.New("fuel type is required")
	}
	price, err := s.repo.GetCurrentPrice(time.Now(), normalizedFuelType)
	if err != nil {
		return model.FuelPriceSnapshot{}, err
	}
	return price, nil
}

func (s *PriceService) ListVersions(limit int) ([]model.PriceVersion, error) {
	return s.repo.ListVersions(limit)
}

func (s *PriceService) CreatePriceVersion(versionTag string, effectiveFrom time.Time, pricesPerLiter map[string]float64) (model.PriceVersion, error) {
	versionTag = strings.TrimSpace(versionTag)
	if versionTag == "" {
		versionTag = fmt.Sprintf("v-%s", time.Now().UTC().Format("20060102-150405"))
	}
	if effectiveFrom.IsZero() {
		return model.PriceVersion{}, errors.New("effectiveFrom is required")
	}

	items := make([]model.SeededFuelPrice, 0, len(DefaultFuelCatalog))
	for _, catalog := range DefaultFuelCatalog {
		priceRub, ok := pricesPerLiter[catalog.FuelType]
		if !ok {
			return model.PriceVersion{}, fmt.Errorf("price for fuel type %q is required", catalog.FuelType)
		}
		if priceRub <= 0 {
			return model.PriceVersion{}, fmt.Errorf("price for fuel type %q must be > 0", catalog.FuelType)
		}
		items = append(items, model.SeededFuelPrice{
			FuelType:    catalog.FuelType,
			DisplayName: catalog.DisplayName,
			Grade:       catalog.Grade,
			PriceMinor:  int64(math.Round(priceRub * 100)),
		})
	}

	return s.repo.CreatePriceVersion(versionTag, effectiveFrom.UTC(), items)
}

func (s *PriceService) ListCurrentPrices() ([]model.FuelPriceView, error) {
	rows, err := s.repo.ListCurrentPrices(time.Now())
	if err != nil {
		return nil, err
	}
	result := make([]model.FuelPriceView, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.FuelPriceView{
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

func (s *PriceService) ApplySelectionPricing(tx *model.Transaction, lockTTL time.Duration) error {
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

func (s *PriceService) RepriceIfNeeded(tx *model.Transaction, lockTTL time.Duration, now time.Time) (bool, error) {
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
		return amountMinorFromPreset(preset, pricePerLiterMinor)
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
