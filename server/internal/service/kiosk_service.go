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
}

func NewKioskService() *KioskService {
	return &KioskService{
		updatedAt: time.Now().UTC(),
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
	defer s.mu.Unlock()

	s.maintenance = enabled
	if enabled {
		s.reason = strings.TrimSpace(reason)
	} else {
		s.reason = ""
	}
	s.updatedAt = time.Now().UTC()

	return KioskState{
		Maintenance: s.maintenance,
		Reason:      s.reason,
		UpdatedAt:   s.updatedAt,
	}
}
