package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const allowedOriginsEnvKey = "ALLOWED_ORIGINS"

// newCorsMiddleware создает middleware CORS с конфигурацией из env.
func newCorsMiddleware() gin.HandlerFunc {
	allowedOrigins := resolveAllowedOrigins()

	config := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}

	return cors.New(config)
}

// resolveAllowedOrigins возвращает список origin для CORS.
func resolveAllowedOrigins() []string {
	originsFromEnv := strings.TrimSpace(os.Getenv(allowedOriginsEnvKey))
	if originsFromEnv != "" {
		parsed := splitCSV(originsFromEnv)
		log.Printf("CORS configured from %s: %v", allowedOriginsEnvKey, parsed)
		return parsed
	}

	devOrigins := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}

	if gin.Mode() == gin.DebugMode {
		log.Printf("CORS %s is empty, using debug defaults: %v", allowedOriginsEnvKey, devOrigins)
		return devOrigins
	}

	log.Printf("CORS %s is empty outside debug mode, fallback defaults are used: %v", allowedOriginsEnvKey, devOrigins)
	return devOrigins
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		result = append(result, value)
	}

	return result
}
