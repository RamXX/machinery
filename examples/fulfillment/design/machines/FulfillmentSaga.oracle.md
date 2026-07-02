# Generated transition oracle: `fulfillmentSaga`

Generated from `FulfillmentSaga.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one test case.

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

| test id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|
| T-FULF-01 | Reserving | after:reserveTimeout | - | Failed | recordReserveTimeout |
| T-FULF-02 | Reserving | onDone:reserveInventory | - | Paying | markReserved |
| T-FULF-03 | Reserving | onError:reserveInventory | - | Failed | recordReserveFailed |
| T-FULF-04 | Paying | after:payTimeout | - | Compensating | recordPayTimeout |
| T-FULF-05 | Paying | onDone:capturePayment | - | Shipping | markPaid |
| T-FULF-06 | Paying | onError:capturePayment | - | Compensating | recordPayFailed |
| T-FULF-07 | Shipping | after:shipTimeout | - | Compensating | recordShipTimeout |
| T-FULF-08 | Shipping | onDone:dispatchShipment | - | Completed | markShipped |
| T-FULF-09 | Shipping | onError:dispatchShipment | - | Compensating | recordShipFailed |
| T-FULF-10 | Compensating | after:compensateTimeout | - | compensateRetry | recordCompensateTimeout |
| T-FULF-11 | Compensating | onDone:compensate | - | Failed | recordCompensated |
| T-FULF-12 | Compensating | onError:compensate | - | compensateRetry | recordCompensateError |
| T-FULF-13 | compensateRetry | after:compensateBackoff | - | Compensating | incrementRetries |
| T-FULF-14 | compensateRetry | always | retriesExhausted | FailedDirty | recordCompensationIncomplete |

Total transitions (test cases): 14
