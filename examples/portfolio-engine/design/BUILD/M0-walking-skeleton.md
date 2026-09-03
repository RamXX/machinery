# M0 - Walking skeleton

## Outcome

`pf recommend` accepts a fixed two-index fixture, builds a deduplicated candidate set, fetches
cached prices, runs the real optimizer, persists one Ready recommendation and Proposed portfolio
in DuckDB, and prints the portfolio id. The demo also forces one feed failure and observes retry.

## Domain context

Use CandidateSet securities as the only optimizer input. RecommendationRun moves from Collecting
to Optimizing to Ready and Ready requires a portfolio (`run-ready-has-portfolio`). Portfolio starts
Proposed. Preserve `candidate-deduped`, `candidate-from-top-30`, `portfolio-size-16`,
`portfolio-holdings-deduped`, and full non-negative weights. IDs and status values are exactly those
in `domain.modelith.yaml`; do not introduce parallel schemas.

## Architecture context

Own `pf.cli` command parsing, `pf.app` orchestration, the `pf.feed` port, pure `pf.optimizer`, and
`pf.repo` DuckDB persistence only. Dependency flow is CLI -> app -> domain ports, with adapters
implementing ports. Use one real temporary DuckDB file. Read the market-data key from environment,
never log it, create store files as 0600, and map each residual failure to a distinct loud exit.

## Behavior and oracles

Parse `machines/RecommendationRun.oracle.md` in the conformance test. Prove `RECO-f89da8`
(fetch done to Optimizing), `RECO-d6fcf9` (optimize done to Ready and recordPortfolio), and forced
failure `RECO-040944` followed by bounded retry. Assert next state and ordered actions for each row.

## TDD and implementation

First write the oracle parser/conformance test and a CLI acceptance test over a fixed cached-price
fixture. Add compile-only ports and stubs, prove the suite fails on missing behavior, then lock the
tests. Implement the narrow path in `pf/domain`, `pf/app`, `pf/feed`, `pf/optimizer`, `pf/repo`, and
`pf/cli`. Run `pytest`, formatter/linter checks, and `machinery check design --impl impl`.

## Risks and recovery

Bound retry count and all provider calls; never fall back to live unpinned market data in tests.
Rollback a failed transaction so neither a half-written run nor portfolio remains. A repeated
command must resolve its idempotency key to the same durable result.

## Acceptance

Exercise the demo against a fresh real DuckDB file, capture the three stable-id results, failure
exit, permissions, tests, import check, and formal verification. Record review evidence in
`acceptance/M0.yaml` against one commit before closing M0.
