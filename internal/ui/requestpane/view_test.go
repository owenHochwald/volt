package requestpane

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestLoadTestConfigurationFitsAndShowsEveryControl(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetSize(100, 26)
	pane.toggleLoadTestMode()

	view := pane.View()
	for _, text := range []string{
		"LOAD TEST CONFIGURATION",
		"Concurrency:",
		"Total Requests:",
		"QPS (limit):",
		"Timeout:",
		"RUN LOAD TEST",
	} {
		if !strings.Contains(view, text) {
			t.Errorf("load-test view does not contain %q", text)
		}
	}
	if got := lipgloss.Height(view); got > pane.Height {
		t.Fatalf("load-test view height = %d, panel height = %d", got, pane.Height)
	}
}
