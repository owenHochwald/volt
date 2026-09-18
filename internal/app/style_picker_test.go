package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/utils"
)

func tKeyMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")}
}

// TestPickBodyStyle_UnzoomsAndRendersInRequestSlot is a regression test
// for a real feature request: opening the body style picker ("t") while
// the response panel was zoomed must collapse it back (there's no
// request slot to show the picker's list in otherwise), and the picker
// itself must render in that slot rather than inside the response pane's
// own box (which - cramming a separate preview snippet into a fixed-size
// area - could previously blow out the whole layout on a response with
// pathologically long lines, e.g. a minified HTML page).
func TestPickBodyStyle_UnzoomsAndRendersInRequestSlot(t *testing.T) {
	m := setupTestModel(t)
	m.focusedPanel = utils.ResponsePanel

	model, _ := m.Update(zKeyMsg())
	zoomed := model.(Model)
	if !zoomed.zoomedResponse {
		t.Fatal("test setup error: expected \"z\" to zoom the response panel")
	}

	model2, _ := zoomed.Update(tKeyMsg())
	result, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result.zoomedResponse {
		t.Error("expected opening the style picker to un-zoom the response panel")
	}
	if !result.responsePane.IsPickingStyle() {
		t.Fatal("expected \"t\" to open the style picker")
	}

	out := result.View()
	if !strings.Contains(out, "Body syntax style") {
		t.Errorf("expected the picker to render in the request pane's slot, got:\n%s", out)
	}
}
