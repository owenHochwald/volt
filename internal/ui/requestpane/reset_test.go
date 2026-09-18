package requestpane

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

// TestRequestPane_ResetRequest_ReturnsToBlankState is a regression test
// for a real feature request: after loading a request and editing its
// fields, there was no way to discard all of it and get back to a fresh,
// blank request short of restarting the app - ctrl+r should reproduce
// exactly the state SetupRequestPane starts on.
func TestRequestPane_ResetRequest_ReturnsToBlankState(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {Type: "string"}})
	req := &http.Request{
		Method: http.POST,
		URL:    "https://api.example.com/hosts/{id}",
		Name:   "getHost",
		Body:   body,
		Headers: map[string]string{
			"X-Env": "prod",
		},
	}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	pane.NameInput.SetValue("edited name")
	pane.URLInput.SetValue("https://api.example.com/hosts/{id}/extra")
	if len(pane.PathParams) != 1 {
		t.Fatalf("expected 1 path param, got %+v", pane.PathParams)
	}
	pane.PathParams[0].Input.SetValue("42")
	if len(pane.BodyParams) != 1 {
		t.Fatalf("expected 1 body param, got %+v", pane.BodyParams)
	}
	pane.BodyParams[0].Input.SetValue("widget-1")
	pane.ParseErrors = []string{"some previous error"}

	result, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: 'r', Mod: tea.ModCtrl}))

	want := http.NewDefaultRequest()
	if result.Request.Name != want.Name {
		t.Errorf("expected Name=%q, got %q", want.Name, result.Request.Name)
	}
	if result.Request.Method != want.Method {
		t.Errorf("expected Method=%q, got %q", want.Method, result.Request.Method)
	}
	if result.Request.URL != want.URL {
		t.Errorf("expected URL=%q, got %q", want.URL, result.Request.URL)
	}
	if result.NameInput.Value() != want.Name {
		t.Errorf("expected the name input to match the default request's name %q, got %q", want.Name, result.NameInput.Value())
	}
	if len(result.PathParams) != 0 {
		t.Errorf("expected no path params (default request's URL has no {placeholders}), got %+v", result.PathParams)
	}
	if len(result.BodyParams) != 0 {
		t.Errorf("expected no body params (default request's body is empty), got %+v", result.BodyParams)
	}
	if len(result.ParseErrors) != 0 {
		t.Errorf("expected ParseErrors to be cleared, got %v", result.ParseErrors)
	}
}

// TestRequestPane_ResetRequest_ClearsPersistedLastSentRequest is a
// regression test for a real feature request: resetting the pane must
// also clear whatever was persisted as "the last sent request" (see
// ui.SaveLastSentRequestCmd/ApplySnapshot), otherwise the discarded
// request would silently come right back on the next app start - the
// pane looking reset is not the same as the *saved* state being reset.
func TestRequestPane_ResetRequest_ClearsPersistedLastSentRequest(t *testing.T) {
	pane, db := setupTestPane(t)
	pane.PanelFocused = true

	if err := db.SaveLastSentRequest(&http.Request{Method: http.GET, URL: "https://api.example.com/x"}); err != nil {
		t.Fatalf("test setup error saving snapshot: %v", err)
	}
	if got, err := db.LoadLastSentRequest(); err != nil || got == nil {
		t.Fatalf("test setup error: expected a saved snapshot, got %+v err=%v", got, err)
	}

	_, cmd := pane.Update(tea.KeyPressMsg(tea.Key{Code: 'r', Mod: tea.ModCtrl}))
	if cmd == nil {
		t.Fatal("expected a non-nil command to clear the persisted snapshot")
	}
	runAllCmds(cmd)

	got, err := db.LoadLastSentRequest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected the persisted snapshot to be cleared after a reset, got %+v", got)
	}
}

// TestRequestPane_ResetRequest_ExitsLoadTestMode verifies a reset while in
// load-test mode also drops back to normal mode - a reset request should
// behave exactly like a fresh app start, which is always normal mode.
func TestRequestPane_ResetRequest_ExitsLoadTestMode(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.PanelFocused = true
	pane.toggleLoadTestMode()
	if !pane.LoadTestMode {
		t.Fatal("test setup error: expected load test mode to be on")
	}

	result, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: 'r', Mod: tea.ModCtrl}))
	if result.LoadTestMode {
		t.Error("expected reset to exit load test mode")
	}
}

// TestRequestPane_DefaultFocus_IsMethodURL is a regression test for a
// real feature request: a freshly set up pane, and a reset one, should
// both start focus on the method+URL line, not the Name field ahead of
// it (see fieldsContent's new order: server, name, method+URL, ...) -
// schemas rarely need their Name touched, so landing there by default
// would just be in the way; shift+tab reaches it in one step if it ever
// is needed.
func TestRequestPane_DefaultFocus_IsMethodURL(t *testing.T) {
	pane, _ := setupTestPane(t)
	if _, ok := pane.FocusManager.Current().(*ui.MethodSelector); !ok {
		t.Fatalf("expected SetupRequestPane to start focus on MethodSelector, got %T", pane.FocusManager.Current())
	}

	// Move focus elsewhere (Name), then reset - it must land back on
	// method+URL, not wherever it happened to be before the reset.
	pane.FocusManager.FocusIndex(0)
	if _, ok := pane.FocusManager.Current().(*textinput.Model); !ok {
		t.Fatalf("test setup error: expected NameInput focused, got %T", pane.FocusManager.Current())
	}
	pane.PanelFocused = true

	result, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: 'r', Mod: tea.ModCtrl}))
	if _, ok := result.FocusManager.Current().(*ui.MethodSelector); !ok {
		t.Errorf("expected reset to focus MethodSelector, got %T", result.FocusManager.Current())
	}
}

// TestRequestPane_ReinitRequestPane_DefaultsFocusToMethodURL verifies
// loading a different request (e.g. selecting one from the sidebar) also
// defaults focus to method+URL, regardless of whatever was focused in
// whatever request was open before.
func TestRequestPane_ReinitRequestPane_DefaultsFocusToMethodURL(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.FocusManager.FocusIndex(0) // Name, on the default blank request

	req := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/hosts/{id}",
		Name:    "getHost",
		Headers: map[string]string{},
	}
	pane.reinitRequestPane(req)

	if _, ok := pane.FocusManager.Current().(*ui.MethodSelector); !ok {
		t.Errorf("expected loading a request to default focus to MethodSelector, got %T", pane.FocusManager.Current())
	}
}
