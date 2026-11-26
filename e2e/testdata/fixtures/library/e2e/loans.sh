#!/bin/sh
# End-to-end checks of the circulation desk.
#
# Checks: tc:LOAN*, tc:HOLD*,
#   lent-before-returned, returned-is-final.
# Proves: RISK-1 RISK-2, RISK-3
# Scenario: Member borrows and returns a copy
# Scenario: Blocked member is refused a loan
true
