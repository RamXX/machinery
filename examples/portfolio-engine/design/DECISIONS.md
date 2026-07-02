# DECISIONS

Design decisions for the Drawdown Portfolio Recommender, and how it exercises the tooling fixes.

## Why this domain

- Chosen to be entirely unlike the CRM: a batch optimization pipeline over market data, in Python,
  not a CRUD-plus-lifecycle app in Go. It stresses the fixed tools on a fresh vocabulary so any
  hardcoded CRM leak would surface.

## Phase 0 / 1

- Q: Target language? A: Python. Why: the natural language for a numerical/optimization tool; also
  different from the CRM (Go), so the C4 and realization notes differ.
- Q: Local store? A: DuckDB (embedded columnar file), via the `duckdb` client, sole-imported by
  pf.repo. Why: suits price series; local; different from the CRM's graph store.
- Q: Objective? A: minimize historical maximum drawdown over a lookback; select exactly 16 of the
  deduped top-30-per-index universe. Straight from the prompt.
- Q: maxDrawdown / weight types? A: integers in basis points. Why: keeps the model integer-typed and
  the persisted schema exact.
- Q: Does the run auto-start collecting? A: yes; a run is created in Collecting and driven to a
  terminal state by the one process. Why: a batch job, not an interactive record; no external events.

## Phase 2

- Q: Boundaries? A: pf.cli -> pf.app -> {pf.domain, pf.optimizer, pf.feed, pf.repo} -> pf.model;
  pf.feed sole importer of the market-data client, pf.repo sole importer of DuckDB. Why: keeps domain
  and optimizer pure; isolates each external behind one boundary.
- Q: Event-contract table? A: N/A. Why: one synchronous process per command, no bus.

## Phase 3 and the formal layer (exercising the fixes)

- Q: Which machines? A: RecommendationRun (lifecycle), Portfolio (lifecycle), MarketDataFeed
  (operational). Index/Security/CandidateSet/Holding/Optimizer are pure data or pure transforms,
  waived.
- Q: Portfolio persist-overlay names? A: committing / commitRetry / reverted, NOT the
  persisting/persistRetry/rolledBack defaults. Why: to prove the linear-lifecycle pattern reads its
  overlay names from the annotation (fix #1); the default names never appear in this design.
- Q: How is the run pipeline proved formally? A: the new terminal-lifecycle pattern
  (RecommendationRun.semantics.yaml), proving completeness (a Ready run has its portfolio),
  terminal absorption, and termination. Why: the run is a forward pipeline, not a win/lose/reopen
  lifecycle; this is the pattern added to fix the lumpy-coverage finding.
- Q: Where is _exhaustive used, and where avoided? A: RecommendationRun's collectRetry and the
  MarketDataFeed breaker avoid it (after-escape and event-driven transitions, which TLC verifies);
  only Portfolio's reverted rollback router uses it (its guard set is provably total). Why: to
  demonstrate the preferred fallback pattern and confine _exhaustive to a genuinely-total case; G3
  surfaces the one remaining use as a warn (fix #3).
- Q: Which invariants are machine-enforced vs attested? A: 6 unit-backed (run-ready-has-portfolio,
  portfolio-accept-role, portfolio-reopen-role, portfolio-accepted-has-date, portfolio-review-forward,
  feed-circuit-breaks) and 12 attested/structural (the optimizer-output and reference-data rules).
  Gx now reports the split (fix #2), so "enforced" does not overstate verification.
