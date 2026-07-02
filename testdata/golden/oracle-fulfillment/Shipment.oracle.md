# Generated transition oracle: `shipment`

Generated from `Shipment.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Pending | atomic | - | - |
| Dispatched | atomic | - | - |
| InTransit | atomic | - | - |
| Delivered | final | - | - |
| Lost | final | - | - |
| dispatching | atomic | - | - |
| carrierRetry | atomic | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-SHIP-01 | SHIP-26175d | Pending | on:dispatch | - | dispatching | setCarrierDispatch |
| T-SHIP-02 | SHIP-e816f2 | Dispatched | on:markInTransit | - | persisting | setPendingInTransit |
| T-SHIP-03 | SHIP-285f38 | Dispatched | on:deliver | - | persisting | setPendingDelivered |
| T-SHIP-04 | SHIP-0407ce | Dispatched | on:markLost | - | persisting | setPendingLost |
| T-SHIP-05 | SHIP-7cf986 | InTransit | on:deliver | - | persisting | setPendingDelivered |
| T-SHIP-06 | SHIP-883bf9 | InTransit | on:markLost | - | persisting | setPendingLost |
| T-SHIP-07 | SHIP-6a449a | dispatching | after:carrierTimeout | - | carrierRetry | recordCarrierTimeout |
| T-SHIP-08 | SHIP-50ce91 | dispatching | onDone:carrierDispatch | - | persisting | captureTrackingId, setPendingDispatched |
| T-SHIP-09 | SHIP-8c1a12 | dispatching | onError:carrierDispatch | isErrRetryable | carrierRetry | recordCarrierError |
| T-SHIP-10 | SHIP-de1d57 | dispatching | onError:carrierDispatch | - | rolledBack | recordDispatchFailed |
| T-SHIP-11 | SHIP-e77b68 | carrierRetry | after:carrierRetryBackoff | - | dispatching | incrementCarrierRetries |
| T-SHIP-12 | SHIP-bf7654 | carrierRetry | always | carrierRetriesExhausted | rolledBack | recordCarrierExhausted |
| T-SHIP-13 | SHIP-bf524c | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-SHIP-14 | SHIP-6c696f | persisting | onDone:persistShipment | pendingIsDispatched | Dispatched | commitStatus |
| T-SHIP-15 | SHIP-421b8d | persisting | onDone:persistShipment | pendingIsInTransit | InTransit | commitStatus |
| T-SHIP-16 | SHIP-029bb7 | persisting | onDone:persistShipment | pendingIsDelivered | Delivered | commitStatus |
| T-SHIP-17 | SHIP-62ec25 | persisting | onDone:persistShipment | pendingIsLost | Lost | commitStatus |
| T-SHIP-18 | SHIP-a6bc90 | persisting | onDone:persistShipment | - | rolledBack | recordRoutingError |
| T-SHIP-19 | SHIP-a3871a | persisting | onError:persistShipment | isErrUnavailable | persistRetry | recordError |
| T-SHIP-20 | SHIP-95f0e8 | persisting | onError:persistShipment | isErrConflict | persistRetry | recordConflict |
| T-SHIP-21 | SHIP-cad1a0 | persisting | onError:persistShipment | - | rolledBack | recordUnknownError |
| T-SHIP-22 | SHIP-4457a6 | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-SHIP-23 | SHIP-d01378 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-SHIP-24 | SHIP-125d9c | rolledBack | always | priorIsPending | Pending | - |
| T-SHIP-25 | SHIP-e7b4be | rolledBack | always | priorIsDispatched | Dispatched | - |
| T-SHIP-26 | SHIP-8146f7 | rolledBack | always | priorIsInTransit | InTransit | - |

Total transitions (test cases): 26
