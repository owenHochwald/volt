package requestpane

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

// TestRequestPane_ScrollFieldsDown_RevealsFieldsBeyondTheBox is a
// regression test for a real bug found while building this: PageDown
// appeared to do nothing at all, in any terminal size, once a request had
// enough fields (e.g. a wide imported body schema) that they didn't fit in
// the box. Two separate issues combined to cause it:
//
//  1. FieldsViewport.lines are only populated inside View(), which has a
//     value receiver - its SetContent call never persisted back to the
//     model Update() operates on, so from Update()'s perspective the
//     viewport's content was always empty, and pgdown against empty
//     content is a no-op (bubbles' AtBottom() reads true immediately).
//  2. FieldsViewport.Height was likewise only ever set inside View(), so
//     even after fixing (1), Update() would compute page-scroll amounts
//     against a Height of 0.
//
// Both must stay fixed together for this to pass.
func TestRequestPane_ScrollFieldsDown_RevealsFieldsBeyondTheBox(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{
		Name:    "test",
		Method:  http.POST,
		URL:     "https://api.example.com/small",
		Body:    `{"field1":"<string>","field2":"<string>","field3":"<string>"}`,
		Headers: map[string]string{},
	}
	pane.reinitRequestPane(req)
	pane.SetWidth(150)
	// A deliberately short box - enough for the method/URL line and Name,
	// not enough to also fit the Body Params section without scrolling.
	pane.SetHeight(7)
	pane.PanelFocused = true

	before := pane.View()
	if strings.Contains(before, "field3") {
		t.Fatal("expected field3 to start out scrolled below the visible box")
	}

	scrolled, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))

	if scrolled.FieldsViewport.YOffset() == 0 {
		t.Fatal("expected pgdown to move the scroll offset")
	}
	after := scrolled.View()
	if !strings.Contains(after, "field3") {
		t.Errorf("expected field3 to be visible after pgdown, got:\n%s", after)
	}

	// And pgup should bring it back to the top.
	back, _ := scrolled.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp}))
	if back.FieldsViewport.YOffset() != 0 {
		t.Errorf("expected pgup to return to the top, got YOffset=%d", back.FieldsViewport.YOffset())
	}
}

// TestRequestPane_ScrollToFocused_RevealsNewlyAddedArrayItem is a
// regression test for a real bug reported while using an actual OpenAPI
// spec: adding a second item to an array-of-objects body param (see
// ArrayParam) placed its fields below the box, but nothing scrolled to
// reveal them - the item existed and accepted keystrokes, but was
// completely invisible until manually paging down by the right amount.
func TestRequestPane_ScrollToFocused_RevealsNewlyAddedArrayItem(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"tags": {Type: "array", ItemsRefName: "Tag", Items: tagItemSchema()},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.SetWidth(150)
	// Short enough that item 2's fields don't fit alongside item 1's.
	pane.SetHeight(14)
	pane.PanelFocused = true

	// Tab order (this request has no group, so no ServerSelector - see
	// ShowServerSelector): Name, Method, URL, ArrayCountControl. Focus
	// starts on Method (see defaultFocusIndex), not Name.
	for i := 0; i < 2; i++ {
		pane.FocusManager.Next()
	}
	if _, ok := pane.FocusManager.Current().(*ArrayCountControl); !ok {
		t.Fatalf("expected the array count control to be focused, got %T", pane.FocusManager.Current())
	}

	// Add a second item, then tab into its first field - exactly what a
	// user filling it in would do.
	result, _ := pane.currentMode.HandleInput(&pane, tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))
	if len(result.ArrayParams[0].Items) != 2 {
		t.Fatalf("expected 2 items after adding one, got %d", len(result.ArrayParams[0].Items))
	}

	model2, _ := result.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})) // -> item1.key
	model3, _ := model2.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})) // -> item1.value
	final, _ := model3.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))  // -> item2.key

	focused := final.FocusManager.Current()
	if focused != &final.ArrayParams[0].Items[1].Fields[0].Input {
		t.Fatalf("expected item 2's key field to be focused, got %T", focused)
	}

	rendered := final.View()
	if !strings.Contains(rendered, "Item 2:") {
		t.Errorf("expected scrolling to have brought item 2 into view, got:\n%s", rendered)
	}
}

// TestRequestPane_FieldsContent_DescriptionShiftsMethodURLOffsetByOneLine is
// a regression test for the fieldsContent() refactor that replaced
// hardcoded fieldOffset line numbers with a dynamically computed
// primaryLineNum := len(elements) - 1 (see the description-under-Name line
// added alongside http.Request.Description). MethodSelector/URLInput's own
// recorded line - unlike PathParams/BodyParams/etc, which derive their
// offsets from lineCount() and so aren't at risk here - comes directly from
// primaryLineNum, so it must land on the exact right line whether or not a
// description is present, or scroll-to-focused would leave Method/URL
// scrolled just outside the visible box after tabbing to them.
func TestRequestPane_FieldsContent_DescriptionShiftsMethodURLOffsetByOneLine(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{Method: http.GET, URL: "https://api.example.com/widgets", Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.SetWidth(150)
	_, offsetsNoDesc := pane.fieldsContent()

	req.Description = "Lists every widget in the catalog."
	pane.reinitRequestPane(req)
	_, offsetsWithDesc := pane.fieldsContent()

	lineFor := func(offsets []fieldOffset, target ui.Focusable) int {
		for _, o := range offsets {
			if o.focusable == target {
				return o.line
			}
		}
		t.Fatalf("target field not found in offsets")
		return -1
	}

	// Asserted as absolute line numbers, not just "withDesc == noDesc + 1":
	// a uniform off-by-N bug in primaryLineNum's formula would shift both
	// sides by the same amount and still satisfy a same-delta check.
	if got := lineFor(offsetsNoDesc, pane.MethodSelector); got != 2 {
		t.Errorf("expected MethodSelector at line 2 with no description (nameLine, blank, primaryLine), got %d", got)
	}
	if got := lineFor(offsetsWithDesc, pane.MethodSelector); got != 3 {
		t.Errorf("expected MethodSelector at line 3 with a description (nameLine, descLine, blank, primaryLine), got %d", got)
	}
}
