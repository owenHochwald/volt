package responsepane

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestResponsePane_RequestTab_ShowsSentRequest(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	// Wide/tall enough that nothing wraps or clips - content now goes
	// through the viewport (see renderActiveTabContent), sized the same
	// way a real terminal would be; the default test dimensions (width 20,
	// viewport height 10) would otherwise clip "Bearer secret" mid-word and
	// cut the Body section off the bottom entirely.
	pane.SetWidth(200)
	pane.SetHeight(50)

	sent := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/widgets/7",
		Headers: map[string]string{"Authorization": "Bearer secret", "X-Env": "prod"},
		Body:    `{"name":"widget-7"}`,
	}
	pane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK"}, sent)

	// Mirrors what Update()'s DirectTab/TabNav handlers do: set activeTab,
	// then refresh the viewport's content for it - every tab now goes
	// through the viewport (see renderActiveTabContent), so switching tabs
	// without this would leave stale content from whichever tab was active
	// when SetResponse last ran.
	pane.activeTab = int(TabRequest)
	pane.updateViewportForActiveTab()
	content := pane.renderActiveTabContent()

	for _, want := range []string{"POST", sent.URL, "Authorization", "Bearer secret", "X-Env", "widget-7"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected the Request tab to contain %q, got:\n%s", want, content)
		}
	}
}

// TestResponsePane_RequestTab_ShowsSentRequestEvenOnConnectionError is a
// regression test for a real bug: renderNormalView (view.go) used to call
// m.viewport.SetContent(m.Response.Error) unconditionally on every render
// whenever Response.Error was set (e.g. "dial tcp ...: connection refused"),
// stomping over whatever updateViewportForActiveTab had put there for the
// *active* tab - so switching to the Request tab after a connection error
// still showed nothing but the raw error text, with no way to see what was
// actually sent. Exercises the real path: SetResponse with an Error-only
// Response plus a real sent Request, switch tabs via Update() (not by
// poking activeTab directly), then render via View() (not
// renderActiveTabContent) since the bug was specifically in View()'s own
// override.
func TestResponsePane_RequestTab_ShowsSentRequestEvenOnConnectionError(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(200)
	pane.SetHeight(50)

	sent := &http.Request{
		Method: http.POST,
		URL:    "http://localhost:1500/host",
	}
	pane.SetResponse(&http.Response{Error: `Post "http://localhost:1500/host": dial tcp 127.0.0.1:1500: connect: connection refused`}, sent)

	model, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: []rune("4")[0], Text: "4"}))
	result := model
	if TabIndex(result.activeTab) != TabRequest {
		t.Fatalf("expected tab 4 (Request) selected, got %v", result.activeTab)
	}

	content := result.View()
	for _, want := range []string{"POST", sent.URL} {
		if !strings.Contains(content, want) {
			t.Errorf("expected the Request tab to show the sent request (%q) even after a connection error, got:\n%s", want, content)
		}
	}
	if strings.Contains(content, "connection refused") {
		t.Errorf("expected the Request tab to NOT show the raw connection error, got:\n%s", content)
	}
}

// TestResponsePane_TimingTab_ExplainsFailedSendInsteadOfZeroDuration is a
// regression test for a real bug report: a connection error (e.g. "dial
// tcp ...: connection refused") never completes a round trip, so
// Response.Duration stays at its Go zero value - rendering that as "Total
// Duration: 0s" read as if the request somehow finished instantly, when
// really nothing was ever measured. The Timing tab should say so plainly
// instead.
func TestResponsePane_TimingTab_ExplainsFailedSendInsteadOfZeroDuration(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(30)
	pane.SetResponse(&http.Response{Error: `dial tcp 127.0.0.1:1: connect: connection refused`}, &http.Request{Method: http.POST, URL: "http://x"})

	model, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: []rune("3")[0], Text: "3"}))
	result := model

	content := result.View()
	if strings.Contains(content, "Total Duration: 0s") || strings.Contains(content, "0 ms") {
		t.Errorf("expected no misleading zero-duration timing after a failed send, got:\n%s", content)
	}
	if !strings.Contains(content, "no timing data available") {
		t.Errorf("expected an explicit explanation that the send failed before completing, got:\n%s", content)
	}
}

func TestResponsePane_RequestTab_NoSentRequestYet(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.activeTab = int(TabRequest)
	pane.Response = &http.Response{StatusCode: 200, Status: "200 OK"}
	pane.updateViewportForActiveTab()

	content := pane.renderActiveTabContent()
	if content == "" {
		t.Error("expected a placeholder message, got empty content")
	}
}

func TestResponsePane_DirectTab4_SwitchesToRequestTab(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK"}, &http.Request{Method: http.GET, URL: "http://x"})

	model, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: []rune("4")[0], Text: "4"}))
	result := model
	if TabIndex(result.activeTab) != TabRequest {
		t.Errorf("expected pressing '4' to select TabRequest, got tab %d", result.activeTab)
	}
}

func TestResponsePane_DirectTab4_IgnoredDuringLoadTest(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetLoadTestStats(&http.LoadTestStats{TotalRequests: 10})

	model, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: []rune("4")[0], Text: "4"}))
	result := model
	if result.activeTab == int(TabRequest) {
		t.Error("expected '4' to be a no-op during a load test, which has no Request tab")
	}
}

// TestResponsePane_AllFourTabs_ScrollWhenContentOverflows is a regression
// test for a real bug: only the Body tab actually went through the
// scrollable viewport - Headers/Timing/Request each rendered their own
// content directly (see the old renderActiveTabContent), so a response
// with enough headers (or a large "sent request" dump) overflowed the box
// with no way to scroll to the rest, and no indication anything was cut
// off. Verifies pgdn actually moves each of the 4 tabs and pgup returns to
// the top, and that the scroll indicator reflects it.
func TestResponsePane_AllFourTabs_ScrollWhenContentOverflows(t *testing.T) {
	headers := make(map[string][]string, 41)
	headers["Content-Type"] = []string{"application/json"}
	for i := 0; i < 40; i++ {
		headers[fmt.Sprintf("X-Header-%02d", i)] = []string{"some-value"}
	}
	sentHeaders := make(map[string]string, 40)
	for i := 0; i < 40; i++ {
		sentHeaders[fmt.Sprintf("X-Sent-%02d", i)] = "some-value"
	}

	// A large, multi-line JSON body so Body has as much to overflow with as
	// the other three tabs.
	bigBody := "{"
	for i := 0; i < 40; i++ {
		if i > 0 {
			bigBody += ","
		}
		bigBody += fmt.Sprintf(`"field%02d":"value"`, i)
	}
	bigBody += "}"

	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	// Deliberately tiny - not enough for 40 headers/fields, nor even
	// Timing's fixed handful of lines.
	pane.SetHeight(8)

	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    headers,
		Duration:   123,
		Body:       bigBody,
	}, &http.Request{Method: http.POST, URL: "http://x", Headers: sentHeaders})

	for _, tt := range []struct {
		tabKey string
		tab    TabIndex
	}{
		{"1", TabBody},
		{"2", TabHeaders},
		{"3", TabTiming},
		{"4", TabRequest},
	} {
		t.Run(tt.tabKey, func(t *testing.T) {
			model, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: []rune(tt.tabKey)[0], Text: tt.tabKey}))
			result := model
			if TabIndex(result.activeTab) != tt.tab {
				t.Fatalf("expected tab %v selected, got %v", tt.tab, result.activeTab)
			}

			if result.viewport.TotalLineCount() <= result.viewport.VisibleLineCount() {
				t.Skipf("tab %v's content happens to fit the test viewport - nothing to scroll", tt.tab)
			}
			if indicator := result.scrollIndicator(); indicator == "" || !strings.Contains(indicator, "more below") {
				t.Errorf("expected a 'more below' scroll indicator before scrolling, got %q", indicator)
			}

			model2, _ := result.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
			scrolled := model2
			if scrolled.viewport.YOffset() == 0 {
				t.Errorf("expected pgdown to move tab %v's scroll offset", tt.tab)
			}

			model3, _ := scrolled.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp}))
			back := model3
			if back.viewport.YOffset() != 0 {
				t.Errorf("expected pgup to return tab %v to the top, got YOffset=%d", tt.tab, back.viewport.YOffset())
			}
		})
	}
}

func TestResponsePane_ScrollIndicator_EmptyWhenContentFits(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(50)
	pane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: `{"ok":true}`}, &http.Request{Method: http.GET, URL: "http://x"})

	if indicator := pane.scrollIndicator(); indicator != "" {
		t.Errorf("expected no scroll indicator when everything fits, got %q", indicator)
	}
}

func TestResponsePane_MaxTabs(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	if got := pane.getMaxTabs(); got != 4 {
		t.Errorf("expected 4 tabs in normal mode (Body/Headers/Timing/Request), got %d", got)
	}

	pane.isLoadTest = true
	if got := pane.getMaxTabs(); got != 3 {
		t.Errorf("expected 3 tabs in load test mode, got %d", got)
	}
}
