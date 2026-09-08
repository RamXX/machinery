#!/usr/bin/env bash
# Gate every registered example with its exact capabilities.
#
# This is the single owner of the accepted example-gate policy. The local
# preflight (phase 11), the CI gates job and the Makefile check target all
# call this script, so the three mirrors cannot drift apart again.
#
# Two truthful policies mirror the accepted golden corpus and the MAC-hgz1 v2
# attestation migration:
#
#   1. An example with a registered implementation (go-crm) carries an
#      accepted current implementation review, so it must stay strictly green
#      under --warnings-as-errors, with --impl and --complete as its inventory
#      row declares.
#   2. A design-only example is honestly plan-only: its behavioral
#      current-class claims are attested as plans, so a plain check must pass
#      with exactly the expected missing-current warnings, never promoted. The
#      expected warning set is derived from that design's own committed
#      attestations.yaml, never hardcoded here, and it is diffed rather than
#      grepped: a claim attested current expects no warning, and any other
#      warning, error or drift fails the gate.
#
# Usage: scripts/example-gates.sh [machinery-binary] [inventory-manifest]
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
cd "$repo_root"

machinery=${1:-.bin/machinery}
manifest=${2:-examples/inventory.tsv}
[[ -x "$machinery" ]] || { echo "example gates need an executable machinery binary: $machinery" >&2; exit 1; }

work=$(mktemp -d)
cleanup() { rm -rf -- "$work"; }
trap cleanup EXIT

rows=$work/rows
check_out=$work/check.out
expected=$work/expected.warns
actual=$work/actual.warns

scripts/example-inventory.sh rows "$manifest" >"$rows"
[[ -s "$rows" ]] || { echo "example inventory produced no rows" >&2; exit 1; }

gated=0
while IFS=$'\t' read -r -a row; do
  design=${row[0]}
  impl=${row[1]}
  complete=${row[5]}
  gated=$((gated + 1))
  if [[ "$impl" != - ]]; then
    args=("$design" --warnings-as-errors --impl "$impl")
    [[ "$complete" == no ]] || args+=(--complete)
    if ! "$machinery" check "${args[@]}"; then
      echo "gate suite: $design" >&2
      exit 1
    fi
    continue
  fi
  # Derive the exact plan-only warning set from the design's own attestations.
  awk '
    BEGIN {
      current["gt.conformance-test-shape"] = 1
      current["g4.standin-coverage"] = 1
      current["g4.pack-event-discipline"] = 1
    }
    function emit() {
      if (claim != "" && current[claim] && kind != "current")
        printf "  warn   %s: plan only; current implementation review missing\n", claim
    }
    /^  - claim:/ { emit(); claim = $3; kind = ""; next }
    /^    kind:/ { kind = $2 }
    END { emit() }
  ' "$design/attestations.yaml" >"$expected"
  if ! "$machinery" check "$design" >"$check_out"; then
    cat "$check_out" >&2
    echo "gate suite: $design" >&2
    exit 1
  fi
  { grep '^  warn ' "$check_out" >"$actual" || true; }
  if ! diff -u "$expected" "$actual"; then
    cat "$check_out" >&2
    echo "gate suite warnings drifted from the plan-only expectation: $design" >&2
    exit 1
  fi
done <"$rows"

[[ "$gated" -gt 0 ]] || { echo "example gates ran no example" >&2; exit 1; }
echo "example gates: $gated registered example design suites passed"
