#!/usr/bin/env bash
# Compatibility entrypoint. The Go verifier is the single source of truth for
# generation ownership, exact artifact inventory, relational layers, pack
# refinement, and TLC failure semantics.
set -euo pipefail

machinery="${MACHINERY:-machinery}"
command -v "$machinery" >/dev/null 2>&1 || {
  echo "machinery binary not found on PATH, or set MACHINERY=/path/to/machinery" >&2
  exit 1
}

[ "$#" -eq 1 ] || {
  echo "usage: verify_formal.sh <design-dir>" >&2
  exit 2
}
exec "$machinery" verify-formal "$1"
