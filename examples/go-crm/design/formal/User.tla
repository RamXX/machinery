---- MODULE User ----
EXTENDS Naturals

\* Generated from User.machine.json by tools/tla_gen.py. Control-flow model.
CONSTANT MaxRetries
VARIABLES st, rc
vars == << st, rc >>

States == {"Active", "Disabled", "persistRetry", "persisting", "rolledBack"}
Domain == {"Active", "Disabled"}
Overlay == {"persistRetry", "persisting", "rolledBack"}

TypeOK == st \in States /\ rc \in 0..MaxRetries
Init == st = "Active" /\ rc = 0

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

T1 == st = "Active" /\ st' = "persisting" /\ rc' = 0
T2 == st = "Active" /\ st' = "Active" /\ rc' = 0
T3 == st = "Active" /\ st' = "Active" /\ rc' = 0
T4 == st = "Disabled" /\ st' = "persisting" /\ rc' = 0
T5 == st = "Disabled" /\ st' = "Disabled" /\ rc' = 0
T6 == st = "Disabled" /\ st' = "Disabled" /\ rc' = 0
T7 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T8 == st = "persisting" /\ st' = "Active" /\ rc' = 0
T9 == st = "persisting" /\ st' = "Disabled" /\ rc' = 0
T10 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T11 == st = "persisting" /\ st' = "persistRetry" /\ rc' = rc
T12 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T13 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T14 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T15 == st = "persisting" /\ st' = "rolledBack" /\ rc' = rc
T16 == st = "rolledBack" /\ st' = "Active" /\ rc' = 0
T17 == st = "rolledBack" /\ st' = "Disabled" /\ rc' = 0
RetryExhausted == st = "persistRetry" /\ rc >= MaxRetries /\ st' = "rolledBack" /\ rc' = rc
RetryAgain == st = "persistRetry" /\ rc < MaxRetries /\ st' = "persisting" /\ rc' = rc + 1

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6
OverlayNext == T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ RetryExhausted \/ RetryAgain
Next == DomainNext \/ OverlayNext

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====