package difftext

import (
	"reflect"
	"testing"
)

func TestLines_IdenticalInputs_AllEqual(t *testing.T) {
	a := []string{"one", "two", "three"}
	got := Lines(a, a)
	want := []Line{{Equal, "one"}, {Equal, "two"}, {Equal, "three"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %+v, want %+v", got, want)
	}
}

func TestLines_EmptyA_AllInsert(t *testing.T) {
	got := Lines(nil, []string{"one", "two"})
	want := []Line{{Insert, "one"}, {Insert, "two"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %+v, want %+v", got, want)
	}
}

func TestLines_EmptyB_AllDelete(t *testing.T) {
	got := Lines([]string{"one", "two"}, nil)
	want := []Line{{Delete, "one"}, {Delete, "two"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %+v, want %+v", got, want)
	}
}

func TestLines_SingleLineChanged(t *testing.T) {
	a := []string{"name: alice", "age: 30"}
	b := []string{"name: alice", "age: 31"}
	got := Lines(a, b)
	want := []Line{
		{Equal, "name: alice"},
		{Delete, "age: 30"},
		{Insert, "age: 31"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %+v, want %+v", got, want)
	}
}

func TestLines_InsertedLineInMiddle(t *testing.T) {
	a := []string{"one", "three"}
	b := []string{"one", "two", "three"}
	got := Lines(a, b)
	want := []Line{
		{Equal, "one"},
		{Insert, "two"},
		{Equal, "three"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %+v, want %+v", got, want)
	}
}

func TestLines_RemovedLineInMiddle(t *testing.T) {
	a := []string{"one", "two", "three"}
	b := []string{"one", "three"}
	got := Lines(a, b)
	want := []Line{
		{Equal, "one"},
		{Delete, "two"},
		{Equal, "three"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %+v, want %+v", got, want)
	}
}

// TestLines_ReconstructsB verifies a generic invariant of the diff -
// concatenating the Equal+Insert lines in order reproduces b exactly, no
// matter what a was - a correctness check independent of any specific
// expected sequence.
func TestLines_ReconstructsB(t *testing.T) {
	a := []string{"alpha", "beta", "gamma", "delta"}
	b := []string{"alpha", "gamma", "epsilon", "delta", "zeta"}

	var reconstructed []string
	for _, l := range Lines(a, b) {
		if l.Op == Equal || l.Op == Insert {
			reconstructed = append(reconstructed, l.Text)
		}
	}
	if !reflect.DeepEqual(reconstructed, b) {
		t.Errorf("reconstructed b = %+v, want %+v", reconstructed, b)
	}
}

// TestLines_FallsBackWhenTooLarge verifies the byLine fallback kicks in
// for inputs that would otherwise need an oversized LCS table, and that
// it's still a technically valid (if non-minimal) diff.
func TestLines_FallsBackWhenTooLarge(t *testing.T) {
	n := 3000 // 3000*3000 = 9,000,000 > maxCells
	a := make([]string, n)
	b := make([]string, n)
	for i := range a {
		a[i] = "a"
		b[i] = "b"
	}

	got := Lines(a, b)
	if len(got) != 2*n {
		t.Fatalf("expected %d lines (all delete + all insert), got %d", 2*n, len(got))
	}
	for i := 0; i < n; i++ {
		if got[i].Op != Delete {
			t.Fatalf("expected line %d to be a Delete, got %+v", i, got[i])
		}
	}
	for i := n; i < 2*n; i++ {
		if got[i].Op != Insert {
			t.Fatalf("expected line %d to be an Insert, got %+v", i, got[i])
		}
	}
}
