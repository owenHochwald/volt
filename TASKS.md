# Volt Roadmap

This roadmap tracks the next Volt milestones and the product direction behind
them.

## Product Direction

Volt is evolving from a terminal HTTP client and standalone load tester into an
API development and performance-verification system with three first-class
surfaces:

- an interactive TUI for developers exploring requests and responses;
- a composable CLI for repeatable local and CI execution; and
- a portable skill that teaches coding agents how to verify the APIs they build.

The shared load engine and result model remain the product core. Agent and CI
workflows must use those same contracts rather than becoming separate
implementations.

AI-native does not mean embedding a model in Volt. It means making Volt safe and
effective for tool use: discoverable commands, secret-safe inputs, bounded
execution, machine-readable results, reproducible scenarios, explicit
assertions, and evidence an agent or CI job can explain.

CI evaluation should grade a service against declared budgets or a controlled
baseline. Do not publish a universal performance score that changes with runner
hardware, network placement, or neighboring workloads.

## Working Principles

- Correctness comes before comparative performance claims.
- Keep exact accounting separate from optional live UI/telemetry snapshots.
- Preserve raw benchmark evidence and document every public claim.
- Keep terminal actions context-aware and test every advertised keybinding.
- Keep secrets out of process arguments, logs, result files, and committed
  scenarios.
- Require explicit targets and bounded traffic for agent and CI workflows.
- Keep machine-readable stdout stable and send human diagnostics to stderr.
- Treat result schemas and exit codes as public compatibility contracts.
- Do not start distributed load generation until execution, statistics, scenarios,
  and mergeable latency distributions are stable.

## Milestone 1: Load-Testing Correctness

Branch: `feature/concurrency-and-loadtest-upgrades`

Status: Complete on `feature/concurrency-and-loadtest-upgrades`.

Implementation commits:

- `fa49e8cde688f323d7a7c5b6448b0018d1bcdb0e`
- `15ef4a964d444c82f45cc1c128fed7b096f6ef5f`

- [x] Make duration mode stop at the requested duration instead of estimating a
      request count.
- [x] Apply one global rate limit across all workers.
- [x] Guarantee exact final request, failure, status, and byte accounting.
- [x] Stop dropping worker statistics that contribute to final results.
- [x] Compute percentiles from request latency samples rather than batch averages.
- [x] Preserve sub-millisecond latency precision.
- [x] Calculate mean latency using the actual number of measured requests.
- [x] Define and test HTTP status, transport error, timeout, and cancellation
      classification.
- [x] Wire `context.Context` cancellation from CLI signals through every worker.
- [x] Apply `-keepalive` and `-no-keepalive` to the HTTP engine.
- [x] Add deterministic tests for duration, global QPS, cancellation, latency
      distribution, byte counts, errors, and keep-alive behavior.
- [x] Run formatting, vet, normal tests, race tests, and benchmark comparisons.

Verification:

- `go test ./...`, `go test -race ./...`, and `go vet ./...` pass.
- A 10,000-request test reconciles exact totals, status counts, percentile sample
  count, and payload bytes.
- Local-loopback throughput on Apple M4 remained effectively flat across
  concurrency 10, 50, 100, and 500.

Completion gate:

- Duration and QPS stay within documented tolerances.
- Cancellation terminates all workers without leaks.
- Exact counters reconcile at the end of every run.
- Synthetic latency distributions produce expected mean and percentile results.
- Existing throughput does not regress without a measured and documented reason.

## Milestone 2: Bubble Tea v2 and Terminal UX

Reference:
<https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md>

Branch: `feature/bubbletea-migration-and-ui-overheaul`

Status: Complete on `feature/bubbletea-migration-and-ui-overheaul`.

Implementation commits:

- `9b38e21` — migrate terminal UI to Bubble Tea v2
- `03ac2b1` — add tested terminal action registry
- `31df4fd` — enforce contextual Vim keybindings and focus
- `0e74248` — preserve raw request body content
- `45d633a` — add responsive terminal layouts
- `9f3af7f` — surface load test HTTP failures in results
- `15a415e` — reject short malformed request URLs safely
- `8ef89c9` — surface terminal actions and load test outcomes
- `fcae702` — verify every contextual keybinding
- `4798837` — define contextual q behavior
- `ac1acc3` — stabilize viewport and field navigation
- `abc9357` — derive tab labels from action registry
- `a034c61` — keep long help bindings on one line

Target stable versions assessed on 2026-07-25:

- Bubble Tea `v2.0.8`
- Bubbles `v2.1.1`
- Lip Gloss `v2.0.5`

Sequence this work after Milestone 1 because both migrations touch the app-level
load-test message flow and `go.mod`.

### Dependency migration

- [x] Confirm compatible Bubble Tea v2, Bubbles v2, and Lip Gloss v2 versions.
- [x] Follow the official v2 import-path and API migration guidance.
- [x] Update event, command, keyboard, window-size, and renderer APIs.
- [x] Keep the dependency migration separate from load-engine changes.
- [x] Add UI regression tests before changing interaction semantics.

### Vim-oriented action and keymap system

- [x] Introduce an action registry with context, keys, description, and priority.
- [x] Generate all help content from the action registry.
- [x] Reserve Tab and Shift+Tab for field navigation.
- [x] Use a separate, unambiguous panel-navigation binding.
- [x] Leave arrow keys available to text inputs and text areas.
- [x] Use Ctrl+Enter or Alt+Enter to submit; keep Enter for editing and accepting
      suggestions.
- [x] Make `?` context-aware and provide an always-global help key such as F1.
- [x] Define when `q` quits and when it inserts text.
- [x] Properly focus and blur controls when moving between panels.
- [x] Support raw JSON and other raw request bodies.
- [x] Add responsive narrow-terminal and short-terminal layouts.
- [x] Surface save, delete, copy, request, and load-test errors and confirmations.
- [x] Add table-driven tests for every action in every relevant context.

Verification:

- `go test ./...`, `go test -race ./...`, and `go vet ./...` pass.
- A production `go build ./cmd/volt` succeeds.
- Responsive snapshots fill wide, narrow, short, and minimum supported terminal
  sizes without negative dimensions or overflow.
- A real 88×35 PTY smoke test verified alternate-screen startup, compact panel
  navigation, context-aware help, and clean quit behavior.
- Load-test results visibly classify HTTP 4xx/5xx responses as failures and show
  sorted status-code and error-class breakdowns.
- Every commit uses the repository owner's configured Git identity and contains
  no AI attribution or co-author trailer.

Completion gate:

- Every displayed shortcut maps to one tested action.
- No key is ambiguously consumed in the same context.
- The entire normal workflow is navigable without a mouse or arrow keys.
- Text editing, multiline bodies, suggestions, modal help, and quitting behave
  consistently.
- Supported terminal sizes render without negative dimensions or clipped controls.

## Milestone 3: Reproducible wrk, hey, and k6 Benchmarks

References:

- <https://github.com/wg/wrk>
- <https://github.com/rakyll/hey>
- <https://grafana.com/docs/k6/latest/using-k6/k6-options/reference/>

- [ ] Build a dedicated harness instead of extending the public-endpoint script.
- [ ] Pin exact Volt, wrk, hey, and k6 versions.
- [ ] Provide controlled endpoints for empty, 1 KB, JSON, fixed-latency, and error
      responses.
- [ ] Run the target separately from the load generator.
- [ ] Use separate machines for headline results.
- [ ] Match HTTP version, keep-alive, connection count, response handling, and
      request bodies.
- [ ] Include warm-up, randomized tool order, cooldown, and at least five runs.
- [ ] Validate generated load using server-side request counts.
- [ ] Record CPU, RSS, throughput, transfer rate, errors, and latency percentiles.
- [ ] Publish raw JSON/CSV, hardware and OS details, commands, tool versions,
      commit SHA, medians, and confidence intervals.
- [ ] Keep hosted CI benchmarks limited to smoke and regression detection.
- [ ] Remove the Postman throughput comparison.

Completion gate:

- A new contributor can reproduce the benchmark from documented commands.
- Raw results reconcile with server-side counts.
- Each comparison clearly documents differences in the tools' execution models.

## Milestone 4: Releases and Homebrew

GoReleaser reference:
<https://www.goreleaser.com/customization/publish/homebrew_casks/>

- [ ] Run tests, vet, and a benchmark smoke test in CI.
- [ ] Run race tests on a schedule and before releases.
- [ ] Validate the GoReleaser configuration before merging.
- [ ] Add `volt version` with version, commit, and build date.
- [ ] Decide and document manual-draft versus automatic release publication.
- [ ] Publish checksums, an SBOM, and artifact signing/provenance.
- [ ] Create `owenHochwald/homebrew-tap`.
- [ ] Replace the obsolete commented `brews` setup with `homebrew_casks`.
- [ ] Resolve and document lowercase/uppercase module identity before v1.
- [ ] Move the pull-request template to
      `.github/pull_request_template.md`.

Completion gate:

- A tagged release passes all checks and publishes verifiable cross-platform
  artifacts.
- Homebrew installation and upgrade work from a clean machine.
- The binary reports the same version as its release.

## Milestone 5: Agent-Safe CLI Inputs and Result Contracts

Make the CLI safe and predictable enough for developers, coding agents, and CI
jobs to use without shell-quoting fragile payloads or inferring success from
human-readable output.

### Functional preflight and request inputs

- [ ] Add `volt request` for one functional request with structured output
      before any concurrent load is sent.
- [ ] Add `--body-file <path|->` with stdin support for JSON and arbitrary raw
      request bodies.
- [ ] Add secret-aware header inputs that accept an environment-variable name
      and resolve its value inside Volt without placing the secret in process
      arguments.
- [ ] Add a versioned `--request <path|->` JSON request specification for URL,
      method, headers, body sources, timeout, and environment references.
- [ ] Define configuration precedence across flags, request files, environment
      references, and defaults.
- [ ] Add `--dry-run` to print the normalized target and workload with all
      secret values redacted.
- [ ] Reject unresolved secret references before sending traffic.
- [ ] Keep inline `-H` and `-b` behavior compatible while documenting their
      process-list exposure on shared systems.
- [ ] Add table-driven tests for files, stdin, empty bodies, malformed specs,
      missing variables, duplicate headers, redaction, and backward
      compatibility.

### Machine-readable results and outcomes

- [ ] Version the JSON result schema.
- [ ] Include Volt version, run ID, timestamps, target host, method, requested
      workload, achieved workload, timeout, keep-alive mode, interruption state,
      and redacted request identity.
- [ ] Keep stdout valid machine-readable output in JSON mode and write progress,
      warnings, and diagnostics to stderr.
- [ ] Define stable exit codes for invalid usage, failed execution, request
      failures, interruption, and unmet assertions.
- [ ] Add assertions for maximum failure rate, allowed status codes, minimum
      throughput, and maximum p50, p95, and p99 latency.
- [ ] Make assertion results explicit in JSON rather than encoding them only in
      the exit code.
- [ ] Add `volt version` and `volt doctor` so an agent or CI job can verify the
      installed version, supported capabilities, and environment before a run.
- [ ] Publish the schema and exit-code contract with compatibility tests.

Completion gate:

- An agent can preflight and load test an authenticated JSON request without
  exposing the credential or body in its command arguments.
- Invalid configuration and unresolved secrets send zero requests.
- JSON stdout parses cleanly on success, request failure, assertion failure,
  interruption, and execution failure.
- Exit codes and JSON outcomes agree in every tested state.
- Existing `volt bench` commands remain compatible.

## Milestone 6: Reproducible Scenarios and CI Evaluation

Turn one-off commands into reviewable performance specifications that run the
same way locally, from a coding agent, and in CI.

### Scenario format and execution

- [ ] Define a versioned `volt.yaml` and equivalent JSON scenario schema.
- [ ] Represent named requests, environment references, concurrency, rate,
      duration or request count, timeout, keep-alive behavior, and assertions.
- [ ] Support explicit warm-up, steady-state, and bounded stress stages.
- [ ] Support multiple weighted endpoints without hiding per-endpoint failures
      or latency distributions.
- [ ] Define safe setup and teardown hooks for disposable test data without
      allowing arbitrary shell execution by default.
- [ ] Support OpenAPI-assisted scenario scaffolding while requiring the user or
      agent to choose safe example data, authentication references, and traffic
      limits.
- [ ] Write a fully redacted run manifest beside every result so another
      developer can reproduce the workload.
- [ ] Record runner hardware, OS, Volt version, source revision, target
      environment, and load-generator placement as result metadata.
- [ ] Validate scenarios without sending traffic.

### Comparisons, reports, and grading

- [ ] Add `volt compare before.json after.json` with absolute and percentage
      deltas for throughput, latency, failures, status codes, and transfer.
- [ ] Refuse misleading comparisons when material workload or environment
      metadata differs unless the user explicitly overrides the warning.
- [ ] Support repeated runs and median or confidence-interval summaries for
      regression decisions.
- [ ] Produce concise terminal, JSON, Markdown, and JUnit-compatible reports.
- [ ] Grade against scenario-defined budgets or an approved baseline; show every
      contributing metric and never hide the verdict behind an opaque score.
- [ ] Distinguish functional failure, performance-budget failure,
      infrastructure failure, and inconclusive/noisy measurement.
- [ ] Add CI examples that start a service, wait for readiness, run a bounded
      scenario, upload raw results and the redacted manifest, and publish a
      pull-request summary.
- [ ] Keep hosted shared-runner examples focused on smoke tests and controlled
      before/after regression checks rather than headline capacity claims.

### Agent distribution

- [ ] Publish and version the `volt-load-testing` companion skill.
- [ ] Document personal and project installation for Codex, Claude Code, and
      other Agent Skills-compatible tools.
- [ ] Keep the skill concise and make it discover current CLI capabilities
      before constructing commands.
- [ ] Add realistic skill evaluations for authenticated requests, mutating
      endpoints, missing authorization, smoke-test failure, baseline
      comparison, and CI report generation.
- [ ] Do not make MCP a milestone or dependency. Reconsider a local MCP adapter
      only after CLI schemas stabilize and only if typed tool discovery provides
      clear value beyond the skill and command.

Completion gate:

- The same committed scenario produces equivalent requested load and assertion
  semantics locally, under an agent, and in CI.
- Every CI verdict links to raw machine-readable evidence and a redacted,
  reproducible run manifest.
- Comparisons detect mismatched workloads and runner environments.
- A clean agent session can discover the skill, preflight the endpoint, run a
  bounded scenario, and explain the result without exposing secrets.

## Milestone 7: JSON Request Interpolation and QUERY Support

Replace the UI request pane's line-oriented key/value parsing with full JSON
interpolation and parsing for request data. Preserve user-entered JSON structure
and provide first-class support for the newly developed `QUERY` HTTP verb.

### Request parsing and interpolation

- [ ] Define the supported interpolation syntax, evaluation order, escaping
      rules, and behavior for missing or invalid values.
- [ ] Parse request JSON as JSON instead of converting key/value lines into a
      map and marshaling that map back to JSON.
- [ ] Apply interpolation recursively across JSON objects, arrays, and scalar
      values while preserving valid JSON types where applicable.
- [ ] Decide whether headers use the same JSON/interpolation representation or
      retain a dedicated header editor, and document the boundary between the
      two formats.
- [ ] Preserve raw request bodies, nested objects, arrays, duplicate-free field
      ordering where meaningful, and intentional `null`, boolean, and numeric
      values.
- [ ] Surface syntax, interpolation, and validation errors in the request pane
      without silently replacing the body with `{}`.
- [ ] Add migration guidance or compatibility handling for saved requests that
      still use the legacy key/value format.
- [ ] Add table-driven tests for nested JSON, arrays, escaped strings, scalar
      interpolation, missing values, malformed JSON, and legacy input.

### QUERY method support

- [ ] Add `QUERY` to the UI method selector and request validation rules.
- [ ] Ensure `QUERY` is preserved through request construction, persistence,
      normal sends, and load-test configuration.
- [ ] Verify the HTTP client, headers, body handling, and response rendering for
      `QUERY` requests.
- [ ] Add focused tests covering UI selection, validation, serialization, and a
      real client request using the `QUERY` method.
- [ ] Update help text, documentation, examples, and any method-specific UI
      behavior to describe `QUERY` accurately.

Completion gate:

- Valid JSON request structure reaches the HTTP client unchanged except for the
  documented interpolation results.
- Invalid JSON and interpolation failures are visible, actionable, and do not
  corrupt the previously valid request state.
- Existing saved requests remain usable or have a documented, tested migration
  path.
- `QUERY` works consistently from the UI through normal requests, persistence,
  and load tests.
- Focused tests and the full repository test suite pass.

## Milestone 8: OpenTelemetry Metrics

This starts only after Milestone 1 establishes a stable result model.

- [ ] Add asynchronous OTLP metric export.
- [ ] Export request, failure, status-class, byte, and cancellation counters.
- [ ] Export request-latency histograms.
- [ ] Export active-worker and achieved-rate gauges.
- [ ] Use low-cardinality run ID, method, target host, scenario, and status-class
      attributes.
- [ ] Never use raw URLs, headers, or bodies as labels.
- [ ] Do not emit a span per request by default.
- [ ] Flush exporters at run completion without blocking workers.

Completion gate:

- Telemetry counters reconcile with the final local result.
- Exporting can be disabled with effectively no hot-path cost.
- Slow or unavailable collectors cannot stall load generation.

## Final Documentation and Community Upgrade

- [ ] Replace `CLAUDE.md` with accurate architecture, commands, constraints, and
      current milestone guidance.
- [ ] Add an accurate root `AGENTS.md`.
- [ ] Split architecture, benchmark methodology, and performance results into
      focused documents.
- [ ] Triage existing issues into reproducible bugs, roadmap features, and
      questions.
- [ ] Publish genuinely scoped `good first issue` tasks.
- [ ] Add a security policy, code of conduct, contributor setup check, and public
      roadmap.
- [ ] Recognize outside contributors in release notes.
- [ ] Track release downloads, outside contributors, repeat contributors, real CI
      usage, companion-skill adoption, and progress toward 500 stars.
- [ ] Launch the reproducible benchmark report and architecture article with the
      Homebrew release.

Completion gate:

- Contributor documentation matches the code and passes a clean-machine setup
  check.
- Public performance claims link to reproducible raw evidence.
- Community metrics measure sustained usage and contribution, not stars alone.
