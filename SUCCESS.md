# Volt UI Success Criteria

This document tracks the product decisions and completion boundary for Volt's
Controlled Voltage interface work. The canonical visual rules remain in
[DESIGN_SYSTEM.md](DESIGN_SYSTEM.md), and the public theme format remains in
[CUSTOMIZATION.md](CUSTOMIZATION.md).

## Current foundation

The colorway customization branch is successful when it provides:

- One semantic `Theme` shared by the application and reusable UI components.
- The Controlled Voltage default palette with distinct brand, charge, signal,
  status, HTTP method, and chart roles.
- Centralized panel, tab, action, badge, notice, and text styles.
- Explicit theme injection instead of package-global component colors.
- A consistent focus language built from rails, title treatment, and state
  labels.
- A compact sidebar command trail showing the latest ten typed command
  characters.
- A documented, versioned YAML theme format with safe fallback behavior.

## Custom theme scope

Volt supports YAML theme files only. A theme may override semantic colors for
the application, HTTP methods, and charts. Per-component overrides are not part
of the initial public contract because the centralized components already
provide a cohesive result.

Theme loading should stay intentionally lightweight:

- Check two documented user-facing file locations.
- Ignore unsupported fields so newer or locally annotated files remain usable.
- Reject invalid recognized values without partially applying the theme.
- Fall back to the Controlled Voltage default.
- Show a concise startup notification containing the useful validation errors.
- Never prevent Volt from starting because a custom theme is missing or
  invalid.

## Next UI milestones

The continuation branch should focus on the experiences users will notice most:

1. Add keyboard-first Settings under the `?` surface, using `h/l`, `j/k`,
   `enter`, and `esc`.
2. Show the full ASCII Volt mark at startup, then compress it into the
   two-row `VOLT / ACTIVE TAB` command-center header.
3. Redesign load testing as Volt's visual centerpiece with live progress,
   stable counters, compact metrics, sparklines, and clear completed, partial,
   failed, and cancelled states.
4. Finish remaining visual-state migration, including busy, empty, disabled,
   error, narrow, ANSI-256, and monochrome behavior.

These milestones should continue as small, coherent, preferably single-file
commits. The current colorway branch should remain the reviewable semantic
design foundation rather than growing to contain the entire settings and
load-testing redesign.
