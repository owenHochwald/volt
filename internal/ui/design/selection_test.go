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

// TestSaveBodyStyleWritesPrivateYAML verifies saveBodyStyle writes
// BodyStyle into the same config.yaml the theme selection lives in, and
// defaults Theme to "default" when nothing's ever been selected yet
// (config.yaml must round-trip with a non-empty theme - see
// loadSavedThemeSelection).
func TestSaveBodyStyleWritesPrivateYAML(t *testing.T) {
	configDir := t.TempDir()

	assert.NoError(t, saveBodyStyle(configDir, "dracula"))

	path := filepath.Join(configDir, "volt", themeSelectionFilename)
	data, err := os.ReadFile(path)
	assert.NoError(t, err)

	var config AppConfig
	assert.NoError(t, yaml.Unmarshal(data, &config))
	assert.Equal(t, ThemeSchemaVersion, config.Version)
	assert.Equal(t, "default", config.Theme)
	assert.Equal(t, "dracula", config.BodyStyle)
}

// TestSaveBodyStylePreservesExistingTheme and
// TestSaveThemeSelectionPreservesExistingBodyStyle are regression tests
// for the two save paths sharing one file: saving one of Theme/BodyStyle
// must never clobber whichever the other one already had.
func TestSaveBodyStylePreservesExistingTheme(t *testing.T) {
	configDir := t.TempDir()
	assert.NoError(t, saveThemeSelection(configDir, "mono"))
	assert.NoError(t, saveBodyStyle(configDir, "nord"))

	config := loadConfigFile(configDir)
	assert.Equal(t, "mono", config.Theme)
	assert.Equal(t, "nord", config.BodyStyle)
}

func TestSaveThemeSelectionPreservesExistingBodyStyle(t *testing.T) {
	configDir := t.TempDir()
	assert.NoError(t, saveBodyStyle(configDir, "nord"))
	assert.NoError(t, saveThemeSelection(configDir, "mono"))

	config := loadConfigFile(configDir)
	assert.Equal(t, "mono", config.Theme)
	assert.Equal(t, "nord", config.BodyStyle)
}

// TestLoadBodyStyle_RoundTrips exercises the real, exported
// SaveBodyStyle/LoadBodyStyle pair (unlike the unexported tests above,
// which inject configDir directly) - redirecting os.UserConfigDir via
// XDG_CONFIG_HOME so it never touches the real machine's config.
func TestLoadBodyStyle_RoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	got, err := LoadBodyStyle()
	assert.NoError(t, err)
	assert.Equal(t, "", got)

	assert.NoError(t, SaveBodyStyle("dracula"))
	got, err = LoadBodyStyle()
	assert.NoError(t, err)
	assert.Equal(t, "dracula", got)

	assert.NoError(t, SaveBodyStyle("nord"))
	got, err = LoadBodyStyle()
	assert.NoError(t, err)
	assert.Equal(t, "nord", got)
}
