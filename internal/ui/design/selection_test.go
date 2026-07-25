package design

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"gopkg.in/yaml.v3"
)

func TestSaveThemeSelectionWritesPrivateYAML(t *testing.T) {
	configDir := t.TempDir()

	assert.NoError(t, saveThemeSelection(configDir, "adaptive"))

	path := filepath.Join(configDir, "volt", themeSelectionFilename)
	data, err := os.ReadFile(path)
	assert.NoError(t, err)

	var config AppConfig
	assert.NoError(t, yaml.Unmarshal(data, &config))
	assert.Equal(t, ThemeSchemaVersion, config.Version)
	assert.Equal(t, "adaptive", config.Theme)
	assert.Equal(t, MotionSystem, config.Motion)

	info, err := os.Stat(path)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSaveThemeSelectionAtomicallyReplacesValue(t *testing.T) {
	configDir := t.TempDir()
	assert.NoError(t, saveThemeSelection(configDir, "default"))
	assert.NoError(t, saveThemeSelection(configDir, "mono"))

	path := filepath.Join(configDir, "volt", themeSelectionFilename)
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "theme: mono")

	entries, err := os.ReadDir(filepath.Dir(path))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, themeSelectionFilename, entries[0].Name())
}

func TestSaveThemeSelectionRejectsUnsupportedValue(t *testing.T) {
	err := saveThemeSelection(t.TempDir(), "theme.json")
	assert.Error(t, err)
}
