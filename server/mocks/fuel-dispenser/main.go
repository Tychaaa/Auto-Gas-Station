package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type fuelingStartRequest struct {
	TransactionID string  `json:"transactionId"`
	PumpID        string  `json:"pumpId"`
	NozzleID      string  `json:"nozzleId"`
	OrderMode     string  `json:"orderMode"`
	AmountRub     int64   `json:"amountRub"`
	Liters        float64 `json:"liters"`
	Scenario      string  `json:"scenario"`
}

type fuelingStartResponse struct {
	SessionID string `json:"sessionId,omitempty"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

var mockFuelingCounter uint64

func main() {
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = gin.DebugMode
	}
	gin.SetMode(mode)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "mock-column",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// POST /api/v1/fueling/start имитирует старт отпуска топлива на стороне колонки
	r.POST("/api/v1/fueling/start", func(c *gin.Context) {
		var req fuelingStartRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, fuelingStartResponse{
				Status: "invalid_request",
				Error:  "invalid request body",
			})
			return
		}

		// Через scenario можно быстро проверить отказ и таймаут
		switch req.Scenario {
		case "timeout_start":
			time.Sleep(4 * time.Second)
			c.JSON(http.StatusGatewayTimeout, fuelingStartResponse{
				Status: "timeout",
				Error:  "mock timeout while starting fueling",
			})
			return
		case "fail_start":
			c.JSON(http.StatusBadGateway, fuelingStartResponse{
				Status: "rejected",
				Error:  "mock start rejected",
			})
			return
		}

		// Для успешного старта возвращаем session id как идентификатор отпуска
		sessionID := nextSessionID()
		c.JSON(http.StatusOK, fuelingStartResponse{
			SessionID: sessionID,
			Status:    "authorized",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("mock column started on :%s", port)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("mock column failed: %v", err)
	}
}

// nextSessionID генерирует id mock сессии отпуска топлива
func nextSessionID() string {
	n := atomic.AddUint64(&mockFuelingCounter, 1)
	return fmt.Sprintf("fuel_%d_%06d", time.Now().UnixNano(), n)
}