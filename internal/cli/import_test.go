package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
)

func setupTestDB(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func writeSpec(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}
	return path
}

func TestRunImport_SavesAndDedupes(t *testing.T) {
	db := setupTestDB(t)
	specPath := writeSpec(t, samplePetstoreYAML)

	summary, err := RunImport(specPath, "", "", false, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Imported != 5 {
		t.Errorf("expected 5 imported, got %d", summary.Imported)
	}
	if summary.Skipped != 0 {
		t.Errorf("expected 0 skipped on first import, got %d", summary.Skipped)
	}

	saved, err := db.Load()
	if err != nil {
		t.Fatalf("failed to load saved requests: %v", err)
	}
	if len(saved) != 5 {
		t.Fatalf("expected 5 rows in db, got %d", len(saved))
	}

	// Re-importing the same spec must not create duplicates.
	summary, err = RunImport(specPath, "", "", false, db)
	if err != nil {
		t.Fatalf("unexpected error on re-import: %v", err)
	}
	if summary.Imported != 0 || summary.Skipped != 5 {
		t.Errorf("expected re-import to skip all 5, got imported=%d skipped=%d", summary.Imported, summary.Skipped)
	}

	saved, err = db.Load()
	if err != nil {
		t.Fatalf("failed to load saved requests: %v", err)
	}
	if len(saved) != 5 {
		t.Fatalf("expected still 5 rows after re-import, got %d", len(saved))
	}
}

func TestRunImport_MissingFile(t *testing.T) {
	db := setupTestDB(t)
	if _, err := RunImport("/does/not/exist.yaml", "", "", false, db); err == nil {
		t.Fatal("expected an error for a missing spec file")
	}
}

func TestRunImport_UngroupedBaseURLIsSavedAsDefaultServer(t *testing.T) {
	db := setupTestDB(t)
	specPath := writeSpec(t, samplePetstoreYAML)

	if _, err := RunImport(specPath, "http://localhost:9000", "", false, db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	server, err := db.GetServerByName(defaultServerName)
	if err != nil || server == nil {
		t.Fatalf("expected a %q server to be saved, err=%v", defaultServerName, err)
	}
	if server.BaseURL != "http://localhost:9000" {
		t.Errorf("expected base URL http://localhost:9000, got %q", server.BaseURL)
	}

	// Requests themselves stay absolute (ungrouped import), unaffected by
	// the server bookkeeping.
	saved, err := db.Load()
	if err != nil {
		t.Fatalf("failed to load saved requests: %v", err)
	}
	for _, r := range saved {
		if !strings.HasPrefix(r.URL, "http://localhost:9000") {
			t.Errorf("expected an absolute request URL, got %q", r.URL)
		}
	}
}

func TestRunImport_GroupStoresRelativeURLsAndAdoptsSpecServer(t *testing.T) {
	db := setupTestDB(t)
	specPath := writeSpec(t, samplePetstoreYAML) // declares servers: https://api.example.com/v1

	summary, err := RunImport(specPath, "", "petstore", false, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Imported != 5 {
		t.Fatalf("expected 5 imported, got %d", summary.Imported)
	}

	group, err := db.GetGroupByName("petstore")
	if err != nil || group == nil {
		t.Fatalf("expected group %q to exist, err=%v", "petstore", err)
	}
	if group.ServerID == nil {
		t.Fatal("expected group to have adopted the spec's server")
	}

	server, err := db.GetServerByName("petstore")
	if err != nil || server == nil {
		t.Fatalf("expected an auto-created server named %q, err=%v", "petstore", err)
	}
	if server.BaseURL != "https://api.example.com/v1" {
		t.Errorf("expected adopted base URL https://api.example.com/v1, got %q", server.BaseURL)
	}

	saved, err := db.Load()
	if err != nil {
		t.Fatalf("failed to load saved requests: %v", err)
	}
	for _, r := range saved {
		if r.IsAbsoluteURL() {
			t.Errorf("expected a relative URL for grouped import, got %q", r.URL)
		}
		if r.GroupID == nil || *r.GroupID != group.ID {
			t.Errorf("expected request to belong to group %d, got %v", group.ID, r.GroupID)
		}
		resolved, err := db.ResolveEffectiveURL(&r)
		if err != nil {
			t.Fatalf("failed to resolve effective URL: %v", err)
		}
		if !strings.HasPrefix(resolved, "https://api.example.com/v1") {
			t.Errorf("expected resolved URL to use the group's server, got %q", resolved)
		}
	}

	// Switching the group's server should redirect all its requests at once.
	if _, err := db.UpsertServer("petstore", "http://localhost:9000"); err != nil {
		t.Fatalf("failed to update server: %v", err)
	}
	resolved, err := db.ResolveEffectiveURL(&saved[0])
	if err != nil {
		t.Fatalf("failed to resolve effective URL after server switch: %v", err)
	}
	if !strings.HasPrefix(resolved, "http://localhost:9000") {
		t.Errorf("expected request to follow the group's new server, got %q", resolved)
	}
}

// TestRunImport_ForceOverwritesExistingRequests verifies `volt import
// -force` refreshes already-saved requests (name/headers/body) from the
// spec's current content instead of skipping them - the fix for a real gap:
// without force, reimporting a spec with minor changes (a new required
// field, a renamed summary, etc.) left every matched request's stale
// original schema untouched forever, since Save is INSERT-only and there
// was no update path at all before this.
func TestRunImport_ForceOverwritesExistingRequests(t *testing.T) {
	db := setupTestDB(t)
	specPath := writeSpec(t, samplePetstoreYAML)

	if _, err := RunImport(specPath, "", "", false, db); err != nil {
		t.Fatalf("unexpected error on initial import: %v", err)
	}

	saved, err := db.Load()
	if err != nil {
		t.Fatalf("failed to load saved requests: %v", err)
	}
	var photoReq *http.Request
	for i := range saved {
		if saved[i].Method == http.PUT {
			photoReq = &saved[i]
		}
	}
	if photoReq == nil {
		t.Fatal("expected the PUT /pets/{petId}/photos request to exist")
	}
	if photoReq.Name != "Upload a pet photo" {
		t.Fatalf("expected initial name %q, got %q", "Upload a pet photo", photoReq.Name)
	}

	// A changed spec - same paths/methods (so it dedup-matches), but a
	// different summary text on the one operation with no operationId (so
	// its Name is spec-derived - see operationName - and will visibly
	// change if, and only if, force actually updates it).
	changedYAML := strings.Replace(samplePetstoreYAML, "summary: Upload a pet photo", "summary: Upload a photo v2", 1)
	changedSpecPath := writeSpec(t, changedYAML)

	// Without force: still skipped, name stays stale.
	summary, err := RunImport(changedSpecPath, "", "", false, db)
	if err != nil {
		t.Fatalf("unexpected error on non-force reimport: %v", err)
	}
	if summary.Imported != 0 || summary.Updated != 0 || summary.Skipped != 5 {
		t.Errorf("expected a non-force reimport to skip all 5 unchanged, got imported=%d updated=%d skipped=%d", summary.Imported, summary.Updated, summary.Skipped)
	}
	saved, err = db.Load()
	if err != nil {
		t.Fatalf("failed to load saved requests: %v", err)
	}
	for i := range saved {
		if saved[i].Method == http.PUT && saved[i].Name != "Upload a pet photo" {
			t.Errorf("expected a non-force reimport to leave the stale name alone, got %q", saved[i].Name)
		}
	}

	// With force: same 5 requests match by group+method+url, so nothing new
	// is Imported and nothing is Skipped - all 5 are Updated in place.
	summary, err = RunImport(changedSpecPath, "", "", true, db)
	if err != nil {
		t.Fatalf("unexpected error on force reimport: %v", err)
	}
	if summary.Imported != 0 || summary.Updated != 5 || summary.Skipped != 0 {
		t.Errorf("expected a force reimport to update all 5, got imported=%d updated=%d skipped=%d", summary.Imported, summary.Updated, summary.Skipped)
	}

	saved, err = db.Load()
	if err != nil {
		t.Fatalf("failed to load saved requests: %v", err)
	}
	if len(saved) != 5 {
		t.Fatalf("expected still exactly 5 rows after a force reimport (no duplicates), got %d", len(saved))
	}
	var updatedPhotoReq *http.Request
	for i := range saved {
		if saved[i].Method == http.PUT {
			updatedPhotoReq = &saved[i]
		}
	}
	if updatedPhotoReq == nil {
		t.Fatal("expected the PUT /pets/{petId}/photos request to still exist")
	}
	if updatedPhotoReq.Name != "Upload a photo v2" {
		t.Errorf("expected force reimport to refresh the name to %q, got %q", "Upload a photo v2", updatedPhotoReq.Name)
	}
	// Identity (id/method/url) must survive a force update unchanged - it's
	// an in-place refresh, not a delete+reinsert.
	if updatedPhotoReq.ID != photoReq.ID {
		t.Errorf("expected force update to keep the same row id (%d), got %d - looks like a duplicate insert instead of an update", photoReq.ID, updatedPhotoReq.ID)
	}
	if updatedPhotoReq.URL != photoReq.URL {
		t.Errorf("expected URL to stay %q, got %q", photoReq.URL, updatedPhotoReq.URL)
	}
}

func TestRunImport_GroupDedupIsIndependentOfUngrouped(t *testing.T) {
	db := setupTestDB(t)
	specPath := writeSpec(t, samplePetstoreYAML)

	// Import once ungrouped (absolute URLs from the spec's own server)...
	if _, err := RunImport(specPath, "", "", false, db); err != nil {
		t.Fatalf("unexpected error on ungrouped import: %v", err)
	}
	// ...and once into a group (relative URLs). Same paths, different
	// group scope, so nothing should be skipped as a duplicate.
	summary, err := RunImport(specPath, "", "petstore", false, db)
	if err != nil {
		t.Fatalf("unexpected error on grouped import: %v", err)
	}
	if summary.Imported != 5 || summary.Skipped != 0 {
		t.Errorf("expected the grouped import to be independent of the ungrouped one, got imported=%d skipped=%d", summary.Imported, summary.Skipped)
	}
}
