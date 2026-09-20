package requestpane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

// TestMethodSelector_ArrowKeysCycleSameAsHL verifies the right/left arrow
// keys cycle the HTTP method exactly like l/h - a regression test for a
// bug report that arrow keys silently did nothing on enum/bool/method
// controls (only h/l were ever wired).
func TestMethodSelector_ArrowKeysCycleSameAsHL(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.PanelFocused = true
	if _, ok := pane.FocusManager.Current().(*ui.MethodSelector); !ok {
		t.Fatalf("expected MethodSelector focused by default, got %T", pane.FocusManager.Current())
	}
	before := pane.MethodSelector.Current()

	result, _ := pane.currentMode.HandleInput(&pane, tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	if result.MethodSelector.Current() == before {
		t.Fatalf("expected the right arrow key to advance the method past %q", before)
	}

	back, _ := result.currentMode.HandleInput(result, tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}))
	if back.MethodSelector.Current() != before {
		t.Fatalf("expected the left arrow key to return to %q, got %q", before, back.MethodSelector.Current())
	}
}

// TestEnumSelector_ArrowKeysCycleSameAsHL mirrors the above for a body
// param backed by an EnumSelector (also used for booleans).
func TestEnumSelector_ArrowKeysCycleSameAsHL(t *testing.T) {
	pane, _ := setupTestPane(t)
	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"status": {Type: "string", Enum: []string{"active", "inactive"}},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	for pane.FocusManager.Current() != nil {
		if _, ok := pane.FocusManager.Current().(*EnumSelector); ok {
			break
		}
		pane.FocusManager.Next()
	}
	enum, ok := pane.FocusManager.Current().(*EnumSelector)
	if !ok {
		t.Fatal("expected to find an EnumSelector in the focus cycle")
	}
	if got := enum.Value(); got != "" {
		t.Fatalf("expected the enum (no schema default) to start unset, got %q", got)
	}

	result, _ := pane.currentMode.HandleInput(&pane, tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	afterRight, ok := result.FocusManager.Current().(*EnumSelector)
	if !ok {
		t.Fatal("expected the EnumSelector to still be focused")
	}
	if got := afterRight.Value(); got != "active" {
		t.Fatalf("expected the right arrow key to select the first option, got %q", got)
	}

	back, _ := result.currentMode.HandleInput(result, tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}))
	afterLeft, ok := back.FocusManager.Current().(*EnumSelector)
	if !ok {
		t.Fatal("expected the EnumSelector to still be focused")
	}
	if got := afterLeft.Value(); got != "" {
		t.Fatalf("expected the left arrow key to return to unset, got %q", got)
	}
}

// TestArrayCountControl_ArrowKeysAddAndRemoveItems mirrors the above for
// an ArrayCountControl.
func TestArrayCountControl_ArrowKeysAddAndRemoveItems(t *testing.T) {
	pane, _ := setupTestPane(t)
	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"tags": {Type: "array", ItemsRefName: "Tag", Items: tagItemSchema()},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	for i := 0; i < 2; i++ {
		pane.FocusManager.Next()
	}
	if _, ok := pane.FocusManager.Current().(*ArrayCountControl); !ok {
		t.Fatalf("expected the array count control to be focused, got %T", pane.FocusManager.Current())
	}

	result, _ := pane.currentMode.HandleInput(&pane, tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	if len(result.ArrayParams[0].Items) != 2 {
		t.Fatalf("expected the right arrow key to add a second item, got %d", len(result.ArrayParams[0].Items))
	}

	back, _ := result.currentMode.HandleInput(result, tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}))
	if len(back.ArrayParams[0].Items) != 1 {
		t.Fatalf("expected the left arrow key to remove it again, got %d", len(back.ArrayParams[0].Items))
	}
}
