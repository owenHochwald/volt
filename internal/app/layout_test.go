package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/utils"
)

func TestCalculateLayoutUsesResponsiveModes(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		mode          layoutMode
	}{
		{name: "spacious", width: 160, height: 50, mode: layoutWide},
		{name: "narrow", width: 88, height: 35, mode: layoutFocused},
		{name: "short", width: 140, height: 24, mode: layoutFocused},
		{name: "minimum supported", width: 60, height: 20, mode: layoutFocused},
		{name: "too narrow", width: 59, height: 30, mode: layoutTooSmall},
		{name: "too short", width: 100, height: 19, mode: layoutTooSmall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := calculateLayout(tt.width, tt.height)
			if layout.mode != tt.mode {
				t.Fatalf("mode = %v, want %v", layout.mode, tt.mode)
			}
			if layout.width < 0 || layout.height < 0 ||
				layout.headerHeight < 0 || layout.contentHeight < 0 ||
				layout.sidebarWidth < 0 || layout.mainWidth < 0 ||
				layout.requestHeight < 0 || layout.responseHeight < 0 {
				t.Fatalf("layout contains a negative dimension: %+v", layout)
			}
		})
	}
}

func TestResponsiveViewNeverExceedsTerminal(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
	}{
		{name: "wide", width: 160, height: 50},
		{name: "narrow", width: 88, height: 35},
		{name: "short", width: 140, height: 24},
		{name: "minimum", width: 60, height: 20},
		{name: "unsupported", width: 40, height: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newTestModel(t)
			for _, panel := range []utils.Panel{
				utils.SidebarPanel,
				utils.RequestPanel,
				utils.ResponsePanel,
			} {
				model.setFocusedPanel(panel)
				updated, _ := model.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
				model = updated.(Model)
				assertViewFits(t, model.View().Content, tt.width, tt.height)
				if tt.width >= minTerminalWidth && tt.height >= minTerminalHeight {
					assertViewFills(t, model.View().Content, tt.width, tt.height)
				}
			}
		})
	}
}

func TestFocusedLayoutFitsLoadTestAndHelpViews(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.RequestPanel)
	updated, _ := model.Update(appKeyPress('l', "", tea.ModCtrl))
	model = updated.(Model)
	if !model.requestPane.LoadTestMode {
		t.Fatal("control-l did not enable load-test mode")
	}

	updated, _ = model.Update(tea.WindowSizeMsg{Width: 72, Height: 22})
	model = updated.(Model)
	assertViewFits(t, model.View().Content, 72, 22)
	assertViewFills(t, model.View().Content, 72, 22)

	updated, _ = model.Update(appKeyPress(tea.KeyF1, "", 0))
	model = updated.(Model)
	if !model.showHelpModal {
		t.Fatal("F1 did not open help")
	}
	assertViewFits(t, model.View().Content, 72, 22)
	assertViewFills(t, model.View().Content, 72, 22)
}

func TestWideLoadTestConfigurationKeepsEveryControlVisible(t *testing.T) {
	model := newTestModel(t)
	model.width = 160
	model.height = 35
	model.setStartupFrame(ui.HeaderFrameCompact)
	model.setFocusedPanel(utils.RequestPanel)
	model.applyLayout(model.currentLayout())

	updated, _ := model.Update(appKeyPress('l', "", tea.ModCtrl))
	model = updated.(Model)
	rendered := ansi.Strip(model.View().Content)

	for _, text := range []string{
		"Concurrency:",
		"Total Requests:",
		"QPS (limit):",
		"Timeout:",
		"RUN LOAD TEST",
	} {
		if !strings.Contains(rendered, text) {
			t.Errorf("wide load-test layout does not contain %q", text)
		}
	}
	assertViewFills(t, model.View().Content, 160, 35)
}

func TestWideActiveLoadTestPrioritizesResults(t *testing.T) {
	model := newTestModel(t)
	model.width = 160
	model.height = 35
	model.setStartupFrame(ui.HeaderFrameCompact)
	model.requestPane.LoadTestMode = true

	configuring := model.currentLayout()
	if configuring.requestHeight <= configuring.responseHeight {
		t.Fatalf("configuration split = %d/%d, want request editor larger", configuring.requestHeight, configuring.responseHeight)
	}

	model.requestPane.RequestInProgress = true
	running := model.currentLayout()
	if running.responseHeight <= running.requestHeight {
		t.Fatalf("running split = %d/%d, want result centerpiece larger", running.requestHeight, running.responseHeight)
	}
}

func TestResizePreservesFocusedPanelAndEditorContent(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.RequestPanel)
	model.requestPane.FocusManager.Next()
	model.requestPane.URLInput.SetValue("https://example.com/resize")

	for _, size := range []tea.WindowSizeMsg{
		{Width: 160, Height: 50},
		{Width: 88, Height: 35},
		{Width: 140, Height: 24},
		{Width: 60, Height: 20},
	} {
		updated, _ := model.Update(size)
		model = updated.(Model)
		if model.focusedPanel != utils.RequestPanel {
			t.Fatalf("resize to %dx%d changed focused panel", size.Width, size.Height)
		}
		if got := model.requestPane.URLInput.Value(); got != "https://example.com/resize" {
			t.Fatalf("resize to %dx%d changed URL to %q", size.Width, size.Height, got)
		}
		assertViewFits(t, model.View().Content, size.Width, size.Height)
		assertViewFills(t, model.View().Content, size.Width, size.Height)
	}
}

func assertViewFills(t *testing.T, view string, width, height int) {
	t.Helper()
	if got := lipgloss.Width(view); got != width {
		t.Fatalf("view width = %d, want terminal width %d", got, width)
	}
	if got := lipgloss.Height(view); got != height {
		t.Fatalf("view height = %d, want terminal height %d", got, height)
	}
}

func assertViewFits(t *testing.T, view string, width, height int) {
	t.Helper()
	if got := lipgloss.Width(view); got > width {
		t.Fatalf("view width = %d, terminal width = %d", got, width)
	}
	if got := lipgloss.Height(view); got > height {
		t.Fatalf("view height = %d, terminal height = %d", got, height)
	}
}
