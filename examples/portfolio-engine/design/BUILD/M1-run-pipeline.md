# M1 - Run pipeline slice

## Outcome

A recommendation run completes successfully, retries transient collection failures to a fixed
bound, and terminates loudly on exhausted collection or optimizer failure. Ready and Failed absorb
later events.

## Domain context

RecommendationRun owns `status`, retry count, candidate-set identity, and the resulting portfolio
identity. Enforce `run-ready-has-portfolio`, `run-forward-only`, and `run-terminal-absorbing`.
Retry overlay state is execution-only and is never persisted.

## Architecture context

`pf.app` is the single writer. It invokes the feed port during Collecting, invokes the pure
optimizer during Optimizing, and persists only domain states through `pf.repo`. Both invokes have
explicit timeout/error mapping. Keep retry/backoff deterministic under an injected clock.

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
result and performs no duplicate optimizer or persistence effect.

## Acceptance

Run all eight stable-id cases, invariant properties, timeout/error contracts, and crash-boundary
tests. Capture `pytest`, `machinery check design --impl impl`, and formal evidence in
`acceptance/M1.yaml` for the reviewed commit.
