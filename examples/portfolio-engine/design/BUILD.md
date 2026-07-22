# BUILD: Drawdown Portfolio Recommender

Mode: manifest (root of a sharded design; shards in design/BUILD/<Component>.md).

This root is the entry-point manifest over `design/` and the shards under `design/BUILD/`. It
carries what every shard shares: the glossary, the domain model, the Architecture Contract, the
traceability matrix, the cross-context test spec, the shared toolchain and pins, the
state-migration protocol, and the milestone map. Each shard carries the behavior, the test
specification, and the build plan for one stateful component:

- `BUILD/RecommendationRun.md` (the run pipeline, plus the optimizer and reference-data logic it
  drives)
- `BUILD/Portfolio.md` (the review lifecycle and its commit overlay)
- `BUILD/MarketDataFeed.md` (the circuit breaker)

The zero-context claim applies to the design tree as a whole; self-containment applies per shard.
The machine JSON and the transition tables are referenced, never pasted; those files are what the
deterministic gates check.

## 1. Purpose and scope

A local command-line tool, written in Python, that recommends a stock portfolio. It draws the top 30
constituents of each configured market index, dedupes them into a candidate universe, collects the
price history for every candidate, and selects exactly 16 stocks minimizing historical maximum
drawdown. A produced portfolio is reviewed by a manager and accepted or rejected. There is no
server; all state lives in a local DuckDB file; the one network dependency is a market-data provider,
reached through a circuit breaker.

In scope: index constituent refresh, candidate deduplication, price collection with bounded retry, the
16-of-N min-drawdown optimization, the run lifecycle, the portfolio review lifecycle, a market-data
circuit breaker, and backup/restore. Out of scope: order execution or trading, real-time streaming,
multi-user servers, and any recommendation objective other than lowest maximum drawdown.

## 2. Glossary

- **Index**: a named market index with a ranked constituent list; only its top 30 by rank are
  eligible candidates.
- **Security**: a tradable stock with a unique ticker.
- **Constituent / Candidate / Source**: a security's membership in an index; a security in the
  optimization universe; an index a candidate set was drawn from.
- **CandidateSet**: the deduped union of the top 30 of every configured index, as of a date.
- **RecommendationRun (Run)**: one optimization job: collect prices, optimize, end Ready or Failed.
- **Portfolio**: a recommended set of exactly 16 holdings with a computed maximum drawdown, reviewed
  by a manager.
- **Holding**: one position: a candidate security and its weight (basis points).
- **Analyst / Manager / Admin**: the roles. An Analyst starts runs; a Manager (or Admin) reviews and
  decides portfolios.
- **maximum drawdown**: the largest peak-to-trough decline of a portfolio's value over the lookback
  window; the objective the optimizer minimizes.

## 3. Domain model (the what)

Source of truth: `design/domain.modelith.yaml` (lints clean, 0/0). Rendered: `design/domain.modelith.md`.

### Entities and relationships

```mermaid
erDiagram
    CandidateSet {}
    Holding {}
    Index {}
    Portfolio {}
    RecommendationRun {}
    Security {}
    CandidateSet }o--o{ Index : "Source"
    CandidateSet }o--o{ Security : "Candidate"
    Holding }o--|| Security : "Candidate"
    Index }o--o{ Security : "Constituent"
    Portfolio ||--o{ Holding : "Candidate"
    RecommendationRun }o--|| CandidateSet : "n:1"
    RecommendationRun ||--|| Portfolio : "Candidate"
```

### Data dictionary (the one canonical schema)

Every persisted row also carries a `version` integer for the optimistic lock where the placement
table (section 4) says so.

- **Index**: `id string`, `name string`, `provider string`, `asOf timestamp`; constituents (ranked)
  to Security (n:n).
- **Security**: `ticker string` (unique), `name string`, `sector string`.
- **CandidateSet**: `id string`, `asOf timestamp`, `size integer` (derived: distinct candidate
  count after dedup); Source to Index (n:n), Candidate to Security (n:n).
- **RecommendationRun**: `id string`, `requestedAt timestamp`, `lookbackDays integer`,
  `status RunStatus`; to CandidateSet (n:1), to Portfolio (1:1).
- **Portfolio**: `id string`, `maxDrawdown integer` (basis points), `status PortfolioStatus`,
  `proposedAt timestamp`, `acceptedAt timestamp` (set at Accepted); owns 16 Holdings.
- **Holding**: `id string`, `weight integer` (basis points); to Security (n:1).

Enums: `RunStatus = {Collecting, Optimizing, Ready, Failed}` (Collecting/Optimizing working, Ready/
Failed terminal); `PortfolioStatus = {Proposed, UnderReview, Accepted, Rejected}` (Proposed/
UnderReview open, Accepted/Rejected decided).

### Invariants (non-negotiable rules, by id)

- `index-top-30` (Index): only the top 30 constituents by rank feed a CandidateSet.
- `ticker-unique` (Security): every Security has a ticker unique across all securities.
- `candidate-deduped` (CandidateSet): no Security appears more than once in a CandidateSet.
- `candidate-from-top-30` (CandidateSet): every candidate is a top-30 constituent of some source Index.
- `run-ready-has-portfolio` (RecommendationRun): a Ready run references exactly one produced
  Portfolio; no partial success.
- `run-forward-only` (RecommendationRun): a run only advances Collecting to Optimizing to Ready, or
  fails; never backward.
- `run-terminal-absorbing` (RecommendationRun): a Ready or Failed run accepts no further actions.
- `portfolio-size-16` (Portfolio): a Portfolio holds exactly 16 Holdings.
- `portfolio-holdings-deduped` (Portfolio): no Security appears in more than one Holding.
- `portfolio-from-candidates` (Portfolio): every Holding's Security is in the run's CandidateSet.
- `portfolio-has-drawdown` (Portfolio): every Portfolio records the maxDrawdown it minimized.
- `portfolio-review-forward` (Portfolio): a Portfolio only advances Proposed to UnderReview, decides
  Accepted or Rejected, or is reopened to UnderReview; never otherwise backward.
- `portfolio-accept-role` (Portfolio): only a Manager or Admin may accept or reject a Portfolio.
- `portfolio-reopen-role` (Portfolio): only a Manager or Admin may reopen a decided Portfolio.
- `portfolio-accepted-has-date` (Portfolio): an Accepted Portfolio records an acceptedAt timestamp.
- `holding-weight-nonneg` (Holding): a Holding weight is never negative (no shorts).
- `holding-weights-sum-full` (Holding): a Portfolio's Holding weights sum to 10000 basis points.
- `feed-circuit-breaks` (model-level): repeated market-data failures open the circuit so calls
  fast-fail instead of hanging, and a Collecting run fails cleanly.

## 4. Architecture (the how)

Source: `design/workspace.dsl` and `design/ARCHITECTURE.md`. One Python process; seven code
boundaries plus the embedded store and the external provider.

### Containers and technology

- **pf.cli** (Python): parse, output, exit codes.
- **pf.app** (Python): run orchestration (collect, optimize, persist) and review commands; the
  load-act-save loop.
- **pf.domain** (Python): the RecommendationRun and Portfolio machines as pure transition functions,
  guards, invariant predicates. No I/O.
- **pf.optimizer** (Python): a pure, deterministic transform selecting the 16-of-N min-drawdown
  portfolio. No machine (pure logic gets a contract spec).
- **pf.feed** (Python): the market-data adapter; holds the circuit breaker; the sole importer of the
  provider client.
- **pf.repo** (Python): the sole importer of the DuckDB client; persistence, optimistic version
  checks, integrity check, backup/restore.
- **pf.model** (Python): entity types and enums (section 3's schema); no other layer restates it.
- **store** (DuckDB): the embedded local columnar file. **mkt** (MarketData): the external HTTP
  provider.

```mermaid
C4Container
  title DrawdownRecommender containers
  Person(analyst, "Analyst", "Starts runs")
  System_Boundary(pf, "DrawdownRecommender") {
    Container(cli, "CLI", "Python", "Parse, output")
    Container(app, "Application", "Python", "Orchestration, review, load-act-save")
    Container(domain, "Domain", "Python", "Run/Portfolio machines, guards")
    Container(optimizer, "Optimizer", "Python", "16-of-N min-drawdown")
    Container(feed, "Feed", "Python", "Market-data adapter + breaker")
    Container(repo, "Repository", "Python", "Persistence, locking, integrity")
    Container(model, "Model", "Python", "Types and enums")
    ContainerDb(store, "Store", "DuckDB", "Embedded local columnar db")
  }
  System_Ext(mkt, "MarketData", "HTTP provider")
  Rel(app, feed, "Fetches prices")
  Rel(feed, mkt, "Fetches", "HTTPS")
  Rel(app, repo, "Loads/saves")
  Rel(repo, store, "Reads/writes", "SQL")
```

### Deployment topology

One process on one machine, one local DuckDB file, one HTTP provider. A `recommend` command runs a
whole run to completion; review commands are separate invocations. `backup` copies the file; `restore`
replaces it. Offline once prices are cached; the provider is only needed while Collecting.

### Architecture Contract (boundaries + dependency rules)

The coding agent must not introduce a cross-boundary dependency outside `allow`. `pf.feed` is the sole
importer of the provider client; `pf.repo` the sole importer of DuckDB. Full contract:
`design/ARCHITECTURE.md` section 5. Allowed edges: cli->app, cli->model; app->{domain, optimizer,
feed, repo, model}; domain->model; optimizer->model; feed->model; feed->external.marketdata;
repo->model; repo->external.duckdb. Everything else denied; the two `pf.* -> external.*` blanket
denies are overridden only by the feed and repo allows.

### Interface contracts, event-contract, persistence and placement, NFR record

See `design/ARCHITECTURE.md` sections 6-10. Summary: interface contracts pin request/response shape,
enumerated errors, and idempotency for each boundary (these are the `onError` branches in the shard
behavior sections). The event-contract table is N/A (one synchronous process per command; no bus).
Persistence: Portfolio is a versioned row under an optimistic lock (two managers may review at once),
realized with the commit overlay; RecommendationRun is single-writer (no lock overlay); the optimizer
is a pure transform; Index/Security/CandidateSet/Holding are versioned rows without a lifecycle
machine. NFR: role-based authz for portfolio decisions; market-data key from env, never logged; store
file 0600; thousands of securities not millions; correctness over speed; residual failures print a
loud, distinct message with a distinct exit code.

## 5. Behavior: the component shards

Three machines, one shard each. The JSON files are the source; neither this root nor any shard
pastes them. Each shard carries its component's lifecycle narration, named-unit contract references,
test specification (oracle rows by stable id, guard-branch completeness, named-unit plan), and
build-plan milestones.

| component | machine (source) | oracle (generated) | matrix | shard |
|---|---|---|---|---|
| RecommendationRun | `machines/RecommendationRun.machine.json` | `machines/RecommendationRun.oracle.md` (8 rows) | `machines/RecommendationRun.matrix.md` | `BUILD/RecommendationRun.md` |
| Portfolio | `machines/Portfolio.machine.json` | `machines/Portfolio.oracle.md` (19 rows) | `machines/Portfolio.matrix.md` | `BUILD/Portfolio.md` |
| MarketDataFeed | `machines/MarketDataFeed.machine.json` | `machines/MarketDataFeed.oracle.md` (6 rows) | `machines/MarketDataFeed.matrix.md` | `BUILD/MarketDataFeed.md` |

Pure logic with no machine: the optimizer (a pure transform under a contract spec) and the
reference-data builds (Index refresh, Security upsert, CandidateSet dedup). Their plan and tests
live in the RecommendationRun shard, because the run pipeline is what invokes them.

## 6. Traceability matrix

Every invariant from section 3, its enforcement point, its component, its interface contract, and the
test id(s). Machine-enforced rows cite oracle STABLE ids; structural/prose rows cite named property
tests. Gx-trace reports the split (unit-backed vs attested).

| invariant id | enforced by (guard / structural) | in component | interface contract | test id(s) |
|---|---|---|---|---|
| `index-top-30` | structural: only rank <= 30 rows feed a build | pf.app, pf.repo | app->repo build | PROP-index-top-30 |
| `ticker-unique` | structural: upsert keys on ticker | pf.app, pf.repo | app->repo upsert | PROP-ticker-unique |
| `candidate-deduped` | structural: build dedupes by ticker | pf.app | app->repo build | PROP-candidate-deduped |
| `candidate-from-top-30` | structural: build draws only from index top 30 | pf.app | app->repo build | PROP-candidate-from-top-30 |
| `run-ready-has-portfolio` | action `recordPortfolio`; formal `Inv_Complete` | pf.domain | app->optimizer | RECO-d6fcf9 |
| `run-forward-only` | structural: the run graph; formal `Inv_TerminalAbsorbing` and forward chain | pf.domain | app->domain transition | RECO-f89da8, RECO-d6fcf9 |
| `run-terminal-absorbing` | structural: Ready and Failed are `final`; formal `Inv_TerminalAbsorbing` | pf.domain | app->domain transition | RECO-61506b, RECO-ed98c7 |
| `portfolio-size-16` | structural: the optimizer returns exactly 16 holdings | pf.optimizer | app->optimizer | PROP-portfolio-size-16 |
| `portfolio-holdings-deduped` | structural: distinct securities in the selection | pf.optimizer | app->optimizer | PROP-portfolio-holdings-deduped |
| `portfolio-from-candidates` | structural: the optimizer selects only from candidates | pf.optimizer | app->optimizer | PROP-portfolio-from-candidates |
| `portfolio-has-drawdown` | structural: the optimizer records the minimized maxDrawdown | pf.optimizer | app->optimizer | PROP-portfolio-has-drawdown |
| `portfolio-review-forward` | action `commit`/`setPending*` + structural; formal `StageForward` | pf.domain | app->domain transition | PORT-27d66f, PORT-d1647b |
| `portfolio-accept-role` | guard `canDecide` | pf.domain | app->domain accept/reject | PORT-2bf44c, PORT-a41039, PORT-ddb44c, PORT-351dec |
| `portfolio-reopen-role` | guard `canReopen` | pf.domain | app->domain reopen | PORT-db3bb9, PORT-9facf7 |
| `portfolio-accepted-has-date` | action `recordAccepted`; formal `Inv_CloseDate` | pf.domain | app->domain accept | PORT-d1647b |
| `holding-weight-nonneg` | structural: optimizer weights are non-negative | pf.optimizer | app->optimizer | PROP-holding-weight-nonneg |
| `holding-weights-sum-full` | structural: optimizer weights sum to 10000 bps | pf.optimizer | app->optimizer | PROP-holding-weights-sum-full |
| `feed-circuit-breaks` | guard `atThreshold` + action `recordTrip` | pf.feed | app->feed | MARK-acc7d7 |

No invariant is left unenforced. The structural rows are made true by the optimizer contract or the
build/upsert logic rather than a runtime guard; each is property-tested (section 7).

## 7. Cross-context test spec

Per-component transition tests, guard-branch completeness, and named-unit plans live in the shards.
This section fixes what crosses components: the shared test-id conventions, the contract tests at
every boundary, and the property tests.

### 7.1 Conventions

Transition tests key on the oracle STABLE id (e.g. `PORT-d1647b`), never the row number; row numbers
renumber when the design changes, stable ids do not. Property tests are named `PROP-<invariant-id>`,
one per invariant in section 3. Generated tests live under `test/generated/`, apart from
hand-written ones.

### 7.2 Contract tests per boundary

- cli->app: result and exit-code mapping.
- app->repo: Save/Load under version guards (ConflictError on a stale version).
- feed->mkt: error mapping and breaker behavior (breaker specifics in `BUILD/MarketDataFeed.md`).
- app->optimizer: shape and InfeasibleError.

### 7.3 Property tests

One per invariant, named `PROP-<invariant-id>` (section 6): generate random valid and invalid
inputs and assert the invariant holds or the operation is rejected. Notably `portfolio-size-16`,
`portfolio-holdings-deduped`, `portfolio-from-candidates`, `holding-weight-nonneg`, and
`holding-weights-sum-full` are properties of the optimizer output over random candidate universes
and price matrices. The machine-enforced invariants also carry formal proofs: `PortfolioData.tla`
(`StageForward`, `Inv_CloseDate`), `RecommendationRunData.tla` (`Inv_Complete`,
`Inv_TerminalAbsorbing`, `Live_Terminates`), and the control-flow `Live_OverlayResolves` for each
machine.

## 8. State migration

`Portfolio` persists its `status`, `acceptedAt`, and a `version`; `RecommendationRun` persists its
`status`. This is a greenfield design, so there are **no persisted instances yet**: the first run
starts from an empty store, no migration required.

Protocol for future lifecycle changes: when a `PortfolioStatus` or `RunStatus` value is renamed,
split, or removed, ship a mapping table from each old persisted value to its new state, applied once
on `Open()` over every row, or an explicit drain rule. The overlay states (committing/commitRetry/
reverted for Portfolio; collectRetry for the run) are never persisted (they exist only within a
command's execution), so renaming them needs no migration. Regenerate the oracles after any machine
change; the stable-id diff is the affected-test list.

## 9. Milestone map

The milestones are numbered globally (M0 to M5) and live in the shards; each shard's build plan
carries the full milestone blocks with their DoD lines. Build them in numeric order; every milestone
is green before the next starts.

| milestone | title | shard |
|---|---|---|
| M0 | Walking skeleton | `BUILD/RecommendationRun.md` |
| M1 | Run pipeline slice | `BUILD/RecommendationRun.md` |
| M2 | Feed breaker slice | `BUILD/MarketDataFeed.md` |
| M3 | Optimizer slice | `BUILD/RecommendationRun.md` |
| M4 | Portfolio review slice | `BUILD/Portfolio.md` |
| M5 | Reference-data and operations slice | `BUILD/RecommendationRun.md` |

## 10. Language realization notes

Target language: Python.

- The RecommendationRun and Portfolio machines become explicit `status` fields plus pure transition
  functions in `pf.domain`: `run_transition(cur, trigger, ctx) -> (next, actions, err)` and
  `portfolio_transition(cur, event, ctx) -> (next, actions, err)`, each a dispatch over
  `(cur, event)` returning the next state, ordered action names, and a `RejectedError` when no
  guarded branch applies. The Portfolio commit overlay is driven by `pf.app`: it calls the transition
  to compute `pending`, invokes `repo.save` under the version guard, and on `ConflictError` loops
  with backoff up to `MaxRetries` before rolling back.
- The MarketDataFeed breaker is a small object in `pf.feed` holding `failures`/`threshold` and a
  `closed|open|halfOpen` state; the transition function is the one in the machine.
- Persistence uses the explicit persisted-state-plus-optimistic-lock pattern for `Portfolio`
  (`WHERE version = expected`, bump on success). `RecommendationRun` is single-writer.
- The optimizer is a pure function; keep it dependency-injected so tests run it deterministically on
  fixed price fixtures. No state-machine library; the transitions are small dispatch tables.

### Toolchain and versions

- Python 3.12.x, managed with `uv` (commit `uv.lock`).
- `duckdb` (pinned in the lockfile), imported only by `pf.repo`.
- The market-data client (`httpx` plus a thin provider wrapper), imported only by `pf.feed`.
- Numerical work (drawdown, optimization) with `numpy`/`pandas` (pinned), used only inside
  `pf.optimizer`.
- Tests: `pytest`; property tests with `hypothesis`; the transition tests read the oracle rows.
- Lint/type: `ruff` and `mypy` (pin versions in CI).
- Design gates (the `machinery` binary, from the example root; design in `design/`):
  `machinery oracle design/machines` (regenerate and commit the oracles after any machine change);
  `machinery check design` (all design gates; add `--impl .` once code exists);
  `machinery verify-formal design` (regenerate and TLC-check the formal suite; needs Java 11+).

## 11. Hard-TDD protocol (read before writing any code)

1. **RED precondition.** Run `machinery check design` and require ZERO blocking findings before
   deriving any test. The oracles are the test spec; a red design means the spec itself cannot be
   trusted, and tests derived from it test the wrong things with confidence. Fix the design first,
   never the tests.
2. **Derivation.** A test-writer agent reads section 6, section 7, and the shard test
   specifications, and writes the full suite from the spec: transition tests keyed on the oracle
   STABLE id (e.g. `PORT-d1647b`), the falsifying-clause tests (per shard), the named-unit tests
   (per shard), and the contract and property tests (7.2, 7.3). A runtime that cannot spawn a
   fresh-context test-writer runs RED then GREEN sequentially with the same single agent; the
   derivation rule is unchanged (tests come from the spec, never from implementation intentions),
   and the gate runs in steps 1 and 3 separate the phases in place of context isolation.
3. **RED exit gate**, all three checks required before anything locks:
   a. Coverage of the spec: every oracle row's stable id appears whole-token somewhere in the suite
      (Gt-tests holds this deterministically once `--impl` points at the suite), every guard's
      falsifying case has a test, every invariant in section 3 has its `PROP-` property test.
   b. Architecture: `machinery check design --impl .` is green over the compile skeleton, stubs,
      and scaffolding the tests stand on (G4-import skips test files but checks everything they
      import), so the suite never forces the implementer to reproduce a boundary violation.
   c. The suite RUNS and is red for the right reason: failing assertions on missing behavior, never
      import or syntax errors inside the tests themselves.
4. **The tests are then LOCKED.** The implementer may not modify them to pass.
5. **The implementer** writes `pf.model`, `pf.repo`, `pf.feed`, `pf.optimizer`, `pf.domain`,
   `pf.app`, `pf.cli` until the locked tests pass, honoring the Architecture Contract (feed is the
   sole importer of the provider client; repo the sole importer of DuckDB; no cross-boundary edge
   outside `allow`).
6. **GREEN acceptance bar**, both together: the locked suite passes AND
   `machinery check design --impl .` is green again. Code that passes the tests by crossing a
   boundary fails the gate; code that respects the boundaries but fails a test is not done.
   Coverage target: >= 80% combined; integration tests use the real store, no mocks.
7. Generated tests live apart from hand-written tests (a `test/generated/` directory), so
   regeneration never clobbers hand-written ones.
8. A wrong test is a design defect: fix the design and this document, rerun `machinery oracle
   design/machines` and `machinery check design`, and regenerate the affected tests (the stable-id
   diff is the affected-test list). Do not adjust a test to pass.

## 12. Open questions and residual risks

- **Objective is single (min drawdown), by design.** The recommender optimizes only for lowest
  maximum drawdown; it ignores return, liquidity, and sector concentration. Named risk: the 16-stock
  minimum-drawdown portfolio may be poorly diversified or low-return. Out of scope to fix here.
- **Optimizer feasibility depends on data coverage.** If fewer than 16 candidates have full price
  history over the lookback, the run ends Failed (InfeasibleError). Residual: a thin candidate
  universe yields no recommendation; the operator sees the infeasibility cause.
- **Market-data provider has no deployable mitigation.** The circuit breaker bounds the damage
  (fast-fail, bounded run retries) but cannot manufacture data; a prolonged outage means no fresh
  recommendation. Cached prices allow offline reruns.
- **DuckDB corruption loses data since the last backup.** Recovery is `restore` from a `backup`; the
  loud `CorruptError` abort prevents silent corruption. Recommend scheduled backups (out of scope).
- **maxDrawdown stored in basis points as an integer** to keep the model integer-typed; if
  sub-basis-point precision is ever needed, widen the unit rather than switching to float in the
  persisted schema.
