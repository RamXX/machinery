#!/bin/sh
# Smoke test of the depot command line: builds the binary and drives it for real.
set -eu
cd "$(dirname "$0")/.."
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
go build -o "$work/depot" ./cmd/depot
depot() { "$work/depot" --db "$work/db.json" "$@"; }

expect() {
  # expect <want> <got>
  [ "$1" = "$2" ] || { echo "smoke: want '$1', got '$2'" >&2; exit 1; }
}

expect "shipment s1 created" "$(depot ship create s1)"
expect "shipment s1 packed" "$(depot ship pack s1)"
expect "shipment s1 dispatched" "$(depot ship dispatch s1)"
expect "shipment s1 delivered" "$(depot ship deliver s1)"
expect "shipment s1 delivered" "$(depot ship show s1)"

expect "shipment s2 created" "$(depot ship create s2)"
if depot ship dispatch s2 2>"$work/err"; then echo "smoke: dispatch of an unpacked shipment succeeded" >&2; exit 1; fi
grep -q '^SHIPMENT_NOT_PACKED' "$work/err" || { echo "smoke: missing SHIPMENT_NOT_PACKED" >&2; exit 1; }
expect "shipment s2 cancelled" "$(depot ship cancel s2)"

expect "invoice s1 subtotal=2000 tax=165 total=2165" "$(depot invoice s1 2000)"
echo "smoke: ok"
