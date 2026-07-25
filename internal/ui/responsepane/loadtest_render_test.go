package responsepane

import (
	"strings"
	"testing"
	"time"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestLoadTestOverviewTreatsHTTPFailuresAsFailures(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetLoadTestStats(failingLoadTestStats())

	overview := pane.renderLoadTestOverview()
	for _, expected := range []string{
		"Success",
		"1 (33.3%)",
		"Failed",
		"2 (66.7%)",
	} {
		if !strings.Contains(overview, expected) {
			t.Errorf("overview does not contain %q:\n%s", expected, overview)
		}
	}
}

func TestLoadTestErrorsIncludeSortedStatusBreakdown(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetLoadTestStats(failingLoadTestStats())

	rendered := pane.renderLoadTestErrors()
	for _, expected := range []string{
		"HTTP Status Codes",
		"204",
		"400",
		"503",
		"HTTP 4xx responses",
		"HTTP 5xx responses",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("error view does not contain %q:\n%s", expected, rendered)
		}
	}
	if index400, index503 := strings.Index(rendered, "400"), strings.Index(rendered, "503"); index400 > index503 {
		t.Fatalf("status codes are not sorted:\n%s", rendered)
	}
}

func TestLoadTestStatusCallsOutFailures(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetLoadTestStats(failingLoadTestStats())

	rendered := pane.renderLoadTestView()
	if !strings.Contains(rendered, "2 failed") {
		t.Fatalf("status does not call out failures:\n%s", rendered)
	}
}

func TestLoadTestStatsBuildBoundedIntervalLatencySeries(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	start := time.Now()

	for i := 1; i <= maxLatencySamples+5; i++ {
		pane.SetLoadTestStats(&http.LoadTestStats{
			StartTime:         start,
			TotalRequests:     1000,
			CompletedRequests: i * 10,
			TotalDuration:     time.Duration(i*i) * 100 * time.Millisecond,
		})
	}

	if got := len(pane.latencySamples); got != maxLatencySamples {
		t.Fatalf("latency samples = %d, want %d", got, maxLatencySamples)
	}
	if got := pane.latencySamples[len(pane.latencySamples)-1]; got != 730*time.Millisecond {
		t.Fatalf("latest interval latency = %s, want 730ms", got)
	}
}

func TestLoadTestStatsPreserveSelectedTabDuringLiveUpdates(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	start := time.Now()
	pane.SetLoadTestStats(&http.LoadTestStats{
		StartTime:     start,
		TotalRequests: 100,
	})
	pane.activeTab = int(TabLoadTestLatency)

	pane.SetLoadTestStats(&http.LoadTestStats{
		StartTime:         start,
		TotalRequests:     100,
		CompletedRequests: 50,
		TotalDuration:     500 * time.Millisecond,
	})

	if got := TabIndex(pane.activeTab); got != TabLoadTestLatency {
		t.Fatalf("active tab = %d, want latency tab", got)
	}
}

func failingLoadTestStats() *http.LoadTestStats {
	start := time.Now().Add(-time.Second)
	return &http.LoadTestStats{
		StartTime:         start,
		EndTime:           start.Add(time.Second),
		TotalRequests:     3,
		CompletedRequests: 3,
		FailedRequests:    2,
		Percentiles:       nil,
		Errors: map[string]int64{
			"http_5xx": 1,
			"http_4xx": 1,
		},
		StatusCodes: map[int]int64{
			503: 1,
			204: 1,
			400: 1,
		},
	}
}
