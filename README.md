<div>

# Volt

**Terminal-native API development and performance testing for humans, AI agents, and CI**

<br>

[![GitHub release](https://img.shields.io/github/release/owenHochwald/Volt.svg)](https://github.com/owenHochwald/Volt/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/owenHochwald/Volt)](https://goreportcard.com/report/github.com/owenHochwald/Volt)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

[Installation](#installation) • [Quick Start](#quick-start) • [AI Agents and CI](#ai-agents-and-ci) • [Customization](CUSTOMIZATION.md) • [Keybindings](docs/keybindings.md) • [Why Volt?](#why-volt) • [CLI Mode](#cli-load-testing)

![Demo](demo.gif)

</div>

---

## Overview

Volt is a terminal-native API development and performance-testing system. Its
interactive TUI makes requests easy to explore, while its composable CLI gives
developers, coding agents, and CI jobs the same high-performance load engine for
repeatable verification.

Volt is growing beyond a standalone TUI load tester into a workflow that can
help teams develop an endpoint, verify it functionally, measure it under
representative traffic, compare changes, and preserve evidence for review and
production-readiness decisions.

| Workflow | Best for | Interface |
|---|---|---|
| Interactive exploration | Creating requests and inspecting responses | `volt` TUI |
| Repeatable load tests | Local development and performance investigations | `volt bench` |
| Coding-agent verification | Measuring an API during implementation or review | Volt companion skill + CLI |
| CI evidence | Saving machine-readable results and detecting regressions | JSON output |

**Perfect for teams who:**

- Live in the terminal and hate context switching
- Want Postman's features without the Electron bloat
- Love Vim keybindings and keyboard-driven workflows
- Need a fast, scriptable HTTP client with a focused UI
- Want coding agents to evaluate the APIs they build
- Want reproducible performance evidence in CI


> **Note**: This is an active learning project. Performance optimizations are ongoing, and contributions/feedback are welcome :)

Volt's theme and color customization system is specified in
[Themes and Customization](CUSTOMIZATION.md). Custom themes are small YAML
files placed at `<user-config-dir>/volt/theme.yaml` or `~/.volt/theme.yaml`;
start by overriding only the semantic colors you want to change. Visual and
interaction decisions are maintained in the
[Volt Design System](DESIGN_SYSTEM.md).

## Why Volt?

|  | Postman | Insomnia | HTTPie | curl | **Volt** |
|---|:---:|:---:|:---:|:---:|:---:|
| **Terminal-native** | ❌ | ❌ | ✅ | ✅ | ✅ |
| **Interactive TUI** | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Vim keybindings** | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Syntax highlighting** | ✅ | ✅ | ✅ | ❌ | ✅ |
| **Save collections** | ✅ | ✅ | ❌ | ❌ | ✅ |
| **Zero install** | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Memory footprint** | ~500MB | ~300MB | ~50MB | <5MB | **~15MB** |
| **Startup time** | ~3s | ~2s | <1s | instant | **instant** |

<!-- BEGIN GENERATED BENCHMARK COMPARISON -->
### Throughput comparison

No benchmark results are checked in yet. Generate a fresh comparison on the
machine whose performance you want to document.
<!-- END GENERATED BENCHMARK COMPARISON -->

The comparison is generated from controlled local endpoints with pinned versions
of Volt, wrk, hey, and k6. Docker is the only runtime dependency:

```bash
# Short integration check
make benchmark-smoke

# Five 10-second runs for every workload and concurrency level
make benchmark-comparison
```

Progress is written to stderr and the paste-ready Markdown section is written to
stdout. Review a completed run, then replace the marked block above. To keep a
dated copy without changing the command:

```bash
make benchmark-comparison > benchmarks/comparison-$(date +%F).md
```

---

## Installation

Volt is distributed as a single binary with no dependencies. The fastest way to install is using Go's built-in package manager.

### Quick Install

If you have Go installed, you can install Volt with a single command:

```bash
go install github.com/owenHochwald/Volt/cmd/volt@latest # install
volt # run and verify
```


You should see the Volt TUI interface launch. Press `Esc` twice to quit.
**Updating Volt:**
To update to the latest version, simply run the install command again.

**Troubleshooting:**
If you get a "command not found" error, ensure `$GOPATH/bin` is in your PATH:
```bash
# Add to your ~/.bashrc, ~/.zshrc, or equivalent
export PATH="$PATH:$(go env GOPATH)/bin"
```
---

## Quick Start

Once installed, launch Volt's interactive interface:
```bash
volt
```
**Basic usage:**
- Type a URL and press `Alt+Enter` (`Option+Enter` on macOS) to make a request
- Press `?` to see all keybindings
- Press `Esc` twice to quit, or `Ctrl+C` to quit immediately

See the [keybindings guide](docs/keybindings.md) for every contextual shortcut
and macOS Option-key setup.

## AI Agents and CI

Volt's CLI is the tool-use interface for Codex, Claude Code, and other coding
agents. An agent can construct an authenticated request, begin with a bounded
smoke test, save JSON results, and report throughput, latency, status codes, and
failures alongside the code change it made.

Start with [Using Volt with AI coding agents](AI.md). The portable
[`volt-load-testing` skill](skills/volt-load-testing/SKILL.md) teaches agents
when and how to use Volt, including authorization checks, staged load, secret
handling, and before/after comparisons.

The current JSON output can also be saved as a CI artifact:

```bash
volt bench \
  -url http://127.0.0.1:8080/health \
  -c 10 \
  -d 30s \
  -rate 100 \
  -json \
  -o volt-results.json
```

Today, CI consumers must inspect the JSON result because HTTP request failures
do not automatically make the Volt process exit nonzero. Stable assertions,
exit codes, secret-safe inputs, scenario files, comparison reports, and
CI-native output are planned as future milestones.

## CLI Load Testing

Use the `bench` subcommand for non-interactive local, agent, and CI workflows.

```bash
volt bench [flags]
```

### Examples

```bash
# Basic throughput test
volt bench -url http://localhost:8080 -c 100 -d 30s

# POST request with custom headers
volt bench -url http://localhost:8080/api -m POST \
  -b '{"test":true}' -H "Content-Type: application/json"

# JSON output to file for CI/CD
volt bench -url http://localhost:8080 -c 50 -d 60s -json -o results.json

# Rate-limited testing
volt bench -url http://localhost:8080 -c 10 -d 30s -rate 1000

# For help!
volt bench -h
```

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](./LICENSE) file for details.

## Star History

If you find Volt useful, please consider giving it a star ⭐ on GitHub!

---
