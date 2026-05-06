package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/watchdog"
)

// WatchdogMode описывает режим работы watchdog в проде:
// "serial" — реальный обмен с ESP32, "disabled" — заглушка.
type WatchdogMode string

const (
	WatchdogModeSerial   WatchdogMode = "serial"
	WatchdogModeDisabled WatchdogMode = "disabled"
)

// WatchdogSnapshot — текущее состояние watchdog для UI.
// Время в Time/нанах сериализуется в RFC3339 на уровне DTO.
type WatchdogSnapshot struct {
	Mode               WatchdogMode
	Online             bool
	LastHeartbeatAt    time.Time
	LastHeartbeatAgoMs int64
	EspUptimeMs        int64
	LastError          string
}

// WatchdogService держит фоновый heartbeat-тикер и кэшированное состояние,
// чтобы хендлер GET /api/v1/admin/system/watchdog не дёргал serial-порт
// на каждый запрос (порт — узкое место и общий ресурс).
type WatchdogService struct {
	adapter           watchdog.Adapter
	mode              WatchdogMode
	heartbeatInterval time.Duration
	rebootGrace       time.Duration
	kiosk             *KioskService

	mu              sync.RWMutex
	online          bool
	lastHeartbeatAt time.Time
	espUptimeMs     int64
	lastError       string

	stop context.CancelFunc
	done chan struct{}
}

// WatchdogConfig описывает конфигурацию watchdog-сервиса.
type WatchdogConfig struct {
	Mode              WatchdogMode
	HeartbeatInterval time.Duration
	RebootGrace       time.Duration
}

const (
	defaultHeartbeatInterval = 5 * time.Second
	defaultRebootGrace       = 1 * time.Second
)

type RebootKind string

const (
	RebootKindSoft RebootKind = "soft"
	RebootKindHard RebootKind = "hard"
)

// NewWatchdogService создаёт сервис, но не запускает фоновый цикл.
// Чтобы цикл стартовал — нужно вызвать Start.
func NewWatchdogService(adapter watchdog.Adapter, kiosk *KioskService, cfg WatchdogConfig) *WatchdogService {
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = defaultHeartbeatInterval
	}
	if cfg.RebootGrace <= 0 {
		cfg.RebootGrace = defaultRebootGrace
	}
	if cfg.Mode == "" {
		cfg.Mode = WatchdogModeDisabled
	}
	return &WatchdogService{
		adapter:           adapter,
		mode:              cfg.Mode,
		heartbeatInterval: cfg.HeartbeatInterval,
		rebootGrace:       cfg.RebootGrace,
		kiosk:             kiosk,
	}
}

// Start запускает фоновую горутину heartbeat. Для режима disabled ничего
// не делает: snapshot всегда вернёт mode=disabled, online=false.
func (s *WatchdogService) Start() {
	if s.mode != WatchdogModeSerial || s.adapter == nil {
		return
	}
	if s.stop != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	s.done = make(chan struct{})

	go s.loop(ctx)
}

// Stop корректно завершает фоновую горутину, не закрывая адаптер
// (за закрытие отвечает владелец адаптера в app.go).
func (s *WatchdogService) Stop() {
	if s.stop == nil {
		return
	}
	s.stop()
	<-s.done
	s.stop = nil
	s.done = nil
}

func (s *WatchdogService) loop(ctx context.Context) {
	defer close(s.done)

	s.tick(ctx)

	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *WatchdogService) tick(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, s.heartbeatInterval)
	defer cancel()

	hb, err := s.adapter.Heartbeat(ctx)
	if err != nil {
		s.recordFailure(err)
		return
	}
	s.recordSuccess(hb)
}

func (s *WatchdogService) recordSuccess(hb watchdog.Heartbeat) {
	s.mu.Lock()
	s.online = true
	s.lastHeartbeatAt = time.Now().UTC()
	s.espUptimeMs = hb.UptimeMs
	s.lastError = ""
	s.mu.Unlock()
}

func (s *WatchdogService) recordFailure(err error) {
	s.mu.Lock()
	s.online = false
	s.lastError = err.Error()
	s.mu.Unlock()
	log.Printf("watchdog heartbeat failed: %v", err)
}

// Snapshot отдаёт неблокирующий снимок состояния для GET-эндпоинта.
func (s *WatchdogService) Snapshot() WatchdogSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := WatchdogSnapshot{
		Mode:            s.mode,
		Online:          s.online,
		LastHeartbeatAt: s.lastHeartbeatAt,
		EspUptimeMs:     s.espUptimeMs,
		LastError:       s.lastError,
	}
	if !s.lastHeartbeatAt.IsZero() {
		snapshot.LastHeartbeatAgoMs = time.Since(s.lastHeartbeatAt).Milliseconds()
	}
	return snapshot
}

// RequestReboot handles both reboot modes:
// soft — regular OS reboot command; hard — emergency ESP32 reset.
func (s *WatchdogService) RequestReboot(ctx context.Context, kind RebootKind) error {
	if kind != RebootKindSoft && kind != RebootKindHard {
		return errors.New("invalid reboot kind")
	}

	reason := "перезагрузка терминала (ОС)"
	if kind == RebootKindHard {
		reason = "аварийная перезагрузка терминала (ESP32)"
	}
	if s.kiosk != nil {
		s.kiosk.SetMaintenance(true, reason)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.rebootGrace):
	}

	if kind == RebootKindSoft {
		if err := s.runSoftReboot(ctx); err != nil {
			log.Printf("soft reboot failed: %v", err)
			return err
		}
		log.Printf("soft reboot scheduled")
		return nil
	}

	if s.adapter == nil || s.mode != WatchdogModeSerial {
		return watchdog.ErrWatchdogDisabled
	}
	if err := s.adapter.RequestReset(ctx); err != nil {
		log.Printf("watchdog hard reset failed: %v", err)
		return err
	}
	log.Printf("hard reboot requested via ESP32")
	return nil
}

func (s *WatchdogService) RequestReset(ctx context.Context) error {
	return s.RequestReboot(ctx, RebootKindHard)
}

func (s *WatchdogService) runSoftReboot(ctx context.Context) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// /t 10 — даёт UI 10 секунд показать оверлей перед перезагрузкой.
		// /f — принудительно закрывает приложения без предупреждения.
		cmd = exec.CommandContext(ctx, "shutdown.exe", "/r", "/t", "10", "/f")
	default:
		cmd = exec.CommandContext(ctx, "shutdown", "-r", "now")
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("shutdown failed: %w; output: %q", err, string(out))
	}
	return nil
}

// IsDisabled проверяет, нужно ли отдавать клиенту 503/disabled-ответ.
func (s *WatchdogService) IsDisabled() bool {
	return s.mode != WatchdogModeSerial || s.adapter == nil
}

// IsErrDisabled удобно вызывать из хендлера, чтобы не тащить пакет
// watchdog в transport-слой.
func IsErrDisabled(err error) bool { return errors.Is(err, watchdog.ErrWatchdogDisabled) }
