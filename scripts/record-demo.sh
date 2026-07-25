#!/usr/bin/env bash
# Regenerate demo.gif with a deterministic local endpoint and isolated settings.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

command -v vhs >/dev/null || {
  echo "vhs is required; install it from https://github.com/charmbracelet/vhs" >&2
  exit 1
}
command -v python3 >/dev/null || {
  echo "python3 is required to run the local demo endpoint" >&2
  exit 1
}

demo_tmp="$(mktemp -d)"
demo_home="$demo_tmp/home"
mkdir -p "$demo_home"

python3 -m http.server 18080 --bind 127.0.0.1 --directory "$repo_root" >/dev/null 2>&1 &
server_pid=$!
cleanup() {
  kill "$server_pid" 2>/dev/null || true
  wait "$server_pid" 2>/dev/null || true
  rm -rf "$demo_tmp"
}
trap cleanup EXIT INT TERM

go build -o volt ./cmd/volt/main.go
HOME="$demo_home" vhs demo-current.tape
