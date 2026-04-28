package http

import (
	nethttp "net/http"

	"AUTO-GAS-STATION/server/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

func NewRouter(allowedOrigins []string, transactionHandler *handlers.TransactionHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(NewCorsMiddleware(allowedOrigins))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{"status": "ok"})
	})

	RegisterTransactionRoutes(router, transactionHandler)
	return router
}

func RegisterTransactionRoutes(r *gin.Engine, h *handlers.TransactionHandler) {
	v1 := r.Group("/api/v1")
	v1.GET("/fuel-prices", h.FuelPrices)
	v1.POST("/transactions", h.CreateTransaction)
	v1.GET("/transactions/:id", h.GetTransaction)

	tx := v1.Group("/transactions/:id")
	{
		tx.PUT("/selection", h.UpdateSelection)
		tx.POST("/payment/start", h.StartPayment)
		tx.POST("/payment/status", h.PaymentStatus)
		tx.POST("/fiscalization/start", NotImplemented("fiscalization start"))
		tx.POST("/fiscalization/complete", NotImplemented("fiscalization complete"))
		tx.POST("/fiscalization/fail", NotImplemented("fiscalization fail"))
	}
}

func NotImplemented(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(nethttp.StatusNotImplemented, gin.H{"error": "not implemented", "route": name})
	}
}
