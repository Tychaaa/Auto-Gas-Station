package handlers

import (
	"errors"
	nethttp "net/http"
	"strings"

	"AUTO-GAS-STATION/server/internal/adapter/fueling"
	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type FuelingHandler struct {
	store   *repository.TransactionStore
	adapter fueling.Adapter
}

func NewFuelingHandler(store *repository.TransactionStore, adapter fueling.Adapter) *FuelingHandler {
	return &FuelingHandler{store: store, adapter: adapter}
}

func (h *FuelingHandler) Start(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}

	tx, ok := h.store.Get(id)
	if !ok {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	var req dto.FuelingStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.PumpID) == "" || strings.TrimSpace(req.NozzleID) == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "pumpId and nozzleId are required"})
		return
	}
	if h.adapter == nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": service.ErrFuelingAdapterUnavailable.Error()})
		return
	}
	if err := tx.ValidateSelection(); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startResult, err := h.adapter.StartFueling(c.Request.Context(), fueling.StartInput{
		TransactionID:  tx.ID,
		PumpID:         req.PumpID,
		NozzleID:       req.NozzleID,
		OrderMode:      tx.OrderMode,
		AmountRub:      tx.AmountRub,
		Liters:         tx.Liters,
		UnitPriceMinor: tx.UnitPriceMinor,
		Scenario:       req.Scenario,
	})
	if err != nil {
		updated, updateErr := h.store.Update(id, func(tx *model.Transaction) error {
			if tx.Status == model.TransactionStatusPaid {
				return tx.AbortFuelingFromPaid(err.Error())
			}
			if tx.Status == model.TransactionStatusFueling {
				return tx.MarkFuelingFailed(err.Error())
			}
			tx.FuelingStatus = model.FuelingStatusFailed
			tx.FuelingError = err.Error()
			tx.Status = model.TransactionStatusFailed
			return nil
		})
		if updateErr != nil {
			c.JSON(nethttp.StatusInternalServerError, gin.H{"error": updateErr.Error()})
			return
		}

		c.JSON(nethttp.StatusBadGateway, gin.H{
			"error":            err.Error(),
			"providerStatus":   "",
			"fuelingStarted":   false,
			"fuelingSessionId": "",
			"transaction":      updated,
		})
		return
	}

	updated, updateErr := h.store.Update(id, func(tx *model.Transaction) error {
		if tx.Status != model.TransactionStatusPaid {
			return errors.New("fueling can only be started from paid")
		}
		if err := tx.BeginFueling(startResult.SessionID); err != nil {
			return err
		}
		if startResult.ProviderStatus == "dispensing" {
			return tx.MarkFuelingDispensing()
		}
		return nil
	})
	if updateErr != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": updateErr.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, gin.H{
		"fuelingStarted":   true,
		"providerStatus":   startResult.ProviderStatus,
		"fuelingSessionId": startResult.SessionID,
		"transaction":      updated,
	})
}

func (h *FuelingHandler) Progress(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}
	if h.adapter == nil {
		c.JSON(nethttp.StatusBadGateway, gin.H{"error": service.ErrFuelingAdapterUnavailable.Error()})
		return
	}

	tx, ok := h.store.Get(id)
	if !ok {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if tx.FuelingSessionID == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "fueling session id is required"})
		return
	}
	if tx.Status != model.TransactionStatusFueling {
		c.JSON(nethttp.StatusOK, tx)
		return
	}

	statusResult, err := h.adapter.GetFuelingStatus(c.Request.Context(), fueling.StatusInput{
		SessionID: tx.FuelingSessionID,
	})
	if err != nil {
		updated, updateErr := h.store.Update(id, func(tx *model.Transaction) error {
			return tx.MarkFuelingFailed(err.Error())
		})
		if updateErr != nil {
			c.JSON(nethttp.StatusInternalServerError, gin.H{"error": updateErr.Error()})
			return
		}
		c.JSON(nethttp.StatusBadGateway, gin.H{
			"error":       err.Error(),
			"transaction": updated,
		})
		return
	}

	updated, updateErr := h.store.Update(id, func(tx *model.Transaction) error {
		if statusResult.Error != "" {
			return tx.MarkFuelingFailed(statusResult.Error)
		}
		if statusResult.ProviderStatus == "dispensing" && tx.FuelingStatus == model.FuelingStatusStarting {
			if err := tx.MarkFuelingDispensing(); err != nil {
				return err
			}
		}
		if statusResult.ProviderStatus == "dispensing" && tx.FuelingStatus == model.FuelingStatusDispensing {
			return tx.UpdateDispensedLiters(statusResult.DispensedLiters)
		}
		if statusResult.Completed {
			if tx.FuelingStatus == model.FuelingStatusStarting {
				if err := tx.MarkFuelingDispensing(); err != nil {
					return err
				}
			}
			if tx.FuelingStatus == model.FuelingStatusDispensing {
				return tx.CompleteFuelingDispense(statusResult.DispensedLiters, statusResult.Partial)
			}
		}
		return nil
	})
	if updateErr != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": updateErr.Error()})
		return
	}

	c.JSON(nethttp.StatusOK, gin.H{
		"providerStatus": statusResult.ProviderStatus,
		"transaction":    updated,
	})
}
