#!/usr/bin/env bash
# Compatibility entrypoint. Formal generation, pinned tools, the exact Java
# runtime closure, scratch isolation, and process-tree cleanup are owned by the
# machinery binary. Usage: tlc.sh <spec.tla> [spec.cfg]
set -euo pipefail

[ "$#" -ge 1 ] && [ "$#" -le 2 ] || {
  echo "usage: tlc.sh <spec.tla> [spec.cfg]" >&2
  exit 2
}

machinery="${MACHINERY:-machinery}"
command -v "$machinery" >/dev/null 2>&1 || {
  echo "machinery binary not found on PATH, or set MACHINERY=/path/to/machinery" >&2
  exit 1
}

spec_dir="$(cd "$(dirname "$1")" && pwd -P)"
[ "$(basename "$spec_dir")" = "formal" ] || {
  echo "tlc.sh requires a spec under <design-dir>/formal; use machinery verify-formal <design-dir>" >&2
  exit 2
}

if [ "$#" -eq 2 ] && [ "$(cd "$(dirname "$2")" && pwd -P)/$(basename "$2")" != "$spec_dir/$(basename "${1%.tla}.cfg")" ]; then
  echo "tlc.sh only accepts the canonical sibling .cfg; use machinery verify-formal for the owned suite" >&2
  exit 2
fi

exec "$machinery" verify-formal "$(dirname "$spec_dir")"
