# Generated transition oracle: `fulfillmentSaga`

Generated from `FulfillmentSaga.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.5-dev -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Reserving | atomic | - | - |
| Paying | atomic | - | - |
| Shipping | atomic | - | - |
| Compensating | atomic | - | - |
| compensateRetry | atomic | - | - |
| Completed | final | - | - |
| Failed | final | - | - |
| FailedDirty | final | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-FULF-01 | FULF-4244df | Reserving | after:reserveTimeout | - | Failed | recordReserveTimeout |
| T-FULF-02 | FULF-ee2ed2 | Reserving | onDone:reserveInventory | - | Paying | markReserved |
| T-FULF-03 | FULF-d85462 | Reserving | onError:reserveInventory | - | Failed | recordReserveFailed |
| T-FULF-04 | FULF-de719f | Paying | after:payTimeout | - | Compensating | recordPayTimeout |
| T-FULF-05 | FULF-bba0be | Paying | onDone:capturePayment | - | Shipping | markPaid |
| T-FULF-06 | FULF-81e3cf | Paying | onError:capturePayment | - | Compensating | recordPayFailed |
| T-FULF-07 | FULF-6382a8 | Shipping | after:shipTimeout | - | Compensating | recordShipTimeout |
| T-FULF-08 | FULF-6ec4e1 | Shipping | onDone:dispatchShipment | - | Completed | markShipped |
| T-FULF-09 | FULF-9c8dcb | Shipping | onError:dispatchShipment | - | Compensating | recordShipFailed |
| T-FULF-10 | FULF-3c3b48 | Compensating | after:compensateTimeout | - | compensateRetry | recordCompensateTimeout |
| T-FULF-11 | FULF-0e0e20 | Compensating | onDone:compensate | - | Failed | recordCompensated |
| T-FULF-12 | FULF-6fa040 | Compensating | onError:compensate | - | compensateRetry | recordCompensateError |
| T-FULF-13 | FULF-076518 | compensateRetry | after:compensateBackoff | - | Compensating | incrementRetries |
| T-FULF-14 | FULF-1720f5 | compensateRetry | always | retriesExhausted | FailedDirty | recordCompensationIncomplete |

Total transitions (test cases): 14
