package app

// sidebarWidth returns the sidebar's column width, scaled with the
// terminal but clamped so it never gets unusably narrow on small terminals
// or wastes space on very wide ones.
func (m Model) sidebarWidth() int {
	width := m.width / 5
	if width < 20 {
		width = 20
	}
	if width > 80 {
		width = 80
	}
	return width
}

// sidebarContentHeight returns the height available for the sidebar's
// list.Model. It must match the box the sidebar is actually rendered into
// in view.go (mainHeight-4) - update.go's SetSize and view.go's box size
// diverging is exactly what caused the right panel to visibly shift while
// navigating a long list, so both now go through this single calculation.
func (m Model) sidebarContentHeight() int {
	contentHeight := m.height - 5
	mainHeight := contentHeight - 2
	return mainHeight - 4
}

// mainWidth returns the width available to the request/response column,
// to the right of the sidebar.
func (m Model) mainWidth() int {
	return m.width - m.sidebarWidth() - 4
}

// mainHeight returns the height available to the request+response column.
func (m Model) mainHeight() int {
	contentHeight := m.height - 5
	return contentHeight - 2
}

// boxBorderLines is how many extra lines a lipgloss bordered box (request/
// response's RoundedBorder) renders on top of the content height passed to
// .Height() - one for the top border, one for the bottom.
const boxBorderLines = 2

// rightColumnHeight is the total rendered height (borders included) the
// request and response boxes together must add up to, so the right column
// ends at the same row as the sidebar instead of falling short and leaving
// blank space beneath it. Mirrors what the sidebar box itself actually
// renders to: view.go's sidebarView is given mainHeight directly and
// renders a content height of mainHeight-4 (matching sidebarContentHeight,
// used for the list's own internal sizing) plus 2 border lines, i.e.
// mainHeight-2 total.
func (m Model) rightColumnHeight() int {
	return m.mainHeight() - 2
}

// requestHeight/requestWidth are the box dimensions view.go actually
// renders the request pane into. update.go's WindowSizeMsg handler must
// call m.requestPane.SetHeight/SetWidth with these same values (not just
// view.go, which alone doesn't persist them - see requestPane.View's value
// receiver) so that a later Update() - e.g. handling a pgup/pgdown scroll -
// sees a FieldsViewport sized to match what's actually on screen, rather
// than the zero value.
func (m Model) requestHeight() int {
	// ~35% of the right column's content height goes to the request editor;
	// responseHeight takes what's left (not a fixed proportion of its own),
	// so the two always sum to rightColumnHeight exactly - no leftover gap
	// regardless of rounding.
	content := m.rightColumnHeight() - 2*boxBorderLines
	return content * 35 / 100
}

func (m Model) requestWidth() int {
	return m.mainWidth() - 10
}

// responseHeight/responseWidth are the box dimensions view.go renders the
// response pane into - see requestHeight for why this is "whatever's left"
// rather than its own independent proportion.
func (m Model) responseHeight() int {
	content := m.rightColumnHeight() - 2*boxBorderLines
	return content - m.requestHeight()
}

func (m Model) responseWidth() int {
	return m.mainWidth() - 10
}
