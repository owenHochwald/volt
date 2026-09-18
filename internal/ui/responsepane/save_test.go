package responsepane

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// TestResponsePane_ToggleRawBody_SwitchesBodyTabBetweenRawAndFormatted is a
// regression test for a real feature request: the Body tab always showed
// the pretty-printed/highlighted version (see formatContentByType), with
// no way to see the response body exactly as received (raw's own text,
// stripped of the syntax-highlighting ANSI codes both views now carry -
// see highlightRawBody - must still be the exact, unformatted bytes;
// only pretty gets reformatted/re-indented - see formatJSON).
func TestResponsePane_ToggleRawBody_SwitchesBodyTabBetweenRawAndFormatted(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(30)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       `{"a":1,"b":2}`,
	}, &http.Request{Method: http.GET, URL: "http://x"})

	pretty := pane.View()
	if !strings.Contains(pretty, `"a"`) {
		t.Fatalf("expected the pretty view to show formatted JSON, got:\n%s", pretty)
	}

	model, _ := pane.Update(keyRunes("r"))
	result, ok := model.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", model)
	}

	raw := ansi.Strip(result.View())
	if !strings.Contains(raw, `{"a":1,"b":2}`) {
		t.Errorf("expected the raw view to contain the exact, unformatted response body, got:\n%s", raw)
	}

	// Toggling back must return to the formatted view.
	model2, _ := result.Update(keyRunes("r"))
	back, ok := model2.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", model2)
	}
	if back.rawBody {
		t.Error("expected a second toggle to turn raw view back off")
	}
}

// TestResponsePane_RawBody_IsSyntaxHighlightedButNotReformatted is a
// regression test for a real feature request: raw view used to show
// completely unstyled, flat text - now it's highlighted (see
// highlightRawBody) the same as pretty, using whatever style is
// currently applied (see effectiveBodyStyle), just without pretty's own
// reformatting/re-indentation step (see formatJSON) - the raw text
// itself (ANSI codes stripped) must stay byte-for-byte what was
// received.
func TestResponsePane_RawBody_IsSyntaxHighlightedButNotReformatted(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(30)
	compact := `{"a":1,"b":2}`
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       compact,
	}, &http.Request{Method: http.GET, URL: "http://x"})

	model, _ := pane.Update(keyRunes("r"))
	result := model.(*ResponsePane)

	raw := result.View()
	if !strings.Contains(raw, "\x1b[") {
		t.Errorf("expected the raw view to carry syntax highlighting (ANSI escapes), got plain text:\n%s", raw)
	}
	if !strings.Contains(ansi.Strip(raw), compact) {
		t.Errorf("expected the raw view's text to stay exactly as received (no re-indentation), got:\n%s", ansi.Strip(raw))
	}
}

// TestResponsePane_ToggleRawBody_OnlyAffectsBodyTab verifies "r" is a
// no-op on any tab other than Body - there's nothing for it to toggle on
// Headers/Timing/Request, which don't go through formatContentByType at
// all.
func TestResponsePane_ToggleRawBody_OnlyAffectsBodyTab(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(30)
	pane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: `{"a":1}`}, &http.Request{Method: http.GET, URL: "http://x"})

	model, _ := pane.Update(keyRunes("2")) // Headers tab
	onHeaders, ok := model.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", model)
	}

	model2, _ := onHeaders.Update(keyRunes("r"))
	result, ok := model2.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", model2)
	}
	if result.rawBody {
		t.Error("expected 'r' to have no effect while on a tab other than Body")
	}
}

// TestResponsePane_SaveToFile_WritesRawBodyRegardlessOfToggle is an
// end-to-end regression test for the other half of the feature request:
// saving the response body to a user-named file - and specifically that
// it always saves the *raw* bytes, even while the pretty/formatted view
// is what's currently on screen (rawBody == false).
func TestResponsePane_SaveToFile_WritesRawBodyRegardlessOfToggle(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(30)
	rawBody := `{"a":1,"b":2}`
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       rawBody,
	}, &http.Request{Method: http.GET, URL: "http://x"})

	if pane.rawBody {
		t.Fatal("test setup error: expected to start in pretty (non-raw) view")
	}

	// Open the save prompt.
	model2, cmd := pane.Update(keyRunes("s"))
	opened, ok := model2.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", model2)
	}
	if !opened.savingToFile {
		t.Fatal("expected 's' to open the save-to-file prompt")
	}
	_ = cmd

	// Type a filename.
	path := filepath.Join(t.TempDir(), "response.json")
	current := opened
	for _, r := range path {
		m, _ := current.Update(keyRunes(string(r)))
		current = m.(*ResponsePane)
	}

	// Confirm.
	final, saveCmd := current.Update(tea.KeyMsg{Type: tea.KeyEnter})
	afterSave, ok := final.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", final)
	}
	if afterSave.savingToFile {
		t.Error("expected confirming the filename to close the prompt")
	}
	if saveCmd == nil {
		t.Fatal("expected a non-nil save command")
	}

	msg := saveCmd()
	resultMsg, ok := msg.(SaveResponseBodyMsg)
	if !ok {
		t.Fatalf("expected SaveResponseBodyMsg, got %T", msg)
	}
	if resultMsg.Err != nil {
		t.Fatalf("unexpected save error: %v", resultMsg.Err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if string(data) != rawBody {
		t.Errorf("expected the saved file to contain the raw body %q, got %q", rawBody, string(data))
	}
}

// TestResponsePane_SaveToFile_EscCancelsWithoutSaving verifies pressing
// esc closes the prompt without dispatching any save.
func TestResponsePane_SaveToFile_EscCancelsWithoutSaving(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(30)
	pane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: "hello"}, &http.Request{Method: http.GET, URL: "http://x"})

	model, _ := pane.Update(keyRunes("s"))
	opened := model.(*ResponsePane)

	model2, _ := opened.Update(keyRunes("some-name"))
	typed := model2.(*ResponsePane)

	final, cmd := typed.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result, ok := final.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", final)
	}
	if result.savingToFile {
		t.Error("expected esc to close the prompt")
	}
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, ok := msg.(SaveResponseBodyMsg); ok {
				t.Error("expected esc to never dispatch a save")
			}
		}
	}
}

func TestResolveSavePath(t *testing.T) {
	if _, err := resolveSavePath(""); err == nil {
		t.Error("expected an empty filename to be rejected")
	}
	if _, err := resolveSavePath("   "); err == nil {
		t.Error("expected a whitespace-only filename to be rejected")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory available: %v", err)
	}
	got, err := resolveSavePath("~/response.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "response.json")
	if got != want {
		t.Errorf("expected ~ expansion to %q, got %q", want, got)
	}

	got, err = resolveSavePath("relative/path.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "relative/path.json" {
		t.Errorf("expected a relative path to pass through unchanged, got %q", got)
	}
}
