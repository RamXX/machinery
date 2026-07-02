---- MODULE Task ----
EXTENDS Naturals

\* Generated from Task.machine.json by tools/tla_gen.py. Control-flow model.
CONSTANT MaxRetries
VARIABLES st, rc
vars == << st, rc >>

States == {"Cancelled", "Done", "InProgress", "Open", "persistRetry", "persisting", "rolledBack"}
Domain == {"Cancelled", "Done", "InProgress", "Open"}
Overlay == {"persistRetry", "persisting", "rolledBack"}
Final == {"Cancelled", "Done"}

TypeOK == st \in States /\ rc \in 0..MaxRetries
Init == st = "Open" /\ rc = 0

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

T1 == st = "Open" /\ st' = "persisting" /\ rc' = 0
T2 == st = "Open" /\ st' = "Open" /\ rc' = 0
T3 == st = "Open" /\ st' = "persisting" /\ rc' = 0
T4 == st = "Open" /\ st' = "Open" /\ rc' = 0
T5 == st = "Open" /\ st' = "persisting" /\ rc' = 0
T6 == st = "Open" /\ st' = "Open" /\ rc' = 0
T7 == st = "Open" /\ st' = "persisting" /\ rc' = 0
T8 == st = "Open" /\ st' = "Open" /\ rc' = 0
T9 == st = "InProgress" /\ st' = "InProgress" /\ rc' = 0
T10 == st = "InProgress" /\ st' = "persisting" /\ rc' = 0
T11 == st = "InProgress" /\ st' = "InProgress" /\ rc' = 0
T12 == st = "InProgress" /\ st' = "persisting" /\ rc' = 0
T13 == st = "InProgress" /\ st' = "InProgress" /\ rc' = 0
T14 == st = "InProgress" /\ st' = "persisting" /\ rc' = 0
T15 == st = "InProgress" /\ st' = "InProgress" /\ rc' = 0
T16 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T17 == st = "persisting" /\ st' = "Open" /\ rc' = 0
T18 == st = "persisting" /\ st' = "InProgress" /\ rc' = 0
T19 == st = "persisting" /\ st' = "Done" /\ rc' = 0
T20 == st = "persisting" /\ st' = "Cancelled" /\ rc' = 0
T21 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T22 == st = "persisting" /\ st' = "persistRetry" /\ rc' = rc
T23 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T24 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T25 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T26 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T27 == st = "rolledBack" /\ st' = "Open" /\ rc' = 0
T28 == st = "rolledBack" /\ st' = "InProgress" /\ rc' = 0
RetryExhausted == st = "persistRetry" /\ rc >= MaxRetries /\ st' = "rolledBack" /\ rc' = rc
RetryAgain == st = "persistRetry" /\ rc < MaxRetries /\ st' = "persisting" /\ rc' = rc + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15
OverlayNext == T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ T26 \/ T27 \/ T28 \/ RetryExhausted \/ RetryAgain
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====