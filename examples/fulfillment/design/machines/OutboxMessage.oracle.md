# Generated transition oracle: `outboxMessage`

Generated from `OutboxMessage.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Pending | atomic | - | - |
| Published | atomic | - | - |
| Consumed | final | - | - |
| publishing | atomic | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-OUTB-01 | OUTB-265e80 | Pending | on:publish | - | publishing | loadPayload |
| T-OUTB-02 | OUTB-8ee882 | Published | on:markConsumed | - | persisting | setPendingConsumed |
| T-OUTB-03 | OUTB-7fe006 | publishing | after:busTimeout | - | rolledBack | recordPublishTimeout |
| T-OUTB-04 | OUTB-b71c55 | publishing | onDone:publishToBus | - | persisting | setPendingPublished |
| T-OUTB-05 | OUTB-61e189 | publishing | onError:publishToBus | - | rolledBack | recordPublishError |
| T-OUTB-06 | OUTB-37e30a | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-OUTB-07 | OUTB-563ac6 | persisting | onDone:persistOutboxRow | pendingIsPublished | Published | commitStatus |
| T-OUTB-08 | OUTB-a37134 | persisting | onDone:persistOutboxRow | pendingIsConsumed | Consumed | commitStatus |
| T-OUTB-09 | OUTB-f8e389 | persisting | onDone:persistOutboxRow | - | rolledBack | recordRoutingError |
| T-OUTB-10 | OUTB-ad4b3e | persisting | onError:persistOutboxRow | isErrUnavailable | persistRetry | recordError |
| T-OUTB-11 | OUTB-f7db23 | persisting | onError:persistOutboxRow | isErrConflict | persistRetry | recordConflict |
| T-OUTB-12 | OUTB-6c5ad5 | persisting | onError:persistOutboxRow | - | rolledBack | recordUnknownError |
| T-OUTB-13 | OUTB-6b5079 | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-OUTB-14 | OUTB-073c83 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-OUTB-15 | OUTB-e15197 | rolledBack | always | priorIsPending | Pending | - |
| T-OUTB-16 | OUTB-34ea02 | rolledBack | always | priorIsPublished | Published | - |

Total transitions (test cases): 16
