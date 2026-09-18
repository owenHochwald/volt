package ui

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

type MethodSelector struct {
	methods       []string
	currentMethod int
	focused       bool
	styles        design.Styles
}

func (m *MethodSelector) Focus() tea.Cmd {

	m.focused = true
	return nil
}

func (m *MethodSelector) Blur() {
	m.focused = false

}

func (m *MethodSelector) Current() string {
	return m.methods[m.currentMethod]
}

func (m *MethodSelector) Next() {
	m.currentMethod = (m.currentMethod + 1) % len(m.methods)
}

func (m *MethodSelector) Prev() {
	m.currentMethod--
	if m.currentMethod < 0 {
		m.currentMethod = len(m.methods) - 1
	}
}

func (m *MethodSelector) GetStyle() lipgloss.Style {
	currMethod := m.Current()
	methodStyle := m.styles.Method.GET

	switch currMethod {
	case http.GET:
		methodStyle = m.styles.Method.GET
	case http.POST:
		methodStyle = m.styles.Method.POST
	case http.PUT:
		methodStyle = m.styles.Method.PUT
	case http.PATCH:
		methodStyle = m.styles.Method.PATCH
	case http.DELETE:
		methodStyle = m.styles.Method.DELETE
	}

	// Deliberately no border here (unlike design.NewStyles' methodBase) -
	// that would cost 2 extra vertical lines (top/bottom) for what's
	// really just a short label, in a pane that's often tight on height.
	// Focus is shown with Reverse instead of a border color change.
	return methodStyle.UnsetBorderStyle().Reverse(m.focused)
}

func (m *MethodSelector) SetCurrentIndex(method string) {
	if method == "" || !slices.Contains(m.methods, method) {
		return
	}
	for i, compare := range m.methods {
		if compare == method {
			m.currentMethod = i
		}
	}
}

func NewMethodSelector(optionalStyles ...design.Styles) *MethodSelector {
	styles := design.NewStyles(design.DefaultTheme())

	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}
	methods := []string{
		http.GET,
		http.POST,
		http.PUT,
		http.PATCH,
		http.DELETE,
	}

	return &MethodSelector{
		methods:       methods,
		currentMethod: 0,
		focused:       false,
		styles:        styles,
	}
}
