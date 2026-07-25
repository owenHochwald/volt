---
name: volt-load-testing
description: Safely exercise and evaluate HTTP APIs with the Volt CLI, including authenticated requests, JSON bodies, staged load, machine-readable results, performance baselines, and before/after comparisons. Use when developing or reviewing an API and the user asks to load test, benchmark, measure latency or throughput, check behavior under concurrency, investigate performance regressions, or run Volt.
---

# Volt Load Testing

Use Volt's non-interactive `bench` command. Treat load generation as a
potentially disruptive operation and keep every run authorized and bounded.

## Establish scope

Before sending traffic:

- Confirm that the user owns the target or is authorized to test it.
- Prefer localhost, a preview environment, or a dedicated test environment.
- Do not test production, shared infrastructure, or a third-party API without
  explicit approval and a traffic budget.
- Determine whether the request mutates data. Use disposable data and verify
  that repeating it is safe.
- Read route definitions, an OpenAPI document, tests, or an existing curl
  example to obtain the URL, uppercase method, headers, body, and expected
  successful statuses.
- Use a user-provided service objective. If none exists, report a baseline
  without declaring it a pass or failure.

Never paste, print, or commit a secret. Refer to a credential already supplied
through an environment variable, such as `${API_TOKEN}`.

## Follow the staged workflow

1. Verify that `volt` is on `PATH` and inspect `volt bench -h`.
2. Verify one functional request with the service's tests or curl. Volt does not
   currently provide a CLI single-request preflight.
3. Run a five-request smoke test with one connection and `-rate 1`.
4. Inspect the JSON result before increasing traffic.
5. Run an unrecorded warm-up.
6. Run a representative, explicitly rate-limited baseline.
7. Increase load only when the user requested it and the prior stage remained
   healthy.
8. Save JSON evidence and report the workload with secrets redacted.

Do not jump directly to unlimited traffic. Stop when failures rise unexpectedly
or the target becomes unhealthy.

## Build commands

Use the explicit subcommand and JSON output:

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

For a duration-based baseline:

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

Use `-n` for a fixed count or `-d` for a time window; they are mutually
exclusive. Treat `-rate 0` as unlimited. Keep-alive is enabled unless
`-no-keepalive` is set. Repeat `-H` for multiple headers.

The current CLI accepts only an inline body. Keep payloads in a file while
editing and expand them at invocation. Note that expanded headers and bodies can
be visible in the process list on a shared machine; tell the user when this
limitation matters.

## Interpret results

Read these JSON fields:

- `summary.completedRequests`, `failedRequests`, `successRate`, `throughput`,
  and `durationMs`
- `latency.minMs`, `avgMs`, `p50Ms`, `p90Ms`, `p95Ms`, `p99Ms`, and `maxMs`
- `statusCodes` and `errors`
- `transfer.bytesSent` and `bytesReceived`

A completed Volt command currently exits successfully even if HTTP requests
failed. Inspect `failedRequests`, status codes, and errors rather than treating a
zero process exit code as a passing test.

Report:

- authorized environment and code revision;
- exact method, path, concurrency, rate, duration or count, timeout, and
  keep-alive mode, with secrets redacted;
- completed requests, failures, achieved throughput, p50, p95, and p99;
- status and error breakdowns;
- whether each user-supplied objective was met; and
- measurement limitations, especially shared CPU on a local target.

For before/after comparisons, keep the environment, request, data, concurrency,
rate, execution bound, timeout, keep-alive behavior, and warm-up identical.
Preserve both JSON files. Avoid claiming significance from a single small
difference.

## Handle failures

- If the functional preflight fails, diagnose it before load testing.
- If the target or authorization is ambiguous, do not send load; ask for the
  missing scope.
- If the token is absent, report the environment variable name without asking
  the user to paste its value.
- If the smoke test has failures, stop and report its JSON breakdown.
- If no objective exists, return measurements and recommended next experiments,
  not an invented verdict.
