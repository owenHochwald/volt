package ui

import (
	"strings"
	"testing"
)

func TestHeaderDisplaysInjectedVersion(t *testing.T) {
	tests := []struct {
		name    string
		compact bool
	}{
		{name: "wide"},
		{name: "compact", compact: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := SetupHeader("v9.8.7")
			header.SetSize(120, tt.compact)

			if view := header.View(); !strings.Contains(view, "v9.8.7") {
				t.Fatalf("header does not contain injected version: %q", view)
			}
		})
	}
}
