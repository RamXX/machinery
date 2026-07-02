# Generated transition oracle: `deal`

Generated from `Deal.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Prospecting | atomic | - | - |
| Qualification | atomic | - | - |
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
| T-DEAL-01 | DEAL-baf45f | Prospecting | on:advance | - | persisting | setPendingAdvance |
| T-DEAL-02 | DEAL-be9325 | Prospecting | on:win | - | persisting | setPendingWin |
| T-DEAL-03 | DEAL-275c4a | Prospecting | on:lose | - | persisting | setPendingLose |
| T-DEAL-04 | DEAL-55e7f1 | Qualification | on:advance | - | persisting | setPendingAdvance |
| T-DEAL-05 | DEAL-3d0d86 | Qualification | on:win | - | persisting | setPendingWin |
| T-DEAL-06 | DEAL-305635 | Qualification | on:lose | - | persisting | setPendingLose |
| T-DEAL-07 | DEAL-daac03 | Proposal | on:advance | - | persisting | setPendingAdvance |
| T-DEAL-08 | DEAL-df4442 | Proposal | on:win | - | persisting | setPendingWin |
| T-DEAL-09 | DEAL-e16eea | Proposal | on:lose | - | persisting | setPendingLose |
| T-DEAL-10 | DEAL-3bbe10 | Negotiation | on:win | - | persisting | setPendingWin |
| T-DEAL-11 | DEAL-b5154b | Negotiation | on:lose | - | persisting | setPendingLose |
| T-DEAL-12 | DEAL-e5780c | Won | on:reopen | canReopen | persisting | setPendingReopen |
| T-DEAL-13 | DEAL-ed5b89 | Lost | on:reopen | canReopen | persisting | setPendingReopen |
| T-DEAL-14 | DEAL-ca3632 | persisting | after:PERSIST_TIMEOUT | - | persistRetry | - |
| T-DEAL-15 | DEAL-07bddb | persisting | onDone:persist | pendingIsQualification | Qualification | commit |
| T-DEAL-16 | DEAL-b9f970 | persisting | onDone:persist | pendingIsProposal | Proposal | commit |
| T-DEAL-17 | DEAL-dc6531 | persisting | onDone:persist | pendingIsNegotiation | Negotiation | commit |
| T-DEAL-18 | DEAL-798b5a | persisting | onDone:persist | pendingIsWon | Won | commit, recordClose |
| T-DEAL-19 | DEAL-643e7d | persisting | onDone:persist | pendingIsLost | Lost | commit, recordClose |
| T-DEAL-20 | DEAL-a24441 | persisting | onError:persist | isRetriable | persistRetry | - |
| T-DEAL-21 | DEAL-41781f | persisting | onError:persist | - | rolledBack | - |
| T-DEAL-22 | DEAL-d52f3b | persistRetry | after:RETRY_BACKOFF | - | persisting | incrementRetries |
| T-DEAL-23 | DEAL-8c9948 | persistRetry | always | retriesExhausted | rolledBack | - |
| T-DEAL-24 | DEAL-aea69a | rolledBack | always | priorIsProspecting | Prospecting | - |
| T-DEAL-25 | DEAL-de6908 | rolledBack | always | priorIsQualification | Qualification | - |
| T-DEAL-26 | DEAL-97c3ea | rolledBack | always | priorIsProposal | Proposal | - |
| T-DEAL-27 | DEAL-8a4caf | rolledBack | always | priorIsNegotiation | Negotiation | - |
| T-DEAL-28 | DEAL-9b6ee7 | rolledBack | always | priorIsWon | Won | - |
| T-DEAL-29 | DEAL-21905a | rolledBack | always | priorIsLost | Lost | - |

Total transitions (test cases): 29
