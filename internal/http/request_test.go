package http

import (
	"testing"
)

func TestRequest_Validate(t *testing.T) {
	type fields struct {
		ID      int64
		Name    string
		Method  string
		URL     string
		Headers map[string]string
		Body    string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"empty method", fields{Method: ""}, true},
		{"empty url", fields{URL: ""}, true},
		{"name too long", fields{Name: "name too longname too longname too longname too long"}, true},
		{"valid", fields{Method: GET, URL: "http://localhost"}, false},
		{"valid with id", fields{ID: 1234, Method: GET, URL: "http://localhost"}, false},
		{"valid with name", fields{Name: "test", Method: GET, URL: "http://localhost"}, false},
		{"valid with headers", fields{Method: GET, URL: "http://localhost", Headers: map[string]string{"Content-Type": "application/json"}}, false},
		{"valid with body", fields{Method: GET, URL: "http://localhost", Body: "test"}, false},
		{"invalid method", fields{Method: "GETT", URL: "http://localhost"}, true},
		{"invalid url", fields{Method: GET, URL: "htt://localhost:8080"}, true},
		{"one-character url", fields{Method: GET, URL: "h"}, true},
		{"three-character url", fields{Method: GET, URL: "htt"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				ID:      tt.fields.ID,
				Name:    tt.fields.Name,
				Method:  tt.fields.Method,
				URL:     tt.fields.URL,
				Headers: tt.fields.Headers,
				Body:    tt.fields.Body,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
