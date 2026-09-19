package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/utils"
)

func tKeyMsg() tea.KeyMsg {
	return tea.KeyPressMsg(tea.Key{Code: []rune("t")[0], Text: "t"})
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
	m := newTestModel(t)
	m.setFocusedPanel(utils.ResponsePanel)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	m = sized.(Model)

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

	out := result.View().Content
	if !strings.Contains(out, "Body syntax style") {
		t.Errorf("expected the picker to render in the request pane's slot, got:\n%s", out)
	}
}

func vKeyMsg() tea.KeyMsg {
	return tea.KeyPressMsg(tea.Key{Code: []rune("v")[0], Text: "v"})
}

// TestViewAuthJar_UnzoomsAndRendersInRequestSlot mirrors
// TestPickBodyStyle_UnzoomsAndRendersInRequestSlot for the saved-cookies/
// auth-headers viewer ("v") - the same request-slot overlay pattern, so
// it needs the same un-zoom-first treatment.
func TestViewAuthJar_UnzoomsAndRendersInRequestSlot(t *testing.T) {
	m := newTestModel(t)
	m.setFocusedPanel(utils.ResponsePanel)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	m = sized.(Model)
	m.responsePane.SetResponse(
		&http.Response{StatusCode: 200, Status: "200 OK"},
		&http.Request{Method: http.GET, URL: "https://api.example.com/x"},
	)

	model, _ := m.Update(zKeyMsg())
	zoomed := model.(Model)
	if !zoomed.zoomedResponse {
		t.Fatal("test setup error: expected \"z\" to zoom the response panel")
	}

	model2, _ := zoomed.Update(vKeyMsg())
	result, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result.zoomedResponse {
		t.Error("expected opening the jar viewer to un-zoom the response panel")
	}
	if !result.responsePane.IsViewingJar() {
		t.Fatal("expected \"v\" to open the jar viewer")
	}

	out := result.View().Content
	if !strings.Contains(out, "Saved cookies & auth for api.example.com") {
		t.Errorf("expected the viewer to render in the request pane's slot, got:\n%s", out)
	}
}
