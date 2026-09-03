#!/usr/bin/env bash
# Emit the complete, byte-sorted C4 workspace corpus. This runs outside Bash
# process substitution so find/sort status is observable by preflight.
set -euo pipefail

root=${1:-examples}
inventory=$(mktemp)
trap 'rm -f "$inventory"' EXIT
if [[ "$root" == examples ]]; then
  if ! scripts/example-inventory.sh c4 >"$inventory"; then
    echo "C4 workspace discovery failed; example inventory is untrusted" >&2
    exit 1
  fi
elif ! find "$root" -type f -name workspace.dsl | LC_ALL=C sort >"$inventory"; then
  echo "C4 workspace discovery failed; inventory is untrusted" >&2
  exit 1
fi
if [[ ! -s "$inventory" ]]; then
  echo "C4 workspace discovery returned an unexpected empty corpus under $root" >&2
  exit 1
fi
cat "$inventory"
