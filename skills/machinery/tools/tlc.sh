#!/usr/bin/env bash
# Run the TLC model checker on a .tla/.cfg pair, fetching tla2tools.jar into a
# cache on first use. Requires Java 11+. Usage: tlc.sh <spec.tla> [spec.cfg]
set -euo pipefail

JAR="${TLA_TOOLS_JAR:-$HOME/.cache/machinery/tla2tools.jar}"
URL="https://github.com/tlaplus/tlaplus/releases/latest/download/tla2tools.jar"

tla="$1"
cfg="${2:-${tla%.tla}.cfg}"

if [ ! -f "$JAR" ]; then
  mkdir -p "$(dirname "$JAR")"
  echo "fetching tla2tools.jar into $JAR"
  curl -fsSL -o "$JAR" "$URL"
fi

dir="$(cd "$(dirname "$tla")" && pwd)"
java -XX:+UseParallelGC -cp "$JAR" tlc2.TLC \
  -config "$dir/$(basename "$cfg")" "$dir/$(basename "$tla")"
