package buildinfo

import "testing"

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty development build", input: "", want: "dev"},
		{name: "go development marker", input: "(devel)", want: "dev"},
		{name: "explicit development build", input: "dev", want: "dev"},
		{name: "release without prefix", input: "0.1.3", want: "v0.1.3"},
		{name: "release with prefix", input: "v0.1.3", want: "v0.1.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeVersion(tt.input); got != tt.want {
				t.Fatalf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
