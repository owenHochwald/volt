package responsepane

import (
	"fmt"
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// TestResponsePane_View_TrailerSurvivesLargeScrollableBody is a
// regression test for a real bug (and its follow-up): with a large
// pretty-printed response body (tall enough that the scroll indicator
// shows), head's actual rendered height - tab bar spacing, etc. - only
// ever approximates SetHeight's reserved budget, and used to come out one
// or more lines taller than budgeted; View()'s outer MaxHeight then
// silently clipped whatever came after in the same combined block - first
// just the footer hint, then (once head's clipping was fixed to spare the
// footer) the scroll indicator instead, since it was still part of the
// same clippable block as head. Both need to survive: the indicator (the
// only cue that pgup/pgdn would reveal anything more) and the footer hint
// (how users discover "r"/"s"/"y"/"/"/"gg" etc in the first place).
func TestResponsePane_View_TrailerSurvivesLargeScrollableBody(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(30)

	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < 300; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"id": %d, "title": "padding text to make lines long enough to matter"}`, i)
	}
	b.WriteString("]")
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       b.String(),
	}, &http.Request{})

	out := pane.View()
	if !strings.Contains(out, "more below") {
		t.Errorf("expected the scroll indicator to survive a large scrollable body, got:\n%s", out)
	}
	if !strings.Contains(out, "save to file") {
		t.Errorf("expected the footer key-binding hint to survive a large scrollable body, got:\n%s", out)
	}
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least the indicator and footer lines, got:\n%s", out)
	}
	if !strings.Contains(lines[len(lines)-1], "save to file") {
		t.Errorf("expected the footer hint to be the last rendered line, got last line: %q", lines[len(lines)-1])
	}
	if !strings.Contains(lines[len(lines)-2], "more below") {
		t.Errorf("expected the scroll indicator directly above the footer, got: %q", lines[len(lines)-2])
	}
}
