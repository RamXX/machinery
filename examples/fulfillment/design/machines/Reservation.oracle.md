# Generated transition oracle: `reservation`

Generated from `Reservation.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Held | atomic | - | - |
| Committed | final | - | - |
| Released | final | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-RESE-01 | RESE-1581b4 | Held | on:commit | - | persisting | setPendingCommitted |
| T-RESE-02 | RESE-05202b | Held | on:release | - | persisting | setPendingReleased |
| T-RESE-03 | RESE-e34282 | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-RESE-04 | RESE-883427 | persisting | onDone:persistReservation | pendingIsCommitted | Committed | commitStatus |
| T-RESE-05 | RESE-618adb | persisting | onDone:persistReservation | pendingIsReleased | Released | commitStatus |
| T-RESE-06 | RESE-b80083 | persisting | onDone:persistReservation | - | rolledBack | recordRoutingError |
| T-RESE-07 | RESE-df5ac0 | persisting | onError:persistReservation | isErrUnavailable | persistRetry | recordError |
| T-RESE-08 | RESE-aca482 | persisting | onError:persistReservation | isErrConflict | persistRetry | recordConflict |
| T-RESE-09 | RESE-191a1c | persisting | onError:persistReservation | - | rolledBack | recordUnknownError |
| T-RESE-10 | RESE-892929 | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-RESE-11 | RESE-25f172 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-RESE-12 | RESE-d73c65 | rolledBack | always | priorIsHeld | Held | - |

Total transitions (test cases): 12
