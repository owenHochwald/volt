package requestpane

import (
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

func TestSyncRequestPreservesRawBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "json", body: "{\n  \"enabled\": true,\n  \"items\": [1, 2]\n}"},
		{name: "xml", body: "<request><enabled>true</enabled></request>"},
		{name: "graphql", body: "query User { user(id: 1) { name } }"},
		{name: "plain text", body: "hello = world"},
		{name: "empty", body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestRequestPane(t)
			pane.Body.SetValue(tt.body)

			pane.syncRequest()

			if pane.Request.Body != tt.body {
				t.Fatalf("request body = %q, want exact raw body %q", pane.Request.Body, tt.body)
			}
		})
	}
}

func TestReinitRequestPanePreservesRawBody(t *testing.T) {
	tests := []string{
		"",
		"{}",
		"{\"name\":\"volt\"}",
		"unstructured request body",
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			pane := newTestRequestPane(t)
			request := &http.Request{
				Method:  http.POST,
				URL:     "https://example.com",
				Headers: map[string]string{},
				Body:    body,
			}

			pane.reinitRequestPane(request)

			if got := pane.Body.Value(); got != body {
				t.Fatalf("body editor value = %q, want %q", got, body)
			}
		})
	}
}
