package requestpane

import (
	"errors"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/owenHochwald/Volt/internal/http"
)

// RequestCopiedMsg reports the outcome of CopyAsCurl ("ctrl+y") - mirrors
// responsepane.ResponseCopiedMsg.
type RequestCopiedMsg struct {
	Err error
}

// copyAsCurl builds a curl command reproducing exactly what would
// actually be sent right now (see http.BuildCurlCommand/resolvedRequest -
// path/query params substituted, server headers/variables merged in,
// absolute URL) and copies it to the system clipboard.
func (m *RequestPane) copyAsCurl() tea.Cmd {
	return copyCurlCommand(http.BuildCurlCommand(m.resolvedRequest()), clipboard.WriteAll)
}

// copyCurlCommand mirrors responsepane's own copyCommand - write is taken
// as a parameter (rather than calling clipboard.WriteAll directly) so a
// test can substitute a stub instead of touching the real system
// clipboard.
func copyCurlCommand(command string, write func(string) error) tea.Cmd {
	return func() tea.Msg {
		if write == nil {
			return RequestCopiedMsg{Err: errors.New("clipboard writer is unavailable")}
		}
		return RequestCopiedMsg{Err: write(command)}
	}
}
