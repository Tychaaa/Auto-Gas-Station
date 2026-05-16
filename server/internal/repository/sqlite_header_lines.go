package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"AUTO-GAS-STATION/server/internal/model"
	_ "modernc.org/sqlite"
)

// ErrHeaderLineNotFound - строка заголовка с таким ID не найдена.
var ErrHeaderLineNotFound = errors.New("header line not found")

type SQLiteHeaderLinesRepository struct {
	db *sql.DB
}

func NewSQLiteHeaderLinesRepository(dbPath string) (*SQLiteHeaderLinesRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open header lines sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &SQLiteHeaderLinesRepository{db: db}, nil
}

func (r *SQLiteHeaderLinesRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// List возвращает все строки заголовка, упорядоченные по position.
func (r *SQLiteHeaderLinesRepository) List(ctx context.Context) ([]model.HeaderLine, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, position, text FROM kkt_header_lines ORDER BY position ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list header lines: %w", err)
	}
	defer rows.Close()

	var lines []model.HeaderLine
	for rows.Next() {
		var l model.HeaderLine
		if err := rows.Scan(&l.ID, &l.Position, &l.Text); err != nil {
			return nil, fmt.Errorf("scan header line: %w", err)
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// Replace полностью заменяет список заголовков (bulk replace в одной транзакции).
func (r *SQLiteHeaderLinesRepository) Replace(ctx context.Context, lines []model.HeaderLine) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM kkt_header_lines`); err != nil {
		return fmt.Errorf("clear header lines: %w", err)
	}
	for i, l := range lines {
		pos := l.Position
		if pos == 0 {
			pos = i + 1
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO kkt_header_lines (position, text) VALUES (?, ?)`,
			pos, l.Text,
		); err != nil {
			return fmt.Errorf("insert header line %d: %w", i, err)
		}
	}
	return tx.Commit()
}

// Create добавляет одну строку заголовка.
func (r *SQLiteHeaderLinesRepository) Create(ctx context.Context, line model.HeaderLine) (model.HeaderLine, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO kkt_header_lines (position, text) VALUES (?, ?)`,
		line.Position, line.Text,
	)
	if err != nil {
		return model.HeaderLine{}, fmt.Errorf("create header line: %w", err)
	}
	id, _ := res.LastInsertId()
	line.ID = id
	return line, nil
}

// Update обновляет существующую строку заголовка.
func (r *SQLiteHeaderLinesRepository) Update(ctx context.Context, line model.HeaderLine) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE kkt_header_lines SET position = ?, text = ? WHERE id = ?`,
		line.Position, line.Text, line.ID,
	)
	if err != nil {
		return fmt.Errorf("update header line: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrHeaderLineNotFound
	}
	return nil
}

// Delete удаляет строку заголовка по ID.
func (r *SQLiteHeaderLinesRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM kkt_header_lines WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete header line: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrHeaderLineNotFound
	}
	return nil
}
