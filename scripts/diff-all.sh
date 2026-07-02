#!/usr/bin/env bash
# scripts/diff-all.sh
# Drives diff-tool.sh across the whole corpus for every subcommand.
# Prints a per-case PASS/FAIL scoreboard; nonzero exit if any FAIL.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$ROOT/.bin"
go build -o "$ROOT/.bin/machinery" "$ROOT/cmd/machinery" || { echo "go build failed"; exit 99; }

pass=0; fail=0
run() {  # run <label> <cmd...>
  "$@" >/dev/null 2>&1 && { echo "  PASS  $1"; pass=$((pass+1)); } || { echo "  FAIL  $1"; fail=$((fail+1)); }
}

for ex in go-crm fulfillment portfolio-engine; do
  run "lint-$ex" "$ROOT/scripts/diff-tool.sh" lint "$ROOT/examples/$ex/design/machines"
  run "oracle-$ex" "$ROOT/scripts/diff-tool.sh" oracle "$ROOT/examples/$ex/design/machines"
done
echo ""
echo "$pass passed, $fail failed"
exit $([ "$fail" = 0 ] && echo 0 || echo 1)
