#!/bin/sh
# Fixture evidence emitter. It stands in for a real end-to-end suite: it appends
# one receipt per obligation to the file machinery names in MACHINERY_RECEIPTS.
#
# machinery:verifies tc:ORDE-*
# machinery:verifies sc:customer-pays-and-the-order-ships sc:cancelled-order-refuses-payment
# machinery:verifies inv:paid-before-shipped inv:terminal-is-final inv:customer-email-unique
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
pass sc:customer-pays-and-the-order-ships '[]' 'order placed, paid and shipped'
pass sc:cancelled-order-refuses-payment '[]' 'pay refused on a cancelled order'
pass inv:paid-before-shipped '[]' 'ship from Pending refused'
pass inv:terminal-is-final '[]' 'no event changes a Shipped or Cancelled order'
pass inv:customer-email-unique '[]' 'second customer with the same email refused'
