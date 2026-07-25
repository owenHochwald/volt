package app

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/responsepane"
	"github.com/owenHochwald/Volt/internal/utils"
)

func TestStorageResultsSurfaceConfirmationsAndErrors(t *testing.T) {
	tests := []struct {
		name  string
		msg   any
		level ui.NotificationLevel
		text  string
		hint  string
	}{
		{
			name:  "save success",
			msg:   ui.RequestSavedMsg{Request: &http.Request{Name: "health"}},
			level: ui.NotificationSuccess,
			text:  "saved",
		},
		{
			name:  "save failure",
			msg:   ui.RequestSavedMsg{Err: errors.New("disk full")},
			level: ui.NotificationError,
			text:  "access saved",
			hint:  "disk space",
		},
		{
			name:  "delete success",
			msg:   ui.RequestDeletedMsg{ID: 42},
			level: ui.NotificationSuccess,
			text:  "deleted",
		},
		{
			name:  "delete failure",
			msg:   ui.RequestDeletedMsg{Err: errors.New("database locked")},
			level: ui.NotificationError,
			text:  "saved requests are busy",
			hint:  "try again",
		},
		{
			name:  "load failure",
			msg:   ui.RequestsLoadingMsg{Err: errors.New("corrupt database")},
			level: ui.NotificationError,
			text:  "update saved",
			hint:  "try again",
		},
		{
			name:  "copy success",
			msg:   responsepane.ResponseCopiedMsg{},
			level: ui.NotificationSuccess,
			text:  "copied",
		},
		{
			name:  "copy failure",
			msg:   responsepane.ResponseCopiedMsg{Err: errors.New("clipboard unavailable")},
			level: ui.NotificationError,
			text:  "couldn't copy",
			hint:  "clipboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newTestModel(t)
			updated, _ := model.Update(tt.msg)
			got := updated.(Model)

			if got.notification.Level != tt.level {
				t.Fatalf("level = %s, want %s", got.notification.Level, tt.level)
			}
			if !strings.Contains(strings.ToLower(got.notification.Text), tt.text) {
				t.Fatalf("notification = %q, want it to contain %q", got.notification.Text, tt.text)
			}
			if tt.hint != "" && !strings.Contains(strings.ToLower(got.notification.Hint), tt.hint) {
				t.Fatalf("notification hint = %q, want it to contain %q", got.notification.Hint, tt.hint)
			}
		})
	}
}

func TestRequestAndLoadTestOutcomesSurfaceStatus(t *testing.T) {
	tests := []struct {
		name  string
		msg   any
		level ui.NotificationLevel
		text  string
		hint  string
	}{
		{
			name: "request transport error",
			msg: http.ResultMsg{Response: &http.Response{
				Error: "connection refused",
			}},
			level: ui.NotificationError,
			text:  "connect to the server",
			hint:  "server is running",
		},
		{
			name: "request HTTP failure",
			msg: http.ResultMsg{Response: &http.Response{
				StatusCode: 503,
				Status:     "503 Service Unavailable",
			}},
			level: ui.NotificationError,
			text:  "server returned http 503",
			hint:  "try again",
		},
		{
			name:  "load-test runtime error",
			msg:   http.LoadTestErrorMsg{Error: errors.New("unable to start")},
			level: ui.NotificationError,
			text:  "couldn't start the load test",
			hint:  "settings",
		},
		{
			name: "load-test HTTP failures",
			msg: http.LoadTestCompleteMsg{Stats: &http.LoadTestStats{
				CompletedRequests: 3,
				FailedRequests:    2,
			}},
			level: ui.NotificationError,
			text:  "2 requests failed",
			hint:  "url and connection",
		},
		{
			name: "load-test success",
			msg: http.LoadTestCompleteMsg{Stats: &http.LoadTestStats{
				CompletedRequests: 3,
			}},
			level: ui.NotificationSuccess,
			text:  "3 requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newTestModel(t)
			updated, _ := model.Update(tt.msg)
			got := updated.(Model)

			if got.notification.Level != tt.level {
				t.Fatalf("level = %s, want %s", got.notification.Level, tt.level)
			}
			if !strings.Contains(strings.ToLower(got.notification.Text), tt.text) {
				t.Fatalf("notification = %q, want it to contain %q", got.notification.Text, tt.text)
			}
			if tt.hint != "" && !strings.Contains(strings.ToLower(got.notification.Hint), tt.hint) {
				t.Fatalf("notification hint = %q, want it to contain %q", got.notification.Hint, tt.hint)
			}
			if got.focusedPanel != utils.ResponsePanel {
				t.Fatalf("focused panel = %d, want response", got.focusedPanel)
			}
		})
	}
}

func TestNotificationIsVisibleInStatusLine(t *testing.T) {
	model := newTestModel(t)
	model.notification = ui.Notification{
		Level: ui.NotificationError,
		Text:  "load test failed",
		Hint:  "try again",
	}

	if view := model.View().Content; !strings.Contains(view, "load test failed") || !strings.Contains(view, "try again") {
		t.Fatalf("view does not contain notification:\n%s", view)
	}
}

func TestRequestErrorsHideRawTransportDetailsAndShowRecoveryHint(t *testing.T) {
	model := newTestModel(t)
	updated, _ := model.Update(http.ResultMsg{Response: &http.Response{Error: "dial tcp 127.0.0.1:8080: connect: connection refused"}})
	got := updated.(Model)

	if strings.Contains(got.notification.Text, "127.0.0.1") {
		t.Fatalf("notification exposes transport detail: %q", got.notification.Text)
	}
	if !strings.Contains(strings.ToLower(got.notification.Hint), "server is running") {
		t.Fatalf("notification hint = %q", got.notification.Hint)
	}
	if got.responsePane.Response == nil || strings.Contains(got.responsePane.Response.Error, "127.0.0.1") {
		t.Fatalf("response exposes transport detail: %#v", got.responsePane.Response)
	}
}

func TestUnexpectedUIPanicRecoversWithoutStackTrace(t *testing.T) {
	model := newTestModel(t)
	updated, cmd := recoverUpdate(model, func() (tea.Model, tea.Cmd) {
		panic("sensitive implementation detail")
	})
	got := updated.(Model)

	if cmd != nil {
		t.Fatal("panic recovery returned an unexpected command")
	}
	if strings.Contains(got.notification.Text, "sensitive") {
		t.Fatalf("panic detail leaked into notification: %q", got.notification.Text)
	}
	if !strings.Contains(got.notification.Hint, "?") {
		t.Fatalf("panic recovery hint = %q", got.notification.Hint)
	}
}

func TestLoadTestCancelBindingCancelsActiveRun(t *testing.T) {
	model := newTestModel(t)
	canceled := false
	model.loadTestCancel = func() {
		canceled = true
	}
	model.setFocusedPanel(utils.ResponsePanel)

	updated, _ := model.Update(appKeyPress('x', "", tea.ModCtrl))
	got := updated.(Model)

	if !canceled {
		t.Fatal("control-x did not cancel the active run")
	}
	if !got.loadTestCanceled {
		t.Fatal("model did not record user cancellation")
	}
	if got.notification.Level != ui.NotificationWarning {
		t.Fatalf("notification level = %s, want warning", got.notification.Level)
	}
}
