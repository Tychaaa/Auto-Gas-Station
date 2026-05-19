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

// ErrKKTShiftReportNotFound - запись о Z-отчёте с таким ID не найдена.
var ErrKKTShiftReportNotFound = errors.New("kkt shift report not found")

type SQLiteKKTShiftReportsRepository struct {
	db *sql.DB
}

func NewSQLiteKKTShiftReportsRepository(dbPath string) (*SQLiteKKTShiftReportsRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open kkt shift reports sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &SQLiteKKTShiftReportsRepository{db: db}, nil
}

func (r *SQLiteKKTShiftReportsRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// Save сохраняет запись о закрытии смены и возвращает присвоенный id.
func (r *SQLiteKKTShiftReportsRepository) Save(ctx context.Context, rep model.KKTShiftReport) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO kkt_shift_reports (shift_number, fd_number, fiscal_sign, closed_at) VALUES (?, ?, ?, ?)`,
		rep.ShiftNumber, rep.FDNumber, rep.FiscalSign, rep.ClosedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	)
	if err != nil {
		return 0, fmt.Errorf("save kkt shift report: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// List возвращает записи о закрытиях смены, отсортированные от новых к старым.
func (r *SQLiteKKTShiftReportsRepository) List(ctx context.Context, limit, offset int) ([]model.KKTShiftReport, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, shift_number, fd_number, fiscal_sign, closed_at
		 FROM kkt_shift_reports
		 ORDER BY closed_at DESC, id DESC
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list kkt shift reports: %w", err)
	}
	defer rows.Close()

	var reps []model.KKTShiftReport
	for rows.Next() {
		var rep model.KKTShiftReport
		var closedAt string
		if err := rows.Scan(&rep.ID, &rep.ShiftNumber, &rep.FDNumber, &rep.FiscalSign, &closedAt); err != nil {
			return nil, fmt.Errorf("scan kkt shift report: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, closedAt); err == nil {
			rep.ClosedAt = t
		}
		reps = append(reps, rep)
	}
	return reps, rows.Err()
}

// Delete удаляет запись по ID.
func (r *SQLiteKKTShiftReportsRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM kkt_shift_reports WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete kkt shift report: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrKKTShiftReportNotFound
	}
	return nil
}
