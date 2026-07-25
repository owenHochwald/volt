package requestpane

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestPlainEnterDoesNotSubmitFromEditableField(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.FocusManager.Next()
	pane.URLInput.SetValue("https://example.com")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", 0))

	if updated.RequestInProgress {
		t.Fatal("plain enter started a request")
	}
	if cmd != nil {
		t.Fatal("plain enter produced a submit command")
	}
}

func TestRequestSubmissionKeyBindings(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "meta enter alias", key: keyPress(tea.KeyEnter, "", tea.ModAlt)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestRequestPane(t)
			pane.SetFocused(true)
			pane.FocusManager.Next()
			pane.URLInput.SetValue("https://example.com")

			updated, cmd := pane.Update(tt.key)

			if !updated.RequestInProgress {
				t.Fatal("submit binding did not start a request")
			}
			if cmd == nil {
				t.Fatal("submit binding did not produce a command")
			}
		})
	}
}

func TestEnterActivatesFocusedSendButton(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.URLInput.SetValue("https://example.com")
	pane.FocusManager.Current().Blur()
	pane.FocusManager = pane.currentMode.GetFocusManagerWithIndex(&pane, int(FieldSubmitButton))

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", 0))

	if !updated.RequestInProgress {
		t.Fatal("enter on the send button did not start a request")
	}
	if cmd == nil {
		t.Fatal("enter on the send button did not produce a command")
	}
}

func TestPrintableCommandKeysRemainEditable(t *testing.T) {
	tests := []string{"?", "q", "h", "j", "k", "l"}

	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			pane := newTestRequestPane(t)
			pane.SetFocused(true)
			pane.FocusManager.Next()

			updated, _ := pane.Update(keyPress(rune(text[0]), text, 0))

			if got := updated.URLInput.Value(); got != text {
				t.Fatalf("URL value = %q, want %q", got, text)
			}
		})
	}
}

func TestSetFocusedBlursAndRestoresCurrentControl(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.FocusManager.Next()
	if !pane.URLInput.Focused() {
		t.Fatal("URL input is not focused before leaving panel")
	}

	pane.SetFocused(false)
	if pane.URLInput.Focused() {
		t.Fatal("URL input remained focused after leaving panel")
	}

	pane.SetFocused(true)
	if !pane.URLInput.Focused() {
		t.Fatal("URL input focus was not restored on panel entry")
	}
}

func TestTabAndShiftTabOnlyNavigateFields(t *testing.T) {
	tests := []struct {
		name  string
		start FieldIndex
		key   tea.KeyPressMsg
		want  FieldIndex
	}{
		{
			name:  "tab advances",
			start: FieldMethodSelector,
			key:   keyPress(tea.KeyTab, "", 0),
			want:  FieldURL,
		},
		{
			name:  "shift tab moves backward",
			start: FieldURL,
			key:   keyPress(tea.KeyTab, "", tea.ModShift),
			want:  FieldMethodSelector,
		},
		{
			name:  "shift tab wraps",
			start: FieldMethodSelector,
			key:   keyPress(tea.KeyTab, "", tea.ModShift),
			want:  FieldSubmitButton,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestRequestPane(t)
			pane.FocusManager.Current().Blur()
			pane.FocusManager = pane.currentMode.GetFocusManagerWithIndex(&pane, int(tt.start))
			pane.SetFocused(true)

			updated, _ := pane.Update(tt.key)

			if got := FieldIndex(updated.FocusManager.CurrentIndex()); got != tt.want {
				t.Fatalf("focus = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestArrowKeysDoNotChangeFocusedField(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.FocusManager.Next()
	pane.URLInput.SetValue("https://example.com")
	pane.URLInput.CursorEnd()

	updated, _ := pane.Update(keyPress(tea.KeyLeft, "", 0))

	if got := updated.FocusManager.CurrentIndex(); got != int(FieldURL) {
		t.Fatalf("left arrow changed focus to index %d", got)
	}
	if got, want := updated.URLInput.Position(), len("https://example.com")-1; got != want {
		t.Fatalf("left arrow cursor position = %d, want %d", got, want)
	}
}

func TestInvalidHeadersBlockSubmissionAndSurfaceError(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.URLInput.SetValue("https://example.com")
	pane.Headers.SetValue("missing delimiter")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tea.ModAlt))

	if updated.RequestInProgress {
		t.Fatal("invalid headers started a request")
	}
	if cmd == nil {
		t.Fatal("invalid headers did not produce a notification")
	}
	msg, ok := cmd().(ui.NotificationMsg)
	if !ok {
		t.Fatalf("message type = %T, want ui.NotificationMsg", cmd())
	}
	if !strings.Contains(strings.ToLower(msg.Notification.Text), "header") {
		t.Fatalf("notification = %q, want header error", msg.Notification.Text)
	}
	if !strings.Contains(strings.ToLower(msg.Notification.Hint), "one header-name") {
		t.Fatalf("notification hint = %q, want header recovery guidance", msg.Notification.Hint)
	}
}

func TestInvalidLoadTestConfigBlocksStartAndSurfacesError(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.toggleLoadTestMode()
	pane.URLInput.SetValue("https://example.com")
	pane.LoadTestConcurrency.SetValue("not-a-number")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tea.ModAlt))

	if updated.RequestInProgress {
		t.Fatal("invalid load-test config remained in progress")
	}
	if cmd == nil {
		t.Fatal("invalid load-test config did not produce a notification")
	}
	msg, ok := cmd().(ui.NotificationMsg)
	if !ok {
		t.Fatalf("message type = %T, want ui.NotificationMsg", cmd())
	}
	if !strings.Contains(strings.ToLower(msg.Notification.Text), "concurrency") {
		t.Fatalf("notification = %q, want concurrency error", msg.Notification.Text)
	}
	if !strings.Contains(msg.Notification.Hint, "?") {
		t.Fatalf("notification hint = %q, want keybinding guidance", msg.Notification.Hint)
	}
}

func TestInvalidURLOffersLikelyCorrection(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetFocused(true)
	pane.URLInput.SetValue("example.com/health")

	updated, cmd := pane.Update(keyPress(tea.KeyEnter, "", tea.ModAlt))

	if updated.RequestInProgress {
		t.Fatal("invalid URL started a request")
	}
	msg, ok := cmd().(ui.NotificationMsg)
	if !ok {
		t.Fatalf("message type = %T, want ui.NotificationMsg", cmd())
	}
	if !strings.Contains(msg.Notification.Hint, "https://example.com/health") {
		t.Fatalf("notification hint = %q, want URL correction", msg.Notification.Hint)
	}
}

func newTestRequestPane(t *testing.T) RequestPane {
	t.Helper()

	db, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "volt.db"))
	if err != nil {
		t.Fatalf("create test storage: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test storage: %v", err)
		}
	})
	return SetupRequestPane(db, keybindings.DefaultKeyMap())
}

func keyPress(code rune, text string, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{
		Code: code,
		Text: text,
		Mod:  mod,
	})
}
