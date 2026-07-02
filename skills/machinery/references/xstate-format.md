# XState format (v5 JSON-serializable subset, enforced)

The design artifact is the **machine config**, not code. Guards, actions, and actors are
referenced by **string name** and implemented separately by the coding agent. That string-referenced
config is exactly the JSON-serializable subset, and it is valid to paste into the Stately visualizer
for validation. We author the structure; the coding agent authors the implementations.

The subset below is **enforced** by `machinery lint`, which G3 runs: unknown keys are hard
lint errors, not silently skipped. Parallel and history states, root-level `on`, non-string guards,
and array transition targets are rejected. Do not author outside the subset.

## The enforced subset

- **root keys**: `id`, `initial`, `context`, `states`, `description`, `meta`, `version`, plus the
  underscore annotations `_comment`, `_delays`, `_lifecycle_of`, `_role`, `_component`.
- **state keys**: `on`, `after`, `always`, `invoke`, `entry`, `exit`, `states`, `initial`, `type`,
  `id`, `meta`, `description`, `tags`, `onDone`, `output`, plus `_comment`, `_exhaustive`, `_ignores`.
- **invoke keys**: `src`, `input`, `id`, `onDone`, `onError`, `_comment`.
- **state types**: atomic (default), compound (has `states`), final. **Parallel and history are
  rejected.** Model an orthogonal concern as its own operational machine, or as context flags read
  by guards; model "resume where we left off" as an explicit context field plus guarded routing.
- **guards must be strings** naming a boolean predicate. **targets must be single strings** (a
  sibling name or `#id.path.to.state`); array targets are unsupported.
- **no root-level `on`**: every transition belongs to a state.

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

## Transition shape

`{ "target": "...", "guard": "guardName", "actions": ["actionA", "actionB"] }`

- `target` is a single string: a sibling name, or `#id.path.to.state` for cross-tree targets.
- `guard` is a **string** naming a boolean predicate over `(context, event)`. It maps to an
  invariant id. v5 uses `guard` (v4 used `cond`); author v5.
- `actions` are **string** names of effects (or `{"type": name}` objects). Entry and exit effects
  go in `entry` / `exit`.

## Machine annotations (checked, not decorative)

Three annotations carry meaning the deterministic gates consume. Every machine must be classifiable
as either a lifecycle machine or an operational machine; a filename matching a Modelith entity
counts as the lifecycle claim.

**`_role: "operational"`** on a machine that is not the lifecycle of a Modelith entity, for example
a command-execution envelope:

```jsonc
{ "id": "CommandExecution", "_role": "operational", "initial": "opening", ... }
```

**`_lifecycle_of: "<Entity>"`** when the machine's filename does not match the entity name:

```jsonc
// in DealAggregate.machine.json
{ "id": "dealAggregate", "_lifecycle_of": "Deal", ... }
```

**`_exhaustive: "<reason>"`** on a state whose `always` list is fully guarded and has no unguarded
escape (no `after`, `on`, or `invoke`). The reason must state why the guard set is total; if no
guard is true the machine is stuck, so this is the liveness side condition, made visible. Without
the annotation, that shape is a lint ERROR. **The formal layer cannot verify this claim** (rung-3
TLA+ erases guards), so a false `_exhaustive` yields a green liveness proof for a machine that can
deadlock: `machinery_check` G3 emits a warn naming every `_exhaustive` state for exactly this
reason. Reserve `_exhaustive` for guard sets that are provably total (a `prior`/`pending` field whose
codomain is exactly the enumerated states, as in a rollback router). Otherwise prefer an unguarded
fallback branch, which TLC does check:

```jsonc
"routing": {
  "_exhaustive": "pendingStage is set only by the setPending* actions, whose codomain is exactly the five guarded stages",
  "always": [
    { "target": "Qualified",   "guard": "pendingIsQualified" },
    { "target": "Proposal",    "guard": "pendingIsProposal" },
    { "target": "Negotiation", "guard": "pendingIsNegotiation" },
    { "target": "Won",         "guard": "pendingIsWon" },
    { "target": "Lost",        "guard": "pendingIsLost" }
  ]
}
```

**`_ignores: {event: reason}`** on a resting state, for event completeness (next section):

```jsonc
"Lead": {
  "on": { "advance": { "target": "persisting", "guard": "guardCanAdvance", "actions": "setPendingAdvance" } },
  "_ignores": { "reopen": "reopen applies only to terminal stages; guardCanReopen rejects it upstream" }
}
```

## Explicit ignores and event completeness (deterministically checked)

A **resting state** is a top-level, non-final state with no `invoke` and no `always`: it sits
waiting for external events. Every resting state must handle or explicitly ignore **every** event
the machine reacts to anywhere. "Explicitly ignore" means an `_ignores` entry mapping the event
name to a reason string. A state may not both handle and ignore the same event. Transient states
(with `invoke` or `always`, or lowerCamel overlay states) resolve internally before an external
event is processed; final states reject structurally. The lint checks all of this; a missing
handler with no `_ignores` entry is an ERROR.

## State kinds (use hierarchy to manage complexity)

- **atomic** - a leaf state.
- **compound** - has `initial` + nested `states`. Group a lifecycle phase and its substates. A
  compound state's own `onDone` fires when its child final state is reached.
- **final** - `"type": "final"`. Terminal; rejects all events structurally.

Parallel and history states are **not** in the subset (see above).

## Failure-mode idioms (this is where the value is)

Every side-effecting operation is an `invoke`. Enumerating its outcomes forces you to name the
failure modes. Standard shapes:

- **async op** -> `invoke` with `onDone` (success) and `onError` (failure). No `onError` is a bug,
  and a lint error.
- **timeout** -> `after: { OP_TIMEOUT: { target: "failed" } }` on the invoking state. Every
  invoking state needs one; the lint enforces it.
- **retry with backoff** -> `onError` to a `retrying` state; `after` a backoff delay back to the op;
  a guard `retriesExhausted` on a context counter routes to terminal failure when spent.
- **circuit breaker** -> track consecutive failures in a context counter; a guard `circuitClosed`
  gates the invoke; an `open` / `degraded` overlay state serves the fallback and `after` a cooldown
  probes half-open. If the breaker truly varies independently of the flow, give it its own
  operational machine (parallel regions are outside the subset).
- **compensation / saga** -> `onError` to a `compensating` state that invokes rollback actors, then to
  a stable state.
- **degrade** -> when the breaker is open, route to a `degraded` state serving a fallback rather than
  hard-failing.

Delays (`PAYMENT_TIMEOUT`, `RETRY_BACKOFF`) are **named**; their millisecond values live in the
`delays` implementation map, so the config stays declarative and the bounds come from C4.

## Choreography (machines reacting to each other over a bus)

Sagas above are orchestration: one machine drives the compensation. In choreography, machines react
to each other's events over a bus, and two rules apply:

- **Every consumed external event must appear in the event-contract table** in ARCHITECTURE.md
  (see `references/c4-standalone.md`): producer, consumer, payload by Modelith attribute reference,
  delivery guarantee, ordering assumption, dedupe key. Bus coupling is invisible to G4-import; the
  table is the governing artifact.
- **The consumer must handle redelivery.** With at-least-once delivery, an event can arrive in a
  state where it is stale. Consuming it there is an `_ignores` entry carrying the dedupe reasoning,
  not an accident:

```jsonc
"Paid": {
  "on": { "SHIP": { "target": "shipping" } },
  "_ignores": { "PAYMENT_CONFIRMED": "already Paid; at-least-once redelivery, deduped by paymentId" }
}
```

## Named-unit contracts (the coding agent implements these)

For every machine, produce a table the coding agent and the test-writer both consume. Each row also
states its test type (unit / integration / property) and its fixture (real dependency or fake, and
which):

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `hasValidPaymentMethod` | guard | `(ctx, evt) -> bool` | true iff a non-expired method is on file | invariant `payment-method-valid` | unit | fake clock |
| `chargePayment` | actor | `(input) -> Charge` | charges once; idempotent by `orderId` | C4 rel `order.service -> payments.api` | integration | contract-tested payments fake |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | - | unit | none |

G3 requires a row for every guard, action, and actor the machine fires; a missing row is DRIFT.
Side-effect and idempotency contracts (the "charges once" class) are integration or property tests
against the real dependency or a contract-tested fake; they are never derivable from transition tests.

## Helper keys, and importing into Stately or @xstate/graph

For humans, a machine file also carries two underscore-prefixed helper keys that are NOT part of the
XState config schema (the lint accepts exactly these plus the annotations above):

- `_comment` - a header note stating placement and concurrency serialization (from C4 section 7).
- `_delays` - the named delays with their millisecond bounds and rationale, for example
  `"persistTimeout": "10000 ms - LadybugDB write timeout"`. The `after` blocks reference these by name.

Before loading a machine into Stately Studio or `@xstate/graph`, strip every `_`-prefixed key and
supply the real implementations via `setup({ actors, guards, actions, delays })`: the string-named
guards, actions, and actors, plus the named delays with the ms values taken from `_delays`, all go
there. The state graph itself imports unchanged.

## Validation before you call Gate 3 done

Deterministic (run the tools; do not eyeball):

- `machinery lint design/machines` checks subset conformance, reachability, dead ends,
  `invoke` with `onError` plus `after`, shadowed branches, guarded-always exhaustiveness
  (`_exhaustive`), and resting-state event completeness (`_ignores`).
- `machinery oracle design/machines` generates `<M>.oracle.md`; commit it. G3 regenerates it in
  memory and diffs; a stale committed oracle is DRIFT. Tests key on the oracle's stable ids.

Judgment (the lint cannot check these; you must):

- **Invariant coverage**: every Modelith invariant is enforced by a guard or is structurally
  impossible, and the guard's semantics actually enforce the invariant it names.
- **Failure coverage**: every C4 mitigation row's residual behavior has its transition.
- **Path generation**: `@xstate/graph` `getShortestPaths` / `getSimplePaths` enumerate covering
  paths for multi-step tests on top of the per-transition oracle rows.

Per-instance concurrency is out of the machine's scope and lives in C4 (actor mailbox vs row lock).
Note it in the machine's `_comment` so the coding agent serializes events correctly.
