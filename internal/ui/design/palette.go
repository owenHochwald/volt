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
