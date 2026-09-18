package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/storage"
)

func f4KeyMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyF4}
}

// TestToggleHeader_HidesAndRestoresBanner is a regression test for a real
// feature request: "f4" should show/hide the top banner (logo + help
// hint) from anywhere.
func TestToggleHeader_HidesAndRestoresBanner(t *testing.T) {
	m := setupTestModel(t)
	if !m.showHeader {
		t.Fatal("test setup error: expected the header to start shown")
	}

	model, _ := m.Update(f4KeyMsg())
	hidden, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model)
	}
	if hidden.showHeader {
		t.Error("expected \"f4\" to hide the header")
	}

	model2, _ := hidden.Update(f4KeyMsg())
	restored, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if !restored.showHeader {
		t.Error("expected a second \"f4\" to restore the header")
	}
}

// TestToggleHeader_OtherPanelsFillFreedSpace is a regression test for the
// core ask behind this feature: hiding the header must hand its space to
// the panels below, not just leave it blank - mainHeight (and everything
// derived from it) should grow by exactly the header's own height once
// it's gone, and the overall rendered view should stay the same total
// height (modulo the same one line of slack it always had).
func TestToggleHeader_OtherPanelsFillFreedSpace(t *testing.T) {
	m := setupTestModel(t)
	model, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	shown := model.(Model)

	shownHeaderHeight := shown.headerHeight()
	if shownHeaderHeight == 0 {
		t.Fatal("test setup error: expected the header to have a nonzero rendered height")
	}
	shownMainHeight := shown.mainHeight()
	shownViewHeight := lipgloss.Height(shown.View())

	model2, _ := shown.Update(f4KeyMsg())
	hidden := model2.(Model)

	if hidden.headerHeight() != 0 {
		t.Errorf("expected headerHeight() to be 0 once hidden, got %d", hidden.headerHeight())
	}
	if got, want := hidden.mainHeight(), shownMainHeight+shownHeaderHeight; got != want {
		t.Errorf("expected mainHeight() to grow by exactly the header's height (%d), got %d (want %d)", shownHeaderHeight, got, want)
	}
	if got, want := lipgloss.Height(hidden.View()), shownViewHeight; got != want {
		t.Errorf("expected the overall rendered height to stay %d once the header's space is reclaimed, got %d", want, got)
	}
}

// TestSetupModel_RestoresShowHeader verifies a saved "hidden" choice (see
// storage.SaveShowHeader) is restored the next time the app starts.
func TestSetupModel_RestoresShowHeader(t *testing.T) {
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.SaveShowHeader(false); err != nil {
		t.Fatalf("failed to save show-header: %v", err)
	}

	m := SetupModel(db)
	if m.showHeader {
		t.Error("expected SetupModel to restore a previously saved \"hidden\" header")
	}
}
