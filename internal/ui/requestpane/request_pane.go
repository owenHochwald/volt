package requestpane

import (
	"time"

	"charm.land/bubbles/v2/stopwatch"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// FieldIndex represents the index of a focusable field in the request pane
type FieldIndex int

const (
	FieldMethodSelector FieldIndex = iota
	FieldURL
	FieldName
	FieldHeaders
	FieldBody
	FieldSubmitButton
)

// Load test mode field indices (extend from base fields)
const (
	FieldLTConcurrency FieldIndex = iota + 5
	FieldLTTotalReqs
	FieldLTQPS
	FieldLTTimeout
	FieldLTSubmit
)

// RequestPane is the main component for handling HTTP request input
type RequestPane struct {
	Client *http.Client

	Stopwatch stopwatch.Model
	Quitting  bool

	PanelFocused bool

	FocusManager *ui.FocusManager

	MethodSelector *ui.MethodSelector
	URLInput       *textinput.Model
	NameInput      *textinput.Model
	Headers        *textarea.Model
	Body           *textarea.Model
	SubmitButton   *ui.SubmitButton

	Request *http.Request

	Height int
	Width  int

	ParseErrors []string

	HeadersExpanded bool
	BodyExpanded    bool

	RequestInProgress bool

	DB     *storage.SQLiteStorage
	keys   keybindings.KeyMap
	styles design.Styles

	// Load test mode fields
	LoadTestMode        bool
	currentMode         ModeStrategy
	LoadTestConcurrency *textinput.Model
	LoadTestTotalReqs   *textinput.Model
	LoadTestQPS         *textinput.Model
	LoadTestTimeout     *textinput.Model
}

// Init initializes the request pane
func (m RequestPane) Init() tea.Cmd {
	return textinput.Blink
}

// SetFocused sets the panel focus state
func (m *RequestPane) SetFocused(focused bool) {
	m.PanelFocused = focused
	if m.FocusManager == nil {
		return
	}
	if focused {
		m.FocusManager.Current().Focus()
		return
	}
	m.FocusManager.Current().Blur()
}

// SetHeight sets the height of the request pane
func (m *RequestPane) SetHeight(height int) {
	m.SetSize(m.Width, height)
}

// SetSize updates all editor dimensions from the available panel content.
func (m *RequestPane) SetSize(width, height int) {
	m.Width = max(width, 1)
	m.Height = max(height, 1)

	m.URLInput.SetWidth(max(m.Width-12, 10))
	m.NameInput.SetWidth(max(m.Width-8, 10))
	m.Headers.SetWidth(max(m.Width-10, 10))
	m.Body.SetWidth(max(m.Width-10, 10))

	editorHeight := clampDimension((m.Height-10)/2, 2, 5)
	if m.LoadTestMode {
		editorHeight = clampDimension((m.Height-16)/2, 1, 3)
	}
	m.Headers.SetHeight(editorHeight)
	m.Body.SetHeight(editorHeight)
}

// IsEditing reports whether the focused control accepts printable text.
func (m RequestPane) IsEditing() bool {
	if m.FocusManager == nil {
		return false
	}
	index := FieldIndex(m.FocusManager.CurrentIndex())
	if index == FieldMethodSelector {
		return false
	}
	if m.LoadTestMode {
		return index != FieldLTSubmit
	}
	return index != FieldSubmitButton
}

func clampDimension(value, minimum, maximum int) int {
	return min(max(value, minimum), maximum)
}

// GetCurrentMethod returns the currently selected HTTP method
func (m *RequestPane) GetCurrentMethod() string {
	return m.MethodSelector.Current()
}

// ResultMsgCleanup resets the stopwatch and request state after a response
func (m *RequestPane) ResultMsgCleanup() {
	m.Stopwatch.Stop()
	m.Stopwatch = stopwatch.New(stopwatch.WithInterval(10 * time.Millisecond))
	m.RequestInProgress = false
}

// ExitLoadTestMode exits load test mode and resets state
func (m *RequestPane) ExitLoadTestMode() {
	if m.FocusManager != nil {
		m.FocusManager.Current().Blur()
	}

	m.LoadTestMode = false
	m.RequestInProgress = false

	m.currentMode = &NormalMode{}
	m.FocusManager = m.currentMode.GetFocusManager(m)
}
