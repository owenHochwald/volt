package utils

import (
	"strconv"

	"charm.land/lipgloss/v2"
)

var (
	successStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("76"))
	permErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	serverErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("160"))
	weirdErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("96"))
)

func MapStatusCodeToColor(statusCode int) lipgloss.Style {
	statusString := strconv.Itoa(statusCode)
	if len(statusString) != 3 {
		return weirdErrorStyle
	}

	switch statusString[0] {
	case '2':
		return successStyle
	case '3':
		return permErrorStyle
	case '4', '5':
		return serverErrorStyle
	default:
		return weirdErrorStyle
	}

}
