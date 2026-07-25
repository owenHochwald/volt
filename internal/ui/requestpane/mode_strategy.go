package requestpane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui"
)

// ModeStrategy defines the interface for different request pane modes (normal, load test, etc.)
type ModeStrategy interface {
	// HandleInput handles keyboard input for this mode
	HandleInput(m *RequestPane, msg tea.KeyPressMsg) (*RequestPane, tea.Cmd)

	// GetFocusManager returns the focus manager for this mode
	GetFocusManager(m *RequestPane) *ui.FocusManager

	// GetFocusManagerWithIndex returns the focus manager for this mode starting at a specific index
	GetFocusManagerWithIndex(m *RequestPane, index int) *ui.FocusManager

	// OnEnter is called when entering this mode
	OnEnter(m *RequestPane)

	// OnExit is called when exiting this mode
	OnExit(m *RequestPane)
}
