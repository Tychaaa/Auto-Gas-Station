package http

import (
	nethttp "net/http"

	"AUTO-GAS-STATION/server/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	allowedOrigins []string,
	adminAuth AdminAuthConfig,
	transactionHandler *handlers.TransactionHandler,
	paymentHandler *handlers.PaymentHandler,
	fuelingHandler *handlers.FuelingHandler,
	adminHandler *handlers.AdminHandler,
	kioskHandler *handlers.KioskHandler,
	watchdogHandler *handlers.WatchdogHandler,
	equipmentHandler *handlers.EquipmentHandler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(NewCorsMiddleware(allowedOrigins))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{"status": "ok"})
	})

	RegisterTransactionRoutes(router, transactionHandler, paymentHandler)
	RegisterFuelingRoutes(router, fuelingHandler)
	RegisterKioskRoutes(router, kioskHandler)
	RegisterAdminRoutes(router, adminAuth, adminHandler, kioskHandler, watchdogHandler, equipmentHandler)
	return router
}

func RegisterTransactionRoutes(r *gin.Engine, transactions *handlers.TransactionHandler, payments *handlers.PaymentHandler) {
	v1 := r.Group("/api/v1")
	v1.GET("/fuel-prices", transactions.FuelPrices)
	v1.POST("/transactions", transactions.CreateTransaction)
	v1.GET("/transactions/:id", transactions.GetTransaction)

	tx := v1.Group("/transactions/:id")
	{
		tx.PUT("/selection", transactions.UpdateSelection)
		tx.POST("/inactivity-timeout", transactions.InactivityTimeout)
		tx.POST("/payment/start", payments.Start)
		tx.POST("/payment/status", payments.Status)
		// Фискализация запускается автоматически после успешной оплаты (см. PaymentService).
		// Состояние чека всегда отдается в полях FiscalStatus / ReceiptNumber транзакции.
	}
}

func RegisterFuelingRoutes(r *gin.Engine, h *handlers.FuelingHandler) {
	v1 := r.Group("/api/v1")
	tx := v1.Group("/transactions/:id")
	{
		tx.POST("/fueling/start", h.Start)
		tx.POST("/fueling/progress", h.Progress)
	}
}

func RegisterKioskRoutes(r *gin.Engine, h *handlers.KioskHandler) {
	v1 := r.Group("/api/v1")
	v1.GET("/kiosk/state", h.State)
	v1.GET("/kiosk/events", h.Events)
	v1.POST("/kiosk/screen", h.SetScreen)
}

func RegisterAdminRoutes(r *gin.Engine, auth AdminAuthConfig, admin *handlers.AdminHandler, kiosk *handlers.KioskHandler, watchdog *handlers.WatchdogHandler, equipment *handlers.EquipmentHandler) {
	v1 := r.Group("/api/v1")
	group := v1.Group("/admin", NewAdminAuthMiddleware(auth))
	{
		group.GET("/prices/versions", admin.ListPriceVersions)
		group.POST("/prices/versions", admin.CreatePriceVersion)
		group.DELETE("/prices/versions/:id", admin.DeletePriceVersion)
		group.GET("/transactions", admin.ListTransactions)
		group.GET("/transactions/:id", admin.GetTransaction)
		group.POST("/maintenance", kiosk.SetMaintenance)
		group.GET("/system/watchdog", watchdog.Status)
		group.POST("/system/reboot", watchdog.Reboot)
		group.POST("/equipment/dispenser/check", equipment.CheckDispenser)

		group.GET("/shift/status", admin.ShiftStatus)
		group.POST("/shift/open", admin.OpenShift)
		group.POST("/shift/close", admin.CloseShift)
		group.GET("/shift/reports", admin.ListShiftReports)
		group.DELETE("/shift/reports/:id", admin.DeleteShiftReport)

		group.POST("/reports/calc-status", admin.CalcStatusReport)
		group.GET("/reports/calc-status/history", admin.ListCalcReports)
		group.DELETE("/reports/calc-status/history/:id", admin.DeleteCalcReport)

		group.GET("/kkt/header-lines", admin.ListHeaderLines)
		group.PUT("/kkt/header-lines", admin.ReplaceHeaderLines)
		group.POST("/kkt/header-lines", admin.CreateHeaderLine)
		group.PUT("/kkt/header-lines/:id", admin.UpdateHeaderLine)
		group.DELETE("/kkt/header-lines/:id", admin.DeleteHeaderLine)
	}
}

