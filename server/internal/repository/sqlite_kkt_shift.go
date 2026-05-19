package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"AUTO-GAS-STATION/server/internal/model"
	_ "modernc.org/sqlite"
)

// ErrKKTShiftNotFound - состояние смены ещё не записано (смена ни разу не открывалась после старта).
var ErrKKTShiftNotFound = errors.New("kkt shift state not found")

type SQLiteKKTShiftRepository struct {
	db *sql.DB
}

func NewSQLiteKKTShiftRepository(dbPath string) (*SQLiteKKTShiftRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open kkt shift sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &SQLiteKKTShiftRepository{db: db}, nil
}

func (r *SQLiteKKTShiftRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// Load возвращает сохранённое состояние смены или nil, если запись отсутствует.
func (r *SQLiteKKTShiftRepository) Load(ctx context.Context) (*model.KKTShiftState, error) {
	var shiftNumber uint16
	var openedAtStr string
	err := r.db.QueryRowContext(ctx, `SELECT shift_number, opened_at FROM kkt_shift_state WHERE id = 1`).
		Scan(&shiftNumber, &openedAtStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load kkt shift state: %w", err)
	}
	openedAt, err := time.Parse(time.RFC3339, openedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse kkt shift opened_at %q: %w", openedAtStr, err)
	}
	return &model.KKTShiftState{ShiftNumber: shiftNumber, OpenedAt: openedAt}, nil
}

// Save сохраняет (upsert) состояние открытой смены.
func (r *SQLiteKKTShiftRepository) Save(ctx context.Context, state model.KKTShiftState) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO kkt_shift_state (id, shift_number, opened_at) VALUES (1, ?, ?)
         ON CONFLICT(id) DO UPDATE SET shift_number = excluded.shift_number, opened_at = excluded.opened_at`,
		state.ShiftNumber, state.OpenedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("save kkt shift state: %w", err)
	}
	return nil
}

// Clear удаляет запись состояния смены (смена закрыта).
func (r *SQLiteKKTShiftRepository) Clear(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM kkt_shift_state WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("clear kkt shift state: %w", err)
	}
	return nil
}
