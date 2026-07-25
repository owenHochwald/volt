package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

type Header struct {
	width   int
	compact bool
	version string
	styles  design.Styles
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
			h.styles.Header.Logo.Render("⚡ VOLT"),
			"  ",
			h.styles.Header.Metadata.Render(h.version+" • Terminal-native HTTP client"),
		)
	}

	asciiArt := `██╗   ██╗ ██████╗ ██╗  ████████╗
██║   ██║██╔═══██╗██║  ╚══██╔══╝
██║   ██║██║   ██║██║     ██║
╚██╗ ██╔╝██║   ██║██║     ██║
 ╚████╔╝ ╚██████╔╝███████╗██║
  ╚═══╝   ╚═════╝ ╚══════╝╚═╝   `

	logo := h.styles.Header.Logo.Render(asciiArt)

	help := h.styles.Header.Metadata.Render(h.version + " • Terminal-native HTTP client")

	return lipgloss.JoinHorizontal(lipgloss.Left, logo, "\t", help)
}

func (h *Header) SetSize(width int, compact bool) {
	h.width = max(width, 0)
	h.compact = compact
}

func (h *Header) SetStyles(styles design.Styles) {
	h.styles = styles
}

func SetupHeader(version string, optionalStyles ...design.Styles) *Header {
	styles := design.NewStyles(design.DefaultTheme())
	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}

	return &Header{
		width:   80,
		version: version,
		styles:  styles,
	}
}
