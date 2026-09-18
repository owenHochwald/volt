package responsepane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
)

// formatJSON attempts to format JSON content with indentation
func formatJSON(content string) string {
	var pretty bytes.Buffer
	err := json.Indent(&pretty, []byte(content), "", "    ")
	if err != nil {
		return content
	}
	return pretty.String()
}

// defaultBodyStyle is the chroma style used for the Body tab's pretty
// rendering until the user picks a different one (see the "t" style
// picker) - restored on every app start unless something else was saved
// (see design.SaveBodyStyle/LoadBodyStyle).
const defaultBodyStyle = "onedark"

// highlightContent applies syntax highlighting using chroma's styleName
// (see ResponsePane.bodyStyleName) - falling back to defaultBodyStyle for
// an empty/unrecognized name, the same fallback chroma's own styles.Get
// already applies internally, spelled out here since callers building
// live previews (see style_picker.go) want that same behavior without
// going through a full ResponsePane.
func highlightContent(content, lexer, styleName string) string {
	if styleName == "" {
		styleName = defaultBodyStyle
	}
	var buf bytes.Buffer
	err := quick.Highlight(&buf, content, lexer, "terminal256", styleName)
	if err != nil {
		return content
	}
	return buf.String()
}

// getContentLexer maps content type to chroma lexer name
func getContentLexer(contentType string) string {
	switch {
	case strings.Contains(contentType, "application/json"):
		return "json"
	case strings.Contains(contentType, "text/html"):
		return "html"
	case strings.Contains(contentType, "text/plain"):
		return "plaintext"
	case strings.Contains(contentType, "application/xml"), strings.Contains(contentType, "text/xml"):
		return "xml"
	default:
		return ""
	}
}

// formatContentByType formats response body based on content type,
// syntax-highlighted with styleName (see ResponsePane.bodyStyleName).
func formatContentByType(body, contentType, styleName string) string {
	switch {
	case strings.Contains(contentType, "application/json"):
		formatted := formatJSON(body)
		return highlightContent(formatted, "json", styleName)

	case strings.Contains(contentType, "image/jpeg"), strings.Contains(contentType, "image/"):
		return fmt.Sprintf("Sorry, we don't support %s yet!", contentType)

	case strings.Contains(contentType, "text/html"):
		return highlightContent(body, "html", styleName)

	case strings.Contains(contentType, "text/plain"):
		return highlightContent(body, "plaintext", styleName)

	case strings.Contains(contentType, "application/xml"), strings.Contains(contentType, "text/xml"):
		return highlightContent(body, "xml", styleName)

	case strings.Contains(contentType, "application/graphql"):
		return "Sorry, we don't support GraphQL yet!"

	case strings.Contains(contentType, "multipart/form-data"):
		return "Sorry, we don't support multipart/form-data yet!"

	default:
		return fmt.Sprintf("Unhandled Content-Type: %s\n", contentType)
	}
}

// highlightRawBody applies syntax highlighting to body exactly as
// received, for ToggleRawBody's ("r") raw view - unlike
// formatContentByType, it never reformats/re-indents first (see
// formatJSON), since "raw" means the bytes as received, just recolored
// rather than left flat and unstyled the way it used to be. Content
// types with no recognized lexer (see getContentLexer - a binary image,
// say) fall back to body completely untouched, same as raw view already
// did for everything before this.
func highlightRawBody(body, contentType, styleName string) string {
	lexer := getContentLexer(contentType)
	if lexer == "" {
		return body
	}
	return highlightContent(body, lexer, styleName)
}
