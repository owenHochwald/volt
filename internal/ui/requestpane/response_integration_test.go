package requestpane

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/responsepane"
)

// TestRequestPane_SendThenConnectionError_StillShowsBodyAndTiming is an
// end-to-end regression test for a real bug report: filling in a request's
// required fields, sending it, and getting a connection error (the target
// host refused the connection) left the Response pane's Request tab showing
// no body and the Timing tab showing nothing useful - even though the
// request was genuinely built and dispatched (the connection error only
// happens *after* a real send attempt). This drives the exact same path a
// live send takes - resolvedRequest() with real BodyParams filled in - and
// feeds its result into a real ResponsePane exactly as
// internal/app/update.go's http.ResultMsg handler does, to catch any gap
// between what's actually sent and what's shown for it.
func TestRequestPane_SendThenConnectionError_StillShowsBodyAndTiming(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name":  {Type: "string", Required: true},
		"email": {Type: "string", Required: true},
		"age":   {Type: "integer", Required: true},
	})
	req := &http.Request{Method: http.POST, URL: "http://127.0.0.1:1/host", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	if len(pane.BodyParams) != 3 {
		t.Fatalf("expected 3 required body params, got %+v", pane.BodyParams)
	}
	values := map[string]string{"name": "server-1", "email": "a@b.com", "age": "5"}
	for i := range pane.BodyParams {
		v, ok := values[pane.BodyParams[i].Name]
		if !ok {
			t.Fatalf("unexpected body param %q", pane.BodyParams[i].Name)
		}
		pane.BodyParams[i].Input.SetValue(v)
	}

	// Mirrors handleSubmit's own send-attempt path (see mode_normal.go) up
	// to the point of actually dispatching the HTTP call, so this exercises
	// the same resolvedRequest() a real send would use.
	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyCtrlP})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}
	if !result.RequestInProgress {
		t.Fatalf("expected filled required fields to pass validation and start sending, got ParseErrors=%v", result.ParseErrors)
	}

	sentReq := result.resolvedRequest()
	if sentReq.Body == "" || sentReq.Body == "{}" {
		t.Fatalf("expected resolvedRequest() to carry the filled-in body, got %q", sentReq.Body)
	}
	for _, want := range []string{"server-1", "a@b.com"} {
		if !strings.Contains(sentReq.Body, want) {
			t.Errorf("expected resolved request body to contain %q, got %q", want, sentReq.Body)
		}
	}

	// A real connection-refused error - Duration stays 0 since no round
	// trip ever completed, exactly like internal/http.Client.Send's
	// err != nil branch (client.go) which returns &Response{Error: ...}
	// with every other field at its zero value.
	errResp := &http.Response{Error: `Post "http://127.0.0.1:1/host": dial tcp 127.0.0.1:1: connect: connection refused`}

	rp := responsepane.SetupResponsePane(nil, keybindings.DefaultKeyMap())
	rp.SetWidth(200)
	rp.SetHeight(50)
	// Mirrors internal/app/update.go's http.ResultMsg case exactly:
	// m.responsePane.SetResponse(msg.Response, msg.Request).
	rp.SetResponse(errResp, sentReq)

	tabModel, _ := rp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	afterTab4, ok := tabModel.(*responsepane.ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", tabModel)
	}
	requestTabContent := afterTab4.View()
	for _, want := range []string{"POST", sentReq.URL, "server-1", "a@b.com"} {
		if !strings.Contains(requestTabContent, want) {
			t.Errorf("expected the Request tab to show the sent request body even after a connection error, missing %q, got:\n%s", want, requestTabContent)
		}
	}

	timingModel, _ := afterTab4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	afterTab3, ok := timingModel.(*responsepane.ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", timingModel)
	}
	timingContent := afterTab3.View()
	if !strings.Contains(timingContent, "Request Timing") || !strings.Contains(timingContent, "no timing data available") {
		t.Errorf("expected the Timing tab to explain a failed send has no real duration to show (not a misleading 0s/0ms), got:\n%s", timingContent)
	}
}
