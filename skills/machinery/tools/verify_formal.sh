#!/usr/bin/env bash
# Regenerate and TLC-model-check the whole formal suite for a design, from source.
# Generic: it discovers machines, semantics annotations, and composition annotations,
# regenerates every model, and checks every spec that has a config.
#
# SOURCE (hand-authored, version these):  design/machines/*.machine.json and, in
#   design/formal/, the *.semantics.yaml / *.composition.yaml annotations and any
#   hand-authored *.tla+*.cfg (e.g. System). GENERATED (regenerated every run,
#   safe to gitignore): the other design/formal/*.tla and *.cfg. Source and
#   generated share design/formal/ because TLC requires EXTENDS/INSTANCE'd modules
#   to sit next to the spec that references them.
#
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
  [ -e "$comp" ] || continue
  coord="$(python3 -c "import yaml; print(yaml.safe_load(open('$comp'))['coordinator'])")"
  python3 "$here/compose_gen.py" "$comp" "$mdir/$coord.machine.json" "$fdir" >/dev/null
done

pass=0
fail=0
for tla in "$fdir"/*.tla; do
  base="${tla%.tla}"
  [ -f "$base.cfg" ] || continue
  name="$(basename "$base")"
  # Pass requires BOTH a zero TLC exit code and the no-error line, so a Java
  # crash, a download failure, or a TLC message change can never read as PASS.
  if out="$(bash "$here/tlc.sh" "$tla" 2>&1)" && grep -q "No error has been found" <<<"$out"; then
    printf "  PASS  %s\n" "$name"; pass=$((pass + 1))
  else
    printf "  FAIL  %s\n" "$name"; fail=$((fail + 1))
    printf '%s\n' "$out" | tail -n 40 | sed 's/^/        /'
  fi
done

echo ""
echo "$pass passed, $fail failed"
[ "$fail" -eq 0 ]
