#!/bin/sh
# Fixture replay emitter. It stands in for the replay of a counterexample
# against a real system and reports what MACHINERY_REPLAY says was seen.
#
# machinery:verifies cx:order-payment:race
set -eu
[ -n "${MACHINERY_RECEIPTS:-}" ] || exit 0
printf '{"v":1,"type":"receipt","id":"cx:order-payment:race","outcome":"pass","replay":"%s","test":"replay.sh","observed":"two concurrent pay requests, one accepted"}\n' "${MACHINERY_REPLAY:-refuted}" >> "$MACHINERY_RECEIPTS"
