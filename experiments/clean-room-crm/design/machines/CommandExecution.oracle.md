# Generated transition oracle: `commandExecution`

Generated from `CommandExecution.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| opening | atomic | - | - |
| authenticating | atomic | - | - |
| authorizing | atomic | - | - |
| executing | atomic | - | - |
| done | final | refreshSession | - |
| rejected | final | emitRejection | - |
| refused | final | emitBusyNotice | - |
| failedCorrupt | final | emitCorruptAlert | - |
| failedError | final | emitError | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-COMM-01 | COMM-79e794 | opening | after:OPEN_TIMEOUT | - | failedError | - |
| T-COMM-02 | COMM-8950a7 | opening | onDone:openDb | - | authenticating | - |
| T-COMM-03 | COMM-5a71e9 | opening | onError:openDb | isCorrupt | failedCorrupt | - |
| T-COMM-04 | COMM-fe4a12 | opening | onError:openDb | - | failedError | - |
| T-COMM-05 | COMM-aa1d7f | authenticating | after:AUTH_TIMEOUT | - | failedError | - |
| T-COMM-06 | COMM-9fe39b | authenticating | onDone:loadSession | sessionActive | authorizing | - |
| T-COMM-07 | COMM-a958de | authenticating | onDone:loadSession | - | rejected | - |
| T-COMM-08 | COMM-7779df | authenticating | onError:loadSession | - | failedError | - |
| T-COMM-09 | COMM-62f083 | authorizing | after:AUTHZ_TIMEOUT | - | failedError | - |
| T-COMM-10 | COMM-70240a | authorizing | onDone:checkScope | permitted | executing | - |
| T-COMM-11 | COMM-913535 | authorizing | onDone:checkScope | - | rejected | - |
| T-COMM-12 | COMM-4afbeb | authorizing | onError:checkScope | - | failedError | - |
| T-COMM-13 | COMM-1782af | executing | after:EXEC_TIMEOUT | - | refused | - |
| T-COMM-14 | COMM-0a38db | executing | onDone:execute | - | done | - |
| T-COMM-15 | COMM-c6fd71 | executing | onError:execute | isConflict | refused | - |
| T-COMM-16 | COMM-3e8561 | executing | onError:execute | isRejected | rejected | - |
| T-COMM-17 | COMM-11031f | executing | onError:execute | - | failedError | - |

Total transitions (test cases): 17
