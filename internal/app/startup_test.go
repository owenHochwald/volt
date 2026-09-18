package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/utils"
)

func TestStartupHeaderReclaimsLayoutSpace(t *testing.T) {
	model := newTestModel(t)
	model.width = 160
	model.height = 50
	model.applyLayout(model.currentLayout())

	assert.Equal(t, 7, model.currentLayout().headerHeight)

	model, nextCmd := updateAppModel(model, startupAdvanceMsg{
		frame: ui.HeaderFrameCompressed,
	})
	assert.Equal(t, ui.HeaderFrameCompressed, model.startupFrame)
	assert.Equal(t, 2, model.currentLayout().headerHeight)
	assert.NotZero(t, nextCmd)

	model, _ = updateAppModel(model, startupAdvanceMsg{
		frame: ui.HeaderFrameCompact,
	})
	assert.Equal(t, ui.HeaderFrameCompact, model.startupFrame)
	assert.Equal(t, 2, model.currentLayout().headerHeight)
	assertViewFills(t, model.View().Content, 160, 50)
}

func TestFirstKeyDismissesStartupWithoutSwallowingInput(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.RequestPanel)

	model, _ = updateAppModel(model, appKeyPress('l', "l", tea.ModCtrl))

	assert.Equal(t, ui.HeaderFrameCompact, model.startupFrame)
	assert.True(t, model.requestPane.LoadTestMode)
	assert.True(t, strings.Contains(model.View().Content, "LOAD TEST"))
}

func TestCompactHeaderShowsFocusedPanelAndMode(t *testing.T) {
	model := newTestModel(t)
	model.width = 120
	model.height = 35
	model.setStartupFrame(ui.HeaderFrameCompact)
	model.setFocusedPanel(utils.ResponsePanel)

	layout := model.currentLayout()
	header := model.headerView(layout)
	view := model.View().Content
	assert.Equal(t, 2, lipgloss.Height(header))
	assert.True(t, strings.Contains(view, "/  RESPONSE"))
	assert.True(t, strings.Contains(view, "NORMAL"))
	assert.Equal(t, 120, lipgloss.Width(view))
	assert.Equal(t, 35, lipgloss.Height(view))
}
