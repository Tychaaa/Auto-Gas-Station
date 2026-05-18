package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"AUTO-GAS-STATION/server/internal/adapter/azt"
	"AUTO-GAS-STATION/server/internal/adapter/fiscal"
	adapterfueling "AUTO-GAS-STATION/server/internal/adapter/fueling"
	"AUTO-GAS-STATION/server/internal/adapter/payment"
	"AUTO-GAS-STATION/server/internal/adapter/watchdog"
	"AUTO-GAS-STATION/server/internal/config"
	"AUTO-GAS-STATION/server/internal/database"
	"AUTO-GAS-STATION/server/internal/repository"
	"AUTO-GAS-STATION/server/internal/service"
	transporthttp "AUTO-GAS-STATION/server/internal/transport/http"
	"AUTO-GAS-STATION/server/internal/transport/http/handlers"
)

type App struct {
	config          Config
	server          *http.Server
	watchdogService *service.WatchdogService
	watchdogAdapter watchdog.Adapter
	shiftService    *service.ShiftService
}

type Config = config.Config

func New(cfg Config) (*App, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	if err := database.Migrate(context.Background(), cfg.DBPath); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	priceRepo, err := repository.NewSQLitePriceRepository(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	txRepo, err := repository.NewSQLiteTransactionRepository(cfg.DBPath)
	if err != nil {
		_ = priceRepo.Close()
		return nil, err
	}

	kktShiftRepo, err := repository.NewSQLiteKKTShiftRepository(cfg.DBPath)
	if err != nil {
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, fmt.Errorf("init kkt shift repository: %w", err)
	}

	headerLinesRepo, err := repository.NewSQLiteHeaderLinesRepository(cfg.DBPath)
	if err != nil {
		_ = kktShiftRepo.Close()
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, fmt.Errorf("init header lines repository: %w", err)
	}

	shiftReportsRepo, err := repository.NewSQLiteKKTShiftReportsRepository(cfg.DBPath)
	if err != nil {
		_ = headerLinesRepo.Close()
		_ = kktShiftRepo.Close()
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, fmt.Errorf("init kkt shift reports repository: %w", err)
	}

	calcReportsRepo, err := repository.NewSQLiteKKTCalcReportsRepository(cfg.DBPath)
	if err != nil {
		_ = shiftReportsRepo.Close()
		_ = headerLinesRepo.Close()
		_ = kktShiftRepo.Close()
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, fmt.Errorf("init kkt calc reports repository: %w", err)
	}

	priceService := service.NewPriceService(priceRepo)
	kioskService := service.NewKioskService()

	seeder := service.NewPricingSeeder(priceService, cfg.PricingSeedPath)
	if err := seeder.SeedIfEmpty(context.Background()); err != nil {
		_ = calcReportsRepo.Close()
		_ = shiftReportsRepo.Close()
		_ = headerLinesRepo.Close()
		_ = kktShiftRepo.Close()
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, fmt.Errorf("seed prices: %w", err)
	}

	hasPrices, err := priceService.HasAnyVersion(context.Background())
	if err != nil {
		_ = calcReportsRepo.Close()
		_ = shiftReportsRepo.Close()
		_ = headerLinesRepo.Close()
		_ = kktShiftRepo.Close()
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, fmt.Errorf("check prices: %w", err)
	}
	if !hasPrices {
		slog.Warn("no price versions found, setting kiosk to maintenance", "reason", service.KioskReasonNoPrices)
		kioskService.SetMaintenance(true, service.KioskReasonNoPrices)
	}

	// ShiftService создаётся до KKTAdapter — adapter будет установлен через SetAdapter.
	shiftService := service.NewShiftService(
		nil,
		kktShiftRepo,
		headerLinesRepo,
		shiftReportsRepo,
		calcReportsRepo,
		kioskService,
		slog.Default(),
		service.ShiftServiceConfig{AutoCloseAt: cfg.FiscalKKT.AutoCloseAt},
	)

	paymentAdapter := payment.NewVendotekMockAdapter(cfg.VendotekMockBaseURL, 5*time.Second)
	fuelingAdapter, err := adapterfueling.NewAZTSerialAdapter(azt.SerialConfig{
		Port:     cfg.FuelSerial.Port,
		Baud:     cfg.FuelSerial.Baud,
		DataBits: cfg.FuelSerial.DataBits,
		StopBits: cfg.FuelSerial.StopBits,
		Parity:   cfg.FuelSerial.Parity,
		Address:  cfg.FuelSerial.Address,
	})
	if err != nil {
		_ = calcReportsRepo.Close()
		_ = shiftReportsRepo.Close()
		_ = headerLinesRepo.Close()
		_ = kktShiftRepo.Close()
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, err
	}

	fiscalAdapter, err := fiscal.NewKKTAdapter(fiscal.KKTAdapterOptions{
		Config:              cfg.FiscalKKT,
		Logger:              slog.Default(),
		HeaderLinesProvider: shiftService,
		ShiftStateSink:      shiftService,
		ZReportSink:         shiftService,
	})
	if err != nil {
		_ = calcReportsRepo.Close()
		_ = shiftReportsRepo.Close()
		_ = headerLinesRepo.Close()
		_ = kktShiftRepo.Close()
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, fmt.Errorf("init fiscal adapter: %w", err)
	}

	shiftService.SetAdapter(fiscalAdapter)
	shiftService.StartAutoClose()

	transactionService := service.NewTransactionService(txRepo, priceService, cfg.SelectionPriceLock)
	if cfg.InactivitySweepEnabled {
		transactionService.StartSweeper(context.Background(), cfg.InactivityTimeout, cfg.InactivitySweepInterval)
	}
	fiscalService := service.NewFiscalService(txRepo, fiscalAdapter)
	paymentService := service.NewPaymentService(txRepo, priceService, paymentAdapter, fiscalService, cfg.SelectionPriceLock)

	watchdogAdapter, err := buildWatchdogAdapter(cfg.Watchdog)
	if err != nil {
		_ = headerLinesRepo.Close()
		_ = kktShiftRepo.Close()
		_ = txRepo.Close()
		_ = priceRepo.Close()
		return nil, err
	}
	watchdogService := service.NewWatchdogService(watchdogAdapter, kioskService, service.WatchdogConfig{
		Mode:              service.WatchdogMode(cfg.Watchdog.Mode),
		HeartbeatInterval: cfg.Watchdog.HeartbeatInterval,
	})
	watchdogService.Start()

	transactionHandler := handlers.NewTransactionHandler(transactionService, priceService)
	paymentHandler := handlers.NewPaymentHandler(paymentService)
	fuelingHandler := handlers.NewFuelingHandler(txRepo, fuelingAdapter)
	adminHandler := handlers.NewAdminHandler(priceService, txRepo, kioskService, shiftService)
	kioskHandler := handlers.NewKioskHandler(kioskService)
	watchdogHandler := handlers.NewWatchdogHandler(watchdogService)
	equipmentHandler := handlers.NewEquipmentHandler(fuelingAdapter, fiscalAdapter)

	router := transporthttp.NewRouter(
		cfg.AllowedOrigins,
		transporthttp.AdminAuthConfig{Username: cfg.AdminUsername, Password: cfg.AdminPassword},
		transactionHandler,
		paymentHandler,
		fuelingHandler,
		adminHandler,
		kioskHandler,
		watchdogHandler,
		equipmentHandler,
	)
	server := &http.Server{
		Addr:              "127.0.0.1:" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	return &App{
		config:          cfg,
		server:          server,
		watchdogService: watchdogService,
		watchdogAdapter: watchdogAdapter,
		shiftService:    shiftService,
	}, nil
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

// Shutdown корректно останавливает HTTP-сервер и фоновые сервисы.
func (a *App) Shutdown(ctx context.Context) error {
	err := a.server.Shutdown(ctx)
	if a.shiftService != nil {
		a.shiftService.Stop()
	}
	a.watchdogService.Stop()
	return err
}

// buildWatchdogAdapter создаёт реальный SerialAdapter если WATCHDOG_MODE=serial,
// иначе — заглушку Disabled. Ошибка открытия порта не валит приложение:
// логируем и проваливаемся в Disabled, чтобы можно было запускаться без
// подключённой ESP32 во время разработки и обслуживания.
func buildWatchdogAdapter(cfg config.WatchdogConfig) (watchdog.Adapter, error) {
	if cfg.Mode != "serial" {
		return watchdog.NewDisabled(), nil
	}
	adapter, err := watchdog.NewSerialAdapter(watchdog.SerialConfig{
		Port:            cfg.Port,
		Baud:            cfg.Baud,
		ExchangeTimeout: cfg.ExchangeTimeout,
	})
	if err != nil {
		log.Printf("watchdog serial adapter unavailable, falling back to disabled: %v", err)
		return watchdog.NewDisabled(), nil
	}
	return adapter, nil
}
