---- MODULE OrdersContract ----
EXTENDS Naturals

\* Generated from OrdersContract.machine.json by machinery tla. Control-flow model.
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

States == {"Done", "Open", "Paid"}
Domain == {"Done", "Open", "Paid"}
Overlay == {}
Final == {"Done"}

TypeOK == st \in States /\ TRUE
Init == st = "Open"

  \* T1: Open -on:markPaid-> Paid
  \* T2: Open -on:markDeclined-> Done
  \* T3: Open -on:cancel-> Done
  \* T4: Paid -on:ship-> Done

T1 == st = "Open" /\ st' = "Paid"
T2 == st = "Open" /\ st' = "Done"
T3 == st = "Open" /\ st' = "Done"
T4 == st = "Paid" /\ st' = "Done"
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4
OverlayNext == FALSE
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
