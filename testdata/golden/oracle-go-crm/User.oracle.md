# Generated transition oracle: `user`

Generated from `User.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Active | atomic | - | - |
| Disabled | atomic | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |
| T-USER-02 | USER-2b2218 | Active | on:disable | - | (internal) | recordAuthorityDenied |
| T-USER-03 | USER-0ef83a | Active | on:enable | - | (internal) | recordAlreadyActive |
| T-USER-04 | USER-e59219 | Disabled | on:enable | guardAdminAuthority | persisting | setPendingEnable |
| T-USER-05 | USER-ffd41a | Disabled | on:enable | - | (internal) | recordAuthorityDenied |
| T-USER-06 | USER-799d7d | Disabled | on:disable | - | (internal) | recordAlreadyDisabled |
| T-USER-07 | USER-a986a8 | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-USER-08 | USER-930b15 | persisting | onDone:saveUser | pendingIsActive | Active | commitStatus |
| T-USER-09 | USER-dd6c98 | persisting | onDone:saveUser | pendingIsDisabled | Disabled | commitStatus |
| T-USER-10 | USER-7b324b | persisting | onDone:saveUser | - | rolledBack | recordRoutingError |
| T-USER-11 | USER-dde0a6 | persisting | onError:saveUser | isErrLocked | persistRetry | recordError |
| T-USER-12 | USER-d2cfe6 | persisting | onError:saveUser | isErrConstraint | rolledBack | recordConstraint |
| T-USER-13 | USER-8e0d4c | persisting | onError:saveUser | isErrDiskFull | rolledBack | recordDiskFull |
| T-USER-14 | USER-388821 | persisting | onError:saveUser | isErrTimeout | rolledBack | recordTimeout |
| T-USER-15 | USER-838e85 | persisting | onError:saveUser | - | rolledBack | recordUnknownError |
| T-USER-16 | USER-081a5d | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-USER-17 | USER-1c13da | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-USER-18 | USER-adccd9 | rolledBack | always | priorIsActive | Active | - |
| T-USER-19 | USER-7cf0fc | rolledBack | always | priorIsDisabled | Disabled | - |

Total transitions (test cases): 19
