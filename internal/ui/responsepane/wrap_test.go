package responsepane

import (
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// TestResponsePane_ToggleWrap_WrapsLongLines is a regression test for a
// real feature request: "w" should wrap long lines to the pane's width
// instead of the viewport's default of truncating them (with horizontal
// scroll available) - useful for reading something like a minified page
// without scrolling sideways through it.
func TestResponsePane_ToggleWrap_WrapsLongLines(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetWidth(40)
	pane.SetHeight(20)

	longLine := strings.Repeat("x", 200)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"text/plain"}},
		Body:       longLine,
	}, &http.Request{Method: http.GET, URL: "http://x"})
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	if pane.wrapBody {
		t.Fatal("test setup error: expected wrap to be off by default")
	}
	// viewport.View() always pads to a fixed Height regardless of
	// content, so TotalLineCount() (the actual, unpadded content line
	// count) is what actually reflects whether wrapping happened.
	beforeLines := pane.viewport.TotalLineCount()

	model, _ := pane.Update(keyRunes("w"))
	result := model
	if !result.wrapBody {
		t.Fatal("expected \"w\" to turn wrapping on")
	}

	afterLines := result.viewport.TotalLineCount()
	if afterLines <= beforeLines {
		t.Errorf("expected wrapping the 200-char line at width 40 to produce more content lines (had %d, got %d)", beforeLines, afterLines)
	}

	// Toggling back off must return to the unwrapped (truncated) view.
	model2, _ := result.Update(keyRunes("w"))
	back := model2
	if back.wrapBody {
		t.Error("expected a second \"w\" to turn wrapping back off")
	}
}

// TestWrapLines_PreservesColorOnEveryWrappedLine is a regression test for
// a real bug: ansi.Wrap only opens a styled run's SGR code on the first
// physical line it produces from a long logical one, and closes it on the
// last - the lines in between carry no color of their own. Invisible as
// long as the whole run stays on screen together, but the viewport this
// feeds shows a scrolled window of individual lines - once the first
// line (carrying the opening code) scrolled out of view, every line
// after it rendered with no color at all. Every physical line must carry
// its own copy of whatever style was active at its start.
func TestWrapLines_PreservesColorOnEveryWrappedLine(t *testing.T) {
	colored := "\x1b[38;5;212m" + strings.Repeat("A", 100) + "\x1b[m"
	wrapped := wrapLines(colored, 20)

	lines := strings.Split(wrapped, "\n")
	if len(lines) < 3 {
		t.Fatalf("test setup error: expected at least 3 wrapped lines, got %d: %q", len(lines), wrapped)
	}
	for i, line := range lines {
		if !strings.Contains(line, "\x1b[38;5;212m") {
			t.Errorf("expected wrapped line %d to carry its own color code, got %q", i, line)
		}
	}
}
