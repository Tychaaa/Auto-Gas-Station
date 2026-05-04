package watchdog

import (
	"context"
	"errors"
)

// ErrWatchdogDisabled возвращается заглушкой Disabled, чтобы вызывающий код
// мог отличить «watchdog не настроен» от реальной ошибки обмена.
var ErrWatchdogDisabled = errors.New("watchdog adapter is disabled")

// Adapter описывает минимальный набор операций по работе с ESP32-watchdog
// поверх serial. Реальная реализация — Serial, для разработки без железа —
// Disabled.
type Adapter interface {
	Heartbeat(ctx context.Context) (Heartbeat, error)
	Status(ctx context.Context) (Status, error)
	RequestReset(ctx context.Context) error
	Close() error
}

// Heartbeat — ответ ESP32 на команду PING.
type Heartbeat struct {
	UptimeMs int64
}

// Status — ответ ESP32 на команду STATUS.
type Status struct {
	UptimeMs           int64
	LastHeartbeatAgoMs int64
}
