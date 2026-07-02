---- MODULE Reservation ----
EXTENDS Naturals

\* Generated from Reservation.machine.json by tools/tla_gen.py. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: sound for safety; for liveness the
\*      guard lists must be exhaustive. machine_lint enforces an unguarded
\*      fallback or an _exhaustive note on every fully guarded always-list.
\*      - rolledBack: only Held has transitions into the persist overlay (Committed and Released are final), so priorStatus is always Held and the single priorIsHeld guard is total
\*   2. Every invoke resolves exactly once (onDone or onError; no lost or
\*      duplicated completion) and every after timer eventually fires.
\*   3. Single machine instance; no interleaving with other instances or
\*      machines, no message loss/duplication/reordering between machines.
\*   4. Context data, event payloads, action effects, and real time (the
\*      _delays values) are not modeled at this rung; the data-refined rung
\*      (refine_gen) and the implementation tests carry those.
CONSTANT MaxRetries
VARIABLES st, rc1
vars == << st, rc1 >>

States == {"Committed", "Held", "Released", "persistRetry", "persisting", "rolledBack"}
Domain == {"Committed", "Held", "Released"}
Overlay == {"persistRetry", "persisting", "rolledBack"}
Final == {"Committed", "Released"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Held" /\ rc1 = 0

  \* T1: Held -on:commit-> persisting
  \* T2: Held -on:release-> persisting
  \* T3: persisting -after:persistTimeout-> rolledBack
  \* T4: persisting -onDone:persistReservation-> Committed
  \* T5: persisting -onDone:persistReservation-> Released
  \* T6: persisting -onDone:persistReservation-> rolledBack
  \* T7: persisting -onError:persistReservation-> persistRetry
  \* T8: persisting -onError:persistReservation-> persistRetry
  \* T9: persisting -onError:persistReservation-> rolledBack
  \* T10: rolledBack -always-> Held

T1 == st = "Held" /\ st' = "persisting" /\ rc1' = 0
T2 == st = "Held" /\ st' = "persisting" /\ rc1' = 0
T3 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T4 == st = "persisting" /\ st' = "Committed" /\ rc1' = 0
T5 == st = "persisting" /\ st' = "Released" /\ rc1' = 0
T6 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T7 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T8 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T9 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T10 == st = "rolledBack" /\ st' = "Held" /\ rc1' = 0
RetryExhausted_persistRetry == st = "persistRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc1 < MaxRetries /\ st' = "persisting" /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2
OverlayNext == T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
