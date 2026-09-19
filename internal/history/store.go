// Package history remembers a bounded number of past sends (each a
// request paired with the response it got back) in memory only, for the
// lifetime of the process - never persisted to disk or the database (see
// the response pane's "H" viewer, which is what actually browses it).
package history

import (
	"sync"
	"time"

	"github.com/owenHochwald/Volt/internal/http"
)

// Entry is one past send - the fully-resolved request that was sent
// (path/query params substituted, server headers/variables merged in,
// absolute URL - the same shape responsepane.ResponsePane.SentRequest
// already carries) paired with the response it got back.
type Entry struct {
	ID          int64
	SentRequest *http.Request
	Response    *http.Response
	SentAt      time.Time
}

// defaultCapacity bounds how many entries Store keeps - a normal send
// (never a load test's own thousands of requests - see Store's own
// package doc) is added one at a time interactively, so 100 is enough to
// browse back through a real debugging session without the history
// growing unbounded for the life of a long-running app instance.
const defaultCapacity = 100

// Store is an in-memory, concurrency-safe, fixed-capacity ring of past
// sends, newest first. Every method is nil-receiver-safe, matching this
// codebase's existing "nil db/jar is a no-op" convention, so callers that
// haven't been given a real Store (most tests) need no nil check of
// their own.
type Store struct {
	mu       sync.Mutex
	capacity int
	nextID   int64
	// entries is kept newest-first (index 0 is the most recent) - Record
	// prepends, Entries returns a copy in that same order.
	entries []Entry
}

// NewStore returns an empty Store holding up to defaultCapacity entries.
func NewStore() *Store {
	return NewStoreWithCapacity(defaultCapacity)
}

// NewStoreWithCapacity is NewStore with an explicit capacity - split out
// mainly so tests can use a small one without recording 100 entries to
// exercise eviction.
func NewStoreWithCapacity(capacity int) *Store {
	return &Store{capacity: capacity}
}

// Record adds a new entry as the newest one, evicting the oldest if the
// store is already at capacity. Returns the ID assigned to it (see
// Entry.ID), e.g. for a future "diff two entries" feature to reference it
// by.
func (s *Store) Record(sentRequest *http.Request, response *http.Response, sentAt time.Time) int64 {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	entry := Entry{ID: s.nextID, SentRequest: sentRequest, Response: response, SentAt: sentAt}
	s.entries = append([]Entry{entry}, s.entries...)
	if s.capacity > 0 && len(s.entries) > s.capacity {
		s.entries = s.entries[:s.capacity]
	}
	return entry.ID
}

// Entries returns every recorded entry, newest first.
func (s *Store) Entries() []Entry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Entry(nil), s.entries...)
}

// Delete removes the entry with the given ID, if any.
func (s *Store) Delete(id int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.entries {
		if e.ID == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return
		}
	}
}

// Clear removes every recorded entry.
func (s *Store) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
}
