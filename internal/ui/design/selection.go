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
