package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"AUTO-GAS-STATION/server/internal/model"
	_ "modernc.org/sqlite"
)

var ErrDispenserNotFound = errors.New("dispenser not found")

type SQLiteDispenserRepository struct {
	db *sql.DB
}

func NewSQLiteDispenserRepository(dbPath string) (*SQLiteDispenserRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open dispenser sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &SQLiteDispenserRepository{db: db}, nil
}

func (r *SQLiteDispenserRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

var defaultFuelTypes = map[int]string{
	1: "АИ-92",
	2: "АИ-95",
	3: "АИ-100",
	4: "ДТ",
}

func (r *SQLiteDispenserRepository) InitDispensers(addresses []int) error {
	for pos, addr := range addresses {
		label := fmt.Sprintf("Колонка %d", pos+1)
		fuelType := defaultFuelTypes[pos+1]
		_, err := r.db.Exec(`
			INSERT INTO dispensers (id, fuel_type, label, sort_order, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				label      = excluded.label,
				sort_order = excluded.sort_order`,
			addr, fuelType, label, pos+1, time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("init dispenser addr=%d: %w", addr, err)
		}
	}
	return nil
}

func (r *SQLiteDispenserRepository) Count() (int, error) {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM dispensers`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count dispensers: %w", err)
	}
	return count, nil
}

func (r *SQLiteDispenserRepository) List() ([]*model.Dispenser, error) {
	rows, err := r.db.Query(`SELECT id, fuel_type, label, enabled, sort_order, updated_at FROM dispensers ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list dispensers: %w", err)
	}
	defer rows.Close()

	var result []*model.Dispenser
	for rows.Next() {
		d, err := scanDispenser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dispenser: %w", err)
		}
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dispensers: %w", err)
	}
	return result, nil
}

func (r *SQLiteDispenserRepository) GetByFuelType(fuelType string) (*model.Dispenser, error) {
	row := r.db.QueryRow(
		`SELECT id, fuel_type, label, enabled, sort_order, updated_at FROM dispensers WHERE fuel_type = ? AND enabled = 1`,
		fuelType,
	)
	d, err := scanDispenser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDispenserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dispenser by fuel type: %w", err)
	}
	return d, nil
}

func (r *SQLiteDispenserRepository) GetByID(id int) (*model.Dispenser, error) {
	row := r.db.QueryRow(`SELECT id, fuel_type, label, enabled, sort_order, updated_at FROM dispensers WHERE id = ?`, id)
	d, err := scanDispenser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDispenserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dispenser by id: %w", err)
	}
	return d, nil
}

func (r *SQLiteDispenserRepository) Update(id int, fuelType string, enabled bool) (*model.Dispenser, error) {
	now := time.Now().UTC()
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := r.db.Exec(
		`UPDATE dispensers SET fuel_type = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		fuelType, enabledInt, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update dispenser: %w", err)
	}
	return r.GetByID(id)
}

func (r *SQLiteDispenserRepository) Add() (*model.Dispenser, error) {
	now := time.Now().UTC()
	var maxID int
	if err := r.db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM dispensers`).Scan(&maxID); err != nil {
		return nil, fmt.Errorf("get max dispenser id: %w", err)
	}
	newID := maxID + 1
	label := fmt.Sprintf("Колонка %d", newID)
	_, err := r.db.Exec(
		`INSERT INTO dispensers (id, fuel_type, label, enabled, updated_at) VALUES (?, '', ?, 1, ?)`,
		newID, label, now,
	)
	if err != nil {
		return nil, fmt.Errorf("add dispenser: %w", err)
	}
	return r.GetByID(newID)
}

func (r *SQLiteDispenserRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM dispensers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete dispenser: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrDispenserNotFound
	}
	return nil
}

type dispenserScanner interface {
	Scan(dest ...any) error
}

func scanDispenser(s dispenserScanner) (*model.Dispenser, error) {
	var d model.Dispenser
	var enabledInt int
	if err := s.Scan(&d.ID, &d.FuelType, &d.Label, &enabledInt, &d.SortOrder, &d.UpdatedAt); err != nil {
		return nil, err
	}
	d.Enabled = enabledInt == 1
	return &d, nil
}
