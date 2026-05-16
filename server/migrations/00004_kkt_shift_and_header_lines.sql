-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS kkt_shift_state (
    id           INTEGER PRIMARY KEY CHECK (id = 1),
    shift_number INTEGER NOT NULL,
    opened_at    TEXT    NOT NULL -- RFC3339
);

CREATE TABLE IF NOT EXISTS kkt_header_lines (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    position INTEGER NOT NULL,
    text     TEXT    NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS kkt_header_lines;
DROP TABLE IF EXISTS kkt_shift_state;
-- +goose StatementEnd
