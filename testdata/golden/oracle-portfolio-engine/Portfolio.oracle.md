# Generated transition oracle: `portfolio`

Generated from `Portfolio.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Proposed | atomic | - | - |
| UnderReview | atomic | - | - |
| Accepted | atomic | - | - |
| Rejected | atomic | - | - |
| committing | atomic | - | - |
| commitRetry | atomic | - | - |
| reverted | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-PORT-01 | PORT-27d66f | Proposed | on:advance | - | committing | setPendingAdvance |
| T-PORT-02 | PORT-2bf44c | Proposed | on:accept | canDecide | committing | setPendingAccept |
| T-PORT-03 | PORT-a41039 | Proposed | on:reject | canDecide | committing | setPendingReject |
| T-PORT-04 | PORT-ddb44c | UnderReview | on:accept | canDecide | committing | setPendingAccept |
| T-PORT-05 | PORT-351dec | UnderReview | on:reject | canDecide | committing | setPendingReject |
| T-PORT-06 | PORT-db3bb9 | Accepted | on:reopen | canReopen | committing | setPendingReopen |
| T-PORT-07 | PORT-9facf7 | Rejected | on:reopen | canReopen | committing | setPendingReopen |
| T-PORT-08 | PORT-5e6be0 | committing | after:COMMIT_TIMEOUT | - | commitRetry | - |
| T-PORT-09 | PORT-f43140 | committing | onDone:persistDecision | pendingIsUnderReview | UnderReview | commit |
| T-PORT-10 | PORT-d1647b | committing | onDone:persistDecision | pendingIsAccepted | Accepted | commit, recordAccepted |
| T-PORT-11 | PORT-fb8c92 | committing | onDone:persistDecision | pendingIsRejected | Rejected | commit |
| T-PORT-12 | PORT-40b6e7 | committing | onError:persistDecision | isRetriable | commitRetry | - |
| T-PORT-13 | PORT-c4a186 | committing | onError:persistDecision | - | reverted | - |
| T-PORT-14 | PORT-f6e220 | commitRetry | after:RETRY_BACKOFF | - | committing | incRetries |
| T-PORT-15 | PORT-cba032 | commitRetry | always | retriesExhausted | reverted | - |
| T-PORT-16 | PORT-8c0400 | reverted | always | priorIsProposed | Proposed | - |
| T-PORT-17 | PORT-3cb0b6 | reverted | always | priorIsUnderReview | UnderReview | - |
| T-PORT-18 | PORT-53d34b | reverted | always | priorIsAccepted | Accepted | - |
| T-PORT-19 | PORT-3390a7 | reverted | always | priorIsRejected | Rejected | - |

Total transitions (test cases): 19
