# Generated transition oracle: `order`

Generated from `Order.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.4-dev -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Pending | atomic | - | - |
| Confirmed | atomic | - | - |
| Reserved | atomic | - | - |
| Paid | atomic | - | - |
| Shipped | atomic | - | - |
| Delivered | final | - | - |
| Cancelled | final | - | - |
| Failed | final | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-ORDE-01 | ORDE-eb2d3b | Pending | on:confirm | guardCanConfirm | persisting | setPendingConfirmed |
| T-ORDE-02 | ORDE-0e3d1c | Pending | on:confirm | - | (internal) | recordConfirmDenied |
| T-ORDE-03 | ORDE-105d03 | Pending | on:cancel | guardCanCancel | persisting | setPendingCancelled |
| T-ORDE-04 | ORDE-1ded14 | Pending | on:cancel | - | (internal) | recordCancelDenied |
| T-ORDE-05 | ORDE-3c919f | Confirmed | on:markReserved | - | persisting | setPendingReserved |
| T-ORDE-06 | ORDE-45acde | Confirmed | on:cancel | guardCanCancel | persisting | setPendingCancelled |
| T-ORDE-07 | ORDE-f66b57 | Confirmed | on:cancel | - | (internal) | recordCancelDenied |
| T-ORDE-08 | ORDE-df0887 | Confirmed | on:fail | - | persisting | setPendingFailed |
| T-ORDE-09 | ORDE-fb2298 | Reserved | on:markPaid | - | persisting | setPendingPaid |
| T-ORDE-10 | ORDE-d86830 | Reserved | on:fail | - | persisting | setPendingFailed |
| T-ORDE-11 | ORDE-cce1ac | Paid | on:markShipped | - | persisting | setPendingShipped |
| T-ORDE-12 | ORDE-c08239 | Paid | on:fail | - | persisting | setPendingFailed |
| T-ORDE-13 | ORDE-45dbe0 | Shipped | on:markDelivered | - | persisting | setPendingDelivered |
| T-ORDE-14 | ORDE-22f51c | Shipped | on:fail | - | persisting | setPendingFailed |
| T-ORDE-15 | ORDE-6ef582 | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-ORDE-16 | ORDE-6cd784 | persisting | onDone:persistOrder | pendingIsConfirmed | Confirmed | commitStatus |
| T-ORDE-17 | ORDE-9cace2 | persisting | onDone:persistOrder | pendingIsReserved | Reserved | commitStatus |
| T-ORDE-18 | ORDE-264dbd | persisting | onDone:persistOrder | pendingIsPaid | Paid | commitStatus |
| T-ORDE-19 | ORDE-7b4e42 | persisting | onDone:persistOrder | pendingIsShipped | Shipped | commitStatus |
| T-ORDE-20 | ORDE-d7360c | persisting | onDone:persistOrder | pendingIsDelivered | Delivered | commitStatus |
| T-ORDE-21 | ORDE-1db084 | persisting | onDone:persistOrder | pendingIsCancelled | Cancelled | commitStatus |
| T-ORDE-22 | ORDE-9bc690 | persisting | onDone:persistOrder | pendingIsFailed | Failed | commitStatus |
| T-ORDE-23 | ORDE-b00d3f | persisting | onDone:persistOrder | - | rolledBack | recordRoutingError |
| T-ORDE-24 | ORDE-7c6693 | persisting | onError:persistOrder | isErrUnavailable | persistRetry | recordError |
| T-ORDE-25 | ORDE-b0f09e | persisting | onError:persistOrder | isErrConflict | persistRetry | recordConflict |
| T-ORDE-26 | ORDE-d42479 | persisting | onError:persistOrder | - | rolledBack | recordUnknownError |
| T-ORDE-27 | ORDE-2dad10 | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-ORDE-28 | ORDE-9933e0 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-ORDE-29 | ORDE-b98c4d | rolledBack | always | priorIsPending | Pending | - |
| T-ORDE-30 | ORDE-00eb2c | rolledBack | always | priorIsConfirmed | Confirmed | - |
| T-ORDE-31 | ORDE-01d062 | rolledBack | always | priorIsReserved | Reserved | - |
| T-ORDE-32 | ORDE-a4c813 | rolledBack | always | priorIsPaid | Paid | - |
| T-ORDE-33 | ORDE-87faa7 | rolledBack | always | priorIsShipped | Shipped | - |

Total transitions (test cases): 33
