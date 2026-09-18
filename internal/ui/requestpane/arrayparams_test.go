package requestpane

import (
	"encoding/json"
	"strings"
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

	// Tab order (this request has no group, so no ServerSelector - see
	// ShowServerSelector): Name, Method, URL -> ArrayCountControl. Focus
	// starts on Method (see defaultFocusIndex), not Name.
	for i := 0; i < 2; i++ {
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

// TestNewSimpleArrayParam_StartsWithOnePrePopulatedUnnamedItem verifies a
// "simple" array (its items have no object structure - see
// http.BodyFieldSchema.ItemType, e.g. create_host's real-world
// "ipv4_pool: {items: {type: number}, type: array}") builds one item with
// exactly one unnamed field of that type, not an Enum picker and not a
// named sub-field group like a "complex" array gets.
func TestNewSimpleArrayParam_StartsWithOnePrePopulatedUnnamedItem(t *testing.T) {
	a := newSimpleArrayParam("ipv4_pool", "number")

	if !a.IsSimple() {
		t.Fatal("expected IsSimple() to be true for a simple array")
	}
	if len(a.Items) != 1 {
		t.Fatalf("expected 1 pre-populated item, got %d", len(a.Items))
	}
	if len(a.Items[0].Fields) != 1 {
		t.Fatalf("expected exactly 1 unnamed field, got %+v", a.Items[0].Fields)
	}
	f := a.Items[0].Fields[0]
	if f.Name != simpleArrayItemName {
		t.Errorf("expected the item field to be unnamed, got %q", f.Name)
	}
	if f.Type != "number" {
		t.Errorf("expected the item field's type to be %q, got %q", "number", f.Type)
	}
	if f.Enum != nil {
		t.Error("expected a plain Input, not an Enum picker")
	}
	if a.itemLabel() != "number" {
		t.Errorf("expected itemLabel to fall back to the item type %q, got %q", "number", a.itemLabel())
	}
}

// TestRequestPane_SimpleArrayField_BuildsFlatJSONArray is an end-to-end
// regression test for the real bug report against create_host's
// ipv4_pool field: a "simple" array used to fall through to a single plain
// text field with no way to actually express "an array of values" at all.
// It should now be a repeatable list (add/remove items, one independent
// input per item - avoiding any delimiter/escaping question for values
// that might contain a comma or quote) that serializes to a real flat JSON
// array of the declared item type.
func TestRequestPane_SimpleArrayField_BuildsFlatJSONArray(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"ipv4_pool": {Type: "array", ItemType: "number"},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/host", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.ArrayParams) != 1 {
		t.Fatalf("expected 1 array param, got BodyParams=%+v ArrayParams=%+v", pane.BodyParams, pane.ArrayParams)
	}
	a := &pane.ArrayParams[0]
	if !a.IsSimple() {
		t.Fatal("expected ipv4_pool to become a simple ArrayParam")
	}

	a.Items[0].Fields[0].Input.SetValue("10")
	a.AddItem()
	a.Items[1].Fields[0].Input.SetValue("20")
	a.AddItem()
	a.Items[2].Fields[0].Input.SetValue("30")

	resolved := pane.resolvedRequest()
	var got map[string][]float64
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	want := []float64{10, 20, 30}
	if len(got["ipv4_pool"]) != len(want) {
		t.Fatalf("expected ipv4_pool=%v, got %v", want, got["ipv4_pool"])
	}
	for i := range want {
		if got["ipv4_pool"][i] != want[i] {
			t.Errorf("expected ipv4_pool[%d]=%v, got %v", i, want[i], got["ipv4_pool"][i])
		}
	}
}

// TestRequestPane_SimpleArrayField_CommaAndQuoteNeedNoEscaping verifies a
// string item containing a comma and a double quote - the two characters
// that would need escaping (or be ambiguous with the delimiter) in a
// comma-separated single-field format - round-trips byte-for-byte, since
// each item is its own independent input with no delimiter involved at
// all; the quote is auto-escaped only at the JSON-encoding step (see
// rawJSONStringValue), never something the user has to do by hand.
func TestRequestPane_SimpleArrayField_CommaAndQuoteNeedNoEscaping(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"labels": {Type: "array", ItemType: "string"},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/host", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	a := &pane.ArrayParams[0]
	a.Items[0].Fields[0].Input.SetValue(`a,b"c`)

	resolved := pane.resolvedRequest()
	var got map[string][]string
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	want := `a,b"c`
	if len(got["labels"]) != 1 || got["labels"][0] != want {
		t.Errorf("expected labels=[%q], got %v", want, got["labels"])
	}
}

// TestRequestPane_StringField_ValidEscapeSequencePassesThroughUnescaped
// verifies typing a real JSON escape sequence directly into a string field
// (e.g. "\n", the two characters backslash+n) reaches the server as that
// escape sequence - a genuine newline once *the server* parses the JSON -
// rather than being re-escaped into the literal four characters "\\n"
// (which a server would parse back as text "\n", not an actual line
// break). This is what a downstream JSON parser (here, encoding/json
// standing in for "the server") sees once it decodes our request body.
func TestRequestPane_StringField_ValidEscapeSequencePassesThroughUnescaped(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"note": {Type: "string"}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.BodyParams[0].Input.SetValue(`line1\nline2`)

	if msg, ok := ValidateParam(pane.BodyParams[0]); !ok {
		t.Fatalf("expected a valid \\n escape sequence to pass validation, got %q", msg)
	}

	resolved := pane.resolvedRequest()
	var got map[string]string
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	want := "line1\nline2" // an actual newline, not the two characters \ and n
	if got["note"] != want {
		t.Errorf("expected note=%q (with a real newline), got %q", want, got["note"])
	}
}

// TestRequestPane_StringField_InvalidEscapeSequenceBlocksSend verifies a
// backslash that isn't part of a valid JSON escape (e.g. a Windows path
// like "C:\Users") is caught by validation and blocks sending, rather than
// reaching BuildRequestBody and silently corrupting the *entire* request
// body into "{}" (its defensive fallback on a JSON marshal error).
func TestRequestPane_StringField_InvalidEscapeSequenceBlocksSend(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"note": {Type: "string"}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.BodyParams[0].Input.SetValue(`C:\Users\name`)

	if _, ok := ValidateParam(pane.BodyParams[0]); ok {
		t.Error("expected a bare backslash with no valid JSON escape after it to fail validation")
	}
}

// TestRequestPane_SimpleArrayField_OmitsUntouchedItems verifies an item
// added but never typed into is left out of the sent JSON, same "no
// value, don't send it" rule a top-level param follows (see
// BuildRequestBody).
func TestRequestPane_SimpleArrayField_OmitsUntouchedItems(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"ipv4_pool": {Type: "array", ItemType: "number"},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/host", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	a := &pane.ArrayParams[0]
	a.Items[0].Fields[0].Input.SetValue("10")
	a.AddItem() // left empty on purpose

	resolved := pane.resolvedRequest()
	var got map[string][]float64
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	if len(got["ipv4_pool"]) != 1 || got["ipv4_pool"][0] != 10 {
		t.Errorf("expected the untouched second item to be omitted, got %v", got["ipv4_pool"])
	}
}

// TestValidateArrayParams_SimpleArray_FlagsBadEntryWithoutDanglingDot
// verifies an invalid entry in a simple array's item produces an error
// message naming "name[index]" rather than "name[index]." with a
// dangling, meaningless empty field name (see arrayItemErrorPrefix).
func TestValidateArrayParams_SimpleArray_FlagsBadEntryWithoutDanglingDot(t *testing.T) {
	a := newSimpleArrayParam("ipv4_pool", "number")
	a.Items[0].Fields[0].Input.SetValue("not-a-number")

	errs := ValidateArrayParams([]ArrayParam{a})
	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %v", errs)
	}
	if got := errs[0]; !strings.HasPrefix(got, "ipv4_pool[0]:") {
		t.Errorf("expected an error prefixed %q (no dangling dot), got %q", "ipv4_pool[0]:", got)
	}
}
