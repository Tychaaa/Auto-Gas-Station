package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"AUTO-GAS-STATION/server/internal/model"
	"AUTO-GAS-STATION/server/internal/service"
	_ "modernc.org/sqlite"
)

type SQLitePriceRepository struct {
	db *sql.DB
}

func NewSQLitePriceRepository(dbPath string) (*SQLitePriceRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open pricing sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &SQLitePriceRepository{db: db}, nil
}

func (r *SQLitePriceRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *SQLitePriceRepository) InitSchema() error {
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

func (r *SQLitePriceRepository) SeedIfEmpty(items []model.SeededFuelPrice) error {
	var count int64
	if err := r.db.QueryRow(`SELECT COUNT(1) FROM price_versions`).Scan(&count); err != nil {
		return fmt.Errorf("count price versions: %w", err)
	}
	if count > 0 {
		return nil
	}
	_, err := r.CreatePriceVersion("v1-initial", time.Now().UTC(), items)
	if err != nil {
		return fmt.Errorf("seed initial prices: %w", err)
	}
	return nil
}

func (r *SQLitePriceRepository) CreatePriceVersion(versionTag string, effectiveFrom time.Time, items []model.SeededFuelPrice) (model.PriceVersion, error) {
	if versionTag == "" {
		return model.PriceVersion{}, errors.New("version tag is required")
	}
	if len(items) == 0 {
		return model.PriceVersion{}, errors.New("at least one fuel price is required")
	}

	now := time.Now().UTC()
	effectiveFromUTC := effectiveFrom.UTC()

	tx, err := r.db.Begin()
	if err != nil {
		return model.PriceVersion{}, fmt.Errorf("begin create version tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.Exec(
		`INSERT INTO price_versions(version_tag, effective_from, created_at) VALUES (?, ?, ?)`,
		versionTag, effectiveFromUTC, now,
	)
	if err != nil {
		return model.PriceVersion{}, fmt.Errorf("insert price version: %w", err)
	}
	versionID, err := res.LastInsertId()
	if err != nil {
		return model.PriceVersion{}, fmt.Errorf("resolve version id: %w", err)
	}

	result := model.PriceVersion{
		ID:            versionID,
		VersionTag:    versionTag,
		EffectiveFrom: effectiveFromUTC,
		CreatedAt:     now,
		Items:         make([]model.PriceVersionItem, 0, len(items)),
	}

	for _, item := range items {
		if _, err = tx.Exec(
			`INSERT INTO fuel_prices(
				price_version_id, fuel_type, display_name, grade, price_per_liter_minor, currency, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			versionID,
			item.FuelType,
			item.DisplayName,
			item.Grade,
			item.PriceMinor,
			service.DefaultPricingCurrency,
			now,
		); err != nil {
			return model.PriceVersion{}, fmt.Errorf("insert fuel price for %s: %w", item.FuelType, err)
		}
		result.Items = append(result.Items, model.PriceVersionItem{
			FuelType:      item.FuelType,
			DisplayName:   item.DisplayName,
			Grade:         item.Grade,
			PricePerLiter: float64(item.PriceMinor) / 100.0,
			Currency:      service.DefaultPricingCurrency,
		})
	}

	if err = tx.Commit(); err != nil {
		return model.PriceVersion{}, fmt.Errorf("commit create version tx: %w", err)
	}
	return result, nil
}

func (r *SQLitePriceRepository) ListVersions(limit int) ([]model.PriceVersion, error) {
	query := `
SELECT id, version_tag, effective_from, created_at
FROM price_versions
ORDER BY effective_from DESC, id DESC`
	args := []any{}
	if limit > 0 {
		query += "\nLIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list price versions: %w", err)
	}
	defer rows.Close()

	var versions []model.PriceVersion
	for rows.Next() {
		var v model.PriceVersion
		if err := rows.Scan(&v.ID, &v.VersionTag, &v.EffectiveFrom, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan price version: %w", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate price versions: %w", err)
	}

	for i := range versions {
		items, err := r.listVersionItems(versions[i].ID)
		if err != nil {
			return nil, err
		}
		versions[i].Items = items
	}
	return versions, nil
}

func (r *SQLitePriceRepository) listVersionItems(versionID int64) ([]model.PriceVersionItem, error) {
	const q = `
SELECT fuel_type, display_name, grade, price_per_liter_minor, currency
FROM fuel_prices
WHERE price_version_id = ?
ORDER BY fuel_type ASC`

	rows, err := r.db.Query(q, versionID)
	if err != nil {
		return nil, fmt.Errorf("list version items: %w", err)
	}
	defer rows.Close()

	var items []model.PriceVersionItem
	for rows.Next() {
		var item model.PriceVersionItem
		var priceMinor int64
		if err := rows.Scan(&item.FuelType, &item.DisplayName, &item.Grade, &priceMinor, &item.Currency); err != nil {
			return nil, fmt.Errorf("scan version item: %w", err)
		}
		item.PricePerLiter = float64(priceMinor) / 100.0
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate version items: %w", err)
	}
	return items, nil
}

func (r *SQLitePriceRepository) GetCurrentPrice(now time.Time, fuelType string) (model.FuelPriceSnapshot, error) {
	const q = `
SELECT pv.id, pv.version_tag, pv.effective_from, fp.fuel_type, fp.display_name, fp.grade, fp.price_per_liter_minor, fp.currency
FROM fuel_prices fp
INNER JOIN price_versions pv ON pv.id = fp.price_version_id
WHERE fp.fuel_type = ? AND pv.effective_from <= ?
ORDER BY pv.effective_from DESC, pv.id DESC
LIMIT 1;`

	var snapshot model.FuelPriceSnapshot
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
		return model.FuelPriceSnapshot{}, fmt.Errorf("price for fuel type %q not found", fuelType)
	}
	if err != nil {
		return model.FuelPriceSnapshot{}, fmt.Errorf("query current price: %w", err)
	}
	return snapshot, nil
}

func (r *SQLitePriceRepository) ListCurrentPrices(now time.Time) ([]model.FuelPriceSnapshot, error) {
	const q = `
WITH current_version AS (
	SELECT id, version_tag, effective_from
	FROM price_versions
	WHERE effective_from <= ?
	ORDER BY effective_from DESC, id DESC
	LIMIT 1
)
SELECT cv.id, cv.version_tag, cv.effective_from, fp.fuel_type, fp.display_name, fp.grade, fp.price_per_liter_minor, fp.currency
FROM fuel_prices fp
INNER JOIN current_version cv ON cv.id = fp.price_version_id
ORDER BY fp.fuel_type ASC;`

	rows, err := r.db.Query(q, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("list current prices: %w", err)
	}
	defer rows.Close()

	var result []model.FuelPriceSnapshot
	for rows.Next() {
		var row model.FuelPriceSnapshot
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
