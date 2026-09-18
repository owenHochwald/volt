#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/compose.yaml"

if ! command -v docker >/dev/null 2>&1; then
    echo "benchmark comparison requires Docker" >&2
    exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
    echo "benchmark comparison requires Docker Compose v2" >&2
    exit 1
fi

VOLT_COMMIT="$(git -C "${REPOSITORY_ROOT}" rev-parse --short=12 HEAD)"
BENCH_HOST_OS="$(uname -sm)"
if command -v sysctl >/dev/null 2>&1; then
    BENCH_HOST_CPU="$(sysctl -n machdep.cpu.brand_string 2>/dev/null || true)"
fi
if [[ -z "${BENCH_HOST_CPU:-}" ]] && command -v lscpu >/dev/null 2>&1; then
    BENCH_HOST_CPU="$(lscpu | awk -F: '/Model name/ {sub(/^[[:space:]]+/, "", $2); print $2; exit}')"
fi
BENCH_HOST_CPU="${BENCH_HOST_CPU:-unknown}"
BENCH_DOCKER_VERSION="$(docker version --format '{{.Client.Version}}' 2>/dev/null || docker --version)"
BENCH_SEED="${BENCH_SEED:-$(date +%Y%m%d)}"

export VOLT_COMMIT BENCH_HOST_OS BENCH_HOST_CPU BENCH_DOCKER_VERSION BENCH_SEED

cleanup() {
    docker compose -f "${COMPOSE_FILE}" down --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

echo "Building pinned benchmark tools..." >&2
docker compose -f "${COMPOSE_FILE}" build runner target >&2
echo "Starting controlled benchmark target..." >&2
docker compose -f "${COMPOSE_FILE}" up -d target >&2
echo "Running comparison; progress is written to stderr..." >&2
docker compose -f "${COMPOSE_FILE}" run --rm runner
