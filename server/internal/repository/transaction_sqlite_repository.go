package repository

import (
	"database/sql"
	"encoding/json"
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


const selectTransactionSQL = `SELECT
	id, fuel_type, order_mode, amount_rub, liters, preset,
	price_version_id, price_version_tag, unit_price_minor, computed_amount_minor, currency,
	pricing_snapshot_at, price_locked_until, price_was_repriced,
	status, payment_status, fiscal_status, fueling_status,
	payment_provider, payment_session_id, payment_error, fiscal_error, receipt_number,
	fueling_error, fueling_session_id, dispensed_liters, dispense_complete, dispense_partial,
	abandon_reason, payment_slip, created_at, updated_at
FROM transactions`

const insertTransactionSQL = `INSERT INTO transactions (
	id, fuel_type, order_mode, amount_rub, liters, preset,
	price_version_id, price_version_tag, unit_price_minor, computed_amount_minor, currency,
	pricing_snapshot_at, price_locked_until, price_was_repriced,
	status, payment_status, fiscal_status, fueling_status,
	payment_provider, payment_session_id, payment_error, fiscal_error, receipt_number,
	fueling_error, fueling_session_id, dispensed_liters, dispense_complete, dispense_partial,
	abandon_reason, payment_slip, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const updateTransactionSQL = `UPDATE transactions SET
	fuel_type=?, order_mode=?, amount_rub=?, liters=?, preset=?,
	price_version_id=?, price_version_tag=?, unit_price_minor=?, computed_amount_minor=?, currency=?,
	pricing_snapshot_at=?, price_locked_until=?, price_was_repriced=?,
	status=?, payment_status=?, fiscal_status=?, fueling_status=?,
	payment_provider=?, payment_session_id=?, payment_error=?, fiscal_error=?, receipt_number=?,
	fueling_error=?, fueling_session_id=?, dispensed_liters=?, dispense_complete=?, dispense_partial=?,
	abandon_reason=?, payment_slip=?, updated_at=?
WHERE id = ?`

func (r *SQLiteTransactionRepository) Create(tx *model.Transaction) (*model.Transaction, error) {
	now := time.Now()
	copyTx := *tx
	copyTx.ID = r.nextID()
	copyTx.CreatedAt = now
	copyTx.UpdatedAt = now

	sqlTx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin create tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = sqlTx.Rollback()
		}
	}()

	if _, err = sqlTx.Exec(insertTransactionSQL, insertArgs(&copyTx)...); err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	if _, err = sqlTx.Exec(
		`INSERT INTO transaction_events (transaction_id, event_type, occurred_at, detail) VALUES (?, ?, ?, ?)`,
		copyTx.ID, string(model.TxEventCreated), now.UTC(), "",
	); err != nil {
		return nil, fmt.Errorf("insert created event: %w", err)
	}
	if err = sqlTx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create tx: %w", err)
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

	oldTx := *tx

	if err = apply(tx); err != nil {
		return nil, err
	}
	tx.UpdatedAt = time.Now()

	if _, err = sqlTx.Exec(updateTransactionSQL, updateArgs(tx)...); err != nil {
		return nil, fmt.Errorf("update transaction: %w", err)
	}

	events := detectEvents(&oldTx, tx)
	for _, ev := range events {
		if _, err = sqlTx.Exec(
			`INSERT INTO transaction_events (transaction_id, event_type, occurred_at, detail) VALUES (?, ?, ?, ?)`,
			tx.ID, string(ev.EventType), ev.OccurredAt.UTC(), ev.Detail,
		); err != nil {
			return nil, fmt.Errorf("insert transaction event %s: %w", ev.EventType, err)
		}
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

func (r *SQLiteTransactionRepository) GetEvents(txID string) ([]model.TransactionEvent, error) {
	rows, err := r.db.Query(
		`SELECT id, transaction_id, event_type, occurred_at, detail FROM transaction_events WHERE transaction_id = ? ORDER BY id ASC`,
		txID,
	)
	if err != nil {
		return nil, fmt.Errorf("get transaction events: %w", err)
	}
	defer rows.Close()

	var result []model.TransactionEvent
	for rows.Next() {
		var ev model.TransactionEvent
		var eventTypeStr string
		if err := rows.Scan(&ev.ID, &ev.TransactionID, &eventTypeStr, &ev.OccurredAt, &ev.Detail); err != nil {
			return nil, fmt.Errorf("scan event row: %w", err)
		}
		ev.EventType = model.TransactionEventType(eventTypeStr)
		result = append(result, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return result, nil
}

// detectEvents сравнивает состояние транзакции до и после Update и возвращает
// список событий, которые нужно записать в журнал.
func detectEvents(old, next *model.Transaction) []model.TransactionEvent {
	var events []model.TransactionEvent
	now := time.Now()
	add := func(et model.TransactionEventType, detail string) {
		events = append(events, model.TransactionEvent{
			TransactionID: next.ID,
			EventType:     et,
			OccurredAt:    now,
			Detail:        detail,
		})
	}

	// Изменение выбора (оба в selection, но что-то поменялось)
	if old.Status == model.TransactionStatusSelection && next.Status == model.TransactionStatusSelection {
		if old.FuelType != next.FuelType || old.OrderMode != next.OrderMode ||
			old.AmountRub != next.AmountRub || old.Liters != next.Liters || old.Preset != next.Preset {
			add(model.TxEventSelectionUpdated, next.FuelType)
		}
	}

	// Переходы платёжного контура
	if old.Status == model.TransactionStatusSelection && next.Status == model.TransactionStatusPaymentPending {
		add(model.TxEventPaymentStarted, "")
	}
	if old.Status == model.TransactionStatusPaymentPending && next.Status == model.TransactionStatusPaid {
		add(model.TxEventPaymentApproved, "")
	}
	if old.Status == model.TransactionStatusPaymentPending && next.Status == model.TransactionStatusFailed {
		add(model.TxEventPaymentDeclined, next.PaymentError)
	}

	// Переходы фискального контура
	if old.Status == model.TransactionStatusPaid && next.Status == model.TransactionStatusFiscalizing {
		add(model.TxEventFiscalizingStarted, "")
	}
	if old.Status == model.TransactionStatusFiscalizing && next.FiscalStatus == model.FiscalStatusDone {
		add(model.TxEventReceiptIssued, next.ReceiptNumber)
	}
	if old.Status == model.TransactionStatusFiscalizing && next.Status == model.TransactionStatusFailed {
		add(model.TxEventFiscalFailed, next.FiscalError)
	}

	// Переходы топливного контура
	if old.Status == model.TransactionStatusPaid && next.Status == model.TransactionStatusFueling {
		add(model.TxEventFuelingStarted, "")
	}
	if old.FuelingStatus != model.FuelingStatusDispensing && next.FuelingStatus == model.FuelingStatusDispensing {
		add(model.TxEventFuelingDispensing, "")
	}
	if !old.DispenseComplete && next.DispenseComplete {
		detail := fmt.Sprintf("%.3f л", next.DispensedLiters)
		if next.DispensePartial {
			detail += " (частично)"
		}
		add(model.TxEventFuelingCompleted, detail)
	}
	if old.Status == model.TransactionStatusFueling && next.Status == model.TransactionStatusFailed {
		add(model.TxEventFuelingFailed, next.FuelingError)
	}

	// paid → failed (AbortFuelingFromPaid)
	if old.Status == model.TransactionStatusPaid && next.Status == model.TransactionStatusFailed {
		add(model.TxEventFailed, next.FuelingError)
	}

	// Терминальные состояния
	if old.Status != model.TransactionStatusCompleted && next.Status == model.TransactionStatusCompleted {
		add(model.TxEventCompleted, "")
	}
	if old.Status != model.TransactionStatusAbandoned && next.Status == model.TransactionStatusAbandoned {
		add(model.TxEventAbandoned, next.AbandonReason)
	}

	return events
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
	var paymentSlipJSON sql.NullString

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
		&paymentSlipJSON,
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

	if paymentSlipJSON.Valid && paymentSlipJSON.String != "" {
		var slip model.PaymentSlip
		if err := json.Unmarshal([]byte(paymentSlipJSON.String), &slip); err == nil {
			tx.PaymentSlip = &slip
		}
	}

	return &tx, nil
}

func marshalSlip(slip *model.PaymentSlip) sql.NullString {
	if slip == nil {
		return sql.NullString{}
	}
	raw, err := json.Marshal(slip)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(raw), Valid: true}
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
		marshalSlip(tx.PaymentSlip),
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
		marshalSlip(tx.PaymentSlip),
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
