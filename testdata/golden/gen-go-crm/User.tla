---- MODULE User ----
EXTENDS Naturals

\* Generated from User.machine.json by tools/tla_gen.py. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      - UNVERIFIED, state rolledBack: both domain states (Active, Disabled) can enter the persist overlay, so priorStatus ranges over {Active, Disabled}; both priorIs* guards are present
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

States == {"Active", "Disabled", "persistRetry", "persisting", "rolledBack"}
Domain == {"Active", "Disabled"}
Overlay == {"persistRetry", "persisting", "rolledBack"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Active" /\ rc1 = 0

  \* T1: Active -on:disable-> persisting
  \* T2: Active -on:disable-> Active
  \* T3: Active -on:enable-> Active
  \* T4: Disabled -on:enable-> persisting
  \* T5: Disabled -on:enable-> Disabled
  \* T6: Disabled -on:disable-> Disabled
  \* T7: persisting -after:persistTimeout-> rolledBack
  \* T8: persisting -onDone:saveUser-> Active
  \* T9: persisting -onDone:saveUser-> Disabled
  \* T10: persisting -onDone:saveUser-> rolledBack
  \* T11: persisting -onError:saveUser-> persistRetry
  \* T12: persisting -onError:saveUser-> rolledBack
  \* T13: persisting -onError:saveUser-> rolledBack
  \* T14: persisting -onError:saveUser-> rolledBack
  \* T15: persisting -onError:saveUser-> rolledBack
  \* T16: rolledBack -always-> Active
  \* T17: rolledBack -always-> Disabled

T1 == st = "Active" /\ st' = "persisting" /\ rc1' = 0
T2 == st = "Active" /\ st' = "Active" /\ rc1' = 0
T3 == st = "Active" /\ st' = "Active" /\ rc1' = 0
T4 == st = "Disabled" /\ st' = "persisting" /\ rc1' = 0
T5 == st = "Disabled" /\ st' = "Disabled" /\ rc1' = 0
T6 == st = "Disabled" /\ st' = "Disabled" /\ rc1' = 0
T7 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T8 == st = "persisting" /\ st' = "Active" /\ rc1' = 0
T9 == st = "persisting" /\ st' = "Disabled" /\ rc1' = 0
T10 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T11 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T12 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T13 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T14 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T15 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T16 == st = "rolledBack" /\ st' = "Active" /\ rc1' = 0
T17 == st = "rolledBack" /\ st' = "Disabled" /\ rc1' = 0
RetryExhausted_persistRetry == st = "persistRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc1 < MaxRetries /\ st' = "persisting" /\ rc1' = rc1 + 1

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6
OverlayNext == T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
