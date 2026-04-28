package main

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// kioskRuntimeState — актуальное состояние киоска для фронта
// После рестарта сервиса состояние сбрасывается в false
// Персистентность тут не нужна: админ снова включит режим одной кнопкой
type kioskRuntimeState struct {
	mu          sync.RWMutex
	maintenance bool
	reason      string
	updatedAt   time.Time
}

// kioskStateSingleton — глобальный стор, потому что это буквально состояние одного железа
var kioskStateSingleton = &kioskRuntimeState{
	updatedAt: time.Now().UTC(),
}

type kioskStatePayload struct {
	Maintenance bool      `json:"maintenance"`
	Reason      string    `json:"reason"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (s *kioskRuntimeState) snapshot() kioskStatePayload {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return kioskStatePayload{
		Maintenance: s.maintenance,
		Reason:      s.reason,
		UpdatedAt:   s.updatedAt,
	}
}

func (s *kioskRuntimeState) setMaintenance(enabled bool, reason string) kioskStatePayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maintenance = enabled
	if enabled {
		s.reason = strings.TrimSpace(reason)
	} else {
		s.reason = ""
	}
	s.updatedAt = time.Now().UTC()
	return kioskStatePayload{
		Maintenance: s.maintenance,
		Reason:      s.reason,
		UpdatedAt:   s.updatedAt,
	}
}

// getKioskStateHandler отдает текущее состояние киоска
// Публичная ручка без auth — киоск-браузер пуллит ее каждые ~3 секунды
func getKioskStateHandler(c *gin.Context) {
	c.JSON(http.StatusOK, kioskStateSingleton.snapshot())
}

type setMaintenanceRequest struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

// setMaintenanceHandler переключает режим тех работ
// Вызывается админом из /admin/ по кнопке
func setMaintenanceHandler(c *gin.Context) {
	var req setMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	snapshot := kioskStateSingleton.setMaintenance(req.Enabled, req.Reason)
	c.JSON(http.StatusOK, snapshot)
}
