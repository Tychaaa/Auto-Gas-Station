package handlers

import (
	"context"
	nethttp "net/http"
	"time"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

// rebootContextTimeout ограничивает фоновую RequestReset, иначе при
// зависшем serial вызов может висеть бесконечно.
const rebootContextTimeout = 10 * time.Second

// WatchdogHandler обслуживает админские ручки контроля ESP32 watchdog.
type WatchdogHandler struct {
	service *service.WatchdogService
}

func NewWatchdogHandler(svc *service.WatchdogService) *WatchdogHandler {
	return &WatchdogHandler{service: svc}
}

// Status — GET /api/v1/admin/system/watchdog. Отдаёт кэшированный
// snapshot, не дёргая serial-порт на каждый запрос.
func (h *WatchdogHandler) Status(c *gin.Context) {
	if h.service == nil {
		c.JSON(nethttp.StatusOK, dto.WatchdogStatusView{Mode: "disabled"})
		return
	}

	snapshot := h.service.Snapshot()
	view := dto.WatchdogStatusView{
		Mode:               string(snapshot.Mode),
		Online:             snapshot.Online,
		LastHeartbeatAgoMs: snapshot.LastHeartbeatAgoMs,
		EspUptimeMs:        snapshot.EspUptimeMs,
		LastError:          snapshot.LastError,
	}
	if !snapshot.LastHeartbeatAt.IsZero() {
		view.LastHeartbeatAt = snapshot.LastHeartbeatAt.Format(time.RFC3339)
	}
	c.JSON(nethttp.StatusOK, view)
}

// Reboot — POST /api/v1/admin/system/reboot. Отвечает сразу 202 и в фоне
// инициирует перезагрузку через ESP32. На случай ошибки сам факт всё
// равно остаётся в логах сервера.
func (h *WatchdogHandler) Reboot(c *gin.Context) {
	if h.service == nil || h.service.IsDisabled() {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "watchdog is not configured"})
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), rebootContextTimeout)
		defer cancel()
		_ = h.service.RequestReset(ctx)
	}()

	c.JSON(nethttp.StatusAccepted, gin.H{"status": "reboot requested"})
}
