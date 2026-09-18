package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetForTest clears package-level state between tests, since Enable is
// deliberately idempotent (a real call site only ever calls it once) and
// tests need a clean slate each time.
func resetForTest(t *testing.T) {
	t.Helper()
	mu.Lock()
	logFile = nil
	logger = nil
	mu.Unlock()
	t.Cleanup(func() { Close() })
}

// TestLogf_NoopWhenNotEnabled is a regression guard for a real
// requirement: logging must stay off unless explicitly turned on (see
// main.go's -v/--debug flag) - Logf must never create a file or write
// anything on its own.
func TestLogf_NoopWhenNotEnabled(t *testing.T) {
	resetForTest(t)

	if Enabled() {
		t.Fatal("expected logging to start disabled")
	}
	// Must not panic, error, or create any file - just silently do nothing.
	Logf("this should go nowhere: %d", 42)
}

func TestEnable_WritesToFile(t *testing.T) {
	resetForTest(t)

	path := filepath.Join(t.TempDir(), "volt.log")
	if err := Enable(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Enabled() {
		t.Fatal("expected Enabled() to be true after Enable")
	}

	Logf("hello %s", "world")
	if err := Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello world") {
		t.Errorf("expected the log file to contain the logged line, got %q", string(data))
	}
}

// TestEnable_CreatesParentDirectory verifies the log directory doesn't
// need to already exist (mirrors storage.NewSQLiteStorage's own
// os.MkdirAll for ~/.volt).
func TestEnable_CreatesParentDirectory(t *testing.T) {
	resetForTest(t)

	path := filepath.Join(t.TempDir(), "nested", "dir", "volt.log")
	if err := Enable(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected the log file to exist, got %v", err)
	}
}

// TestEnable_Idempotent verifies a second Enable call (even with a
// different path) is a no-op - only one log file is ever open at a time.
func TestEnable_Idempotent(t *testing.T) {
	resetForTest(t)

	first := filepath.Join(t.TempDir(), "first.log")
	second := filepath.Join(t.TempDir(), "second.log")

	if err := Enable(first); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Enable(second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	Logf("only in first")
	Close()

	if _, err := os.Stat(second); err == nil {
		t.Error("expected the second path to never be created, since Enable was already active")
	}
	data, err := os.ReadFile(first)
	if err != nil {
		t.Fatalf("failed to read first log file: %v", err)
	}
	if !strings.Contains(string(data), "only in first") {
		t.Errorf("expected the line in the first (already-active) log file, got %q", string(data))
	}
}

// TestLogf_NoopAfterClose verifies Close fully disables logging again -
// not just closing the file handle but also making Logf a no-op, matching
// the pre-Enable state.
func TestLogf_NoopAfterClose(t *testing.T) {
	resetForTest(t)

	path := filepath.Join(t.TempDir(), "volt.log")
	if err := Enable(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Enabled() {
		t.Error("expected Enabled() to be false after Close")
	}
	// Must not panic (writing to a closed file) - just silently do nothing.
	Logf("should not panic or write")
}
