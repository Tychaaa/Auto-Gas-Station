package http

import (
	nethttp "net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewCorsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	return cors.New(config)
}

type AdminAuthConfig struct {
	Username string
	Password string
}

func NewAdminAuthMiddleware(cfg AdminAuthConfig) gin.HandlerFunc {
	basicAuth := gin.BasicAuth(gin.Accounts{cfg.Username: cfg.Password})

	return func(c *gin.Context) {
		if !isLoopbackClient(c.ClientIP()) {
			c.AbortWithStatusJSON(nethttp.StatusForbidden, gin.H{
				"error": "admin endpoints are available from loopback only",
			})
			return
		}
		basicAuth(c)
	}
}

func isLoopbackClient(clientIP string) bool {
	switch clientIP {
	case "127.0.0.1", "::1":
		return true
	}
	return false
}
