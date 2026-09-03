#!/usr/bin/env bash
# Emit the complete, byte-sorted C4 workspace corpus through one bounded,
# fail-closed inventory. This runs outside Bash process substitution so every
# discovery/build/write failure remains observable by preflight.
set -euo pipefail

root=${1:-examples}
script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
work=$(mktemp -d)
inventory=$work/c4.inventory
tree_inventory=$work/tree-inventory
cleanup() {
  rm -rf -- "$work"
}
trap cleanup EXIT
if [[ "$root" == examples ]]; then
  if ! "$script_dir/example-inventory.sh" c4 >"$inventory"; then
    echo "C4 workspace discovery failed; example inventory is untrusted" >&2
    exit 1
  fi
else
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
  if ! "$tree_inventory" -root "$root" -file-name workspace.dsl -regular-files-only \
    -max-entries 4096 -max-depth 64 -max-bytes 8388608 -timeout 15s >"$inventory"; then
    echo "C4 workspace discovery failed; bounded inventory is untrusted" >&2
    exit 1
  fi
fi
if [[ ! -s "$inventory" ]]; then
  echo "C4 workspace discovery returned an unexpected empty corpus under $root" >&2
  exit 1
fi
cat "$inventory"
