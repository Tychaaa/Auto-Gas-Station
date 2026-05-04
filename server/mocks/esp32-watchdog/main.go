// ESP32 watchdog mock — отвечает на тот же serial-протокол PING/RESET/STATUS,
// что и реальная прошивка, но вместо замыкания GPIO просто пишет в лог.
//
// Запуск: со стороны мока — открываем «вторую половину» виртуальной COM-пары
// (com0com на Windows, socat -d -d pty,raw,echo=0 pty,raw,echo=0 на Linux),
// а сервер слушает первую половину через WATCHDOG_PORT.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.bug.st/serial"
)

type mockConfig struct {
	Port    string
	Baud    int
	Timeout time.Duration
}

type mockState struct {
	startedAt     time.Time
	lastHeartbeat time.Time
}

func main() {
	loadEnv()

	cfg := mockConfig{
		Port:    envString("MOCK_WATCHDOG_PORT", "COM6"),
		Baud:    envInt("MOCK_WATCHDOG_BAUD", 115200),
		Timeout: envDuration("MOCK_WATCHDOG_TIMEOUT", 60*time.Second),
	}

	port, err := openSerial(cfg)
	if err != nil {
		log.Fatalf("watchdog mock: open serial failed: %v", err)
	}
	defer port.Close()

	state := &mockState{
		startedAt:     time.Now(),
		lastHeartbeat: time.Now(),
	}

	log.Printf("watchdog mock: started on %s @ %d, simulated timeout=%s", cfg.Port, cfg.Baud, cfg.Timeout)

	// Фоновая горутина, имитирующая аппаратный watchdog: если давно не было
	// heartbeat, мок «нажимает кнопку reset» и логирует это.
	go watchdogTicker(state, cfg.Timeout)

	buf := make([]byte, 256)
	line := make([]byte, 0, 64)
	for {
		n, err := port.Read(buf)
		if err != nil {
			log.Fatalf("watchdog mock: read failed: %v", err)
		}
		if n == 0 {
			continue
		}
		for i := 0; i < n; i++ {
			c := buf[i]
			if c == '\n' || c == '\r' {
				if len(line) == 0 {
					continue
				}
				handleLine(port, state, string(line))
				line = line[:0]
				continue
			}
			line = append(line, c)
		}
	}
}

func handleLine(port serial.Port, state *mockState, raw string) {
	cmd := strings.ToUpper(strings.TrimSpace(raw))
	now := time.Now()
	uptime := now.Sub(state.startedAt).Milliseconds()

	switch cmd {
	case "PING":
		state.lastHeartbeat = now
		respond(port, fmt.Sprintf("PONG %d", uptime))
	case "RESET":
		respond(port, "OK")
		log.Printf("watchdog mock: >>>>> RESET BUTTON PRESSED <<<<<")
		state.lastHeartbeat = now
	case "STATUS":
		ago := now.Sub(state.lastHeartbeat).Milliseconds()
		respond(port, fmt.Sprintf("STATUS uptime=%d last_heartbeat_ago=%d", uptime, ago))
	default:
		log.Printf("watchdog mock: unknown command %q", cmd)
		respond(port, "ERR unknown")
	}
}

func respond(port serial.Port, line string) {
	if _, err := port.Write([]byte(line + "\n")); err != nil {
		log.Printf("watchdog mock: write failed: %v", err)
	}
}

func watchdogTicker(state *mockState, timeout time.Duration) {
	ticker := time.NewTicker(timeout / 2)
	defer ticker.Stop()
	wasFiring := false
	for range ticker.C {
		ago := time.Since(state.lastHeartbeat)
		if ago < timeout {
			wasFiring = false
			continue
		}
		if wasFiring {
			continue
		}
		wasFiring = true
		log.Printf("watchdog mock: timeout (%s without heartbeat), simulated RESET", ago.Round(time.Second))
		state.lastHeartbeat = time.Now()
	}
}

func openSerial(cfg mockConfig) (serial.Port, error) {
	if strings.TrimSpace(cfg.Port) == "" {
		return nil, fmt.Errorf("MOCK_WATCHDOG_PORT is required")
	}
	mode := &serial.Mode{
		BaudRate: cfg.Baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(cfg.Port, mode)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", cfg.Port, err)
	}
	if err := port.SetReadTimeout(serial.NoTimeout); err != nil {
		_ = port.Close()
		return nil, fmt.Errorf("set read timeout: %w", err)
	}
	_ = port.ResetInputBuffer()
	_ = port.ResetOutputBuffer()
	return port, nil
}

func loadEnv() {
	candidates := []string{
		"server/mocks/esp32-watchdog/.env",
		"mocks/esp32-watchdog/.env",
		".env",
	}
	for _, path := range candidates {
		if err := godotenv.Load(path); err == nil {
			log.Printf("watchdog mock: loaded env from %s", path)
			return
		}
	}
	log.Printf("watchdog mock: .env not found, using system environment")
}

func envString(name, fallback string) string {
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

func envDuration(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
