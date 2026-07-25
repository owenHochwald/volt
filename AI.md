# Using Volt with AI coding agents

Volt's CLI is the primary interface for Codex, Claude Code, and other coding
agents. It lets an agent exercise an API from the same terminal where the agent
builds and tests it, then consume structured performance results without
driving the interactive TUI.

This guide is for agents and developers **using Volt on another service**. It is
not contributor guidance for developing Volt itself.

## What Volt gives an agent

An agent can use Volt to:

- verify that an API remains responsive under concurrent requests;
- measure throughput and p50, p95, and p99 latency;
- observe HTTP status codes, transport failures, and timeouts;
- reproduce an exact request method, headers, bearer token, and body;
- compare results before and after a code change; and
- save machine-readable JSON for analysis, CI artifacts, or a pull request.

Volt is most useful after functional tests pass. It complements API tests; it
does not prove that a response body is semantically correct.

## Safe operating contract

Load tests create real traffic and can change real data.

1. Test only targets the user owns or is explicitly authorized to test.
2. Prefer a local, preview, or dedicated load-test environment.
3. Confirm that non-GET requests are safe and idempotent, or use disposable
   test data.
4. Start with one connection and a low global request rate.
5. Increase load in bounded stages. Do not jump directly to unlimited traffic.
6. Do not test production, third-party APIs, or shared services without
   explicit approval and an agreed traffic budget.
7. Stop if the target becomes unhealthy or failures rise unexpectedly.
8. Never print, commit, or place secrets directly in a prompt. Reference an
   environment variable that the user has populated.

## Agent workflow

### 1. Discover the target

Read the service's route definitions, OpenAPI document, tests, or existing curl
examples. Determine:

- URL and uppercase HTTP method;
- required headers and authentication;
- a representative request body;
- whether the operation mutates data;
- expected successful status codes;
- the environment the user authorized; and
- any latency, error-rate, or throughput objective.

Do not invent a performance threshold. If the user has not supplied one, report
the measured baseline and label it as a baseline rather than pass or fail.

### 2. Verify one request

Volt currently exposes load testing through the CLI, not a single-request
command. Verify the request with the service's tests or curl before adding
concurrency:

```bash
curl --fail-with-body \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data-binary @request.json \
  http://127.0.0.1:8080/api/widgets
```

Do not load test a request that is already functionally incorrect.

### 3. Run a bounded smoke test

Use a fixed request count, one connection, and a low rate:

```bash
volt bench \
  -url http://127.0.0.1:8080/api/widgets \
  -m POST \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -H "Content-Type: application/json" \
  -b "$(<request.json)" \
  -c 1 \
  -n 5 \
  -rate 1 \
  -json \
  -o volt-smoke.json
```

The token is expanded by the shell; its value should never be written in the
command text or result file. Be aware that the expanded header and inline body
can still be visible in the process list on a shared machine. Secret-safe input
flags are part of the proposed roadmap below.

Inspect `summary.failedRequests`, `statusCodes`, and `errors`. A completed Volt
process currently exits successfully even when some HTTP requests fail, so the
JSON result—not only the process exit code—is the source of truth.

### 4. Establish a baseline

Choose load that represents the service rather than the largest number the
machine can generate. Keep the rate explicit:

```bash
volt bench \
  -url http://127.0.0.1:8080/api/widgets \
  -m GET \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -c 10 \
  -d 30s \
  -rate 100 \
  -t 5s \
  -json \
  -o volt-baseline.json
```

Use `-n` for deterministic request counts or `-d` for a time window. They are
mutually exclusive. `-rate` is the global requests-per-second limit; zero means
unlimited. Keep-alive is enabled unless `-no-keepalive` is supplied.

Run an unrecorded warm-up before a result that will be compared. For before/after
measurements, keep the target environment, request, rate, concurrency, duration,
timeout, and keep-alive behavior identical.

### 5. Interpret and report

JSON output contains:

```text
summary.completedRequests
summary.failedRequests
summary.successRate
summary.throughput
summary.durationMs
latency.minMs
latency.avgMs
latency.p50Ms
latency.p90Ms
latency.p95Ms
latency.p99Ms
latency.maxMs
statusCodes
errors
transfer.bytesSent
transfer.bytesReceived
```

An agent should report:

- the exact workload, excluding secret values;
- target environment and relevant code revision;
- completed and failed requests;
- achieved requests per second;
- p50, p95, and p99 latency;
- status-code and error breakdowns;
- whether a user-supplied objective was met; and
- limitations such as a local target sharing CPU with the load generator.

Preserve the JSON file when the result supports a regression claim. Do not
claim that a small difference is meaningful without repeated runs in a
controlled environment.

## Current CLI reference

```text
volt bench
  -url <url>          required; http:// or https://
  -m <method>         GET, POST, PUT, DELETE, or PATCH
  -H "Key: Value"     repeatable request header
  -b <body>           inline request body
  -c <connections>    concurrency; default 50
  -d <duration>       execution duration; default 10s
  -n <count>          fixed request count; replaces duration
  -t <duration>       per-request timeout; default 30s
  -rate <rps>         global rate limit; 0 is unlimited
  -keepalive          enable keep-alive; default
  -no-keepalive       disable keep-alive
  -json               JSON result on stdout or in -o
  -q                  one-line human-readable result
  -o <path>           write the result to a file
```

Prefer the explicit `volt bench` form even though Volt currently accepts bench
flags without the subcommand.

## Install the companion skill

The portable skill in
[`skills/volt-load-testing`](skills/volt-load-testing/SKILL.md) teaches a coding
agent this workflow only when load testing is relevant.

For personal Codex use:

```bash
mkdir -p ~/.codex/skills
cp -R skills/volt-load-testing ~/.codex/skills/
```

For personal Claude Code use:

```bash
mkdir -p ~/.claude/skills
cp -R skills/volt-load-testing ~/.claude/skills/
```

The same folder can be copied into the agent's project-level skill directory
when a team wants to commit the workflow with an API repository. Review any
skill before installing it.

Example prompts:

```text
Use $volt-load-testing to establish a local baseline for GET /api/search at
100 requests per second.
```

```text
Use Volt to compare this branch against main. Keep the request rate and all
other test settings identical, save both JSON results, and summarize the delta.
```

```text
Use Volt to test this authenticated POST endpoint. Verify one request first,
use API_TOKEN from the environment, and begin with a five-request smoke test.
```

## Path to a more AI-native Volt

The CLI plus companion skill is the right first integration. Coding agents
already have a shell, and a CLI remains useful to humans and CI without adding
another protocol surface.

### Priority 1: safe, composable inputs and outcomes

- Add `volt request` for one functional preflight request with structured
  output.
- Add `--body-file <path|->` so agents do not need shell quoting or inline
  payloads.
- Add secret-aware inputs such as
  `--header-env "Authorization=API_TOKEN:Bearer "` without exposing values in
  command arguments, logs, or saved configuration.
- Add `--request <path|->` for a versioned JSON request specification and
  stdin-based composition.
- Add `--dry-run` that prints a redacted, normalized request and workload.
- Version the JSON result schema and include Volt version, target host, method,
  requested workload, achieved workload, timestamps, and interruption state.
- Define stable exit codes for usage errors, execution errors, request failures,
  and unmet assertions.
- Add performance assertions such as maximum failure rate, minimum throughput,
  and maximum p95/p99 latency.
- Keep stdout machine-readable in JSON mode and send diagnostics to stderr.
- Add `volt version` and `volt doctor` so an agent can verify availability and
  capabilities cheaply.

### Priority 2: reproducible API scenarios

- Add a declarative, versioned `volt.yaml` or JSON scenario format.
- Support rate/concurrency stages for warm-up, steady state, and bounded stress.
- Support multiple weighted endpoints and named scenarios.
- Support environment-variable references without persisting resolved secrets.
- Import operations from OpenAPI, while requiring the user or agent to choose
  safe example data and traffic limits.
- Add `volt compare before.json after.json` with absolute and percentage deltas.
- Preserve a fully redacted run manifest beside each result.

### Priority 3: distribution and integrations

- Publish the companion skill as an installable artifact for Codex, Claude Code,
  and other Agent Skills-compatible tools.
- Provide CI examples that upload JSON artifacts, publish reviewable reports,
  and enforce explicit service budgets.
- Support transparent CI evaluation against declared thresholds or a controlled
  baseline. Avoid a universal score that changes with runner hardware or network
  placement.
- Do not prioritize MCP. Volt is a local command and the CLI plus skill already
  matches how coding agents work. Reconsider a local MCP adapter only after the
  CLI schemas are stable and only if typed tool discovery provides clear value;
  it must wrap the same contracts rather than become a second implementation.

The near-term definition of “AI native” is therefore not “add an AI model to
Volt.” It is: make every useful operation discoverable, bounded, secret-safe,
machine-readable, reproducible, and easy for an agent to verify.
