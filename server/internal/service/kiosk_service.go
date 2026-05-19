package service

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// KioskReasonNoPrices - причина maintenance, выставляемая автоматически при старте без версий цен
const KioskReasonNoPrices = "Цены не настроены: добавьте версию цен через админ-панель"

// KioskReasonShiftClosing - причина maintenance на время Z-отчёта ККТ.
const KioskReasonShiftClosing = "Закрытие смены ККТ, подождите…"

type KioskState struct {
	Maintenance bool      `json:"maintenance"`
	Reason      string    `json:"reason"`
	Screen      string    `json:"screen"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type kioskStateFile struct {
	Maintenance bool      `json:"maintenance"`
	Reason      string    `json:"reason"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type KioskService struct {
	mu          sync.RWMutex
	maintenance bool
	reason      string
	screen      string
	updatedAt   time.Time

	stateFile string

	subsMu      sync.Mutex
	subscribers map[uint64]chan KioskState
	nextSubID   uint64
}

func NewKioskService(stateFile string) *KioskService {
	svc := &KioskService{
		stateFile:   stateFile,
		updatedAt:   time.Now().UTC(),
		subscribers: make(map[uint64]chan KioskState),
	}
	if stateFile != "" {
		data, err := os.ReadFile(stateFile)
		if err == nil {
			var s kioskStateFile
			if err := json.Unmarshal(data, &s); err == nil {
				svc.maintenance = s.Maintenance
				svc.reason = s.Reason
				svc.updatedAt = s.UpdatedAt
			} else {
				slog.Warn("kiosk state: malformed state file, starting with defaults", "err", err)
			}
		}
		// os.ErrNotExist при первом запуске — норма
	}
	return svc
}

func (s *KioskService) persistState() {
	if s.stateFile == "" {
		return
	}
	data, err := json.Marshal(kioskStateFile{
		Maintenance: s.maintenance,
		Reason:      s.reason,
		UpdatedAt:   s.updatedAt,
	})
	if err != nil {
		slog.Error("kiosk state: marshal failed", "err", err)
		return
	}
	tmp := s.stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		slog.Error("kiosk state: write tmp failed", "err", err)
		return
	}
	if err := os.Rename(tmp, s.stateFile); err != nil {
		slog.Error("kiosk state: rename failed", "err", err)
	}
}

func (s *KioskService) Snapshot() KioskState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return KioskState{
		Maintenance: s.maintenance,
		Reason:      s.reason,
		Screen:      s.screen,
		UpdatedAt:   s.updatedAt,
	}
}

func (s *KioskService) SetMaintenance(enabled bool, reason string) KioskState {
	s.mu.Lock()
	s.maintenance = enabled
	if enabled {
		s.reason = strings.TrimSpace(reason)
	} else {
		s.reason = ""
	}
	s.updatedAt = time.Now().UTC()
	s.persistState()
	state := KioskState{
		Maintenance: s.maintenance,
		Reason:      s.reason,
		Screen:      s.screen,
		UpdatedAt:   s.updatedAt,
	}
	s.mu.Unlock()

	s.broadcast(state)
	return state
}

func (s *KioskService) SetScreen(screen string) KioskState {
	screen = strings.TrimSpace(screen)
	if len(screen) > 64 {
		screen = screen[:64]
	}

	s.mu.Lock()
	s.screen = screen
	s.updatedAt = time.Now().UTC()
	state := KioskState{
		Maintenance: s.maintenance,
		Reason:      s.reason,
		Screen:      s.screen,
		UpdatedAt:   s.updatedAt,
	}
	s.mu.Unlock()

	s.broadcast(state)
	return state
}

func (s *KioskService) Subscribe() (uint64, chan KioskState) {
	ch := make(chan KioskState, 4)
	s.subsMu.Lock()
	id := s.nextSubID
	s.nextSubID++
	s.subscribers[id] = ch
	s.subsMu.Unlock()
	return id, ch
}

func (s *KioskService) Unsubscribe(id uint64) {
	s.subsMu.Lock()
	if ch, ok := s.subscribers[id]; ok {
		delete(s.subscribers, id)
		close(ch)
	}
	s.subsMu.Unlock()
}

// ClearMaintenanceIfReason снимает maintenance только если текущая причина совпадает с переданной
// Не затрагивает maintenance, включённый по другим причинам (watchdog, ручной admin)
func (s *KioskService) ClearMaintenanceIfReason(reason string) bool {
	s.mu.Lock()
	if !s.maintenance || s.reason != reason {
		s.mu.Unlock()
		return false
	}
	s.maintenance = false
	s.reason = ""
	s.updatedAt = time.Now().UTC()
	s.persistState()
	state := KioskState{
		Maintenance: s.maintenance,
		Reason:      s.reason,
		Screen:      s.screen,
		UpdatedAt:   s.updatedAt,
	}
	s.mu.Unlock()

	s.broadcast(state)
	return true
}

func (s *KioskService) broadcast(state KioskState) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- state:
		default:
		}
	}
}
