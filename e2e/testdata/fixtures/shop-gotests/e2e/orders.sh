#!/bin/sh
# Fixture evidence emitter of the Go-suite project. It claims and emits the transition cases
# only, which the run adapter never proves: every other id of this project is claimed by the Go
# suite alone.
#
# machinery:verifies tc:ORDE-*
set -eu
[ -n "${MACHINERY_RECEIPTS:-}" ] || exit 0

pass() {
  # pass <id> <effects-json-array> <observed>
  printf '{"v":1,"type":"receipt","id":"%s","outcome":"pass","test":"orders.sh","effects":%s,"observed":"%s"}\n' "$1" "$2" "$3" >> "$MACHINERY_RECEIPTS"
}

pass tc:ORDE-2e883d '["stampAttempt"]' 'Pending -> paying'
pass tc:ORDE-70d94f '["recordPayRefused"]' 'pay refused, still Pending'
pass tc:ORDE-1ded14 '["recordCancellation"]' 'Pending -> Cancelled'
pass tc:ORDE-ad945c '["recordPaymentTimeout"]' 'paying -> Pending after timeout'
pass tc:ORDE-8bd6e7 '[]' 'paying -> Paid; charge record not observed'
pass tc:ORDE-267d12 '["recordPaymentFailure"]' 'paying -> Pending on provider error'
pass tc:ORDE-e6d28f '["recordShipment","notifyCustomer"]' 'Paid -> Shipped'
pass tc:ORDE-507e4f '["refundPayment","recordCancellation"]' 'Paid -> Cancelled'
pass tc:ORDE-2fa2af '[]' 'ship refused with ORDER_NOT_PAID, still Pending'
pass tc:ORDE-8799a1 '[]' 'pay refused with ORDER_ALREADY_PAID, still Paid'
