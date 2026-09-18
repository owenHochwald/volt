package design

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type MotionMode string

const (
	MotionSystem  MotionMode = "system"
	MotionFull    MotionMode = "full"
	MotionReduced MotionMode = "reduced"
)

// Theme is the complete, validated set of semantic values consumed by the UI.
// Configuration and inheritance are resolved before constructing this value.
type Theme struct {
	Name   string
	Colors Colors
	Motion MotionMode
}

type Colors struct {
	Canvas, Surface, SurfaceRaised color.Color
	Border, Text, TextMuted        color.Color
	Brand, BrandStrong, Charge     color.Color
	Signal, Info                   color.Color
	Success, Warning, Error        color.Color
	MethodGET, MethodPOST          color.Color
	MethodPUT, MethodPATCH         color.Color
	MethodDELETE                   color.Color
	ChartPrimary, ChartSecondary   color.Color
	ChartGood, ChartBad            color.Color
}

func DefaultTheme() Theme {
	return themeFromPalette("default", ControlledVoltagePalette(), false)
}

func ANSITheme() Theme {
	return themeFromPalette("default", ControlledVoltagePalette(), true)
}

func themeFromPalette(name string, palette Palette, ansi bool) Theme {
	resolve := func(value PaletteValue) color.Color {
		if ansi {
			return lipgloss.Color(value.ANSI256)
		}
		return lipgloss.Color(value.TrueColor)
	}

	return Theme{
		Name:   name,
		Motion: MotionSystem,
		Colors: Colors{
			Canvas:         resolve(palette.Canvas),
			Surface:        resolve(palette.Surface),
			SurfaceRaised:  resolve(palette.SurfaceRaised),
			Border:         resolve(palette.Border),
			Text:           resolve(palette.Text),
			TextMuted:      resolve(palette.TextMuted),
			Brand:          resolve(palette.Brand),
			BrandStrong:    resolve(palette.BrandStrong),
			Charge:         resolve(palette.Charge),
			Signal:         resolve(palette.Signal),
			Info:           resolve(palette.Info),
			Success:        resolve(palette.Success),
			Warning:        resolve(palette.Warning),
			Error:          resolve(palette.Error),
			MethodGET:      resolve(palette.MethodGET),
			MethodPOST:     resolve(palette.MethodPOST),
			MethodPUT:      resolve(palette.MethodPUT),
			MethodPATCH:    resolve(palette.MethodPATCH),
			MethodDELETE:   resolve(palette.MethodDELETE),
			ChartPrimary:   resolve(palette.ChartPrimary),
			ChartSecondary: resolve(palette.ChartSecondary),
			ChartGood:      resolve(palette.ChartGood),
			ChartBad:       resolve(palette.ChartBad),
		},
	}
}
