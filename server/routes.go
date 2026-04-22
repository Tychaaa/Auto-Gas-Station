package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO(Тимофей): подключение SQLite и реальные адаптеры Vendotek / АТОЛ
// TODO(Артём): реализация отпуска топлива и сервисного контура (см. transaction.go: BeginFueling и др.).

// Роуты для создания транзакции, оплаты и фискализации
func registerPaymentRoutes(r *gin.Engine) {
	// Базовая группа API версии v1
	v1 := r.Group("/api/v1")

	// Основные операции с транзакцией
	v1.GET("/fuel-prices", fuelPricesHandler)
	v1.POST("/transactions", createTransactionHandler)
	v1.GET("/transactions/:id", getTransactionHandler)

	// Действия по конкретной транзакции
	tx := v1.Group("/transactions/:id")
	{
		tx.PUT("/selection", updateSelectionHandler)
		tx.POST("/payment/start", paymentStartHandler)
		tx.POST("/payment/status", paymentStatusHandler)
		tx.POST("/fiscalization/start", notImplemented("fiscalization start"))
		tx.POST("/fiscalization/complete", notImplemented("fiscalization complete"))
		tx.POST("/fiscalization/fail", notImplemented("fiscalization fail"))
	}
}

func registerFuelAndTerminalRoutes(r *gin.Engine) {
	// Базовая группа API версии v1
	v1 := r.Group("/api/v1")

	// Роуты процесса отпуска топлива
	tx := v1.Group("/transactions/:id")
	{
		tx.POST("/fueling/start", fuelingStartHandler)
		tx.POST("/fueling/dispensing", notImplemented("fueling dispensing"))
		tx.POST("/fueling/progress", fuelingProgressHandler)
		tx.POST("/fueling/complete", fuelingProgressHandler)
		tx.POST("/fueling/abort-paid", notImplemented("fueling abort-paid"))
		tx.POST("/fueling/fail", notImplemented("fueling fail"))
	}

	// Роуты терминала самообслуживания
	term := v1.Group("/terminal")
	{
		term.POST("/heartbeat", notImplemented("terminal heartbeat"))
		term.GET("/status", notImplemented("terminal status"))
		term.POST("/reboot-request", notImplemented("terminal reboot-request"))
		term.PUT("/config", notImplemented("terminal config"))
	}
}

func notImplemented(name string) gin.HandlerFunc {
	// Общая заглушка для роутов, которые пока не реализованы
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "not implemented",
			"route": name,
		})
	}
}
