-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS kkt_shift_reports (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    shift_number INTEGER NOT NULL,
    fd_number    INTEGER NOT NULL,
    fiscal_sign  INTEGER NOT NULL,
    closed_at    TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_kkt_shift_reports_closed_at
    ON kkt_shift_reports(closed_at DESC);

CREATE TABLE IF NOT EXISTS kkt_calc_reports (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    fd_number              INTEGER NOT NULL,
    fiscal_sign            INTEGER NOT NULL,
    unconfirmed_count      INTEGER NOT NULL,
    first_unconfirmed_date TEXT    NULL,
    kkt_datetime           TEXT    NULL,
    created_at             TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_kkt_calc_reports_created_at
    ON kkt_calc_reports(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS kkt_calc_reports;
DROP TABLE IF EXISTS kkt_shift_reports;
-- +goose StatementEnd
