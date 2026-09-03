#!/usr/bin/env bash
# Validate and print the closed repository shell-script lint corpus.
set -euo pipefail

manifest=${1:-scripts/shellcheck-files.txt}
[[ -f "$manifest" && ! -L "$manifest" ]] || { echo "ShellCheck inventory $manifest must be a regular file" >&2; exit 1; }

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
repo_root=$(CDPATH='' cd -- "$script_dir/.." && pwd -P)
work=$(mktemp -d)
discovered=$work/discovered
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
if ! "$tree_inventory" \
  -literal .githooks/pre-push -literal install.sh \
  -root hooks -root scripts -root skills/machinery/tools -file-suffix .sh \
  -max-entries 100000 -max-depth 64 -max-bytes 33554432 -timeout 15s >"$discovered"; then
  echo "bounded ShellCheck corpus discovery failed" >&2
  exit 1
fi

while IFS= read -r script; do
  [[ -f "$script" && ! -L "$script" ]] || { echo "ShellCheck corpus entry must be a regular non-symlink file: $script" >&2; exit 1; }
done <"$discovered"

LC_ALL=C sort -c "$manifest" || { echo "ShellCheck inventory must be byte-sorted and unique" >&2; exit 1; }
if ! cmp -s "$manifest" "$discovered"; then
  echo "ShellCheck inventory does not exactly match repository shell scripts" >&2
  diff -u "$manifest" "$discovered" >&2 || true
  exit 1
fi
cat "$manifest"
