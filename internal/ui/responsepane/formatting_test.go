package responsepane

import (
	"strings"
	"testing"
)

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPart string
	}{
		{
			name:     "valid JSON formats correctly",
			input:    `{"name":"test","value":123}`,
			wantPart: "    ",
		},
		{
			name:     "invalid JSON returns original",
			input:    `{invalid json}`,
			wantPart: "{invalid json}",
		},
		{
			name:     "empty string returns empty",
			input:    "",
			wantPart: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatJSON(tt.input)
			if !strings.Contains(got, tt.wantPart) {
				t.Errorf("formatJSON() = %v, want to contain %v", got, tt.wantPart)
			}
		})
	}
}

func TestGetContentLexer(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{
			name:        "JSON content type",
			contentType: "application/json",
			want:        "json",
		},
		{
			name:        "JSON with charset",
			contentType: "application/json; charset=utf-8",
			want:        "json",
		},
		{
			name:        "HTML content type",
			contentType: "text/html",
			want:        "html",
		},
		{
			name:        "Plain text",
			contentType: "text/plain",
			want:        "plaintext",
		},
		{
			name:        "XML application type",
			contentType: "application/xml",
			want:        "xml",
		},
		{
			name:        "XML text type",
			contentType: "text/xml",
			want:        "xml",
		},
		{
			name:        "Unknown type",
			contentType: "application/octet-stream",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getContentLexer(tt.contentType)
			if got != tt.want {
				t.Errorf("getContentLexer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatContentByType(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantContain string
	}{
		{
			name:        "JSON content",
			body:        `{"test":"value"}`,
			contentType: "application/json",
			wantContain: "test",
		},
		{
			name:        "HTML content",
			body:        "<html><body>test</body></html>",
			contentType: "text/html",
			wantContain: "html",
		},
		{
			name:        "Image type shows unsupported message",
			body:        "",
			contentType: "image/jpeg",
			wantContain: "don't support",
		},
		{
			name:        "GraphQL shows unsupported message",
			body:        "",
			contentType: "application/graphql",
			wantContain: "don't support",
		},
		{
			name:        "Unknown type shows unhandled message",
			body:        "test",
			contentType: "application/custom",
			wantContain: "Unhandled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatContentByType(tt.body, tt.contentType, defaultBodyStyle)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("formatContentByType() = %v, want to contain %v", got, tt.wantContain)
			}
		})
	}
}
