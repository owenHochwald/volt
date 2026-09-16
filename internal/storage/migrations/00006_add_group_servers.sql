-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS group_servers (
    group_id INTEGER NOT NULL REFERENCES request_groups(id) ON DELETE CASCADE,
    server_id INTEGER NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, server_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO group_servers (group_id, server_id)
SELECT id, server_id FROM request_groups WHERE server_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS group_servers;
-- +goose StatementEnd
