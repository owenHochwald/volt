package design

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"gopkg.in/yaml.v3"
)

func ParseThemeConfig(data []byte) (ThemeConfig, error) {
	var config ThemeConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&config); err != nil {
		return ThemeConfig{}, fmt.Errorf("parse theme: %w", err)
	}
	return config, nil
}

func ResolveThemeConfig(config ThemeConfig) (Theme, error) {
	if config.Version != ThemeSchemaVersion {
		return Theme{}, fmt.Errorf(
			"unsupported theme schema version %d; expected %d",
			config.Version,
			ThemeSchemaVersion,
		)
	}
	if strings.TrimSpace(config.Name) == "" {
		return Theme{}, fmt.Errorf("theme name is required")
	}

	baseName := strings.ToLower(strings.TrimSpace(config.Extends))
	if baseName == "" {
		baseName = "default"
	}
	if baseName != "default" && baseName != "volt" {
		return Theme{}, fmt.Errorf("unknown base theme %q", config.Extends)
	}

	theme := DefaultTheme()
	theme.Name = config.Name

	overrides := []struct {
		name         string
		value        ColorValue
		target       *color.Color
		allowDefault bool
	}{
		{"canvas", config.Colors.Canvas, &theme.Colors.Canvas, true},
		{"surface", config.Colors.Surface, &theme.Colors.Surface, true},
		{"surface_raised", config.Colors.SurfaceRaised, &theme.Colors.SurfaceRaised, true},
		{"border", config.Colors.Border, &theme.Colors.Border, true},
		{"text", config.Colors.Text, &theme.Colors.Text, true},
		{"text_muted", config.Colors.TextMuted, &theme.Colors.TextMuted, true},
		{"brand", config.Colors.Brand, &theme.Colors.Brand, false},
		{"brand_strong", config.Colors.BrandStrong, &theme.Colors.BrandStrong, false},
		{"charge", config.Colors.Charge, &theme.Colors.Charge, false},
		{"signal", config.Colors.Signal, &theme.Colors.Signal, false},
		{"info", config.Colors.Info, &theme.Colors.Info, false},
		{"success", config.Colors.Success, &theme.Colors.Success, false},
		{"warning", config.Colors.Warning, &theme.Colors.Warning, false},
		{"error", config.Colors.Error, &theme.Colors.Error, false},
		{"methods.get", config.Methods.GET, &theme.Colors.MethodGET, false},
		{"methods.post", config.Methods.POST, &theme.Colors.MethodPOST, false},
		{"methods.put", config.Methods.PUT, &theme.Colors.MethodPUT, false},
		{"methods.patch", config.Methods.PATCH, &theme.Colors.MethodPATCH, false},
		{"methods.delete", config.Methods.DELETE, &theme.Colors.MethodDELETE, false},
		{"charts.primary", config.Charts.Primary, &theme.Colors.ChartPrimary, false},
		{"charts.secondary", config.Charts.Secondary, &theme.Colors.ChartSecondary, false},
		{"charts.good", config.Charts.Good, &theme.Colors.ChartGood, false},
		{"charts.bad", config.Charts.Bad, &theme.Colors.ChartBad, false},
	}

	applied := false
	for _, override := range overrides {
		if override.value == "" {
			continue
		}
		resolved, err := override.value.Resolve()
		if err != nil {
			return Theme{}, fmt.Errorf("%s: %w", override.name, err)
		}
		if _, isDefault := resolved.(lipgloss.NoColor); isDefault && !override.allowDefault {
			return Theme{}, fmt.Errorf("%s cannot use terminal default color", override.name)
		}
		*override.target = resolved
		applied = true
	}

	if !applied {
		return Theme{}, fmt.Errorf("theme must override at least one color")
	}

	return theme, nil
}
