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

// Verbose controls whether goose's migration logging (e.g. "goose: no
// migrations to run") is shown. Off by default, so it doesn't clutter the
// TUI's terminal (those lines print before/after the alt screen and are
// otherwise the only thing left in scrollback once the program exits) or a
// CLI subcommand's plain output; set from main before opening any storage,
// via the -v/--debug flag.
var Verbose bool

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
	if !Verbose {
		goose.SetLogger(goose.NopLogger())
	}

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
	q := `INSERT INTO requests (name, method, url, headers, body, group_id, server_id) VALUES (?, ?, ?, ?, ?, ?, ?)`

	// database/sql's default converter turns a nil *int64 into SQL NULL and
	// dereferences a non-nil one, so request.GroupID/ServerID can be passed
	// as-is.
	res, err := s.db.Exec(q, request.Name, request.Method, request.URL, headerString, request.Body, request.GroupID, request.ServerID)
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
	q := `SELECT id, name, method, url, headers, body, group_id, server_id FROM requests`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []http.Request

	for rows.Next() {
		var (
			id       int64
			name     string
			method   string
			url      string
			headers  string
			body     string
			groupID  sql.NullInt64
			serverID sql.NullInt64
		)

		if err := rows.Scan(&id, &name, &method, &url, &headers, &body, &groupID, &serverID); err != nil {
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
		if serverID.Valid {
			sid := serverID.Int64
			request.ServerID = &sid
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
	headerString, err := serializeHeaders(server.Headers)
	if err != nil {
		return err
	}
	q := `INSERT INTO servers (name, base_url, headers) VALUES (?, ?, ?)`
	res, err := s.db.Exec(q, server.Name, server.BaseURL, headerString)
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
	q := `SELECT id, name, base_url, headers FROM servers WHERE name = ?`
	return scanServer(s.db.QueryRow(q, name))
}

func (s *SQLiteStorage) GetServerByID(id int64) (*http.Server, error) {
	q := `SELECT id, name, base_url, headers FROM servers WHERE id = ?`
	return scanServer(s.db.QueryRow(q, id))
}

// scanServer reads one servers row (id, name, base_url, headers) from a
// *sql.Row, shared by GetServerByName/GetServerByID so the headers
// deserialization only lives in one place.
func scanServer(row *sql.Row) (*http.Server, error) {
	var server http.Server
	var headerString string
	err := row.Scan(&server.ID, &server.Name, &server.BaseURL, &headerString)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	headers, err := deserializeHeaders(headerString)
	if err != nil {
		return nil, err
	}
	server.Headers = headers
	return &server, nil
}

func (s *SQLiteStorage) ListServers() ([]http.Server, error) {
	q := `SELECT id, name, base_url, headers FROM servers ORDER BY name`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []http.Server
	for rows.Next() {
		var server http.Server
		var headerString string
		if err := rows.Scan(&server.ID, &server.Name, &server.BaseURL, &headerString); err != nil {
			return nil, err
		}
		headers, err := deserializeHeaders(headerString)
		if err != nil {
			return nil, err
		}
		server.Headers = headers
		servers = append(servers, server)
	}
	return servers, nil
}

// UpdateServerHeaders replaces the named server's default headers - the
// ones merged into every request that resolves against it (see
// http.Server.MergeHeaders).
func (s *SQLiteStorage) UpdateServerHeaders(name string, headers map[string]string) error {
	headerString, err := serializeHeaders(headers)
	if err != nil {
		return err
	}
	q := `UPDATE servers SET headers = ? WHERE name = ?`
	res, err := s.db.Exec(q, headerString, name)
	if err != nil {
		return err
	}
	return checkRowsAffected(res, "server", name)
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

// ResolveEffectiveServer returns the server a request's relative URL and
// default headers resolve against: the request's own ServerID if set
// (taking priority), otherwise its group's assigned server, or nil if
// neither applies (or the assigned server no longer exists).
func (s *SQLiteStorage) ResolveEffectiveServer(request *http.Request) (*http.Server, error) {
	if request.ServerID != nil {
		return s.GetServerByID(*request.ServerID)
	}
	if request.GroupID == nil {
		return nil, nil
	}

	q := `SELECT s.id, s.name, s.base_url, s.headers FROM request_groups g JOIN servers s ON s.id = g.server_id WHERE g.id = ?`
	server, err := scanServer(s.db.QueryRow(q, *request.GroupID))
	if err != nil {
		return nil, err
	}
	return server, nil
}

// ResolveEffectiveURL returns request.URL unchanged if it's already
// absolute or resolves against no server (directly, or via its group).
// Otherwise it joins the URL onto that server's base URL.
func (s *SQLiteStorage) ResolveEffectiveURL(request *http.Request) (string, error) {
	if request.IsAbsoluteURL() {
		return request.URL, nil
	}
	server, err := s.ResolveEffectiveServer(request)
	if err != nil {
		return "", err
	}
	if server == nil {
		return request.URL, nil
	}
	return request.EffectiveURL(server), nil
}
