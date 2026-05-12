package service

import (
	"strings"
	"sync"
	"time"
)

// KioskReasonNoPrices - причина maintenance, выставляемая автоматически при старте без версий цен
const KioskReasonNoPrices = "Цены не настроены: добавьте версию цен через админ-панель"

type KioskState struct {
	Maintenance bool      `json:"maintenance"`
	Reason      string    `json:"reason"`
	Screen      string    `json:"screen"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type KioskService struct {
	mu          sync.RWMutex
	maintenance bool
	reason      string
	screen      string
	updatedAt   time.Time

	subsMu      sync.Mutex
	subscribers map[uint64]chan KioskState
	nextSubID   uint64
}

func NewKioskService() *KioskService {
	return &KioskService{
		updatedAt:   time.Now().UTC(),
		subscribers: make(map[uint64]chan KioskState),
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
