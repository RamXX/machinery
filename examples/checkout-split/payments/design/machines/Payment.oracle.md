# Generated transition oracle: `payment`

Generated from `Payment.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.5-dev -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Requested | atomic | - | - |
| Captured | atomic | - | - |
| Declined | final | - | - |
| Refunded | final | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-PAYM-01 | PAYM-975859 | Requested | on:capture | - | Captured | markPaid |
| T-PAYM-02 | PAYM-36d434 | Requested | on:decline | - | Declined | markDeclined |
| T-PAYM-03 | PAYM-8835f6 | Captured | on:refund | - | Refunded | recordRefund |

Total transitions (test cases): 3
