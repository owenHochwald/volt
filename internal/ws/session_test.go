package ws

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// echoServer accepts one WebSocket connection and echoes every text
// message it receives back verbatim, until the client closes.
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			if err := conn.Write(r.Context(), websocket.MessageText, data); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func wsURL(server *httptest.Server) string {
	return "ws" + strings.TrimPrefix(server.URL, "http")
}

// unixEchoServer starts the same echo handler as echoServer, but listening
// on a fresh Unix domain socket in t.TempDir() rather than a TCP port -
// returns the "unix://<socket-path>" URL Connect expects (see
// utils.ParseUnixSocketURL).
func unixEchoServer(t *testing.T) string {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "test.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			if err := conn.Write(r.Context(), websocket.MessageText, data); err != nil {
				return
			}
		}
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	return "unix://" + socketPath
}

// TestConnect_UnixSocket_SendAndReceive_Echo verifies Connect recognizes a
// "unix://" URL and dials the named Unix domain socket instead of a TCP
// host - the debugging scenario this exists for (a local daemon speaking
// WebSocket over a socket, not a port).
func TestConnect_UnixSocket_SendAndReceive_Echo(t *testing.T) {
	session, err := Connect(unixEchoServer(t))
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	if err := session.Send("hello over unix socket"); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case event := <-session.Events():
		if event.Closed {
			t.Fatalf("expected a message event, got a close: %+v", event)
		}
		if event.Entry.Text != "hello over unix socket" {
			t.Errorf("expected echoed text %q, got %q", "hello over unix socket", event.Entry.Text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the echoed message")
	}
}

func TestConnect_SendAndReceive_Echo(t *testing.T) {
	server := echoServer(t)
	session, err := Connect(wsURL(server))
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	if err := session.Send("hello"); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case event := <-session.Events():
		if event.Closed {
			t.Fatalf("expected a message event, got a close: %+v", event)
		}
		if event.Entry.Kind != EntryReceived {
			t.Errorf("expected EntryReceived, got %v", event.Entry.Kind)
		}
		if event.Entry.Text != "hello" {
			t.Errorf("expected echoed text %q, got %q", "hello", event.Entry.Text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the echoed message")
	}
}

func TestConnect_BadURL_ReturnsError(t *testing.T) {
	_, err := Connect("ws://127.0.0.1:1/nope")
	if err == nil {
		t.Fatal("expected an error connecting to a closed port")
	}
}

// TestSession_Close_ReportsACleanCloseWithNoError verifies a
// user-initiated Close() surfaces as a Closed event with a nil Err - not
// as if something had gone wrong (see closeErr).
func TestSession_Close_ReportsACleanCloseWithNoError(t *testing.T) {
	server := echoServer(t)
	session, err := Connect(wsURL(server))
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	session.Close()

	select {
	case event := <-session.Events():
		if !event.Closed {
			t.Fatalf("expected a Closed event, got %+v", event)
		}
		if event.Err != nil {
			t.Errorf("expected a clean close (nil Err), got %v", event.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the close event")
	}

	// The channel itself is closed right after the Closed event, so a
	// further receive returns immediately with ok=false rather than
	// blocking forever.
	select {
	case _, ok := <-session.Events():
		if ok {
			t.Error("expected the events channel to be closed after the Closed event")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the events channel to close")
	}
}

// TestSession_RemoteClose_ReportsClosed verifies the server closing the
// connection (rather than the client) still surfaces as a Closed event.
func TestSession_RemoteClose_ReportsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		conn.Close(websocket.StatusNormalClosure, "bye")
	}))
	t.Cleanup(server.Close)

	session, err := Connect(wsURL(server))
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	select {
	case event := <-session.Events():
		if !event.Closed {
			t.Fatalf("expected a Closed event, got %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the remote close to be reported")
	}
}
