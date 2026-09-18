package requestpane

import (
	"time"

	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// RequestPane is the main component for handling HTTP request input
type RequestPane struct {
	Client *http.Client

	Stopwatch stopwatch.Model
	Quitting  bool

	PanelFocused bool

	FocusManager *ui.FocusManager

	// FieldsViewport scrolls everything from the method/URL line through
	// Headers/Body(Params) (and, in load-test mode, the load-test config
	// fields) - i.e. everything except the pinned Send button/help line
	// below it. Without this, a request with many body params (a wide
	// OpenAPI schema) could render taller than its box, which lipgloss's
	// Height() doesn't clip (only pads), corrupting the whole layout - see
	// View() and SetHeight/SetWidth for how it's kept sized to match.
	FieldsViewport viewport.Model

	// ServerSelector picks which configured server (if any) this request
	// resolves against directly - independent of any group assignment (see
	// http.Request.ServerID) - cycled the same way as the method, and
	// rebuilt from Servers whenever the pane loads a request (see
	// rebuildServerSelector). It's the first field, ahead of even the
	// method/URL, since it changes where every relative URL below it goes.
	ServerSelector *EnumSelector
	Servers        []http.Server

	MethodSelector *ui.MethodSelector
	URLInput       *textinput.Model
	// PathParams holds one editable field per {name} placeholder in the
	// URL (e.g. "/cluster/{cluster_id}/repository"), kept in sync with the
	// URL by syncPathParams. They sit between the URL and Name fields.
	PathParams []Param
	NameInput  *textinput.Model
	Headers    *textarea.Model
	// BodyParams holds one editable field per top-level property of a JSON
	// request body (populated by `volt import` from the OpenAPI
	// requestBody schema), kept in sync with Body by syncBodyParams. When
	// non-empty, they're shown instead of the freeform Body textarea.
	BodyParams []Param
	// ArrayParams holds one repeatable item-group per top-level "array of
	// objects" property (its schema's "items" was a $ref to an object),
	// also kept in sync with Body by syncBodyParams - a "simple" array
	// (items with no object structure) stays a plain BodyParams text field
	// instead.
	ArrayParams  []ArrayParam
	Body         *textarea.Model
	SubmitButton *ui.SubmitButton

	Request *http.Request

	Height int
	Width  int

	ParseErrors []string

	// ShowRequiredErrors switches on the required-empty half of the inline
	// per-field error display (see fieldError/renderParamSection) - set once
	// a send has actually been blocked on a still-empty required field (see
	// handleSubmit in mode_normal.go/mode_loadtest.go), and cleared again by
	// reinitRequestPane, so a freshly loaded/reset/cleared request never
	// starts out with fields already flagged red before anything's been
	// attempted.
	ShowRequiredErrors bool

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

// approxFooterLines is a rough, constant estimate of the pinned footer's
// height (blank separator + Send button + stopwatch line + help hint),
// used only to give FieldsViewport a sane Height for Update() to compute
// pgup/pgdown scroll amounts against. View() overrides this every render
// with the footer's *actual* rendered height (which varies with
// RequestInProgress/LoadTestMode), so this estimate only needs to be
// close, not exact - it never affects what's actually drawn.
const approxFooterLines = 4

// SetHeight sets the height of the request pane and gives FieldsViewport a
// matching (approximate - see approxFooterLines) height, so that
// Update() - e.g. a pgup/pgdown keypress - has a usable viewport to scroll
// even before the next View() call refines it. Without this,
// FieldsViewport.Height stays at its zero value from Update()'s
// perspective (View() has a value receiver, so its own adjustments to
// FieldsViewport never persist back - see View()), and page-scrolling by a
// zero height is a no-op.
func (m *RequestPane) SetHeight(height int) {
	m.Height = height
	m.FieldsViewport.Height = max(height-approxFooterLines, 1)
}

// SetWidth sets the panel's available content width and resizes its text
// inputs to fit it, so long URLs/names/headers/bodies use the space
// available on a wide terminal instead of staying clipped to a fixed
// width, and don't overflow a narrow one.
func (m *RequestPane) SetWidth(width int) {
	m.Width = width
	m.FieldsViewport.Width = width

	const (
		minInputWidth   = 20
		methodBoxWidth  = 9 // rendered "┌─────┐ " prefix before the URL input
		nameLabelWidth  = 5 // "Name " prefix before the name input
		fieldLabelWidth = 8 // "Headers "/"Body    " prefix before the textareas
		safetyMargin    = 2
	)

	m.URLInput.Width = max(width-methodBoxWidth-safetyMargin, minInputWidth)
	m.NameInput.Width = max(width-nameLabelWidth-safetyMargin, minInputWidth)

	textAreaWidth := max(width-fieldLabelWidth-safetyMargin, minInputWidth)
	m.Headers.SetWidth(textAreaWidth)
	m.Body.SetWidth(textAreaWidth)

	// maxParamInputWidth caps a path/body param's field instead of
	// stretching it to fill the pane the way URL/Name do - a param value is
	// typically short, and stretching it left no room for the inline
	// validation comment renderParamSection (view.go) shows to its right
	// for an invalid value.
	const maxParamInputWidth = 40
	resizeParams := func(params []Param) {
		for i := range params {
			labelWidth := len(params[i].Name) + 1 // "name "
			params[i].Input.Width = min(max(width-labelWidth-safetyMargin, minInputWidth), maxParamInputWidth)
		}
	}
	resizeParams(m.PathParams)
	resizeParams(m.BodyParams)
	for i := range m.ArrayParams {
		for j := range m.ArrayParams[i].Items {
			resizeParams(m.ArrayParams[i].Items[j].Fields)
		}
	}
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

// MarkSaved records saved.ID onto this pane's request, right after a
// successful ctrl+s (see ui.RequestSavedMsg, dispatched by app.Model
// regardless of which panel is currently focused - a save can still be in
// flight after the user has already moved on). Without this,
// m.Request.ID would stay at its pre-save value (0 for a brand new
// request), so the *next* ctrl+s would always insert another row (see
// storage.SQLiteStorage.Save's ID != 0 + unchanged-Name check) instead of
// updating the one just created.
//
// Guarded on Method/URL/Name still matching saved: if the pane moved on
// to a different request while the save was in flight, saved no longer
// describes what's currently loaded, so stamping its ID here would
// silently misattribute it to the wrong request.
func (m *RequestPane) MarkSaved(saved *http.Request) {
	if m.Request == nil || saved == nil {
		return
	}
	if m.Request.Method == saved.Method && m.Request.URL == saved.URL && m.Request.Name == saved.Name {
		m.Request.ID = saved.ID
	}
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
