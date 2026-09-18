# Volt Design System

This document is the canonical visual and interaction specification for Volt.
Use it when designing, implementing, or reviewing any Bubble Tea screen or
Lip Gloss component. User-facing theme configuration is specified separately
in [CUSTOMIZATION.md](CUSTOMIZATION.md).

## Contents

1. [Design intent](#design-intent)
2. [Current UI findings](#current-ui-findings)
3. [Design principles](#design-principles)
4. [Color system](#color-system)
5. [Typography and symbols](#typography-and-symbols)
6. [Focus language](#focus-language)
7. [Layout and chrome](#layout-and-chrome)
8. [Component vocabulary](#component-vocabulary)
9. [Load-testing experience](#load-testing-experience)
10. [Motion](#motion)
11. [Settings experience](#settings-experience)
12. [Accessibility and terminal support](#accessibility-and-terminal-support)
13. [Implementation architecture](#implementation-architecture)
14. [Delivery sequence](#delivery-sequence)
15. [Review checklist](#review-checklist)

## Design intent

Volt should feel like a precision instrument carrying controlled current:
fast, sharp, energetic, and trustworthy. Its visual metaphor is a dark
performance machine receiving a purple nitro boost, not an uncontrolled neon
arcade.

The product expression has three layers:

- **Voltage:** violet identifies Volt, navigation, selection, and focus.
- **Charge:** acid yellow-green identifies execution, live activity, and the
  primary action.
- **Instrument:** near-black surfaces, cool neutrals, stable alignment, and
  compact data displays make Volt credible as a developer tool.

The interface must remain calm while idle. It becomes visibly energized only
when the user focuses a control, sends a request, or runs a load test.

## Current UI findings

The current UI has useful density, a responsive three-panel structure, clear
HTTP method colors, and an established violet identity. The current screenshots
also expose the next design problems:

- The six-line logo consumes working space after startup.
- Most regions use equally prominent boxes, so hierarchy depends primarily on
  border color.
- Focus is expressed inconsistently through magenta borders, purple fills,
  bold text, and component-local colors.
- The saved-request list retains the visual language of the default Bubbles
  delegate instead of Volt.
- Labels and values are readable, but field spacing and alignment vary.
- Load-test results are text-first and leave large areas unused.
- Success, HTTP method, focus, brand, and chart colors are sometimes overloaded.
- Raw color literals in multiple packages make custom themes fragile.

Preserve the information architecture and keyboard-first behavior. Replace the
styling language beneath it.

## Design principles

### Hierarchy before decoration

Use spacing, alignment, text weight, and restrained separators before adding
another border or color.

### Electricity is state

Violet and charge colors communicate focus or energy. Do not scatter them as
decoration. Idle screens should use mostly neutral tokens.

### Color is never the only signal

Pair color with a word, symbol, border weight, or position. A focused rail,
`LIVE` label, and `●` indicator must remain understandable in monochrome.

### One component, one behavior

Tabs, fields, notices, badges, and panels must render consistently across the
request pane, response pane, help, settings, and load testing.

### Fast means stable

Avoid layout movement, flicker, decorative animation, and redraws that do not
communicate a meaningful state change.

### Themes preserve semantics

Custom themes may change appearance, but they may not redefine what focus,
success, warning, failure, or live activity means.

## Color system

### Default "Controlled Voltage" palette

| Token | True color | ANSI-256 fallback | Meaning |
| --- | --- | ---: | --- |
| `canvas` | `#090B10` | `232` | Application background |
| `surface` | `#11151D` | `233` | Panels and inputs |
| `surface_raised` | `#181E29` | `234` | Modals and selected regions |
| `border` | `#30394A` | `239` | Passive structure |
| `text` | `#EDF2FF` | `255` | Primary content |
| `text_muted` | `#7F8A9D` | `245` | Hints and metadata |
| `brand` | `#9B6CFF` | `135` | Focus and brand identity |
| `brand_strong` | `#7038E8` | `93` | Selected backgrounds |
| `charge` | `#D8FF3E` | `191` | Send, running, live, primary action |
| `signal` | `#3DE4E8` | `44` | Timing and informational data |
| `info` | `#68B7FF` | `75` | Informational notices |
| `success` | `#5EE08A` | `78` | Successful outcomes |
| `warning` | `#FFC857` | `221` | Warnings and load-test caution |
| `error` | `#FF647C` | `204` | Failures and destructive actions |

The default palette is a contract, not a list of literals to copy into
components. Components consume semantic theme tokens.

### Dedicated data roles

HTTP methods and data visualization must not borrow status colors implicitly.
Expose dedicated theme roles:

| Role | Default | Use |
| --- | --- | --- |
| `method_get` | `#5EE08A` | GET badge |
| `method_post` | `#FFC857` | POST badge |
| `method_put` | `#68B7FF` | PUT badge |
| `method_patch` | `#B78CFF` | PATCH badge |
| `method_delete` | `#FF647C` | DELETE badge |
| `chart_primary` | `#9B6CFF` | Primary plot or sparkline |
| `chart_secondary` | `#3DE4E8` | Comparison plot |
| `chart_good` | `#5EE08A` | Healthy region |
| `chart_bad` | `#FF647C` | Failure region |

A successful DELETE response uses `success` for its outcome and
`method_delete` for its method badge. Those are independent facts.

### Color application

- Use `brand` for focus rails, active navigation, and the Volt mark.
- Use `brand_strong` sparingly behind selected tabs or badges.
- Use `charge` for the primary action and active execution only.
- Use `signal` for latency, timing, and informational live data.
- Use status tokens only for the matching semantic outcome.
- Use `surface_raised` for modals and selected blocks, not every panel.
- Limit a normal idle screen to neutrals plus one brand accent.

## Typography and symbols

Volt cannot control the user's terminal font. Create hierarchy with case,
weight, spacing, and color:

- Product and compact panel titles: bold uppercase.
- Field labels: title case or uppercase, consistently aligned within a group.
- Values and body content: normal weight.
- Metadata, units, descriptions, and hints: `text_muted`.
- Live metrics: bold numeric value, muted unit, short uppercase label.
- Avoid bolding entire containers; bold only the information that leads the eye.

Preferred symbols:

| Symbol | Meaning |
| --- | --- |
| `⚡` or `ϟ` | Volt identity or startup |
| `● LIVE` | Active operation |
| `✓` | Completed successfully |
| `!` | Warning |
| `×` | Failure or destructive action |
| `›` | Navigation or disclosure |
| `┃` | Focused or energized rail |
| `│` | Passive rail |

Unicode must have an ASCII fallback (`*`, `!`, `x`, `>`, `|`) where terminal
capability or width handling requires it.

## Focus language

Focus must change at least two perceivable properties.

### Container focus

```text
Passive    │ Request
Focused    ┃ REQUEST
Running    ┃ REQUEST  ● LIVE
```

- Passive: neutral thin rail and normal title.
- Focused: bright violet heavy rail and bright, bold title.
- Running: focused rail plus charge-colored `● LIVE`.
- Error: focused rail remains violet; the error is shown separately in red.

Do not replace brand focus with an error-colored border. Focus and validation
are separate dimensions.

### Input focus

- Brighten the label from `text_muted` to `text`.
- Show a violet cursor or leading marker.
- Retain visible input boundaries in no-color mode.
- Show validation text adjacent to the field; do not communicate it only by a
  red border.

### Tabs

- Use the same tab component in app navigation, responses, help, and settings.
- Prefer a violet underline or compact filled segment.
- Keep inactive tabs neutral and legible.
- Include direct-selection keys where they exist, for example `[1] Body`.
- In no-color mode, wrap the active tab with `[` and `]` or add `●`.

### Actions

- Primary idle action: charge foreground or compact charge fill.
- Focused primary action: charge fill, dark text, bold label.
- Busy action: charge activity frame plus explicit state such as `SENDING`.
- Destructive action: error color and explicit verb.
- Disabled action: muted, with no focus affordance.

## Layout and chrome

### Startup and compact header

The large ASCII logo is a startup signature, not permanent working chrome.

Startup behavior:

1. Render the full ASCII `VOLT` mark immediately.
2. Do not block input or initialization while it is displayed.
3. Dismiss it on the first meaningful keypress or after a short maximum.
4. Use at most three compression frames and complete the transition within
   750 ms.
5. Skip intermediate frames when reduced motion is enabled.

The steady-state header occupies two rows:

```text
⚡ VOLT  /  REQUEST                         NORMAL     v0.2.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

- Left: product and active workspace/panel.
- Right: operating mode and version.
- `NORMAL`, `LOAD TEST`, and future modes use a badge, not a second title.
- At narrow widths, keep `⚡ VOLT`, active panel, and mode; omit version first.
- Preserve the full ASCII logo for startup, About, and promotional captures.

### Panels and rails

- Use neutral separators for passive regions.
- Use a heavy violet left rail for the focused panel.
- Use charge only when that panel is actively executing.
- Reserve complete raised borders for modals.
- Avoid nested rounded boxes.
- Use whitespace and aligned section headings to group content.

### Responsive priorities

Wide layout:

- Saved requests at left.
- Request editor at upper right.
- Response or load-test workspace at lower right.
- Compact command header and one-row status/footer.

Focused layout:

- One active panel consumes the content region.
- The global tab bar identifies the other panels.
- Component behavior and semantics do not change.

Narrow layout:

- Remove descriptions and optional units before truncating primary values.
- Collapse metric grids to one or two columns.
- Keep the active action and error reason visible.

## Component vocabulary

Every reusable component must accept theme input and expose its meaningful
states. Component code must not instantiate raw color literals.

### Panel

Anatomy: rail, title, optional state badge, optional count, content.

States: passive, focused, busy, disabled. Error content is rendered by a Notice
inside the panel rather than changing the focus model.

### Tabs

Anatomy: optional direct key, label, selection indicator.

States: inactive, active, disabled. Tabs support `h/l` movement, direct keys
where available, and no-color selection markers.

### Field

Anatomy: label, value/control, optional hint, optional validation.

States: idle, focused, invalid, disabled. Labels within one form share a
measured width instead of using hand-written padding.

### Action

Anatomy: optional symbol, imperative label, optional shortcut.

States: idle, focused, busy, disabled, destructive. Use specific labels:
`SEND`, `RUN LOAD TEST`, `STOP`, `SAVE THEME`, not generic `OK`.

### Badge

Use for HTTP method, response status, mode, and live state. Badges are compact
and must not become a second button language.

### Notice

Anatomy: level label or symbol, message, optional recovery hint.

States: info, success, warning, error. Keep the existing one-row footer notice
for transient feedback; use an inline Notice when the message belongs to a
specific panel.

### Metric

Anatomy: label, value, unit, optional trend. Align values, not labels, when
metrics appear in a row. Never color every label violet.

### EmptyState

Anatomy: small symbol, short title, one-sentence explanation, next action.

Response example:

```text
                  ⚡
           NO RESPONSE YET

     Enter a URL and press alt+enter
             to send power.
```

### KeyHint

Render the key and action as distinct parts. Use one syntax throughout Volt,
including the footer and help/settings surface.

### Progress

Use a bounded bar when total work is known, an activity frame when it is not,
and a spinner only when neither progress nor activity can be expressed better.
Always pair the graphic with a numeric or textual state.

### RequestListItem

Replace the default Bubbles delegate with a Volt delegate:

- Line one: method badge and request name.
- Line two: shortened host/path in muted text.
- Selected: violet rail, bright name, and raised surface when supported.
- Preserve filtering, counts, and wrapped `j/k` navigation.
- Avoid coloring both lines with the same accent.

## Load-testing experience

Load testing is Volt's visual centerpiece. It should look like a live electrical
instrument, then settle into a clear report.

### Live overview

```text
● LIVE    18,420 / 25,000 requests                         74%
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╺━━━━━━━━━━━━━━━

  8,941 req/s       11.2 ms p50       0.04% errors
  ███████████████   ███████░░░░       ▏

LATENCY CURRENT
▁▂▂▃▃▄▅▄▃▅▆▅▇▆▅▃▂
```

Required live information:

- Completed and total requests or elapsed and target duration.
- Progress percentage when bounded.
- Current throughput.
- p50 latency and, when space allows, p95.
- Error count and rate.
- Rolling latency sparkline.
- Explicit stop shortcut.

Do not derive the sparkline from a single percentile snapshot. Maintain a
bounded rolling series of interval samples in the presentation model.

### Result states

Success:

- `✓ COMPLETE` success badge.
- Final throughput, p50, p95, p99, duration, and request count.
- Success/failure distribution.

Partial failure:

- Warning summary when useful results exist but some requests failed.
- Error rate and top failure reasons remain visible without switching tabs.

Failure:

- `× FAILED` error badge.
- Plain-language cause and recovery hint.
- Preserve zero values without presenting them as healthy metrics.

Cancelled:

- `STOPPED` neutral/warning badge.
- Show the partial report and elapsed duration.

### Load-test tabs

- Overview: live/final summary.
- Latency: percentile table, rolling sparkline, min/max, and distribution when
  enough samples exist.
- Errors: grouped causes, counts, rates, and representative messages.

The active operation indicator may animate. Borders, titles, and completed
reports do not.

## Motion

Motion communicates state; it is never ambient decoration.

| Motion | Cadence | Lifetime |
| --- | --- | --- |
| Startup compression | 80–120 ms/frame | At most 750 ms total |
| Request activity | 80–125 ms/frame | While request is pending |
| Load-test live refresh | 100–250 ms | While test is active |
| Notice transition | One state change | No looping |

Suggested activity frames:

```text
ϟ····  ·ϟ···  ··ϟ··  ···ϟ·  ····ϟ
```

Rules:

- Stop every ticker when its operation ends.
- Do not redraw idle screens.
- Avoid visual counters updating faster than humans can read.
- Use stable widths so changing values do not move neighboring components.
- Reduced motion replaces sequences with a static `●` or text state.

## Settings experience

`?` continues to open the assistance surface. Expand it into a compact command
center with `HELP` and `SETTINGS` tabs rather than hiding settings among shortcut
rows.

Navigation:

- `h/l`: switch Help and Settings tabs, or switch settings categories.
- `j/k`: move through rows.
- `enter`: open a control, toggle a value, or apply a preview.
- `h/l`: change the selected option while a choice control is active.
- `esc`: leave a control, then close the surface.
- Direct shortcuts shown by KeyHint take precedence where documented.

Settings categories:

- Appearance: active theme and live preview.
- Motion: system, full, or reduced.
- Accessibility: no-color/mono mode and Unicode fallback.
- Theme files: import, export, reload, and validation status.

Theme preview is transactional:

1. Snapshot the active theme.
2. Apply selection in memory to the whole visible UI.
3. `SAVE` persists atomically.
4. `CANCEL` restores the snapshot.
5. Parse or validation failures preserve the last valid theme and show a Notice.

## Accessibility and terminal support

- Support true color, ANSI-256, and no-color rendering.
- Treat terminal-default background as a valid canvas value.
- Never rely on red/green alone.
- Keep text readable when bold or faint support is absent.
- Test symbol widths and provide ASCII fallbacks.
- Keep focus visible in monochrome using rail weight and title treatment.
- Respect `NO_COLOR` and explicit mono configuration.
- Prefer terminal capability detection to assumptions about dark backgrounds.
- Do not use blinking text.

## Implementation architecture

Create a semantic design layer:

```text
internal/ui/design/
├── palette.go       # Raw colors and ANSI fallbacks
├── theme.go         # Semantic Theme tokens and built-ins
├── typography.go    # Labels, values, headings, muted text
├── borders.go       # Panel, rail, and modal definitions
├── components.go    # Shared style constructors
└── motion.go        # Durations, frames, reduced-motion rules
```

The conceptual theme model is:

```go
type Theme struct {
    Name   string
    Colors Colors
    Motion MotionMode
}

type Colors struct {
    Canvas, Surface, SurfaceRaised lipgloss.Color
    Text, TextMuted, Border        lipgloss.Color
    Brand, BrandStrong, Charge     lipgloss.Color
    Info, Signal                   lipgloss.Color
    Success, Warning, Error        lipgloss.Color
    MethodGET, MethodPOST          lipgloss.Color
    MethodPUT, MethodPATCH         lipgloss.Color
    MethodDELETE                   lipgloss.Color
    ChartPrimary, ChartSecondary   lipgloss.Color
}
```

The final Go types may be nested for readability. Preserve these semantic
boundaries.

Use dependency injection:

- Construct the active Theme at application startup.
- Pass it to app-level and component constructors.
- Derive Lip Gloss styles from the Theme.
- Avoid mutable package-global styles.
- Keep the parsed configuration separate from the resolved, validated Theme.
- Resolve inheritance and fallbacks before components receive the Theme.

User configuration, discovery order, schema, and built-in themes are specified
in [CUSTOMIZATION.md](CUSTOMIZATION.md).

### Verification

For each migrated component:

- Add state-oriented tests before changing rendering.
- Test active, inactive, focused, busy, error, and no-color states as relevant.
- Verify wide, focused, minimum supported, and narrow layouts.
- Assert dimensions after styling; borders and padding must not overflow.
- Prefer semantic assertions plus stable render snapshots over snapshots tied
  to incidental ANSI escape ordering.
- Run the full Go test suite after each coherent migration.

## Delivery sequence

1. Add theme types, default resolution, and configuration validation.
2. Migrate raw colors without changing layout.
3. Add startup-to-compact header behavior.
4. Introduce shared Panel, Tabs, Field, Action, Badge, and Notice styles.
5. Replace the saved-request list delegate.
6. Add EmptyState and KeyHint components.
7. Add settings and transactional theme preview.
8. Redesign live and completed load-test views.
9. Add adaptive, mono, and named built-in themes.
10. Add import/export and document the stable configuration format.

Keep each step buildable and independently reviewable.

## Review checklist

Before accepting a Volt UI change, confirm:

- It uses semantic theme roles and introduces no unexplained raw color.
- It follows the focus language and remains clear without color.
- It reuses or deliberately extends the component vocabulary.
- Idle presentation is calm; energy appears only with focus or activity.
- Loading, success, partial failure, failure, cancellation, and empty states are
  considered where applicable.
- Keyboard navigation, visible key hints, and focus return behavior are defined.
- Wide, focused, narrow, true-color, ANSI-256, and mono behavior are considered.
- Animation has a purpose, a bounded cadence, and a stop condition.
- The large ASCII logo appears only during startup, About, or promotional use.
- Load-test changes improve live comprehension rather than merely adding color.
- Relevant tests fail before the behavioral change and pass afterward.
