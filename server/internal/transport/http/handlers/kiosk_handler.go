package handlers

import (
	"encoding/json"
	"fmt"
	nethttp "net/http"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type KioskHandler struct {
	kiosk *service.KioskService
}

func NewKioskHandler(kiosk *service.KioskService) *KioskHandler {
	return &KioskHandler{kiosk: kiosk}
}

func (h *KioskHandler) State(c *gin.Context) {
	c.JSON(nethttp.StatusOK, h.kiosk.Snapshot())
}

func (h *KioskHandler) Events(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	id, ch := h.kiosk.Subscribe()
	defer h.kiosk.Unsubscribe(id)

	writeEvent(c, h.kiosk.Snapshot())

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case state, ok := <-ch:
			if !ok {
				return
			}
			writeEvent(c, state)
		}
	}
}

func (h *KioskHandler) SetMaintenance(c *gin.Context) {
	var req dto.SetMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, h.kiosk.SetMaintenance(req.Enabled, req.Reason))
}

func writeEvent(c *gin.Context, state service.KioskState) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	c.Writer.Flush()
}
