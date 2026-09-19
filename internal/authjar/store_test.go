package authjar

import (
	"net/http"
	"testing"
)

func cookies(pairs ...[2]string) []*http.Cookie {
	out := make([]*http.Cookie, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, &http.Cookie{Name: p[0], Value: p[1]})
	}
	return out
}

func TestStore_RememberCookies_BuildsCookieHeaderSortedByName(t *testing.T) {
	s := NewStore()
	s.RememberCookies("api.example.com", cookies([2]string{"session", "abc"}, [2]string{"csrftoken", "xyz"}))

	got := s.HeadersFor("api.example.com")
	if got["Cookie"] != "csrftoken=xyz; session=abc" {
		t.Errorf("Cookie header = %q, want %q", got["Cookie"], "csrftoken=xyz; session=abc")
	}
}

func TestStore_RememberCookies_OverwritesSameName(t *testing.T) {
	s := NewStore()
	s.RememberCookies("api.example.com", cookies([2]string{"session", "old"}))
	s.RememberCookies("api.example.com", cookies([2]string{"session", "new"}))

	got := s.HeadersFor("api.example.com")
	if got["Cookie"] != "session=new" {
		t.Errorf("Cookie header = %q, want %q", got["Cookie"], "session=new")
	}
}

func TestStore_RememberAuthHeader_RoundTrips(t *testing.T) {
	s := NewStore()
	s.RememberAuthHeader("api.example.com", "Authorization", "Bearer token123")

	got := s.HeadersFor("api.example.com")
	if got["Authorization"] != "Bearer token123" {
		t.Errorf("Authorization header = %q, want %q", got["Authorization"], "Bearer token123")
	}
}

func TestStore_HeadersFor_UnknownHost_ReturnsNil(t *testing.T) {
	s := NewStore()
	if got := s.HeadersFor("unknown.example.com"); got != nil {
		t.Errorf("expected nil for an unknown host, got %v", got)
	}
}

func TestStore_HeadersFor_KeepsHostsSeparate(t *testing.T) {
	s := NewStore()
	s.RememberCookies("a.example.com", cookies([2]string{"session", "a-value"}))
	s.RememberCookies("b.example.com", cookies([2]string{"session", "b-value"}))

	if got := s.HeadersFor("a.example.com")["Cookie"]; got != "session=a-value" {
		t.Errorf("host a Cookie = %q, want %q", got, "session=a-value")
	}
	if got := s.HeadersFor("b.example.com")["Cookie"]; got != "session=b-value" {
		t.Errorf("host b Cookie = %q, want %q", got, "session=b-value")
	}
}

func TestStore_Snapshot_SortedByName(t *testing.T) {
	s := NewStore()
	s.RememberCookies("api.example.com", cookies([2]string{"session", "abc"}, [2]string{"csrftoken", "xyz"}))
	s.RememberAuthHeader("api.example.com", "Authorization", "Bearer token")

	snap := s.Snapshot("api.example.com")
	if len(snap.Cookies) != 2 || snap.Cookies[0].Name != "csrftoken" || snap.Cookies[1].Name != "session" {
		t.Errorf("unexpected cookie snapshot: %+v", snap.Cookies)
	}
	if len(snap.AuthHeaders) != 1 || snap.AuthHeaders[0].Name != "Authorization" {
		t.Errorf("unexpected auth header snapshot: %+v", snap.AuthHeaders)
	}
	if snap.IsEmpty() {
		t.Error("expected a non-empty snapshot")
	}
}

func TestStore_Snapshot_UnknownHost_IsEmpty(t *testing.T) {
	s := NewStore()
	snap := s.Snapshot("unknown.example.com")
	if !snap.IsEmpty() {
		t.Errorf("expected an empty snapshot, got %+v", snap)
	}
}

func TestStore_DeleteCookie_RemovesOnlyThatCookie(t *testing.T) {
	s := NewStore()
	s.RememberCookies("api.example.com", cookies([2]string{"session", "abc"}, [2]string{"csrftoken", "xyz"}))

	s.DeleteCookie("api.example.com", "csrftoken")

	got := s.HeadersFor("api.example.com")
	if got["Cookie"] != "session=abc" {
		t.Errorf("Cookie header = %q, want %q", got["Cookie"], "session=abc")
	}
}

func TestStore_DeleteAuthHeader_RemovesOnlyThatHeader(t *testing.T) {
	s := NewStore()
	s.RememberAuthHeader("api.example.com", "Authorization", "Bearer token")
	s.RememberAuthHeader("api.example.com", "X-Api-Key", "secret")

	s.DeleteAuthHeader("api.example.com", "Authorization")

	got := s.HeadersFor("api.example.com")
	if _, ok := got["Authorization"]; ok {
		t.Error("expected Authorization to be deleted")
	}
	if got["X-Api-Key"] != "secret" {
		t.Errorf("X-Api-Key = %q, want %q", got["X-Api-Key"], "secret")
	}
}

func TestStore_Clear_RemovesEverythingForHost(t *testing.T) {
	s := NewStore()
	s.RememberCookies("api.example.com", cookies([2]string{"session", "abc"}))
	s.RememberAuthHeader("api.example.com", "Authorization", "Bearer token")
	s.RememberCookies("other.example.com", cookies([2]string{"session", "xyz"}))

	s.Clear("api.example.com")

	if got := s.HeadersFor("api.example.com"); got != nil {
		t.Errorf("expected nothing remembered for api.example.com, got %v", got)
	}
	if got := s.HeadersFor("other.example.com"); got == nil {
		t.Error("expected other.example.com to be unaffected by clearing api.example.com")
	}
}

// TestStore_NilStore_IsSafe verifies every method is a safe no-op on a nil
// *Store - responsepane/requestpane are handed one before an app.Model
// even exists, in every test that doesn't call SetJar.
func TestStore_NilStore_IsSafe(t *testing.T) {
	var s *Store

	s.RememberCookies("host", cookies([2]string{"a", "b"}))
	s.RememberAuthHeader("host", "Authorization", "Bearer x")
	s.DeleteCookie("host", "a")
	s.DeleteAuthHeader("host", "Authorization")
	s.Clear("host")

	if got := s.HeadersFor("host"); got != nil {
		t.Errorf("expected nil from a nil *Store, got %v", got)
	}
	if snap := s.Snapshot("host"); !snap.IsEmpty() {
		t.Errorf("expected an empty snapshot from a nil *Store, got %+v", snap)
	}
}
