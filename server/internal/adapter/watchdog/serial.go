package watchdog

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

// SerialConfig описывает параметры подключения к ESP32 через UART.
// ESP32 dev-board подключается к Lenovo обычным USB-кабелем, со стороны
// ОС появляется виртуальный COM-порт, через который и идёт обмен.
type SerialConfig struct {
	Port            string
	Baud            int
	ExchangeTimeout time.Duration
}

// SerialAdapter — реальный адаптер ESP32 watchdog. Открывает COM-порт один
// раз на весь жизненный цикл и сериализует все обмены через mutex,
// потому что serial port — единственный ресурс и не допускает конкурентных
// чтений/записей.
type SerialAdapter struct {
	cfg     SerialConfig
	mu      sync.Mutex
	port    serial.Port
	timeout time.Duration
}

const defaultExchangeTimeout = 2 * time.Second

// NewSerialAdapter открывает serial-порт на указанной скорости. На стороне
// ESP32 firmware работает в режиме 8N1 на той же скорости (по умолчанию
// 115200).
func NewSerialAdapter(cfg SerialConfig) (*SerialAdapter, error) {
	if strings.TrimSpace(cfg.Port) == "" {
		return nil, fmt.Errorf("watchdog serial port is required")
	}
	if cfg.Baud <= 0 {
		cfg.Baud = 115200
	}
	if cfg.ExchangeTimeout <= 0 {
		cfg.ExchangeTimeout = defaultExchangeTimeout
	}

	mode := &serial.Mode{
		BaudRate: cfg.Baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(cfg.Port, mode)
	if err != nil {
		return nil, fmt.Errorf("open watchdog serial %s: %w", cfg.Port, err)
	}
	if err := port.SetReadTimeout(cfg.ExchangeTimeout); err != nil {
		_ = port.Close()
		return nil, fmt.Errorf("set watchdog read timeout: %w", err)
	}
	_ = port.ResetInputBuffer()
	_ = port.ResetOutputBuffer()

	return &SerialAdapter{
		cfg:     cfg,
		port:    port,
		timeout: cfg.ExchangeTimeout,
	}, nil
}

func (a *SerialAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.port == nil {
		return nil
	}
	err := a.port.Close()
	a.port = nil
	return err
}

func (a *SerialAdapter) Heartbeat(ctx context.Context) (Heartbeat, error) {
	line, err := a.exchange(ctx, "PING")
	if err != nil {
		return Heartbeat{}, err
	}
	rest, ok := stripPrefix(line, "PONG")
	if !ok {
		return Heartbeat{}, fmt.Errorf("watchdog: unexpected response %q", line)
	}
	uptime, err := parseInt64(rest)
	if err != nil {
		return Heartbeat{}, fmt.Errorf("watchdog: parse PONG uptime: %w", err)
	}
	return Heartbeat{UptimeMs: uptime}, nil
}

func (a *SerialAdapter) Status(ctx context.Context) (Status, error) {
	line, err := a.exchange(ctx, "STATUS")
	if err != nil {
		return Status{}, err
	}
	rest, ok := stripPrefix(line, "STATUS")
	if !ok {
		return Status{}, fmt.Errorf("watchdog: unexpected response %q", line)
	}
	uptime, _ := parseField(rest, "uptime=")
	ago, _ := parseField(rest, "last_heartbeat_ago=")
	return Status{UptimeMs: uptime, LastHeartbeatAgoMs: ago}, nil
}

func (a *SerialAdapter) RequestReset(ctx context.Context) error {
	line, err := a.exchange(ctx, "RESET")
	if err != nil {
		return err
	}
	if strings.TrimSpace(line) != "OK" {
		return fmt.Errorf("watchdog: unexpected RESET response %q", line)
	}
	return nil
}

// exchange отправляет одну команду и читает ответную строку до перевода
// строки. Берёт mutex чтобы не было гонки между фоновым heartbeat и
// командой RESET от админки.
func (a *SerialAdapter) exchange(ctx context.Context, cmd string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.port == nil {
		return "", fmt.Errorf("watchdog: serial port is closed")
	}

	_ = a.port.ResetInputBuffer()

	if _, err := a.port.Write([]byte(cmd + "\n")); err != nil {
		return "", fmt.Errorf("watchdog: write %s: %w", cmd, err)
	}

	deadline := time.Now().Add(a.timeout)
	buf := make([]byte, 64)
	line := make([]byte, 0, 64)

	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("watchdog: %s timeout after %s", cmd, a.timeout)
		}

		n, err := a.port.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				c := buf[i]
				if c == '\n' || c == '\r' {
					if len(line) > 0 {
						return string(line), nil
					}
					continue
				}
				line = append(line, c)
			}
		}
		if err != nil {
			return "", fmt.Errorf("watchdog: read %s: %w", cmd, err)
		}
		if n == 0 && len(line) == 0 {
			return "", fmt.Errorf("watchdog: %s timeout after %s", cmd, a.timeout)
		}
	}
}

func stripPrefix(line, prefix string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, prefix) {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	return rest, true
}

func parseInt64(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

// parseField ищет в строке "key=value" заданный ключ и возвращает целое
// значение. Если ключа нет — возвращает 0 без ошибки, чтобы STATUS можно
// было расширять без поломки клиента.
func parseField(line, key string) (int64, bool) {
	for _, token := range strings.Fields(line) {
		if !strings.HasPrefix(token, key) {
			continue
		}
		raw := strings.TrimPrefix(token, key)
		v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	return 0, false
}
