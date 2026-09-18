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
			text:  "disk full",
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
			text:  "database locked",
		},
		{
			name:  "load failure",
			msg:   ui.RequestsLoadingMsg{Err: errors.New("corrupt database")},
			level: ui.NotificationError,
			text:  "corrupt database",
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
			text:  "clipboard unavailable",
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
		})
	}
}

func TestRequestAndLoadTestOutcomesSurfaceStatus(t *testing.T) {
	tests := []struct {
		name  string
		msg   any
		level ui.NotificationLevel
		text  string
	}{
		{
			name: "request transport error",
			msg: http.ResultMsg{Response: &http.Response{
				Error: "connection refused",
			}},
			level: ui.NotificationError,
			text:  "connection refused",
		},
		{
			name: "request HTTP failure",
			msg: http.ResultMsg{Response: &http.Response{
				StatusCode: 503,
				Status:     "503 Service Unavailable",
			}},
			level: ui.NotificationError,
			text:  "503",
		},
		{
			name:  "load-test runtime error",
			msg:   http.LoadTestErrorMsg{Error: errors.New("unable to start")},
			level: ui.NotificationError,
			text:  "unable to start",
		},
		{
			name: "load-test HTTP failures",
			msg: http.LoadTestCompleteMsg{Stats: &http.LoadTestStats{
				CompletedRequests: 3,
				FailedRequests:    2,
			}},
			level: ui.NotificationError,
			text:  "2 failed",
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
	}

	if view := model.View().Content; !strings.Contains(view, "load test failed") {
		t.Fatalf("view does not contain notification:\n%s", view)
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
