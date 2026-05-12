-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS price_versions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    version_tag    TEXT    NOT NULL UNIQUE,
    effective_from DATETIME NOT NULL,
    created_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS fuel_prices (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    price_version_id     INTEGER NOT NULL,
    fuel_type            TEXT    NOT NULL,
    display_name         TEXT    NOT NULL,
    grade                TEXT    NOT NULL,
    price_per_liter_minor INTEGER NOT NULL,
    currency             TEXT    NOT NULL DEFAULT 'RUB',
    created_at           DATETIME NOT NULL,
    UNIQUE(price_version_id, fuel_type),
    FOREIGN KEY(price_version_id) REFERENCES price_versions(id)
);

CREATE INDEX IF NOT EXISTS idx_fuel_prices_fuel_type_version
    ON fuel_prices(fuel_type, price_version_id);

CREATE INDEX IF NOT EXISTS idx_price_versions_effective_from
    ON price_versions(effective_from);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_price_versions_effective_from;
DROP INDEX IF EXISTS idx_fuel_prices_fuel_type_version;
DROP TABLE IF EXISTS fuel_prices;
DROP TABLE IF EXISTS price_versions;
-- +goose StatementEnd
