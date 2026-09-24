#!/usr/bin/env bash
# The datalog-parity lane: run the Soufflé parity tests of internal/datalog
# (the evaluator's program corpus) and internal/gates (every shipped rule file
# over every bundled example), and fail if Soufflé was not actually used.
#
# It runs inside the image built from scripts/souffle.dockerfile, where
# souffle is on PATH. The Dagger DatalogParity job and the hosted
# datalog-parity job both call this script; to reproduce locally:
#
#   docker build --platform linux/amd64 -f scripts/souffle.dockerfile -t machinery-souffle scripts
#   docker run --rm --platform linux/amd64 -v "$PWD:/src:ro" -w /src machinery-souffle scripts/datalog-parity.sh
set -euo pipefail

command -v souffle >/dev/null || { echo "datalog-parity needs souffle on PATH" >&2; exit 1; }
souffle --version | sed -n 's/^Version: /souffle /p'
go version

log=$(mktemp)
trap 'rm -f "$log"' EXIT
go test -count=1 -v ./internal/datalog/ ./internal/gates/ -run Parity >"$log" 2>&1 || { cat "$log"; exit 1; }
if grep -q 'souffle not on PATH' "$log"; then
  cat "$log"
  echo "a parity test skipped its Soufflé half" >&2
  exit 1
fi
grep -E '^(ok|--- PASS: TestParity|--- PASS: TestRulesParity)|rules parity:' "$log"
