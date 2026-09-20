package http

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/owenHochwald/Volt/internal/apperror"
)

// unixSocketServer starts an HTTP server listening on a fresh Unix domain
// socket in t.TempDir() instead of a TCP port - the Unix-socket
// counterpart to httptest.NewServer, which only ever listens on TCP.
// Returns the socket's path; the server is closed automatically via
// t.Cleanup.
func unixSocketServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
	}
	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	return socketPath
}

// TestClient_Send_DoesNotCancelContextBeforeBodyIsRead is a regression
// test for a real bug: the request's timeout context used to be cancelled
// (see the old makeCustomRequest's deferred cancel()) as soon as
// c.Client.Do returned with the response *headers* - before Send got
// around to reading the body at all. A response small/fast enough to
// already be fully buffered by then happened to still work (nothing left
// to block on), but any response that genuinely streamed in more than one
// piece - this handler deliberately flushes headers, waits, then writes
// more - failed with "context canceled" on the body read, even though the
// request itself succeeded and nothing was actually slow enough to
// legitimately time out.
func TestClient_Send_DoesNotCancelContextBeforeBodyIsRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("first-chunk-"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(50 * time.Millisecond)
		w.Write([]byte("second-chunk"))
	}))
	defer server.Close()

	client := InitClient(500*time.Millisecond, false)
	req := &Request{Method: http.MethodGet, URL: server.URL}

	res := make(chan *Response)
	go client.Send(req, res)
	result := <-res

	if result.Error != "" {
		t.Fatalf("expected no error reading the body, got %q", result.Error)
	}
	want := "first-chunk-second-chunk"
	if result.Body != want {
		t.Errorf("expected body %q, got %q", want, result.Body)
	}
}

// TestClient_Send_UnixSocket verifies a "unix://" URL (see
// utils.ParseUnixSocketURL) is dialed over the named Unix domain socket
// instead of being sent to a real TCP host - the debugging scenario this
// exists for (a local daemon listening on a socket, not a port).
func TestClient_Send_UnixSocket(t *testing.T) {
	var gotPath string
	socketPath := unixSocketServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte("ok from unix socket"))
	})

	client := InitClient(time.Second, false)
	req := &Request{Method: http.MethodGet, URL: "unix://" + socketPath + ":/health"}

	res := make(chan *Response)
	go client.Send(req, res)
	result := <-res

	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Body != "ok from unix socket" {
		t.Errorf("expected body %q, got %q", "ok from unix socket", result.Body)
	}
	if gotPath != "/health" {
		t.Errorf("expected the server to see request path %q, got %q", "/health", gotPath)
	}
}

// TestClient_Send_UnixSocket_BareSocketPathDefaultsToRoot verifies a
// "unix://<path>" URL with no request path at all (see
// utils.ParseUnixSocketURL) hits "/" on the socket, for a bare health
// check against a daemon's root.
func TestClient_Send_UnixSocket_BareSocketPathDefaultsToRoot(t *testing.T) {
	var gotPath string
	socketPath := unixSocketServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte("root"))
	})

	client := InitClient(time.Second, false)
	req := &Request{Method: http.MethodGet, URL: "unix://" + socketPath}

	res := make(chan *Response)
	go client.Send(req, res)
	result := <-res

	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if gotPath != "/" {
		t.Errorf("expected the server to see request path %q, got %q", "/", gotPath)
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

// TestClient_Send_PopulatesTiming verifies a successful send fills in
// Response.Timing - the first request on a fresh client opens its own
// connection (Reused false, TCPConnection > 0), a second request to the
// same server reuses it (Reused true, TCPConnection == 0) since InitClient
// keeps a client-wide keep-alive transport across calls.
func TestClient_Send_PopulatesTiming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := InitClient(time.Second, false)

	send := func() *Response {
		res := make(chan *Response)
		go client.Send(&Request{Method: http.MethodGet, URL: server.URL}, res)
		return <-res
	}

	first := send()
	if first.Error != "" {
		t.Fatalf("unexpected error: %v", first.Error)
	}
	if first.Timing == nil {
		t.Fatal("expected Timing to be populated")
	}
	if first.Timing.Reused {
		t.Error("expected the first request's connection not to be reused")
	}
	if first.Timing.TCPConnection <= 0 {
		t.Error("expected a nonzero TCP connection time for a fresh connection")
	}
	if first.Timing.TimeToFirstByte <= 0 {
		t.Error("expected a nonzero time to first byte")
	}

	second := send()
	if second.Timing == nil {
		t.Fatal("expected Timing to be populated")
	}
	if !second.Timing.Reused {
		t.Error("expected the second request to reuse the first's connection")
	}
	if second.Timing.TCPConnection != 0 {
		t.Errorf("expected no TCP connection time on a reused connection, got %s", second.Timing.TCPConnection)
	}
}

// TestClient_Send_NoTimingOnFailure verifies a request that never reaches
// a server (so Client.Send returns before assembling a Response at all)
// leaves Timing nil rather than a zero-valued, misleadingly "measured"
// struct.
func TestClient_Send_NoTimingOnFailure(t *testing.T) {
	client := InitClient(200*time.Millisecond, false)
	res := make(chan *Response)
	go client.Send(&Request{Method: http.MethodGet, URL: "http://127.0.0.1:1"}, res)
	result := <-res

	if result.Error == "" {
		t.Fatal("expected an error connecting to a closed port")
	}
	if result.Timing != nil {
		t.Errorf("expected no Timing on a failed request, got %+v", result.Timing)
	}
}
