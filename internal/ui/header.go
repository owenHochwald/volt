package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

const (
	HeaderFrameFull = iota
	HeaderFrameCompressed
	HeaderFrameCompact
)

type Header struct {
	width        int
	startupFrame int
	version      string
	activePanel  string
	mode         string
	styles       design.Styles
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
	switch h.startupFrame {
	case HeaderFrameFull:
		return h.fullView()
	case HeaderFrameCompressed:
		return h.compressedView()
	default:
		return h.compactView()
	}
}

func (h *Header) fullView() string {
	asciiArt := `██╗   ██╗ ██████╗ ██╗  ████████╗
██║   ██║██╔═══██╗██║  ╚══██╔══╝
██║   ██║██║   ██║██║     ██║
╚██╗ ██╔╝██║   ██║██║     ██║
 ╚████╔╝ ╚██████╔╝███████╗██║
  ╚═══╝   ╚═════╝ ╚══════╝╚═╝   `

	logo := h.styles.Header.Logo.Render(asciiArt)
	help := h.styles.Header.Metadata.Render(h.version + " • Terminal-native HTTP client")
	content := lipgloss.JoinHorizontal(lipgloss.Top, logo, "  ", help)
	return lipgloss.JoinVertical(lipgloss.Left, content, h.separator())
}

func (h *Header) compressedView() string {
	mark := h.styles.Header.Logo.Render("⚡  V O L T")
	context := h.styles.Header.Metadata.Render(h.activePanel + "  •  " + h.mode)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Left, mark, "  ", context),
		h.separator(),
	)
}

func (h *Header) compactView() string {
	left := h.styles.Header.Logo.Render("⚡ VOLT") +
		h.styles.Header.Metadata.Render("  /  "+h.activePanel)
	rightText := h.mode + "     " + h.version
	if lipgloss.Width(left)+lipgloss.Width(rightText)+1 > h.width {
		rightText = h.mode
	}
	right := h.styles.Header.Metadata.Bold(true).Render(rightText)
	gap := max(h.width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	line := left + strings.Repeat(" ", gap) + right
	line = lipgloss.NewStyle().MaxWidth(max(h.width, 1)).Render(line)
	return lipgloss.JoinVertical(lipgloss.Left, line, h.separator())
}

func (h *Header) separator() string {
	return lipgloss.NewStyle().
		Foreground(h.styles.Panel.Base.GetBorderLeftForeground()).
		MaxWidth(max(h.width, 1)).
		Render(strings.Repeat("━", max(h.width, 1)))
}

func (h *Header) SetSize(width int, compact ...bool) {
	h.width = max(width, 0)
	if len(compact) > 0 {
		if compact[0] {
			h.startupFrame = HeaderFrameCompact
		} else {
			h.startupFrame = HeaderFrameFull
		}
	}
}

func (h *Header) SetContext(activePanel, mode string) {
	h.activePanel = strings.ToUpper(activePanel)
	h.mode = strings.ToUpper(mode)
}

func (h *Header) SetStartupFrame(frame int) {
	h.startupFrame = min(max(frame, HeaderFrameFull), HeaderFrameCompact)
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
		width:        80,
		startupFrame: HeaderFrameFull,
		version:      version,
		activePanel:  "SIDEBAR",
		mode:         "NORMAL",
		styles:       styles,
	}
}
