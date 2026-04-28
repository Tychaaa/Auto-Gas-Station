package main

import (
	"log"

	"AUTO-GAS-STATION/server/internal/app"
	"AUTO-GAS-STATION/server/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config init failed: %v", err)
	}
	gin.SetMode(cfg.GinMode)

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("app init failed: %v", err)
	}

	log.Printf("server started on %s", application.Addr())
	if err := application.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
