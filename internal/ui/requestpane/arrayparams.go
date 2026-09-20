package requestpane

import (
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

// maxArrayItems caps how many items an ArrayParam can grow to via its
// count control (see ArrayCountControl) - a sane ceiling against runaway
// "add" presses rather than a schema-declared limit.
const maxArrayItems = 20

// ArrayItem is one entry in an ArrayParam - a set of named sub-fields
// mirroring the array's item schema's properties, sorted by name the same
// way every item uses.
type ArrayItem struct {
	Fields []Param
}

// ArrayParam is a body param whose schema declared "type: array" - a
// repeatable list of ArrayItem, composed into a JSON array when the
// request is built (see BuildRequestBody). Covers two schema shapes:
//
//   - "complex" (its "items" was a $ref to an object - see
//     cli.extractFieldSchema): itemSchema is set, each ArrayItem holds one
//     Param per named property, and the array serializes to a JSON array
//     of objects.
//   - "simple" (its items had no object structure to compose - e.g.
//     items: {type: number}, see http.BodyFieldSchema.ItemType): ItemType
//     is set instead of itemSchema, each ArrayItem holds exactly one
//     unnamed Param (its own independent single-line input - no
//     delimiter, so no escaping question for commas/quotes/etc. within a
//     value), and the array serializes to a flat JSON array of scalars.
type ArrayParam struct {
	Name string
	// ItemsRefName is the referenced schema's component name (e.g. "Tag"),
	// shown as a label for the item block so it reads as "array of Tag"
	// rather than just "array". Only ever set for a "complex" array.
	ItemsRefName string
	// itemSchema is the template every new item is built from (see
	// newArrayItem) - the item schema's own properties, by name. Nil for a
	// "simple" array (see ItemType below).
	itemSchema map[string]http.BodyFieldSchema
	// ItemType marks this as a "simple" array and gives each item's lone
	// field its type (e.g. "number") - mutually exclusive with itemSchema.
	ItemType string
	Items    []ArrayItem
	// CountControl is this array's "how many items" field - wired up by
	// the caller (see syncBodyParams) to point back at this exact
	// ArrayParam once it's in its final resting slice, not at a
	// since-copied local variable.
	CountControl *ArrayCountControl
}

// IsSimple reports whether this is a "simple" array (see ArrayParam) -
// each item is one unnamed scalar field rather than a named group of
// sub-fields.
func (a *ArrayParam) IsSimple() bool {
	return a.ItemType != ""
}

// simpleArrayItemName is the single field name new{Array,}Item gives a
// "simple" array's lone item field - it has no real property name (it's
// not a named object property, just "the value"), and both
// renderArrayParamSection (view.go) and ValidateArrayParams/
// RequiredArrayFieldErrors below special-case this exact name to skip
// showing/reporting a name that isn't meaningful.
const simpleArrayItemName = ""

// newArrayItem builds one item's fields from schema, in a stable
// (alphabetical) order, taking each field's schema default the same way a
// top-level body param does (see paramFromFieldSchema). For a "simple"
// array (schema is nil, itemType is set), the item is instead a single
// unnamed field of that type.
func newArrayItem(schema map[string]http.BodyFieldSchema, itemType string) ArrayItem {
	if itemType != "" {
		return ArrayItem{Fields: []Param{paramFromFieldSchema(simpleArrayItemName, http.BodyFieldSchema{Type: itemType})}}
	}

	names := make([]string, 0, len(schema))
	for name := range schema {
		names = append(names, name)
	}
	sort.Strings(names)

	fields := make([]Param, len(names))
	for i, name := range names {
		fields[i] = paramFromFieldSchema(name, schema[name])
	}
	return ArrayItem{Fields: fields}
}

// newArrayParam builds a "complex" ArrayParam over itemSchema, starting
// with a single pre-populated item - most APIs expect at least one entry,
// and starting empty would hide the item's fields entirely until the user
// finds the count control. CountControl is left nil; the caller wires it
// up once this ArrayParam is in its final resting slice (see
// syncBodyParams).
func newArrayParam(name, itemsRefName string, itemSchema map[string]http.BodyFieldSchema) ArrayParam {
	return ArrayParam{
		Name:         name,
		ItemsRefName: itemsRefName,
		itemSchema:   itemSchema,
		Items:        []ArrayItem{newArrayItem(itemSchema, "")},
	}
}

// newSimpleArrayParam builds a "simple" ArrayParam (see ArrayParam) over
// itemType, starting with one pre-populated (empty) item, same rationale
// as newArrayParam.
func newSimpleArrayParam(name, itemType string) ArrayParam {
	return ArrayParam{
		Name:     name,
		ItemType: itemType,
		Items:    []ArrayItem{newArrayItem(nil, itemType)},
	}
}

// AddItem appends one more pre-populated item, up to maxArrayItems.
func (a *ArrayParam) AddItem() {
	if len(a.Items) >= maxArrayItems {
		return
	}
	a.Items = append(a.Items, newArrayItem(a.itemSchema, a.ItemType))
}

// RemoveLastItem removes the most recently added item, if any remain.
// Items are only ever removed from the end - there's no per-item "delete
// this one" control - so the array's item count is the only state the
// count control (and the focus cycle built from it) needs to track.
func (a *ArrayParam) RemoveLastItem() {
	if len(a.Items) == 0 {
		return
	}
	a.Items = a.Items[:len(a.Items)-1]
}

// itemLabel names what one item is, for display - the referenced schema's
// name if known (e.g. "Tag"), the item type for a "simple" array (e.g.
// "number"), else the field's own name.
func (a *ArrayParam) itemLabel() string {
	if a.ItemsRefName != "" {
		return a.ItemsRefName
	}
	if a.ItemType != "" {
		return a.ItemType
	}
	return a.Name
}

// ArrayCountControl is an ArrayParam's "how many items" field - cycled with
// the same left/right keys as MethodSelector/EnumSelector (right appends
// one more item, left drops the last one) rather than a literal button:
// Enter is already claimed globally for submitting the request (see
// NormalMode/LoadTestMode.HandleInput), so it isn't available here to
// activate one.
type ArrayCountControl struct {
	array   *ArrayParam
	focused bool
}

// Next appends one more item.
func (c *ArrayCountControl) Next() { c.array.AddItem() }

// Prev removes the most recently added item.
func (c *ArrayCountControl) Prev() { c.array.RemoveLastItem() }

// Focus/Blur satisfy ui.Focusable so an ArrayCountControl can sit in the
// same tab cycle as the fields around it.
func (c *ArrayCountControl) Focus() tea.Cmd {
	c.focused = true
	return nil
}

func (c *ArrayCountControl) Blur() {
	c.focused = false
}

func (c *ArrayCountControl) View() string {
	style := unfocusedButtonStyle
	if c.focused {
		style = focusedButtonStyle
	}
	return style.Render(fmt.Sprintf("‹ %d item(s) › (h/l or ←/→ add/remove)", len(c.array.Items)))
}

// arrayParamFocusables exposes each ArrayParam's count control and every
// current item's fields for the focus cycle. Pointers are taken directly
// into the given slice (via each item's Fields, through paramFocusables),
// so - like paramFocusables itself - this must only be called against a
// slice that isn't about to be reassigned; syncBodyParams guarantees that
// by rebuilding the focus manager immediately after, and so does adding or
// removing an item (see mode_normal.go/mode_loadtest.go's
// *ArrayCountControl case).
func arrayParamFocusables(arrays []ArrayParam) []ui.Focusable {
	var focusables []ui.Focusable
	for i := range arrays {
		a := &arrays[i]
		focusables = append(focusables, a.CountControl)
		for itemIdx := range a.Items {
			focusables = append(focusables, paramFocusables(a.Items[itemIdx].Fields)...)
		}
	}
	return focusables
}

// arrayItemErrorPrefix names one item field for an error message - "name
// [index].field" for a "complex" array's named sub-field, or plain
// "name[index]" for a "simple" array's single unnamed field (see
// simpleArrayItemName) - dropping the dangling "." a blank field name
// would otherwise leave behind.
func arrayItemErrorPrefix(arrayName string, index int, fieldName string) string {
	if fieldName == simpleArrayItemName {
		return fmt.Sprintf("%s[%d]", arrayName, index)
	}
	return fmt.Sprintf("%s[%d].%s", arrayName, index, fieldName)
}

// ValidateArrayParams checks every item field of every array param with
// ValidateParam, returning one "name[index].field: reason" message per
// violation (see arrayItemErrorPrefix), or nil if everything's within
// bounds - the array counterpart of ValidateParams, called alongside it
// before a request is actually sent (see mode_normal.go and
// mode_loadtest.go's handleSubmit).
func ValidateArrayParams(arrays []ArrayParam) []string {
	var errs []string
	for _, a := range arrays {
		for i, item := range a.Items {
			for _, f := range item.Fields {
				if msg, ok := ValidateParam(f); !ok {
					errs = append(errs, fmt.Sprintf("%s: %s", arrayItemErrorPrefix(a.Name, i, f.Name), msg))
				}
			}
		}
	}
	return errs
}

// RequiredArrayFieldErrors is ValidateArrayParams' RequiredFieldErrors
// counterpart: every item field of every array param that's required but
// still empty, one "name[index].field: required" message each.
func RequiredArrayFieldErrors(arrays []ArrayParam) []string {
	var errs []string
	for _, a := range arrays {
		for i, item := range a.Items {
			for _, f := range item.Fields {
				if f.Required && f.Value() == "" {
					errs = append(errs, fmt.Sprintf("%s: required", arrayItemErrorPrefix(a.Name, i, f.Name)))
				}
			}
		}
	}
	return errs
}
