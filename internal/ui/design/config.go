package design

const ThemeSchemaVersion = 1

type ColorValue string

// AppConfig selects a theme and motion preference from Volt's user
// configuration file.
type AppConfig struct {
	Version int        `json:"version" yaml:"version"`
	Theme   string     `json:"theme" yaml:"theme"`
	Motion  MotionMode `json:"motion" yaml:"motion"`
}

// ThemeConfig is the parsed, unresolved representation of a user theme.
// Missing values inherit from Extends before the UI receives a Theme.
type ThemeConfig struct {
	Version    int                              `json:"version" yaml:"version"`
	Name       string                           `json:"name" yaml:"name"`
	Extends    string                           `json:"extends" yaml:"extends"`
	Colors     ColorOverrides                   `json:"colors,omitempty" yaml:"colors,omitempty"`
	Methods    MethodOverrides                  `json:"methods,omitempty" yaml:"methods,omitempty"`
	Charts     ChartOverrides                   `json:"charts,omitempty" yaml:"charts,omitempty"`
	Components map[string]map[string]ColorValue `json:"components,omitempty" yaml:"components,omitempty"`
}

type ColorOverrides struct {
	Canvas        ColorValue `json:"canvas,omitempty" yaml:"canvas,omitempty"`
	Surface       ColorValue `json:"surface,omitempty" yaml:"surface,omitempty"`
	SurfaceRaised ColorValue `json:"surface_raised,omitempty" yaml:"surface_raised,omitempty"`
	Border        ColorValue `json:"border,omitempty" yaml:"border,omitempty"`
	Text          ColorValue `json:"text,omitempty" yaml:"text,omitempty"`
	TextMuted     ColorValue `json:"text_muted,omitempty" yaml:"text_muted,omitempty"`
	Brand         ColorValue `json:"brand,omitempty" yaml:"brand,omitempty"`
	BrandStrong   ColorValue `json:"brand_strong,omitempty" yaml:"brand_strong,omitempty"`
	Charge        ColorValue `json:"charge,omitempty" yaml:"charge,omitempty"`
	Signal        ColorValue `json:"signal,omitempty" yaml:"signal,omitempty"`
	Info          ColorValue `json:"info,omitempty" yaml:"info,omitempty"`
	Success       ColorValue `json:"success,omitempty" yaml:"success,omitempty"`
	Warning       ColorValue `json:"warning,omitempty" yaml:"warning,omitempty"`
	Error         ColorValue `json:"error,omitempty" yaml:"error,omitempty"`
}

type MethodOverrides struct {
	GET    ColorValue `json:"get,omitempty" yaml:"get,omitempty"`
	POST   ColorValue `json:"post,omitempty" yaml:"post,omitempty"`
	PUT    ColorValue `json:"put,omitempty" yaml:"put,omitempty"`
	PATCH  ColorValue `json:"patch,omitempty" yaml:"patch,omitempty"`
	DELETE ColorValue `json:"delete,omitempty" yaml:"delete,omitempty"`
}

type ChartOverrides struct {
	Primary   ColorValue `json:"primary,omitempty" yaml:"primary,omitempty"`
	Secondary ColorValue `json:"secondary,omitempty" yaml:"secondary,omitempty"`
	Good      ColorValue `json:"good,omitempty" yaml:"good,omitempty"`
	Bad       ColorValue `json:"bad,omitempty" yaml:"bad,omitempty"`
}
