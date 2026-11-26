# Machine format

A machine is one JSON file, `<Name>.machine.json`, in the machines directory
(`design.machines` in `machinery.yaml`). It uses the JSON-serializable subset of XState v5, with
semantics pinned to XState 5.33.4. Guards, actions and actors are named by string and implemented in
code; the file describes structure only. A file in this subset can be pasted into the Stately
visualizer after its `_`-prefixed keys are removed.

`docs/formats.md` is the contract: section 5.2 (what a machine is), 6.2 (the `machines` stage), 7.2
(cases) and 7.3 (runtime tables). Machines kept in a JSON dialect of the project's own are read
through `import.machines` and translated into this format in memory (section 5.2.1). This guide is the working reference for authors. `machinery lint`
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
- **Reachability.** Every state can be reached from the initial state or from an entry state
  (`_entries`). A non-final leaf must have a transition that leaves it (no dead ends, no
  absorbing states), unless it declares `_terminal`. A final state has no transitions.
- **Entries.** `_entries` names existing top-level states other than `initial`, each once.
- **Terminal states.** `_terminal` is a reason on a non-final leaf state that has transitions and
  none that leaves it.
- **Invokes.** Every `invoke` has a `src`, an `onError`, and its state has a non-empty `after`.
- **Branch order.** An unguarded branch is last, no two branches of a list share a guard, and no
  branch list is empty.
- **Always cycles.** A cycle of unguarded `always` transitions loops forever and is an error.
- **Event completeness.** Every resting state handles or ignores every event (`_ignores`).
- **Guard-false dispositions.** A fully guarded handler declares `_refusal`; a fully guarded
  `always` list declares `_exhaustive`.
- **Guard clauses.** `_guards` is well formed and names guards the machine uses.
- **Fail-closed tags.** Not-checked states never reach an accepting state.
- **Failure states and events.** No state is tagged both `failure` and `accepting`.
  `_failure_events` names events some state handles or ignores, none of them also in
  `_external_events`.
- **Aliases.** `_aliases` maps events some state handles or ignores to action names, none of
  them also in `_external_events` or `_failure_events`.
- **Dismissed hints.** `_not_failures` maps events and state paths the machine has to reasons,
  none of them a declared failure.
- **Callers.** `_requires` is a non-empty list of distinct, non-empty names.
- **Serves.** `_serves` is a non-empty list of distinct `entity:<Name>` and
  `act:<Entity>.<action>` ids.
- **Aspects.** `_aspect` is a short name matching `^[A-Za-z][A-Za-z0-9_-]*$`.
- **Collisions.** Two machines may not share a case-id tag or a machine id.

Name hints print as `warn   <file>: <message>` and never fail the stage (see "Failure idioms").

The stage prints `checked: <n> machines, <n> states, <n> transitions`. It is not applicable when
`design.machines` is absent, and not demonstrated (`nothing to check`) when the directory holds no
machine file. A file that cannot be parsed is an ERROR.

## The subset

- **Root keys:** `id`, `initial`, `context`, `states`, `description`, `meta`, `version`, and the
  annotations `_comment`, `_delays`, `_lifecycle_of`, `_aspect`, `_role`, `_serves`, `_oracle_tag`,
  `_guards`, `_external_events`, `_failure_events`, `_aliases`, `_not_failures`, `_entries`.
- **State keys:** `on`, `after`, `always`, `invoke`, `entry`, `exit`, `states`, `initial`, `type`, `id`,
  `meta`, `description`, `tags`, `onDone`, `output`, and the annotations `_comment`, `_exhaustive`,
  `_ignores`, `_refusal`, `_terminal`.
- **Transition keys:** `target`, `guard`, `actions`, `description`, `_comment`, `_requires`. A typo such as
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

Every machine binds to the model, as one of two kinds (`docs/formats.md` section 5.2):

- a **lifecycle machine** of a model entity: the root key `_lifecycle_of: "<Entity>"`, or a file
  base name equal to the entity name. Top-level states whose name starts with an uppercase letter
  are the **domain states**; lowercase-first states are transient overlay states.
- an **operational machine**: `_role: "operational"`, for example a command-execution envelope,
  with `_serves` naming what it exists for.

An entity whose state lives in more than one attribute (an order's payment and its delivery, a
parcel and its customs check) may have one lifecycle machine per attribute: each is an
**aspect** of the lifecycle and declares `_aspect`, a short name unique for the entity. Each
aspect's domain states are the values of an enum of its own, and every action of the entity is
an event of at least one aspect. A second lifecycle machine without `_aspect` is an error.

`machinery lint` reports a machine that binds to nothing (rule `machine-binding`), and an entity
that declares actions and has no lifecycle machine (rule `entity-machine`; an entity with no
state of its own is waived, with the reason).

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
is a stale marker and an error: remove it, or wire the event. An action of the model is a caller's
command and is never listed here: stage `xref-model-machine` reports one that is, for every
action of the machine's entity, or of every entity behind what an operational machine serves.

**`_aliases`** (root, optional). An object that maps an event name to the name of the model
action it is: `"_aliases": {"handed_back": "giveBack"}` when the machine calls `handed_back` what
the model calls `giveBack`. Rules `event-action` and `action-event` then read the event as that
action (`docs/formats.md` section 6.4), which must be an action of the machine's entity, or of
what an operational machine serves. Each value is a non-empty string, and each event is one some
state handles or ignores (a stale entry is an error). An aliased event is a caller's action, so it
is in neither `_external_events` nor `_failure_events`. An alias hides nothing: an event that is
itself an action of the model takes no alias, an event that states only ignore takes none (it
would stand for nothing), an action that has an event of its own name is the
action of no alias (several events may share an action that has none), the stage prints one line
for every alias in effect and counts them, and an aliased event whose name looks like a failure
is still hinted. Prefer one name in the model and the machine; the alias is for a machine whose
names cannot change.

**`_not_failures`** (root, optional). An object that maps a name to the reason it only looks
like a failure, which dismisses its name hint ("Failure idioms"):
`"_not_failures": {"count_aborted": "a stocktake that is called off changes no loan"}`. A key is
an event some state handles or ignores, or the dotted path of a state (a stale key is an error);
a reason is a non-empty string. A declared failure is not dismissed: a key that is in
`_failure_events`, or a state tagged `failure`, is an error. So is a key that dismisses nothing:
a name with no failure word in it, or an action of the lifecycle entity, which is never hinted.

**`_serves`** (root, required on an operational machine). A non-empty array of the model ids the
machine exists for, each `entity:<Name>` or `act:<Entity>.<action>`, none twice:
`"_serves": ["act:Order.pay", "act:Order.ship"]` for an envelope that runs those commands. Stage
`machines` holds the shape; stage `xref-model-machine` holds that each id is in the model. It is
an error on a lifecycle machine, which serves its entity.

**`_aspect`** (root, optional). A short name, `^[A-Za-z][A-Za-z0-9_-]*$`, on a lifecycle machine
that holds one aspect of its entity's lifecycle: `"_aspect": "scan"`. Two lifecycle machines of
one entity both declare it, with different names. Stage `machines` holds the shape; stage
`xref-model-machine` holds the rest (`docs/formats.md` section 6.4). It is an error on an
operational machine.

**`_failure_events`** (root, optional). An array of event names that report a failure arriving
from outside: a dependency failed, a crash was detected, a lease was lost. The shape and the
staleness rule are those of `_external_events`, and an event may not be in both lists: a failure
event is external already. See "Failure idioms".

**`_entries`** (root, optional). A non-empty array of top-level state names, besides `initial`, in
which an instance may be created directly: an imported or migrated instance. Each names an
existing top-level state, none is the initial state, none appears twice. Reachability starts from
the initial state and from every entry. The runtime table lists them under `entries`, and the TLA+
skeleton lets an instance start in any of them (`docs/formats.md` sections 7.3 and 16.2).

```json
"_entries": ["Active"]
```

**`_terminal`** (state). A reason string on a non-final leaf state that is never left by design but
still answers events: a durable resource keeps refusing after its end. Without it such a state is
an absorbing state, an error. The claim is checked: `_terminal` on a final state, on a compound
state, on a state a transition leaves, or on a state with no transition at all (a dead end) is an
error, and so is an empty reason. Event completeness applies to it as to any resting state.

```json
"Closed": {
  "_terminal": "a closed account is kept for audit; every later request is refused with ACCOUNT_CLOSED",
  "on": {"deposit": {"actions": "recordRefusal"}, "close": {"actions": "recordRefusal"}}
}
```

**`_requires`** (transition branch, optional). A non-empty array of distinct, non-empty strings
naming who may cause the transition: an actor or a role, or `system/<name>` for an internal
caller. The lint checks the shape only. The runtime table row carries it, sorted, as `requires`.

```json
"approve": {"target": "Approved", "_requires": ["Manager", "system/autoApprover"]}
```

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
transition case are the named actions XState executes for it, from the source's initial leaf: the
exit actions of the states left (innermost first), the transition's own actions, then the entry
actions of the states entered (outermost first, down through the target's initial children). A
target that is the source or one of its descendants does not exit the source, so a self-target on
an atomic state runs only its own actions. `docs/formats.md` section 7.2 has the exact rule.

`machinery gen` writes the cases to `obligations.json` and the runtime table of each machine to
`tables/<machine id>.table.json` (section 7.3). A service looks transitions up in that table. The
`xstate` stage checks that a table means what XState would execute for the same machine.

## Failure idioms

A failure reaches a machine in one of two ways, and each has an idiom. In both, tag the state that
records the failure with the XState tag `failure`. Stage `xref-failure` (`docs/formats.md` section
6.6) then asks for a failure mode for each failure state and each failure event: what the system
must do when it happens. A state tagged both `failure` and `accepting` is an error: a state that
records that something went wrong cannot report caller-facing success.

**The invoke idiom.** The machine calls out itself. Every side effect is an `invoke`, and listing
its outcomes is how failure modes get named.

- **Async operation:** `invoke` with `onDone` and `onError`. A missing `onError` is an error.
- **Timeout:** an `after` on the invoking state. Every invoking state needs one.
- **Retry with backoff:** `onError` to a `retrying` state, an `after` back to the operation, and a
  guard such as `retriesExhausted` that routes to a failure state.
- **Compensation:** `onError` to a `compensating` state that invokes the rollback, then to a stable
  state.
- **Degrade:** route to a `degraded` state that serves a fallback rather than failing hard.

```json
"charging": {
  "invoke": {
    "src": "chargePayment",
    "onDone": {"target": "Paid", "actions": "recordCharge"},
    "onError": {"target": "PaymentFailed", "actions": "recordPaymentFailure"}
  },
  "after": {"payTimeout": {"target": "PaymentFailed", "actions": "recordPaymentTimeout"}}
},
"PaymentFailed": {
  "tags": "failure",
  "on": {"retry": {"target": "charging"}, "cancel": {"target": "Cancelled"}}
}
```

The failure modes name `invoke:order.chargePayment` and `state:order.PaymentFailed`.

**The event idiom.** Someone else does the work and reports back: a worker, a supervisor or a lease
manager sends an event into a resting state (`jobFailed`, `crashDetected`, `leaseLost`). List
those events in `_failure_events` and tag the state they lead to `failure`:

```json
{
  "id": "job",
  "_role": "operational",
  "_serves": ["act:Report.generate"],
  "initial": "Queued",
  "_failure_events": ["jobFailed"],
  "states": {
    "Queued": {
      "on": {"claim": {"target": "Running", "actions": "grantLease", "_requires": ["system/worker"]}},
      "_ignores": {
        "finish": "nothing runs yet; a stray result is refused with JOB_NOT_RUNNING",
        "jobFailed": "nothing runs yet; a stray report is refused with JOB_NOT_RUNNING",
        "retry": "the job is already queued"
      }
    },
    "Running": {
      "on": {
        "finish": {"target": "Done", "actions": "recordResult", "_requires": ["system/worker"]},
        "jobFailed": {"target": "Failed", "actions": "recordFailure", "_requires": ["system/worker"]}
      },
      "_ignores": {
        "claim": "already claimed; a second worker is refused with JOB_CLAIMED",
        "retry": "the job is running; refused with JOB_RUNNING"
      }
    },
    "Failed": {
      "tags": "failure",
      "on": {"retry": {"target": "Queued", "actions": "recordRetry", "_requires": ["Operator"]}},
      "_ignores": {
        "claim": "a failed job is retried first; refused with JOB_FAILED",
        "finish": "a late result of a failed run is dropped",
        "jobFailed": "already failed; the repeated report is dropped"
      }
    },
    "Done": {"type": "final"}
  }
}
```

The failure modes name `event:job.jobFailed` (what arrives) and `state:job.Failed` (what is
recorded). A failure event arrives from outside, so it needs no model action
(`docs/formats.md` section 6.4), and a test makes it happen in each state that handles or ignores
it.

**Name hints.** The lint warns about a state whose name contains `fail`, `error`, `abort`,
`crash` or `timeout` (in any case) and is not tagged `failure`, and about an event with such a
name that is neither in `_failure_events` nor an action of the lifecycle entity; an event in
`_external_events` or in `_aliases` is still hinted. When the name means something else, dismiss
the hint with the reason in the root `_not_failures`, keyed by the event name or the dotted path
of the state. A warning never fails the stage; tag the state, list the event, or dismiss the hint
when the name means something else.

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
as such. The tag `failure` is described in "Failure idioms". Other tags are inert, and a machine
without these tags is unaffected.

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
