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

func (r *SQLiteDispenserRepository) InitDispensers(count int) error {
	for i := 1; i <= count; i++ {
		label := fmt.Sprintf("Колонка %d", i)
		_, err := r.db.Exec(
			`INSERT OR IGNORE INTO dispensers (id, fuel_type, label, updated_at) VALUES (?, '', ?, ?)`,
			i, label, time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("init dispenser %d: %w", i, err)
		}
	}
	return nil
}

func (r *SQLiteDispenserRepository) List() ([]*model.Dispenser, error) {
	rows, err := r.db.Query(`SELECT id, fuel_type, label, updated_at FROM dispensers ORDER BY id ASC`)
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
	row := r.db.QueryRow(`SELECT id, fuel_type, label, updated_at FROM dispensers WHERE fuel_type = ?`, fuelType)
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
	row := r.db.QueryRow(`SELECT id, fuel_type, label, updated_at FROM dispensers WHERE id = ?`, id)
	d, err := scanDispenser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDispenserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dispenser by id: %w", err)
	}
	return d, nil
}

func (r *SQLiteDispenserRepository) SetFuelType(id int, fuelType string) (*model.Dispenser, error) {
	now := time.Now().UTC()
	_, err := r.db.Exec(
		`UPDATE dispensers SET fuel_type = ?, updated_at = ? WHERE id = ?`,
		fuelType, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("set dispenser fuel type: %w", err)
	}
	return r.GetByID(id)
}

type dispenserScanner interface {
	Scan(dest ...any) error
}

func scanDispenser(s dispenserScanner) (*model.Dispenser, error) {
	var d model.Dispenser
	if err := s.Scan(&d.ID, &d.FuelType, &d.Label, &d.UpdatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}
