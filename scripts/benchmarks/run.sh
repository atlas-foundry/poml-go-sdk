#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${ROOT}/benchmarks/results"
mkdir -p "${OUT_DIR}"

GO_BENCH_FLAGS=${GO_BENCH_FLAGS:-"-bench=Benchmark -run=^$ -benchmem -count=1"}
echo "[go] running benchmarks with flags: ${GO_BENCH_FLAGS}"
pushd "${ROOT}" >/dev/null
/usr/bin/env go test ${GO_BENCH_FLAGS} -json ./poml > "${OUT_DIR}/go_bench.json"
popd >/dev/null

# Python benchmark runner (uses published poml package).
echo "[python] running python_bench.py"
python3 "${ROOT}/scripts/benchmarks/python_bench.py" "${OUT_DIR}/py_bench.json" || true

# TypeScript benchmark runner (uses published poml package if available).
TS_DIR="${ROOT}/benchmarks/tmp_ts"
mkdir -p "${TS_DIR}"
pushd "${TS_DIR}" >/dev/null
if [[ ! -f package.json ]]; then
  npm init -y >/dev/null 2>&1 || true
fi
# Attempt install; tolerate failures to keep run going.
npm install poml --no-save >/dev/null 2>&1 || npm install @microsoft/poml --no-save >/dev/null 2>&1 || true
echo "[ts] running ts_bench.js"
node "${ROOT}/scripts/benchmarks/ts_bench.js" "${OUT_DIR}/ts_bench.json" || true
popd >/dev/null

echo "[done] results stored in ${OUT_DIR}"
