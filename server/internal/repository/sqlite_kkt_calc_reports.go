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

// ErrKKTCalcReportNotFound - запись об отчёте о состоянии расчётов с таким ID не найдена.
var ErrKKTCalcReportNotFound = errors.New("kkt calc report not found")

type SQLiteKKTCalcReportsRepository struct {
	db *sql.DB
}

func NewSQLiteKKTCalcReportsRepository(dbPath string) (*SQLiteKKTCalcReportsRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open kkt calc reports sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &SQLiteKKTCalcReportsRepository{db: db}, nil
}

func (r *SQLiteKKTCalcReportsRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// Save сохраняет запись об отчёте о состоянии расчётов и возвращает присвоенный id.
func (r *SQLiteKKTCalcReportsRepository) Save(ctx context.Context, rep model.KKTCalcReport) (int64, error) {
	var firstDate *string
	if rep.FirstUnconfirmedDate != nil {
		s := rep.FirstUnconfirmedDate.Format("2006-01-02")
		firstDate = &s
	}
	var kktDT *string
	if rep.KKTDateTime != nil {
		s := rep.KKTDateTime.UTC().Format("2006-01-02T15:04:05Z07:00")
		kktDT = &s
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO kkt_calc_reports
		 (fd_number, fiscal_sign, unconfirmed_count, first_unconfirmed_date, kkt_datetime, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		rep.FDNumber, rep.FiscalSign, rep.UnconfirmedCount,
		firstDate, kktDT,
		rep.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	)
	if err != nil {
		return 0, fmt.Errorf("save kkt calc report: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// List возвращает записи об отчётах о состоянии расчётов, отсортированные от новых к старым.
func (r *SQLiteKKTCalcReportsRepository) List(ctx context.Context, limit, offset int) ([]model.KKTCalcReport, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, fd_number, fiscal_sign, unconfirmed_count, first_unconfirmed_date, kkt_datetime, created_at
		 FROM kkt_calc_reports
		 ORDER BY created_at DESC, id DESC
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list kkt calc reports: %w", err)
	}
	defer rows.Close()

	var reps []model.KKTCalcReport
	for rows.Next() {
		var rep model.KKTCalcReport
		var firstDate sql.NullString
		var kktDT sql.NullString
		var createdAt string
		if err := rows.Scan(
			&rep.ID, &rep.FDNumber, &rep.FiscalSign, &rep.UnconfirmedCount,
			&firstDate, &kktDT, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan kkt calc report: %w", err)
		}
		if firstDate.Valid {
			if t, err := time.Parse("2006-01-02", firstDate.String); err == nil {
				rep.FirstUnconfirmedDate = &t
			}
		}
		if kktDT.Valid {
			if t, err := time.Parse(time.RFC3339, kktDT.String); err == nil {
				rep.KKTDateTime = &t
			}
		}
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			rep.CreatedAt = t
		}
		reps = append(reps, rep)
	}
	return reps, rows.Err()
}

// Delete удаляет запись по ID.
func (r *SQLiteKKTCalcReportsRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM kkt_calc_reports WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete kkt calc report: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrKKTCalcReportNotFound
	}
	return nil
}
