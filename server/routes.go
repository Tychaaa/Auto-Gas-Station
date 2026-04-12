package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO(Тимофей): маршруты создания транзакции и оплаты; подключение БД/общего хранилища.
// TODO(Артём): реализация отпуска топлива и сервисного контура (см. transaction.go: BeginFueling и др.).

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
