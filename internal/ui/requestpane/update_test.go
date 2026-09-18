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

func TestModifiedEnterSubmitsFromAnyRequestField(t *testing.T) {
	tests := []struct {
		name string
		mod  tea.KeyMod
	}{
		{name: "control", mod: tea.ModCtrl},
		{name: "alt", mod: tea.ModAlt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestRequestPane(t)
			pane.SetFocused(true)
			pane.FocusManager.Next()
			pane.URLInput.SetValue("https://example.com")

			updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tt.mod))

			if !updated.RequestInProgress {
				t.Fatal("modified enter did not start a request")
			}
			if cmd == nil {
				t.Fatal("modified enter did not produce a submit command")
			}
		})
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

func TestInvalidHeadersBlockSubmissionAndSurfaceError(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.URLInput.SetValue("https://example.com")
	pane.Headers.SetValue("missing delimiter")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tea.ModCtrl))

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
}

func TestInvalidLoadTestConfigBlocksStartAndSurfacesError(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.toggleLoadTestMode()
	pane.URLInput.SetValue("https://example.com")
	pane.LoadTestConcurrency.SetValue("not-a-number")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tea.ModCtrl))

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
