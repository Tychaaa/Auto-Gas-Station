-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS dispensers (
    id         INTEGER  PRIMARY KEY,
    fuel_type  TEXT     NOT NULL DEFAULT '',
    label      TEXT     NOT NULL DEFAULT '',
    enabled    INTEGER  NOT NULL DEFAULT 1,
    sort_order INTEGER  NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN dispenser_id INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS dispensers;
-- +goose StatementEnd
