# Generated transition oracle: `fulfillmentSaga`

Generated from `FulfillmentSaga.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one test case.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Reserving | atomic | - | - |
| Paying | atomic | - | - |
| Shipping | atomic | - | - |
| CompensatingPay | atomic | - | - |
| CompensatingReserve | atomic | - | - |
| releaseRetry | atomic | - | - |
| Completed | final | - | - |
| Failed | final | - | - |

## Transitions

| test id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|
| T-FULF-01 | Reserving | after:reserveTimeout | - | Failed | recordReserveTimeout |
| T-FULF-02 | Reserving | onDone:reserveInventory | - | Paying | markReserved |
| T-FULF-03 | Reserving | onError:reserveInventory | - | Failed | recordReserveFailed |
| T-FULF-04 | Paying | after:payTimeout | - | CompensatingReserve | recordPayTimeout |
| T-FULF-05 | Paying | onDone:capturePayment | - | Shipping | markPaid |
| T-FULF-06 | Paying | onError:capturePayment | - | CompensatingReserve | recordPayFailed |
| T-FULF-07 | Shipping | after:shipTimeout | - | CompensatingPay | recordShipTimeout |
| T-FULF-08 | Shipping | onDone:dispatchShipment | - | Completed | markShipped |
| T-FULF-09 | Shipping | onError:dispatchShipment | - | CompensatingPay | recordShipFailed |
| T-FULF-10 | CompensatingPay | after:refundTimeout | - | CompensatingReserve | recordRefundTimeout |
| T-FULF-11 | CompensatingPay | onDone:refundPayment | - | CompensatingReserve | recordRefunded |
| T-FULF-12 | CompensatingPay | onError:refundPayment | - | CompensatingReserve | recordRefundFailed |
| T-FULF-13 | CompensatingReserve | after:releaseTimeout | - | releaseRetry | recordReleaseTimeout |
| T-FULF-14 | CompensatingReserve | onDone:releaseReservations | - | Failed | recordReleased |
| T-FULF-15 | CompensatingReserve | onError:releaseReservations | - | releaseRetry | recordReleaseError |
| T-FULF-16 | releaseRetry | after:releaseBackoff | - | CompensatingReserve | incrementRetries |
| T-FULF-17 | releaseRetry | always | retriesExhausted | Failed | recordReleaseGaveUp |

Total transitions (test cases): 17
