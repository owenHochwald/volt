package app

// sidebarWidth returns the sidebar's column width, scaled with the
// terminal but clamped so it never gets unusably narrow on small terminals
// or wastes space on very wide ones.
func (m Model) sidebarWidth() int {
	width := m.width / 5
	if width < 20 {
		width = 20
	}
	if width > 60 {
		width = 60
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

// requestHeight/requestWidth are the box dimensions view.go actually
// renders the request pane into. update.go's WindowSizeMsg handler must
// call m.requestPane.SetHeight/SetWidth with these same values (not just
// view.go, which alone doesn't persist them - see requestPane.View's value
// receiver) so that a later Update() - e.g. handling a pgup/pgdown scroll -
// sees a FieldsViewport sized to match what's actually on screen, rather
// than the zero value.
func (m Model) requestHeight() int {
	return int(float64(m.mainHeight())/2.2) - 10
}

func (m Model) requestWidth() int {
	return m.mainWidth() - 10
}
