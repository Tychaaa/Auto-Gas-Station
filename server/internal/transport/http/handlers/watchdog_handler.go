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

// Reboot — POST /api/v1/admin/system/reboot.
// method=soft: reboot by OS command
// method=hard: emergency reset via ESP32 watchdog.
func (h *WatchdogHandler) Reboot(c *gin.Context) {
	if h.service == nil {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "watchdog service is not available"})
		return
	}

	var req dto.AdminSystemRebootRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	kind := service.RebootKind(req.Method)
	if kind == service.RebootKindHard && h.service.IsDisabled() {
		c.JSON(nethttp.StatusServiceUnavailable, gin.H{"error": "ESP32 watchdog is not configured"})
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), rebootContextTimeout)
		defer cancel()
		_ = h.service.RequestReboot(ctx, kind)
	}()

	c.JSON(nethttp.StatusAccepted, gin.H{"status": "reboot requested", "method": req.Method})
}
