# M2 - Feed breaker slice

## Outcome

The market-data breaker trips at its threshold, fast-fails while open without a provider call,
enters half-open after cooldown, and closes only after a successful probe.

## Domain context

The operational breaker owns failures, threshold, and `closed|open|halfOpen`. It enforces
`feed-circuit-breaks`; no breaker field belongs to the persisted portfolio domain.

## Architecture context

Implement the breaker inside `pf.feed` around the market-data provider adapter. Inject clock and
provider. The app sees the feed port's stable error envelope, never provider-specific exceptions.
Breaker state is in-memory per command invocation and has no state migration.

## Behavior and oracles

Parse `machines/MarketDataFeed.oracle.md` and assert next state/actions for `MARK-acc7d7`,
`MARK-9e6205`, `MARK-81fc92`, `MARK-609444`, `MARK-2bed99`, and `MARK-775b8f`. Test one failure below
threshold and one at threshold; test successful and failed half-open probes.

## TDD and implementation

Write table-driven oracle tests, an invariant property, and a provider spy proving open-state
fast-fail makes zero network calls. Lock RED after formatter, linter, and import checks. Implement
the small breaker object and stable error mapping, then run the complete feed contract tests.

## Risks and recovery

Use bounded counters and an injected monotonic clock; do not depend on wall-clock sleeps. A process
restart safely resets this nonpersistent protective state. Provider timeout remains bounded by the
outer run budget.

## Acceptance

Demonstrate trip, fast-fail, half-open, reclose, and reopen paths with call counts. Capture all six
stable ids, the property result, and implementation check in `acceptance/M2.yaml` for one commit.
