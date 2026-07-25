# Themes and Customization

Volt supports lightweight YAML color themes. A custom theme changes semantic
colors without changing layout, keyboard behavior, component geometry, or the
meaning of focus and status. The visual rules behind those meanings live in the
[Volt Design System](DESIGN_SYSTEM.md).

Theme loading must never prevent Volt from starting. If a recognized value is
invalid, Volt keeps the Controlled Voltage default and displays a concise
startup notification with the useful errors.

## Built-in modes

The initial built-in modes are:

| Name | Purpose |
| --- | --- |
| `default` | Controlled Voltage, Volt's branded violet-and-charge theme |
| `adaptive` | Selects values suitable for the detected terminal background |
| `mono` | No-color accessibility and terminal-compatibility fallback |

`volt` is accepted as an alias for `default`. More bundled palettes can be
added later without expanding the custom-theme schema.

## Configuration discovery

Settings persists the selected mode or custom file path in the first available
user configuration location. Volt reads selection files in this order:

1. `<user-config-dir>/volt/config.yaml`
2. `~/.volt/config.yaml`

When neither selection file exists, Volt checks two automatic custom-theme
locations:

1. `<user-config-dir>/volt/theme.yaml`
2. `~/.volt/theme.yaml`

`<user-config-dir>` comes from Go's `os.UserConfigDir`, which respects the
platform convention:

- Linux: `${XDG_CONFIG_HOME:-~/.config}`
- macOS: `~/Library/Application Support`
- Windows: `%AppData%`

`VOLT_THEME` has the highest priority and may select a built-in mode or provide
an explicit `.yaml` file path. Otherwise the first existing selection file
wins, followed by the first automatic custom-theme file. When no valid
selection exists, Volt uses `default`.

Volt does not search the current working directory. A project must not be able
to change Volt's appearance merely because Volt was launched inside it.

Example layouts:

```text
<user-config-dir>/
└── volt/
    ├── config.yaml
    └── theme.yaml

~/
└── .volt/
    ├── config.yaml
    └── theme.yaml
```

Only files ending in `.yaml` are supported. JSON theme files and `.yml`
aliases are intentionally outside the public contract.

The small selection file written by Settings is also YAML:

```yaml
version: 1
theme: adaptive
motion: system
```

`theme` may contain `default`, `adaptive`, `mono`, or a custom `.yaml` path.
Relative paths resolve from the directory containing `config.yaml`.

## Creating a theme

Start with the smallest possible file and add only the semantic roles you want
to change:

```yaml
version: 1
name: Purple Machine
extends: default

colors:
  brand: "#A66CFF"
  brand_strong: "#7038E8"
  charge: "#D8FF3E"
  signal: "#3DE4E8"

methods:
  get: "#5EE08A"
  post: "#FFC857"
  put: "#68B7FF"
  patch: "#B78CFF"
  delete: "#FF647C"

charts:
  primary: "#A66CFF"
  secondary: "#3DE4E8"
  good: "#5EE08A"
  bad: "#FF647C"
```

Copy the file to either automatic location and restart Volt. During
development, point `VOLT_THEME` at the file to try it without moving it:

```bash
VOLT_THEME=./purple-machine.yaml volt
```

Custom files extend a complete built-in mode, so omitted roles inherit safe
values. Per-component overrides are not supported initially. Keeping themes
semantic makes every shared component look cohesive and prevents custom files
from coupling themselves to Volt's internal layout.

## Theme schema

Recognized top-level fields:

| Field | Type | Meaning |
| --- | --- | --- |
| `version` | integer | Schema version; currently `1` |
| `name` | string | Human-readable theme name |
| `extends` | string | Built-in base; initially `default` or `volt` |
| `colors` | mapping | Core semantic color overrides |
| `methods` | mapping | HTTP method color overrides |
| `charts` | mapping | Load-test and response chart overrides |

Core color roles:

```text
canvas surface surface_raised border text text_muted
brand brand_strong charge signal info success warning error
```

HTTP method roles:

```text
get post put patch delete
```

Chart roles:

```text
primary secondary good bad
```

Unknown fields are ignored. This allows annotations and forward-compatible
fields without making Volt fail at startup. Misspelled recognized fields are
therefore ignored as well, so users should compare their keys with the lists
above when a color does not change.

## Color values

Accepted values:

- True color: `#RRGGBB`.
- Short true color: `#RGB`, normalized to `#RRGGBB`.
- ANSI-256 index: integer or numeric string from `0` through `255`.
- Terminal default: `default`, for canvas and neutral surface/text roles.

Invalid values include named colors such as `purple`, alpha colors, empty
strings, malformed hex values, and ANSI indices outside `0` through `255`.

## Validation and fallback

Loading stays intentionally small and predictable:

1. Read the first selected YAML file.
2. Parse recognized fields and ignore unsupported fields.
3. Resolve omitted values from the built-in base.
4. Validate every recognized color.
5. Activate the complete theme only when validation succeeds.

If loading fails:

- Volt starts with the `default` theme.
- No partial custom colors are applied.
- The source file is never modified.
- A startup Notice summarizes the file and useful validation errors.
- Multiple independent errors should be reported together when practical.

Missing automatic files are normal and do not produce a warning. An explicit
`VOLT_THEME` path that is missing does produce a warning because the user asked
Volt to load it.

## Keyboard-first Settings

The `?` assistance surface will contain `HELP` and `SETTINGS` tabs. Appearance
settings will list built-in modes and discovered custom YAML themes, show a
whole-application preview, and expose the active file and validation status.

Keyboard behavior:

```text
h/l      switch Help and Settings, or change an active choice
j/k      move between rows
enter    edit, preview, or apply
esc      leave edit mode, then close
```

Preview is transactional: entering preview snapshots the current theme,
`SAVE` atomically persists the choice to
`<user-config-dir>/volt/config.yaml`, and `CANCEL` restores the snapshot.

## Implementation requirements

- Keep parsed YAML data separate from the complete resolved `Theme`.
- Pass the resolved theme explicitly; do not add package-global mutable styles.
- Cache derived Lip Gloss styles and rebuild only when the theme changes.
- Keep loading and theme swaps atomic from the view's perspective.
- Do not let custom files alter spacing, borders, animation frames, or keys.
- Respect `NO_COLOR` by resolving to `mono`.
- Keep user theme data free of secrets and request content.

Required tests:

- Partial YAML overrides inherit correctly.
- Unknown fields are ignored.
- Invalid recognized values fall back without preventing startup.
- Both automatic paths are searched in the documented order.
- A saved Settings selection takes precedence over automatic theme files.
- `VOLT_THEME` takes precedence over a saved Settings selection.
- A missing automatic file quietly selects `default`.
- An invalid explicit path produces a startup warning.
- Built-in modes resolve every semantic role.
- ANSI-256 and mono preserve visible focus and status distinctions.
- Preview cancel restores the exact prior theme.

## Initial completion boundary

The initial customization work includes:

- `default`, `adaptive`, and `mono` built-in modes.
- YAML semantic color overrides.
- Two automatic discovery paths plus explicit `VOLT_THEME` selection.
- Safe default fallback and startup validation notices.
- Keyboard-first selection and transactional preview in Settings.

Per-component overrides, JSON files, theme import/export workflows, and a large
built-in palette catalog are not required for the initial experience.
