# Generated transition oracle: `deal`

Generated from `Deal.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one test case.

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

| test id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|
| T-DEAL-01 | Lead | on:advanceStage | guardCanAdvance | persisting | setPendingAdvance |
| T-DEAL-02 | Lead | on:advanceStage | - | (internal) | recordAdvanceDenied |
| T-DEAL-03 | Lead | on:win | guardCanWin | persisting | setPendingWin |
| T-DEAL-04 | Lead | on:win | - | (internal) | recordWinDenied |
| T-DEAL-05 | Lead | on:lose | guardCanLose | persisting | setPendingLose |
| T-DEAL-06 | Lead | on:lose | - | (internal) | recordLoseDenied |
| T-DEAL-07 | Lead | on:reopen | - | (internal) | recordReopenNotTerminal |
| T-DEAL-08 | Qualified | on:advanceStage | guardCanAdvance | persisting | setPendingAdvance |
| T-DEAL-09 | Qualified | on:advanceStage | - | (internal) | recordAdvanceDenied |
| T-DEAL-10 | Qualified | on:win | guardCanWin | persisting | setPendingWin |
| T-DEAL-11 | Qualified | on:win | - | (internal) | recordWinDenied |
| T-DEAL-12 | Qualified | on:lose | guardCanLose | persisting | setPendingLose |
| T-DEAL-13 | Qualified | on:lose | - | (internal) | recordLoseDenied |
| T-DEAL-14 | Qualified | on:reopen | - | (internal) | recordReopenNotTerminal |
| T-DEAL-15 | Proposal | on:advanceStage | guardCanAdvance | persisting | setPendingAdvance |
| T-DEAL-16 | Proposal | on:advanceStage | - | (internal) | recordAdvanceDenied |
| T-DEAL-17 | Proposal | on:win | guardCanWin | persisting | setPendingWin |
| T-DEAL-18 | Proposal | on:win | - | (internal) | recordWinDenied |
| T-DEAL-19 | Proposal | on:lose | guardCanLose | persisting | setPendingLose |
| T-DEAL-20 | Proposal | on:lose | - | (internal) | recordLoseDenied |
| T-DEAL-21 | Proposal | on:reopen | - | (internal) | recordReopenNotTerminal |
| T-DEAL-22 | Negotiation | on:advanceStage | - | (internal) | recordAdvanceDenied |
| T-DEAL-23 | Negotiation | on:win | guardCanWin | persisting | setPendingWin |
| T-DEAL-24 | Negotiation | on:win | - | (internal) | recordWinDenied |
| T-DEAL-25 | Negotiation | on:lose | guardCanLose | persisting | setPendingLose |
| T-DEAL-26 | Negotiation | on:lose | - | (internal) | recordLoseDenied |
| T-DEAL-27 | Negotiation | on:reopen | - | (internal) | recordReopenNotTerminal |
| T-DEAL-28 | Won | on:reopen | guardCanReopen | persisting | setPendingReopen |
| T-DEAL-29 | Won | on:reopen | - | (internal) | recordReopenDenied |
| T-DEAL-30 | Won | on:advanceStage | - | (internal) | recordTerminalRejected |
| T-DEAL-31 | Won | on:win | - | (internal) | recordTerminalRejected |
| T-DEAL-32 | Won | on:lose | - | (internal) | recordTerminalRejected |
| T-DEAL-33 | Lost | on:reopen | guardCanReopen | persisting | setPendingReopen |
| T-DEAL-34 | Lost | on:reopen | - | (internal) | recordReopenDenied |
| T-DEAL-35 | Lost | on:advanceStage | - | (internal) | recordTerminalRejected |
| T-DEAL-36 | Lost | on:win | - | (internal) | recordTerminalRejected |
| T-DEAL-37 | Lost | on:lose | - | (internal) | recordTerminalRejected |
| T-DEAL-38 | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-DEAL-39 | persisting | onDone:saveDeal | pendingIsQualified | Qualified | commitStage |
| T-DEAL-40 | persisting | onDone:saveDeal | pendingIsProposal | Proposal | commitStage |
| T-DEAL-41 | persisting | onDone:saveDeal | pendingIsNegotiation | Negotiation | commitStage |
| T-DEAL-42 | persisting | onDone:saveDeal | pendingIsWon | Won | commitStage, commitCloseDate |
| T-DEAL-43 | persisting | onDone:saveDeal | pendingIsLost | Lost | commitStage |
| T-DEAL-44 | persisting | onDone:saveDeal | - | rolledBack | recordRoutingError |
| T-DEAL-45 | persisting | onError:saveDeal | isErrLocked | persistRetry | recordError |
| T-DEAL-46 | persisting | onError:saveDeal | isErrConstraint | rolledBack | recordConstraint |
| T-DEAL-47 | persisting | onError:saveDeal | isErrDiskFull | rolledBack | recordDiskFull |
| T-DEAL-48 | persisting | onError:saveDeal | isErrTimeout | rolledBack | recordTimeout |
| T-DEAL-49 | persisting | onError:saveDeal | - | rolledBack | recordUnknownError |
| T-DEAL-50 | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-DEAL-51 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-DEAL-52 | rolledBack | always | priorIsLead | Lead | - |
| T-DEAL-53 | rolledBack | always | priorIsQualified | Qualified | - |
| T-DEAL-54 | rolledBack | always | priorIsProposal | Proposal | - |
| T-DEAL-55 | rolledBack | always | priorIsNegotiation | Negotiation | - |
| T-DEAL-56 | rolledBack | always | priorIsWon | Won | - |
| T-DEAL-57 | rolledBack | always | priorIsLost | Lost | - |

Total transitions (test cases): 57
