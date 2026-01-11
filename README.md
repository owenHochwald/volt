<div>

# Volt

**A blazingly fast, terminal-native HTTP client and load tester with Vim keybindings**

<br>

[![GitHub release](https://img.shields.io/github/release/owenHochwald/Volt.svg)](https://github.com/owenHochwald/Volt/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/owenHochwald/volt)](https://goreportcard.com/report/github.com/owenHochwald/volt)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

[Installation](#installation) • [Quick Start](#quick-start) • [Why Volt?](#why-volt)  •[CLI Mode](#CLI-load-testing)

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

### Throughput Benchmarks (Apple M4)

*Testing against a zero-latency local endpoint to measure engine overhead:*

| Concurrency | Requests/Sec |
|-------------|--------------|
| 10          | 141,533      |
| 50          | 208,035      |
| **100**     | **213,885**  |
| 500         | 92,891       |

---

## Installation

Volt is distributed as a single binary with no dependencies. The fastest way to install is using Go's built-in package manager.

### Quick Install

If you have Go installed, you can install Volt with a single command:

```bash
go install github.com/owenHochwald/volt/cmd/volt@latest # install
volt # run and verify
```


You should see the Volt TUI interface launch. Press `q` to quit.
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
- Type a URL and press `alt+Enter` to make a request
- Press `?` to see all keybindings
- Press `q` to quit

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

