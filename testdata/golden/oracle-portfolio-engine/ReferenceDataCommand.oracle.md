# Generated transition oracle: `referenceDataCommand`

Generated from `ReferenceDataCommand.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.6.9 -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| idle | atomic | - | - |
| refreshing | atomic | - | - |
| upserting | atomic | - | - |
| building | atomic | - | - |
| succeeded | final | - | - |
| failed | final | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-REFE-01 | REFE-33a2ae | idle | on:refresh | - | refreshing | - |
| T-REFE-02 | REFE-6e63db | idle | on:upsert | - | upserting | - |
| T-REFE-03 | REFE-62d231 | idle | on:build | - | building | - |
| T-REFE-04 | REFE-cb1193 | refreshing | after:COMMAND_TIMEOUT | - | failed | cancelReferenceOperation, recordReferenceTimeout |
| T-REFE-05 | REFE-c51bb5 | refreshing | onDone:selectEligibleConstituents | - | succeeded | - |
| T-REFE-06 | REFE-6b30a5 | refreshing | onError:selectEligibleConstituents | - | failed | recordReferenceError |
| T-REFE-07 | REFE-8c9720 | upserting | after:COMMAND_TIMEOUT | - | failed | cancelReferenceOperation, recordReferenceTimeout |
| T-REFE-08 | REFE-cc283e | upserting | onDone:upsertSecurityByTicker | - | succeeded | - |
| T-REFE-09 | REFE-cc10bd | upserting | onError:upsertSecurityByTicker | - | failed | recordReferenceError |
| T-REFE-10 | REFE-f67aa3 | building | after:COMMAND_TIMEOUT | - | failed | cancelReferenceOperation, recordReferenceTimeout |
| T-REFE-11 | REFE-074a67 | building | onDone:buildCandidateSet | - | succeeded | - |
| T-REFE-12 | REFE-c60610 | building | onError:buildCandidateSet | - | failed | recordReferenceError |

Total transitions (test cases): 12
