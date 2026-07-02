#!/usr/bin/env bash
# Generate and TLC-model-check the whole formal suite for a design:
#   - every machine's control-flow model (safety + liveness + deadlock-freedom)
#   - the Deal data refinement (the real domain invariants)
#   - the Deal refinement mapping (subsystem refines its abstract contract)
# Usage: verify_formal.sh <design-dir>
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
design="${1:?usage: verify_formal.sh <design-dir>}"
mdir="$design/machines"
fdir="$design/formal"
mkdir -p "$fdir"

for mj in "$mdir"/*.machine.json; do
  python3 "$here/tla_gen.py" "$mj" "$fdir" >/dev/null
done

pass=0
fail=0
run() { # label  spec.tla
  if bash "$here/tlc.sh" "$fdir/$2" 2>&1 | grep -q "No error has been found"; then
    printf "  PASS  %s\n" "$1"; pass=$((pass + 1))
  else
    printf "  FAIL  %s\n" "$1"; fail=$((fail + 1))
  fi
}

echo "control-flow (safety: retry bounded; liveness: overlay resolves; deadlock-free):"
for m in Deal Task User Session CommandExecution; do run "$m" "$m.tla"; done
echo "data-refined domain invariants:"
run "DealData: stage-forward, persist atomicity, won-has-closedate" "DealData.tla"
echo "refinement (composition substrate for the recursion):"
run "DealData refines DealContract" "DealRefinement.tla"

echo ""
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
