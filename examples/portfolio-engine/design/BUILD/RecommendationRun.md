# BUILD shard: RecommendationRun (run pipeline, optimizer, reference data)

Shard of the manifest root `design/BUILD.md` (Mode: manifest). The root carries the glossary, the
domain model, the Architecture Contract, the traceability matrix, the shared toolchain, the
state-migration protocol, and the hard-TDD protocol; this shard carries the RecommendationRun
component: its behavior, its test specification, and its build-plan milestones (M0, M1, M3, M5 of
the root milestone map). The optimizer and the reference-data builds have no machine of their own;
their plan and tests live here because the run pipeline is what invokes them.

## 5. Behavior

### RecommendationRun (`design/machines/RecommendationRun.machine.json`)

A forward pipeline: Collecting invokes the price fetch (through the feed breaker); on success it
moves to Optimizing, which invokes the optimizer; success records the portfolio and reaches Ready.
A fetch failure or timeout retries a bounded number of times (collectRetry) then fails; an optimizer
failure or timeout fails directly. Ready and Failed are terminal. Single writer, so no persist
overlay on the run. Named-unit contracts and failure catalog:
`design/machines/RecommendationRun.matrix.md` (1 guard, 4 actions, 2 actors). The `fetchPrices` and
`optimize` actors are integration/side-effect contracts, not derivable from transition tests.

### Pure logic driven by the run (no machine)

- **pf.optimizer**: the pure 16-of-N min-drawdown transform, invoked from Optimizing. Its contract
  is the optimizer invariants in the root traceability matrix (`portfolio-size-16`,
  `portfolio-holdings-deduped`, `portfolio-from-candidates`, `portfolio-has-drawdown`,
  `holding-weight-nonneg`, `holding-weights-sum-full`).
- **Reference data**: Index refresh (`index-top-30`), Security upsert (`ticker-unique`),
  CandidateSet build (`candidate-deduped`, `candidate-from-top-30`); plus `backup`/`restore` and
  the corruption abort path in `pf.repo`.

## 7. Test specification

The transition test spec IS the generated `design/machines/RecommendationRun.oracle.md` (8 rows).
Do not restate the table; tests key on the STABLE id, never the row number:

| stable id | transition |
|---|---|
| RECO-c7bb09 | Collecting, fetch timeout, to collectRetry |
| RECO-f89da8 | Collecting, fetch done, to Optimizing |
| RECO-040944 | Collecting, fetch error, to collectRetry |
| RECO-c85bd8 | Optimizing, optimize timeout, to Failed |
| RECO-d6fcf9 | Optimizing, optimize done, to Ready (recordPortfolio) |
| RECO-ed98c7 | Optimizing, optimize error, to Failed |
| RECO-0d730c | collectRetry, backoff elapsed, to Collecting (incRetries) |
| RECO-61506b | collectRetry, retriesExhausted, to Failed |

### Guard-branch completeness

`retriesExhausted` = (retries >= MaxRetries). One test below the bound retries (RECO-0d730c), one
at the bound routes to Failed (RECO-61506b). No conjunction guards, so no falsifying triples.

### Named-unit test plan

Per the matrix: the pipeline actions are unit tests over context; the actors are integration tests:
`fetchPrices` against a contract-tested market-data fake plus a breaker-open fixture; `optimize`
runs the real optimizer on a fixed, deterministic price fixture.

### Property tests owned by this shard

`PROP-run-ready-has-portfolio` derivations aside (machine-enforced, cited by stable id in the root
matrix), this shard owns the optimizer and reference-data properties: `PROP-portfolio-size-16`,
`PROP-portfolio-holdings-deduped`, `PROP-portfolio-from-candidates`, `PROP-portfolio-has-drawdown`,
`PROP-holding-weight-nonneg`, `PROP-holding-weights-sum-full`, `PROP-index-top-30`,
`PROP-ticker-unique`, `PROP-candidate-deduped`, `PROP-candidate-from-top-30`.

## 8. State migration

`RecommendationRun` persists its `status`; no persisted instances yet. The protocol is the root's
section 8; collectRetry is never persisted, so renaming it needs no migration.

## 9. Build plan

**M0 - Walking skeleton (thinnest end-to-end slice through one real boundary).** `pf recommend` over a
tiny two-index fixture with cached prices: build a `CandidateSet` (dedup), start a
`RecommendationRun`, fetch prices through the feed (breaker closed), optimize a 16-of-N fixture,
reach Ready with a `Portfolio`. Prove the topology, one real DuckDB write, and one real optimizer
run. The skeleton instantiates the NFR-record mechanisms every later milestone copies: the distinct
exit code and loud message per residual failure, the market-data key read from env (never logged),
and the 0600 store file. DoD: `RECO-f89da8` then `RECO-d6fcf9` pass against a real store and a
forced feed failure drives `RECO-040944` then the bounded retry; contract tests for the crossed
boundaries green; no cross-boundary violation (G4-import clean); the formal suite still green.

**M1 - Run pipeline slice.** All RecommendationRun transitions, `run-ready-has-portfolio`,
`run-forward-only`, `run-terminal-absorbing`, and the collectRetry bound; green before the next.
DoD: all 8 RecommendationRun oracle rows covered by stable id, the three listed invariants
property-tested, its contract tests green, G4-import clean, formal suite still green.

**M3 - Optimizer slice.** `portfolio-size-16`, `portfolio-holdings-deduped`, `portfolio-from-candidates`,
`portfolio-has-drawdown`, `holding-weight-nonneg`, `holding-weights-sum-full` as property tests over
the pure optimizer. DoD: the six listed invariants property-tested over the pure optimizer, its
contract tests green, G4-import clean, formal suite still green.

**M5 - Reference-data and operations slice.** Index refresh (`index-top-30`), Security upsert
(`ticker-unique`), CandidateSet build (`candidate-deduped`, `candidate-from-top-30`), then `backup`/
`restore` and the corruption abort path. DoD: the listed invariants property-tested, a backup then
restore round trip green, the corruption abort loud, its contract tests green, G4-import clean,
formal suite still green.
