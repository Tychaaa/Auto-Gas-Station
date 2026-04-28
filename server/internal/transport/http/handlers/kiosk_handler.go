package handlers

import (
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

func (h *KioskHandler) SetMaintenance(c *gin.Context) {
	var req dto.SetMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, h.kiosk.SetMaintenance(req.Enabled, req.Reason))
}
