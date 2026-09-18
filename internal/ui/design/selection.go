package design

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const themeSelectionFilename = "config.yaml"

func SaveThemeSelection(source string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("find user config directory: %w", err)
	}
	return saveThemeSelection(configDir, source)
}

func saveThemeSelection(configDir, source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("theme selection cannot be empty")
	}
	if _, builtIn := builtInTheme(source); !builtIn &&
		!strings.EqualFold(filepath.Ext(source), ".yaml") {
		return fmt.Errorf("theme selection must be a built-in mode or .yaml path")
	}

	current := loadConfigFile(configDir)
	return writeConfigFile(configDir, AppConfig{
		Version:   ThemeSchemaVersion,
		Theme:     source,
		Motion:    MotionSystem,
		BodyStyle: current.BodyStyle,
	})
}

// SaveBodyStyle persists name (a chroma style name, e.g. "monokai" or
// "onedark" - see responsepane's style picker) as the response pane's Body
// tab syntax highlighting style, in the same config.yaml the general UI
// theme selection lives in (see saveThemeSelection) rather than the
// database - both are the same kind of thing, a persisted display
// preference, not application data.
func SaveBodyStyle(name string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("find user config directory: %w", err)
	}
	return saveBodyStyle(configDir, name)
}

func saveBodyStyle(configDir, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("body style cannot be empty")
	}

	current := loadConfigFile(configDir)
	theme := current.Theme
	if theme == "" {
		// Nothing's ever been saved to config.yaml yet (SaveThemeSelection
		// requires a non-empty theme to round-trip - see
		// loadSavedThemeSelection) - "default" matches DefaultTheme, the
		// same fallback loadUserTheme itself reaches for when nothing's
		// configured.
		theme = "default"
	}
	motion := current.Motion
	if motion == "" {
		motion = MotionSystem
	}
	return writeConfigFile(configDir, AppConfig{
		Version:   ThemeSchemaVersion,
		Theme:     theme,
		Motion:    motion,
		BodyStyle: name,
	})
}

// LoadBodyStyle returns the style name saved by SaveBodyStyle, or ("", nil)
// if none was ever saved (including when config.yaml itself doesn't exist
// yet) - callers (see responsepane.SetupResponsePane) fall back to their
// own default in that case, the same way a missing/unset theme selection
// falls back to DefaultTheme.
func LoadBodyStyle() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	homeDir, _ := os.UserHomeDir()

	for _, path := range configFilePaths(configDir, homeDir) {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		var config AppConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return "", fmt.Errorf("parse %s: %w", path, err)
		}
		return strings.TrimSpace(config.BodyStyle), nil
	}
	return "", nil
}

// loadConfigFile best-effort reads configDir's own config.yaml (not the
// ~/.volt fallback loadSavedThemeSelection/LoadBodyStyle also check - every
// save always writes to configDir, never that fallback), returning a
// zero-value AppConfig if it doesn't exist or fails to parse. Used by
// saveThemeSelection/saveBodyStyle to preserve whichever field the current
// save isn't itself changing.
func loadConfigFile(configDir string) AppConfig {
	path := filepath.Join(configDir, "volt", themeSelectionFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}
	}
	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return AppConfig{}
	}
	return config
}

// writeConfigFile atomically replaces configDir's config.yaml with config -
// shared by saveThemeSelection/saveBodyStyle so a save of one field can
// never race a concurrent save of the other into a half-written file.
func writeConfigFile(configDir string, config AppConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	directory := filepath.Join(configDir, "volt")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create theme config directory: %w", err)
	}

	temp, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary theme config: %w", err)
	}
	tempPath := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("secure temporary theme config: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary theme config: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temporary theme config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary theme config: %w", err)
	}

	path := filepath.Join(directory, themeSelectionFilename)
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("activate theme config: %w", err)
	}
	removeTemp = false
	return nil
}

// configFilePaths lists config.yaml's possible locations, in the order
// they're checked: configDir/volt first, then the ~/.volt fallback -
// shared by loadSavedThemeSelection and LoadBodyStyle so the two can never
// disagree about where a saved selection might be found.
func configFilePaths(configDir, homeDir string) []string {
	paths := make([]string, 0, 2)
	if configDir != "" {
		paths = append(paths, filepath.Join(configDir, "volt", themeSelectionFilename))
	}
	if homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, ".volt", themeSelectionFilename))
	}
	return paths
}

func loadSavedThemeSelection(configDir, homeDir string) (string, string, error, bool) {
	for _, path := range configFilePaths(configDir, homeDir) {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", path, err, true
		}

		var config AppConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return "", path, fmt.Errorf("parse selection: %w", err), true
		}
		if config.Version != ThemeSchemaVersion {
			return "", path, fmt.Errorf(
				"unsupported config version %d; expected %d",
				config.Version,
				ThemeSchemaVersion,
			), true
		}
		selection := strings.TrimSpace(config.Theme)
		if selection == "" {
			return "", path, fmt.Errorf("theme selection is required"), true
		}
		return selection, path, nil, true
	}

	return "", "", nil, false
}
