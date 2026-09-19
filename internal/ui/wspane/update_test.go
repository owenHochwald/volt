package wspane

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/coder/websocket"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ws"
)

func setupTestPane(t *testing.T) WSPane {
	t.Helper()
	pane := SetupWSPane(keybindings.DefaultKeyMap())
	pane.Open()
	return pane
}

func keyRunes(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: s, Code: rune(s[0])})
}

func enterMsg() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
}

func tabMsg() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
}

// TestWSPane_TabCyclesThroughThreeFields verifies URL/Connect/Compose is
// a fixed 3-stop cycle.
func TestWSPane_TabCyclesThroughThreeFields(t *testing.T) {
	pane := setupTestPane(t)
	if pane.focus != focusURL {
		t.Fatalf("expected to start on focusURL, got %d", pane.focus)
	}

	for i, want := range []focusField{focusConnectButton, focusCompose, focusURL} {
		pane, _ = pane.Update(tabMsg())
		if pane.focus != want {
			t.Errorf("tab %d: expected focus=%d, got %d", i+1, want, pane.focus)
		}
	}
}

// TestWSPane_Esc_EmitsCloseWithoutTearingDownASession verifies esc only
// signals the parent to hide the modal - it must not touch the session
// itself (see the package doc: hiding isn't disconnecting).
func TestWSPane_Esc_EmitsCloseWithoutTearingDownASession(t *testing.T) {
	pane := setupTestPane(t)
	pane.status = statusConnected

	_, cmd := pane.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if cmd == nil {
		t.Fatal("expected a close command")
	}
	if _, ok := cmd().(CloseWSModalMsg); !ok {
		t.Fatalf("expected CloseWSModalMsg, got %T", cmd())
	}
}

// TestWSPane_Enter_OnEmptyURL_DoesNotDial verifies pressing enter without
// a URL reports a system note instead of trying to connect to "".
func TestWSPane_Enter_OnEmptyURL_DoesNotDial(t *testing.T) {
	pane := setupTestPane(t)

	pane, cmd := pane.Update(enterMsg())
	if cmd != nil {
		t.Error("expected no connect command for an empty URL")
	}
	if len(pane.log) != 1 || !strings.Contains(pane.log[0].Text, "Enter a URL") {
		t.Errorf("expected a system note about the missing URL, got %+v", pane.log)
	}
}

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

// TestWSPane_FullLifecycle_ConnectSendReceiveDisconnect drives WSPane
// through a real connection (against a local echo server) exactly the
// way the app would: typing a URL, pressing enter to connect, running
// the resulting Cmd synchronously (the same way bubbletea would, just
// without the runtime), composing and sending a message, receiving its
// echo, then disconnecting.
func TestWSPane_FullLifecycle_ConnectSendReceiveDisconnect(t *testing.T) {
	server := echoServer(t)
	pane := setupTestPane(t)

	for _, r := range wsURL(server) {
		pane, _ = pane.Update(keyRunes(string(r)))
	}

	pane, cmd := pane.Update(enterMsg())
	if pane.status != statusConnecting {
		t.Fatalf("expected statusConnecting after enter, got %d", pane.status)
	}
	if cmd == nil {
		t.Fatal("expected a connect command")
	}

	connectResult := cmd()
	pane, cmd = pane.Update(connectResult)
	if pane.status != statusConnected {
		t.Fatalf("expected statusConnected, got %d (log: %+v)", pane.status, pane.log)
	}
	if cmd == nil {
		t.Fatal("expected the connect success to arm the event-wait command")
	}

	// Move to the compose field and send a message.
	pane, _ = pane.Update(tabMsg())
	pane, _ = pane.Update(tabMsg())
	if pane.focus != focusCompose {
		t.Fatalf("expected focusCompose, got %d", pane.focus)
	}
	for _, r := range "hello" {
		pane, _ = pane.Update(keyRunes(string(r)))
	}
	pane, sendCmdResult := pane.Update(enterMsg())
	if sendCmdResult == nil {
		t.Fatal("expected a send command")
	}
	if pane.composeInput.Value() != "" {
		t.Error("expected the compose field to clear immediately on send")
	}

	pane, _ = pane.Update(sendCmdResult())
	if len(pane.log) == 0 || pane.log[len(pane.log)-1].Kind != ws.EntrySent || pane.log[len(pane.log)-1].Text != "hello" {
		t.Fatalf("expected a sent log entry for %q, got %+v", "hello", pane.log)
	}

	// Drain the event-wait command until the echoed message arrives.
	waitCmd := cmd
	var gotEcho bool
	for i := 0; i < 5 && !gotEcho; i++ {
		msg := waitCmd()
		var eventCmd tea.Cmd
		pane, eventCmd = pane.Update(msg)
		if em, ok := msg.(EventMsg); ok && !em.Event.Closed && em.Event.Entry.Kind == ws.EntryReceived {
			gotEcho = true
			if em.Event.Entry.Text != "hello" {
				t.Errorf("expected echoed text %q, got %q", "hello", em.Event.Entry.Text)
			}
		}
		waitCmd = eventCmd
	}
	if !gotEcho {
		t.Fatal("never received the echoed message")
	}

	// Disconnect: enter on the URL/button focus toggles it off.
	pane, _ = pane.Update(tabMsg()) // compose -> URL
	if pane.focus != focusURL {
		t.Fatalf("expected focusURL, got %d", pane.focus)
	}
	pane, _ = pane.Update(enterMsg())

	select {
	case event, ok := <-func() <-chan ws.Event { return pane.session.Events() }():
		if ok && !event.Closed {
			t.Fatalf("expected a Closed event or a closed channel, got %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the disconnect to take effect")
	}
}
