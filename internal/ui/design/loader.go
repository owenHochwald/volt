package design

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const themeEnvironmentVariable = "VOLT_THEME"

type ThemeLoadResult struct {
	Theme   Theme
	Source  string
	Warning string
}

type themeLoadOptions struct {
	Selection string
	ConfigDir string
	HomeDir   string
}

func LoadUserTheme() ThemeLoadResult {
	configDir, _ := os.UserConfigDir()
	homeDir, _ := os.UserHomeDir()
	return loadUserTheme(themeLoadOptions{
		Selection: os.Getenv(themeEnvironmentVariable),
		ConfigDir: configDir,
		HomeDir:   homeDir,
	})
}

func loadUserTheme(options themeLoadOptions) ThemeLoadResult {
	selection := strings.TrimSpace(options.Selection)
	if selection != "" {
		if theme, ok := builtInTheme(selection); ok {
			return ThemeLoadResult{Theme: theme, Source: theme.Name}
		}
		if !strings.EqualFold(filepath.Ext(selection), ".yaml") {
			return defaultThemeResult(fmt.Sprintf(
				"%s must name a built-in mode or a .yaml file; using default",
				themeEnvironmentVariable,
			))
		}
		return loadThemeFile(filepath.Clean(selection))
	}

	paths := make([]string, 0, 2)
	if options.ConfigDir != "" {
		paths = append(paths, filepath.Join(options.ConfigDir, "volt", "theme.yaml"))
	}
	if options.HomeDir != "" {
		paths = append(paths, filepath.Join(options.HomeDir, ".volt", "theme.yaml"))
	}

	for _, path := range paths {
		result, exists := loadAutomaticThemeFile(path)
		if exists {
			return result
		}
	}
	return defaultThemeResult("")
}

func loadAutomaticThemeFile(path string) (ThemeLoadResult, bool) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ThemeLoadResult{}, false
	}
	if err != nil {
		return defaultThemeResult(themeWarning(path, err)), true
	}
	return resolveThemeFile(path, data), true
}

func loadThemeFile(path string) ThemeLoadResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultThemeResult(themeWarning(path, err))
	}
	return resolveThemeFile(path, data)
}

func resolveThemeFile(path string, data []byte) ThemeLoadResult {
	config, err := ParseThemeConfig(data)
	if err != nil {
		return defaultThemeResult(themeWarning(path, err))
	}
	theme, err := ResolveThemeConfig(config)
	if err != nil {
		return defaultThemeResult(themeWarning(path, err))
	}
	return ThemeLoadResult{Theme: theme, Source: path}
}

func builtInTheme(name string) (Theme, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "default", "volt":
		return DefaultTheme(), true
	default:
		return Theme{}, false
	}
}

func defaultThemeResult(warning string) ThemeLoadResult {
	return ThemeLoadResult{
		Theme:   DefaultTheme(),
		Source:  "default",
		Warning: warning,
	}
}

func themeWarning(path string, err error) string {
	return fmt.Sprintf("Theme %s: %v; using default", path, err)
}
