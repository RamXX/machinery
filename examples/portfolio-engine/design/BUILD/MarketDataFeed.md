# BUILD shard: MarketDataFeed (circuit breaker)

Shard of the manifest root `design/BUILD.md` (Mode: manifest). The root carries the glossary, the
domain model, the Architecture Contract, the traceability matrix, the shared toolchain, the
state-migration protocol, and the hard-TDD protocol; this shard carries the MarketDataFeed
component: its behavior, its test specification, and its build-plan milestone (M2 of the root
milestone map).

## 5. Behavior

### MarketDataFeed (`design/machines/MarketDataFeed.machine.json`, `_role: operational`)

A circuit breaker over the provider: closed (calls flow, failures counted), open (calls fast-fail
after a threshold trip), halfOpen (one trial probe recloses or reopens). Enforces
`feed-circuit-breaks`. Named-unit contracts: `design/machines/MarketDataFeed.matrix.md` (2 guards,
3 actions, no actors). The breaker state is in-memory per invocation and never persisted.

## 7. Test specification

The transition test spec IS the generated `design/machines/MarketDataFeed.oracle.md` (6 rows). Do
not restate the table; tests key on the STABLE id, never the row number:

| stable id | transition |
|---|---|
| MARK-acc7d7 | closed, failure, atThreshold, to open (recordTrip) |
| MARK-9e6205 | closed, failure, to closed (incFailures) |
| MARK-81fc92 | closed, success, to closed (resetFailures) |
| MARK-609444 | open, cooldown elapsed, to halfOpen |
| MARK-2bed99 | halfOpen, probeResult, probeSucceeded, to closed (resetFailures) |
| MARK-775b8f | halfOpen, probeResult, to open (recordTrip) |

### Guard-branch completeness

- `atThreshold` = (failures + 1 >= threshold). One test just below the threshold stays closed
  (MARK-9e6205), one at the threshold trips to open (MARK-acc7d7). Covers `feed-circuit-breaks`.
- `probeSucceeded`: one probe-success test recloses (MARK-2bed99), one probe-failure test reopens
  (MARK-775b8f).

No conjunction guards, so no falsifying triples.

### Named-unit test plan

Per the matrix: `atThreshold`/`probeSucceeded` and the counter actions are unit tests over the
breaker object; the breaker-open fast-fail path is exercised as a fixture by the `fetchPrices`
integration tests in `BUILD/RecommendationRun.md`. Contract test at feed->mkt: error mapping and
breaker behavior (a tripped breaker fast-fails without a network call).

## 8. State migration

N/A - the breaker state is in-memory per command invocation and never persisted (root section 8).

## 9. Build plan

The design-wide walking skeleton is M0 in `BUILD/RecommendationRun.md`; it crosses this shard's
boundary (fetch through the feed, breaker closed) before M2 begins. Walking skeleton: N/A - the
design-wide skeleton lives in the RecommendationRun shard (M0).

**M2 - Feed breaker slice.** MarketDataFeed closed/open/halfOpen, `feed-circuit-breaks`. DoD: all 6
MarketDataFeed oracle rows covered by stable id (MARK-acc7d7 through MARK-775b8f above),
`feed-circuit-breaks` property-tested, its contract tests green, G4-import clean, formal suite
still green.
