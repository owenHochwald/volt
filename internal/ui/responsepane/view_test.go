package responsepane

import (
	"fmt"
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// TestResponsePane_View_FooterSurvivesLargeScrollableBody is a regression
// test for a real bug: with a large pretty-printed response body (tall
// enough that the scroll indicator shows), body's actual rendered height
// - tab bar spacing, the scroll indicator, etc. - only ever approximates
// SetHeight's reserved budget, and used to come out one line taller than
// budgeted; View()'s outer MaxHeight then silently clipped the *last*
// line of the whole combined block, which was always the pinned
// key-binding footer hint - the single most useful line to lose, since
// it's how users discover "r"/"s"/"y"/"/"/"gg" etc in the first place.
func TestResponsePane_View_FooterSurvivesLargeScrollableBody(t *testing.T) {
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
	if !strings.Contains(out, "save to file") {
		t.Errorf("expected the footer key-binding hint to survive a large scrollable body, got:\n%s", out)
	}
	lines := strings.Split(out, "\n")
	if len(lines) == 0 || !strings.Contains(lines[len(lines)-1], "save to file") {
		t.Errorf("expected the footer hint to be the last rendered line, got last line: %q", lines[len(lines)-1])
	}
}
