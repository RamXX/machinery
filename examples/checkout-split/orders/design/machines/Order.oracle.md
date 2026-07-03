# Generated transition oracle: `order`

Generated from `Order.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Placed | atomic | request | - |
| Paid | atomic | - | - |
| Shipped | final | - | - |
| Declined | final | - | - |
| Cancelled | final | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-ORDE-01 | ORDE-33e568 | Placed | on:markPaid | - | Paid | - |
| T-ORDE-02 | ORDE-e080b0 | Placed | on:markDeclined | - | Declined | - |
| T-ORDE-03 | ORDE-4d2d74 | Placed | on:cancel | - | Cancelled | recordCancel |
| T-ORDE-04 | ORDE-e6d28f | Paid | on:ship | - | Shipped | recordShipment |

Total transitions (test cases): 4
