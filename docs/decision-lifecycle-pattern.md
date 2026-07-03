# Design draft: `decision-lifecycle` rung-4 refinement pattern

Status: DRAFT for review (not implemented). Prompted by dogfood feedback from hexcrm:
`Proposal` and `ImportBatch` are event-driven decision lifecycles that carry rung-3 proofs
only, because no `machinery refine` pattern fits them.

## 1. Why the three existing patterns do not fit

`refine` dispatches exactly three patterns (`internal/refine/refine.go:1238`):

- `linear-lifecycle` and `saga` model persistence/compensation overlays (invoke, retry, rollback).
- `terminal-lifecycle` (`refine.go:540-560`) is the closest, but it requires an ordered `phases`
  forward chain with `initial == phases[0]` and a **success/failure split**
  (`success_terminal` + `failure_terminals`). Its TLA+ (`refine.go:722-828`) advances phase i to
  phase i+1 and models failures with retry/failRoute machinery.

An event-driven decision lifecycle is a different shape: one or more **resting states** that fan out
to **co-equal terminal outcomes** on **human command events**. Approved / Rejected / Withdrawn are
not success-vs-failure, there is no ordered phase march, and the transitions are `on` commands, not
`invoke`/`onDone`. Forcing it into `terminal-lifecycle` mislabels a legitimate outcome as a
"failure", caps outcomes at the success/failure binary, and drags in retry semantics that do not
apply. Hence the gap.

## 2. Scope: the shape this pattern covers

A machine qualifies as a decision lifecycle when:

- Top-level states partition into **decision states** (non-final) and **outcomes** (final).
- `initial` is a decision state.
- Every transition out of a decision state is an `on` **command** event (no `invoke`, `after`,
  or `always` in v1; those belong to `linear-lifecycle`). Guarded branches on one command are
  allowed (they abstract to nondeterministic choice among that command's targets).
- Command targets are decision states or outcomes. Back-edges between decision states (a
  `revise` loop: `PendingReview --revise--> Draft`) are allowed.

Out of scope for v1 (candidates for v2, noted so the boundary is explicit): timeout-driven
auto-expiry (`after` to an outcome), nested/compound decision states, and decision lifecycles that
also carry a persistence overlay (model the overlay with `linear-lifecycle` on a separate machine).

## 3. Semantics schema

```yaml
pattern: decision-lifecycle
action: <ModelithActionName>        # required by all patterns, for machine reconciliation
decision_states: [Draft, PendingReview]
outcomes: [Approved, Rejected, Withdrawn]
overlay:
  decided: resolved                 # boolean flag name; default "decided"
```

Nothing is hardcoded to a domain: every state name comes from the annotation and is reconciled
against the machine, matching the convention in the other patterns.

## 4. Reconciliation rules (`ReconcileDecision`)

Fail loudly (`die`) on any mismatch, mirroring `ReconcileTerminal`:

1. `decision_states` and `outcomes` both non-empty; declared sets are disjoint.
2. Domain match: machine's upper-first top states `==` `decision_states ∪ outcomes` (`setEq`).
3. Every `outcome` is a `final` state; no `decision_state` is final.
4. `machine.initial ∈ decision_states`.
5. Every decision state's transitions are `on` commands only; reject `invoke`/`after`/`always` with
   a message pointing at `linear-lifecycle`.
6. Every command target is in `decision_states ∪ outcomes` (structural; lint already guarantees no
   dangling).
7. Completeness: every declared outcome is the target of some command (no unreachable outcome), and
   every reached terminal is a declared outcome (no undeclared terminal).
8. No-trap: every decision state has at least one command path toward an outcome (no decision state
   that can only cycle among decision states).

## 5. TLA+ emission (`EmitDecision`), consistent with the codebase

Module `<mid>Data`, same skeleton as `terminal-lifecycle` (header with
GENERATED / RECONCILED / Proves / STILL ASSUMED, `TypeOK`, `Init`, action operators, `Terminated`,
`Prog`/`Next`/`Spec`).

```
Decisions == { ...decision_states... }
Outcomes  == { ...outcomes... }
Terminal  == Outcomes
VARIABLES st, decided
vars == << st, decided >>

TypeOK == st \in (Decisions \cup Outcomes) /\ decided \in BOOLEAN
Init   == st = "<initial>" /\ decided = FALSE

\* one operator per (decision state D, command c); guards erased -> nondeterministic
\* choice among c's reconciled targets T1..Tk
Cmd_D_c == st = "D" /\ st' \in { "T1", ..., "Tk" } /\ decided' = (decided \/ st' \in Outcomes)

Terminated == st \in Terminal /\ UNCHANGED vars
Prog       == Cmd_... \/ ...                 \* all command operators
Decisive   == \* disjunction of the command operators whose target set meets Outcomes
Next       == Prog \/ Terminated
Spec       == Init /\ [][Next]_vars /\ SF_vars(Decisive)
```

Properties emitted (the rung-4 payload):

- `TypeOK` (INVARIANT).
- `Inv_DecidedTracksTerminal == decided <=> (st \in Terminal)` (INVARIANT). The flag means exactly
  "a decision has been recorded"; it is monotone because outcomes are absorbing.
- `Inv_TerminalAbsorbing == [][ (st \in Terminal) => (st' = st) ]_st` (PROPERTY). The decision is
  irreversible.
- `Live_Decides == (st \notin Terminal) ~> (st \in Terminal)` (PROPERTY), under `SF_vars(Decisive)`.

Completeness (every declared outcome reachable) is asserted structurally at reconcile time and
additionally falls out of TLC's state-space exploration.

## 6. What it proves, and what it does not (STILL ASSUMED header)

Proves: the decision is irreversible (terminal absorption); the `decided`/`resolved` flag exactly
tracks whether a decision was recorded and never reverts (matching the Modelith "resolved stays
resolved" invariant); every declared outcome is reachable and no undeclared terminal exists.

Does NOT prove, stated explicitly in the header:
- **Unconditional termination.** A `revise` loop (PendingReview -> Draft -> PendingReview) can run
  forever; that is intended (a human may keep revising). Liveness is therefore conditional on strong
  fairness of the **decisive** commands: if a decisive command is eventually taken, an outcome is
  reached. Rung 3 already proves the raw machine has no deadlock and no stuck-half-done state.
- Single-instance execution; no cross-instance interleaving (same caveat as every other pattern).
- The semantics of each command guard in code (carried by the named-unit contracts into tests, as
  usual).

## 7. Mapping to the two hexcrm entities (acceptance cases)

- **Proposal** (confirm exact state names against hexcrm):
  `decision_states: [Draft, PendingReview]`, `outcomes: [Approved, Rejected, Withdrawn]`,
  `overlay.decided: resolved`. Commands: `submit` (Draft->PendingReview), `approve`/`reject`
  (PendingReview->Approved/Rejected), `withdraw` (->Withdrawn), optional `revise`
  (PendingReview->Draft, a back-edge). Rung-4 then proves resolved is irreversible and tracks
  terminality; liveness is conditional on someone eventually deciding.
- **ImportBatch**: same pattern with its own decision state(s) and outcomes (e.g. an
  awaiting-confirmation state fanning to Imported / Discarded / Failed on `confirm`/`discard`).
  Confirm the exact shape from the hexcrm model; the pattern accommodates 1..n decision states and
  2..n outcomes.

Both entities become the golden/example fixtures for the pattern (see below), which is the real test
that the design holds on independent designs.

## 8. Implementation plan

1. `internal/refine/refine.go`: add `ReconcileDecision` and `EmitDecision`; add
   `case "decision-lifecycle"` to the dispatch (`refine.go:1238`) and to the unsupported-pattern
   message.
2. Tests: `internal/refine/*_test.go` reconcile + emit tests (a clean fixture, plus mismatch cases:
   non-final outcome, invoke in a decision state, unreachable outcome, decision-state trap).
3. Adversarial regression: a `RefineExperiments` row (`internal/experiments/experiments.go:95`) +
   runner, e.g. "decision-lifecycle outcome not final" -> reconciliation fails.
4. Example: add a small `decision-lifecycle` machine + semantics under an example design (or a new
   `examples/` entry modeled on Proposal), so `make verify-formal` TLC-checks it and the golden
   corpus captures `refine` output for it.
5. Docs: one line in the skill's pattern list and in `references/xstate-format.md`; note in the
   README that rung-4 coverage is per-pattern and list the four patterns.

Rough effort once this design is accepted: a few hours of agent work, dominated by `EmitDecision`
plus its tests and the example.

## 9. Open questions for review

1. **Liveness stance.** Is "conditional on strong fairness of decisive commands" the right honesty
   level, or do you want v1 to only accept **loop-free** decision lifecycles (no back-edges) and
   prove unconditional `Live_Decides`, deferring revise-loops to v2? Loop-free is simpler and proves
   more; it may not fit Proposal if it has a revise edge.
2. **Timeout auto-expiry.** Do either of your entities expire on `after` (e.g. a proposal
   auto-withdraws after N days)? If so, v1 should allow one `after`-to-outcome edge per decision
   state rather than defer it.
3. **Flag semantics.** Is a single `decided`/`resolved` boolean enough, or does an entity need to
   record **which** outcome (an enum), which would change `Inv_DecidedTracksTerminal` into a
   per-outcome mapping invariant?
