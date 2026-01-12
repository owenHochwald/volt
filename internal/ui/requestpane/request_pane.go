package requestpane

import (
	"time"

	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui"
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

	ParseErrors []string

	HeadersExpanded bool
	BodyExpanded    bool

	RequestInProgress bool

	DB   *storage.SQLiteStorage
	keys keybindings.KeyMap

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
}

// SetHeight sets the height of the request pane
func (m *RequestPane) SetHeight(height int) {
	m.Height = height
}

// GetCurrentMethod returns the currently selected HTTP method
func (m *RequestPane) GetCurrentMethod() string {
	return m.MethodSelector.Current()
}

// ResultMsgCleanup resets the stopwatch and request state after a response
func (m *RequestPane) ResultMsgCleanup() {
	m.Stopwatch.Stop()
	m.Stopwatch = stopwatch.NewWithInterval(10 * time.Millisecond)
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
