# Generated transition oracle: `commandExecution`

Generated from `CommandExecution.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Parsing | atomic | - | - |
| Opening | atomic | setPhaseOpen | - |
| DBLocked | atomic | - | - |
| ResolvingSession | atomic | - | - |
| Authorizing | atomic | - | - |
| Executing | atomic | setPhaseExecute | - |
| Rendering | atomic | renderOutput | - |
| Done | final | recordSuccessExit | - |
| Denied | final | recordDeniedExit | - |
| ValidationFailed | final | recordValidationExit | - |
| DBError | final | recordDBErrorExit | - |
| Corrupt | final | recordCorruptExit | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-COMM-01 | COMM-44671c | Parsing | always | guardParseOk | Opening | captureArgs |
| T-COMM-02 | COMM-6f50f3 | Parsing | always | - | ValidationFailed | recordParseError |
| T-COMM-03 | COMM-5e3106 | Opening | after:openTimeout | - | DBError | recordTimeout |
| T-COMM-04 | COMM-5bc5e0 | Opening | onDone:openDatabase | - | ResolvingSession | captureTx |
| T-COMM-05 | COMM-ea53cf | Opening | onError:openDatabase | isErrLocked | DBLocked | recordError |
| T-COMM-06 | COMM-b9aee2 | Opening | onError:openDatabase | isErrCorrupt | Corrupt | recordCorrupt |
| T-COMM-07 | COMM-8a2a55 | Opening | onError:openDatabase | isErrUnavailable | DBError | recordUnavailable |
| T-COMM-08 | COMM-343b9c | Opening | onError:openDatabase | - | DBError | recordOpenError |
| T-COMM-09 | COMM-215300 | DBLocked | after:dbRetryBackoff | phaseIsOpen | Opening | incrementRetries |
| T-COMM-10 | COMM-71162c | DBLocked | after:dbRetryBackoff | phaseIsExecute | Executing | incrementRetries |
| T-COMM-11 | COMM-00c530 | DBLocked | always | retriesExhausted | DBError | recordLockExhausted |
| T-COMM-12 | COMM-35a500 | ResolvingSession | after:sessionResolveTimeout | - | DBError | recordTimeout |
| T-COMM-13 | COMM-968d17 | ResolvingSession | onDone:resolveSession | - | Authorizing | captureActor |
| T-COMM-14 | COMM-ed4f93 | ResolvingSession | onError:resolveSession | isErrNoSession | Denied | recordNeedLogin |
| T-COMM-15 | COMM-cc7919 | ResolvingSession | onError:resolveSession | isErrExpired | Denied | recordNeedLogin |
| T-COMM-16 | COMM-22d79f | ResolvingSession | onError:resolveSession | isErrLocked | DBLocked | recordError |
| T-COMM-17 | COMM-2151b8 | ResolvingSession | onError:resolveSession | - | DBError | recordSessionError |
| T-COMM-18 | COMM-8c204a | Authorizing | always | guardAuthorized | Executing | recordAllowed |
| T-COMM-19 | COMM-7f1685 | Authorizing | always | - | Denied | recordDenyReason |
| T-COMM-20 | COMM-84ddf1 | Executing | after:queryTimeout | - | DBError | ensureRolledBack, recordTimeout |
| T-COMM-21 | COMM-5d7be9 | Executing | onDone:executeInTx | - | Rendering | captureResult |
| T-COMM-22 | COMM-ec7aeb | Executing | onError:executeInTx | isErrConstraint | ValidationFailed | ensureRolledBack, recordConstraint |
| T-COMM-23 | COMM-d6cfde | Executing | onError:executeInTx | isErrLocked | DBLocked | ensureRolledBack, recordError |
| T-COMM-24 | COMM-8be203 | Executing | onError:executeInTx | isErrConflict | DBLocked | ensureRolledBack, recordConflict |
| T-COMM-25 | COMM-40743b | Executing | onError:executeInTx | isErrDiskFull | DBError | ensureRolledBack, recordDiskFull |
| T-COMM-26 | COMM-cb11e8 | Executing | onError:executeInTx | isErrTimeout | DBError | ensureRolledBack, recordTimeout |
| T-COMM-27 | COMM-0b53b2 | Executing | onError:executeInTx | - | DBError | ensureRolledBack, recordExecuteError |
| T-COMM-28 | COMM-121e81 | Rendering | always | - | Done | - |

Total transitions (test cases): 28
