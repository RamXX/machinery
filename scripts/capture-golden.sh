#!/usr/bin/env bash
# scripts/capture-golden.sh
# Run every Python tool over the whole corpus; write stdout+stderr+exit-code+
# emitted-files to testdata/golden/<case>/. Deterministic by construction.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TOOLS="$ROOT/skills/machinery/tools"
GOLDEN="$ROOT/testdata/golden"

rm -rf "$GOLDEN"; mkdir -p "$GOLDEN"

cap() {  # cap <case> <args...>  (runs python3 TOOLS/<first-arg-after-case>)
  local case="$1"; shift
  local out="$GOLDEN/$case"; mkdir -p "$out"
  python3 "$TOOLS/$1" "${@:2}" >"$out/stdout.txt" 2>"$out/stderr.txt" && echo 0 >"$out/exitcode.txt" \
    || echo $? >"$out/exitcode.txt"
}

# --- lint ---
for ex in go-crm fulfillment portfolio-engine; do
  cap "lint-$ex" machine_lint.py "$ROOT/examples/$ex/design/machines"
done
# --- oracle (scratch dir, keep emitted) ---
for ex in go-crm fulfillment portfolio-engine; do
  d=$(mktemp -d); cp "$ROOT"/examples/$ex/design/machines/*.machine.json "$d/"
  mkdir -p "$GOLDEN/oracle-$ex"
  python3 "$TOOLS/oracle_gen.py" "$d" >"$GOLDEN/oracle-$ex/stdout.txt" 2>&1 || true
  cp "$d"/*.oracle.md "$GOLDEN/oracle-$ex/" 2>/dev/null || true
  rm -rf "$d"
done
# --- check ---
cap "check-go-crm" machinery_check.py "$ROOT/examples/go-crm/design" --impl "$ROOT/examples/go-crm/impl"
cap "check-fulfillment" machinery_check.py "$ROOT/examples/fulfillment/design"
cap "check-portfolio" machinery_check.py "$ROOT/examples/portfolio-engine/design"
# --- tla + refine + compose (scratch formal dir) ---
for ex in go-crm fulfillment portfolio-engine; do
  d="$ROOT/examples/$ex/design"; scratch=$(mktemp -d); mkdir -p "$GOLDEN/gen-$ex"
  for mj in "$d"/machines/*.machine.json; do python3 "$TOOLS/tla_gen.py" "$mj" "$scratch" >/dev/null 2>&1 || true; done
  for sem in "$d"/formal/*.semantics.yaml; do
    [ -e "$sem" ] || continue; m=$(basename "$sem" .semantics.yaml)
    python3 "$TOOLS/refine_gen.py" "$d/machines/$m.machine.json" "$sem" "$scratch" >/dev/null 2>&1 || true
  done
  for comp in "$d"/formal/*.composition.yaml; do
    [ -e "$comp" ] || continue
    coord=$(python3 -c "import yaml; print(yaml.safe_load(open('$comp'))['coordinator'])")
    python3 "$TOOLS/compose_gen.py" "$comp" "$d/machines/$coord.machine.json" "$scratch" >/dev/null 2>&1 || true
  done
  cp "$scratch"/*.tla "$scratch"/*.cfg "$GOLDEN/gen-$ex/" 2>/dev/null || true
  rm -rf "$scratch"
done

echo "golden captured under $GOLDEN"
