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
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
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
	result, ok := model.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", model)
	}
	if !result.wrapBody {
		t.Fatal("expected \"w\" to turn wrapping on")
	}

	afterLines := result.viewport.TotalLineCount()
	if afterLines <= beforeLines {
		t.Errorf("expected wrapping the 200-char line at width 40 to produce more content lines (had %d, got %d)", beforeLines, afterLines)
	}

	// Toggling back off must return to the unwrapped (truncated) view.
	model2, _ := result.Update(keyRunes("w"))
	back, ok := model2.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", model2)
	}
	if back.wrapBody {
		t.Error("expected a second \"w\" to turn wrapping back off")
	}
}
