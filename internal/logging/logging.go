// Package logging provides a simple, file-backed logger for troubleshooting
// what Volt actually did (particularly HTTP request/response handling -
// see internal/http/client.go) without corrupting the TUI's terminal:
// stdout/stderr aren't usable for this once bubbletea takes over the
// screen (see tea.WithAltScreen in cmd/volt/main.go), so this writes to a
// file instead. Disabled by default (Logf is then a silent no-op) - see
// main.go's -v/--debug flag, which is what actually turns it on.
package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	mu      sync.Mutex
	logger  *log.Logger
	logFile *os.File
)

// DefaultLogPath returns ~/.volt/volt.log - the same directory
// storage.DefaultDBPath keeps the sqlite database in.
func DefaultLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}
	return filepath.Join(home, ".volt", "volt.log"), nil
}

// Enable turns on file logging at path, appending to it if it already
// exists (so a session's log isn't lost the next time Volt runs).
// Idempotent - calling it again while already enabled is a no-op, even
// with a different path, since only one log file is ever open at a time.
func Enable(path string) error {
	mu.Lock()
	defer mu.Unlock()
	if logger != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	logFile = f
	logger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
	return nil
}

// Close flushes and closes the log file, if logging was enabled - call
// once on the way out (see main.go) so the last few lines aren't lost to
// buffering.
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if logFile == nil {
		return nil
	}
	err := logFile.Close()
	logFile = nil
	logger = nil
	return err
}

// Enabled reports whether Enable has succeeded and logging is currently
// active.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return logger != nil
}

// Logf writes one formatted line to the log file if logging is enabled -
// a silent no-op otherwise, so call sites (e.g. every HTTP send) don't
// need their own "is logging on" check before calling this.
func Logf(format string, args ...interface{}) {
	mu.Lock()
	l := logger
	mu.Unlock()
	if l == nil {
		return
	}
	l.Printf(format, args...)
}
