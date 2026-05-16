package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	go func() {
		log.Printf("server started on %s", application.Addr())
		if err := application.Run(); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server exited")
}
