# Machine format

A machine is one JSON file, `<Name>.machine.json`, in the machines directory
(`design.machines` in `machinery.yaml`). It uses the JSON-serializable subset of XState v5, with
semantics pinned to XState 5.33.4. Guards, actions and actors are named by string and implemented in
code; the file describes structure only. A file in this subset can be pasted into the Stately
visualizer after its `_`-prefixed keys are removed.

`docs/formats.md` is the contract: section 5.2 (what a machine is), 6.2 (the `machines` stage), 7.2
(cases) and 7.3 (runtime tables). This guide is the working reference for authors. `machinery lint`
enforces everything below, and an unknown key is an error, never silently skipped.

## What the lint enforces

`machinery lint --stage machines` reads every `*.machine.json` and reports each finding as
`ERROR  <file>: <message>`. The rules, grouped:

- **Subset conformance.** Only the keys listed below. Parallel and history states, a root-level
  `on`, non-string guards and array targets are rejected.
- **Name patterns.** State and event names match `^[A-Za-z][A-Za-z0-9_]*$`, because they are hashed
  into case ids. Guard, action and actor names match `^[A-Za-z_][A-Za-z0-9_]*$`.
- **Strict target resolution.** See "Targets and state ids".
- **Declared delays.** Every `after` key is a name declared in `_delays`, and every declared delay
  is used by an `after`.
- **Reachability.** Every state can be reached from the initial state. A non-final leaf must have a
  transition that leaves it (no dead ends, no absorbing states). A final state has no transitions.
- **Invokes.** Every `invoke` has a `src`, an `onError`, and its state has a non-empty `after`.
- **Branch order.** An unguarded branch is last, no two branches of a list share a guard, and no
  branch list is empty.
- **Always cycles.** A cycle of unguarded `always` transitions loops forever and is an error.
- **Event completeness.** Every resting state handles or ignores every event (`_ignores`).
- **Guard-false dispositions.** A fully guarded handler declares `_refusal`; a fully guarded
  `always` list declares `_exhaustive`.
- **Guard clauses.** `_guards` is well formed and names guards the machine uses.
- **Fail-closed tags.** Not-checked states never reach an accepting state.
- **Collisions.** Two machines may not share a case-id tag or a machine id.

The stage prints `checked: <n> machines, <n> states, <n> transitions`. It is not applicable when
`design.machines` is absent, and not demonstrated (`nothing to check`) when the directory holds no
machine file. A file that cannot be parsed is an ERROR.

## The subset

- **Root keys:** `id`, `initial`, `context`, `states`, `description`, `meta`, `version`, and the
  annotations `_comment`, `_delays`, `_lifecycle_of`, `_role`, `_oracle_tag`, `_guards`, `_external_events`.
- **State keys:** `on`, `after`, `always`, `invoke`, `entry`, `exit`, `states`, `initial`, `type`, `id`,
  `meta`, `description`, `tags`, `onDone`, `output`, and the annotations `_comment`, `_exhaustive`,
  `_ignores`, `_refusal`.
- **Transition keys:** `target`, `guard`, `actions`, `description`, `_comment`. A typo such as
  `tagret` is an error: it would otherwise become a silent internal transition.
- **Invoke keys:** `src` (mandatory, a non-empty string naming the actor), `input`, `id`, `onDone`,
  `onError`, `_comment`.
- **State types:** atomic (the default), compound (has `states` and an `initial` that is one of its
  children), final. A state with `onDone` must be compound and have a final descendant, or the
  transition could never fire.
- **Events:** explicit names only. The wildcard `*` and platform events (`done.*`, `error.*`) are
  outside the subset; use invoke `onDone` and `onError`.
- **Guards** are strings naming a boolean predicate. An empty string is not a guard.
- **Targets** are single strings (see below).
- **Actions** are name strings, or `{"type": "<name>"}` objects that carry only `type`. Parameters
  belong in the implementation, not in the design. Entry and exit effects go in `entry` and `exit`.
- **Empty containers are errors.** An empty branch array under `on:<event>`, `after:<delay>`,
  `always` or an invoke `onDone`/`onError` would swallow its trigger silently; an empty `after: {}`
  on an invoking state declares no timeout.

Parallel and history states are not supported. Model an orthogonal concern as its own operational
machine, or as context read by guards.

## Targets and state ids

Resolution is strict, with no fallbacks.

- A **bare name** resolves only among the siblings of the source state (the source itself included,
  for an explicit self-transition). It never reaches a cousin or an ancestor.
- **`#id.path.to.state`** resolves the head against the machine id or a state-level `id`, then each
  following segment strictly as a child of the previous node. A missing segment is a dangling
  target and the message names it.
- **`.child`** relative targets are rejected. Use a sibling name or the `#id.path` form.
- A **dotted target without `#`** is rejected.
- A **state-level `id`** must be a non-empty identifier, unique in the machine, and different from
  the machine id.
- A transition with no `target` is internal: the state is not exited.

## Example

```json
{
  "id": "order",
  "initial": "Pending",
  "_delays": {"payTimeout": "5000 ms - upper bound on the payment provider call"},
  "states": {
    "Pending": {
      "on": {
        "pay": [
          {"target": "paying", "guard": "hasPaymentMethod", "actions": "stampAttempt"},
          {"actions": "recordPayRefused"}
        ],
        "cancel": {"target": "Cancelled", "actions": "recordCancellation"}
      },
      "_ignores": {"ship": "an unpaid order cannot ship; the API refuses with ORDER_NOT_PAID"}
    },
    "paying": {
      "invoke": {
        "src": "chargePayment",
        "onDone": {"target": "Paid", "actions": "recordCharge"},
        "onError": {"target": "Pending", "actions": "recordPaymentFailure"}
      },
      "after": {"payTimeout": {"target": "Pending", "actions": "recordPaymentTimeout"}}
    },
    "Paid": {
      "on": {"ship": {"target": "Shipped", "actions": "recordShipment"}},
      "_ignores": {"pay": "already paid; a repeated pay is refused with ORDER_ALREADY_PAID"}
    },
    "Shipped": {"type": "final", "entry": "notifyCustomer"},
    "Cancelled": {"type": "final"}
  }
}
```

## Lifecycle and operational machines

Every machine is one of two kinds (`docs/formats.md` section 5.2):

- a **lifecycle machine** of a model entity: the root key `_lifecycle_of: "<Entity>"`, or a file
  base name equal to the entity name. Top-level states whose name starts with an uppercase letter
  are the **domain states**; lowercase-first states are transient overlay states.
- an **operational machine**: `_role: "operational"`, for example a command-execution envelope.

## Annotations

These are checked, not decorative.

**`_delays`** (root). Maps each delay name to `"<ms> - <rationale>"`. The leading integer is the
bound in milliseconds, and the runtime table carries it. The rule runs both ways: an `after` key
that is not declared is an error, a raw numeric key (`"5000"`) is an error, and a declared delay no
`after` consumes is an error.

**`_ignores`** (state). Maps an event name to the reason the state does not react to it. See
"Event completeness".

**`_refusal`** (state). Maps a handler to what happens when no guard of that handler admits the
trigger. See "Guard-false dispositions".

**`_exhaustive`** (state). A reason string, valid on a state whose `always` list is fully guarded
and has no unguarded escape. See "Guard-false dispositions".

**`_guards`** (root). Declares the clauses of a guard. See "Guard clauses".

**`_oracle_tag`** (root, optional). 2 to 8 characters matching `[A-Z0-9]`. It overrides the tag of
the case ids, which is otherwise the machine `id` uppercased and cut to four characters. Two
machines whose tags collide (`Deal` and `DealAggregate` both give `DEAL`) are an error that names both
files: set `_oracle_tag` on one. Changing a tag changes every case id of that machine, so choose it
once.

**`_external_events`** (root, optional). An array of event names this machine receives that are not
model actions, for example events that arrive from a bus or a peer subsystem (`docs/formats.md`
section 6.4 reads it). Each entry is a non-empty string. An entry that no state handles or ignores
is a stale marker and an error: remove it, or wire the event.

**`_comment`** (any level). Free text.

## Event completeness

A **resting state** is a leaf state that starts with an uppercase letter, is not final, and has no
`invoke` or `always` on itself or any ancestor: it sits waiting for outside events. Every resting
state must handle or explicitly ignore every event the machine reacts to anywhere. Handlers and
`_ignores` on an ancestor count, because an unhandled event bubbles up. A state may not both handle
and ignore an event, and an ignore must have a reason:

```json
"Paid": {
  "on": {"ship": {"target": "Shipped"}},
  "_ignores": {"pay": "already paid; a repeated pay is refused with ORDER_ALREADY_PAID"}
}
```

Each ignore is a case: a test makes the event happen in that state and observes that nothing
changes.

## Guard-false dispositions

A handler whose every branch has a guard, with no unguarded sibling, silently absorbs its trigger
when every guard is false. That can be right (a refusal raised at the command boundary) or a
deadlock, and the file cannot show which. The author says which.

`_refusal` names the handler and states the disposition. Handler names are the event name for an
`on` entry, `after:<delay>` for a timer, `<src>.onDone` and `<src>.onError` for an invoke, and
`onDone` for a state's own completion:

```json
"Proposed": {
  "on": {"accept": {"target": "committing", "guard": "canDecide"}},
  "_refusal": {"accept": "canDecide false is an authorization refusal: the API returns AuthzError and the aggregate is untouched"}
}
```

Without the entry the lint reports the handler. An entry for a handler that is not fully guarded, or
that does not exist, is a stale claim and an error. Prefer an unguarded fallback branch when one
fits the behavior; it always admits and needs no entry.

`_exhaustive` answers the same question for `always` lists. A state whose `always` list is fully
guarded and that has no `after`, `on` or `invoke` is stuck if no guard is true. Either add an
unguarded fallback branch, or write `_exhaustive` with the reason the guards are total. The lint
cannot check that claim, so reserve it for guard sets that are total by construction.

## Guard clauses

A guard that is a conjunction hides bugs behind its first false clause. A test that makes the whole
guard false proves one clause at most. `_guards` declares the clauses:

```json
"_guards": {"canWinDeal": ["callerMayWriteDeal", "amountIsPositive"]}
```

- The key is a guard the machine uses. A key that no transition uses is an error.
- The value is a list of at least two clause names, each matching `^[a-z][A-Za-z0-9]*$`, without
  duplicates.
- Declare `_guards` for every guard that checks more than one thing.

## Cases

Each machine yields cases, and each case is something a test must observe on the running system.
Their ids, titles and effects are defined in `docs/formats.md` section 7.2.

| Kind | One per | The test observes |
|---|---|---|
| `transition` | Each transition branch: `on`, `after`, `always`, invoke `onDone` and `onError`, state `onDone` | The machine moves to `target` (or stays, when internal) and every effect happens. |
| `ignore` | Each `_ignores` entry | The event is refused or absorbed as documented; the state does not change; no effect happens. |
| `refusal` | Each `_refusal` entry | With every guard of the handler false, the state does not change; no effect happens. |
| `clause` | Each clause of the guard of each guarded transition | With only that clause false and every other true, the guarded branch does not fire. |

Branch order is evaluation order: the first branch whose guard holds wins. The effects of a
transition case are the exit actions of the source (when there is a target), the transition's own
actions, then the entry actions of the target (when there is a target).

`machinery gen` writes the cases to `obligations.json` and the runtime table of each machine to
`tables/<machine id>.table.json` (section 7.3). A service looks transitions up in that table. The
`xstate` stage checks that a table means what XState would execute for the same machine.

## Failure idioms

Every side effect is an `invoke`. Listing its outcomes is how failure modes get named.

- **Async operation:** `invoke` with `onDone` and `onError`. A missing `onError` is an error.
- **Timeout:** an `after` on the invoking state. Every invoking state needs one.
- **Retry with backoff:** `onError` to a `retrying` state, an `after` back to the operation, and a
  guard such as `retriesExhausted` that routes to a terminal failure.
- **Compensation:** `onError` to a `compensating` state that invokes the rollback, then to a stable
  state.
- **Degrade:** route to a `degraded` state that serves a fallback rather than failing hard.

A per-state `after` is cancelled when its state exits. A bound that must hold across several
states needs a timer on each of them.

## Fail closed by contract

A component that reports a verdict reports success only when every applicable check passed. Mark
which states are which with XState `tags` (a string or an array of strings):

- `accepting`: the state reports caller-facing success.
- `unchecked` or `omitted`: an applicable check has not run, or an input it needs was absent.

The lint reports an ERROR for:

- a transition of any kind (`on`, `after`, `always`, invoke `onDone`/`onError`, a compound's
  `onDone`) from an `unchecked` or `omitted` state, or a descendant of one, that enters an
  `accepting` state (entering a compound whose initial child is accepting counts);
- an `accepting` state entered at machine start (the root `initial` and its initial child chain);
- a state tagged both `accepting` and `unchecked` or `omitted`;
- a `tags` value that is neither a string nor an array of non-empty strings.

Route a not-checked path to a failing outcome, or to an explicit not-checked state the caller sees
as such. Other tags are inert, and a machine without these tags is unaffected.

## Named units

Every guard, action and actor name the machine uses is implemented in code under that exact name.
The runtime table lists them under `units`, sorted. The contracts of those units (what a guard
reads, what an actor writes) live with the code and its tests, not in the machine file.

## Workflow

1. Edit the machine file.
2. `machinery lint --stage machines`, until it is clean.
3. `machinery gen` to refresh the cases and tables. It refuses to write while the stage has an
   ERROR.
4. Commit the machine and the generated files together; `machinery gen --check` fails on drift.
