package importpane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/cli"
	"github.com/owenHochwald/Volt/internal/storage"
)

// ImportResultMsg carries the outcome of a background OpenAPI import.
type ImportResultMsg struct {
	Summary *cli.ImportSummary
	Err     error
}

// RunImportCmd runs the same import logic as `volt import` in the
// background (it touches disk and the database, so it's kept off the UI
// goroutine even though local SQLite calls are normally fast).
func RunImportCmd(specPath, baseURL, group string, force bool, db *storage.SQLiteStorage) tea.Cmd {
	return func() tea.Msg {
		summary, err := cli.RunImport(specPath, baseURL, group, force, db)
		return ImportResultMsg{Summary: summary, Err: err}
	}
}
