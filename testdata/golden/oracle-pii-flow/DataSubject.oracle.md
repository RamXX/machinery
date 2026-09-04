# Generated transition oracle: `datasubject`

Generated from `DataSubject.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.6.5 -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Active | atomic | - | - |
| Erased | final | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-DATA-01 | DATA-f61f71 | Active | on:erase | - | Erased | recordErasure |

Total transitions (test cases): 1
