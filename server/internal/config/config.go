package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

const DefaultVendotekMockBaseURL = "http://localhost:8082"

type Config struct {
	GinMode             string
	Port                string
	AllowedOrigins      []string
	PricingDBPath       string
	SelectionPriceLock  time.Duration
	VendotekMockBaseURL string
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
	return Config{
		GinMode:             mode,
		Port:                envString("PORT", "8080"),
		AllowedOrigins:      resolveAllowedOrigins(mode),
		PricingDBPath:       envString("PRICING_DB_PATH", service.DefaultPricingDBPath),
		SelectionPriceLock:  lockTTL,
		VendotekMockBaseURL: envString("VENDOTEK_MOCK_BASE_URL", DefaultVendotekMockBaseURL),
	}, nil
}

func envString(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
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
