package requestpane

import (
	"os"
	"testing"
)

// TestMain isolates this package's tests from the real user config
// directory - responsepane.SetupResponsePane (used by this package's own
// integration tests) loads its persisted body style (see
// design.LoadBodyStyle) from there unless redirected, and without this,
// tests would read (and could flake on) whatever real config.yaml happens
// to exist on the machine running them instead of exercising a
// clean-slate default.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "volt-requestpane-test-config")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	os.Setenv("XDG_CONFIG_HOME", dir)
	os.Setenv("HOME", dir)

	os.Exit(m.Run())
}
