// Package ws is a minimal WebSocket client used by wspane's live session
// screen - deliberately independent of internal/http's request/response
// model, since a WebSocket connection is long-lived and bidirectional
// rather than a single round trip.
package ws

import (
	"context"
	"errors"
	"time"

	"github.com/coder/websocket"
)

// dialTimeout bounds how long Connect waits for the initial handshake -
// the connection itself, once established, has no timeout (see Session's
// own long-lived ctx).
const dialTimeout = 10 * time.Second

// sendTimeout bounds a single Send call - a write that never completes
// (e.g. a dead connection the read side hasn't noticed yet) would
// otherwise block whatever tea.Cmd called it indefinitely.
const sendTimeout = 5 * time.Second

// EntryKind identifies what a LogEntry represents.
type EntryKind int

const (
	EntrySent EntryKind = iota
	EntryReceived
	// EntrySystem is a connection lifecycle notice (connected, closed,
	// error) rather than an actual WebSocket message - shown in the same
	// log so the whole session's timeline reads as one list.
	EntrySystem
)

// LogEntry is one line of a session's message log.
type LogEntry struct {
	Kind EntryKind
	Text string
	At   time.Time
}

// Event is what a Session's read loop reports asynchronously via
// Events() - either a received message (Entry set, Closed false) or the
// connection closing (Closed true; Err is nil for an expected, clean
// close - see Session.Close - and non-nil for a dropped or refused
// connection).
type Event struct {
	Entry  LogEntry
	Closed bool
	Err    error
}

// Session is one WebSocket connection's lifecycle - dial once (Connect),
// read the resulting Events() until a Closed one arrives, Send zero or
// more messages meanwhile, optionally Close it yourself.
type Session struct {
	conn   *websocket.Conn
	events chan Event
	cancel context.CancelFunc
}

// Connect dials rawURL (ws:// or wss://) and starts a background read
// loop forwarding every received message - and the eventual close - to
// Events(). The dial itself blocks under dialTimeout; callers invoke this
// from a tea.Cmd (see wspane's connectCmd), the same way http.Client.Send
// is only ever called from inside a Cmd, never directly from Update.
func Connect(rawURL string) (*Session, error) {
	dialCtx, dialCancel := context.WithTimeout(context.Background(), dialTimeout)
	defer dialCancel()

	conn, _, err := websocket.Dial(dialCtx, rawURL, nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{conn: conn, events: make(chan Event, 32), cancel: cancel}
	go s.readLoop(ctx)
	return s, nil
}

// Events returns the channel Connect's read loop reports on - closed
// (the Go channel itself) right after the one Event with Closed set to
// true, so a range over it, or a single receive that checks ok, both
// terminate correctly.
func (s *Session) Events() <-chan Event {
	return s.events
}

func (s *Session) readLoop(ctx context.Context) {
	defer close(s.events)
	for {
		_, data, err := s.conn.Read(ctx)
		if err != nil {
			s.events <- Event{Closed: true, Err: closeErr(err)}
			return
		}
		s.events <- Event{Entry: LogEntry{Kind: EntryReceived, Text: string(data), At: time.Now()}}
	}
}

// closeErr suppresses the error a user-initiated Close() causes its own
// blocked Read to return (a plain context cancellation or normal closure,
// not a real problem) so the UI doesn't report an intentional disconnect
// as a failure.
func closeErr(err error) error {
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// Send writes text as one text frame. Unlike a received message, the
// caller already knows the outcome the moment this returns, so it's up
// to the caller to append its own EntrySent LogEntry on success - Send
// itself doesn't touch Events().
func (s *Session) Send(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	return s.conn.Write(ctx, websocket.MessageText, []byte(text))
}

// Close disconnects cleanly - the read loop's blocked Read call unblocks
// with an error closeErr recognizes as expected, so exactly one Closed
// event (Err: nil) still reaches Events(), the same path a remote-
// initiated close or a real connection error also reports through.
func (s *Session) Close() {
	s.cancel()
	_ = s.conn.Close(websocket.StatusNormalClosure, "")
}
