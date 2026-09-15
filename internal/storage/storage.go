package storage

import (
	"os"
	"path/filepath"

	"github.com/owenHochwald/Volt/internal/http"
)

type Storage interface {
	Save(requests *http.Request) error
	Load() ([]http.Request, error)
	Delete(id int64) error
	GetAllURLs() ([]string, error)
}

// DefaultDBPath returns the path to Volt's SQLite database in the user's
// home directory (~/.volt/volt.db), used by both the TUI and CLI commands.
func DefaultDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".volt", "volt.db"), nil
}
