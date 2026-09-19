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
