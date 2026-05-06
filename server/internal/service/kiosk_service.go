package service

import (
	"strings"
	"sync"
	"time"
)

type KioskState struct {
	Maintenance bool      `json:"maintenance"`
	Reason      string    `json:"reason"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type KioskService struct {
	mu          sync.RWMutex
	maintenance bool
	reason      string
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
