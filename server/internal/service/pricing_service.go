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
	DeletePriceVersion(id int64) error
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

func (s *PriceService) DeleteVersion(id int64) error {
	return s.repo.DeletePriceVersion(id)
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
		versionTag = time.Now().UTC().Format(time.RFC3339)
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
	computedLiters, amountMinor, err := ComputeOrderTotals(tx.OrderMode, tx.AmountRub, tx.Liters, tx.Preset, price.PricePerLiterMinor)
	if err != nil {
		return err
	}

	now := time.Now()
	tx.PriceVersionID = price.PriceVersionID
	tx.PriceVersionTag = price.PriceVersionTag
	tx.UnitPriceMinor = price.PricePerLiterMinor
	tx.Liters = computedLiters
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
	computedLiters, amountMinor, err := ComputeOrderTotals(tx.OrderMode, tx.AmountRub, tx.Liters, tx.Preset, price.PricePerLiterMinor)
	if err != nil {
		return false, err
	}

	tx.PriceVersionID = price.PriceVersionID
	tx.PriceVersionTag = price.PriceVersionTag
	tx.UnitPriceMinor = price.PricePerLiterMinor
	tx.Liters = computedLiters
	tx.ComputedAmountMinor = amountMinor
	tx.Currency = price.Currency
	tx.PricingSnapshotAt = now
	tx.PriceLockedUntil = now.Add(lockTTL)
	tx.PriceWasRepriced = true
	return true, nil
}

// ComputeOrderTotals возвращает литры (с сеткой 0.01 л) и сумму в копейках.
//
// Правило безопасного округления:
//   - amount / fast_<N>: литры ceil (клиент получает не меньше оплаченного)
//   - liters / liters_<N>: рубли floor (клиент платит не больше, чем стоит объём)
func ComputeOrderTotals(orderMode string, amountRub int64, liters float64, preset string, pricePerLiterMinor int64) (computedLiters float64, amountMinor int64, err error) {
	switch orderMode {
	case "amount":
		if amountRub <= 0 {
			return 0, 0, errors.New("amountRub must be > 0")
		}
		if pricePerLiterMinor <= 0 {
			return 0, 0, errors.New("price per liter must be > 0")
		}
		amountMinor = amountRub * 100
		litersCenti := ceilDiv(amountMinor*100, pricePerLiterMinor)
		computedLiters = float64(litersCenti) / 100
		return computedLiters, amountMinor, nil
	case "liters":
		if liters <= 0 {
			return 0, 0, errors.New("liters must be > 0")
		}
		if pricePerLiterMinor <= 0 {
			return 0, 0, errors.New("price per liter must be > 0")
		}
		return totalsFromLiters(liters, pricePerLiterMinor)
	case "preset":
		return totalsFromPreset(preset, pricePerLiterMinor)
	default:
		return 0, 0, errors.New("invalid order mode")
	}
}

// ceilDiv делит a на b с округлением вверх (a, b > 0).
func ceilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}

// totalsFromLiters вычисляет (литры, копейки) для режима liters / liters_<N>.
// Сумма округляется вниз, чтобы клиент не платил за объём, которого не получит.
func totalsFromLiters(liters float64, pricePerLiterMinor int64) (computedLiters float64, amountMinor int64, err error) {
	litersCenti := int64(math.Round(liters * 100))
	if litersCenti <= 0 {
		return 0, 0, errors.New("liters rounds to zero")
	}
	amountMinor = (litersCenti * pricePerLiterMinor) / 100 // целочисленное деление = floor
	if amountMinor <= 0 {
		return 0, 0, errors.New("computed amount is not positive")
	}
	return float64(litersCenti) / 100, amountMinor, nil
}

func totalsFromPreset(preset string, pricePerLiterMinor int64) (computedLiters float64, amountMinor int64, err error) {
	normalizedPreset := strings.TrimSpace(preset)
	if normalizedPreset == "" {
		return 0, 0, errors.New("preset is required")
	}
	if rawAmount, ok := strings.CutPrefix(normalizedPreset, "fast_"); ok {
		amountRub, parseErr := strconv.ParseInt(rawAmount, 10, 64)
		if parseErr != nil || amountRub <= 0 {
			return 0, 0, fmt.Errorf("invalid amount preset: %s", normalizedPreset)
		}
		if pricePerLiterMinor <= 0 {
			return 0, 0, errors.New("price per liter must be > 0")
		}
		minor := amountRub * 100
		litersCenti := ceilDiv(minor*100, pricePerLiterMinor)
		return float64(litersCenti) / 100, minor, nil
	}
	if rawLiters, ok := strings.CutPrefix(normalizedPreset, "liters_"); ok {
		l, parseErr := strconv.ParseFloat(rawLiters, 64)
		if parseErr != nil || l <= 0 {
			return 0, 0, fmt.Errorf("invalid liters preset: %s", normalizedPreset)
		}
		if pricePerLiterMinor <= 0 {
			return 0, 0, errors.New("price per liter must be > 0")
		}
		return totalsFromLiters(l, pricePerLiterMinor)
	}
	return 0, 0, fmt.Errorf("unknown preset: %s", normalizedPreset)
}
