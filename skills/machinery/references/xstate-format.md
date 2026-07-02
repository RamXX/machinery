# XState format (v5 JSON-serializable subset)

The design artifact is the **machine config**, not code. Guards, actions, and actors are
referenced by **string name** and implemented separately by the coding agent. That string-referenced
config is exactly the JSON-serializable subset, and it is valid to paste into the Stately visualizer
for validation. We author the structure; the coding agent authors the implementations.

## What goes in the JSON

```jsonc
{
  "id": "order",
  "initial": "pending",
  "context": {                     // the extended state (data). Shape references Modelith attributes.
    "orderId": null,
    "retries": 0
  },
  "states": {
    "pending": {
      "on": {
        "PAY": { "target": "paying", "guard": "hasValidPaymentMethod" }
      }
    },
    "paying": {
      "invoke": {                  // every side effect is an invoked actor
        "src": "chargePayment",    // implemented separately; maps to a C4 relationship
        "input": { "orderId": "context.orderId" },
        "onDone": { "target": "paid", "actions": "recordCharge" },
        "onError": { "target": "paymentRetry", "actions": "logChargeError" }
      },
      "after": {                   // timeout is a delayed transition
        "PAYMENT_TIMEOUT": { "target": "paymentRetry" }
      }
    },
    "paymentRetry": {
      "always": [                  // guarded automatic transition
        { "target": "paymentFailed", "guard": "retriesExhausted" }
      ],
      "after": {
        "RETRY_BACKOFF": { "target": "paying", "actions": "incrementRetries" }
      }
    },
    "paid": { "type": "final" },
    "paymentFailed": { "type": "final" }
  }
}
```

## State kinds (use hierarchy to manage complexity)

- **atomic** - a leaf state.
- **compound** - has `initial` + nested `states`. Group a lifecycle phase and its substates.
- **parallel** - `"type": "parallel"` with orthogonal regions. Use for concerns that vary
  independently, for example a circuit-breaker region running alongside the main flow.
- **final** - `"type": "final"`. A parent with all regions final can fire `onDone`.
- **history** - `"type": "history"` (shallow or deep) to resume where a compound state left off.

## Transition shape

`{ "target": "...", "guard": "guardName", "actions": ["actionA", "actionB"] }`

- `target` is a sibling name, or `#id.path.to.state` for cross-tree targets.
- `guard` is a **string** naming a boolean predicate over `(context, event)`. It maps to an
  invariant id. v5 uses `guard` (v4 used `cond`); author v5.
- `actions` are **string** names of effects. Entry and exit effects go in `entry` / `exit`.

## Failure-mode idioms (this is where the value is)

Every side-effecting operation is an `invoke`. Enumerating its outcomes forces you to name the
failure modes. Standard shapes:

- **async op** -> `invoke` with `onDone` (success) and `onError` (failure). No `onError` is a bug.
- **timeout** -> `after: { OP_TIMEOUT: { target: "failed" } }` on the invoking state.
- **retry with backoff** -> `onError` to a `retrying` state; `after` a backoff delay back to the op;
  a guard `retriesExhausted` on a context counter routes to terminal failure when spent.
- **circuit breaker** -> a parallel region tracking `closed` / `open` / `halfOpen`; a guard
  `circuitClosed` gates the invoke; trip on consecutive `onError`.
- **compensation / saga** -> `onError` to a `compensating` state that invokes rollback actors, then to
  a stable state.
- **degrade** -> when the breaker is open, route to a `degraded` state serving a fallback rather than
  hard-failing.

Delays (`PAYMENT_TIMEOUT`, `RETRY_BACKOFF`) are **named**; their millisecond values live in the
`delays` implementation map, so the config stays declarative and the bounds come from C4.

## Named-unit contracts (the coding agent implements these)

For every machine, produce a table the coding agent and the test-writer both consume:

| name | kind | signature | contract (pre / post) | maps to |
|---|---|---|---|---|
| `hasValidPaymentMethod` | guard | `(ctx, evt) -> bool` | true iff a non-expired method is on file | invariant `payment-method-valid` |
| `chargePayment` | actor | `(input) -> Charge` | charges once; idempotent by `orderId` | C4 rel `order.service -> payments.api` |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | - |

## Helper keys, and importing into Stately or @xstate/graph

For humans, a machine file may carry two underscore-prefixed helper keys that are NOT part of the
XState config schema:

- `_comment` - a header note stating placement and concurrency serialization (from C4 section 7).
- `_delays` - the named delays with their millisecond bounds and rationale, for example
  `"persistTimeout": "10000 ms - LadybugDB write timeout"`. The `after` blocks reference these by name.

Before loading a machine into Stately Studio or `@xstate/graph`, strip every `_`-prefixed key and
supply the real implementations via `setup({ actors, guards, actions, delays })`: the string-named
guards, actions, and actors, plus the named delays with the ms values taken from `_delays`, all go
there. The state graph itself imports unchanged.

## Validation before you call Gate 3 done

- **Reachability**: every non-initial state is the target of some transition.
- **No dead ends**: every non-final state has at least one outgoing transition (event, `after`, or `always`).
- **Event completeness**: for each state, every event that can arrive is handled or explicitly ignored.
- **Invariant coverage**: every Modelith invariant is enforced by a guard or is structurally impossible.
- **Path generation**: `@xstate/graph` `getShortestPaths` / `getSimplePaths` enumerate covering paths.
  Each path is a hard-TDD test case. The count of transitions is the minimum test count.

Per-instance concurrency is out of the machine's scope and lives in C4 (actor mailbox vs row lock).
Note it in the machine's header comment so the coding agent serializes events correctly.
