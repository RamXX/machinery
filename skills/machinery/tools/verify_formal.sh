#!/usr/bin/env bash
# Regenerate and TLC-model-check the whole formal suite for a design, from source.
# Generic: it discovers machines, semantics annotations, and composition annotations,
# regenerates every model, and checks every spec that has a config.
#
# SOURCE (hand-authored, version these):  design/machines/*.machine.json and, in
#   design/formal/, the *.semantics.yaml / *.composition.yaml annotations and any
#   hand-authored *.tla+*.cfg (e.g. System). GENERATED (regenerated every run,
#   committed; the nightly regen job asserts the committed copies match): the
#   other design/formal/*.tla and *.cfg. Source and generated share
#   design/formal/ because TLC requires EXTENDS/INSTANCE'd modules to sit next
#   to the spec that references them.
#
#   *.machine.json       -> machinery tla      control-flow (safety + liveness + deadlock)
#   *.semantics.yaml     -> machinery refine   data invariants + contract + refinement
#   *.composition.yaml   -> machinery compose  cross-aggregate invariants
#   hand-authored *.tla with a *.cfg           also checked (e.g. System)
# Usage: verify_formal.sh <design-dir>
# Needs the machinery binary on PATH (see install.sh); override with MACHINERY=/path/to/machinery.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
machinery="${MACHINERY:-machinery}"
command -v "$machinery" >/dev/null 2>&1 || {
  echo "machinery binary not found on PATH (see install.sh), or set MACHINERY=/path/to/machinery" >&2
  exit 1
}
design="${1:?usage: verify_formal.sh <design-dir>}"
mdir="$design/machines"
fdir="$design/formal"
mkdir -p "$fdir"

for mj in "$mdir"/*.machine.json; do
  [ -e "$mj" ] && "$machinery" tla "$mj" "$fdir" >/dev/null
done
for sem in "$fdir"/*.semantics.yaml; do
  [ -e "$sem" ] || continue
  m="$(basename "$sem" .semantics.yaml)"
  "$machinery" refine "$mdir/$m.machine.json" "$sem" "$fdir" >/dev/null
done
for comp in "$fdir"/*.composition.yaml; do
  [ -e "$comp" ] || continue
  # coordinator: <Name> is a top-level scalar; strip any trailing comment.
  coord="$(sed -n 's/^coordinator:[[:space:]]*//p' "$comp" | head -n 1 | sed 's/[[:space:]]*#.*$//;s/[[:space:]]*$//')"
  "$machinery" compose "$comp" "$mdir/$coord.machine.json" "$fdir" >/dev/null
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
