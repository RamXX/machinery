---- MODULE Order ----
\* machinery-version: v0.3.4-dev
EXTENDS Naturals

\* Generated from Order.machine.json by machinery tla. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      (none here: every guarded branch list has an unguarded fallback)
\*   2. Every invoke resolves exactly once (onDone or onError; no lost or
\*      duplicated completion) and every after timer eventually fires.
\*   3. Single machine instance; no interleaving with other instances or
\*      machines, no message loss/duplication/reordering between machines.
\*   4. Context data, event payloads, action effects, and real time (the
\*      _delays values) are not modeled at this rung; the data-refined rung
\*      (refine_gen) and the implementation tests carry those.
CONSTANT MaxRetries
VARIABLES st
vars == << st >>

States == {"Cancelled", "Declined", "Paid", "Placed", "Shipped"}
Domain == {"Cancelled", "Declined", "Paid", "Placed", "Shipped"}
Overlay == {}
Final == {"Cancelled", "Declined", "Shipped"}

TypeOK == st \in States /\ TRUE
Init == st = "Placed"

  \* T1: Placed -on:markPaid-> Paid
  \* T2: Placed -on:markDeclined-> Declined
  \* T3: Placed -on:cancel-> Cancelled
  \* T4: Paid -on:ship-> Shipped

T1 == st = "Placed" /\ st' = "Paid"
T2 == st = "Placed" /\ st' = "Declined"
T3 == st = "Placed" /\ st' = "Cancelled"
T4 == st = "Paid" /\ st' = "Shipped"
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4
OverlayNext == FALSE
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
