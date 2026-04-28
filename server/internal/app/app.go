package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/payment"
	"AUTO-GAS-STATION/server/internal/config"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	transporthttp "AUTO-GAS-STATION/server/internal/transport/http"
	"AUTO-GAS-STATION/server/internal/transport/http/handlers"
)

type App struct {
	config Config
	server *http.Server
}

type Config = config.Config

func New(cfg Config) (*App, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.PricingDBPath), 0o755); err != nil {
		return nil, fmt.Errorf("create pricing directory: %w", err)
	}

	priceRepo, err := repository.NewSQLitePriceRepository(cfg.PricingDBPath)
	if err != nil {
		return nil, err
	}
	if err := priceRepo.InitSchema(); err != nil {
		_ = priceRepo.Close()
		return nil, err
	}
	if err := priceRepo.SeedIfEmpty(service.DefaultFuelCatalog); err != nil {
		_ = priceRepo.Close()
		return nil, err
	}

	priceService := service.NewPriceService(priceRepo)
	transactionStore := repository.NewTransactionStore()
	paymentAdapter := payment.NewVendotekMockAdapter(cfg.VendotekMockBaseURL, 5*time.Second)
	transactionHandler := handlers.NewTransactionHandler(transactionStore, priceService, paymentAdapter, cfg.SelectionPriceLock)

	router := transporthttp.NewRouter(cfg.AllowedOrigins, transactionHandler)
	server := &http.Server{
		Addr:              "127.0.0.1:" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	return &App{config: cfg, server: server}, nil
}

func (a *App) Run() error {
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *App) Addr() string {
	return a.server.Addr
}
