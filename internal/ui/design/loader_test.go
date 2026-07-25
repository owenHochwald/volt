package design

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
)

func TestThemeLoaderUsesDocumentedPathOrder(t *testing.T) {
	configDir := t.TempDir()
	homeDir := t.TempDir()
	writeThemeFile(t, filepath.Join(configDir, "volt", "theme.yaml"), "#111111")
	writeThemeFile(t, filepath.Join(homeDir, ".volt", "theme.yaml"), "#222222")

	result := loadUserTheme(themeLoadOptions{
		ConfigDir: configDir,
		HomeDir:   homeDir,
	})

	assert.Equal(t, filepath.Join(configDir, "volt", "theme.yaml"), result.Source)
	assert.Equal(t, lipgloss.Color("#111111"), result.Theme.Colors.Brand)
	assert.Equal(t, "", result.Warning)
}

func TestThemeLoaderFallsBackToLegacyHomePath(t *testing.T) {
	configDir := t.TempDir()
	homeDir := t.TempDir()
	path := filepath.Join(homeDir, ".volt", "theme.yaml")
	writeThemeFile(t, path, "#222222")

	result := loadUserTheme(themeLoadOptions{
		ConfigDir: configDir,
		HomeDir:   homeDir,
	})

	assert.Equal(t, path, result.Source)
	assert.Equal(t, lipgloss.Color("#222222"), result.Theme.Colors.Brand)
	assert.Equal(t, "", result.Warning)
}

func TestThemeLoaderFallsBackWithUsefulWarning(t *testing.T) {
	configDir := t.TempDir()
	path := filepath.Join(configDir, "volt", "theme.yaml")
	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoError(t, os.WriteFile(path, []byte(`
version: 1
name: Broken
colors:
  brand: purple
`), 0o600))

	result := loadUserTheme(themeLoadOptions{ConfigDir: configDir})

	assert.Equal(t, "default", result.Source)
	assert.Equal(t, DefaultTheme(), result.Theme)
	assert.True(t, strings.Contains(result.Warning, path))
	assert.True(t, strings.Contains(result.Warning, "invalid ANSI-256 color"))
}

func TestThemeLoaderMissingFilesAreQuiet(t *testing.T) {
	result := loadUserTheme(themeLoadOptions{
		ConfigDir: t.TempDir(),
		HomeDir:   t.TempDir(),
	})

	assert.Equal(t, "default", result.Source)
	assert.Equal(t, DefaultTheme(), result.Theme)
	assert.Equal(t, "", result.Warning)
}

func TestThemeLoaderExplicitYAMLWins(t *testing.T) {
	configDir := t.TempDir()
	explicitPath := filepath.Join(t.TempDir(), "machine.yaml")
	writeThemeFile(t, filepath.Join(configDir, "volt", "theme.yaml"), "#111111")
	writeThemeFile(t, explicitPath, "#333333")

	result := loadUserTheme(themeLoadOptions{
		Selection: explicitPath,
		ConfigDir: configDir,
	})

	assert.Equal(t, explicitPath, result.Source)
	assert.Equal(t, lipgloss.Color("#333333"), result.Theme.Colors.Brand)
}

func TestThemeLoaderRejectsUnsupportedExplicitFormat(t *testing.T) {
	result := loadUserTheme(themeLoadOptions{Selection: "theme.json"})

	assert.Equal(t, DefaultTheme(), result.Theme)
	assert.True(t, strings.Contains(result.Warning, ".yaml"))
}

func writeThemeFile(t *testing.T, path, brand string) {
	t.Helper()
	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	content := []byte("version: 1\nname: Test\ncolors:\n  brand: \"" + brand + "\"\n")
	assert.NoError(t, os.WriteFile(path, content, 0o600))
}
