---
name: volt-ui-design
description: Design, implement, or review Volt's Bubble Tea and Lip Gloss terminal UI using the repository's Controlled Voltage design system. Use for Volt color themes, theme configuration, settings, layout, focus behavior, reusable UI components, motion, startup/header treatments, load-test visualization, accessibility, screenshots, or any change under internal/ui or internal/app that affects presentation or interaction.
---

# Volt UI Design

Apply Volt's canonical design language without inventing component-local colors
or interaction patterns.

## Load the canonical references

Before UI work, read `../../../DESIGN_SYSTEM.md` completely.

For themes, settings, configuration discovery, imports, exports, or custom
colors, also read `../../../CUSTOMIZATION.md` completely.

Treat those files as authoritative. Update the canonical document in the same
change when intentionally extending the system.

## Workflow

1. Inspect the affected models, views, update paths, layout tests, and current
   screenshots or captures.
2. List the relevant states before styling: idle, focused, busy, empty, success,
   partial failure, failure, disabled, cancelled, and narrow as applicable.
3. Reuse the documented component vocabulary and semantic theme roles. Do not
   add raw color literals outside the design/theme layer.
4. Preserve keyboard-first behavior. Define focus entry, movement, activation,
   cancellation, and focus return for new interactions.
5. Follow the repository's test-driven workflow. Add a failing state or layout
   test, implement the smallest behavior, then refactor.
6. Verify wide, focused, minimum-size, ANSI-256, and mono behavior in
   proportion to the change.
7. Check animation cadence, stop conditions, stable widths, and reduced-motion
   behavior for any timed UI.
8. Run focused tests and then the full Go test suite before handoff.

## Design constraints

- Keep the idle interface neutral and controlled. Use violet for brand/focus and
  charge for primary action/live execution.
- Make focus visible through at least two properties and understandable without
  color.
- Keep success, warning, error, HTTP methods, charts, brand, and focus as
  separate semantic roles.
- Reserve the large ASCII logo for startup, About, and promotional output.
- Prefer rails, separators, spacing, and alignment to nested rounded boxes.
- Treat load testing as the visual centerpiece, with live progress, stable
  metrics, and a real bounded time series for sparklines.
- Do not animate idle screens or decorative borders.
- Do not let user themes alter layout, spacing, key behavior, or semantic
  meaning.

## Theme implementation

Keep configuration data separate from the complete, validated Theme consumed by
components. Resolve inheritance and fallbacks before constructing styles. Pass
Theme explicitly; avoid mutable package-global styles.

When changing the public theme schema, preserve versioning, safe fallback,
atomic preview/apply behavior, and the YAML-only format described in the
customization contract.

## Review output

For design reviews, lead with concrete findings ordered by user impact and cite
the relevant file or screenshot. Distinguish implementation defects from
intentional future design work.
