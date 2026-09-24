package requestpane

import (
	"slices"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

// TestRefreshURLSuggestions_PicksUpNewlySavedURL is a regression test for a
// real bug report: a URL saved during the current session never showed up
// as an autocomplete suggestion (see NewURLInput) until the app was
// restarted, since URLInput's suggestion list was only ever built once,
// at startup.
func TestRefreshURLSuggestions_PicksUpNewlySavedURL(t *testing.T) {
	pane, db := setupTestPane(t)

	const freshURL = "https://api.example.com/just-saved"
	if slices.Contains(pane.URLInput.AvailableSuggestions(), freshURL) {
		t.Fatal("test setup error: the URL shouldn't be a suggestion before it's even saved")
	}

	if err := db.Save(&http.Request{Method: http.GET, URL: freshURL}); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	pane.RefreshURLSuggestions()

	if !slices.Contains(pane.URLInput.AvailableSuggestions(), freshURL) {
		t.Errorf("expected %q to be a suggestion after RefreshURLSuggestions, got %v", freshURL, pane.URLInput.AvailableSuggestions())
	}
}

// TestRefreshURLSuggestions_NilDBIsANoop matches this package's existing
// "nil DB is a no-op" convention (see e.g. resolvedRequest's own DB nil
// check) - a pane never wired up with a real database (most other tests)
// must not panic when this is called.
func TestRefreshURLSuggestions_NilDBIsANoop(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.DB = nil

	pane.RefreshURLSuggestions()
}
