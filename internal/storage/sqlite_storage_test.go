package storage

import (
	"testing"

	_ "modernc.org/sqlite"

	"github.com/alecthomas/assert/v2"
	"github.com/owenHochwald/Volt/internal/http"
)

func setupTestDB(t *testing.T) *SQLiteStorage {
	t.Helper()

	// creation and teardown of the database
	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store
}

func TestSQLiteStorage_SaveLoadDelete(t *testing.T) {
	db := setupTestDB(t)

	req := &http.Request{
		Name:   "test",
		Method: "GET",
		URL:    "http://localhost:8080",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: "test",
	}

	err := db.Save(req)

	assert.NoError(t, err)
	assert.NotEqual(t, 0, req.ID, "Saving should set ID")

	requests, err := db.Load()

	assert.NoError(t, err)
	assert.Equal(t, 1, len(requests))
	assert.Equal(t, requests[0], *req)
	assert.NotEqual(t, 0, requests[0].ID)

	err = db.Delete(req.ID)
	assert.NoError(t, err)

	requests, err = db.Load()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(requests))
}

// TestSQLiteStorage_UpdateRequestSchema verifies UpdateRequestSchema (used
// by `volt import -force`) refreshes name/headers/body/path param schema
// in place - same row id, no new row created - and leaves
// method/url/group_id/server_id alone, since those are either the match
// identity or a user-controlled association unrelated to a spec's content.
func TestSQLiteStorage_UpdateRequestSchema(t *testing.T) {
	db := setupTestDB(t)

	groupID := int64(7)
	req := &http.Request{
		Name:    "old name",
		Method:  "GET",
		URL:     "http://localhost:8080/widgets/{id}",
		Headers: map[string]string{"X-Old": "1"},
		Body:    "old body",
		GroupID: &groupID,
	}
	assert.NoError(t, db.Save(req))
	originalID := req.ID

	newPathSchema := map[string]http.BodyFieldSchema{"id": {Type: "integer", Required: true}}
	err := db.UpdateRequestSchema(originalID, "new name", map[string]string{"X-New": "2"}, "new body", newPathSchema)
	assert.NoError(t, err)

	requests, err := db.Load()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(requests), "update must not create a new row")

	updated := requests[0]
	assert.Equal(t, originalID, updated.ID)
	assert.Equal(t, "new name", updated.Name)
	assert.Equal(t, "new body", updated.Body)
	assert.Equal(t, map[string]string{"X-New": "2"}, updated.Headers)
	assert.Equal(t, newPathSchema, updated.PathParamSchema)
	// Untouched by design.
	assert.Equal(t, "GET", updated.Method)
	assert.Equal(t, "http://localhost:8080/widgets/{id}", updated.URL)
	assert.Equal(t, groupID, *updated.GroupID)
}

// TestSQLiteStorage_LastSentRequest_RoundTrips verifies
// SaveLastSentRequest/LoadLastSentRequest preserve the full snapshot -
// base request template plus every field value - byte-for-byte.
func TestSQLiteStorage_LastSentRequest_RoundTrips(t *testing.T) {
	db := setupTestDB(t)

	snapshot := &http.RequestSnapshot{
		Request: http.Request{
			Method:  "POST",
			URL:     "https://api.example.com/hosts/{host_id}",
			Headers: map[string]string{"X-Env": "prod"},
			PathParamSchema: map[string]http.BodyFieldSchema{
				"host_id": {Type: "integer", Required: true},
			},
		},
		PathParamValues: map[string]string{"host_id": "42"},
		BodyParamValues: map[string]string{"name": "widget"},
		ArrayParamValues: map[string][]map[string]string{
			"tags": {{"key": "env", "value": "prod"}},
		},
	}

	assert.NoError(t, db.SaveLastSentRequest(snapshot))

	got, err := db.LoadLastSentRequest()
	assert.NoError(t, err)
	if got == nil {
		t.Fatal("expected a non-nil snapshot")
	}
	assert.Equal(t, snapshot.Request.Method, got.Request.Method)
	assert.Equal(t, snapshot.Request.URL, got.Request.URL)
	assert.Equal(t, snapshot.PathParamValues, got.PathParamValues)
	assert.Equal(t, snapshot.BodyParamValues, got.BodyParamValues)
	assert.Equal(t, snapshot.ArrayParamValues, got.ArrayParamValues)
}

// TestSQLiteStorage_LastSentRequest_NoneSavedYet verifies a fresh database
// (nothing ever sent) returns (nil, nil), not an error - the ordinary
// starting state, not a failure.
func TestSQLiteStorage_LastSentRequest_NoneSavedYet(t *testing.T) {
	db := setupTestDB(t)

	got, err := db.LoadLastSentRequest()
	assert.NoError(t, err)
	if got != nil {
		t.Errorf("expected nil when nothing's been sent yet, got %+v", got)
	}
}

// TestSQLiteStorage_ClearLastSentRequest_RemovesIt verifies
// ClearLastSentRequest (used when the request pane is reset to blank -
// see requestpane's resetToBlank) makes LoadLastSentRequest go back to
// returning (nil, nil), so a discarded request doesn't reappear on the
// next app start.
func TestSQLiteStorage_ClearLastSentRequest_RemovesIt(t *testing.T) {
	db := setupTestDB(t)

	assert.NoError(t, db.SaveLastSentRequest(&http.RequestSnapshot{Request: http.Request{Method: "GET", URL: "https://a.example.com"}}))

	got, err := db.LoadLastSentRequest()
	assert.NoError(t, err)
	if got == nil {
		t.Fatal("test setup error: expected a saved snapshot before clearing it")
	}

	assert.NoError(t, db.ClearLastSentRequest())

	got, err = db.LoadLastSentRequest()
	assert.NoError(t, err)
	if got != nil {
		t.Errorf("expected nil after clearing, got %+v", got)
	}
}

// TestSQLiteStorage_ClearLastSentRequest_NoopWhenNothingSaved verifies
// clearing when nothing was ever saved is a no-op, not an error.
func TestSQLiteStorage_ClearLastSentRequest_NoopWhenNothingSaved(t *testing.T) {
	db := setupTestDB(t)
	assert.NoError(t, db.ClearLastSentRequest())
}

// TestSQLiteStorage_LastSentRequest_OverwritesPrevious verifies only the
// most recent send is kept - saving a second snapshot replaces the first
// rather than accumulating a history.
func TestSQLiteStorage_LastSentRequest_OverwritesPrevious(t *testing.T) {
	db := setupTestDB(t)

	assert.NoError(t, db.SaveLastSentRequest(&http.RequestSnapshot{Request: http.Request{Method: "GET", URL: "https://a.example.com"}}))
	assert.NoError(t, db.SaveLastSentRequest(&http.RequestSnapshot{Request: http.Request{Method: "POST", URL: "https://b.example.com"}}))

	got, err := db.LoadLastSentRequest()
	assert.NoError(t, err)
	if got == nil {
		t.Fatal("expected a non-nil snapshot")
	}
	assert.Equal(t, "POST", got.Request.Method)
	assert.Equal(t, "https://b.example.com", got.Request.URL)
}

func TestSQLiteStorage_SaveAndLoad(t *testing.T) {
	db := setupTestDB(t)

	requests, err := db.Load()

	assert.NoError(t, err)
	assert.Equal(t, len(requests), 0)
}

func TestSQLiteStorage_DeleteNonExistent(t *testing.T) {
	db := setupTestDB(t)

	err := db.Delete(999)
	assert.Error(t, err)

}
func TestSQLiteStorage_MultipleRequests(t *testing.T) {
	db := setupTestDB(t)

	requests := []*http.Request{
		{
			Name:   "test1",
			Method: "GET",
			URL:    "http://localhost:8080",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "test",
		},
		{
			Name:    "test2",
			Method:  "POST",
			URL:     "http://localhost:8080",
			Headers: map[string]string{},
		},
		{
			Name:    "test3",
			Method:  "PUT",
			URL:     "http://broken;??/asd",
			Headers: map[string]string{},
		},
	}

	for _, req := range requests {
		if err := db.Save(req); err != nil {
			t.Fatalf("failed to save request: %v", err)
		}
	}

	loaded, err := db.Load()

	assert.Equal(t, len(requests), len(loaded))
	assert.NoError(t, err)
	for _, req := range requests {
		assert.SliceContains(t, loaded, *req)
	}
}

func TestSQLiteStorage_GetAllURLs_MultipleURLs(t *testing.T) {
	db := setupTestDB(t)
	numUniqueRequests := 4

	requests := []*http.Request{
		{
			Name:   "test1",
			Method: "GET",
			URL:    "http://localhost:8080",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "test",
		},
		{
			Name:   "Not a distinct URL",
			Method: "GET",
			URL:    "http://localhost:8080",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "test",
		},
		{
			Name:   "Mostly the same URL, but still distinct",
			Method: "GET",
			URL:    "http://localhost:9090/test",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: "test",
		},
		{
			Name:    "test2",
			Method:  "POST",
			URL:     "http://google.com/api",
			Headers: map[string]string{},
		},
		{
			Name:    "test2",
			Method:  "POST",
			URL:     "http://google.com/api",
			Headers: map[string]string{},
		},
		{
			Name:    "test3",
			Method:  "PUT",
			URL:     "http://broken;??/asd",
			Headers: map[string]string{},
		},
	}

	for _, req := range requests {
		if err := db.Save(req); err != nil {
			t.Fatalf("failed to save request: %v", err)
		}
	}

	urls, err := db.GetAllURLs()

	assert.Equal(t, numUniqueRequests, len(urls))
	assert.NoError(t, err)
	for _, req := range requests {
		assert.SliceContains(t, urls, req.URL)
	}
}

func TestSQLiteStorage_GetAllURLs_NoURLs(t *testing.T) {
	db := setupTestDB(t)

	requests := []*http.Request{}

	for _, req := range requests {
		if err := db.Save(req); err != nil {
			t.Fatalf("failed to save request: %v", err)
		}
	}

	urls, err := db.GetAllURLs()

	assert.Equal(t, len(requests), len(urls))
	assert.NoError(t, err)
}
