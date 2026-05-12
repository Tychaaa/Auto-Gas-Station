package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"AUTO-GAS-STATION/server/internal/model"
)

const (
	DefaultDBPath            = "data/kiosk.db"
	DefaultPricingLockTTLEnv = "10m"
	DefaultPricingCurrency   = "RUB"
)

type PriceRepository interface {
	GetCurrentPrice(now time.Time, fuelType string) (model.FuelPriceSnapshot, error)
	ListCurrentPrices(now time.Time) ([]model.FuelPriceSnapshot, error)
	ListVersions(limit int) ([]model.PriceVersion, error)
	CreatePriceVersion(versionTag string, effectiveFrom time.Time, items []model.SeededFuelPrice) (model.PriceVersion, error)
	HasAnyVersion() (bool, error)
	LatestCatalog() ([]model.SeededFuelPrice, error)
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

// HasAnyVersion проверяет, есть ли в базе хотя бы одна версия цен
func (s *PriceService) HasAnyVersion(ctx context.Context) (bool, error) {
	return s.repo.HasAnyVersion()
}

// SeedInitialVersion добавляет первую версию цен из seed-файла
// Возвращает ошибку, если версии уже существуют
func (s *PriceService) SeedInitialVersion(ctx context.Context, versionTag string, items []model.SeededFuelPrice) (model.PriceVersion, error) {
	ok, err := s.repo.HasAnyVersion()
	if err != nil {
		return model.PriceVersion{}, fmt.Errorf("check existing versions: %w", err)
	}
	if ok {
		return model.PriceVersion{}, errors.New("price versions already exist, seed skipped")
	}
	return s.repo.CreatePriceVersion(versionTag, time.Now().UTC(), items)
}

// CreatePriceVersion создаёт новую версию цен на основе каталога из последней существующей версии
// Если версий ещё нет, нужно сначала вызвать SeedInitialVersion
func (s *PriceService) CreatePriceVersion(versionTag string, effectiveFrom time.Time, pricesPerLiter map[string]float64) (model.PriceVersion, error) {
	versionTag = strings.TrimSpace(versionTag)
	if versionTag == "" {
		versionTag = fmt.Sprintf("v-%s", time.Now().UTC().Format("20060102-150405"))
	}
	if effectiveFrom.IsZero() {
		return model.PriceVersion{}, errors.New("effectiveFrom is required")
	}

	catalog, err := s.repo.LatestCatalog()
	if err != nil {
		return model.PriceVersion{}, fmt.Errorf("load fuel catalog: %w", err)
	}
	if len(catalog) == 0 {
		return model.PriceVersion{}, errors.New("fuel catalog is not initialised: seed initial prices before creating new versions")
	}

	items := make([]model.SeededFuelPrice, 0, len(catalog))
	for _, entry := range catalog {
		priceRub, ok := pricesPerLiter[entry.FuelType]
		if !ok {
			return model.PriceVersion{}, fmt.Errorf("price for fuel type %q is required", entry.FuelType)
		}
		if priceRub <= 0 {
			return model.PriceVersion{}, fmt.Errorf("price for fuel type %q must be > 0", entry.FuelType)
		}
		items = append(items, model.SeededFuelPrice{
			FuelType:    entry.FuelType,
			DisplayName: entry.DisplayName,
			Grade:       entry.Grade,
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
