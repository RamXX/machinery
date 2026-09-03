#!/usr/bin/env bash
# Validate and print the closed repository shell-script lint corpus.
set -euo pipefail

manifest=${1:-scripts/shellcheck-files.txt}
[[ -f "$manifest" && ! -L "$manifest" ]] || { echo "ShellCheck inventory $manifest must be a regular file" >&2; exit 1; }

discovered=$(mktemp)
cleanup() { rm -f "$discovered"; }
trap cleanup EXIT

{
  printf '%s\n' .githooks/pre-push
  printf '%s\n' install.sh
  find hooks scripts skills/machinery/tools -name '*.sh' -print
} | LC_ALL=C sort -u >"$discovered"

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
