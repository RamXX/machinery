#!/usr/bin/env bash
# scripts/diff-tool.sh <subcommand> <args...>
# Runs python3 tools/<py-tool> <args> and machinery <sub> <args>, captures
# stdout/stderr/exit/emitted-files, diffs them. Normalizes only absolute paths.
# Nonzero exit on any difference. The live scoreboard during migration.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TOOLS="$ROOT/skills/machinery/tools"
MACH="$ROOT/bin/machinery"   # built by the harness; falls back to go run
[ -x "$MACH" ] || MACH="$ROOT/.bin/machinery"
[ -x "$MACH" ] || { echo "build machinery first: go build -o .bin/machinery ./cmd/machinery"; exit 99; }

sub="$1"; shift
# map subcommand -> python tool
declare -A PYMAP=( [lint]=machine_lint.py [oracle]=oracle_gen.py [tla]=tla_gen.py [refine]=refine_gen.py [compose]=compose_gen.py [check]=machinery_check.py [ir-dump]=ir_dump.py )
py="${PYMAP[$sub]:-}"

tmp="$(mktemp -d)"
python3 "$TOOLS/$py" "$@" >"$tmp/py.out" 2>"$tmp/py.err"; pyrc=$?
"$MACH" "$sub" "$@" >"$tmp/go.out" 2>"$tmp/go.err"; gorc=$?

# Normalize absolute paths (repo root) -> relative, in both outputs.
for f in "$tmp"/py.out "$tmp"/py.err "$tmp"/go.out "$tmp"/go.err; do
  [ -e "$f" ] || continue
  sed -i.tmp "s#$ROOT/#./#g" "$f" && rm -f "$f.tmp"
done

status=0
if [ "$pyrc" != "$gorc" ]; then echo "EXIT MISMATCH ($sub $*): py=$pyrc go=$gorc"; status=1; fi
if ! diff -q "$tmp/py.out" "$tmp/go.out" >/dev/null 2>&1; then echo "STDOUT MISMATCH ($sub $*)"; diff -u "$tmp/py.out" "$tmp/go.out"; status=1; fi
if ! diff -q "$tmp/py.err" "$tmp/go.err" >/dev/null 2>&1; then echo "STDERR MISMATCH ($sub $*)"; diff -u "$tmp/py.err" "$tmp/go.err"; status=1; fi

rm -rf "$tmp"
[ "$status" = 0 ] && echo "PASS  $sub $*" || true
exit $status
