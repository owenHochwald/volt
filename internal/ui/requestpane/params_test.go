package requestpane

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
)

func intPtr(n int) *int           { return &n }
func floatPtr(f float64) *float64 { return &f }

func paramWithValue(t *testing.T, value string, constraints Param) Param {
	t.Helper()
	constraints.Input = newParamInput()
	constraints.Input.SetValue(value)
	return constraints
}

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name    string
		param   Param
		wantErr bool
	}{
		{"no constraints", paramWithValue(t, "anything", Param{Name: "note"}), false},
		{"min length satisfied", paramWithValue(t, "hello", Param{Name: "name", MinLength: intPtr(3)}), false},
		{"min length violated", paramWithValue(t, "hi", Param{Name: "name", MinLength: intPtr(3)}), true},
		{"max length satisfied", paramWithValue(t, "hi", Param{Name: "name", MaxLength: intPtr(5)}), false},
		{"max length violated", paramWithValue(t, "too long a value", Param{Name: "name", MaxLength: intPtr(5)}), true},
		{"minimum satisfied", paramWithValue(t, "10", Param{Name: "count", Minimum: floatPtr(1)}), false},
		{"minimum violated", paramWithValue(t, "0", Param{Name: "count", Minimum: floatPtr(1)}), true},
		{"maximum satisfied", paramWithValue(t, "50", Param{Name: "count", Maximum: floatPtr(100)}), false},
		{"maximum violated", paramWithValue(t, "500", Param{Name: "count", Maximum: floatPtr(100)}), true},
		{"non-numeric value with numeric constraint is not itself an error", paramWithValue(t, "not-a-number", Param{Name: "count", Minimum: floatPtr(1)}), false},
		{"integer type satisfied", paramWithValue(t, "42", Param{Name: "count", Type: "integer"}), false},
		{"integer type violated by non-numeric text", paramWithValue(t, "abc", Param{Name: "count", Type: "integer"}), true},
		{"integer type violated by a decimal", paramWithValue(t, "4.2", Param{Name: "count", Type: "integer"}), true},
		{"number type satisfied by a decimal", paramWithValue(t, "4.2", Param{Name: "price", Type: "number"}), false},
		{"number type violated by non-numeric text", paramWithValue(t, "abc", Param{Name: "price", Type: "number"}), true},
		{"empty value is never an error, even with a declared type", paramWithValue(t, "", Param{Name: "count", Type: "integer"}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateParams([]Param{tt.param})
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("ValidateParams() errs=%v, wantErr=%v", errs, tt.wantErr)
			}
		})
	}
}

// TestValidateParam_Message verifies the inline comment shown next to an
// invalid field (see renderParamSection in view.go) names the actual
// problem, not just that something's wrong.
func TestValidateParam_Message(t *testing.T) {
	p := paramWithValue(t, "abc", Param{Name: "count", Type: "integer"})
	msg, ok := ValidateParam(p)
	if ok {
		t.Fatal("expected a non-numeric value in an integer field to be invalid")
	}
	if msg == "" {
		t.Error("expected a non-empty problem description")
	}
}

// TestValidateParams_SkipsEnumFields verifies a fixed-choice enum picker
// is never flagged, even though its underlying zero-value Input would
// otherwise trip a MinLength check - an enum can't produce an
// out-of-constraint value to begin with.
func TestValidateParams_SkipsEnumFields(t *testing.T) {
	p := Param{Name: "status", MinLength: intPtr(3), Enum: NewEnumSelector([]string{"a", "b"}, "a")}
	if errs := ValidateParams([]Param{p}); len(errs) > 0 {
		t.Errorf("expected an enum param to never be flagged, got %v", errs)
	}
}

// TestRequestPane_HandleSubmit_BlocksSendOnConstraintViolation is an
// end-to-end check that a body param violating its schema's constraints
// actually prevents sending (RequestInProgress never becomes true) - the
// reason is shown inline next to the field itself (see renderParamSection),
// not listed separately anywhere.
func TestRequestPane_HandleSubmit_BlocksSendOnConstraintViolation(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"count": {Type: "integer", Minimum: floatPtr(1), Maximum: floatPtr(10)},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	if len(pane.BodyParams) != 1 {
		t.Fatalf("expected 1 body param, got %+v", pane.BodyParams)
	}
	pane.BodyParams[0].Input.SetValue("999")

	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyCtrlP})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}
	if result.RequestInProgress {
		t.Error("expected the out-of-range value to block sending")
	}
	msg, ok := ValidateParam(result.BodyParams[0])
	if ok || msg == "" {
		t.Errorf("expected the out-of-range value to still be invalid with a reason, got msg=%q ok=%v", msg, ok)
	}
}

func TestRequiredFieldErrors(t *testing.T) {
	tests := []struct {
		name    string
		param   Param
		wantErr bool
	}{
		{"required and empty", paramWithValue(t, "", Param{Name: "id", Required: true}), true},
		{"required and filled", paramWithValue(t, "5", Param{Name: "id", Required: true}), false},
		{"optional and empty", paramWithValue(t, "", Param{Name: "note", Required: false}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := RequiredFieldErrors([]Param{tt.param})
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("RequiredFieldErrors() errs=%v, wantErr=%v", errs, tt.wantErr)
			}
		})
	}
}

// TestRequiredFieldErrors_NeverFlagsEnumFields verifies a fixed-choice enum
// (including a boolean's true/false switch) with a schema default is never
// flagged as missing, even when marked Required - it always has a real
// value in that case, so "empty" can't happen for it. An enum/boolean with
// *no* declared default is a different story - see
// TestRequiredFieldErrors_FlagsUnsetEnumFieldWithNoDefault below - it does
// start "empty" (see EnumSelector.allowUnset), same as a required text
// field with nothing typed into it yet.
func TestRequiredFieldErrors_NeverFlagsEnumFields(t *testing.T) {
	p := Param{Name: "status", Required: true, Enum: NewEnumSelector([]string{"a", "b"}, "a")}
	if errs := RequiredFieldErrors([]Param{p}); len(errs) > 0 {
		t.Errorf("expected a required enum field with a schema default to never be flagged, got %v", errs)
	}
}

// TestRequiredFieldErrors_FlagsUnsetEnumFieldWithNoDefault verifies a
// required enum/boolean field whose schema declared no default - and so
// starts with no value chosen at all (see EnumSelector.allowUnset) - is
// flagged as missing, same as a required text field left blank. Before
// this, an enum/boolean field could never be "empty" (it always silently
// defaulted to some option), so a required one with no real default could
// never actually block a send even when the user hadn't made a choice.
func TestRequiredFieldErrors_FlagsUnsetEnumFieldWithNoDefault(t *testing.T) {
	p := Param{Name: "status", Required: true, Enum: NewEnumSelector([]string{"a", "b"}, "")}
	errs := RequiredFieldErrors([]Param{p})
	if len(errs) == 0 {
		t.Fatal("expected a required enum field with no schema default, left untouched, to be flagged")
	}

	p.Enum.Next() // moves off the unset state onto the first real option
	if errs := RequiredFieldErrors([]Param{p}); len(errs) > 0 {
		t.Errorf("expected the field to stop being flagged once a real value is chosen, got %v", errs)
	}
}

// TestRequestPane_HandleSubmit_BlocksSendOnMissingRequiredField is an
// end-to-end check that an empty required body field prevents sending, but
// - unlike ValidateParam's live constraint checks - doesn't get flagged
// just by existing empty; only actually attempting to send does.
func TestRequestPane_HandleSubmit_BlocksSendOnMissingRequiredField(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name": {Type: "string", Required: true},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	if len(pane.BodyParams) != 1 {
		t.Fatalf("expected 1 body param, got %+v", pane.BodyParams)
	}
	// Left empty on purpose - never typed into.

	if msg, ok := ValidateParam(pane.BodyParams[0]); !ok {
		t.Errorf("expected ValidateParam to leave an empty required field alone (that's RequiredFieldErrors' job), got %q", msg)
	}

	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyCtrlP})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}
	if result.RequestInProgress {
		t.Error("expected the missing required field to block sending")
	}
	if !result.ShowRequiredErrors {
		t.Error("expected the blocked send to switch on inline required-field errors")
	}
	if msg, ok := fieldError(result.BodyParams[0], result.ShowRequiredErrors); ok || msg == "" {
		t.Errorf("expected the empty required field to now render inline as an error, got msg=%q ok=%v", msg, ok)
	}
}

// TestRequestPane_BlockedSend_ShowsErrorInlineNotInFooter is a regression
// test for a real feature request: a blocked send used to list its reasons
// in the footer, below the Send button - now that same reason renders
// inline right next to the offending field instead (see fieldError/
// renderParamSection), and the footer no longer repeats it, freeing up the
// vertical space the footer's error block used to take.
func TestRequestPane_BlockedSend_ShowsErrorInlineNotInFooter(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.SetWidth(100)
	pane.SetHeight(40)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name": {Type: "string", Required: true},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true
	// name left empty on purpose - never typed into.

	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyCtrlP})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}

	rendered := result.View()
	if !strings.Contains(rendered, "! required") {
		t.Errorf("expected the blocked field's error to render inline, got:\n%s", rendered)
	}
	footer := result.renderFooter()
	if strings.Contains(footer, "! required") {
		t.Errorf("expected the footer to no longer repeat the field error, got:\n%s", footer)
	}
	if len(result.ParseErrors) != 0 {
		t.Errorf("expected ParseErrors to stay empty for a required-field-only block, got %v", result.ParseErrors)
	}
}

// TestRequestPane_HandleSubmit_FocusesFirstErrorField verifies a blocked
// send (see the two tests above) also jumps focus straight to whichever
// field actually caused the block - the *first* one, by field order, not
// just any of them - since ParseErrors alone (small red text under the
// button) is easy to miss, especially on a long imported form with many
// fields.
func TestRequestPane_HandleSubmit_FocusesFirstErrorField(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name":  {Type: "string", Required: true},
		"email": {Type: "string", Required: true},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	if len(pane.BodyParams) != 2 {
		t.Fatalf("expected 2 body params, got %+v", pane.BodyParams)
	}
	// Fill only whichever one sorts second alphabetically ("name"), leaving
	// "email" (sorts first) empty - so the expected jump target is
	// unambiguous regardless of which of the two is "first" by name.
	for i := range pane.BodyParams {
		if pane.BodyParams[i].Name == "name" {
			pane.BodyParams[i].Input.SetValue("server-1")
		}
	}
	// Start focus somewhere unrelated (the URL field), so landing on the
	// failing field can only be explained by the jump, not by already
	// being there.
	pane.FocusManager.FocusIndex(2) // ServerSelector(0), MethodSelector(1), URLInput(2)

	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyCtrlP})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}
	if result.RequestInProgress {
		t.Fatalf("expected the missing required field to block sending")
	}

	current, ok := result.FocusManager.Current().(*textinput.Model)
	if !ok {
		t.Fatalf("expected focus to land on a text input, got %T", result.FocusManager.Current())
	}
	var wantInput *textinput.Model
	for i := range result.BodyParams {
		if result.BodyParams[i].Name == "email" {
			wantInput = &result.BodyParams[i].Input
		}
	}
	if wantInput == nil {
		t.Fatal("test setup error: no \"email\" body param found")
	}
	if current != wantInput {
		t.Errorf("expected focus to jump to the empty required \"email\" field, landed elsewhere")
	}
}

// TestRequestPane_HandleSubmit_FocusesFirstErrorField_InsideArrayItem
// verifies the same jump-to-error behavior reaches into an array item's
// own fields (not just top-level BodyParams), matching
// RequiredArrayFieldErrors' coverage.
func TestRequestPane_HandleSubmit_FocusesFirstErrorField_InsideArrayItem(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"tags": {
			Type:         "array",
			ItemsRefName: "Tag",
			Items: map[string]http.BodyFieldSchema{
				"label": {Type: "string", Required: true},
			},
		},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	if len(pane.ArrayParams) != 1 || len(pane.ArrayParams[0].Items) != 1 {
		t.Fatalf("expected 1 array param with 1 item, got %+v", pane.ArrayParams)
	}
	// "label" left empty on purpose.

	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyCtrlP})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}
	if result.RequestInProgress {
		t.Fatalf("expected the missing required array item field to block sending")
	}

	current, ok := result.FocusManager.Current().(*textinput.Model)
	if !ok {
		t.Fatalf("expected focus to land on a text input, got %T", result.FocusManager.Current())
	}
	wantInput := &result.ArrayParams[0].Items[0].Fields[0].Input
	if current != wantInput {
		t.Errorf("expected focus to jump to the empty required array item field \"label\", landed elsewhere")
	}
}
