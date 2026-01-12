package ui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/http"
)

var (
	methodStyleBase = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Border(lipgloss.NormalBorder())

	getMethodStyle    = methodStyleBase.Foreground(lipgloss.Color("42"))  // Green
	postMethodStyle   = methodStyleBase.Foreground(lipgloss.Color("214")) // Orange
	putMethodStyle    = methodStyleBase.Foreground(lipgloss.Color("117")) // Blue
	patchMethodStyle  = methodStyleBase.Foreground(lipgloss.Color("141")) // Purple
	deleteMethodStyle = methodStyleBase.Foreground(lipgloss.Color("196")) // Red
)

type MethodSelector struct {
	methods       []string
	currentMethod int
	focused       bool
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
	var methodStyle lipgloss.Style

	switch currMethod {
	case http.GET:
		methodStyle = getMethodStyle
	case http.POST:
		methodStyle = postMethodStyle
	case http.PUT:
		methodStyle = putMethodStyle
	case http.PATCH:
		methodStyle = patchMethodStyle
	case http.DELETE:
		methodStyle = deleteMethodStyle
	default:
		methodStyle = methodStyleBase
	}

	if m.focused {
		methodStyle = methodStyle.BorderForeground(focusColor)
	} else {
		methodStyle = methodStyle.BorderForeground(unfocusColor)
	}

	return methodStyle
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

func NewMethodSelector() *MethodSelector {

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
	}
}
