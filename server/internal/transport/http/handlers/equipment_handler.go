package handlers

import (
	"context"
	"net/http"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/fiscal"
	"AUTO-GAS-STATION/server/internal/adapter/fueling"
	"AUTO-GAS-STATION/server/internal/adapter/payment"
	"AUTO-GAS-STATION/server/internal/dto"
	"github.com/gin-gonic/gin"
)

// KKTChecker — минимальный интерфейс для проверки связи с ККТ.
type KKTChecker interface {
	CheckKKT(ctx context.Context) fiscal.KKTCheckResult
}

type EquipmentHandler struct {
	fuelingAdapter   fueling.Adapter
	kktChecker       KKTChecker
	vendotekChecker  payment.VendotekChecker
}

func NewEquipmentHandler(fuelingAdapter fueling.Adapter, kktChecker KKTChecker, vendotekChecker payment.VendotekChecker) *EquipmentHandler {
	return &EquipmentHandler{
		fuelingAdapter:  fuelingAdapter,
		kktChecker:      kktChecker,
		vendotekChecker: vendotekChecker,
	}
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

func (h *EquipmentHandler) CheckKKT(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	result := h.kktChecker.CheckKKT(ctx)
	c.JSON(http.StatusOK, dto.EquipmentKKTCheckView{
		Online:        result.Online,
		Mode:          result.Mode,
		Submode:       result.Submode,
		IsShiftOpen:   result.IsShiftOpen,
		IsReceiptOpen: result.IsReceiptOpen,
		Error:         result.Error,
		CheckedAt:     time.Now().UTC(),
	})
}

func (h *EquipmentHandler) CheckVendotek(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result := h.vendotekChecker.CheckVendotek(ctx)
	c.JSON(http.StatusOK, dto.EquipmentVendotekCheckView{
		Online:       result.Online,
		Status:       result.Status,
		SerialNumber: result.SerialNumber,
		LastOpID:     result.LastOpID,
		Info:         result.Info,
		Error:        result.Error,
		CheckedAt:    time.Now().UTC(),
	})
}
