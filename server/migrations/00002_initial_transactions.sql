-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS transactions (
    id                    TEXT    PRIMARY KEY,
    fuel_type             TEXT    NOT NULL,
    order_mode            TEXT    NOT NULL,
    amount_rub            INTEGER NOT NULL DEFAULT 0,
    liters                REAL    NOT NULL DEFAULT 0,
    preset                TEXT    NOT NULL DEFAULT '',
    price_version_id      INTEGER NOT NULL DEFAULT 0,
    price_version_tag     TEXT    NOT NULL DEFAULT '',
    unit_price_minor      INTEGER NOT NULL DEFAULT 0,
    computed_amount_minor INTEGER NOT NULL DEFAULT 0,
    currency              TEXT    NOT NULL DEFAULT 'RUB',
    pricing_snapshot_at   DATETIME,
    price_locked_until    DATETIME,
    price_was_repriced    INTEGER NOT NULL DEFAULT 0,
    status                TEXT    NOT NULL,
    payment_status        TEXT    NOT NULL,
    fiscal_status         TEXT    NOT NULL,
    fueling_status        TEXT    NOT NULL,
    payment_provider      TEXT    NOT NULL DEFAULT '',
    payment_session_id    TEXT    NOT NULL DEFAULT '',
    payment_error         TEXT    NOT NULL DEFAULT '',
    fiscal_error          TEXT    NOT NULL DEFAULT '',
    receipt_number        TEXT    NOT NULL DEFAULT '',
    fueling_error         TEXT    NOT NULL DEFAULT '',
    fueling_session_id    TEXT    NOT NULL DEFAULT '',
    dispensed_liters      REAL    NOT NULL DEFAULT 0,
    dispense_complete     INTEGER NOT NULL DEFAULT 0,
    dispense_partial      INTEGER NOT NULL DEFAULT 0,
    abandon_reason        TEXT    NOT NULL DEFAULT '',
    created_at            DATETIME NOT NULL,
    updated_at            DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_status     ON transactions(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_status;
DROP INDEX IF EXISTS idx_transactions_created_at;
DROP TABLE IF EXISTS transactions;
-- +goose StatementEnd
