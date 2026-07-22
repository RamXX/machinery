# Generated transition oracle: `marketDataFeed`

Generated from `MarketDataFeed.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.4-dev -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| closed | atomic | - | - |
| open | atomic | - | - |
| halfOpen | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-MARK-01 | MARK-acc7d7 | closed | on:failure | atThreshold | open | recordTrip |
| T-MARK-02 | MARK-9e6205 | closed | on:failure | - | closed | incFailures |
| T-MARK-03 | MARK-81fc92 | closed | on:success | - | closed | resetFailures |
| T-MARK-04 | MARK-609444 | open | after:COOLDOWN | - | halfOpen | - |
| T-MARK-05 | MARK-2bed99 | halfOpen | on:probeResult | probeSucceeded | closed | resetFailures |
| T-MARK-06 | MARK-775b8f | halfOpen | on:probeResult | - | open | recordTrip |

Total transitions (test cases): 6
