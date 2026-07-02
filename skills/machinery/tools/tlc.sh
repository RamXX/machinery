#!/usr/bin/env bash
# Run the TLC model checker on a .tla/.cfg pair, fetching a pinned tla2tools.jar
# into a cache on first use and verifying its checksum. Requires Java 11+.
# Usage: tlc.sh <spec.tla> [spec.cfg]
# Override the pin with TLA_TOOLS_VERSION + TLA_TOOLS_SHA256 (both, deliberately).
set -euo pipefail

TLA_VERSION="${TLA_TOOLS_VERSION:-v1.7.4}"
TLA_SHA256="${TLA_TOOLS_SHA256:-936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88}"
JAR="${TLA_TOOLS_JAR:-$HOME/.cache/machinery/tla2tools-$TLA_VERSION.jar}"
URL="https://github.com/tlaplus/tlaplus/releases/download/$TLA_VERSION/tla2tools.jar"

sha256_of() { (shasum -a 256 "$1" 2>/dev/null || sha256sum "$1") | awk '{print $1}'; }

tla="$1"
cfg="${2:-${tla%.tla}.cfg}"

if [ ! -f "$JAR" ]; then
  mkdir -p "$(dirname "$JAR")"
  echo "fetching tla2tools.jar $TLA_VERSION into $JAR" >&2
  curl -fsSL -o "$JAR.tmp" "$URL"
  got="$(sha256_of "$JAR.tmp")"
  if [ "$got" != "$TLA_SHA256" ]; then
    rm -f "$JAR.tmp"
    echo "checksum mismatch for tla2tools.jar $TLA_VERSION: got $got, want $TLA_SHA256" >&2
    exit 1
  fi
  mv "$JAR.tmp" "$JAR"
fi

dir="$(cd "$(dirname "$tla")" && pwd)"
java -XX:+UseParallelGC -cp "$JAR" tlc2.TLC -cleanup \
  -config "$dir/$(basename "$cfg")" "$dir/$(basename "$tla")"
