# Generated transition oracle: `user`

Generated from `User.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one test case.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Active | atomic | - | - |
| Disabled | atomic | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|
| T-USER-01 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |
| T-USER-02 | Active | on:disable | - | (internal) | recordAuthorityDenied |
| T-USER-03 | Active | on:enable | - | (internal) | recordAlreadyActive |
| T-USER-04 | Disabled | on:enable | guardAdminAuthority | persisting | setPendingEnable |
| T-USER-05 | Disabled | on:enable | - | (internal) | recordAuthorityDenied |
| T-USER-06 | Disabled | on:disable | - | (internal) | recordAlreadyDisabled |
| T-USER-07 | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-USER-08 | persisting | onDone:saveUser | pendingIsActive | Active | commitStatus |
| T-USER-09 | persisting | onDone:saveUser | pendingIsDisabled | Disabled | commitStatus |
| T-USER-10 | persisting | onDone:saveUser | - | rolledBack | recordRoutingError |
| T-USER-11 | persisting | onError:saveUser | isErrLocked | persistRetry | recordError |
| T-USER-12 | persisting | onError:saveUser | isErrConstraint | rolledBack | recordConstraint |
| T-USER-13 | persisting | onError:saveUser | isErrDiskFull | rolledBack | recordDiskFull |
| T-USER-14 | persisting | onError:saveUser | isErrTimeout | rolledBack | recordTimeout |
| T-USER-15 | persisting | onError:saveUser | - | rolledBack | recordUnknownError |
| T-USER-16 | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-USER-17 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-USER-18 | rolledBack | always | priorIsActive | Active | - |
| T-USER-19 | rolledBack | always | priorIsDisabled | Disabled | - |

Total transitions (test cases): 19
