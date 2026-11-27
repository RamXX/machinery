#!/bin/sh
# End-to-end proof of packing: builds the real depot binary, drives it, asserts
# what it does, and only then emits one receipt per obligation it observed.
#
# machinery:verifies tc:SHIP-422414 tc:SHIP-ab88ab tc:SHIP-e82c34
set -eu
cd "$(dirname "$0")/.."
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
go build -o "$work/depot" ./cmd/depot
depot() { "$work/depot" --db "$work/db.json" "$@"; }
fail() { echo "FAIL: $1" >&2; exit 1; }
receipt() {
  # receipt <id> <observed> [<effects-json>]
  [ -n "${MACHINERY_RECEIPTS:-}" ] || return 0
  printf '{"v":1,"type":"receipt","id":"%s","outcome":"pass","test":"shipment_e2e.sh","effects":%s,"observed":"%s"}\n' "$1" "${3:-[]}" "$2" >> "$MACHINERY_RECEIPTS"
}

# Created on:pack -> Packed, effect recordPacking: the store keeps the history
[ "$(depot ship create s1)" = "shipment s1 created" ] || fail "create"
[ "$(depot ship pack s1)" = "shipment s1 packed" ] || fail "pack"
[ "$(depot ship show s1)" = "shipment s1 packed" ] || fail "status after pack"
tr -d ' \n' < "$work/db.json" | grep -q '"history":\["created","packed"\]' || fail "packing not recorded in the store"
receipt tc:SHIP-422414 'Created -> Packed; packing recorded in the store history' '["recordPacking"]'

# Packed ignore:pack: refused, nothing changes
if depot ship pack s1 2>"$work/err"; then fail "second pack was accepted"; fi
grep -q '^SHIPMENT_NOT_CREATED' "$work/err" || fail "refusal code of a second pack"
[ "$(depot ship show s1)" = "shipment s1 packed" ] || fail "status changed on a refused pack"
receipt tc:SHIP-ab88ab 'pack refused with SHIPMENT_NOT_CREATED, still Packed'

# Dispatched ignore:pack: refused, nothing changes
[ "$(depot ship dispatch s1)" = "shipment s1 dispatched" ] || fail "dispatch"
if depot ship pack s1 2>"$work/err"; then fail "pack of a dispatched shipment was accepted"; fi
grep -q '^SHIPMENT_NOT_CREATED' "$work/err" || fail "refusal code of a late pack"
[ "$(depot ship show s1)" = "shipment s1 dispatched" ] || fail "status changed on a refused pack"
receipt tc:SHIP-e82c34 'pack refused with SHIPMENT_NOT_CREATED, still Dispatched'
echo "shipment_e2e: ok"
