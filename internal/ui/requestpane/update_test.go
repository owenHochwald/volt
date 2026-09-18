package requestpane

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestBodyInnerContent(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"empty (e.g. an OpenAPI-imported GET/DELETE request)", "", ""},
		{"single char", "{", ""},
		{"empty JSON object", "{}", ""},
		{"populated JSON object", `{"a":"b"}`, `"a":"b"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bodyInnerContent(tt.body); got != tt.want {
				t.Errorf("bodyInnerContent(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

// TestReinitRequestPane_EmptyBodyDoesNotPanic guards against a real crash:
// selecting a saved request whose Body is "" (as every `volt import`-created
// request's is, since GET/DELETE requests carry no body) used to panic with
// "slice bounds out of range [:-1]" in reinitRequestPane.
func TestReinitRequestPane_EmptyBodyDoesNotPanic(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.GET, URL: "http://example.com/pets", Headers: map[string]string{}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reinitRequestPane panicked on an empty body: %v", r)
		}
	}()

	result, cmd := pane.Update(ui.SetRequestPaneRequestMsg{Request: req})
	if cmd != nil {
		t.Errorf("expected no command, got one")
	}
	if result.Body.Value() != "" {
		t.Errorf("expected an empty body textarea, got %q", result.Body.Value())
	}
}

// TestSetRequestPaneRequestMsg_FocusesPastMethodURL is a regression test
// for a real feature request: picking a request from the sidebar already
// tells you its method and URL (that's what you picked it *for* - it's
// right there in the list), so focus should skip past them to whatever's
// actually left to fill in - the first path param here.
func TestSetRequestPaneRequestMsg_FocusesPastMethodURL(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{
		Method:  http.GET,
		URL:     "https://api.example.com/hosts/{id}",
		Headers: map[string]string{},
	}

	result, _ := pane.Update(ui.SetRequestPaneRequestMsg{Request: req})
	if len(result.PathParams) != 1 {
		t.Fatalf("test setup error: expected 1 path param, got %+v", result.PathParams)
	}
	if result.FocusManager.Current() != &result.PathParams[0].Input {
		t.Errorf("expected focus on the first path param, got %T", result.FocusManager.Current())
	}
}

// TestSetRequestPaneRequestMsg_FocusesBodyWhenNoPathParams verifies the
// same skip-past-method-URL behavior lands on the freeform Body textarea
// when there are no path params and no known body schema - whatever
// comes right after method+URL, not specifically a path param.
func TestSetRequestPaneRequestMsg_FocusesBodyWhenNoPathParams(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/hosts",
		Headers: map[string]string{},
	}

	result, _ := pane.Update(ui.SetRequestPaneRequestMsg{Request: req})
	if len(result.PathParams) != 0 {
		t.Fatalf("test setup error: expected no path params, got %+v", result.PathParams)
	}
	if result.FocusManager.Current() != result.Body {
		t.Errorf("expected focus on the freeform Body textarea, got %T", result.FocusManager.Current())
	}
}

// TestResetToBlank_StillFocusesMethodURL contrasts with the sidebar
// behavior above: resetting to a blank request (see resetToBlank, ctrl+r)
// - unlike loading one picked from the sidebar - has no method/URL worth
// skipping past yet, so it must keep defaulting focus to method+URL
// itself (see defaultFocusIndex), not fall through to whatever the
// SetRequestPaneRequestMsg path would land on.
func TestResetToBlank_StillFocusesMethodURL(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.PanelFocused = true

	result, _ := pane.Update(keyPress('r', "", tea.ModCtrl))
	if _, ok := result.FocusManager.Current().(*ui.MethodSelector); !ok {
		t.Errorf("expected reset to still focus MethodSelector, got %T", result.FocusManager.Current())
	}
}

func TestPlainEnterDoesNotSubmitFromEditableField(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.FocusManager.Next()
	pane.URLInput.SetValue("https://example.com")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", 0))

	if updated.RequestInProgress {
		t.Fatal("plain enter started a request")
	}
	if cmd != nil {
		t.Fatal("plain enter produced a submit command")
	}
}

func TestRequestSubmissionKeyBindings(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "meta enter alias", key: keyPress(tea.KeyEnter, "", tea.ModAlt)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestRequestPane(t)
			pane.SetFocused(true)
			pane.FocusManager.Next()
			pane.URLInput.SetValue("https://example.com")

			updated, cmd := pane.Update(tt.key)

			if !updated.RequestInProgress {
				t.Fatal("submit binding did not start a request")
			}
			if cmd == nil {
				t.Fatal("submit binding did not produce a command")
			}
		})
	}
}

func TestEnterActivatesFocusedSendButton(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.URLInput.SetValue("https://example.com")
	// SubmitButton is index 5, the last component in tab order for a
	// fresh, unloaded pane (see TestTabAndShiftTabOnlyNavigateFields's
	// comment for the full field order).
	pane.FocusManager.FocusIndex(5)

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", 0))

	if !updated.RequestInProgress {
		t.Fatal("enter on the send button did not start a request")
	}
	if cmd == nil {
		t.Fatal("enter on the send button did not produce a command")
	}
}

func TestPrintableCommandKeysRemainEditable(t *testing.T) {
	tests := []string{"?", "q", "h", "j", "k", "l"}

	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			pane := newTestRequestPane(t)
			pane.SetFocused(true)
			pane.FocusManager.Next()

			updated, _ := pane.Update(keyPress(rune(text[0]), text, 0))

			if got := updated.URLInput.Value(); got != text {
				t.Fatalf("URL value = %q, want %q", got, text)
			}
		})
	}
}

func TestSetFocusedBlursAndRestoresCurrentControl(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.FocusManager.Next()
	if !pane.URLInput.Focused() {
		t.Fatal("URL input is not focused before leaving panel")
	}

	pane.SetFocused(false)
	if pane.URLInput.Focused() {
		t.Fatal("URL input remained focused after leaving panel")
	}

	pane.SetFocused(true)
	if !pane.URLInput.Focused() {
		t.Fatal("URL input focus was not restored on panel entry")
	}
}

// TestTabAndShiftTabOnlyNavigateFields exercises tab/shift-tab's generic
// advance/wrap behavior. Field indices below match NormalMode.components'
// tab order for a fresh, unloaded pane (no ServerSelector/PathParams/
// BodyParams - see newTestRequestPane): Name(0), Method(1), URL(2),
// Body(3), Headers(4), Submit(5) - not main's fixed Method/URL/Name/
// Headers/Body/Submit layout, since our field set is dynamic.
func TestTabAndShiftTabOnlyNavigateFields(t *testing.T) {
	const (
		fieldName = iota
		fieldMethod
		fieldURL
		fieldSubmit = 5
	)
	tests := []struct {
		name  string
		start int
		key   tea.KeyPressMsg
		want  int
	}{
		{
			name:  "tab advances",
			start: fieldMethod,
			key:   keyPress(tea.KeyTab, "", 0),
			want:  fieldURL,
		},
		{
			name:  "shift tab moves backward",
			start: fieldURL,
			key:   keyPress(tea.KeyTab, "", tea.ModShift),
			want:  fieldMethod,
		},
		{
			name:  "shift tab wraps",
			start: fieldName,
			key:   keyPress(tea.KeyTab, "", tea.ModShift),
			want:  fieldSubmit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestRequestPane(t)
			pane.FocusManager.Current().Blur()
			pane.FocusManager = pane.currentMode.GetFocusManagerWithIndex(&pane, tt.start)
			pane.SetFocused(true)

			updated, _ := pane.Update(tt.key)

			if got := updated.FocusManager.CurrentIndex(); got != tt.want {
				t.Fatalf("focus = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestArrowKeysDoNotChangeFocusedField(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	// FocusManager starts on MethodSelector (see defaultFocusIndex) - one
	// Next() reaches URL.
	pane.FocusManager.Next()
	pane.URLInput.SetValue("https://example.com")
	pane.URLInput.CursorEnd()

	wantIndex := pane.FocusManager.CurrentIndex()
	updated, _ := pane.Update(keyPress(tea.KeyLeft, "", 0))

	if got := updated.FocusManager.CurrentIndex(); got != wantIndex {
		t.Fatalf("left arrow changed focus to index %d, want %d", got, wantIndex)
	}
	if got, want := updated.URLInput.Position(), len("https://example.com")-1; got != want {
		t.Fatalf("left arrow cursor position = %d, want %d", got, want)
	}
}

func TestInvalidHeadersBlockSubmissionAndSurfaceError(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.URLInput.SetValue("https://example.com")
	pane.Headers.SetValue("missing delimiter")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tea.ModAlt))

	if updated.RequestInProgress {
		t.Fatal("invalid headers started a request")
	}
	if cmd == nil {
		t.Fatal("invalid headers did not produce a notification")
	}
	msg, ok := cmd().(ui.NotificationMsg)
	if !ok {
		t.Fatalf("message type = %T, want ui.NotificationMsg", cmd())
	}
	if !strings.Contains(strings.ToLower(msg.Notification.Text), "header") {
		t.Fatalf("notification = %q, want header error", msg.Notification.Text)
	}
	if !strings.Contains(strings.ToLower(msg.Notification.Hint), "one header-name") {
		t.Fatalf("notification hint = %q, want header recovery guidance", msg.Notification.Hint)
	}
}

func TestInvalidLoadTestConfigBlocksStartAndSurfacesError(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.toggleLoadTestMode()
	pane.URLInput.SetValue("https://example.com")
	pane.LoadTestConcurrency.SetValue("not-a-number")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tea.ModAlt))

	if updated.RequestInProgress {
		t.Fatal("invalid load-test config remained in progress")
	}
	if cmd == nil {
		t.Fatal("invalid load-test config did not produce a notification")
	}
	msg, ok := cmd().(ui.NotificationMsg)
	if !ok {
		t.Fatalf("message type = %T, want ui.NotificationMsg", cmd())
	}
	if !strings.Contains(strings.ToLower(msg.Notification.Text), "concurrency") {
		t.Fatalf("notification = %q, want concurrency error", msg.Notification.Text)
	}
	if !strings.Contains(msg.Notification.Hint, "?") {
		t.Fatalf("notification hint = %q, want keybinding guidance", msg.Notification.Hint)
	}
}

func TestInvalidURLOffersLikelyCorrection(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.URLInput.SetValue("example.com/health")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tea.ModAlt))

	if updated.RequestInProgress {
		t.Fatal("invalid URL started a request")
	}
	msg, ok := cmd().(ui.NotificationMsg)
	if !ok {
		t.Fatalf("message type = %T, want ui.NotificationMsg", cmd())
	}
	if !strings.Contains(msg.Notification.Hint, "https://example.com/health") {
		t.Fatalf("notification hint = %q, want URL correction", msg.Notification.Hint)
	}
}

func newTestRequestPane(t *testing.T) RequestPane {
	t.Helper()

	db, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "volt.db"))
	if err != nil {
		t.Fatalf("create test storage: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test storage: %v", err)
		}
	})
	return SetupRequestPane(db, keybindings.DefaultKeyMap())
}

func keyPress(code rune, text string, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{
		Code: code,
		Text: text,
		Mod:  mod,
	})
}
