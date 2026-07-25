package design

const (
	colorCanvas        = "#090B10"
	colorSurface       = "#11151D"
	colorSurfaceRaised = "#181E29"
	colorBorder        = "#30394A"
	colorText          = "#EDF2FF"
	colorTextMuted     = "#7F8A9D"
	colorBrand         = "#9B6CFF"
	colorBrandStrong   = "#7038E8"
	colorCharge        = "#D8FF3E"
	colorSignal        = "#3DE4E8"
	colorInfo          = "#68B7FF"
	colorSuccess       = "#5EE08A"
	colorWarning       = "#FFC857"
	colorError         = "#FF647C"
	colorMethodPatch   = "#B78CFF"
	ansiCanvas         = "232"
	ansiSurface        = "233"
	ansiSurfaceRaised  = "234"
	ansiBorder         = "239"
	ansiText           = "255"
	ansiTextMuted      = "245"
	ansiBrand          = "135"
	ansiBrandStrong    = "93"
	ansiCharge         = "191"
	ansiSignal         = "44"
	ansiInfo           = "75"
	ansiSuccess        = "78"
	ansiWarning        = "221"
	ansiError          = "204"
	ansiMethodPatch    = "141"
	lightCanvas        = "#F7F8FC"
	lightSurface       = "#EEF1F7"
	lightSurfaceRaised = "#E2E7F0"
	lightBorder        = "#AAB4C3"
	lightText          = "#151923"
	lightTextMuted     = "#5E6878"
	lightBrand         = "#6534D9"
	lightBrandStrong   = "#DCCFFF"
	lightCharge        = "#647500"
	lightSignal        = "#007C83"
	lightInfo          = "#1268C4"
	lightSuccess       = "#187A3B"
	lightWarning       = "#9A6100"
	lightError         = "#C9324F"
	lightMethodPatch   = "#7540B8"
)

// PaletteValue retains the authored true color and intentional ANSI-256
// fallback for a semantic role. The fallback is selected deliberately rather
// than by nearest-color conversion so roles remain distinct.
type PaletteValue struct {
	TrueColor string
	ANSI256   string
}

type Palette struct {
	Canvas, Surface, SurfaceRaised PaletteValue
	Border, Text, TextMuted        PaletteValue
	Brand, BrandStrong, Charge     PaletteValue
	Signal, Info                   PaletteValue
	Success, Warning, Error        PaletteValue
	MethodGET, MethodPOST          PaletteValue
	MethodPUT, MethodPATCH         PaletteValue
	MethodDELETE                   PaletteValue
	ChartPrimary, ChartSecondary   PaletteValue
	ChartGood, ChartBad            PaletteValue
}

// ControlledVoltagePalette is the authored source palette for Volt's default
// theme. UI components consume a resolved Theme instead of this raw palette.
func ControlledVoltagePalette() Palette {
	return Palette{
		Canvas:         PaletteValue{TrueColor: colorCanvas, ANSI256: ansiCanvas},
		Surface:        PaletteValue{TrueColor: colorSurface, ANSI256: ansiSurface},
		SurfaceRaised:  PaletteValue{TrueColor: colorSurfaceRaised, ANSI256: ansiSurfaceRaised},
		Border:         PaletteValue{TrueColor: colorBorder, ANSI256: ansiBorder},
		Text:           PaletteValue{TrueColor: colorText, ANSI256: ansiText},
		TextMuted:      PaletteValue{TrueColor: colorTextMuted, ANSI256: ansiTextMuted},
		Brand:          PaletteValue{TrueColor: colorBrand, ANSI256: ansiBrand},
		BrandStrong:    PaletteValue{TrueColor: colorBrandStrong, ANSI256: ansiBrandStrong},
		Charge:         PaletteValue{TrueColor: colorCharge, ANSI256: ansiCharge},
		Signal:         PaletteValue{TrueColor: colorSignal, ANSI256: ansiSignal},
		Info:           PaletteValue{TrueColor: colorInfo, ANSI256: ansiInfo},
		Success:        PaletteValue{TrueColor: colorSuccess, ANSI256: ansiSuccess},
		Warning:        PaletteValue{TrueColor: colorWarning, ANSI256: ansiWarning},
		Error:          PaletteValue{TrueColor: colorError, ANSI256: ansiError},
		MethodGET:      PaletteValue{TrueColor: colorSuccess, ANSI256: ansiSuccess},
		MethodPOST:     PaletteValue{TrueColor: colorWarning, ANSI256: ansiWarning},
		MethodPUT:      PaletteValue{TrueColor: colorInfo, ANSI256: ansiInfo},
		MethodPATCH:    PaletteValue{TrueColor: colorMethodPatch, ANSI256: ansiMethodPatch},
		MethodDELETE:   PaletteValue{TrueColor: colorError, ANSI256: ansiError},
		ChartPrimary:   PaletteValue{TrueColor: colorBrand, ANSI256: ansiBrand},
		ChartSecondary: PaletteValue{TrueColor: colorSignal, ANSI256: ansiSignal},
		ChartGood:      PaletteValue{TrueColor: colorSuccess, ANSI256: ansiSuccess},
		ChartBad:       PaletteValue{TrueColor: colorError, ANSI256: ansiError},
	}
}

// LightVoltagePalette preserves Controlled Voltage's semantic distinctions on
// a light terminal background without turning the interface pastel or soft.
func LightVoltagePalette() Palette {
	return Palette{
		Canvas:         PaletteValue{TrueColor: lightCanvas, ANSI256: "255"},
		Surface:        PaletteValue{TrueColor: lightSurface, ANSI256: "255"},
		SurfaceRaised:  PaletteValue{TrueColor: lightSurfaceRaised, ANSI256: "254"},
		Border:         PaletteValue{TrueColor: lightBorder, ANSI256: "248"},
		Text:           PaletteValue{TrueColor: lightText, ANSI256: "234"},
		TextMuted:      PaletteValue{TrueColor: lightTextMuted, ANSI256: "242"},
		Brand:          PaletteValue{TrueColor: lightBrand, ANSI256: "92"},
		BrandStrong:    PaletteValue{TrueColor: lightBrandStrong, ANSI256: "189"},
		Charge:         PaletteValue{TrueColor: lightCharge, ANSI256: "64"},
		Signal:         PaletteValue{TrueColor: lightSignal, ANSI256: "30"},
		Info:           PaletteValue{TrueColor: lightInfo, ANSI256: "25"},
		Success:        PaletteValue{TrueColor: lightSuccess, ANSI256: "28"},
		Warning:        PaletteValue{TrueColor: lightWarning, ANSI256: "94"},
		Error:          PaletteValue{TrueColor: lightError, ANSI256: "160"},
		MethodGET:      PaletteValue{TrueColor: lightSuccess, ANSI256: "28"},
		MethodPOST:     PaletteValue{TrueColor: lightWarning, ANSI256: "94"},
		MethodPUT:      PaletteValue{TrueColor: lightInfo, ANSI256: "25"},
		MethodPATCH:    PaletteValue{TrueColor: lightMethodPatch, ANSI256: "91"},
		MethodDELETE:   PaletteValue{TrueColor: lightError, ANSI256: "160"},
		ChartPrimary:   PaletteValue{TrueColor: lightBrand, ANSI256: "92"},
		ChartSecondary: PaletteValue{TrueColor: lightSignal, ANSI256: "30"},
		ChartGood:      PaletteValue{TrueColor: lightSuccess, ANSI256: "28"},
		ChartBad:       PaletteValue{TrueColor: lightError, ANSI256: "160"},
	}
}
