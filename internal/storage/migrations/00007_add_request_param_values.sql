-- +goose Up
-- +goose StatementBegin
ALTER TABLE requests ADD COLUMN param_values TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE requests DROP COLUMN param_values;
-- +goose StatementEnd
