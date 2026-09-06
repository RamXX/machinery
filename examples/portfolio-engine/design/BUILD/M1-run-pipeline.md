# M1 - Run pipeline slice

## Outcome

A recommendation run completes successfully, retries transient collection failures to a fixed
bound, and terminates loudly on exhausted collection or optimizer failure. Ready and Failed absorb
later events.

## Domain context

RecommendationRun owns `status`, retry count, candidate-set identity, and the resulting portfolio
identity. Enforce `run-ready-has-portfolio`, `run-forward-only`, and `run-terminal-absorbing`.
Retry overlay state is execution-only and is never persisted. `lookbackDays` counts observed
closing-price dates including the initial valuation date (integer >= 2; gaps permitted).

## Architecture context

`pf.app` is the single writer. It invokes the feed port during Collecting, invokes the pure
optimizer during Optimizing, and persists only domain states through `pf.repo`. Both invokes have
explicit admission-deadline/error mapping: repository admission (LoadCandidateSet) runs once
BEFORE the run exists and its typed NotFound/Corrupt/IO/Busy failures create no run, make no
feed call and consume zero collection retries; fetch (20000 ms/attempt) and optimizer
(60000 ms) budgets are result-admission deadlines under an injected monotonic clock; retry
backoff is 1000/2000/4000 ms with MaxRetries=3 and no jitter. The frozen CandidateSnapshot,
limits and per-attempt custody are immutable across retries; a binding mismatch fails before
any port call.

## Behavior and oracles

Parse and assert every row of `machines/RecommendationRun.oracle.md`: `RECO-c7bb09`,
`RECO-f89da8`, `RECO-040944`, `RECO-c85bd8`, `RECO-d6fcf9`, `RECO-ed98c7`, `RECO-0d730c`, and
`RECO-61506b`. Assert next state and ordered actions. Test below and at `MaxRetries` separately.

## TDD and implementation

Write the complete table-driven oracle conformance test plus property tests for the three run
invariants. Add contract-tested feed and optimizer fakes with timeout/error injection. Lock RED
only after imports, formatting, and lint are clean; then implement the transition function and
orchestration without editing locked tests.

## Risks and recovery

Bound attempts, backoff, and invocation time. Persist no transient retry state. A crash before the
durable terminal write leaves a resumable nonterminal run; a crash after it re-reads the terminal
result and performs no duplicate optimizer or persistence effect. Durable Ready and durable
Failed both require a confirmed atomic CommitRecommendation receipt; an unresolved publication
is reported as an unknown-outcome residual (exit 12), never as a durable terminal state - test
the persistence barrier independently of the pure transition rows.

## Acceptance

Run all eight stable-id cases, invariant properties, timeout/error contracts, and crash-boundary
tests. Capture `pytest`, `machinery check design --impl impl`, and formal evidence in
`acceptance/M1.yaml` for the reviewed commit.
