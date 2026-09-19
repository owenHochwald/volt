package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/utils"
)

func ctrlKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: r, Mod: tea.ModCtrl})
}

// TestToggleWSDock_EntersDockedModeAndFocusesWSPanel verifies ctrl+t docks
// the WebSocket pane and immediately focuses it, closing the floating
// modal if it happened to be open (showing both at once would be
// redundant - see toggleWSDock).
func TestToggleWSDock_EntersDockedModeAndFocusesWSPanel(t *testing.T) {
	model := newTestModel(t)
	model.showWSModal = true

	updated, _ := model.Update(ctrlKey('t'))
	got := updated.(Model)

	if !got.wsDocked {
		t.Fatal("expected ctrl+t to dock the WebSocket pane")
	}
	if got.showWSModal {
		t.Error("expected docking to close the floating modal")
	}
	if got.focusedPanel != utils.WSPanel {
		t.Fatalf("expected focus to jump to WSPanel, got %d", got.focusedPanel)
	}
}

// TestToggleWSDock_UndockingFromWSPanelFallsBackToResponse verifies
// undocking while WSPanel is focused moves focus somewhere still valid,
// rather than leaving it pointing at a panel no longer in the cycle.
func TestToggleWSDock_UndockingFromWSPanelFallsBackToResponse(t *testing.T) {
	model := newTestModel(t)
	model, _ = updateAppModel(model, ctrlKey('t')) // dock
	if model.focusedPanel != utils.WSPanel {
		t.Fatalf("setup: expected WSPanel focused, got %d", model.focusedPanel)
	}

	updated, _ := model.Update(ctrlKey('t')) // undock
	got := updated.(Model)

	if got.wsDocked {
		t.Fatal("expected ctrl+t to undock")
	}
	if got.focusedPanel != utils.ResponsePanel {
		t.Fatalf("expected focus to fall back to ResponsePanel, got %d", got.focusedPanel)
	}
}

// TestMovePanel_IncludesWSPanelOnlyWhenDocked verifies ctrl+w/ctrl+e cycle
// through 4 panels (including WSPanel) once docked, and only the original
// 3 otherwise - a regression risk since panelCount used to be a fixed
// constant.
func TestMovePanel_IncludesWSPanelOnlyWhenDocked(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.ResponsePanel)

	undocked, _ := updateAppModel(model, ctrlKey('w'))
	if undocked.focusedPanel != utils.SidebarPanel {
		t.Fatalf("undocked: expected ctrl+w from Response to wrap to Sidebar, got %d", undocked.focusedPanel)
	}

	model, _ = updateAppModel(model, ctrlKey('t')) // dock
	model.setFocusedPanel(utils.ResponsePanel)

	docked, _ := updateAppModel(model, ctrlKey('w'))
	if docked.focusedPanel != utils.WSPanel {
		t.Fatalf("docked: expected ctrl+w from Response to advance to WSPanel, got %d", docked.focusedPanel)
	}

	wrapped, _ := updateAppModel(docked, ctrlKey('w'))
	if wrapped.focusedPanel != utils.SidebarPanel {
		t.Fatalf("docked: expected ctrl+w from WSPanel to wrap to Sidebar, got %d", wrapped.focusedPanel)
	}
}

// TestOpenWebSocket_WhenDocked_FocusesInsteadOfOpeningModal verifies
// ctrl+k jumps to the docked panel rather than opening a redundant
// floating overlay on top of it.
func TestOpenWebSocket_WhenDocked_FocusesInsteadOfOpeningModal(t *testing.T) {
	model := newTestModel(t)
	model, _ = updateAppModel(model, ctrlKey('t')) // dock
	model.setFocusedPanel(utils.SidebarPanel)

	updated, _ := model.Update(ctrlKey('k'))
	got := updated.(Model)

	if got.showWSModal {
		t.Error("expected ctrl+k not to open the floating modal while docked")
	}
	if got.focusedPanel != utils.WSPanel {
		t.Fatalf("expected ctrl+k to focus WSPanel while docked, got %d", got.focusedPanel)
	}
}

// TestResizeContentForWSDock_Splits25_50_25 verifies the wide layout's
// height split matches what was asked for: 25% request, 50% response,
// 25% WebSocket.
func TestResizeContentForWSDock_Splits25_50_25(t *testing.T) {
	model := newTestModel(t)
	model.width, model.height = 160, 60
	model, _ = updateAppModel(model, ctrlKey('t')) // dock

	layout := model.currentLayout()
	if layout.mode != layoutWide {
		t.Fatalf("expected layoutWide at 160x60, got mode %d", layout.mode)
	}

	total := layout.requestHeight + layout.responseHeight + layout.wsHeight
	if total != layout.contentHeight {
		t.Errorf("expected the three heights to sum to contentHeight (%d), got %d", layout.contentHeight, total)
	}

	wantRequest := layout.contentHeight * 25 / 100
	wantResponse := layout.contentHeight * 50 / 100
	if layout.requestHeight != wantRequest {
		t.Errorf("requestHeight = %d, want %d (25%%)", layout.requestHeight, wantRequest)
	}
	if layout.responseHeight != wantResponse {
		t.Errorf("responseHeight = %d, want %d (50%%)", layout.responseHeight, wantResponse)
	}
}

// TestFocusedContext_MapsWSPanelToContextWebSocket is a lightweight check
// that the Help modal's contextual open ("?") wouldn't silently fall back
// to the Sidebar tab while WSPanel is focused.
func TestFocusedContext_MapsWSPanelToContextWebSocket(t *testing.T) {
	model := newTestModel(t)
	model.wsDocked = true
	model.setFocusedPanel(utils.WSPanel)

	if got := model.focusedContext(); got != "WebSocket" {
		t.Fatalf("focusedContext() = %q, want %q", got, "WebSocket")
	}
}
