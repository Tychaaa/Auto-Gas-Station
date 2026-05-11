package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"AUTO-GAS-STATION/server/internal/model"
	_ "modernc.org/sqlite"
)

var ErrTransactionNotFound = errors.New("transaction not found")

type SQLiteTransactionRepository struct {
	db      *sql.DB
	counter uint64
}

func NewSQLiteTransactionRepository(dbPath string) (*SQLiteTransactionRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open transaction sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &SQLiteTransactionRepository{db: db}, nil
}

func (r *SQLiteTransactionRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *SQLiteTransactionRepository) InitSchema() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS transactions (
			id                    TEXT PRIMARY KEY,
			fuel_type             TEXT NOT NULL,
			order_mode            TEXT NOT NULL,
			amount_rub            INTEGER NOT NULL DEFAULT 0,
			liters                REAL NOT NULL DEFAULT 0,
			preset                TEXT NOT NULL DEFAULT '',
			price_version_id      INTEGER NOT NULL DEFAULT 0,
			price_version_tag     TEXT NOT NULL DEFAULT '',
			unit_price_minor      INTEGER NOT NULL DEFAULT 0,
			computed_amount_minor INTEGER NOT NULL DEFAULT 0,
			currency              TEXT NOT NULL DEFAULT 'RUB',
			pricing_snapshot_at   DATETIME,
			price_locked_until    DATETIME,
			price_was_repriced    INTEGER NOT NULL DEFAULT 0,
			status                TEXT NOT NULL,
			payment_status        TEXT NOT NULL,
			fiscal_status         TEXT NOT NULL,
			fueling_status        TEXT NOT NULL,
			payment_provider      TEXT NOT NULL DEFAULT '',
			payment_session_id    TEXT NOT NULL DEFAULT '',
			payment_error         TEXT NOT NULL DEFAULT '',
			fiscal_error          TEXT NOT NULL DEFAULT '',
			receipt_number        TEXT NOT NULL DEFAULT '',
			fueling_error         TEXT NOT NULL DEFAULT '',
			fueling_session_id    TEXT NOT NULL DEFAULT '',
			dispensed_liters      REAL NOT NULL DEFAULT 0,
			dispense_complete     INTEGER NOT NULL DEFAULT 0,
			dispense_partial      INTEGER NOT NULL DEFAULT 0,
			abandon_reason        TEXT NOT NULL DEFAULT '',
			created_at            DATETIME NOT NULL,
			updated_at            DATETIME NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);`,
	}
	for _, stmt := range schema {
		if _, err := r.db.Exec(stmt); err != nil {
			return fmt.Errorf("init transactions schema: %w", err)
		}
	}
	return nil
}

const selectTransactionSQL = `SELECT
	id, fuel_type, order_mode, amount_rub, liters, preset,
	price_version_id, price_version_tag, unit_price_minor, computed_amount_minor, currency,
	pricing_snapshot_at, price_locked_until, price_was_repriced,
	status, payment_status, fiscal_status, fueling_status,
	payment_provider, payment_session_id, payment_error, fiscal_error, receipt_number,
	fueling_error, fueling_session_id, dispensed_liters, dispense_complete, dispense_partial,
	abandon_reason, created_at, updated_at
FROM transactions`

const insertTransactionSQL = `INSERT INTO transactions (
	id, fuel_type, order_mode, amount_rub, liters, preset,
	price_version_id, price_version_tag, unit_price_minor, computed_amount_minor, currency,
	pricing_snapshot_at, price_locked_until, price_was_repriced,
	status, payment_status, fiscal_status, fueling_status,
	payment_provider, payment_session_id, payment_error, fiscal_error, receipt_number,
	fueling_error, fueling_session_id, dispensed_liters, dispense_complete, dispense_partial,
	abandon_reason, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const updateTransactionSQL = `UPDATE transactions SET
	fuel_type=?, order_mode=?, amount_rub=?, liters=?, preset=?,
	price_version_id=?, price_version_tag=?, unit_price_minor=?, computed_amount_minor=?, currency=?,
	pricing_snapshot_at=?, price_locked_until=?, price_was_repriced=?,
	status=?, payment_status=?, fiscal_status=?, fueling_status=?,
	payment_provider=?, payment_session_id=?, payment_error=?, fiscal_error=?, receipt_number=?,
	fueling_error=?, fueling_session_id=?, dispensed_liters=?, dispense_complete=?, dispense_partial=?,
	abandon_reason=?, updated_at=?
WHERE id = ?`

func (r *SQLiteTransactionRepository) Create(tx *model.Transaction) (*model.Transaction, error) {
	now := time.Now()
	copyTx := *tx
	copyTx.ID = r.nextID()
	copyTx.CreatedAt = now
	copyTx.UpdatedAt = now

	if _, err := r.db.Exec(insertTransactionSQL, insertArgs(&copyTx)...); err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	return &copyTx, nil
}

func (r *SQLiteTransactionRepository) Get(id string) (*model.Transaction, error) {
	row := r.db.QueryRow(selectTransactionSQL+" WHERE id = ?", id)
	tx, err := scanTransaction(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTransactionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	return tx, nil
}

func (r *SQLiteTransactionRepository) Update(id string, apply func(*model.Transaction) error) (result *model.Transaction, err error) {
	sqlTx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin update tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = sqlTx.Rollback()
		}
	}()

	row := sqlTx.QueryRow(selectTransactionSQL+" WHERE id = ?", id)
	tx, err := scanTransaction(row)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrTransactionNotFound
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("get transaction for update: %w", err)
	}

	if err = apply(tx); err != nil {
		return nil, err
	}
	tx.UpdatedAt = time.Now()

	if _, err = sqlTx.Exec(updateTransactionSQL, updateArgs(tx)...); err != nil {
		return nil, fmt.Errorf("update transaction: %w", err)
	}

	if err = sqlTx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update tx: %w", err)
	}
	return tx, nil
}

func (r *SQLiteTransactionRepository) ListAll() ([]*model.Transaction, error) {
	rows, err := r.db.Query(selectTransactionSQL + " ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var result []*model.Transaction
	for rows.Next() {
		tx, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction row: %w", err)
		}
		result = append(result, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return result, nil
}

func (r *SQLiteTransactionRepository) nextID() string {
	n := atomic.AddUint64(&r.counter, 1)
	return fmt.Sprintf("tx_%d_%06d", time.Now().UnixNano(), n)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTransaction(s scanner) (*model.Transaction, error) {
	var tx model.Transaction
	var pricingSnapshotAt, priceLockedUntil sql.NullTime
	var priceWasRepriced, dispenseComplete, dispensePartial int64
	var status, paymentStatus, fiscalStatus, fuelingStatus string

	err := s.Scan(
		&tx.ID,
		&tx.FuelType,
		&tx.OrderMode,
		&tx.AmountRub,
		&tx.Liters,
		&tx.Preset,
		&tx.PriceVersionID,
		&tx.PriceVersionTag,
		&tx.UnitPriceMinor,
		&tx.ComputedAmountMinor,
		&tx.Currency,
		&pricingSnapshotAt,
		&priceLockedUntil,
		&priceWasRepriced,
		&status,
		&paymentStatus,
		&fiscalStatus,
		&fuelingStatus,
		&tx.PaymentProvider,
		&tx.PaymentSessionID,
		&tx.PaymentError,
		&tx.FiscalError,
		&tx.ReceiptNumber,
		&tx.FuelingError,
		&tx.FuelingSessionID,
		&tx.DispensedLiters,
		&dispenseComplete,
		&dispensePartial,
		&tx.AbandonReason,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	tx.PricingSnapshotAt = fromNullTime(pricingSnapshotAt)
	tx.PriceLockedUntil = fromNullTime(priceLockedUntil)
	tx.PriceWasRepriced = priceWasRepriced != 0
	tx.DispenseComplete = dispenseComplete != 0
	tx.DispensePartial = dispensePartial != 0
	tx.Status = model.TransactionStatus(status)
	tx.PaymentStatus = model.PaymentStatus(paymentStatus)
	tx.FiscalStatus = model.FiscalStatus(fiscalStatus)
	tx.FuelingStatus = model.FuelingStatus(fuelingStatus)

	return &tx, nil
}

func insertArgs(tx *model.Transaction) []any {
	return []any{
		tx.ID,
		tx.FuelType,
		tx.OrderMode,
		tx.AmountRub,
		tx.Liters,
		tx.Preset,
		tx.PriceVersionID,
		tx.PriceVersionTag,
		tx.UnitPriceMinor,
		tx.ComputedAmountMinor,
		tx.Currency,
		nullTime(tx.PricingSnapshotAt),
		nullTime(tx.PriceLockedUntil),
		btoi(tx.PriceWasRepriced),
		string(tx.Status),
		string(tx.PaymentStatus),
		string(tx.FiscalStatus),
		string(tx.FuelingStatus),
		tx.PaymentProvider,
		tx.PaymentSessionID,
		tx.PaymentError,
		tx.FiscalError,
		tx.ReceiptNumber,
		tx.FuelingError,
		tx.FuelingSessionID,
		tx.DispensedLiters,
		btoi(tx.DispenseComplete),
		btoi(tx.DispensePartial),
		tx.AbandonReason,
		tx.CreatedAt.UTC(),
		tx.UpdatedAt.UTC(),
	}
}

func updateArgs(tx *model.Transaction) []any {
	return []any{
		tx.FuelType,
		tx.OrderMode,
		tx.AmountRub,
		tx.Liters,
		tx.Preset,
		tx.PriceVersionID,
		tx.PriceVersionTag,
		tx.UnitPriceMinor,
		tx.ComputedAmountMinor,
		tx.Currency,
		nullTime(tx.PricingSnapshotAt),
		nullTime(tx.PriceLockedUntil),
		btoi(tx.PriceWasRepriced),
		string(tx.Status),
		string(tx.PaymentStatus),
		string(tx.FiscalStatus),
		string(tx.FuelingStatus),
		tx.PaymentProvider,
		tx.PaymentSessionID,
		tx.PaymentError,
		tx.FiscalError,
		tx.ReceiptNumber,
		tx.FuelingError,
		tx.FuelingSessionID,
		tx.DispensedLiters,
		btoi(tx.DispenseComplete),
		btoi(tx.DispensePartial),
		tx.AbandonReason,
		tx.UpdatedAt.UTC(),
		tx.ID,
	}
}

func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.UTC(), Valid: true}
}

func fromNullTime(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func btoi(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
