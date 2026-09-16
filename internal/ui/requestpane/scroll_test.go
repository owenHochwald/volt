package requestpane

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
)

// TestRequestPane_ScrollFieldsDown_RevealsFieldsBeyondTheBox is a
// regression test for a real bug found while building this: PageDown
// appeared to do nothing at all, in any terminal size, once a request had
// enough fields (e.g. a wide imported body schema) that they didn't fit in
// the box. Two separate issues combined to cause it:
//
//  1. FieldsViewport.lines are only populated inside View(), which has a
//     value receiver - its SetContent call never persisted back to the
//     model Update() operates on, so from Update()'s perspective the
//     viewport's content was always empty, and pgdown against empty
//     content is a no-op (bubbles' AtBottom() reads true immediately).
//  2. FieldsViewport.Height was likewise only ever set inside View(), so
//     even after fixing (1), Update() would compute page-scroll amounts
//     against a Height of 0.
//
// Both must stay fixed together for this to pass.
func TestRequestPane_ScrollFieldsDown_RevealsFieldsBeyondTheBox(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{
		Name:    "test",
		Method:  http.POST,
		URL:     "https://api.example.com/small",
		Body:    `{"field1":"<string>","field2":"<string>","field3":"<string>"}`,
		Headers: map[string]string{},
	}
	pane.reinitRequestPane(req)
	pane.SetWidth(150)
	// A deliberately short box - enough for the method/URL line and Name,
	// not enough to also fit the Body Params section without scrolling.
	pane.SetHeight(11)
	pane.PanelFocused = true

	before := pane.View()
	if strings.Contains(before, "field3") {
		t.Fatal("expected field3 to start out scrolled below the visible box")
	}

	model, _ := pane.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	scrolled, ok := model.(RequestPane)
	if !ok {
		t.Fatalf("expected a RequestPane, got %T", model)
	}

	if scrolled.FieldsViewport.YOffset == 0 {
		t.Fatal("expected pgdown to move the scroll offset")
	}
	after := scrolled.View()
	if !strings.Contains(after, "field3") {
		t.Errorf("expected field3 to be visible after pgdown, got:\n%s", after)
	}

	// And pgup should bring it back to the top.
	model, _ = scrolled.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	back, ok := model.(RequestPane)
	if !ok {
		t.Fatalf("expected a RequestPane, got %T", model)
	}
	if back.FieldsViewport.YOffset != 0 {
		t.Errorf("expected pgup to return to the top, got YOffset=%d", back.FieldsViewport.YOffset)
	}
}
