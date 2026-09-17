package requestpane

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
)

// TestRequestPane_Snapshot_CapturesFieldValues verifies Snapshot() records
// every currently filled-in path/body/array param value, keyed by name,
// and omits untouched ones (nothing typed, no schema default) the same
// way BuildRequestBody already does.
func TestRequestPane_Snapshot_CapturesFieldValues(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name":  {Type: "string"},
		"email": {Type: "string"}, // left untouched
		"tags":  {Type: "array", ItemsRefName: "Tag", Items: tagItemSchema()},
	})
	req := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/hosts/{host_id}",
		Body:    body,
		Headers: map[string]string{},
		PathParamSchema: map[string]http.BodyFieldSchema{
			"host_id": {Type: "integer"},
		},
	}
	pane.reinitRequestPane(req)

	for i := range pane.PathParams {
		pane.PathParams[i].Input.SetValue("42")
	}
	for i := range pane.BodyParams {
		if pane.BodyParams[i].Name == "name" {
			pane.BodyParams[i].Input.SetValue("widget-1")
		}
	}
	pane.ArrayParams[0].Items[0].Fields[0].Input.SetValue("env")
	pane.ArrayParams[0].Items[0].Fields[1].Input.SetValue("prod")

	snap := pane.Snapshot()

	if snap.ParamValues.PathParamValues["host_id"] != "42" {
		t.Errorf("expected host_id=42, got %+v", snap.ParamValues.PathParamValues)
	}
	if snap.ParamValues.BodyParamValues["name"] != "widget-1" {
		t.Errorf("expected name=widget-1, got %+v", snap.ParamValues.BodyParamValues)
	}
	if _, ok := snap.ParamValues.BodyParamValues["email"]; ok {
		t.Errorf("expected untouched 'email' to be omitted, got %+v", snap.ParamValues.BodyParamValues)
	}
	tags := snap.ParamValues.ArrayParamValues["tags"]
	if len(tags) != 1 || tags[0]["key"] != "env" || tags[0]["value"] != "prod" {
		t.Errorf("expected tags=[{key:env value:prod}], got %+v", tags)
	}
}

// TestRequestPane_ApplySnapshot_RestoresFieldValues is the round-trip
// counterpart: applying a snapshot to a fresh pane reproduces every field
// value, including growing an array to its saved item count.
func TestRequestPane_ApplySnapshot_RestoresFieldValues(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name": {Type: "string"},
		"tags": {Type: "array", ItemsRefName: "Tag", Items: tagItemSchema()},
	})
	snap := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/hosts/{host_id}",
		Body:    body,
		Headers: map[string]string{},
		PathParamSchema: map[string]http.BodyFieldSchema{
			"host_id": {Type: "integer"},
		},
		ParamValues: http.ParamValues{
			PathParamValues: map[string]string{"host_id": "42"},
			BodyParamValues: map[string]string{"name": "widget-1"},
			ArrayParamValues: map[string][]map[string]string{
				"tags": {
					{"key": "env", "value": "prod"},
					{"key": "team", "value": "volt"},
				},
			},
		},
	}

	pane.ApplySnapshot(snap)

	if len(pane.PathParams) != 1 || pane.PathParams[0].Value() != "42" {
		t.Fatalf("expected host_id=42, got %+v", pane.PathParams)
	}
	var nameVal string
	for _, p := range pane.BodyParams {
		if p.Name == "name" {
			nameVal = p.Value()
		}
	}
	if nameVal != "widget-1" {
		t.Errorf("expected name=widget-1, got %q", nameVal)
	}
	if len(pane.ArrayParams) != 1 || len(pane.ArrayParams[0].Items) != 2 {
		t.Fatalf("expected 2 restored tag items, got %+v", pane.ArrayParams)
	}
	item0 := pane.ArrayParams[0].Items[0]
	if item0.Fields[0].Value() != "env" || item0.Fields[1].Value() != "prod" {
		t.Errorf("expected first item {env prod}, got %+v", item0)
	}
	item1 := pane.ArrayParams[0].Items[1]
	if item1.Fields[0].Value() != "team" || item1.Fields[1].Value() != "volt" {
		t.Errorf("expected second item {team volt}, got %+v", item1)
	}

	// The restored fields must actually be reachable via the focus cycle,
	// not just correct in the underlying slices - AddItem() growing the
	// array requires rebuildFocusManager to pick up the new item.
	found := false
	for i := 0; i < 20; i++ {
		pane.FocusManager.Next()
		if pane.FocusManager.Current() == &pane.ArrayParams[0].Items[1].Fields[0].Input {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected the restored second tag item to be reachable via the focus cycle")
	}
}

// TestRequestPane_HandleSubmit_SavesSnapshotOnlyOnActualSend is an
// end-to-end regression test for the core rule the feature was asked for:
// a request that was never successfully sent (blocked by validation) must
// not be persisted as "the last sent request" - only an actual send
// attempt (validation passed, RequestInProgress became true) saves one.
func TestRequestPane_HandleSubmit_SavesSnapshotOnlyOnActualSend(t *testing.T) {
	pane, db := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name": {Type: "string", Required: true},
	})
	req := &http.Request{Method: http.POST, URL: "http://127.0.0.1:1/host", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true
	// "name" left empty on purpose - blocks the first attempt.

	model, cmd := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyCtrlP})
	blocked, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}
	if blocked.RequestInProgress {
		t.Fatal("expected the missing required field to block sending")
	}
	if cmd != nil {
		runAllCmds(cmd)
	}
	if got, err := db.LoadLastSentRequest(); err != nil || got != nil {
		t.Fatalf("expected nothing saved after a blocked send attempt, got snapshot=%+v err=%v", got, err)
	}

	// Fill in the required field and actually send.
	for i := range blocked.BodyParams {
		blocked.BodyParams[i].Input.SetValue("server-1")
	}
	model2, cmd2 := blocked.currentMode.HandleInput(blocked, tea.KeyMsg{Type: tea.KeyCtrlP})
	sent, ok := model2.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model2)
	}
	if !sent.RequestInProgress {
		t.Fatal("expected the filled-in request to actually send")
	}
	if cmd2 == nil {
		t.Fatal("expected a non-nil command batch for an actual send")
	}
	runAllCmds(cmd2)

	got, err := db.LoadLastSentRequest()
	if err != nil {
		t.Fatalf("unexpected error loading snapshot: %v", err)
	}
	if got == nil {
		t.Fatal("expected a snapshot to be saved after an actual send")
	}
	if got.ParamValues.BodyParamValues["name"] != "server-1" {
		t.Errorf("expected the saved snapshot to carry the filled-in value, got %+v", got.ParamValues.BodyParamValues)
	}
}

// runAllCmds executes cmd and, if it's a batch, every sub-command too -
// mirroring what the real bubbletea runtime does with a tea.Batch, so a
// fire-and-forget command dispatched from handleSubmit (see
// ui.SaveLastSentRequestCmd) actually runs in a test instead of just
// sitting unexecuted as a returned closure.
func runAllCmds(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			runAllCmds(sub)
		}
	}
}
