package responsepane

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestResponseTabLabelsFollowConfiguredBinding(t *testing.T) {
	keys := keybindings.DefaultKeyMap()
	keys.DirectTab = key.NewBinding(key.WithKeys("a", "b", "c"))
	pane := SetupResponsePane(keys)

	rendered := pane.renderTabs()
	for _, expected := range []string{"[a] Body", "[b] Headers", "[c] Timing"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("tab bar does not contain %q: %s", expected, rendered)
		}
	}
}
