#!/usr/bin/env bash
# Validate and query the closed universe of shipped example designs.
set -euo pipefail

action=${1:-rows}
manifest=${2:-examples/inventory.tsv}
[[ ! -L "$manifest" && -f "$manifest" ]] || { echo "example inventory $manifest must be a regular non-symlink file" >&2; exit 1; }

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)

work=$(mktemp -d)
registered=$work/registered
discovered=$work/discovered
rows=$work/rows
pack_registered=$work/pack-registered
pack_discovered=$work/pack-discovered
pack_relations=$work/pack-relations
design_entries=$work/design-entries
all_entries=$work/all-entries
tree_inventory=$work/tree-inventory
cleanup() { rm -rf -- "$work"; }
trap cleanup EXIT

if ! (cd "$repo_root" && go build -o "$tree_inventory" ./scripts/tree-inventory) 2>"$work/build.stderr"; then
  echo "build bounded tree inventory helper failed" >&2
  cat "$work/build.stderr" >&2
  exit 1
fi
if [[ -s "$work/build.stderr" ]]; then
  echo "building bounded tree inventory helper emitted stderr; warnings are forbidden" >&2
  cat "$work/build.stderr" >&2
  exit 1
fi

real_path_components() {
  local path=$1 current='' component
  local -a components
  IFS=/ read -r -a components <<<"$path"
  for component in "${components[@]}"; do
    [[ -n "$component" && "$component" != "." ]] || continue
    current=${current:+$current/}$component
    [[ ! -L "$current" ]] || return 1
  done
}

regular_file() {
  [[ -f "$1" && ! -L "$1" ]] && real_path_components "$1"
}

real_directory() {
  [[ -d "$1" && ! -L "$1" ]] && real_path_components "$1"
}

while IFS=$'\t' read -r design impl formal checker c4 complete pack_role impl_module security extra; do
  [[ -z "$design" || ${design:0:1} == "#" ]] && continue
  [[ -z "${extra:-}" ]] || { echo "example inventory row for $design has extra columns" >&2; exit 1; }
  [[ "$design" =~ ^examples/[^/]+(/[^/]+)?/design$ ]] || { echo "invalid example design path: $design" >&2; exit 1; }
  if [[ "$design" == *".."* ]] || ! real_directory "$design" || ! regular_file "$design/ARCHITECTURE.md"; then
    echo "registered example design is missing, symlinked, or invalid: $design" >&2; exit 1
  fi
  if [[ "$impl" != "-" ]] && { [[ "$impl" == *".."* ]] || ! real_directory "$impl"; }; then
    echo "registered impl is missing, symlinked, or invalid for $design: $impl" >&2; exit 1
  fi
  [[ "$formal" == yes || "$formal" == no ]] || { echo "formal capability for $design must be yes/no" >&2; exit 1; }
  [[ "$c4" == yes || "$c4" == no ]] || { echo "C4 capability for $design must be yes/no" >&2; exit 1; }
  [[ "$complete" == yes || "$complete" == no ]] || { echo "complete capability for $design must be yes/no" >&2; exit 1; }
  [[ "$security" == yes || "$security" == no ]] || { echo "security capability for $design must be yes/no" >&2; exit 1; }
  [[ "$complete" == no || "$impl" != - ]] || { echo "complete example must declare an implementation: $design" >&2; exit 1; }
  if [[ "$formal" == yes ]]; then
    compgen -G "$design/formal/*.cfg" >/dev/null || { echo "formal example has no committed cfg proof: $design" >&2; exit 1; }
  elif compgen -G "$design/formal/*.cfg" >/dev/null; then
    echo "example has formal proofs but is not registered formal=yes: $design" >&2; exit 1
  fi
  if [[ "$c4" == yes ]]; then
    regular_file "$design/workspace.dsl" || { echo "C4 example lacks a regular non-symlink workspace.dsl: $design" >&2; exit 1; }
  elif [[ -e "$design/workspace.dsl" || -L "$design/workspace.dsl" ]]; then
    echo "example has workspace.dsl but is not registered c4=yes: $design" >&2; exit 1
  fi
  if [[ "$checker" != "-" ]]; then
    if [[ "$checker" == *".."* ]] || ! regular_file "$checker" || ! real_directory "$design/checkers"; then
      echo "checker capability is invalid for $design: $checker" >&2; exit 1
    fi
  elif [[ -e "$design/checkers" || -L "$design/checkers" ]]; then
    echo "example has checkers but no checker registry capability: $design" >&2; exit 1
  fi
  case "$pack_role" in
    -)
      [[ ! -e "$design/packmap.yaml" && ! -L "$design/packmap.yaml" && ! -e "$design/packs" && ! -L "$design/packs" ]] || { echo "example has pack decomposition artifacts but no pack role: $design" >&2; exit 1; }
      ;;
    parent)
      compgen -G "$design/packs/*.pack/pack.yaml" >/dev/null || { echo "pack parent has no generated child pack: $design" >&2; exit 1; }
      ;;
    child:*)
      pack_parent=${pack_role#child:}
      if [[ ! "$pack_parent" =~ ^examples/[^/]+(/[^/]+)?/design$ ]] || ! real_directory "$pack_parent"; then
        echo "pack child has invalid parent for $design: $pack_parent" >&2; exit 1
      fi
      if ! regular_file "$design/pack/pack.yaml" || ! regular_file "$design/packmap.yaml"; then
        echo "pack child lacks regular non-symlink pack/pack.yaml or packmap.yaml: $design" >&2; exit 1
      fi
      subsystem=$(basename "$(dirname "$design")")
      regular_file "$pack_parent/packs/$subsystem.pack/pack.yaml" || { echo "pack parent $pack_parent lacks regular non-symlink frozen pack for child $design" >&2; exit 1; }
      ;;
    *) echo "invalid pack role for $design: $pack_role" >&2; exit 1 ;;
  esac
  case "$impl_module" in
    -) [[ "$impl" == - ]] || { echo "implementation has no module type for $design: $impl" >&2; exit 1; } ;;
    go)
      if [[ "$impl" == - ]] || ! regular_file "$impl/go.mod"; then
        echo "Go implementation module is invalid for $design: $impl" >&2; exit 1
      fi
      ;;
    *) echo "unsupported implementation module type for $design: $impl_module" >&2; exit 1 ;;
  esac
  [[ "$security" == no || "$impl_module" == go ]] || { echo "security scanning requires a supported implementation module for $design" >&2; exit 1; }
  printf '%s\n' "$design" >>"$registered"
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$design" "$impl" "$formal" "$checker" "$c4" "$complete" "$pack_role" "$impl_module" "$security" >>"$rows"
done <"$manifest"

[[ -s "$rows" ]] || { echo "example inventory is unexpectedly empty" >&2; exit 1; }
LC_ALL=C sort -c "$registered" || { echo "example inventory design rows must be byte-sorted and unique" >&2; exit 1; }
for capability in formal c4 impl checker complete pack-parent pack-child security; do
  case "$capability" in
    formal) count=$(awk -F '\t' '$3 == "yes" {n++} END {print n+0}' "$rows") ;;
    c4) count=$(awk -F '\t' '$5 == "yes" {n++} END {print n+0}' "$rows") ;;
    impl) count=$(awk -F '\t' '$2 != "-" {n++} END {print n+0}' "$rows") ;;
    checker) count=$(awk -F '\t' '$4 != "-" {n++} END {print n+0}' "$rows") ;;
    complete) count=$(awk -F '\t' '$6 == "yes" {n++} END {print n+0}' "$rows") ;;
    pack-parent) count=$(awk -F '\t' '$7 == "parent" {n++} END {print n+0}' "$rows") ;;
    pack-child) count=$(awk -F '\t' '$7 ~ /^child:/ {n++} END {print n+0}' "$rows") ;;
    security) count=$(awk -F '\t' '$9 == "yes" {n++} END {print n+0}' "$rows") ;;
  esac
  [[ "$count" -gt 0 ]] || { echo "example inventory unexpectedly has no $capability capability" >&2; exit 1; }
done
if ! "$tree_inventory" -root examples -max-entries 100000 -max-depth 64 -max-bytes 33554432 -timeout 15s >"$all_entries"; then
  echo "bounded example design discovery failed" >&2
  exit 1
fi
while IFS= read -r entry; do
  case "$entry" in
    */design|*/design/*) printf '%s\n' "$entry" >>"$design_entries" ;;
  esac
done <"$all_entries"
while IFS= read -r entry; do
  if [[ -L "$entry" ]]; then
    echo "example design entry must not be a symlink: $entry" >&2
    exit 1
  fi
  if [[ "$entry" == */design ]]; then
    if [[ ! -d "$entry" ]]; then
      echo "example design root must be a real directory: $entry" >&2
      exit 1
    fi
    # Git cannot carry an empty directory tree. Payload entries later in the
    # same exact inventory establish whether this root is shipped.
    continue
  fi
  if [[ -d "$entry" ]]; then
    case "$entry" in
      */design/pack|*/design/packs)
        design_root=${entry%%/design/*}/design
        printf '%s\n' "$design_root" >>"$discovered"
        printf '%s\n' "$design_root" >>"$pack_discovered"
        ;;
    esac
    continue
  fi
  if [[ ! -f "$entry" ]]; then
    echo "example design entry must be regular or a real directory: $entry" >&2
    exit 1
  fi
  design_root=${entry%%/design/*}/design
  printf '%s\n' "$design_root" >>"$discovered"
  case "$entry" in
    */design/packmap.yaml|*/design/pack|*/design/pack/*|*/design/packs|*/design/packs/*)
      printf '%s\n' "$design_root" >>"$pack_discovered"
      ;;
  esac
done <"$design_entries"
LC_ALL=C sort -u -o "$discovered" "$discovered"
if ! cmp -s "$registered" "$discovered"; then
  echo "example inventory does not exactly match discovered examples/**/design roots" >&2
  diff -u "$registered" "$discovered" >&2 || true
  exit 1
fi
awk -F '\t' '$7 == "parent" || $7 ~ /^child:/ {print $1}' "$rows" >"$pack_registered"
LC_ALL=C sort -u -o "$pack_discovered" "$pack_discovered"
LC_ALL=C sort -o "$pack_registered" "$pack_registered"
if ! cmp -s "$pack_registered" "$pack_discovered"; then
  echo "example inventory pack roles do not exactly match decomposition artifacts" >&2
  diff -u "$pack_registered" "$pack_discovered" >&2 || true
  exit 1
fi
awk -F '\t' '$7 ~ /^child:/ {sub(/^child:/, "", $7); print $1 "\t" $7}' "$rows" >"$pack_relations"
while IFS=$'\t' read -r child parent; do
  if ! awk -F '\t' -v parent="$parent" '$1 == parent && $7 == "parent" {found=1} END {exit found ? 0 : 1}' "$rows"; then
    echo "pack child $child references an undeclared pack parent: $parent" >&2
    exit 1
  fi
done <"$pack_relations"

case "$action" in
  rows) cat "$rows" ;;
  designs) cut -f1 "$rows" ;;
  formal) awk -F '\t' '$3 == "yes" {print $1}' "$rows" ;;
  c4) awk -F '\t' '$5 == "yes" {print $1 "/workspace.dsl"}' "$rows" ;;
  impls) awk -F '\t' '$2 != "-" {print $1 "\t" $2}' "$rows" ;;
  impl-modules) awk -F '\t' '$8 != "-" {print $1 "\t" $2 "\t" $8}' "$rows" ;;
  checkers) awk -F '\t' '$4 != "-" {print $1 "\t" $4}' "$rows" ;;
  pack-parents) awk -F '\t' '$7 == "parent" {print $1}' "$rows" ;;
  pack-children) awk -F '\t' '$7 ~ /^child:/ {sub(/^child:/, "", $7); print $1 "\t" $7}' "$rows" ;;
  security) awk -F '\t' '$9 == "yes" {print $2 "\t" $8}' "$rows" ;;
  *) echo "usage: $0 {rows|designs|formal|c4|impls|impl-modules|checkers|pack-parents|pack-children|security} [manifest]" >&2; exit 2 ;;
esac
