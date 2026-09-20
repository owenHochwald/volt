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

// serializeFieldSchema/deserializeFieldSchema mirror
// serializeHeaders/deserializeHeaders for http.Request.PathParamSchema/
// QueryParamSchema, except a nil schema round-trips as an empty string
// rather than the literal 4-byte "null" - matching path_param_schema's/
// query_param_schema's migration default (empty string, not SQL NULL)
// for every row saved before each column existed, so Load doesn't need
// to special-case old rows differently from new ones with no path/query
// params.
//
// A non-nil but *empty* schema (map[string]http.BodyFieldSchema{}, as
// opposed to a nil one) is deliberately serialized as a real "{}"
// instead - requestpane.IsSchemaRequest uses PathParamSchema != nil after
// loading a row back as its marker for "this request's schema was
// inspected at import time" (see internal/cli/openapi.go's
// pathParamFields, which now always returns a non-nil map, empty or
// not), and collapsing that distinction to the same "" a hand-typed
// request's genuinely-nil schema produces would make the two
// indistinguishable again once loaded back from the database.
func serializeFieldSchema(schema map[string]http.BodyFieldSchema) (string, error) {
	if schema == nil {
		return "", nil
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func deserializeFieldSchema(jsonStr string) (map[string]http.BodyFieldSchema, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var schema map[string]http.BodyFieldSchema
	if err := json.Unmarshal([]byte(jsonStr), &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

// serializeParamValues/deserializeParamValues mirror
// serializeFieldSchema/deserializeFieldSchema for
// http.Request.ParamValues - an empty/zero value round-trips as an empty
// string (the param_values column's migration default), not the literal
// 4-byte "null".
func serializeParamValues(values http.ParamValues) (string, error) {
	if len(values.PathParamValues) == 0 && len(values.QueryParamValues) == 0 && len(values.BodyParamValues) == 0 && len(values.ArrayParamValues) == 0 {
		return "", nil
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func deserializeParamValues(jsonStr string) (http.ParamValues, error) {
	if jsonStr == "" {
		return http.ParamValues{}, nil
	}
	var values http.ParamValues
	if err := json.Unmarshal([]byte(jsonStr), &values); err != nil {
		return http.ParamValues{}, err
	}
	return values, nil
}

func runMigrations(db *sql.DB) error {
	if !Verbose {
		goose.SetLogger(goose.NopLogger())
	}

	// Set the embedded filesystem for goose
	goose.SetBaseFS(embedMigrations)
	goose.SetLogger(goose.NopLogger())

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

// Save upserts request: an already-saved request (request.ID != 0) whose
// Name hasn't changed since it was last saved is updated in place, so
// resaving an edited request (see requestpane.RequestPane's ctrl+s
// handling) refines the same row instead of piling up a duplicate every
// time. A blank/never-saved request (ID == 0) or one whose Name *has*
// changed is treated as a deliberately new request - "renaming and
// resaving" is how you save a variant alongside the original, not how you
// rename it in place (there's no in-place rename through this path).
func (s *SQLiteStorage) Save(request *http.Request) error {
	headerString, err := serializeHeaders(request.Headers)
	if err != nil {
		return err
	}
	pathParamSchemaString, err := serializeFieldSchema(request.PathParamSchema)
	if err != nil {
		return err
	}
	queryParamSchemaString, err := serializeFieldSchema(request.QueryParamSchema)
	if err != nil {
		return err
	}
	paramValuesString, err := serializeParamValues(request.ParamValues)
	if err != nil {
		return err
	}

	if request.ID != 0 {
		var existingName string
		err := s.db.QueryRow(`SELECT name FROM requests WHERE id = ?`, request.ID).Scan(&existingName)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil && existingName == request.Name {
			q := `UPDATE requests SET name = ?, method = ?, url = ?, headers = ?, body = ?, group_id = ?, server_id = ?, path_param_schema = ?, query_param_schema = ?, param_values = ? WHERE id = ?`
			_, err = s.db.Exec(q, request.Name, request.Method, request.URL, headerString, request.Body, request.GroupID, request.ServerID, pathParamSchemaString, queryParamSchemaString, paramValuesString, request.ID)
			return err
		}
		// Row no longer exists, or the name changed - fall through to
		// insert a new row below.
	}

	q := `INSERT INTO requests (name, method, url, headers, body, group_id, server_id, path_param_schema, query_param_schema, param_values) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// database/sql's default converter turns a nil *int64 into SQL NULL and
	// dereferences a non-nil one, so request.GroupID/ServerID can be passed
	// as-is.
	res, err := s.db.Exec(q, request.Name, request.Method, request.URL, headerString, request.Body, request.GroupID, request.ServerID, pathParamSchemaString, queryParamSchemaString, paramValuesString)
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

// UpdateRequestSchema overwrites an existing request's name, headers,
// body, and path/query param schemas - used by `volt import -force` to
// refresh a previously imported request from a changed OpenAPI spec
// without touching its group/server association or its identity (method/
// url, which is what matched it to the existing row in the first place -
// see cli.dedupKey). Deliberately leaves group_id/server_id/method/url
// alone: those are either the match key itself or a user-controlled
// association unrelated to the spec's content.
func (s *SQLiteStorage) UpdateRequestSchema(id int64, name string, headers map[string]string, body string, pathParamSchema, queryParamSchema map[string]http.BodyFieldSchema) error {
	headerString, err := serializeHeaders(headers)
	if err != nil {
		return err
	}
	pathParamSchemaString, err := serializeFieldSchema(pathParamSchema)
	if err != nil {
		return err
	}
	queryParamSchemaString, err := serializeFieldSchema(queryParamSchema)
	if err != nil {
		return err
	}
	q := `UPDATE requests SET name = ?, headers = ?, body = ?, path_param_schema = ?, query_param_schema = ? WHERE id = ?`
	_, err = s.db.Exec(q, name, headerString, body, pathParamSchemaString, queryParamSchemaString, id)
	return err
}

func (s *SQLiteStorage) Load() ([]http.Request, error) {
	q := `SELECT id, name, method, url, headers, body, group_id, server_id, path_param_schema, query_param_schema, param_values FROM requests`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []http.Request

	for rows.Next() {
		var (
			id               int64
			name             string
			method           string
			url              string
			headers          string
			body             string
			groupID          sql.NullInt64
			serverID         sql.NullInt64
			pathParamSchema  string
			queryParamSchema string
			paramValues      string
		)

		if err := rows.Scan(&id, &name, &method, &url, &headers, &body, &groupID, &serverID, &pathParamSchema, &queryParamSchema, &paramValues); err != nil {
			return nil, err

		}
		headersMap, err := deserializeHeaders(headers)
		if err != nil {
			return nil, err
		}
		pathParamSchemaMap, err := deserializeFieldSchema(pathParamSchema)
		if err != nil {
			return nil, err
		}
		queryParamSchemaMap, err := deserializeFieldSchema(queryParamSchema)
		if err != nil {
			return nil, err
		}
		paramValuesStruct, err := deserializeParamValues(paramValues)
		if err != nil {
			return nil, err
		}
		request := http.Request{
			ID:               id,
			Name:             name,
			Method:           method,
			URL:              url,
			Headers:          headersMap,
			Body:             body,
			PathParamSchema:  pathParamSchemaMap,
			QueryParamSchema: queryParamSchemaMap,
			ParamValues:      paramValuesStruct,
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

// lastSentRequestKey is the app_state row SaveLastSentRequest/
// LoadLastSentRequest read and write - a single well-known key rather
// than a dedicated table, since this is the only piece of app-level
// (non-request) state Volt persists today.
const lastSentRequestKey = "last_sent_request"

// SaveLastSentRequest persists request (schema/template AND its current
// ParamValues - see http.Request.ParamValues) as the request to restore on
// the next app start (see LoadLastSentRequest) - overwriting whatever was
// saved before, since only the *last* send matters. Called only once a
// send actually goes out (see requestpane's handleSubmit), never for a
// request that's merely being edited.
func (s *SQLiteStorage) SaveLastSentRequest(request *http.Request) error {
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	q := `INSERT INTO app_state (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	_, err = s.db.Exec(q, lastSentRequestKey, string(data))
	return err
}

// LoadLastSentRequest returns the request saved by SaveLastSentRequest, or
// (nil, nil) if nothing's been sent yet (a fresh database, or one from
// before this existed) - not an error, since that's the ordinary starting
// state.
func (s *SQLiteStorage) LoadLastSentRequest() (*http.Request, error) {
	q := `SELECT value FROM app_state WHERE key = ?`
	var value string
	if err := s.db.QueryRow(q, lastSentRequestKey).Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var request http.Request
	if err := json.Unmarshal([]byte(value), &request); err != nil {
		return nil, err
	}
	return &request, nil
}

// ClearLastSentRequest deletes whatever SaveLastSentRequest last saved, so
// LoadLastSentRequest goes back to returning (nil, nil) - used when the
// request pane is reset to blank (see requestpane's resetToBlank): without
// this, resetting the *pane* wouldn't reset what the *next* app start
// restores, and the discarded request would silently reappear on restart.
// A no-op, not an error, if nothing was saved to begin with.
func (s *SQLiteStorage) ClearLastSentRequest() error {
	_, err := s.db.Exec(`DELETE FROM app_state WHERE key = ?`, lastSentRequestKey)
	return err
}

// lastGroupFilterKey is the app_state row SaveLastGroupFilter/
// LoadLastGroupFilter read and write - the sidebar's group filter (see
// ui.SidebarPane.groupOptions), so the next app start reopens on whatever
// group was last being worked in instead of always resetting to "All
// groups".
const lastGroupFilterKey = "last_group_filter"

// SaveLastGroupFilter persists name (a group name, or "" for "All
// groups") as the sidebar's group filter to restore on the next app
// start - overwriting whatever was saved before, since only the most
// recent selection matters. Called every time the filter changes (see
// ui.SidebarPane.cycleGroupFilter), including back to "" - an empty
// string is itself a meaningful, deliberately saved choice ("All groups"
// was last selected), not the absence of one.
func (s *SQLiteStorage) SaveLastGroupFilter(name string) error {
	q := `INSERT INTO app_state (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	_, err := s.db.Exec(q, lastGroupFilterKey, name)
	return err
}

// LoadLastGroupFilter returns the group name saved by SaveLastGroupFilter,
// or ("", nil) if nothing's been saved yet (a fresh database, or one from
// before this existed, or the last selection genuinely was "All groups") -
// not an error, since that's the ordinary starting state.
func (s *SQLiteStorage) LoadLastGroupFilter() (string, error) {
	q := `SELECT value FROM app_state WHERE key = ?`
	var value string
	if err := s.db.QueryRow(q, lastGroupFilterKey).Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// lastWSURLKey is the app_state row SaveLastWSURL/LoadLastWSURL read and
// write - the URL of the last WebSocket connection that actually
// succeeded (see wspane's ConnectResultMsg handling), restored as the
// WebSocket pane's URL field on the next app start since a session
// typically keeps reconnecting to the same address.
const lastWSURLKey = "last_ws_url"

// SaveLastWSURL persists url as the WebSocket URL to restore on the next
// app start - overwriting whatever was saved before, since only the most
// recently successful address matters. Called only after a connection
// actually succeeds, never on a failed attempt or a mid-edit URL.
func (s *SQLiteStorage) SaveLastWSURL(url string) error {
	q := `INSERT INTO app_state (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	_, err := s.db.Exec(q, lastWSURLKey, url)
	return err
}

// LoadLastWSURL returns the URL saved by SaveLastWSURL, or ("", nil) if
// nothing's been saved yet (a fresh database, or one from before this
// existed, or no WebSocket connection has ever succeeded) - not an error,
// since that's the ordinary starting state.
func (s *SQLiteStorage) LoadLastWSURL() (string, error) {
	q := `SELECT value FROM app_state WHERE key = ?`
	var value string
	if err := s.db.QueryRow(q, lastWSURLKey).Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
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
	variableString, err := serializeHeaders(server.Variables)
	if err != nil {
		return err
	}
	q := `INSERT INTO servers (name, base_url, headers, variables) VALUES (?, ?, ?, ?)`
	res, err := s.db.Exec(q, server.Name, server.BaseURL, headerString, variableString)
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
	q := `SELECT id, name, base_url, headers, variables FROM servers WHERE name = ?`
	return scanServer(s.db.QueryRow(q, name))
}

func (s *SQLiteStorage) GetServerByID(id int64) (*http.Server, error) {
	q := `SELECT id, name, base_url, headers, variables FROM servers WHERE id = ?`
	return scanServer(s.db.QueryRow(q, id))
}

// scanServer reads one servers row (id, name, base_url, headers,
// variables) from a *sql.Row, shared by GetServerByName/GetServerByID so
// the headers/variables deserialization only lives in one place.
func scanServer(row *sql.Row) (*http.Server, error) {
	var server http.Server
	var headerString, variableString string
	err := row.Scan(&server.ID, &server.Name, &server.BaseURL, &headerString, &variableString)
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
	variables, err := deserializeHeaders(variableString)
	if err != nil {
		return nil, err
	}
	server.Headers = headers
	server.Variables = variables
	return &server, nil
}

func (s *SQLiteStorage) ListServers() ([]http.Server, error) {
	q := `SELECT id, name, base_url, headers, variables FROM servers ORDER BY name`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []http.Server
	for rows.Next() {
		var server http.Server
		var headerString, variableString string
		if err := rows.Scan(&server.ID, &server.Name, &server.BaseURL, &headerString, &variableString); err != nil {
			return nil, err
		}
		headers, err := deserializeHeaders(headerString)
		if err != nil {
			return nil, err
		}
		variables, err := deserializeHeaders(variableString)
		if err != nil {
			return nil, err
		}
		server.Headers = headers
		server.Variables = variables
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

// UpdateServerVariables replaces the named server's {{name}} placeholder
// values - the ones substituted into every request that resolves against
// it (see http.SubstituteVariables).
func (s *SQLiteStorage) UpdateServerVariables(name string, variables map[string]string) error {
	variableString, err := serializeHeaders(variables)
	if err != nil {
		return err
	}
	q := `UPDATE servers SET variables = ? WHERE name = ?`
	res, err := s.db.Exec(q, variableString, name)
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

// DeleteServer deletes the named server along with its group membership
// rows (see group_servers) - SQLite foreign keys aren't enforced here (see
// DeleteGroup's own comment), so this is done explicitly rather than
// relying on the migration's ON DELETE CASCADE. A group whose *default*
// server (request_groups.server_id) is this one is left pointing at a
// now-nonexistent id; ResolveEffectiveServer's GetServerByID lookup
// already treats that as "no server" via sql.ErrNoRows, so this doesn't
// need its own NULL-out - same as before group_servers existed.
func (s *SQLiteStorage) DeleteServer(name string) error {
	server, err := s.GetServerByName(name)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", name)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM group_servers WHERE server_id = ?`, server.ID); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM servers WHERE id = ?`, server.ID)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(res, "server", name); err != nil {
		return err
	}
	return tx.Commit()
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

// GetGroupByID is GetGroupByName's id-keyed counterpart - used wherever
// only a request's numeric GroupID is on hand (e.g.
// requestpane.rebuildServerSelector), not its name.
func (s *SQLiteStorage) GetGroupByID(id int64) (*http.Group, error) {
	q := `SELECT id, name, server_id FROM request_groups WHERE id = ?`
	var group http.Group
	var serverID sql.NullInt64
	err := s.db.QueryRow(q, id).Scan(&group.ID, &group.Name, &serverID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if serverID.Valid {
		sid := serverID.Int64
		group.ServerID = &sid
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

// SetGroupServer sets groupName's *default* server - the one its
// requests' relative URLs resolve against when they don't pick one of
// their own (see ResolveEffectiveServer) - and, since a group's default
// server obviously has to be usable by that group, also adds it to the
// group's allowed set (see AddGroupServer) if it isn't already a member.
func (s *SQLiteStorage) SetGroupServer(groupName, serverName string) error {
	group, err := s.GetGroupByName(groupName)
	if err != nil {
		return err
	}
	if group == nil {
		return fmt.Errorf("group not found: %s", groupName)
	}
	server, err := s.GetServerByName(serverName)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", serverName)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE request_groups SET server_id = ? WHERE id = ?`, server.ID, group.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO group_servers (group_id, server_id) VALUES (?, ?)`, group.ID, server.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// AddGroupServer adds server to groupName's allowed set of servers - the
// only servers a request belonging to that group can pick via its own
// ServerSelector (see requestpane.rebuildServerSelector /
// ListGroupServers), so a request never ends up resolving against a
// server that has nothing to do with its group's API. A no-op (not an
// error) if server is already a member. Does not change the group's
// *default* server (see SetGroupServer) - a group can allow several
// servers (e.g. dev/staging/prod) while defaulting to just one of them.
func (s *SQLiteStorage) AddGroupServer(groupName, serverName string) error {
	group, err := s.GetGroupByName(groupName)
	if err != nil {
		return err
	}
	if group == nil {
		return fmt.Errorf("group not found: %s", groupName)
	}
	server, err := s.GetServerByName(serverName)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", serverName)
	}
	q := `INSERT OR IGNORE INTO group_servers (group_id, server_id) VALUES (?, ?)`
	_, err = s.db.Exec(q, group.ID, server.ID)
	return err
}

// RemoveGroupServer removes server from groupName's allowed set (see
// AddGroupServer). If server was the group's *default* (see
// SetGroupServer), the default is cleared too - a group's default server
// must always be one of its own allowed servers, so leaving it pointing
// at one that's just been disallowed would be a silent inconsistency.
func (s *SQLiteStorage) RemoveGroupServer(groupName, serverName string) error {
	group, err := s.GetGroupByName(groupName)
	if err != nil {
		return err
	}
	if group == nil {
		return fmt.Errorf("group not found: %s", groupName)
	}
	server, err := s.GetServerByName(serverName)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", serverName)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM group_servers WHERE group_id = ? AND server_id = ?`, group.ID, server.ID)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(res, "server in group", fmt.Sprintf("%s/%s", groupName, serverName)); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE request_groups SET server_id = NULL WHERE id = ? AND server_id = ?`, group.ID, server.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListGroupServers returns the servers allowed for groupID (see
// AddGroupServer), sorted by name - what a request belonging to that
// group's own ServerSelector should offer, instead of every server in the
// database. An empty (non-nil) slice, not an error, when the group allows
// none yet.
func (s *SQLiteStorage) ListGroupServers(groupID int64) ([]http.Server, error) {
	q := `SELECT s.id, s.name, s.base_url, s.headers, s.variables FROM group_servers gs JOIN servers s ON s.id = gs.server_id WHERE gs.group_id = ? ORDER BY s.name`
	rows, err := s.db.Query(q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]http.Server, 0)
	for rows.Next() {
		var server http.Server
		var headerString, variableString string
		if err := rows.Scan(&server.ID, &server.Name, &server.BaseURL, &headerString, &variableString); err != nil {
			return nil, err
		}
		headers, err := deserializeHeaders(headerString)
		if err != nil {
			return nil, err
		}
		variables, err := deserializeHeaders(variableString)
		if err != nil {
			return nil, err
		}
		server.Headers = headers
		server.Variables = variables
		servers = append(servers, server)
	}
	return servers, rows.Err()
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

	if _, err := tx.Exec(`DELETE FROM group_servers WHERE group_id = ?`, group.ID); err != nil {
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

	q := `SELECT s.id, s.name, s.base_url, s.headers, s.variables FROM request_groups g JOIN servers s ON s.id = g.server_id WHERE g.id = ?`
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
