# Generated transition oracle: `commandExecution`

Generated from `CommandExecution.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one test case.

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

| test id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|
| T-COMM-01 | Parsing | always | guardParseOk | Opening | captureArgs |
| T-COMM-02 | Parsing | always | - | ValidationFailed | recordParseError |
| T-COMM-03 | Opening | after:openTimeout | - | DBError | recordTimeout |
| T-COMM-04 | Opening | onDone:openDatabase | - | ResolvingSession | captureTx |
| T-COMM-05 | Opening | onError:openDatabase | isErrLocked | DBLocked | recordError |
| T-COMM-06 | Opening | onError:openDatabase | isErrCorrupt | Corrupt | recordCorrupt |
| T-COMM-07 | Opening | onError:openDatabase | isErrUnavailable | DBError | recordUnavailable |
| T-COMM-08 | Opening | onError:openDatabase | - | DBError | recordOpenError |
| T-COMM-09 | DBLocked | after:dbRetryBackoff | phaseIsOpen | Opening | incrementRetries |
| T-COMM-10 | DBLocked | after:dbRetryBackoff | phaseIsExecute | Executing | incrementRetries |
| T-COMM-11 | DBLocked | always | retriesExhausted | DBError | recordLockExhausted |
| T-COMM-12 | ResolvingSession | after:sessionResolveTimeout | - | DBError | recordTimeout |
| T-COMM-13 | ResolvingSession | onDone:resolveSession | - | Authorizing | captureActor |
| T-COMM-14 | ResolvingSession | onError:resolveSession | isErrNoSession | Denied | recordNeedLogin |
| T-COMM-15 | ResolvingSession | onError:resolveSession | isErrExpired | Denied | recordNeedLogin |
| T-COMM-16 | ResolvingSession | onError:resolveSession | isErrLocked | DBLocked | recordError |
| T-COMM-17 | ResolvingSession | onError:resolveSession | - | DBError | recordSessionError |
| T-COMM-18 | Authorizing | always | guardAuthorized | Executing | recordAllowed |
| T-COMM-19 | Authorizing | always | - | Denied | recordDenyReason |
| T-COMM-20 | Executing | after:queryTimeout | - | DBError | ensureRolledBack, recordTimeout |
| T-COMM-21 | Executing | onDone:executeInTx | - | Rendering | captureResult |
| T-COMM-22 | Executing | onError:executeInTx | isErrConstraint | ValidationFailed | ensureRolledBack, recordConstraint |
| T-COMM-23 | Executing | onError:executeInTx | isErrLocked | DBLocked | ensureRolledBack, recordError |
| T-COMM-24 | Executing | onError:executeInTx | isErrConflict | DBLocked | ensureRolledBack, recordConflict |
| T-COMM-25 | Executing | onError:executeInTx | isErrDiskFull | DBError | ensureRolledBack, recordDiskFull |
| T-COMM-26 | Executing | onError:executeInTx | isErrTimeout | DBError | ensureRolledBack, recordTimeout |
| T-COMM-27 | Executing | onError:executeInTx | - | DBError | ensureRolledBack, recordExecuteError |
| T-COMM-28 | Rendering | always | - | Done | - |

Total transitions (test cases): 28
