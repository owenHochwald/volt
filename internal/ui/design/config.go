package design

const ThemeSchemaVersion = 1

type ColorValue string

// AppConfig selects a theme and motion preference from Volt's user
// configuration file.
type AppConfig struct {
	Version int        `yaml:"version"`
	Theme   string     `yaml:"theme"`
	Motion  MotionMode `yaml:"motion"`
}

// ThemeConfig is the parsed, unresolved representation of a user theme.
// Missing values inherit from Extends before the UI receives a Theme.
type ThemeConfig struct {
	Version int             `yaml:"version"`
	Name    string          `yaml:"name"`
	Extends string          `yaml:"extends"`
	Colors  ColorOverrides  `yaml:"colors,omitempty"`
	Methods MethodOverrides `yaml:"methods,omitempty"`
	Charts  ChartOverrides  `yaml:"charts,omitempty"`
}

type ColorOverrides struct {
	Canvas        ColorValue `yaml:"canvas,omitempty"`
	Surface       ColorValue `yaml:"surface,omitempty"`
	SurfaceRaised ColorValue `yaml:"surface_raised,omitempty"`
	Border        ColorValue `yaml:"border,omitempty"`
	Text          ColorValue `yaml:"text,omitempty"`
	TextMuted     ColorValue `yaml:"text_muted,omitempty"`
	Brand         ColorValue `yaml:"brand,omitempty"`
	BrandStrong   ColorValue `yaml:"brand_strong,omitempty"`
	Charge        ColorValue `yaml:"charge,omitempty"`
	Signal        ColorValue `yaml:"signal,omitempty"`
	Info          ColorValue `yaml:"info,omitempty"`
	Success       ColorValue `yaml:"success,omitempty"`
	Warning       ColorValue `yaml:"warning,omitempty"`
	Error         ColorValue `yaml:"error,omitempty"`
}

type MethodOverrides struct {
	GET    ColorValue `yaml:"get,omitempty"`
	POST   ColorValue `yaml:"post,omitempty"`
	PUT    ColorValue `yaml:"put,omitempty"`
	PATCH  ColorValue `yaml:"patch,omitempty"`
	DELETE ColorValue `yaml:"delete,omitempty"`
}

type ChartOverrides struct {
	Primary   ColorValue `yaml:"primary,omitempty"`
	Secondary ColorValue `yaml:"secondary,omitempty"`
	Good      ColorValue `yaml:"good,omitempty"`
	Bad       ColorValue `yaml:"bad,omitempty"`
}
