-- +goose Up
-- +goose StatementBegin
ALTER TABLE requests ADD COLUMN description TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE requests DROP COLUMN description;
-- +goose StatementEnd
