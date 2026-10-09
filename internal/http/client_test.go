package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/owenHochwald/Volt/internal/apperror"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type observedBody struct {
	io.ReadCloser
	readStarted chan struct{}
	once        sync.Once
}

func (b *observedBody) Read(p []byte) (int, error) {
	b.once.Do(func() { close(b.readStarted) })
	return b.ReadCloser.Read(p)
}

func TestClient_SendStreamedResponse(t *testing.T) {
	for _, roundTrip := range []bool{false, true} {
		name := "body timing"
		if roundTrip {
			name = "round trip timing"
		}
		t.Run(name, func(t *testing.T) {
			bodyReady := make(chan struct{})
			var release sync.Once
			releaseBody := func() { release.Do(func() { close(bodyReady) }) }
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte("first "))
				w.(http.Flusher).Flush()
				select {
				case <-bodyReady:
					w.Write([]byte("second"))
				case <-r.Context().Done():
				}
			}))
			defer server.Close()
			defer releaseBody()

			client := InitClient(time.Second, roundTrip)
			defer client.Transport.CloseIdleConnections()
			readStarted := make(chan struct{})
			requestContexts := make(chan context.Context, 1)
			client.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requestContexts <- req.Context()
				res, err := client.Transport.RoundTrip(req)
				if err == nil {
					res.Body = &observedBody{ReadCloser: res.Body, readStarted: readStarted}
				}
				return res, err
			})
			responses := make(chan *Response, 1)
			sendReturned := make(chan struct{})
			go func() {
				client.Send(NewRequestWithParams(GET, server.URL), responses)
				close(sendReturned)
			}()
			requestContext := <-requestContexts
			select {
			case <-readStarted:
			case <-time.After(2 * time.Second):
				t.Fatal("response body read did not start")
			}
			if err := requestContext.Err(); err != nil {
				t.Errorf("request context canceled before reading its body: %v", err)
			}
			releaseBody()
			result := <-responses
			if result.Error != "" {
				t.Fatalf("unexpected error: %s", result.Error)
			}
			if result.StatusCode != http.StatusCreated || result.Body != "first second" {
				t.Fatalf("response = %#v, want status 201 and the complete body", result)
			}
			if result.RoundTrip != roundTrip {
				t.Errorf("round trip timing = %v, want %v", result.RoundTrip, roundTrip)
			}
			select {
			case <-sendReturned:
			case <-time.After(time.Second):
				t.Fatal("Send did not return after delivering its response")
			}
			if err := requestContext.Err(); err != context.Canceled {
				t.Errorf("request context error after Send returned = %v, want context.Canceled", err)
			}
		})
	}
}

func TestClient_SendBodyTimeout(t *testing.T) {
	releaseBody := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("first "))
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-releaseBody:
		}
	}))
	defer server.Close()
	defer close(releaseBody)

	client := InitClient(200*time.Millisecond, true)
	defer client.Transport.CloseIdleConnections()
	readStarted := make(chan struct{})
	client.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		res, err := client.Transport.RoundTrip(req)
		if err == nil {
			res.Body = &observedBody{ReadCloser: res.Body, readStarted: readStarted}
		}
		return res, err
	})
	responses := make(chan *Response, 1)
	go client.Send(NewRequestWithParams(GET, server.URL), responses)
	select {
	case <-readStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("response body read did not start")
	}
	select {
	case result := <-responses:
		if result.Failure == nil || result.Failure.Code != apperror.Timeout {
			t.Fatalf("failure = %#v, want a response body timeout", result.Failure)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Send did not time out during the response body read")
	}
}

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
				if result.Failure == nil || result.Failure.Category != apperror.Network || result.Failure.Code != apperror.Timeout {
					t.Fatalf("failure = %#v, want a timeout network error", result.Failure)
				}
				if result.Failure.Hint == "" {
					t.Fatal("timeout failure did not include a recovery hint")
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
