#!/usr/bin/env python3
"""
Lightweight Python benchmarks for the published `poml` package.
Runs a small suite against the same benchDoc used in Go benches and emits JSON.
"""

from __future__ import annotations

import json
import os
import sys
import time
import traceback
from statistics import mean

try:
    import poml  # type: ignore
    from poml import api as poml_api  # type: ignore
except Exception as exc:  # pragma: no cover - best-effort import guard
    out_path = sys.argv[1] if len(sys.argv) > 1 else "py_bench.json"
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump({"error": f"import poml failed: {exc}"}, f)
    sys.exit(0)

DEFAULT_ITERATIONS = int(os.getenv("PY_BENCH_ITERATIONS", "50"))
CLI_ITERATIONS = int(os.getenv("PY_BENCH_CLI_ITERATIONS", "20"))

BENCH_DOC = """<poml>
  <meta><id>bench.demo</id><version>1.0</version><owner>tester</owner></meta>
  <role>Bench role</role>
  <task>Do a thing</task>
  <system-msg>System hello</system-msg>
  <human-msg>Hello</human-msg>
</poml>"""


def measure(name: str, fn, iterations: int = DEFAULT_ITERATIONS) -> dict:
    start = time.perf_counter_ns()
    errors = []
    for _ in range(iterations):
        try:
            fn()
        except Exception as exc:  # pragma: no cover - benchmark resilience
            errors.append(str(exc))
            break
    elapsed_ns = time.perf_counter_ns() - start
    ns_per_op = elapsed_ns / max(1, (iterations - len(errors)))
    result = {"name": name, "ns_per_op": ns_per_op}
    if errors:
        result["error"] = errors[0]
    return result


def main() -> None:
    out_path = sys.argv[1] if len(sys.argv) > 1 else "py_bench.json"
    results = []

    cases = [
        ("parse_message_dict", lambda: poml_api.poml(BENCH_DOC, format="message_dict")),
        ("parse_dict", lambda: poml_api.poml(BENCH_DOC, format="dict")),
        ("parse_openai_chat", lambda: poml_api.poml(BENCH_DOC, format="openai_chat")),
        ("parse_langchain", lambda: poml_api.poml(BENCH_DOC, format="langchain")),
    ]

    for name, fn in cases:
        results.append(measure(name, fn))

    # Attempt diagram→scene if available
    try:
        import tempfile
        import subprocess
        with tempfile.NamedTemporaryFile("w", suffix=".poml", delete=False) as tmp:
            tmp.write(BENCH_DOC)
            tmp_path = tmp.name
        # Best-effort: call CLI for openai_chat to simulate convert path
        def cli_call() -> None:
            subprocess.run(["python3", "-m", "poml.cli", "-f", tmp_path, "-o", "-", "--format", "dict"], check=True, capture_output=True)
        results.append(measure("cli_dict", cli_call, iterations=CLI_ITERATIONS))
    except Exception as exc:  # pragma: no cover
        results.append({"name": "cli_dict", "error": f"{exc}"})

    payload = {
        "package": "poml (PyPI)",
        "iterations": {"default": DEFAULT_ITERATIONS, "cli": CLI_ITERATIONS},
        "benchmarks": results,
    }
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(payload, f, indent=2)


if __name__ == "__main__":
    try:
        main()
    except Exception:
        out_path = sys.argv[1] if len(sys.argv) > 1 else "py_bench.json"
        with open(out_path, "w", encoding="utf-8") as f:
            json.dump({"error": traceback.format_exc()}, f, indent=2)
