package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

const DefaultVendotekMockBaseURL = "http://localhost:8082"

const (
	defaultFuelPort     = "COM1"
	defaultFuelBaud     = 4800
	defaultFuelDataBits = 7
	defaultFuelStopBits = 2
	defaultFuelParity   = "even"
	defaultFuelAddress  = 1
)

type Config struct {
	GinMode             string
	Port                string
	AllowedOrigins      []string
	PricingDBPath       string
	SelectionPriceLock  time.Duration
	VendotekMockBaseURL string
	AdminUsername       string
	AdminPassword       string
	FuelSerial          FuelSerialConfig
}

type FuelSerialConfig struct {
	Port     string
	Baud     int
	DataBits int
	StopBits int
	Parity   string
	Address  int
}

func Load() (Config, error) {
	lockTTLRaw := envString("SELECTION_PRICE_LOCK_TTL", service.DefaultPricingLockTTLEnv)
	lockTTL, err := time.ParseDuration(lockTTLRaw)
	if err != nil {
		return Config{}, fmt.Errorf("invalid SELECTION_PRICE_LOCK_TTL: %w", err)
	}
	if lockTTL <= 0 {
		return Config{}, fmt.Errorf("SELECTION_PRICE_LOCK_TTL must be > 0")
	}

	mode := envString("GIN_MODE", gin.DebugMode)
	adminUsername := envString("ADMIN_USERNAME", "")
	adminPassword := envString("ADMIN_PASSWORD", "")
	if adminUsername == "" {
		return Config{}, fmt.Errorf("ADMIN_USERNAME is required")
	}
	if adminPassword == "" {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD is required")
	}

	return Config{
		GinMode:             mode,
		Port:                envString("PORT", "8080"),
		AllowedOrigins:      resolveAllowedOrigins(mode),
		PricingDBPath:       envString("PRICING_DB_PATH", service.DefaultPricingDBPath),
		SelectionPriceLock:  lockTTL,
		VendotekMockBaseURL: envString("VENDOTEK_MOCK_BASE_URL", DefaultVendotekMockBaseURL),
		AdminUsername:       adminUsername,
		AdminPassword:       adminPassword,
		FuelSerial: FuelSerialConfig{
			Port:     envString("FUEL_PORT", defaultFuelPort),
			Baud:     envInt("FUEL_BAUD", defaultFuelBaud),
			DataBits: envInt("FUEL_DATABITS", defaultFuelDataBits),
			StopBits: envInt("FUEL_STOPBITS", defaultFuelStopBits),
			Parity:   envString("FUEL_PARITY", defaultFuelParity),
			Address:  envInt("FUEL_ADDRESS", defaultFuelAddress),
		},
	}, nil
}

func envString(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(name string, fallback int) int {
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

func resolveAllowedOrigins(mode string) []string {
	originsFromEnv := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if originsFromEnv != "" {
		return splitCSV(originsFromEnv)
	}
	return []string{"http://localhost:5173", "http://127.0.0.1:5173"}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}
