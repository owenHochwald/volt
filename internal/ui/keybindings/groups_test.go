package keybindings

import (
	"strings"
	"testing"
)

// TestCompactHelp_ShowsBothHalvesOfANavigationPair is a regression test:
// a limit of 5 cut ContextSidebar's registry-sorted list right after
// ActionNavigateUp ("k move up") without ever reaching ActionNavigateDown
// ("j move down", the same priority, declared right after it) - the
// status bar showed only half of a pair a user would expect together.
// Reported as: "в помощи когда находимся в сайдбаре написано про k -
// движение вверх, но нет упоминания про j".
func TestCompactHelp_ShowsBothHalvesOfANavigationPair(t *testing.T) {
	km := DefaultKeyMap()
	help := km.CompactHelp(ContextSidebar, 6)

	if !strings.Contains(help, "move up") {
		t.Errorf("expected %q to mention moving up, got %q", "move up", help)
	}
	if !strings.Contains(help, "move down") {
		t.Errorf("expected %q to mention moving down, got %q", "move down", help)
	}
}

// TestCompactHelp_RespectsLimit verifies the limit still caps the number
// of actions shown - it isn't just a bug-prone magic number bumped
// arbitrarily, but a real, still-enforced bound.
func TestCompactHelp_RespectsLimit(t *testing.T) {
	km := DefaultKeyMap()
	help := km.CompactHelp(ContextSidebar, 1)

	if got := strings.Count(help, " • "); got != 0 {
		t.Errorf("expected a limit of 1 to show a single action (no separators), got %d separators in %q", got, help)
	}
}
