# Generated transition oracle: `task`

Generated from `Task.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Open | atomic | - | - |
| InProgress | atomic | - | - |
| Done | final | - | - |
| Abandoned | final | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-TASK-01 | TASK-2a7cdb | Open | on:start | - | persisting | setPendingStart |
| T-TASK-02 | TASK-696c9c | Open | on:abandon | - | persisting | setPendingAbandon |
| T-TASK-03 | TASK-7d91c2 | InProgress | on:complete | - | persisting | setPendingComplete |
| T-TASK-04 | TASK-9d6932 | InProgress | on:abandon | - | persisting | setPendingAbandon |
| T-TASK-05 | TASK-cff0db | persisting | after:PERSIST_TIMEOUT | - | persistRetry | - |
| T-TASK-06 | TASK-65d383 | persisting | onDone:persist | pendingIsInProgress | InProgress | commit |
| T-TASK-07 | TASK-007103 | persisting | onDone:persist | pendingIsDone | Done | commit |
| T-TASK-08 | TASK-d893c2 | persisting | onDone:persist | pendingIsAbandoned | Abandoned | commit |
| T-TASK-09 | TASK-6044df | persisting | onError:persist | isRetriable | persistRetry | - |
| T-TASK-10 | TASK-f47c62 | persisting | onError:persist | - | rolledBack | - |
| T-TASK-11 | TASK-0eb316 | persistRetry | after:RETRY_BACKOFF | - | persisting | incrementRetries |
| T-TASK-12 | TASK-0dd646 | persistRetry | always | retriesExhausted | rolledBack | - |
| T-TASK-13 | TASK-3f585f | rolledBack | always | priorIsOpen | Open | - |
| T-TASK-14 | TASK-98c3ba | rolledBack | always | priorIsInProgress | InProgress | - |

Total transitions (test cases): 14
