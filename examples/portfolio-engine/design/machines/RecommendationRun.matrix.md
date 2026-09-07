# RecommendationRun machine: named-unit contracts and failure catalog

Transitions are covered by the generated `RecommendationRun.oracle.md`. Forward pipeline:
Collecting to Optimizing to Ready, or Failed. The flaky market-data fetch is retried a bounded
number of times (collectRetry) before the run fails cleanly.

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `retries >= MaxRetries` | bounds the fetch retry loop | unit | none |
| `recordPortfolio` | action | `(ctx) -> ctx` | sets `portfolioId` to the optimizer's result when entering Ready | invariant `run-ready-has-portfolio` | unit | fake optimizer result |
| `incRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | bounds the fetch retry loop | unit | none |
| `publishReady` | action | `(ctx) -> ()` | writes the run's success and the portfolio id to stdout | operator signal | unit | captured stdout |
| `publishFailure` | action | `(ctx) -> ()` | writes the failure cause (market data unavailable, or infeasible) to stderr, non-zero exit | operator signal | unit | captured stderr |
| `assertTerminalAbsorbing` | action | `(state, event) -> bool` | the dispatcher has no outgoing transition for Ready or Failed and returns TerminalError for every later lifecycle event | invariant `run-terminal-absorbing`; RecommendationRun terminal entry | model + unit | generated event enumeration against both final states |
| `fetchPrices` | actor | `(candidateSetId, lookbackDays) -> PriceMatrix` | fetches full price history for every candidate through the feed breaker using the frozen CandidateSnapshot and OptimizerLimits captured at construction (the invoke input stays exactly these two fields); a FeedError or CircuitOpenError routes to collectRetry; only driver-accepted provider outcomes are processed | C4 rel: pf.app to pf.feed to mkt | integration | contract-tested market-data fake plus a breaker-open fixture |
| `optimize` | actor | `(candidates, prices) -> Portfolio` | selects 16 candidate securities minimizing max drawdown and accepts output only when every integer-basis-point weight is non-negative and the exact sum is 10000; InfeasibleError otherwise | C4 rel: pf.app to pf.optimizer; invariants `holding-weight-nonneg`, `holding-weights-sum-full` | integration + property | real optimizer on fixed and generated price/weight fixtures |

Pre-run admission (before the machine starts): `repoPort.LoadCandidateSet(candidateSetId,
limits)` runs once in the command factory; typed NotFoundError/CorruptError/IOError/
BusyError/ValidationError failures return to the CLI unchanged, create no run, make no feed
call and consume zero collection retries. The frozen snapshot and limits are immutable across
retries; a binding mismatch (runId/candidateSetId/lookbackDays/limits vs the validated
request, or a missing accepted CollectedInput) is InternalError(ACTOR_BINDING_MISMATCH)
before any port call.

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| pre-run repository admission failure | `repoPort.LoadCandidateSet` typed error in the command factory | no run is created; no machine transition | surface the typed class/exit unchanged; no feed call, no retry consumption | admission read with finite input budgets; no wall-clock return bound claimed for a blocked database read |
| market-data fetch failure or timeout | `fetchPrices` invoke `onError`, or `after FETCH_TIMEOUT` | Collecting to collectRetry | back off `RETRY_BACKOFF`, `incRetries`, retry the fetch | bounded by `retriesExhausted` (<= MaxRetries); the feed breaker fast-fails while open to avoid hanging (`feed-circuit-breaks`) |
| retries exhausted | guard `retriesExhausted` true in collectRetry | collectRetry to Failed | end Failed; `publishFailure` prints "market data unavailable" | residual: no portfolio produced; run is terminal (`run-terminal-absorbing`) |
| optimization infeasible or timeout | `optimize` invoke `onError` (InfeasibleError), or `after OPTIMIZE_TIMEOUT` | Optimizing to Failed | end Failed; `publishFailure` prints the infeasibility cause | residual: no portfolio; fewer than 16 candidates had full history |
| success | `optimize` invoke `onDone` | Optimizing to Ready | `recordPortfolio` sets the result; durable Ready is announced only after the atomic CommitRecommendation publication receipt (run + portfolio + 16 holdings); `publishReady` prints it | a Ready run always has a portfolio (`run-ready-has-portfolio`), proved by the terminal-lifecycle completeness invariant; the model row observes the resolved transition, the publication barrier is tested separately (M0/M1) |

Resolved-envelope assumption: the 8 oracle rows cover accepted feed/pure-optimizer outcomes
under successful command admission and resolved persistence boundaries. Terminal Failed
persistence is subject to the same write contract; an unresolved publication cannot be
claimed as a durable Ready/Failed completion.
