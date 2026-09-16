package http

// BodySchemaKey is the sentinel top-level JSON key `volt import` uses to
// mark a Request.Body as a machine-generated field-schema template (e.g.
// {"__volt_body_schema__": {"name": {"type": "string"}}}), as opposed to a
// real, previously-saved request body. Shared between internal/cli (which
// produces it from an OpenAPI requestBody schema) and internal/ui/requestpane
// (which parses it to drive the Body Params UI) so the format can't drift
// between the two independently.
const BodySchemaKey = "__volt_body_schema__"

// BodyFieldSchema describes one body field's type/default/enum/limits,
// extracted from an OpenAPI request body schema. Volt's own request model
// only ever deals in flat string fields (a deliberate simplification), so
// Default and Enum entries are already stringified regardless of the
// schema's declared type.
//
// MinLength/MaxLength/Minimum/Maximum are pointers so "the schema didn't
// say" (nil - no constraint enforced) is distinguishable from "the schema
// says 0" (a meaningful bound, e.g. minLength: 0 or minimum: 0).
type BodyFieldSchema struct {
	Type    string   `json:"type,omitempty"`
	Default string   `json:"default,omitempty"`
	Enum    []string `json:"enum,omitempty"`

	// MaxLength is enforced live as the field's CharLimit (typing simply
	// can't go over it). MinLength can't be enforced that way without
	// rejecting perfectly normal in-progress typing, so it's checked at
	// send time instead, same as Minimum/Maximum.
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Minimum   *float64 `json:"minimum,omitempty"`
	Maximum   *float64 `json:"maximum,omitempty"`

	// Items, when Type is "array", describes the array's item schema, one
	// entry per property - only populated for a "complex" array (its
	// "items" is a $ref to an object schema); nil for a "simple" array
	// (items with no object structure, e.g. items: {type: string}), which
	// has no per-property structure to compose. When set, the array is
	// editable as a repeatable set of item groups (see requestpane's
	// ArrayParam) rather than opaque text.
	Items map[string]BodyFieldSchema `json:"items,omitempty"`
	// ItemsRefName is the referenced schema's component name (e.g. "Tag",
	// from "#/components/schemas/Tag"), shown as a label for the array's
	// item block so it reads as "array of Tag" rather than just "array".
	ItemsRefName string `json:"itemsRefName,omitempty"`
	// ItemType is a "simple" array's item type (e.g. "number", "string") -
	// the counterpart to Items for a "complex" one: items with no object
	// structure to compose still declare a scalar type worth keeping, so a
	// simple array is editable as one comma-separated field of that type
	// (see requestpane.paramJSONValue) instead of silently discarding what
	// its entries are supposed to be and falling back to plain text.
	ItemType string `json:"itemType,omitempty"`

	// Required mirrors whether this field's name appeared in its parent
	// object schema's "required" list. An empty required field blocks
	// sending (checked only then, not live while still being filled in -
	// see requestpane.RequiredFieldErrors) and its label is always shown in
	// a distinct color as a hint, regardless of whether it's currently
	// empty.
	Required bool `json:"required,omitempty"`
}
