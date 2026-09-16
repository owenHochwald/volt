package storage

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type SQLiteStorage struct {
	db *sql.DB
}

func serializeHeaders(headers map[string]string) (string, error) {
	data, err := json.Marshal(headers)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
func deserializeHeaders(jsonStr string) (map[string]string, error) {
	var headers map[string]string
	err := json.Unmarshal([]byte(jsonStr), &headers)
	if err != nil {
		return nil, err
	}
	return headers, nil
}

func runMigrations(db *sql.DB) error {
	// Set the embedded filesystem for goose
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}
	return nil
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &SQLiteStorage{db: db}, nil

}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func (s *SQLiteStorage) Save(request *http.Request) error {
	headerString, err := serializeHeaders(request.Headers)
	if err != nil {
		return err
	}
	q := `INSERT INTO requests (name, method, url, headers, body, group_id) VALUES (?, ?, ?, ?, ?, ?)`

	// database/sql's default converter turns a nil *int64 into SQL NULL and
	// dereferences a non-nil one, so request.GroupID can be passed as-is.
	res, err := s.db.Exec(q, request.Name, request.Method, request.URL, headerString, request.Body, request.GroupID)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	request.ID = id
	return nil
}

func (s *SQLiteStorage) Load() ([]http.Request, error) {
	q := `SELECT id, name, method, url, headers, body, group_id FROM requests`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []http.Request

	for rows.Next() {
		var (
			id      int64
			name    string
			method  string
			url     string
			headers string
			body    string
			groupID sql.NullInt64
		)

		if err := rows.Scan(&id, &name, &method, &url, &headers, &body, &groupID); err != nil {
			return nil, err

		}
		headersMap, err := deserializeHeaders(headers)
		if err != nil {
			return nil, err
		}
		request := http.Request{
			ID:      id,
			Name:    name,
			Method:  method,
			URL:     url,
			Headers: headersMap,
			Body:    body,
		}
		if groupID.Valid {
			gid := groupID.Int64
			request.GroupID = &gid
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func (s *SQLiteStorage) Delete(id int64) error {
	q := `DELETE FROM requests WHERE id = ?`
	res, err := s.db.Exec(q, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("request not found: %d", id)
	}
	return nil
}

func (s *SQLiteStorage) GetAllURLs() ([]string, error) {
	q := `SELECT DISTINCT url FROM requests`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func checkRowsAffected(res sql.Result, kind, identifier string) error {
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("%s not found: %s", kind, identifier)
	}
	return nil
}

// --- Servers ---
// A Server is a named base URL (e.g. "prod" -> https://api.example.com)
// that request groups can point at, so a group's requests can be
// redirected at a different host just by switching servers.

func (s *SQLiteStorage) SaveServer(server *http.Server) error {
	q := `INSERT INTO servers (name, base_url) VALUES (?, ?)`
	res, err := s.db.Exec(q, server.Name, server.BaseURL)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	server.ID = id
	return nil
}

func (s *SQLiteStorage) GetServerByName(name string) (*http.Server, error) {
	q := `SELECT id, name, base_url FROM servers WHERE name = ?`
	var server http.Server
	err := s.db.QueryRow(q, name).Scan(&server.ID, &server.Name, &server.BaseURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &server, nil
}

func (s *SQLiteStorage) ListServers() ([]http.Server, error) {
	q := `SELECT id, name, base_url FROM servers ORDER BY name`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []http.Server
	for rows.Next() {
		var server http.Server
		if err := rows.Scan(&server.ID, &server.Name, &server.BaseURL); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

// UpsertServer creates a server with the given name and base URL, or
// updates its base URL if a server with that name already exists.
func (s *SQLiteStorage) UpsertServer(name, baseURL string) (*http.Server, error) {
	existing, err := s.GetServerByName(name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if err := s.UpdateServerURL(name, baseURL); err != nil {
			return nil, err
		}
		existing.BaseURL = baseURL
		return existing, nil
	}
	server := &http.Server{Name: name, BaseURL: baseURL}
	if err := s.SaveServer(server); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *SQLiteStorage) UpdateServerURL(name, baseURL string) error {
	q := `UPDATE servers SET base_url = ? WHERE name = ?`
	res, err := s.db.Exec(q, baseURL, name)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, "server", name)
}

func (s *SQLiteStorage) RenameServer(oldName, newName string) error {
	q := `UPDATE servers SET name = ? WHERE name = ?`
	res, err := s.db.Exec(q, newName, oldName)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, "server", oldName)
}

func (s *SQLiteStorage) DeleteServer(name string) error {
	q := `DELETE FROM servers WHERE name = ?`
	res, err := s.db.Exec(q, name)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, "server", name)
}

// --- Request groups ---
// A Group is a one-level folder for saved requests, optionally bound to a
// Server so its requests' relative URLs resolve against that server.

func (s *SQLiteStorage) SaveGroup(group *http.Group) error {
	q := `INSERT INTO request_groups (name, server_id) VALUES (?, ?)`
	res, err := s.db.Exec(q, group.Name, group.ServerID)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	group.ID = id
	return nil
}

func (s *SQLiteStorage) GetGroupByName(name string) (*http.Group, error) {
	q := `SELECT id, name, server_id FROM request_groups WHERE name = ?`
	var group http.Group
	var serverID sql.NullInt64
	err := s.db.QueryRow(q, name).Scan(&group.ID, &group.Name, &serverID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if serverID.Valid {
		id := serverID.Int64
		group.ServerID = &id
	}
	return &group, nil
}

// GetOrCreateGroup returns the named group, creating an empty one (no
// server assigned) if it doesn't already exist.
func (s *SQLiteStorage) GetOrCreateGroup(name string) (*http.Group, error) {
	existing, err := s.GetGroupByName(name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	group := &http.Group{Name: name}
	if err := s.SaveGroup(group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *SQLiteStorage) ListGroups() ([]http.Group, error) {
	q := `SELECT id, name, server_id FROM request_groups ORDER BY name`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []http.Group
	for rows.Next() {
		var group http.Group
		var serverID sql.NullInt64
		if err := rows.Scan(&group.ID, &group.Name, &serverID); err != nil {
			return nil, err
		}
		if serverID.Valid {
			id := serverID.Int64
			group.ServerID = &id
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (s *SQLiteStorage) SetGroupServer(groupName, serverName string) error {
	server, err := s.GetServerByName(serverName)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", serverName)
	}
	q := `UPDATE request_groups SET server_id = ? WHERE name = ?`
	res, err := s.db.Exec(q, server.ID, groupName)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, "group", groupName)
}

func (s *SQLiteStorage) RenameGroup(oldName, newName string) error {
	q := `UPDATE request_groups SET name = ? WHERE name = ?`
	res, err := s.db.Exec(q, newName, oldName)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, "group", oldName)
}

// DeleteGroup deletes the named group along with every request that
// belongs to it (SQLite foreign keys aren't enforced here, so this is done
// explicitly rather than relying on the migration's ON DELETE SET NULL).
// Returns how many requests were deleted with it.
func (s *SQLiteStorage) DeleteGroup(name string) (int64, error) {
	group, err := s.GetGroupByName(name)
	if err != nil {
		return 0, err
	}
	if group == nil {
		return 0, fmt.Errorf("group not found: %s", name)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM requests WHERE group_id = ?`, group.ID)
	if err != nil {
		return 0, err
	}
	deletedRequests, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if _, err := tx.Exec(`DELETE FROM request_groups WHERE id = ?`, group.ID); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return deletedRequests, nil
}

// ResolveEffectiveURL returns request.URL unchanged if it's already
// absolute or belongs to no group. Otherwise it looks up the request's
// group and, if one is configured, joins the URL onto that server's base
// URL. With no group or no server configured, the relative URL is
// returned as-is.
func (s *SQLiteStorage) ResolveEffectiveURL(request *http.Request) (string, error) {
	if request.IsAbsoluteURL() || request.GroupID == nil {
		return request.URL, nil
	}

	q := `SELECT s.base_url FROM request_groups g JOIN servers s ON s.id = g.server_id WHERE g.id = ?`
	var baseURL string
	err := s.db.QueryRow(q, *request.GroupID).Scan(&baseURL)
	if errors.Is(err, sql.ErrNoRows) {
		return request.URL, nil
	}
	if err != nil {
		return "", err
	}
	return request.EffectiveURL(&http.Server{BaseURL: baseURL}), nil
}
