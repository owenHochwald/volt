package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/utils"
)

func setupTestModel(t *testing.T) Model {
	t.Helper()
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return SetupModel(db)
}

func shiftTabMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyShiftTab}
}

// TestShiftTab_InRequestPanel_DoesNotCyclePanels guards against a
// regression where shift+tab inside the request pane always triggered the
// app-level CyclePanel binding before ever reaching the request pane's own
// PrevField handling (also bound to shift+tab) - so shift-tabbing backward
// through its fields silently moved focus to a different panel entirely
// instead of ever landing on the previous field.
func TestShiftTab_InRequestPanel_DoesNotCyclePanels(t *testing.T) {
	m := setupTestModel(t)
	m.focusedPanel = utils.RequestPanel

	model, _ := m.Update(shiftTabMsg())
	result, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model)
	}
	if result.focusedPanel != utils.RequestPanel {
		t.Errorf("expected shift+tab to stay in the request panel (letting it handle PrevField), got panel %v", result.focusedPanel)
	}
}

// TestShiftTab_InSidebarOrResponsePanel_StillCyclesPanels verifies the fix
// above didn't disable CyclePanel everywhere - only the request panel has
// a competing use for shift+tab.
func TestShiftTab_InSidebarOrResponsePanel_StillCyclesPanels(t *testing.T) {
	m := setupTestModel(t)
	m.focusedPanel = utils.SidebarPanel

	model, _ := m.Update(shiftTabMsg())
	result, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model)
	}
	if result.focusedPanel != utils.RequestPanel {
		t.Errorf("expected shift+tab from the sidebar to cycle to the request panel, got %v", result.focusedPanel)
	}

	m2 := setupTestModel(t)
	m2.focusedPanel = utils.ResponsePanel
	model2, _ := m2.Update(shiftTabMsg())
	result2, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result2.focusedPanel != utils.SidebarPanel {
		t.Errorf("expected shift+tab from the response panel to wrap to the sidebar, got %v", result2.focusedPanel)
	}
}

// TestFKeys_JumpToPanel_EvenMidEditInRequestPane verifies f1/f2/f3 reliably
// switch panels from anywhere, including while the request panel is focused
// - unlike shift+tab, which the request pane also needs for its own
// PrevField and so can't serve as a panel switch there. F-keys were chosen
// over ctrl+1/2/3 (not a real, distinguishable combination over classic
// terminal input) and alt+1/2/3 (grabbed by most terminal emulators
// themselves for their own tab switching before the app ever sees it).
func TestFKeys_JumpToPanel_EvenMidEditInRequestPane(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyType
		want utils.Panel
	}{
		{"f1", tea.KeyF1, utils.SidebarPanel},
		{"f2", tea.KeyF2, utils.RequestPanel},
		{"f3", tea.KeyF3, utils.ResponsePanel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := setupTestModel(t)
			m.focusedPanel = utils.RequestPanel // start mid-edit, worst case for shift+tab

			model, _ := m.Update(tea.KeyMsg{Type: tt.key})
			result, ok := model.(Model)
			if !ok {
				t.Fatalf("expected Model, got %T", model)
			}
			if result.focusedPanel != tt.want {
				t.Errorf("expected %s to focus panel %v, got %v", tt.name, tt.want, result.focusedPanel)
			}
		})
	}
}
