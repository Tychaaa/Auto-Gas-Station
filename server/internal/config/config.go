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

const (
	defaultWatchdogMode              = "disabled"
	defaultWatchdogBaud              = 115200
	defaultWatchdogHeartbeatInterval = "5s"
	defaultWatchdogExchangeTimeout   = "2s"
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
	Watchdog            WatchdogConfig
}

type FuelSerialConfig struct {
	Port     string
	Baud     int
	DataBits int
	StopBits int
	Parity   string
	Address  int
}

// WatchdogConfig — конфигурация ESP32 watchdog. При Mode=="disabled" сервер
// не открывает COM-порт и работает с заглушкой (см. adapter/watchdog/disabled.go).
type WatchdogConfig struct {
	Mode              string
	Port              string
	Baud              int
	HeartbeatInterval time.Duration
	ExchangeTimeout   time.Duration
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

	watchdog, err := loadWatchdog()
	if err != nil {
		return Config{}, err
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
		Watchdog: watchdog,
	}, nil
}

func loadWatchdog() (WatchdogConfig, error) {
	mode := strings.ToLower(strings.TrimSpace(envString("WATCHDOG_MODE", defaultWatchdogMode)))
	switch mode {
	case "serial", "disabled":
	default:
		return WatchdogConfig{}, fmt.Errorf("WATCHDOG_MODE must be 'serial' or 'disabled', got %q", mode)
	}

	heartbeatRaw := envString("WATCHDOG_HEARTBEAT_INTERVAL", defaultWatchdogHeartbeatInterval)
	heartbeat, err := time.ParseDuration(heartbeatRaw)
	if err != nil {
		return WatchdogConfig{}, fmt.Errorf("invalid WATCHDOG_HEARTBEAT_INTERVAL: %w", err)
	}
	if heartbeat <= 0 {
		return WatchdogConfig{}, fmt.Errorf("WATCHDOG_HEARTBEAT_INTERVAL must be > 0")
	}

	timeoutRaw := envString("WATCHDOG_EXCHANGE_TIMEOUT", defaultWatchdogExchangeTimeout)
	timeout, err := time.ParseDuration(timeoutRaw)
	if err != nil {
		return WatchdogConfig{}, fmt.Errorf("invalid WATCHDOG_EXCHANGE_TIMEOUT: %w", err)
	}
	if timeout <= 0 {
		return WatchdogConfig{}, fmt.Errorf("WATCHDOG_EXCHANGE_TIMEOUT must be > 0")
	}

	port := envString("WATCHDOG_PORT", "")
	if mode == "serial" && port == "" {
		return WatchdogConfig{}, fmt.Errorf("WATCHDOG_PORT is required when WATCHDOG_MODE=serial")
	}

	return WatchdogConfig{
		Mode:              mode,
		Port:              port,
		Baud:              envInt("WATCHDOG_BAUD", defaultWatchdogBaud),
		HeartbeatInterval: heartbeat,
		ExchangeTimeout:   timeout,
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
