# Generated transition oracle: `recommendationRun`

Generated from `RecommendationRun.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Collecting | atomic | - | - |
| Optimizing | atomic | - | - |
| collectRetry | atomic | - | - |
| Ready | final | publishReady | - |
| Failed | final | publishFailure | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-RECO-01 | RECO-c7bb09 | Collecting | after:FETCH_TIMEOUT | - | collectRetry | - |
| T-RECO-02 | RECO-f89da8 | Collecting | onDone:fetchPrices | - | Optimizing | - |
| T-RECO-03 | RECO-040944 | Collecting | onError:fetchPrices | - | collectRetry | - |
| T-RECO-04 | RECO-c85bd8 | Optimizing | after:OPTIMIZE_TIMEOUT | - | Failed | - |
| T-RECO-05 | RECO-d6fcf9 | Optimizing | onDone:optimize | - | Ready | recordPortfolio |
| T-RECO-06 | RECO-ed98c7 | Optimizing | onError:optimize | - | Failed | - |
| T-RECO-07 | RECO-0d730c | collectRetry | after:RETRY_BACKOFF | - | Collecting | incRetries |
| T-RECO-08 | RECO-61506b | collectRetry | always | retriesExhausted | Failed | - |

Total transitions (test cases): 8
