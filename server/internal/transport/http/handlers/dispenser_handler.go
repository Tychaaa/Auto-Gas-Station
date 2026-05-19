package handlers

import (
	"errors"
	nethttp "net/http"
	"strconv"

	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type DispenserHandler struct {
	service  *service.DispenserService
	maxCount int
}

func NewDispenserHandler(svc *service.DispenserService, maxCount int) *DispenserHandler {
	return &DispenserHandler{service: svc, maxCount: maxCount}
}

func (h *DispenserHandler) List(c *gin.Context) {
	dispensers, err := h.service.ListDispensers()
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	views := make([]dto.DispenserView, 0, len(dispensers))
	for _, d := range dispensers {
		views = append(views, toDispenserView(d))
	}
	c.JSON(nethttp.StatusOK, views)
}

func (h *DispenserHandler) Assign(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid dispenser id"})
		return
	}

	var req dto.UpdateDispenserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	dispenser, err := h.service.UpdateDispenser(id, req.FuelType, req.Enabled)
	if errors.Is(err, repository.ErrDispenserNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "dispenser not found"})
		return
	}
	if err != nil {
		c.JSON(nethttp.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, toDispenserView(dispenser))
}

func (h *DispenserHandler) Add(c *gin.Context) {
	dispenser, err := h.service.AddDispenser(h.maxCount)
	if err != nil {
		c.JSON(nethttp.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, toDispenserView(dispenser))
}

func (h *DispenserHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid dispenser id"})
		return
	}
	err = h.service.DeleteDispenser(id)
	if errors.Is(err, repository.ErrDispenserNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "dispenser not found"})
		return
	}
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(nethttp.StatusNoContent)
}

func toDispenserView(d *model.Dispenser) dto.DispenserView {
	return dto.DispenserView{
		ID:        d.ID,
		FuelType:  d.FuelType,
		Label:     d.Label,
		Enabled:   d.Enabled,
		UpdatedAt: d.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		// TODO(топливомер): заполнить TankVolume и TankRemaining из сервиса датчика уровня топлива
	}
}
