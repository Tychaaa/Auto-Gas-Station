package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type fuelingStartRequest struct {
	PumpID   string `json:"pumpId"`
	NozzleID string `json:"nozzleId"`
	Scenario string `json:"scenario"`
}

// fuelingStartHandler запускает отпуск топлива через AZT adapter.
func fuelingStartHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transaction id is required",
		})
		return
	}

	tx, ok := transactionStore.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}

	var req fuelingStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Для старта должны быть известны колонка и рукав
	if strings.TrimSpace(req.PumpID) == "" || strings.TrimSpace(req.NozzleID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "pumpId and nozzleId are required",
		})
		return
	}

	if fuelingAdapter == nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "fueling adapter is not configured",
		})
		return
	}

	if err := tx.ValidateSelection(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	startResult, err := fuelingAdapter.StartFueling(c.Request.Context(), FuelingStartInput{
		TransactionID: tx.ID,
		PumpID:        req.PumpID,
		NozzleID:      req.NozzleID,
		OrderMode:     tx.OrderMode,
		AmountRub:     tx.AmountRub,
		Liters:        tx.Liters,
		Scenario:      req.Scenario,
	})
	if err != nil {
		updated, updateErr := transactionStore.Update(id, func(tx *Transaction) error {
			if tx.Status == TransactionStatusPaid {
				return tx.AbortFuelingFromPaid(err.Error())
			}
			if tx.Status == TransactionStatusFueling {
				return tx.MarkFuelingFailed(err.Error())
			}
			tx.FuelingStatus = FuelingStatusFailed
			tx.FuelingError = err.Error()
			tx.Status = TransactionStatusFailed
			return nil
		})
		if updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": updateErr.Error(),
			})
			return
		}

		c.JSON(http.StatusBadGateway, gin.H{
			"error":            err.Error(),
			"providerStatus":   "",
			"fuelingStarted":   false,
			"fuelingSessionId": "",
			"transaction":      updated,
		})
		return
	}

	updated, updateErr := transactionStore.Update(id, func(tx *Transaction) error {
		if tx.Status != TransactionStatusPaid {
			return errors.New("fueling can only be started from paid")
		}
		if err := tx.BeginFueling(startResult.SessionID); err != nil {
			return err
		}
		if startResult.ProviderStatus == "dispensing" {
			if err := tx.MarkFuelingDispensing(); err != nil {
				return err
			}
		}
		return nil
	})
	if updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": updateErr.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fuelingStarted":   true,
		"providerStatus":   startResult.ProviderStatus,
		"fuelingSessionId": startResult.SessionID,
		"transaction":      updated,
	})
}

func fuelingProgressHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "transaction id is required"})
		return
	}
	if fuelingAdapter == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "fueling adapter is not configured"})
		return
	}

	tx, ok := transactionStore.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	if tx.FuelingSessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fueling session id is required"})
		return
	}
	if tx.Status != TransactionStatusFueling {
		c.JSON(http.StatusOK, tx)
		return
	}

	statusResult, err := fuelingAdapter.GetFuelingStatus(c.Request.Context(), FuelingStatusInput{
		SessionID: tx.FuelingSessionID,
		PumpID:    "",
		NozzleID:  "",
	})
	if err != nil {
		updated, updateErr := transactionStore.Update(id, func(tx *Transaction) error {
			return tx.MarkFuelingFailed(err.Error())
		})
		if updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": updateErr.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"error":       err.Error(),
			"transaction": updated,
		})
		return
	}

	updated, updateErr := transactionStore.Update(id, func(tx *Transaction) error {
		if statusResult.Error != "" {
			return tx.MarkFuelingFailed(statusResult.Error)
		}

		if statusResult.ProviderStatus == "dispensing" && tx.FuelingStatus == FuelingStatusStarting {
			if err := tx.MarkFuelingDispensing(); err != nil {
				return err
			}
		}

		if statusResult.ProviderStatus == "dispensing" && tx.FuelingStatus == FuelingStatusDispensing {
			return tx.UpdateDispensedLiters(statusResult.DispensedLiters)
		}

		if statusResult.Completed {
			if tx.FuelingStatus == FuelingStatusStarting {
				if err := tx.MarkFuelingDispensing(); err != nil {
					return err
				}
			}
			if tx.FuelingStatus == FuelingStatusDispensing {
				return tx.CompleteFuelingDispense(statusResult.DispensedLiters, statusResult.Partial)
			}
		}

		return nil
	})
	if updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": updateErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providerStatus": statusResult.ProviderStatus,
		"transaction":    updated,
	})
}
