# MarketDataFeed machine: named-unit contracts and failure catalog

Operational envelope (`_role: operational`): a circuit breaker over the market-data provider, one
instance per process. Transitions are covered by the generated `MarketDataFeed.oracle.md`. It
enforces `feed-circuit-breaks`: repeated provider failures open the circuit so calls fast-fail
instead of hanging.

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `atThreshold` | guard | `(ctx) -> bool` | true iff `failures + 1 >= threshold` (this failure trips the breaker) | invariant `feed-circuit-breaks` | unit | none |
| `probeSucceeded` | guard | `(ctx, evt) -> bool` | true iff the half-open trial call returned data | recloses the circuit on recovery | unit | fake probe result |
| `recordTrip` | action | `(ctx) -> ()` | logs `feed_circuit_open` and marks the breaker open | operator signal for `feed-circuit-breaks` | unit | captured log |
| `incFailures` | action | `(ctx) -> ctx` | `failures := failures + 1` | counts toward the trip threshold | unit | none |
| `resetFailures` | action | `(ctx) -> ctx` | `failures := 0` on a success or a successful probe | clears the count on recovery | unit | none |

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| a provider call fails below the threshold | `failure` event, guard `atThreshold` false | closed to closed | `incFailures`; keep serving | bounded: at the threshold the next failure trips the breaker |
| failures reach the threshold | `failure` event, guard `atThreshold` true | closed to open | `recordTrip`; log `feed_circuit_open`; fast-fail subsequent calls | residual: calls fast-fail as CircuitOpenError while open (`feed-circuit-breaks`); a Collecting run retries then fails cleanly |
| cooldown elapses | `after COOLDOWN` in open | open to halfOpen | allow one trial call | the trial decides: `probeResult` reopens or recloses |
| trial probe fails | `probeResult` event, guard `probeSucceeded` false | halfOpen to open | `recordTrip`; stay open another cooldown | bounded: repeated cooldown-probe cycles, never a hang |
| trial probe succeeds | `probeResult` event, guard `probeSucceeded` true | halfOpen to closed | `resetFailures`; resume normal calls | the provider has recovered |
