# Generated transition oracle: `task`

Generated from `Task.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one test case.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Open | atomic | - | - |
| InProgress | atomic | - | - |
| Done | final | recordTaskClosed | - |
| Cancelled | final | recordTaskClosed | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|
| T-TASK-01 | Open | on:start | guardCanStart | persisting | setPendingStart |
| T-TASK-02 | Open | on:start | - | (internal) | recordStartDenied |
| T-TASK-03 | Open | on:complete | guardCanComplete | persisting | setPendingComplete |
| T-TASK-04 | Open | on:complete | - | (internal) | recordCompleteDenied |
| T-TASK-05 | Open | on:cancel | guardCanCancel | persisting | setPendingCancel |
| T-TASK-06 | Open | on:cancel | - | (internal) | recordCancelDenied |
| T-TASK-07 | Open | on:reassign | guardCanReassign | persisting | setPendingReassign |
| T-TASK-08 | Open | on:reassign | - | (internal) | recordReassignDenied |
| T-TASK-09 | InProgress | on:start | - | (internal) | recordAlreadyStarted |
| T-TASK-10 | InProgress | on:complete | guardCanComplete | persisting | setPendingComplete |
| T-TASK-11 | InProgress | on:complete | - | (internal) | recordCompleteDenied |
| T-TASK-12 | InProgress | on:cancel | guardCanCancel | persisting | setPendingCancel |
| T-TASK-13 | InProgress | on:cancel | - | (internal) | recordCancelDenied |
| T-TASK-14 | InProgress | on:reassign | guardCanReassign | persisting | setPendingReassign |
| T-TASK-15 | InProgress | on:reassign | - | (internal) | recordReassignDenied |
| T-TASK-16 | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-TASK-17 | persisting | onDone:saveTask | pendingIsOpen | Open | commitStatus |
| T-TASK-18 | persisting | onDone:saveTask | pendingIsInProgress | InProgress | commitStatus |
| T-TASK-19 | persisting | onDone:saveTask | pendingIsDone | Done | commitStatus |
| T-TASK-20 | persisting | onDone:saveTask | pendingIsCancelled | Cancelled | commitStatus |
| T-TASK-21 | persisting | onDone:saveTask | - | rolledBack | recordRoutingError |
| T-TASK-22 | persisting | onError:saveTask | isErrLocked | persistRetry | recordError |
| T-TASK-23 | persisting | onError:saveTask | isErrConstraint | rolledBack | recordConstraint |
| T-TASK-24 | persisting | onError:saveTask | isErrDiskFull | rolledBack | recordDiskFull |
| T-TASK-25 | persisting | onError:saveTask | isErrTimeout | rolledBack | recordTimeout |
| T-TASK-26 | persisting | onError:saveTask | - | rolledBack | recordUnknownError |
| T-TASK-27 | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-TASK-28 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-TASK-29 | rolledBack | always | priorIsOpen | Open | - |
| T-TASK-30 | rolledBack | always | priorIsInProgress | InProgress | - |

Total transitions (test cases): 30
