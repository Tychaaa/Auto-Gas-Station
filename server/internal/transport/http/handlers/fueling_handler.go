package handlers

import (
	"errors"
	nethttp "net/http"

	"AUTO-GAS-STATION/server/internal/adapter/fueling"
	"AUTO-GAS-STATION/server/internal/dto"
	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	"github.com/gin-gonic/gin"
)

type FuelingHandler struct {
	store      service.TransactionRepository
	adapter    fueling.Adapter
	dispensers *service.DispenserService
}

func NewFuelingHandler(store service.TransactionRepository, adapter fueling.Adapter, dispensers *service.DispenserService) *FuelingHandler {
	return &FuelingHandler{store: store, adapter: adapter, dispensers: dispensers}
}

func (h *FuelingHandler) Start(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}

	tx, err := h.store.Get(id)
	if errors.Is(err, repository.ErrTransactionNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req dto.FuelingStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid request body"})
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

	dispenser, err := h.dispensers.FindByFuelType(tx.FuelType)
	if errors.Is(err, repository.ErrDispenserNotFound) {
		c.JSON(nethttp.StatusUnprocessableEntity, gin.H{"error": "no dispenser assigned for fuel type " + tx.FuelType})
		return
	}
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	startResult, err := h.adapter.StartFueling(c.Request.Context(), fueling.StartInput{
		TransactionID:  tx.ID,
		AZTAddress:     dispenser.ID,
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
		if err := tx.BeginFueling(startResult.SessionID, dispenser.ID); err != nil {
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

	tx, err := h.store.Get(id)
	if errors.Is(err, repository.ErrTransactionNotFound) {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
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
	if tx.DispenserID == 0 {
		c.JSON(nethttp.StatusUnprocessableEntity, gin.H{"error": "transaction has no dispenser assigned"})
		return
	}

	statusResult, err := h.adapter.GetFuelingStatus(c.Request.Context(), fueling.StatusInput{
		SessionID:  tx.FuelingSessionID,
		AZTAddress: tx.DispenserID,
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
				if err := tx.CompleteFuelingDispense(statusResult.DispensedLiters, statusResult.Partial); err != nil {
					return err
				}
			}
			if tx.FuelingStatus == model.FuelingStatusCompletedWaitingFiscal && tx.FiscalStatus == model.FiscalStatusDone {
				return tx.CompleteAfterFueling()
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
