#!/usr/bin/env python3
"""Run reproducible local comparisons and print paste-ready README Markdown."""

from __future__ import annotations

import json
import os
import platform
import random
import re
import statistics
import subprocess
import sys
import tempfile
import time
import urllib.request
from dataclasses import dataclass
from datetime import date
from pathlib import Path
from typing import Callable


TOOLS = ("volt", "wrk", "hey", "k6")
WORKLOADS = (
    ("Empty response", "/empty"),
    ("1 KiB response", "/bytes/1024"),
    ("JSON response", "/json"),
    ("10 ms delay", "/delay/10ms"),
    ("HTTP 500", "/status/500"),
)


@dataclass(frozen=True)
class ToolResult:
    requests: int
    requests_per_second: float


@dataclass(frozen=True)
class ScenarioResult:
    workload: str
    concurrency: int
    medians: dict[str, float]


def parse_volt(output: str) -> ToolResult:
    try:
        result = json.loads(output)
        return ToolResult(
            requests=int(result["summary"]["completedRequests"]),
            requests_per_second=float(result["summary"]["throughput"]),
        )
    except (KeyError, TypeError, ValueError, json.JSONDecodeError) as error:
        raise ValueError(f"invalid Volt JSON: {error}") from error


def parse_wrk(output: str) -> ToolResult:
    requests = re.search(r"(\d+)\s+requests in\s+", output)
    requests_per_second = re.search(r"Requests/sec:\s*([0-9.]+)", output)
    if not requests or not requests_per_second:
        raise ValueError("wrk output is missing request count or Requests/sec")
    return ToolResult(
        requests=int(requests.group(1)),
        requests_per_second=float(requests_per_second.group(1)),
    )


def parse_hey(output: str) -> ToolResult:
    requests_per_second = re.search(r"Requests/sec:\s*([0-9.]+)", output)
    status_counts = re.findall(r"\[\d+\]\s+(\d+)\s+responses", output)
    if not requests_per_second or not status_counts:
        raise ValueError("hey output is missing Requests/sec or status counts")
    return ToolResult(
        requests=sum(int(count) for count in status_counts),
        requests_per_second=float(requests_per_second.group(1)),
    )


def parse_k6(output: str) -> ToolResult:
    try:
        result = json.loads(output)
        return ToolResult(
            requests=int(result["requests"]),
            requests_per_second=float(result["requests_per_second"]),
        )
    except (KeyError, TypeError, ValueError, json.JSONDecodeError) as error:
        raise ValueError(f"invalid k6 summary JSON: {error}") from error


PARSERS: dict[str, Callable[[str], ToolResult]] = {
    "volt": parse_volt,
    "wrk": parse_wrk,
    "hey": parse_hey,
    "k6": parse_k6,
}


def log(message: str) -> None:
    print(message, file=sys.stderr, flush=True)


def env_int(name: str, default: int) -> int:
    raw = os.environ.get(name, str(default))
    try:
        value = int(raw)
    except ValueError as error:
        raise ValueError(f"{name} must be an integer, got {raw!r}") from error
    if value <= 0:
        raise ValueError(f"{name} must be greater than zero")
    return value


def parse_concurrency(raw: str) -> list[int]:
    try:
        values = [int(item.strip()) for item in raw.split(",")]
    except ValueError as error:
        raise ValueError("BENCH_CONCURRENCY must be comma-separated integers") from error
    if not values or any(value <= 0 for value in values):
        raise ValueError("BENCH_CONCURRENCY values must be greater than zero")
    return values


def request_json(url: str, method: str = "GET") -> dict[str, int]:
    request = urllib.request.Request(url, method=method)
    with urllib.request.urlopen(request, timeout=10) as response:
        if response.status not in (200, 204):
            raise RuntimeError(f"{method} {url} returned HTTP {response.status}")
        if response.status == 204:
            return {}
        return json.load(response)


def wait_for_target(base_url: str) -> None:
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        try:
            request_json(f"{base_url}/healthz")
            return
        except Exception:
            time.sleep(0.25)
    raise RuntimeError(f"benchmark target did not become healthy at {base_url}")


def tool_command(
    tool: str,
    url: str,
    concurrency: int,
    duration: str,
    temporary_directory: Path,
) -> tuple[list[str], dict[str, str], Path | None]:
    environment = os.environ.copy()
    summary_path: Path | None = None

    if tool == "volt":
        return (
            [
                "volt",
                "bench",
                "-url",
                url,
                "-c",
                str(concurrency),
                "-d",
                duration,
                "-keepalive",
                "-H",
                "Accept-Encoding: identity",
                "-json",
            ],
            environment,
            None,
        )
    if tool == "wrk":
        threads = min(concurrency, max(1, os.cpu_count() or 1))
        return (
            [
                "wrk",
                "-t",
                str(threads),
                "-c",
                str(concurrency),
                "-d",
                duration,
                "-H",
                "Accept-Encoding: identity",
                url,
            ],
            environment,
            None,
        )
    if tool == "hey":
        return (
            [
                "hey",
                "-z",
                duration,
                "-c",
                str(concurrency),
                "-disable-compression",
                "-H",
                "Accept-Encoding: identity",
                url,
            ],
            environment,
            None,
        )
    if tool == "k6":
        summary_path = temporary_directory / "k6-summary.json"
        environment.update(
            {
                "TARGET_URL": url,
                "DURATION": duration,
                "VUS": str(concurrency),
                "SUMMARY_PATH": str(summary_path),
            }
        )
        return (
            ["k6", "run", "--quiet", "/opt/benchmark/k6.js"],
            environment,
            summary_path,
        )
    raise ValueError(f"unknown benchmark tool: {tool}")


def run_tool(
    tool: str,
    url: str,
    concurrency: int,
    duration: str,
) -> ToolResult:
    with tempfile.TemporaryDirectory(prefix="volt-benchmark-") as directory:
        command, environment, summary_path = tool_command(
            tool, url, concurrency, duration, Path(directory)
        )
        completed = subprocess.run(
            command,
            check=False,
            capture_output=True,
            text=True,
            env=environment,
        )
        if completed.returncode != 0:
            raise RuntimeError(
                f"{tool} exited with {completed.returncode}\n"
                f"stdout:\n{completed.stdout}\nstderr:\n{completed.stderr}"
            )
        output = (
            summary_path.read_text(encoding="utf-8")
            if summary_path is not None
            else completed.stdout
        )
        try:
            return PARSERS[tool](output)
        except ValueError as error:
            raise RuntimeError(
                f"could not parse {tool} output: {error}\n"
                f"stdout:\n{completed.stdout}\nstderr:\n{completed.stderr}"
            ) from error


def reset_target(base_url: str) -> None:
    request_json(f"{base_url}/__admin/reset", method="POST")


def target_count(base_url: str) -> int:
    return int(request_json(f"{base_url}/__admin/count")["requests"])


def validate_request_count(reported: int, observed: int, concurrency: int) -> int:
    in_flight = observed - reported
    if in_flight < 0 or in_flight > concurrency:
        raise RuntimeError(
            f"tool reported {reported} completed requests but the target counted "
            f"{observed}; expected at most {concurrency} requests still in flight"
        )
    return in_flight


def normalize_version(tool: str, output: str, fallback: str) -> str:
    lines = output.strip().splitlines()
    if not lines:
        return fallback
    version = lines[0].strip()
    for prefix in (f"{tool} version ", f"{tool} "):
        if version.lower().startswith(prefix):
            return version[len(prefix) :]
    return version


def tool_versions() -> dict[str, str]:
    commands = {
        "wrk": ["wrk", "--version"],
        "hey": ["hey", "-version"],
        "k6": ["k6", "version"],
    }
    versions = {"volt": os.environ.get("VOLT_COMMIT", "unknown")}
    fallbacks = {
        "wrk": "4.2.0",
        "hey": "0.1.5",
        "k6": "2.0.0",
    }
    for tool, command in commands.items():
        completed = subprocess.run(command, capture_output=True, text=True, check=False)
        output = completed.stdout or completed.stderr
        versions[tool] = (
            normalize_version(tool, output, fallbacks[tool])
            if completed.returncode == 0
            else fallbacks[tool]
        )
    return versions


def run_comparison() -> tuple[list[ScenarioResult], dict[str, str | int]]:
    target_url = os.environ.get("BENCH_TARGET_URL", "http://target:8080").rstrip("/")
    duration = os.environ.get("BENCH_DURATION", "10s")
    warmup_duration = os.environ.get("BENCH_WARMUP_DURATION", "2s")
    runs = env_int("BENCH_RUNS", 5)
    concurrency_values = parse_concurrency(
        os.environ.get("BENCH_CONCURRENCY", "10,50,100")
    )
    seed = env_int("BENCH_SEED", int(date.today().strftime("%Y%m%d")))
    cooldown = float(os.environ.get("BENCH_COOLDOWN", "1"))
    if cooldown < 0:
        raise ValueError("BENCH_COOLDOWN must not be negative")

    wait_for_target(target_url)
    randomizer = random.Random(seed)
    results: list[ScenarioResult] = []

    for workload, path in WORKLOADS:
        url = target_url + path
        for concurrency in concurrency_values:
            samples = {tool: [] for tool in TOOLS}
            log(f"Warming {workload} at concurrency {concurrency}")
            for tool in TOOLS:
                run_tool(tool, url, concurrency, warmup_duration)

            for repetition in range(1, runs + 1):
                order = list(TOOLS)
                randomizer.shuffle(order)
                log(
                    f"Run {repetition}/{runs}: {workload}, concurrency "
                    f"{concurrency}, order {', '.join(order)}"
                )
                for tool in order:
                    reset_target(target_url)
                    result = run_tool(tool, url, concurrency, duration)
                    observed = target_count(target_url)
                    in_flight = validate_request_count(
                        result.requests, observed, concurrency
                    )
                    if in_flight:
                        log(
                            f"{tool}: target accepted {in_flight} request(s) still "
                            "in flight at the duration boundary"
                        )
                    samples[tool].append(result.requests_per_second)
                    if cooldown:
                        time.sleep(cooldown)

            results.append(
                ScenarioResult(
                    workload=workload,
                    concurrency=concurrency,
                    medians={
                        tool: statistics.median(values)
                        for tool, values in samples.items()
                    },
                )
            )

    metadata: dict[str, str | int] = {
        "date": date.today().isoformat(),
        "duration": duration,
        "runs": runs,
        "seed": seed,
        "host_os": os.environ.get(
            "BENCH_HOST_OS", f"{platform.system()} {platform.machine()}"
        ),
        "host_cpu": os.environ.get(
            "BENCH_HOST_CPU", platform.processor() or "unknown"
        ),
        "docker": os.environ.get("BENCH_DOCKER_VERSION", "unknown"),
    }
    metadata.update({f"{tool}_version": value for tool, value in tool_versions().items()})
    return results, metadata


def markdown(results: list[ScenarioResult], metadata: dict[str, str | int]) -> str:
    def clean(value: object) -> str:
        return str(value).replace("|", "/").replace("\n", " ")

    lines = [
        "<!-- BEGIN GENERATED BENCHMARK COMPARISON -->",
        "### Throughput comparison",
        "",
        (
            f"Generated {clean(metadata['date'])} from Volt "
            f"`{clean(metadata['volt_version'])}` using "
            f"{clean(metadata['runs'])} × {clean(metadata['duration'])} measured "
            f"runs per scenario and shuffle seed {clean(metadata['seed'])}. "
            "Values are median requests/second."
        ),
        "",
        (
            f"Environment: {clean(metadata['host_cpu'])}; "
            f"{clean(metadata['host_os'])}; Docker {clean(metadata['docker'])}."
        ),
        "",
        (
            f"Tools: wrk {clean(metadata['wrk_version'])}; "
            f"hey {clean(metadata['hey_version'])}; "
            f"k6 {clean(metadata['k6_version'])}."
        ),
        "",
        "| Workload | Concurrency | Volt | wrk | hey | k6 |",
        "|---|---:|---:|---:|---:|---:|",
    ]
    for result in results:
        lines.append(
            f"| {clean(result.workload)} | {result.concurrency:,} | "
            f"{result.medians['volt']:,.0f} | {result.medians['wrk']:,.0f} | "
            f"{result.medians['hey']:,.0f} | {result.medians['k6']:,.0f} |"
        )
    lines.extend(
        [
            "",
            (
                "All tools used HTTP/1.1 keep-alive and consumed the complete "
                "response. Their concurrency models differ: Volt uses workers, "
                "wrk uses connections across threads, hey uses workers, and k6 "
                "uses virtual users. Server-side counts are reconciled with up "
                "to one duration-boundary request per concurrent worker in flight."
            ),
            "<!-- END GENERATED BENCHMARK COMPARISON -->",
        ]
    )
    return "\n".join(lines) + "\n"


def main() -> int:
    try:
        results, metadata = run_comparison()
        sys.stdout.write(markdown(results, metadata))
        return 0
    except (OSError, RuntimeError, ValueError) as error:
        log(f"benchmark comparison failed: {error}")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
