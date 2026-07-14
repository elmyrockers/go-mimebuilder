#!/usr/bin/env bash
#
# run_benchmarks.sh
# Runs the full hot-path benchmark suite and the 1M-iteration stress test
# for go-mimebuilder. Output goes straight to the terminal only.

set -euo pipefail

BENCH_COUNT="${BENCH_COUNT:-5}"     # number of times to run -bench for stability
STRESS_RUNS="${STRESS_RUNS:-3}"     # number of times to repeat the stress test

echo "=========================================="
echo " go-mimebuilder :: Benchmark & Stress Suite"
echo "=========================================="
echo "Go version  : $(go version)"
echo "OS/Arch     : $(go env GOOS)/$(go env GOARCH)"
echo "Bench count : $BENCH_COUNT"
echo "Stress runs : $STRESS_RUNS"
echo

# ---- Step 1: Hot-path benchmarks (all Benchmark* funcs, no allocs) -----
echo "--- Running hot-path benchmarks (x${BENCH_COUNT}) ---"
echo

go test -tags benchmark \
  -bench=".*" \
  -benchmem \
  -run=^\$ \
  -count="$BENCH_COUNT" \
  -cpu=1,2,4 \
  .

echo
echo "--- Hot-path benchmarks complete ---"
echo

# ---- Step 2: Stress test (1M iterations, repeated for stability) -------
echo "--- Running stress test (x${STRESS_RUNS}) ---"
echo

for i in $(seq 1 "$STRESS_RUNS"); do
  echo "> Run #$i"
  go test -tags benchmark \
    -run=TestStressMillion \
    -count=1 \
    -v \
    .
  echo
done

echo "--- Stress test complete ---"
echo "=========================================="
echo " Done."
echo "=========================================="