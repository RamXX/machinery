# Generated transition oracle: `task`

Generated from `Task.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.5-dev -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

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

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-TASK-01 | TASK-db41f8 | Open | on:start | guardCanStart | persisting | setPendingStart |
| T-TASK-02 | TASK-2a7cdb | Open | on:start | - | (internal) | recordStartDenied |
| T-TASK-03 | TASK-2019ec | Open | on:complete | guardCanComplete | persisting | setPendingComplete |
| T-TASK-04 | TASK-84d702 | Open | on:complete | - | (internal) | recordCompleteDenied |
| T-TASK-05 | TASK-b819d1 | Open | on:cancel | guardCanCancel | persisting | setPendingCancel |
| T-TASK-06 | TASK-36d38a | Open | on:cancel | - | (internal) | recordCancelDenied |
| T-TASK-07 | TASK-7ab0ac | Open | on:reassign | guardCanReassign | persisting | setPendingReassign |
| T-TASK-08 | TASK-b179c7 | Open | on:reassign | - | (internal) | recordReassignDenied |
| T-TASK-09 | TASK-173f61 | InProgress | on:start | - | (internal) | recordAlreadyStarted |
| T-TASK-10 | TASK-72ad76 | InProgress | on:complete | guardCanComplete | persisting | setPendingComplete |
| T-TASK-11 | TASK-7d91c2 | InProgress | on:complete | - | (internal) | recordCompleteDenied |
| T-TASK-12 | TASK-cdda50 | InProgress | on:cancel | guardCanCancel | persisting | setPendingCancel |
| T-TASK-13 | TASK-d159c9 | InProgress | on:cancel | - | (internal) | recordCancelDenied |
| T-TASK-14 | TASK-2f2bc8 | InProgress | on:reassign | guardCanReassign | persisting | setPendingReassign |
| T-TASK-15 | TASK-91fb4d | InProgress | on:reassign | - | (internal) | recordReassignDenied |
| T-TASK-16 | TASK-b4999d | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-TASK-17 | TASK-6d5eb1 | persisting | onDone:saveTask | pendingIsOpen | Open | commitStatus |
| T-TASK-18 | TASK-ae4260 | persisting | onDone:saveTask | pendingIsInProgress | InProgress | commitStatus |
| T-TASK-19 | TASK-c56bd7 | persisting | onDone:saveTask | pendingIsDone | Done | commitStatus |
| T-TASK-20 | TASK-67b0ff | persisting | onDone:saveTask | pendingIsCancelled | Cancelled | commitStatus |
| T-TASK-21 | TASK-d5bcc8 | persisting | onDone:saveTask | - | rolledBack | recordRoutingError |
| T-TASK-22 | TASK-8d6955 | persisting | onError:saveTask | isErrLocked | persistRetry | recordError |
| T-TASK-23 | TASK-376b22 | persisting | onError:saveTask | isErrConstraint | rolledBack | recordConstraint |
| T-TASK-24 | TASK-dc5fe1 | persisting | onError:saveTask | isErrDiskFull | rolledBack | recordDiskFull |
| T-TASK-25 | TASK-21e793 | persisting | onError:saveTask | isErrTimeout | rolledBack | recordTimeout |
| T-TASK-26 | TASK-be8721 | persisting | onError:saveTask | - | rolledBack | recordUnknownError |
| T-TASK-27 | TASK-168d9b | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-TASK-28 | TASK-0dd646 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-TASK-29 | TASK-3f585f | rolledBack | always | priorIsOpen | Open | - |
| T-TASK-30 | TASK-98c3ba | rolledBack | always | priorIsInProgress | InProgress | - |

Total transitions (test cases): 30
