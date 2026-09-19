-- +goose Up
-- +goose StatementBegin
ALTER TABLE requests ADD COLUMN query_param_schema TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE requests DROP COLUMN query_param_schema;
-- +goose StatementEnd
