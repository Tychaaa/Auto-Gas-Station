package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"

	"AUTO-GAS-STATION/server/internal/model"
)

type seedFile struct {
	VersionTag string         `json:"versionTag"`
	Fuels      []seedFuelItem `json:"fuels"`
}

type seedFuelItem struct {
	FuelType      string  `json:"fuelType"`
	DisplayName   string  `json:"displayName"`
	Grade         string  `json:"grade"`
	PricePerLiter float64 `json:"pricePerLiter"`
}

type PricingSeeder struct {
	prices   *PriceService
	seedPath string
}

func NewPricingSeeder(prices *PriceService, seedPath string) *PricingSeeder {
	return &PricingSeeder{prices: prices, seedPath: seedPath}
}

// SeedIfEmpty читает seed-файл и заполняет начальные цены, если в базе ещё нет версий цен
// Если файл отсутствует или путь пуст, вызов является холостым
func (s *PricingSeeder) SeedIfEmpty(ctx context.Context) error {
	if s.seedPath == "" {
		slog.Info("pricing seed: skip (no seed path configured)")
		return nil
	}

	data, err := os.ReadFile(s.seedPath)
	if errors.Is(err, os.ErrNotExist) {
		slog.Info("pricing seed: skip (seed file absent)", "path", s.seedPath)
		return nil
	}
	if err != nil {
		return fmt.Errorf("read seed file %q: %w", s.seedPath, err)
	}

	var sf seedFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return fmt.Errorf("parse seed file %q: %w", s.seedPath, err)
	}
	if len(sf.Fuels) == 0 {
		return fmt.Errorf("seed file %q: fuels array is empty", s.seedPath)
	}
	for i, f := range sf.Fuels {
		if f.FuelType == "" {
			return fmt.Errorf("seed file %q: fuels[%d].fuelType is empty", s.seedPath, i)
		}
		if f.DisplayName == "" {
			return fmt.Errorf("seed file %q: fuels[%d].displayName is empty", s.seedPath, i)
		}
		if f.PricePerLiter <= 0 {
			return fmt.Errorf("seed file %q: fuels[%d].pricePerLiter must be > 0", s.seedPath, i)
		}
	}

	ok, err := s.prices.HasAnyVersion(ctx)
	if err != nil {
		return fmt.Errorf("check existing price versions: %w", err)
	}
	if ok {
		slog.Info("pricing seed: skip (prices already exist)")
		return nil
	}

	items := make([]model.SeededFuelPrice, 0, len(sf.Fuels))
	for _, f := range sf.Fuels {
		items = append(items, model.SeededFuelPrice{
			FuelType:    f.FuelType,
			DisplayName: f.DisplayName,
			Grade:       f.Grade,
			PriceMinor:  int64(math.Round(f.PricePerLiter * 100)),
		})
	}

	versionTag := sf.VersionTag
	if versionTag == "" {
		versionTag = "v1-initial"
	}

	if _, err := s.prices.SeedInitialVersion(ctx, versionTag, items); err != nil {
		return fmt.Errorf("seed initial prices: %w", err)
	}
	slog.Info("pricing seed: applied", "version", versionTag, "fuels", len(items))
	return nil
}
