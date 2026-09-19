-- +goose Up
-- +goose StatementBegin
ALTER TABLE servers ADD COLUMN variables TEXT NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE servers DROP COLUMN variables;
-- +goose StatementEnd
