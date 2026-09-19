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

// TestLoadAutoRememberCookies_DefaultsToFalse verifies a fresh install (no
// config.yaml at all) never starts silently remembering session state.
func TestLoadAutoRememberCookies_DefaultsToFalse(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	got, err := LoadAutoRememberCookies()
	assert.NoError(t, err)
	assert.False(t, got)
}

func TestAutoRememberCookies_RoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	assert.NoError(t, SaveAutoRememberCookies(true))
	got, err := LoadAutoRememberCookies()
	assert.NoError(t, err)
	assert.True(t, got)

	assert.NoError(t, SaveAutoRememberCookies(false))
	got, err = LoadAutoRememberCookies()
	assert.NoError(t, err)
	assert.False(t, got)
}

func TestAutoRememberAuthHeaders_RoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	assert.NoError(t, SaveAutoRememberAuthHeaders(true))
	got, err := LoadAutoRememberAuthHeaders()
	assert.NoError(t, err)
	assert.True(t, got)

	assert.NoError(t, SaveAutoRememberAuthHeaders(false))
	got, err = LoadAutoRememberAuthHeaders()
	assert.NoError(t, err)
	assert.False(t, got)
}

// TestAutoRememberCookies_ToggleDoesNotAffectAuthHeaders (and its mirror)
// are regression tests for the two flags sharing one config.yaml, the
// same class of bug TestSaveBodyStylePreservesExistingTheme guards
// against: saving one flag must never clobber the other, BodyStyle, or
// Theme.
func TestAutoRememberCookies_ToggleDoesNotAffectAuthHeaders(t *testing.T) {
	configDir := t.TempDir()
	assert.NoError(t, saveThemeSelection(configDir, "mono"))
	assert.NoError(t, saveBodyStyle(configDir, "nord"))
	assert.NoError(t, saveAutoRememberAuthHeaders(configDir, true))

	assert.NoError(t, saveAutoRememberCookies(configDir, true))

	config := loadConfigFile(configDir)
	assert.Equal(t, "mono", config.Theme)
	assert.Equal(t, "nord", config.BodyStyle)
	assert.True(t, config.AutoRememberAuthHeaders)
	assert.True(t, config.AutoRememberCookies)
}

func TestAutoRememberAuthHeaders_ToggleDoesNotAffectCookies(t *testing.T) {
	configDir := t.TempDir()
	assert.NoError(t, saveAutoRememberCookies(configDir, true))

	assert.NoError(t, saveAutoRememberAuthHeaders(configDir, true))

	config := loadConfigFile(configDir)
	assert.True(t, config.AutoRememberCookies)
	assert.True(t, config.AutoRememberAuthHeaders)
}
