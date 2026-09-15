-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS servers (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    base_url TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS request_groups (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    server_id INTEGER REFERENCES servers(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE requests ADD COLUMN group_id INTEGER REFERENCES request_groups(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE requests DROP COLUMN group_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS request_groups;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS servers;
-- +goose StatementEnd
