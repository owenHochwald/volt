package design

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"gopkg.in/yaml.v3"
)

func (c *ColorValue) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("color must be a scalar value")
	}
	return c.set(node.Value)
}

func (c *ColorValue) set(value string) error {
	normalized, _, err := parseColor(value)
	if err != nil {
		return err
	}
	*c = ColorValue(normalized)
	return nil
}

func (c ColorValue) Resolve() (color.Color, error) {
	_, resolved, err := parseColor(string(c))
	return resolved, err
}

func parseColor(value string) (string, color.Color, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, fmt.Errorf("color cannot be empty")
	}
	if strings.EqualFold(value, "default") {
		return "default", lipgloss.NoColor{}, nil
	}

	if strings.HasPrefix(value, "#") {
		hex := strings.ToUpper(value)
		switch len(hex) {
		case 4:
			for _, digit := range hex[1:] {
				if !isHexDigit(digit) {
					return "", nil, fmt.Errorf("invalid color %q", value)
				}
			}
			hex = fmt.Sprintf(
				"#%c%c%c%c%c%c",
				hex[1], hex[1],
				hex[2], hex[2],
				hex[3], hex[3],
			)
		case 7:
			for _, digit := range hex[1:] {
				if !isHexDigit(digit) {
					return "", nil, fmt.Errorf("invalid color %q", value)
				}
			}
		default:
			return "", nil, fmt.Errorf("invalid color %q", value)
		}
		return hex, lipgloss.Color(hex), nil
	}

	index, err := strconv.Atoi(value)
	if err != nil || index < 0 || index > 255 {
		return "", nil, fmt.Errorf("invalid ANSI-256 color %q", value)
	}
	normalized := strconv.Itoa(index)
	return normalized, lipgloss.Color(normalized), nil
}

func isHexDigit(value rune) bool {
	return value >= '0' && value <= '9' ||
		value >= 'A' && value <= 'F' ||
		value >= 'a' && value <= 'f'
}
