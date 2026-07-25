package requestpane

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"github.com/owenHochwald/Volt/internal/storage"
)

// TextInputConfig holds configuration for creating a textinput.Model
type TextInputConfig struct {
	Placeholder     string
	CharLimit       int
	Width           int
	Value           string
	Suggestions     []string
	ShowSuggestions bool
}

// NewConfiguredTextInput creates a textinput.Model with the given configuration
func NewConfiguredTextInput(config TextInputConfig) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = config.Placeholder
	ti.CharLimit = config.CharLimit
	ti.SetWidth(config.Width)
	ti.ShowSuggestions = config.ShowSuggestions
	ti.SetVirtualCursor(true)

	if config.Value != "" {
		ti.SetValue(config.Value)
	}

	if len(config.Suggestions) > 0 {
		ti.SetSuggestions(config.Suggestions)
	}

	return ti
}

// NewURLInput creates a pre-configured URL input field
func NewURLInput(db *storage.SQLiteStorage) textinput.Model {
	urls := []string{"http://", "https://"}

	dbUrls, err := db.GetAllURLs()
	if err != nil {
		urls = []string{}
	}
	urls = append(urls, dbUrls...)

	// Add additional common URLs for quick access
	urls = append(
		urls,
		"http://localhost:3000/health",
		"http://localhost:3000/login",
		"http://localhost:3000/register",
		"http://localhost:3000/logout",
		"http://localhost:3000/me",
		"http://localhost:3000/users/1",
		"http://localhost:3000/posts",
		"http://localhost:3000/posts/42",
		"http://localhost:3000/upload",
		"http://localhost:3000/settings",
		"https://httpbin.org/anything",
	)

	ti := NewConfiguredTextInput(TextInputConfig{
		Value:           "",
		CharLimit:       10000,
		Width:           60,
		Suggestions:     urls,
		ShowSuggestions: true,
	})

	// Really weird hack to make suggestions work
	km := ti.KeyMap
	km.AcceptSuggestion = key.NewBinding(key.WithKeys(
		"enter",
		"right",
	))
	ti.KeyMap = km

	return ti
}

// NewNameInput creates a pre-configured name input field
func NewNameInput() textinput.Model {
	return NewConfiguredTextInput(TextInputConfig{
		Placeholder: "My new awesome request..",
		CharLimit:   40,
		Width:       60,
	})
}

// NewLoadTestInput creates a pre-configured load test input field
func NewLoadTestInput(placeholder string, charLimit, width int) textinput.Model {
	return NewConfiguredTextInput(TextInputConfig{
		Placeholder: placeholder,
		CharLimit:   charLimit,
		Width:       width,
	})
}

// NewHeadersTextArea creates a pre-configured headers textarea
func NewHeadersTextArea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Content-Type = multipart/form-data,\nAuthorization= Bearer ...,"
	ta.SetVirtualCursor(true)
	return ta
}

// NewBodyTextArea creates a pre-configured body textarea
func NewBodyTextArea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "{\n  \"name\": \"volt\"\n}"
	ta.SetVirtualCursor(true)
	return ta
}
