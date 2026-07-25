# Volt v0.2.0

Volt is back and better than ever! Here are some of the changes we have made to
make it the best it can possibly be.

This release gives Volt a cohesive visual identity, makes customization a
first-class keyboard workflow, and transforms load testing into a live
performance instrument.

## A new Controlled Voltage design

Volt now has one semantic design system shared across the entire terminal
interface. The default experience combines a calm near-black instrument panel,
Volt violet for identity and focus, and charge yellow for actions and live
execution.

- Consistent panels, tabs, actions, badges, notices, metrics, and charts.
- One recognizable focus language across every workspace.
- Dedicated colors for HTTP methods, outcomes, and performance data.
- Responsive rendering that stays within the available terminal dimensions.
- A ten-character command trail in the sidebar so Vim movements and counts are
  visible as they are entered.

## Themes that are truly yours

Open the `?` command center and switch to `SETTINGS` to preview themes across
the entire application without restarting.

- Choose from `default`, terminal-aware `adaptive`, and no-color `mono` modes.
- Create lightweight custom themes with semantic YAML color roles.
- Preview changes transactionally, save them, or cancel to restore the exact
  previous appearance.
- Persist the selected theme atomically and reload it at the next startup.
- Use `VOLT_THEME` when you need an explicit one-run override.
- Ignore unsupported YAML fields for forward compatibility.
- Fall back safely and show a useful startup notice when a recognized value is
  invalid.

See [Themes and Customization](CUSTOMIZATION.md) for the complete discovery
order, schema, examples, and validation behavior.

## A startup signature, not permanent chrome

The full ASCII `VOLT` mark now greets you at startup, then compresses into a
compact two-row command-center header.

- The full signature remains visible for two seconds.
- The first meaningful key dismisses it immediately without swallowing the
  intended action.
- The compact header shows the active panel, operating mode, and release
  version.
- Working space is reclaimed as soon as the transition completes.
- Reduced-motion preferences skip the intermediate animation.

## Load testing becomes the centerpiece

Load testing now feels like the reason the product is called Volt.

- Configuration preserves concurrency, total requests, QPS, and timeout while
  changing modes.
- Every setting and the explicit `RUN LOAD TEST` action remains visible in
  supported layouts.
- The configuration editor receives more space before launch; live results
  receive more space while the test is running.
- A bounded progress bar pairs with exact completed and total request counts.
- Live throughput, p50 latency, error rate, and a real rolling
  interval-latency sparkline make performance changes immediately legible.
- Incoming updates preserve the result tab you selected.
- Completed runs settle into explicit `✓ COMPLETE` or `× COMPLETE` states.
- Normal workspace proportions return after completion or failure.

## Keyboard first from end to end

- Use `h/l` to move between Help and Settings or change an active choice.
- Use `j/k` to move through rows.
- Use `enter` to preview or save and `esc` to cancel or close.
- Use `Ctrl+L` from the request workspace to enter load-test mode.
- Use `Ctrl+X` from the response workspace to cancel an active load test.
- Press `F1` anywhere for the complete shortcut reference.

See the [keybindings guide](docs/keybindings.md) for every contextual binding.

## Reliability and verification

The new experience is covered at the component and full-application levels,
including theme discovery and precedence, atomic persistence, transactional
preview, startup transitions, responsive panel bounds, load-test
configuration, rolling live history, progress, and result states.

The full Go test suite passes for this release.

**Full changelog:** [v0.1.3...v0.2.0](https://github.com/owenHochwald/volt/compare/v0.1.3...v0.2.0)
