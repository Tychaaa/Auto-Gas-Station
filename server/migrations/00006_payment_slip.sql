-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN payment_slip TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite не поддерживает DROP COLUMN в старых версиях — оставляем колонку
SELECT 1;
-- +goose StatementEnd
