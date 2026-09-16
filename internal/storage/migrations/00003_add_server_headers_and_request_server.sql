-- +goose Up
-- +goose StatementBegin
ALTER TABLE servers ADD COLUMN headers TEXT NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE requests ADD COLUMN server_id INTEGER REFERENCES servers(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE requests DROP COLUMN server_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE servers DROP COLUMN headers;
-- +goose StatementEnd
