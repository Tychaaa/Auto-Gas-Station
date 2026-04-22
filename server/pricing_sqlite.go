package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	priceService          *PriceService
	selectionPriceLockTTL = 10 * time.Minute
)

type SQLitePriceRepository struct {
	db *sql.DB
}

type seededFuelPrice struct {
	FuelType    string
	DisplayName string
	Grade       string
	PriceMinor  int64
}

func initPricingFromEnv() error {
	dbPath := strings.TrimSpace(os.Getenv("PRICING_DB_PATH"))
	if dbPath == "" {
		dbPath = defaultPricingDBPath
	}

	lockTTLRaw := strings.TrimSpace(os.Getenv("SELECTION_PRICE_LOCK_TTL"))
	if lockTTLRaw == "" {
		lockTTLRaw = defaultPricingLockTTLEnv
	}
	lockTTL, err := time.ParseDuration(lockTTLRaw)
	if err != nil {
		return fmt.Errorf("invalid SELECTION_PRICE_LOCK_TTL: %w", err)
	}
	if lockTTL <= 0 {
		return errors.New("SELECTION_PRICE_LOCK_TTL must be > 0")
	}
	selectionPriceLockTTL = lockTTL

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create pricing directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open pricing sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	repo := &SQLitePriceRepository{db: db}
	if err := repo.initSchema(); err != nil {
		_ = db.Close()
		return err
	}
	if err := repo.seedIfEmpty(); err != nil {
		_ = db.Close()
		return err
	}
	priceService = NewPriceService(repo)
	return nil
}

func (r *SQLitePriceRepository) initSchema() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS price_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version_tag TEXT NOT NULL UNIQUE,
			effective_from DATETIME NOT NULL,
			created_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS fuel_prices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			price_version_id INTEGER NOT NULL,
			fuel_type TEXT NOT NULL,
			display_name TEXT NOT NULL,
			grade TEXT NOT NULL,
			price_per_liter_minor INTEGER NOT NULL,
			currency TEXT NOT NULL DEFAULT 'RUB',
			created_at DATETIME NOT NULL,
			UNIQUE(price_version_id, fuel_type),
			FOREIGN KEY(price_version_id) REFERENCES price_versions(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_fuel_prices_fuel_type_version
			ON fuel_prices(fuel_type, price_version_id);`,
		`CREATE INDEX IF NOT EXISTS idx_price_versions_effective_from
			ON price_versions(effective_from);`,
	}
	for _, stmt := range schema {
		if _, err := r.db.Exec(stmt); err != nil {
			return fmt.Errorf("init pricing schema: %w", err)
		}
	}
	return nil
}

func (r *SQLitePriceRepository) seedIfEmpty() error {
	var count int64
	if err := r.db.QueryRow(`SELECT COUNT(1) FROM price_versions`).Scan(&count); err != nil {
		return fmt.Errorf("count price versions: %w", err)
	}
	if count > 0 {
		return nil
	}

	now := time.Now().UTC()
	versionTag := "v1-initial"
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.Exec(
		`INSERT INTO price_versions(version_tag, effective_from, created_at) VALUES (?, ?, ?)`,
		versionTag, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert initial price version: %w", err)
	}
	versionID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("resolve version id: %w", err)
	}

	defaultPrices := []seededFuelPrice{
		{FuelType: "АИ-92", DisplayName: "АИ-92", Grade: "Регулярный", PriceMinor: 6153},
		{FuelType: "АИ-95", DisplayName: "АИ-95", Grade: "Улучшенный", PriceMinor: 6514},
		{FuelType: "АИ-100", DisplayName: "АИ-100", Grade: "Премиум", PriceMinor: 8780},
		{FuelType: "ДТ", DisplayName: "ДТ", Grade: "Дизель", PriceMinor: 7861},
	}
	for _, item := range defaultPrices {
		if _, err = tx.Exec(
			`INSERT INTO fuel_prices(
				price_version_id, fuel_type, display_name, grade, price_per_liter_minor, currency, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			versionID,
			item.FuelType,
			item.DisplayName,
			item.Grade,
			item.PriceMinor,
			defaultPricingCurrency,
			now,
		); err != nil {
			return fmt.Errorf("insert initial fuel price for %s: %w", item.FuelType, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}

func (r *SQLitePriceRepository) GetCurrentPrice(now time.Time, fuelType string) (FuelPriceSnapshot, error) {
	const q = `
SELECT
	pv.id,
	pv.version_tag,
	pv.effective_from,
	fp.fuel_type,
	fp.display_name,
	fp.grade,
	fp.price_per_liter_minor,
	fp.currency
FROM fuel_prices fp
INNER JOIN price_versions pv ON pv.id = fp.price_version_id
WHERE fp.fuel_type = ?
  AND pv.effective_from <= ?
ORDER BY pv.effective_from DESC, pv.id DESC
LIMIT 1;
`

	var snapshot FuelPriceSnapshot
	err := r.db.QueryRow(q, fuelType, now.UTC()).Scan(
		&snapshot.PriceVersionID,
		&snapshot.PriceVersionTag,
		&snapshot.EffectiveFrom,
		&snapshot.FuelType,
		&snapshot.DisplayName,
		&snapshot.Grade,
		&snapshot.PricePerLiterMinor,
		&snapshot.Currency,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return FuelPriceSnapshot{}, fmt.Errorf("price for fuel type %q not found", fuelType)
	}
	if err != nil {
		return FuelPriceSnapshot{}, fmt.Errorf("query current price: %w", err)
	}
	return snapshot, nil
}

func (r *SQLitePriceRepository) ListCurrentPrices(now time.Time) ([]FuelPriceSnapshot, error) {
	const q = `
WITH current_version AS (
	SELECT id, version_tag, effective_from
	FROM price_versions
	WHERE effective_from <= ?
	ORDER BY effective_from DESC, id DESC
	LIMIT 1
)
SELECT
	cv.id,
	cv.version_tag,
	cv.effective_from,
	fp.fuel_type,
	fp.display_name,
	fp.grade,
	fp.price_per_liter_minor,
	fp.currency
FROM fuel_prices fp
INNER JOIN current_version cv ON cv.id = fp.price_version_id
ORDER BY fp.fuel_type ASC;
`

	rows, err := r.db.Query(q, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("list current prices: %w", err)
	}
	defer rows.Close()

	var result []FuelPriceSnapshot
	for rows.Next() {
		var row FuelPriceSnapshot
		if err := rows.Scan(
			&row.PriceVersionID,
			&row.PriceVersionTag,
			&row.EffectiveFrom,
			&row.FuelType,
			&row.DisplayName,
			&row.Grade,
			&row.PricePerLiterMinor,
			&row.Currency,
		); err != nil {
			return nil, fmt.Errorf("scan current price row: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate current prices: %w", err)
	}
	return result, nil
}
