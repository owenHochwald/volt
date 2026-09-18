#!/usr/bin/env bash
# Regenerate demo.gif against the local demo server with isolated settings.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

command -v vhs >/dev/null || {
  echo "vhs is required; install it from https://github.com/charmbracelet/vhs" >&2
  exit 1
}
demo_tmp="$(mktemp -d)"
demo_home="$demo_tmp/home"
mkdir -p "$demo_home"

cleanup() {
  rm -rf "$demo_tmp"
}
trap cleanup EXIT INT TERM

go build -o volt ./cmd/volt/main.go
env -u NO_COLOR HOME="$demo_home" VOLT_THEME=default vhs demo-current.tape
