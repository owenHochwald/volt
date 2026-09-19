package responsepane

import (
	"os"
	"testing"
)

// TestMain isolates this package's tests from the real user config
// directory - SetupResponsePane loads its persisted body style (see
// design.LoadBodyStyle) from there unless redirected, and without this,
// tests would read (and could flake on) whatever real config.yaml happens
// to exist on the machine running them instead of exercising a
// clean-slate default.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "volt-responsepane-test-config")
	if err != nil {
		panic(err)
	}

	os.Setenv("XDG_CONFIG_HOME", dir)
	os.Setenv("HOME", dir)

	// os.Exit below skips deferred calls, so the cleanup can't be a plain
	// defer - it has to run explicitly, after m.Run() returns but before
	// exiting, or this directory would leak on every test run.
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
