# Themes and Customization

> This document defines the planned public configuration contract. Theme
> loading and the settings UI are not implemented yet.

Volt themes customize the interface without changing the meaning of focus,
activity, success, warning, or failure. The visual rules behind those meanings
live in the [Volt Design System](DESIGN_SYSTEM.md).

## Contents

1. [Built-in themes](#built-in-themes)
2. [Configuration discovery](#configuration-discovery)
3. [Configuration files](#configuration-files)
4. [Theme schema](#theme-schema)
5. [Color values](#color-values)
6. [Fallback and validation](#fallback-and-validation)
7. [Settings and preview](#settings-and-preview)
8. [Import and export](#import-and-export)
9. [Implementation requirements](#implementation-requirements)

## Built-in themes

The initial public preset collection is:

| Name | Purpose |
| --- | --- |
| `default` | Controlled Voltage, Volt's branded violet-and-charge theme |
| `dark` | Lower-saturation dark theme |
| `light` | High-contrast theme for light terminal backgrounds |
| `dracula` | Dracula-inspired mapping of Volt semantic roles |
| `nord` | Nord-inspired mapping of Volt semantic roles |
| `monokai` | Monokai-inspired mapping of Volt semantic roles |
| `adaptive` | Chooses suitable light or dark values from terminal capability |
| `mono` | No-color accessibility and compatibility theme |

`volt` may be accepted as an alias for `default`. Built-in theme names are
case-insensitive in the settings UI and normalized to lowercase in files.

Third-party theme names and descriptions may reference their upstream palette,
but Volt's bundled mappings and documentation must respect the relevant
licenses and trademarks.

## Configuration discovery

Use `os.UserConfigDir` as the platform-aware base and append `volt`.

Expected locations include:

- Linux: `${XDG_CONFIG_HOME:-~/.config}/volt/`
- macOS: `~/Library/Application Support/volt/`
- Windows: `%AppData%\volt\`

At startup, resolve appearance in this order:

1. Explicit future command-line theme or theme-file option.
2. `VOLT_THEME`, containing a built-in name or an explicit file path.
3. `theme` selected in `config.yaml` or `config.json`.
4. The `default` built-in theme.

Custom theme files live in the `themes` directory beneath the Volt
configuration directory. YAML is the documented format; JSON is accepted as an
equivalent machine-oriented format.

```text
volt/
├── config.yaml
└── themes/
    ├── my-purple-machine.yaml
    └── team-standard.json
```

Do not search the current working directory implicitly. A project must not be
able to change Volt's appearance merely because Volt was launched inside it.

## Configuration files

Application configuration selects a theme and motion preference:

```yaml
version: 1
theme: my-purple-machine
motion: system
```

Valid motion values:

- `system`: respect environment and accessibility settings when detectable.
- `full`: use all purposeful Volt transitions.
- `reduced`: use static state indicators and skip transition frames.

A custom theme extends a complete built-in theme and overrides semantic roles:

```yaml
version: 1
name: My Purple Machine
extends: default

colors:
  canvas: "#090B10"
  surface: "#11151D"
  surface_raised: "#181E29"
  border: "#30394A"
  text: "#EDF2FF"
  text_muted: "#7F8A9D"
  brand: "#A66CFF"
  brand_strong: "#7038E8"
  charge: "#D8FF3E"
  signal: "#3DE4E8"
  info: "#68B7FF"
  success: "#5EE08A"
  warning: "#FFC857"
  error: "#FF647C"

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

Most users should stop at semantic color overrides. Optional component
overrides satisfy specialized use cases without making every style property
part of the public API:

```yaml
components:
  panel:
    focused_rail: "#C08BFF"
    running_rail: "#D8FF3E"
  tabs:
    active_foreground: "#FFFFFF"
    active_background: "#7038E8"
  action:
    primary_foreground: "#090B10"
    primary_background: "#D8FF3E"
```

Component overrides are colors only in schema version 1. Border glyphs,
padding, spacing, animation frames, and layout remain controlled by Volt so a
theme cannot break geometry or keyboard affordances.

## Theme schema

All files are versioned. Unknown future versions fail safely with a useful
message.

### Required top-level fields

| Field | Type | Meaning |
| --- | --- | --- |
| `version` | integer | Schema version; initially `1` |
| `name` | string | Human-readable theme name |
| `extends` | string | Complete built-in or custom base theme |

At least one recognized override must be present. All omitted values inherit
from `extends`.

### Semantic color keys

Core roles:

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

Component override groups:

```text
panel tabs field action badge notice metric empty_state key_hint progress
request_list_item
```

The stable list of keys must be exported by the implementation and covered by
schema tests. Unknown keys produce a warning by default and become an error in
strict validation or import flows. This catches misspellings without making a
minor Volt upgrade unnecessarily destructive.

## Color values

Accepted values:

- True color: `#RRGGBB`.
- Short true color: `#RGB`, normalized to `#RRGGBB`.
- ANSI-256 index: integer or numeric string from `0` through `255`.
- Terminal default: `default`, allowed for canvas and selected neutral roles.

Reject:

- Invalid or ambiguous color names.
- Alpha values; terminals do not provide portable alpha compositing.
- Out-of-range ANSI indices.
- Empty strings.

The resolved theme stores both the requested value and a terminal-compatible
rendered color where necessary. ANSI fallback must preserve semantic
distinction, not merely choose the numerically nearest colors.

## Fallback and validation

Theme loading must never prevent Volt from starting.

1. Parse into a configuration structure.
2. Validate schema version, inheritance, keys, and color values.
3. Detect inheritance cycles.
4. Resolve every semantic role into a complete immutable Theme.
5. Validate required contrast and distinct focus/status signals.
6. Activate only after the full theme succeeds.

On failure:

- Keep or load the last valid theme.
- Fall back to `default` when no valid theme has been active.
- Display one concise warning Notice with the file and recovery action.
- Never partially apply a file.
- Never overwrite an invalid user file automatically.

Import validation should report all independent problems in one pass when
practical.

## Settings and preview

The `?` assistance surface contains `HELP` and `SETTINGS` tabs. Appearance
settings provide:

- Theme list with built-in/custom badges.
- A preview showing text, focus, action, status, HTTP methods, and a small chart.
- Motion selection.
- Reload, import, export, save, and cancel actions.
- File path and validation status for custom themes.

Keyboard behavior:

```text
h/l      switch top-level tab or settings category
j/k      move between rows
enter    edit, toggle, preview, or apply
h/l      change a choice while editing
esc      leave edit mode, then close
```

Preview is transactional. It applies to the whole visible app in memory so the
user can judge real panels and response content, not an isolated swatch grid.
`SAVE` writes configuration atomically. `CANCEL` restores the original Theme.

## Import and export

The ticket should support:

- Import from an explicit YAML or JSON path.
- Validate and preview before copying into the theme directory.
- Resolve name collisions explicitly; never silently overwrite.
- Export the current resolved theme as a standalone versioned file.
- Optionally export a minimal file containing only differences from its base.

Import/export actions must not execute content from the theme file. Theme files
are data only.

## Implementation requirements

- Keep parsed configuration types separate from the resolved Theme used by UI
  components.
- Use a semantic design package; do not expose package-global mutable styles.
- Pass Theme to constructors or a stable app-level design context.
- Cache derived Lip Gloss styles and rebuild them only when the Theme changes.
- Make theme swaps atomic from the view's perspective.
- Respect `NO_COLOR` by resolving to `mono` unless the user explicitly selects
  an accessible no-color behavior supported by the implementation.
- Avoid storing secrets or request data in theme files.
- Write files with restrictive normal user permissions and atomic replacement.

### Required tests

- Built-in themes resolve every semantic role.
- YAML and JSON forms produce equivalent resolved themes.
- Partial overrides inherit correctly.
- Invalid colors, versions, keys, and inheritance cycles fail safely.
- Missing files fall back to `default`.
- ANSI-256 and mono resolution preserve focus and status distinctions.
- Preview cancel restores the exact prior Theme.
- Save persists and reloads the selected theme.
- Component overrides cannot change layout geometry.
- Theme switching does not start duplicate animation tickers.

### Ticket completion criteria

- Built-ins: Default, Dark, Light, Dracula, Nord, and Monokai.
- Adaptive and mono system modes.
- Theme switcher within Settings.
- YAML and JSON custom theme support.
- Whole-application preview before applying.
- Semantic and documented per-component color overrides.
- Import and export with validation.
- Startup discovery and safe fallback.
- README link to this contract.
