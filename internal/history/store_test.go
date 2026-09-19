package history

import (
	"testing"
	"time"

	"github.com/owenHochwald/Volt/internal/http"
)

func TestStore_Record_ReturnsIncreasingIDs(t *testing.T) {
	s := NewStore()
	id1 := s.Record(&http.Request{URL: "https://a"}, &http.Response{StatusCode: 200}, time.Now())
	id2 := s.Record(&http.Request{URL: "https://b"}, &http.Response{StatusCode: 200}, time.Now())
	if id2 <= id1 {
		t.Errorf("expected increasing IDs, got %d then %d", id1, id2)
	}
}

func TestStore_Entries_NewestFirst(t *testing.T) {
	s := NewStore()
	s.Record(&http.Request{URL: "https://first"}, &http.Response{}, time.Now())
	s.Record(&http.Request{URL: "https://second"}, &http.Response{}, time.Now())

	entries := s.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].SentRequest.URL != "https://second" {
		t.Errorf("expected the most recent entry first, got %+v", entries[0])
	}
	if entries[1].SentRequest.URL != "https://first" {
		t.Errorf("expected the oldest entry last, got %+v", entries[1])
	}
}

func TestStore_EvictsOldestWhenOverCapacity(t *testing.T) {
	s := NewStoreWithCapacity(2)
	s.Record(&http.Request{URL: "https://a"}, &http.Response{}, time.Now())
	s.Record(&http.Request{URL: "https://b"}, &http.Response{}, time.Now())
	s.Record(&http.Request{URL: "https://c"}, &http.Response{}, time.Now())

	entries := s.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected capacity to cap at 2 entries, got %d", len(entries))
	}
	if entries[0].SentRequest.URL != "https://c" || entries[1].SentRequest.URL != "https://b" {
		t.Errorf("expected [c b] (newest first, a evicted), got %+v", entries)
	}
}

func TestStore_Delete_RemovesOnlyThatEntry(t *testing.T) {
	s := NewStore()
	id1 := s.Record(&http.Request{URL: "https://a"}, &http.Response{}, time.Now())
	id2 := s.Record(&http.Request{URL: "https://b"}, &http.Response{}, time.Now())

	s.Delete(id1)

	entries := s.Entries()
	if len(entries) != 1 || entries[0].ID != id2 {
		t.Errorf("expected only the other entry to remain, got %+v", entries)
	}
}

func TestStore_Clear_RemovesEverything(t *testing.T) {
	s := NewStore()
	s.Record(&http.Request{URL: "https://a"}, &http.Response{}, time.Now())
	s.Record(&http.Request{URL: "https://b"}, &http.Response{}, time.Now())

	s.Clear()

	if entries := s.Entries(); len(entries) != 0 {
		t.Errorf("expected no entries after Clear, got %+v", entries)
	}
}

// TestStore_NilStore_IsSafe verifies every method is a safe no-op on a
// nil *Store, matching authjar.Store's own convention.
func TestStore_NilStore_IsSafe(t *testing.T) {
	var s *Store

	if id := s.Record(&http.Request{}, &http.Response{}, time.Now()); id != 0 {
		t.Errorf("expected Record on a nil Store to return 0, got %d", id)
	}
	s.Delete(1)
	s.Clear()

	if entries := s.Entries(); entries != nil {
		t.Errorf("expected nil entries from a nil Store, got %+v", entries)
	}
}
