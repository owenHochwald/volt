package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	//http2 "github.com/owenHochwald/Volt/internal/http"
)

func TestClient_Send(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		headers        map[string]string
		mockHandler    http.HandlerFunc
		wantStatusCode int
		wantBody       string
		wantErr        bool
	}{
		{
			name:   "GET success",
			method: http.MethodGet,
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "ok",
		},
		{
			name:   "POST with body",
			method: http.MethodPost,
			body:   `{"msg":"hi"}`,
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				data := make([]byte, r.ContentLength)
				r.Body.Read(data)
				if string(data) != `{"msg":"hi"}` {
					http.Error(w, "bad body", http.StatusBadRequest)
					return
				}
				w.Write([]byte("received"))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "received",
		},
		{
			name:   "Timeout error",
			method: http.MethodGet,
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				w.Write([]byte("too slow"))
			},
			wantErr: true,
		},
		{
			name:   "Server error",
			method: http.MethodGet,
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "fail", http.StatusInternalServerError)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "fail\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.mockHandler)
			defer server.Close()

			client := InitClient(100*time.Millisecond, true)

			req := &Request{
				Method:  tt.method,
				URL:     server.URL,
				Body:    tt.body,
				Headers: tt.headers,
			}

			res := make(chan *Response)
			go client.Send(req, res)
			result := <-res

			if tt.wantErr {
				if result.Error == "" {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if result.Error != "" {
				t.Fatalf("unexpected error: %v", result.Error)
			}

			if result.StatusCode != tt.wantStatusCode {
				t.Errorf("expected status %d, got %d", tt.wantStatusCode, result.StatusCode)
			}

			if result.Body != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, result.Body)
			}
		})
	}
}
