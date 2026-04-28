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
	RegisterAdminRoutes(router, adminAuth, adminHandler, kioskHandler)
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
		tx.POST("/payment/start", payments.Start)
		tx.POST("/payment/status", payments.Status)
		tx.POST("/fiscalization/start", NotImplemented("fiscalization start"))
		tx.POST("/fiscalization/complete", NotImplemented("fiscalization complete"))
		tx.POST("/fiscalization/fail", NotImplemented("fiscalization fail"))
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
}

func RegisterAdminRoutes(r *gin.Engine, auth AdminAuthConfig, admin *handlers.AdminHandler, kiosk *handlers.KioskHandler) {
	v1 := r.Group("/api/v1")
	group := v1.Group("/admin", NewAdminAuthMiddleware(auth))
	{
		group.GET("/prices/versions", admin.ListPriceVersions)
		group.POST("/prices/versions", admin.CreatePriceVersion)
		group.GET("/transactions", admin.ListTransactions)
		group.POST("/maintenance", kiosk.SetMaintenance)
	}
}

func NotImplemented(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(nethttp.StatusNotImplemented, gin.H{"error": "not implemented", "route": name})
	}
}
