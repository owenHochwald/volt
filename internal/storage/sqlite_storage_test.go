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
