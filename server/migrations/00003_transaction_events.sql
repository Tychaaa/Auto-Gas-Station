-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS transaction_events (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id TEXT    NOT NULL,
    event_type     TEXT    NOT NULL,
    occurred_at    DATETIME NOT NULL,
    detail         TEXT    NOT NULL DEFAULT '',
    FOREIGN KEY (transaction_id) REFERENCES transactions(id)
);

CREATE INDEX IF NOT EXISTS idx_tx_events_tx_id ON transaction_events(transaction_id, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tx_events_tx_id;
DROP TABLE IF EXISTS transaction_events;
-- +goose StatementEnd
