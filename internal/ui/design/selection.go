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

	data, err := yaml.Marshal(AppConfig{
		Version: ThemeSchemaVersion,
		Theme:   source,
		Motion:  MotionSystem,
	})
	if err != nil {
		return fmt.Errorf("encode theme selection: %w", err)
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

func loadSavedThemeSelection(configDir, homeDir string) (string, string, error, bool) {
	paths := make([]string, 0, 2)
	if configDir != "" {
		paths = append(paths, filepath.Join(configDir, "volt", themeSelectionFilename))
	}
	if homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, ".volt", themeSelectionFilename))
	}

	for _, path := range paths {
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
