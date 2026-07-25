package design

import (
	"encoding/json"
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
	"gopkg.in/yaml.v3"
)

func TestColorValueAcceptsDocumentedFormats(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		unmarshal  func([]byte, any) error
		normalized ColorValue
		color      color.Color
	}{
		{
			name:       "JSON true color",
			input:      `"#a1b2c3"`,
			unmarshal:  json.Unmarshal,
			normalized: "#A1B2C3",
			color:      lipgloss.Color("#A1B2C3"),
		},
		{
			name:       "JSON ANSI number",
			input:      `135`,
			unmarshal:  json.Unmarshal,
			normalized: "135",
			color:      lipgloss.Color("135"),
		},
		{
			name:       "YAML short color",
			input:      `"#a2f"`,
			unmarshal:  yaml.Unmarshal,
			normalized: "#AA22FF",
			color:      lipgloss.Color("#AA22FF"),
		},
		{
			name:       "YAML ANSI number",
			input:      `78`,
			unmarshal:  yaml.Unmarshal,
			normalized: "78",
			color:      lipgloss.Color("78"),
		},
		{
			name:       "terminal default",
			input:      `"default"`,
			unmarshal:  json.Unmarshal,
			normalized: "default",
			color:      lipgloss.NoColor{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var value ColorValue
			assert.NoError(t, test.unmarshal([]byte(test.input), &value))
			assert.Equal(t, test.normalized, value)

			resolved, err := value.Resolve()
			assert.NoError(t, err)
			assert.Equal(t, test.color, resolved)
		})
	}
}

func TestColorValueRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{
		`""`,
		`"#12"`,
		`"#GGGGGG"`,
		`256`,
		`-1`,
		`"purple"`,
		`true`,
	} {
		t.Run(input, func(t *testing.T) {
			var value ColorValue
			assert.Error(t, json.Unmarshal([]byte(input), &value))
		})
	}
}
