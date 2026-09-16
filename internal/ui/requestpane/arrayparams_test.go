package requestpane

import (
	"encoding/json"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
)

func tagItemSchema() map[string]http.BodyFieldSchema {
	return map[string]http.BodyFieldSchema{
		"key":   {Type: "string"},
		"value": {Type: "string"},
	}
}

func TestNewArrayParam_StartsWithOnePrePopulatedItem(t *testing.T) {
	a := newArrayParam("tags", "Tag", tagItemSchema())

	if len(a.Items) != 1 {
		t.Fatalf("expected 1 pre-populated item, got %d", len(a.Items))
	}
	if len(a.Items[0].Fields) != 2 {
		t.Fatalf("expected 2 fields (key, value), got %+v", a.Items[0].Fields)
	}
	if a.Items[0].Fields[0].Name != "key" || a.Items[0].Fields[1].Name != "value" {
		t.Errorf("expected fields sorted by name (key, value), got %q, %q", a.Items[0].Fields[0].Name, a.Items[0].Fields[1].Name)
	}
	if a.itemLabel() != "Tag" {
		t.Errorf("expected itemLabel to use ItemsRefName, got %q", a.itemLabel())
	}
}

func TestArrayParam_AddItem_AppendsUpToACap(t *testing.T) {
	a := newArrayParam("tags", "Tag", tagItemSchema())

	for i := 0; i < maxArrayItems+5; i++ {
		a.AddItem()
	}
	if len(a.Items) != maxArrayItems {
		t.Errorf("expected AddItem to stop at the cap %d, got %d", maxArrayItems, len(a.Items))
	}
}

func TestArrayParam_RemoveLastItem_StopsAtZero(t *testing.T) {
	a := newArrayParam("tags", "Tag", tagItemSchema())
	a.RemoveLastItem()
	if len(a.Items) != 0 {
		t.Fatalf("expected the only item to be removed, got %d", len(a.Items))
	}
	a.RemoveLastItem() // must not panic or go negative
	if len(a.Items) != 0 {
		t.Errorf("expected RemoveLastItem to be a no-op on an empty array, got %d", len(a.Items))
	}
}

// TestRequestPane_SyncBodyParams_SplitsScalarAndArrayFields verifies a body
// schema mixing a plain scalar field with an array-of-ref field ends up
// with the scalar in BodyParams and the array in ArrayParams, not both
// lumped into one or the other.
func TestRequestPane_SyncBodyParams_SplitsScalarAndArrayFields(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name": {Type: "string"},
		"tags": {Type: "array", ItemsRefName: "Tag", Items: tagItemSchema()},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 1 || pane.BodyParams[0].Name != "name" {
		t.Errorf("expected only 'name' in BodyParams, got %+v", pane.BodyParams)
	}
	if len(pane.ArrayParams) != 1 || pane.ArrayParams[0].Name != "tags" {
		t.Fatalf("expected 'tags' in ArrayParams, got %+v", pane.ArrayParams)
	}
	if pane.ArrayParams[0].CountControl == nil {
		t.Fatal("expected the array param's CountControl to be wired up")
	}
}

// TestBuildRequestBody_IncludesArrayParams verifies an ArrayParam's items
// end up as a JSON array of objects in the resolved request body.
func TestBuildRequestBody_IncludesArrayParams(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"tags": {Type: "array", ItemsRefName: "Tag", Items: tagItemSchema()},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.ArrayParams) != 1 {
		t.Fatalf("expected 1 array param, got %+v", pane.ArrayParams)
	}
	pane.ArrayParams[0].Items[0].Fields[0].Input.SetValue("env")
	pane.ArrayParams[0].Items[0].Fields[1].Input.SetValue("prod")
	pane.ArrayParams[0].AddItem()
	pane.ArrayParams[0].Items[1].Fields[0].Input.SetValue("team")
	pane.ArrayParams[0].Items[1].Fields[1].Input.SetValue("volt")

	resolved := pane.resolvedRequest()

	var got map[string][]map[string]string
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	tags := got["tags"]
	if len(tags) != 2 {
		t.Fatalf("expected 2 tag objects, got %v", tags)
	}
	if tags[0]["key"] != "env" || tags[0]["value"] != "prod" {
		t.Errorf("expected first tag {env:prod}, got %v", tags[0])
	}
	if tags[1]["key"] != "team" || tags[1]["value"] != "volt" {
		t.Errorf("expected second tag {team:volt}, got %v", tags[1])
	}

	// The stored request template must stay untouched.
	if pane.Request.Body != req.Body {
		t.Errorf("expected the stored request to keep the schema template, got %q", pane.Request.Body)
	}
}

// TestValidateArrayParams_FlagsInvalidItemField verifies a constraint
// violation inside an array item is reported with a "name[index].field"
// label, same as ValidateParams does for top-level fields.
func TestValidateArrayParams_FlagsInvalidItemField(t *testing.T) {
	a := newArrayParam("tags", "Tag", map[string]http.BodyFieldSchema{
		"key": {Type: "string", MinLength: intPtr(3)},
	})
	a.Items[0].Fields[0].Input.SetValue("ab") // too short

	errs := ValidateArrayParams([]ArrayParam{a})
	if len(errs) != 1 {
		t.Fatalf("expected 1 violation, got %v", errs)
	}
	if errs[0] != "tags[0].key: must be at least 3 character(s)" {
		t.Errorf("unexpected message: %q", errs[0])
	}
}

// TestRequestPane_ArrayCountControl_CyclesViaFocusManager reproduces the
// actual key-handling path (FocusManager.Current() + HandleInput): cycling
// the count control forward must add an item AND make its fields reachable
// via the focus cycle immediately, not just mutate the ArrayParam in
// isolation.
func TestRequestPane_ArrayCountControl_CyclesViaFocusManager(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"tags": {Type: "array", ItemsRefName: "Tag", Items: tagItemSchema()},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	// Tab order: ServerSelector, Method, URL, Name -> ArrayCountControl.
	for i := 0; i < 4; i++ {
		pane.FocusManager.Next()
	}
	if _, ok := pane.FocusManager.Current().(*ArrayCountControl); !ok {
		t.Fatalf("expected the array count control to be focused, got %T", pane.FocusManager.Current())
	}

	beforeFieldCount := 0
	for _, item := range pane.ArrayParams[0].Items {
		beforeFieldCount += len(item.Fields)
	}

	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyRight})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}

	if len(result.ArrayParams[0].Items) != 2 {
		t.Fatalf("expected cycling forward to add a second item, got %d", len(result.ArrayParams[0].Items))
	}

	afterFieldCount := 0
	for _, item := range result.ArrayParams[0].Items {
		afterFieldCount += len(item.Fields)
	}
	if afterFieldCount <= beforeFieldCount {
		t.Errorf("expected more focusable fields after adding an item, before=%d after=%d", beforeFieldCount, afterFieldCount)
	}

	// The new item's fields must actually be reachable via Tab.
	found := false
	for i := 0; i < 10; i++ {
		result.FocusManager.Next()
		if result.FocusManager.Current() == &result.ArrayParams[0].Items[1].Fields[0].Input {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected the newly added item's first field to be reachable via the focus cycle")
	}
}
