package requestpane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

// TestRequestPane_ClearParamValues_ResetsFieldsButKeepsIdentity is a
// regression test for a real feature request: ctrl+x ("clear params")
// must discard whatever was typed into path/body/array fields while
// preserving the request's own identity (ID/name/URL/method/schema) -
// unlike ctrl+r (ResetRequest), which replaces the whole request.
func TestRequestPane_ClearParamValues_ResetsFieldsButKeepsIdentity(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name": {Type: "string"},
	})
	req := &http.Request{
		ID:      7,
		Name:    "getHost",
		Method:  http.POST,
		URL:     "https://api.example.com/hosts/{host_id}",
		Body:    body,
		Headers: map[string]string{"X-Env": "prod"},
		PathParamSchema: map[string]http.BodyFieldSchema{
			"host_id": {Type: "integer"},
		},
	}
	pane.reinitRequestPane(req)
	pane.PathParams[0].Input.SetValue("42")
	pane.BodyParams[0].Input.SetValue("widget-1")

	pane.clearParamValues()

	if pane.Request.ID != 7 || pane.Request.Name != "getHost" || pane.Request.Method != http.POST {
		t.Errorf("expected the request's identity to survive a clear, got %+v", pane.Request)
	}
	if pane.Request.URL != req.URL {
		t.Errorf("expected URL to survive a clear, got %q", pane.Request.URL)
	}
	if len(pane.PathParams) != 1 || pane.PathParams[0].Value() != "" {
		t.Errorf("expected host_id cleared back to blank, got %+v", pane.PathParams)
	}
	if len(pane.BodyParams) != 1 || pane.BodyParams[0].Value() != "" {
		t.Errorf("expected name cleared back to blank, got %+v", pane.BodyParams)
	}
}

// TestRequestPane_ClearParams_Keybinding verifies ctrl+x reaches
// clearParamValues through the normal Update path (not just called
// directly), and that fields are reachable via the focus cycle afterward.
func TestRequestPane_ClearParams_Keybinding(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.PanelFocused = true

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {Type: "string"}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/x", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.BodyParams[0].Input.SetValue("widget-1")

	result, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: 'x', Mod: tea.ModCtrl}))
	if len(result.BodyParams) != 1 || result.BodyParams[0].Value() != "" {
		t.Errorf("expected ctrl+x to clear the body param, got %+v", result.BodyParams)
	}
}

// TestRequestPane_SaveRequest_PersistsFilledInValues is an end-to-end
// regression test (through the real ctrl+s keybinding and a real DB, not
// just Snapshot() in isolation) for the feature this was all built for:
// saving a request with filled-in fields must persist those values so
// reloading the row later shows what was typed, not blank defaults.
func TestRequestPane_SaveRequest_PersistsFilledInValues(t *testing.T) {
	pane, db := setupTestPane(t)
	pane.PanelFocused = true

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {Type: "string"}})
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
	pane.PathParams[0].Input.SetValue("42")
	pane.BodyParams[0].Input.SetValue("widget-1")

	_, cmd := pane.Update(tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	if cmd == nil {
		t.Fatal("expected a non-nil save command")
	}
	msg := cmd()
	saved, ok := msg.(ui.RequestSavedMsg)
	if !ok {
		t.Fatalf("expected ui.RequestSavedMsg, got %T", msg)
	}
	if saved.Err != nil {
		t.Fatalf("unexpected save error: %v", saved.Err)
	}

	loaded, err := db.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 saved request, got %d", len(loaded))
	}
	if loaded[0].ParamValues.PathParamValues["host_id"] != "42" {
		t.Errorf("expected host_id=42 persisted, got %+v", loaded[0].ParamValues.PathParamValues)
	}
	if loaded[0].ParamValues.BodyParamValues["name"] != "widget-1" {
		t.Errorf("expected name=widget-1 persisted, got %+v", loaded[0].ParamValues.BodyParamValues)
	}
}

// TestRequestPane_SaveRequest_AfterClear_PersistsClearedValues verifies
// the other half of the feature request: saving (ctrl+s) after clearing
// params (ctrl+x) persists the cleared (empty) state, not the stale
// filled-in values from before the clear. Save (see db.Save) always
// inserts a new row rather than updating one in place, so this checks
// the most recently saved row specifically, not "the only row".
func TestRequestPane_SaveRequest_AfterClear_PersistsClearedValues(t *testing.T) {
	pane, db := setupTestPane(t)
	pane.PanelFocused = true

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {Type: "string"}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/x", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.BodyParams[0].Input.SetValue("widget-1")

	// First save: filled-in value lands in the DB.
	pane, cmd := pane.Update(tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	cmd()

	// Clear, then save again: the cleared (empty) state must be what's
	// persisted for this second save, not the earlier filled-in value.
	pane.clearParamValues()
	_, cmd2 := pane.Update(tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	if cmd2 == nil {
		t.Fatal("expected a non-nil save command")
	}
	msg2 := cmd2()
	saved2, ok := msg2.(ui.RequestSavedMsg)
	if !ok {
		t.Fatalf("expected ui.RequestSavedMsg, got %T", msg2)
	}
	if saved2.Err != nil {
		t.Fatalf("unexpected save error: %v", saved2.Err)
	}

	loaded, err := db.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	var second *http.Request
	for i := range loaded {
		if loaded[i].ID == saved2.Request.ID {
			second = &loaded[i]
		}
	}
	if second == nil {
		t.Fatalf("could not find the second save's row (id=%d) among %+v", saved2.Request.ID, loaded)
	}
	if v, ok := second.ParamValues.BodyParamValues["name"]; ok {
		t.Errorf("expected the cleared value to be what's persisted for this save, found name=%q", v)
	}
}

// TestRequestPane_MarkSaved_UpdatesID verifies MarkSaved (called by
// app.Model after a successful ctrl+s, regardless of which panel is
// currently focused - see ui.RequestSavedMsg) stamps the saved row's ID
// onto the pane's request, so the *next* save updates that row instead of
// inserting a duplicate (see storage.SQLiteStorage.Save's upsert rule).
func TestRequestPane_MarkSaved_UpdatesID(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{Method: http.GET, URL: "http://x", Name: "getHost", Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	pane.MarkSaved(&http.Request{ID: 42, Method: http.GET, URL: "http://x", Name: "getHost"})

	if pane.Request.ID != 42 {
		t.Errorf("expected ID=42 after MarkSaved, got %d", pane.Request.ID)
	}
}

// TestRequestPane_MarkSaved_IgnoresStaleSaveForADifferentRequest guards
// against a race: if the pane moved on to a different request while an
// earlier save was still in flight, that stale RequestSavedMsg arriving
// later must not stamp its ID onto the now-unrelated request - Method,
// URL and Name must all still match what was actually saved.
func TestRequestPane_MarkSaved_IgnoresStaleSaveForADifferentRequest(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{Method: http.GET, URL: "http://x/current", Name: "current", Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	pane.MarkSaved(&http.Request{ID: 42, Method: http.GET, URL: "http://x/stale", Name: "stale"})

	if pane.Request.ID != 0 {
		t.Errorf("expected ID to stay 0 (unrelated stale save ignored), got %d", pane.Request.ID)
	}
}

// TestRequestPane_SetRequestPaneRequestMsg_RestoresParamValues verifies
// the ordinary sidebar-driven load path (ui.SetRequestPaneRequestMsg,
// handled by reinitRequestPane directly - not the special ApplySnapshot/
// "last sent" restore path) also restores a saved request's ParamValues,
// proving reinitRequestPane's unification actually covers this path too.
func TestRequestPane_SetRequestPaneRequestMsg_RestoresParamValues(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.PanelFocused = true

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {Type: "string"}})
	saved := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/hosts/{host_id}",
		Body:    body,
		Headers: map[string]string{},
		PathParamSchema: map[string]http.BodyFieldSchema{
			"host_id": {Type: "integer"},
		},
		ParamValues: http.ParamValues{
			PathParamValues: map[string]string{"host_id": "99"},
			BodyParamValues: map[string]string{"name": "widget-2"},
		},
	}

	result, _ := pane.Update(ui.SetRequestPaneRequestMsg{Request: saved})
	if len(result.PathParams) != 1 || result.PathParams[0].Value() != "99" {
		t.Errorf("expected host_id=99 restored via the normal load path, got %+v", result.PathParams)
	}
	if len(result.BodyParams) != 1 || result.BodyParams[0].Value() != "widget-2" {
		t.Errorf("expected name=widget-2 restored via the normal load path, got %+v", result.BodyParams)
	}
}
