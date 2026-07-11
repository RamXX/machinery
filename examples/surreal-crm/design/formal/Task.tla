---- MODULE Task ----
EXTENDS Naturals

\* Generated from Task.machine.json by machinery tla. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      - UNVERIFIED, state rolledBack: only Open and InProgress have transitions into the persist overlay, so priorStatus ranges over {Open, InProgress}; both priorIs* guards are present
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

States == {"Cancelled", "Done", "InProgress", "Open", "persistRetry", "persisting", "rolledBack"}
Domain == {"Cancelled", "Done", "InProgress", "Open"}
Overlay == {"persistRetry", "persisting", "rolledBack"}
Final == {"Cancelled", "Done"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Open" /\ rc1 = 0

  \* T1: Open -on:start-> persisting
  \* T2: Open -on:start-> Open
  \* T3: Open -on:complete-> persisting
  \* T4: Open -on:complete-> Open
  \* T5: Open -on:cancel-> persisting
  \* T6: Open -on:cancel-> Open
  \* T7: Open -on:reassign-> persisting
  \* T8: Open -on:reassign-> Open
  \* T9: InProgress -on:start-> InProgress
  \* T10: InProgress -on:complete-> persisting
  \* T11: InProgress -on:complete-> InProgress
  \* T12: InProgress -on:cancel-> persisting
  \* T13: InProgress -on:cancel-> InProgress
  \* T14: InProgress -on:reassign-> persisting
  \* T15: InProgress -on:reassign-> InProgress
  \* T16: persisting -after:persistTimeout-> rolledBack
  \* T17: persisting -onDone:saveTask-> Open
  \* T18: persisting -onDone:saveTask-> InProgress
  \* T19: persisting -onDone:saveTask-> Done
  \* T20: persisting -onDone:saveTask-> Cancelled
  \* T21: persisting -onDone:saveTask-> rolledBack
  \* T22: persisting -onError:saveTask-> persistRetry
  \* T23: persisting -onError:saveTask-> rolledBack
  \* T24: persisting -onError:saveTask-> rolledBack
  \* T25: persisting -onError:saveTask-> rolledBack
  \* T26: persisting -onError:saveTask-> rolledBack
  \* T27: rolledBack -always-> Open
  \* T28: rolledBack -always-> InProgress

T1 == st = "Open" /\ st' = "persisting" /\ rc1' = 0
T2 == st = "Open" /\ st' = "Open" /\ rc1' = 0
T3 == st = "Open" /\ st' = "persisting" /\ rc1' = 0
T4 == st = "Open" /\ st' = "Open" /\ rc1' = 0
T5 == st = "Open" /\ st' = "persisting" /\ rc1' = 0
T6 == st = "Open" /\ st' = "Open" /\ rc1' = 0
T7 == st = "Open" /\ st' = "persisting" /\ rc1' = 0
T8 == st = "Open" /\ st' = "Open" /\ rc1' = 0
T9 == st = "InProgress" /\ st' = "InProgress" /\ rc1' = 0
T10 == st = "InProgress" /\ st' = "persisting" /\ rc1' = 0
T11 == st = "InProgress" /\ st' = "InProgress" /\ rc1' = 0
T12 == st = "InProgress" /\ st' = "persisting" /\ rc1' = 0
T13 == st = "InProgress" /\ st' = "InProgress" /\ rc1' = 0
T14 == st = "InProgress" /\ st' = "persisting" /\ rc1' = 0
T15 == st = "InProgress" /\ st' = "InProgress" /\ rc1' = 0
T16 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T17 == st = "persisting" /\ st' = "Open" /\ rc1' = 0
T18 == st = "persisting" /\ st' = "InProgress" /\ rc1' = 0
T19 == st = "persisting" /\ st' = "Done" /\ rc1' = 0
T20 == st = "persisting" /\ st' = "Cancelled" /\ rc1' = 0
T21 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T22 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T23 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T24 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T25 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T26 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T27 == st = "rolledBack" /\ st' = "Open" /\ rc1' = 0
T28 == st = "rolledBack" /\ st' = "InProgress" /\ rc1' = 0
RetryExhausted_persistRetry == st = "persistRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc1 < MaxRetries /\ st' = "persisting" /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15
OverlayNext == T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ T26 \/ T27 \/ T28 \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
