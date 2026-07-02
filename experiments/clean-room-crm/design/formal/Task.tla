---- MODULE Task ----
EXTENDS Naturals

\* Generated from Task.machine.json by tools/tla_gen.py. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: sound for safety; for liveness the
\*      guard lists must be exhaustive. machine_lint enforces an unguarded
\*      fallback or an _exhaustive note on every fully guarded always-list.
\*      - rolledBack: prior is set by every setPending action to the state the operation departed from, which for Task is only Open or InProgress; terminal states issue no commands
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

States == {"Abandoned", "Done", "InProgress", "Open", "persistRetry", "persisting", "rolledBack"}
Domain == {"Abandoned", "Done", "InProgress", "Open"}
Overlay == {"persistRetry", "persisting", "rolledBack"}
Final == {"Abandoned", "Done"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Open" /\ rc1 = 0

  \* T1: Open -on:start-> persisting
  \* T2: Open -on:abandon-> persisting
  \* T3: InProgress -on:complete-> persisting
  \* T4: InProgress -on:abandon-> persisting
  \* T5: persisting -after:PERSIST_TIMEOUT-> persistRetry
  \* T6: persisting -onDone:persist-> InProgress
  \* T7: persisting -onDone:persist-> Done
  \* T8: persisting -onDone:persist-> Abandoned
  \* T9: persisting -onError:persist-> persistRetry
  \* T10: persisting -onError:persist-> rolledBack
  \* T11: rolledBack -always-> Open
  \* T12: rolledBack -always-> InProgress

T1 == st = "Open" /\ st' = "persisting" /\ rc1' = 0
T2 == st = "Open" /\ st' = "persisting" /\ rc1' = 0
T3 == st = "InProgress" /\ st' = "persisting" /\ rc1' = 0
T4 == st = "InProgress" /\ st' = "persisting" /\ rc1' = 0
T5 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T6 == st = "persisting" /\ st' = "InProgress" /\ rc1' = 0
T7 == st = "persisting" /\ st' = "Done" /\ rc1' = 0
T8 == st = "persisting" /\ st' = "Abandoned" /\ rc1' = 0
T9 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T10 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T11 == st = "rolledBack" /\ st' = "Open" /\ rc1' = 0
T12 == st = "rolledBack" /\ st' = "InProgress" /\ rc1' = 0
RetryExhausted_persistRetry == st = "persistRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc1 < MaxRetries /\ st' = "persisting" /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4
OverlayNext == T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
