#!/usr/bin/env bash
# Regenerate and TLC-model-check the whole formal suite for a design, from source.
# Generic: it discovers machines, semantics annotations, and composition annotations,
# regenerates every model, and checks every spec that has a config.
#   *.machine.json       -> tla_gen      control-flow (safety + liveness + deadlock)
#   *.semantics.yaml     -> refine_gen   data invariants + contract + refinement
#   *.composition.yaml   -> compose_gen  cross-aggregate invariants
#   hand-authored *.tla with a *.cfg     also checked (e.g. System)
# Usage: verify_formal.sh <design-dir>
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
design="${1:?usage: verify_formal.sh <design-dir>}"
mdir="$design/machines"
fdir="$design/formal"
mkdir -p "$fdir"

for mj in "$mdir"/*.machine.json; do
  [ -e "$mj" ] && python3 "$here/tla_gen.py" "$mj" "$fdir" >/dev/null
done
for sem in "$fdir"/*.semantics.yaml; do
  [ -e "$sem" ] || continue
  m="$(basename "$sem" .semantics.yaml)"
  python3 "$here/refine_gen.py" "$mdir/$m.machine.json" "$sem" "$fdir" >/dev/null
done
for comp in "$fdir"/*.composition.yaml; do
  [ -e "$comp" ] && python3 "$here/compose_gen.py" "$comp" "$fdir" >/dev/null
done

pass=0
fail=0
for tla in "$fdir"/*.tla; do
  base="${tla%.tla}"
  [ -f "$base.cfg" ] || continue
  name="$(basename "$base")"
  if bash "$here/tlc.sh" "$tla" 2>&1 | grep -q "No error has been found"; then
    printf "  PASS  %s\n" "$name"; pass=$((pass + 1))
  else
    printf "  FAIL  %s\n" "$name"; fail=$((fail + 1))
  fi
done

echo ""
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
