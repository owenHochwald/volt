package app

import "testing"

// TestSidebarWidth_ClampsBetweenMinAndMax locks in the sidebar's width
// bounds: never narrower than 20 (unusable) or wider than 80 (per a real
// request to cap it there so it doesn't keep eating more of a very wide
// terminal as the 1/5-of-total-width scaling would otherwise let it).
func TestSidebarWidth_ClampsBetweenMinAndMax(t *testing.T) {
	tests := []struct {
		totalWidth int
		want       int
	}{
		{totalWidth: 50, want: 20},   // 50/5=10, below the floor
		{totalWidth: 100, want: 20},  // 100/5=20, right at the floor
		{totalWidth: 200, want: 40},  // 200/5=40, within range
		{totalWidth: 400, want: 80},  // 400/5=80, right at the new cap
		{totalWidth: 1000, want: 80}, // 1000/5=200, well above the cap
	}
	for _, tt := range tests {
		m := Model{width: tt.totalWidth}
		if got := m.sidebarWidth(); got != tt.want {
			t.Errorf("sidebarWidth() with total width %d = %d, want %d", tt.totalWidth, got, tt.want)
		}
	}
}
