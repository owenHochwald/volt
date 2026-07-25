package responsepane

import (
	"github.com/owenHochwald/Volt/internal/ui/design"
)

var (
	responseStyles = design.NewStyles(design.DefaultTheme())

	inactiveTab = responseStyles.Tabs.Inactive
	activeTab   = responseStyles.Tabs.Active

	responseKeyStyle   = responseStyles.Text.ResponseKey
	responseValueStyle = responseStyles.Text.Value
	responseLabelStyle = responseStyles.Text.ResponseLabel

	errorStyle = responseStyles.Badge.Error

	loadTestStatusStyle = responseStyles.Badge.Live

	faintStyle = responseStyles.Text.Faint
)
