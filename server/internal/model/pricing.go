package model

import "time"

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

type PriceVersion struct {
	ID            int64              `json:"id"`
	VersionTag    string             `json:"versionTag"`
	EffectiveFrom time.Time          `json:"effectiveFrom"`
	CreatedAt     time.Time          `json:"createdAt"`
	Items         []PriceVersionItem `json:"items"`
}

type PriceVersionItem struct {
	FuelType      string  `json:"fuelType"`
	DisplayName   string  `json:"displayName"`
	Grade         string  `json:"grade"`
	PricePerLiter float64 `json:"pricePerLiter"`
	Currency      string  `json:"currency"`
}

type SeededFuelPrice struct {
	FuelType    string
	DisplayName string
	Grade       string
	PriceMinor  int64
}
