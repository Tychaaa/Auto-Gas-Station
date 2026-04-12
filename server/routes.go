package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO(Тимофей): подключение SQLite и реальные адаптеры Vendotek / АТОЛ
// TODO(Артём): реализация отпуска топлива и сервисного контура (см. transaction.go: BeginFueling и др.).

// registerPaymentRoutes — контур транзакции / оплаты / фискализации.
func registerPaymentRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	v1.POST("/transactions", notImplemented("transactions create"))
	v1.GET("/transactions/:id", notImplemented("transactions get"))

	tx := v1.Group("/transactions/:id")
	{
		tx.PUT("/selection", notImplemented("transactions selection"))
		tx.POST("/payment/start", notImplemented("payment start"))
		tx.POST("/payment/approve", notImplemented("payment approve"))
		tx.POST("/payment/decline", notImplemented("payment decline"))
		tx.POST("/fiscalization/start", notImplemented("fiscalization start"))
		tx.POST("/fiscalization/complete", notImplemented("fiscalization complete"))
		tx.POST("/fiscalization/fail", notImplemented("fiscalization fail"))
	}
}

func registerFuelAndTerminalRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	tx := v1.Group("/transactions/:id")
	{
		tx.POST("/fueling/start", notImplemented("fueling start"))
		tx.POST("/fueling/dispensing", notImplemented("fueling dispensing"))
		tx.POST("/fueling/progress", notImplemented("fueling progress"))
		tx.POST("/fueling/complete", notImplemented("fueling complete"))
		tx.POST("/fueling/abort-paid", notImplemented("fueling abort-paid"))
		tx.POST("/fueling/fail", notImplemented("fueling fail"))
	}

	term := v1.Group("/terminal")
	{
		term.POST("/heartbeat", notImplemented("terminal heartbeat"))
		term.GET("/status", notImplemented("terminal status"))
		term.POST("/reboot-request", notImplemented("terminal reboot-request"))
		term.PUT("/config", notImplemented("terminal config"))
	}
}

func notImplemented(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "not implemented",
			"route": name,
		})
	}
}
