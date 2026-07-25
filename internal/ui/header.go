package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Header struct {
	width   int
	compact bool
	version string
}

func (h *Header) Init() tea.Cmd {
	return nil
}

func (h *Header) Update(msg tea.Msg) (*Header, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
	}
	return h, nil
}

func (h *Header) View() string {
	if h.compact {
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			HeaderLogoStyle.Render("⚡ VOLT"),
			"  ",
			HeaderHelpStyle.Render(h.version+" • Terminal-native HTTP client"),
		)
	}

	asciiArt := `██╗   ██╗ ██████╗ ██╗  ████████╗
██║   ██║██╔═══██╗██║  ╚══██╔══╝
██║   ██║██║   ██║██║     ██║
╚██╗ ██╔╝██║   ██║██║     ██║
 ╚████╔╝ ╚██████╔╝███████╗██║
  ╚═══╝   ╚═════╝ ╚══════╝╚═╝   `

	logo := HeaderLogoStyle.Render(asciiArt)

	help := HeaderHelpStyle.Render(h.version + " • Terminal-native HTTP client")

	return lipgloss.JoinHorizontal(lipgloss.Left, logo, "\t", help)
}

func (h *Header) SetSize(width int, compact bool) {
	h.width = max(width, 0)
	h.compact = compact
}

func SetupHeader(version string) *Header {
	return &Header{
		width:   80,
		version: version,
	}
}
