package responsepane

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestResponsePane_WSTab_HiddenUntilArmedAndDataArrives(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())

	if pane.hasWSTab() {
		t.Fatal("expected no WebSocket tab before any send or data")
	}
	if got := pane.getMaxTabs(); got != 4 {
		t.Errorf("expected 4 tabs before arming, got %d", got)
	}

	// A message arriving before any send is never shown - there's no
	// "what came back" to attribute it to yet.
	pane.AppendWSData("unsolicited push", time.Now())
	if pane.hasWSTab() {
		t.Fatal("expected data received before any send to be ignored")
	}

	pane.ArmWSStream()
	if pane.hasWSTab() {
		t.Fatal("expected arming alone (no data yet) not to show the tab")
	}

	pane.AppendWSData("hello", time.Now())
	if !pane.hasWSTab() {
		t.Fatal("expected the tab to appear once data arrives after arming")
	}
	if got := pane.getMaxTabs(); got != 5 {
		t.Errorf("expected 5 tabs once the WebSocket tab exists, got %d", got)
	}
}

// TestResponsePane_WSTab_SecondSendKeepsGrowingInsteadOfResetting is a
// regression test: an earlier version cleared the tab's data (and kicked
// you off it back to Body) on every new send, which not only fought the
// tab's own "live" premise but could leave the viewport showing stale
// content when there was no HTTP response to fall back to either. A
// second (or third, ...) send must never erase what came in after the
// first - it all keeps accumulating in one growing, live view.
func TestResponsePane_WSTab_SecondSendKeepsGrowingInsteadOfResetting(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetWidth(200)
	pane.SetHeight(50)
	pane.ArmWSStream()
	pane.AppendWSData("first batch", time.Now())
	pane.activeTab = int(TabWebSocket)
	pane.updateViewportForActiveTab()

	// A second successful send re-arms (idempotently) - it must not
	// clear anything or move activeTab off the tab being watched.
	pane.ArmWSStream()
	if TabIndex(pane.activeTab) != TabWebSocket {
		t.Fatalf("expected a second arm to leave activeTab on TabWebSocket, got %d", pane.activeTab)
	}
	pane.AppendWSData("second batch", time.Now())

	content := pane.renderActiveTabContent()
	if !strings.Contains(content, "first batch") {
		t.Errorf("expected the first send's data to still be there, got:\n%s", content)
	}
	if !strings.Contains(content, "second batch") {
		t.Errorf("expected the second send's data to be shown too, got:\n%s", content)
	}
}

func TestResponsePane_WSTab_DirectTab5_SelectsItOnlyWhenPresent(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())

	model, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: []rune("5")[0], Text: "5"}))
	result := model
	if TabIndex(result.activeTab) == TabWebSocket {
		t.Error("expected '5' to be a no-op before the WebSocket tab exists")
	}

	result.ArmWSStream()
	result.AppendWSData("hello", time.Now())

	model2, _ := result.Update(tea.KeyPressMsg(tea.Key{Code: []rune("5")[0], Text: "5"}))
	final := model2
	if TabIndex(final.activeTab) != TabWebSocket {
		t.Errorf("expected '5' to select the WebSocket tab once it exists, got %d", final.activeTab)
	}
	content := final.renderActiveTabContent()
	if !strings.Contains(content, "hello") {
		t.Errorf("expected the WebSocket tab's content to show the received data, got:\n%s", content)
	}
}

// TestResponsePane_WSTab_RendersWithoutAnyHTTPResponse verifies the
// WebSocket tab works even if no HTTP request has ever been sent - it's
// independent of m.Response entirely.
func TestResponsePane_WSTab_RendersWithoutAnyHTTPResponse(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	if pane.Response != nil {
		t.Fatal("setup: expected no response yet")
	}

	pane.ArmWSStream()
	pane.AppendWSData("live data", time.Now())
	pane.activeTab = int(TabWebSocket)
	pane.updateViewportForActiveTab()

	content := pane.renderActiveTabContent()
	if !strings.Contains(content, "live data") {
		t.Errorf("expected the WebSocket tab to render even with no HTTP response, got:\n%s", content)
	}
}

// TestResponsePane_WSTab_SwitchingAwayWithNoResponse_ClearsStaleContent is
// a regression test: View()'s top-level "no response yet" placeholder
// used to win unconditionally whenever m.Response was nil, which
// incidentally papered over updateViewportForActiveTab's Body/Headers/
// Timing/Request case silently leaving old viewport content in place
// (never observable, since the placeholder always covered for it). Once
// hasWSTab() can keep that placeholder from winning, switching from the
// WebSocket tab back to Body with no HTTP response ever sent must not
// keep showing the WebSocket tab's old content under the Body tab.
func TestResponsePane_WSTab_SwitchingAwayWithNoResponse_ClearsStaleContent(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.ArmWSStream()
	pane.AppendWSData("ws data", time.Now())
	pane.activeTab = int(TabWebSocket)
	pane.updateViewportForActiveTab()

	pane.activeTab = int(TabBody)
	pane.updateViewportForActiveTab()

	content := pane.renderActiveTabContent()
	if strings.Contains(content, "ws data") {
		t.Errorf("expected switching to Body (no HTTP response sent) to clear the WebSocket tab's stale content, got:\n%s", content)
	}
}

// TestResponsePane_WSTab_LiveAppendsWhileBeingViewed verifies new data
// shows up immediately if the WebSocket tab is the one currently active,
// not just the next time a tab switch happens to re-render it.
func TestResponsePane_WSTab_LiveAppendsWhileBeingViewed(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.ArmWSStream()
	pane.AppendWSData("first", time.Now())
	pane.activeTab = int(TabWebSocket)
	pane.updateViewportForActiveTab()

	pane.AppendWSData("second", time.Now())

	content := pane.renderActiveTabContent()
	if !strings.Contains(content, "first") || !strings.Contains(content, "second") {
		t.Errorf("expected both messages live in the currently-viewed tab, got:\n%s", content)
	}
}
