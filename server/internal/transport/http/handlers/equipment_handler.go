package handlers

import (
	"context"
	"net/http"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/fueling"
	"AUTO-GAS-STATION/server/internal/dto"
	"github.com/gin-gonic/gin"
)

type EquipmentHandler struct {
	fuelingAdapter fueling.Adapter
}

func NewEquipmentHandler(fuelingAdapter fueling.Adapter) *EquipmentHandler {
	return &EquipmentHandler{fuelingAdapter: fuelingAdapter}
}

func (h *EquipmentHandler) CheckDispenser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	result, err := h.fuelingAdapter.Check(ctx)
	if err != nil {
		c.JSON(http.StatusOK, dto.EquipmentDispenserCheckView{
			Online:    false,
			Error:     err.Error(),
			CheckedAt: time.Now().UTC(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.EquipmentDispenserCheckView{
		Online:          true,
		StatusCode:      result.StatusCode,
		ReasonCode:      result.ReasonCode,
		ProviderStatus:  result.ProviderStatus,
		DispensedLiters: result.DispensedLiters,
		Completed:       result.Completed,
		CheckedAt:       time.Now().UTC(),
	})
}
