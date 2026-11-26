#!/bin/sh
# Fixture evidence emitter for the provider outage: it stands in for a test that
# makes the payment provider fail and observes the surfaced error.
#
# machinery:verifies fm:payment-provider-down
set -eu
[ -n "${MACHINERY_RECEIPTS:-}" ] || exit 0
printf '{"v":1,"type":"receipt","id":"fm:payment-provider-down","outcome":"pass","test":"outage.sh","observed":"provider down; order back to Pending, PAYMENT_UNAVAILABLE"}\n' >> "$MACHINERY_RECEIPTS"
