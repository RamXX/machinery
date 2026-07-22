# Generated transition oracle: `deal`

Generated from `Deal.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.4-dev -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Lead | atomic | - | - |
| Qualified | atomic | - | - |
| Proposal | atomic | - | - |
| Negotiation | atomic | - | - |
| Won | atomic | - | - |
| Lost | atomic | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-DEAL-01 | DEAL-eb0c40 | Lead | on:advanceStage | guardCanAdvance | persisting | setPendingAdvance |
| T-DEAL-02 | DEAL-38ba11 | Lead | on:advanceStage | - | (internal) | recordAdvanceDenied |
| T-DEAL-03 | DEAL-1fe825 | Lead | on:win | guardCanWin | persisting | setPendingWin |
| T-DEAL-04 | DEAL-e786d8 | Lead | on:win | - | (internal) | recordWinDenied |
| T-DEAL-05 | DEAL-b76457 | Lead | on:lose | guardCanLose | persisting | setPendingLose |
| T-DEAL-06 | DEAL-fdf795 | Lead | on:lose | - | (internal) | recordLoseDenied |
| T-DEAL-07 | DEAL-1d9aa0 | Lead | on:reopen | - | (internal) | recordReopenNotTerminal |
| T-DEAL-08 | DEAL-a14020 | Qualified | on:advanceStage | guardCanAdvance | persisting | setPendingAdvance |
| T-DEAL-09 | DEAL-0c4c47 | Qualified | on:advanceStage | - | (internal) | recordAdvanceDenied |
| T-DEAL-10 | DEAL-492234 | Qualified | on:win | guardCanWin | persisting | setPendingWin |
| T-DEAL-11 | DEAL-81d0ab | Qualified | on:win | - | (internal) | recordWinDenied |
| T-DEAL-12 | DEAL-f7d8b2 | Qualified | on:lose | guardCanLose | persisting | setPendingLose |
| T-DEAL-13 | DEAL-9f48af | Qualified | on:lose | - | (internal) | recordLoseDenied |
| T-DEAL-14 | DEAL-990c3b | Qualified | on:reopen | - | (internal) | recordReopenNotTerminal |
| T-DEAL-15 | DEAL-388687 | Proposal | on:advanceStage | guardCanAdvance | persisting | setPendingAdvance |
| T-DEAL-16 | DEAL-5df488 | Proposal | on:advanceStage | - | (internal) | recordAdvanceDenied |
| T-DEAL-17 | DEAL-7e1e9b | Proposal | on:win | guardCanWin | persisting | setPendingWin |
| T-DEAL-18 | DEAL-df4442 | Proposal | on:win | - | (internal) | recordWinDenied |
| T-DEAL-19 | DEAL-fde084 | Proposal | on:lose | guardCanLose | persisting | setPendingLose |
| T-DEAL-20 | DEAL-e16eea | Proposal | on:lose | - | (internal) | recordLoseDenied |
| T-DEAL-21 | DEAL-44482d | Proposal | on:reopen | - | (internal) | recordReopenNotTerminal |
| T-DEAL-22 | DEAL-708606 | Negotiation | on:advanceStage | - | (internal) | recordAdvanceDenied |
| T-DEAL-23 | DEAL-38140e | Negotiation | on:win | guardCanWin | persisting | setPendingWin |
| T-DEAL-24 | DEAL-3bbe10 | Negotiation | on:win | - | (internal) | recordWinDenied |
| T-DEAL-25 | DEAL-8fde14 | Negotiation | on:lose | guardCanLose | persisting | setPendingLose |
| T-DEAL-26 | DEAL-b5154b | Negotiation | on:lose | - | (internal) | recordLoseDenied |
| T-DEAL-27 | DEAL-69312c | Negotiation | on:reopen | - | (internal) | recordReopenNotTerminal |
| T-DEAL-28 | DEAL-99392a | Won | on:reopen | guardCanReopen | persisting | setPendingReopen |
| T-DEAL-29 | DEAL-5746cc | Won | on:reopen | - | (internal) | recordReopenDenied |
| T-DEAL-30 | DEAL-e0bdaf | Won | on:advanceStage | - | (internal) | recordTerminalRejected |
| T-DEAL-31 | DEAL-d27905 | Won | on:win | - | (internal) | recordTerminalRejected |
| T-DEAL-32 | DEAL-a45f13 | Won | on:lose | - | (internal) | recordTerminalRejected |
| T-DEAL-33 | DEAL-0fef3d | Lost | on:reopen | guardCanReopen | persisting | setPendingReopen |
| T-DEAL-34 | DEAL-7bb594 | Lost | on:reopen | - | (internal) | recordReopenDenied |
| T-DEAL-35 | DEAL-0a25a2 | Lost | on:advanceStage | - | (internal) | recordTerminalRejected |
| T-DEAL-36 | DEAL-0ec705 | Lost | on:win | - | (internal) | recordTerminalRejected |
| T-DEAL-37 | DEAL-e9e60a | Lost | on:lose | - | (internal) | recordTerminalRejected |
| T-DEAL-38 | DEAL-24f320 | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-DEAL-39 | DEAL-5abbd2 | persisting | onDone:saveDeal | pendingIsQualified | Qualified | commitStage |
| T-DEAL-40 | DEAL-da0ce2 | persisting | onDone:saveDeal | pendingIsProposal | Proposal | commitStage |
| T-DEAL-41 | DEAL-47ce0d | persisting | onDone:saveDeal | pendingIsNegotiation | Negotiation | commitStage |
| T-DEAL-42 | DEAL-e5d58e | persisting | onDone:saveDeal | pendingIsWon | Won | commitStage, commitCloseDate |
| T-DEAL-43 | DEAL-03d4fb | persisting | onDone:saveDeal | pendingIsLost | Lost | commitStage |
| T-DEAL-44 | DEAL-92b688 | persisting | onDone:saveDeal | - | rolledBack | recordRoutingError |
| T-DEAL-45 | DEAL-809c09 | persisting | onError:saveDeal | isErrLocked | persistRetry | recordError |
| T-DEAL-46 | DEAL-cf2596 | persisting | onError:saveDeal | isErrConstraint | rolledBack | recordConstraint |
| T-DEAL-47 | DEAL-daae59 | persisting | onError:saveDeal | isErrDiskFull | rolledBack | recordDiskFull |
| T-DEAL-48 | DEAL-41c002 | persisting | onError:saveDeal | isErrTimeout | rolledBack | recordTimeout |
| T-DEAL-49 | DEAL-7d1911 | persisting | onError:saveDeal | - | rolledBack | recordUnknownError |
| T-DEAL-50 | DEAL-450b55 | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-DEAL-51 | DEAL-8c9948 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-DEAL-52 | DEAL-210c14 | rolledBack | always | priorIsLead | Lead | - |
| T-DEAL-53 | DEAL-793e1f | rolledBack | always | priorIsQualified | Qualified | - |
| T-DEAL-54 | DEAL-97c3ea | rolledBack | always | priorIsProposal | Proposal | - |
| T-DEAL-55 | DEAL-8a4caf | rolledBack | always | priorIsNegotiation | Negotiation | - |
| T-DEAL-56 | DEAL-9b6ee7 | rolledBack | always | priorIsWon | Won | - |
| T-DEAL-57 | DEAL-21905a | rolledBack | always | priorIsLost | Lost | - |

Total transitions (test cases): 57
