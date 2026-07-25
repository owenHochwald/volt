<div>

# Volt

**A blazingly fast, terminal-native HTTP client and load tester with Vim keybindings**

<br>

[![GitHub release](https://img.shields.io/github/release/owenHochwald/Volt.svg)](https://github.com/owenHochwald/Volt/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/owenHochwald/Volt)](https://goreportcard.com/report/github.com/owenHochwald/Volt)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

[Installation](#installation) • [Quick Start](#quick-start) • [Keybindings](docs/keybindings.md) • [Customization](CUSTOMIZATION.md) • [UI Progress](SUCCESS.md) • [Why Volt?](#why-volt) • [CLI Mode](#cli-load-testing)

![Demo](demo.gif)

</div>

---

## Overview

Volt is a **keyboard-driven HTTP client** that lives in your terminal. Built as a project with Go and the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework, and high-performance HTTP client design.

**Perfect for developers who:**
- Live in the terminal and hate context switching
- Want Postman's features without the Electron bloat
- Love Vim keybindings and keyboard-driven workflows
- Need a fast, scriptable HTTP client with a beautiful UI


> **Note**: This is an active learning project. Performance optimizations are ongoing, and contributions/feedback are welcome :)

Volt's theme and color customization system is specified in
[Themes and Customization](CUSTOMIZATION.md). Custom themes are small YAML
files placed at `<user-config-dir>/volt/theme.yaml` or `~/.volt/theme.yaml`;
start by overriding only the semantic colors you want to change. Visual and
interaction decisions are maintained in the
[Volt Design System](DESIGN_SYSTEM.md), and current UI milestones are tracked
in [Volt UI Success Criteria](SUCCESS.md).

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

## CLI Load Testing

Volt also includes a powerful little HTTP load testing tool for direct access, accessible via the `bench` subcommand.

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
