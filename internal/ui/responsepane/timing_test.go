package responsepane

import (
	"strings"
	"testing"
	"time"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestRenderTiming_NoTiming_ShowsFallbackNote(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{StatusCode: 200, Status: "200 OK", Duration: 10 * time.Millisecond}

	got := pane.renderTiming()

	if !strings.Contains(got, "No detailed timing breakdown available") {
		t.Errorf("expected a fallback note when Timing is nil, got:\n%s", got)
	}
}

func TestRenderTiming_FreshConnection_ShowsAllPhases(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Duration:   50 * time.Millisecond,
		Timing: &http.Timing{
			DNSLookup:       5 * time.Millisecond,
			TCPConnection:   10 * time.Millisecond,
			TLSHandshake:    15 * time.Millisecond,
			TimeToFirstByte: 40 * time.Millisecond,
			Reused:          false,
		},
	}

	got := pane.renderTiming()

	for _, want := range []string{"DNS Lookup", "TCP Connection", "TLS Handshake", "Time to First Byte"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected timing breakdown to mention %q, got:\n%s", want, got)
		}
	}
}

func TestRenderTiming_ReusedConnection_SkipsConnectionSetupPhases(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Duration:   5 * time.Millisecond,
		Timing: &http.Timing{
			TimeToFirstByte: 3 * time.Millisecond,
			Reused:          true,
		},
	}

	got := pane.renderTiming()

	if strings.Contains(got, "DNS Lookup") || strings.Contains(got, "TCP Connection") {
		t.Errorf("expected connection-setup phases to be skipped for a reused connection, got:\n%s", got)
	}
	if !strings.Contains(got, "reused connection") {
		t.Errorf("expected a note explaining the reused connection, got:\n%s", got)
	}
	if !strings.Contains(got, "Time to First Byte") {
		t.Errorf("expected Time to First Byte to still be shown, got:\n%s", got)
	}
}

// TestRenderTiming_ShowsSentAt verifies the Timing tab shows when the
// request was actually sent, not just how long it took.
func TestRenderTiming_ShowsSentAt(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{StatusCode: 200, Status: "200 OK", Duration: 10 * time.Millisecond}
	pane.SentAt = time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

	got := pane.renderTiming()

	if !strings.Contains(got, "Sent At") || !strings.Contains(got, "15:04:05") {
		t.Errorf("expected the Timing tab to show when the request was sent, got:\n%s", got)
	}
}

// TestRenderTiming_ShowsSentAtEvenOnFailure verifies "Sent At" still shows
// for a request that failed before a response came back (e.g. a dial
// error) - it was still attempted at that time, even though there's no
// duration/breakdown to show alongside it.
func TestRenderTiming_ShowsSentAtEvenOnFailure(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{Error: "connection refused"}
	pane.SentAt = time.Date(2026, 1, 2, 9, 30, 0, 0, time.UTC)

	got := pane.renderTiming()

	if !strings.Contains(got, "09:30:00") {
		t.Errorf("expected \"Sent At\" to show even for a failed request, got:\n%s", got)
	}
}

// TestRenderTiming_OmitsSentAtWhenUnset verifies the zero value (a
// Response built directly in a test, bypassing SetResponse/
// viewHistoryEntry) doesn't render as a garbage "0001-01-01" timestamp.
func TestRenderTiming_OmitsSentAtWhenUnset(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{StatusCode: 200, Status: "200 OK", Duration: 10 * time.Millisecond}

	got := pane.renderTiming()

	if strings.Contains(got, "Sent At") {
		t.Errorf("expected no \"Sent At\" line with SentAt unset, got:\n%s", got)
	}
}

// TestRenderHeaderBar_ShowsSentAtBeforeStatus verifies the persistent
// status bar (shown above the tab bar regardless of which tab is active -
// unlike the Timing tab's own copy, only visible when that tab is
// selected) also leads with when the request was sent, ahead of the
// status code, so it's visible at a glance without switching tabs.
func TestRenderHeaderBar_ShowsSentAtBeforeStatus(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{StatusCode: 200, Status: "200 OK", Duration: 10 * time.Millisecond}
	pane.SentAt = time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

	got := pane.renderHeaderBar()

	sentAtIdx := strings.Index(got, "15:04:05")
	statusIdx := strings.Index(got, "200 OK")
	if sentAtIdx < 0 {
		t.Fatalf("expected the header bar to show when the request was sent, got %q", got)
	}
	if statusIdx < 0 {
		t.Fatalf("expected the header bar to show the status, got %q", got)
	}
	if sentAtIdx >= statusIdx {
		t.Errorf("expected \"Sent At\" before the status code, got %q", got)
	}
}

// TestRenderHeaderBar_OmitsSentAtWhenUnset mirrors
// TestRenderTiming_OmitsSentAtWhenUnset for the header bar.
func TestRenderHeaderBar_OmitsSentAtWhenUnset(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{StatusCode: 200, Status: "200 OK", Duration: 10 * time.Millisecond}

	got := pane.renderHeaderBar()

	if strings.Contains(got, "0001-01-01") {
		t.Errorf("expected no garbage zero-time rendered, got %q", got)
	}
}

func TestRenderTiming_NoTLS_OmitsTLSHandshakeRow(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.Response = &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Duration:   5 * time.Millisecond,
		Timing: &http.Timing{
			DNSLookup:       1 * time.Millisecond,
			TCPConnection:   2 * time.Millisecond,
			TimeToFirstByte: 4 * time.Millisecond,
		},
	}

	got := pane.renderTiming()

	if strings.Contains(got, "TLS Handshake") {
		t.Errorf("expected no TLS Handshake row for a plain HTTP request, got:\n%s", got)
	}
}
