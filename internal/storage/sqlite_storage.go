package storage

import (
	"database/sql"
	"embed"
	"encoding/json"
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
	q := `INSERT INTO requests (name, method, url, headers, body) VALUES (?, ?, ?, ?, ?)`

	res, err := s.db.Exec(q, request.Name, request.Method, request.URL, headerString, request.Body)
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
	q := `SELECT id, name, method, url, headers, body FROM requests`
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
		)

		if err := rows.Scan(&id, &name, &method, &url, &headers, &body); err != nil {
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
