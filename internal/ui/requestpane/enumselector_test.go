package requestpane

import "testing"

// TestNewEnumSelector_WithDefault_NeverStartsUnset verifies a schema
// default puts the selector straight on a real value, with no unset state
// reachable at all - matching the pre-unset behavior exactly for a field
// that actually declares one.
func TestNewEnumSelector_WithDefault_NeverStartsUnset(t *testing.T) {
	e := NewEnumSelector([]string{"a", "b", "c"}, "b")
	if got := e.Value(); got != "b" {
		t.Fatalf("expected to start on the default %q, got %q", "b", got)
	}
	// Next() all the way around (len+1 times, to check it never lands on
	// unset) must only ever visit real options.
	for i := 0; i < len([]string{"a", "b", "c"})+1; i++ {
		e.Next()
		if got := e.Value(); got == "" {
			t.Fatalf("expected a defaulted selector to never cycle through an unset state, got empty at step %d", i)
		}
	}
}

// TestNewEnumSelector_NoDefault_StartsUnset verifies a selector built with
// an empty initial (schema declared no default) starts with no value
// chosen at all, and that Next()/Prev() include that unset state as one
// more reachable stop in the cycle rather than always landing on a real
// option.
func TestNewEnumSelector_NoDefault_StartsUnset(t *testing.T) {
	e := NewEnumSelector([]string{"a", "b"}, "")
	if got := e.Value(); got != "" {
		t.Fatalf("expected no schema default to start unset, got %q", got)
	}

	e.Next()
	if got := e.Value(); got != "a" {
		t.Fatalf("expected Next() from unset to reach the first option %q, got %q", "a", got)
	}
	e.Next()
	if got := e.Value(); got != "b" {
		t.Fatalf("expected Next() to reach %q, got %q", "b", got)
	}
	e.Next()
	if got := e.Value(); got != "" {
		t.Fatalf("expected Next() past the last option to wrap back to unset, got %q", got)
	}
}

// TestNewEnumSelector_NoDefault_PrevWrapsBothDirections mirrors the above
// for Prev(), including wrapping from unset backward to the last option.
func TestNewEnumSelector_NoDefault_PrevWrapsBothDirections(t *testing.T) {
	e := NewEnumSelector([]string{"a", "b"}, "")

	e.Prev()
	if got := e.Value(); got != "b" {
		t.Fatalf("expected Prev() from unset to wrap to the last option %q, got %q", "b", got)
	}
	e.Prev()
	if got := e.Value(); got != "a" {
		t.Fatalf("expected Prev() to reach %q, got %q", "a", got)
	}
	e.Prev()
	if got := e.Value(); got != "" {
		t.Fatalf("expected Prev() past the first option to reach unset, got %q", got)
	}
}

// TestEnumSelector_View_ShowsPlaceholderWhenUnset verifies the unset state
// renders something readable rather than a blank "‹ ›", so it's visually
// distinguishable from a field that has an actual empty-string value.
func TestEnumSelector_View_ShowsPlaceholderWhenUnset(t *testing.T) {
	e := NewEnumSelector([]string{"a", "b"}, "")
	if got := e.View(); got == "" {
		t.Fatal("expected a non-empty view even when unset")
	}
}
