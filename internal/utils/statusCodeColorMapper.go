package utils

import (
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

var statusStyles = design.NewStyles(design.DefaultTheme()).Status

func MapStatusCodeToColor(statusCode int) lipgloss.Style {
	switch statusCode / 100 {
	case 2:
		return statusStyles.Success
	case 3:
		return statusStyles.Redirect
	case 4:
		return statusStyles.ClientError
	case 5:
		return statusStyles.ServerError
	default:
		return statusStyles.Unknown
	}
}
