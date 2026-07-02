---- MODULE CommandExecution ----
EXTENDS Naturals

\* Generated from CommandExecution.machine.json by tools/tla_gen.py. Control-flow model.
CONSTANT MaxRetries
VARIABLES st, rc
vars == << st, rc >>

States == {"Authorizing", "Corrupt", "DBError", "DBLocked", "Denied", "Done", "Executing", "Opening", "Parsing", "Rendering", "ResolvingSession", "ValidationFailed"}
Domain == {"Corrupt", "DBError", "Denied", "Done", "ValidationFailed"}
Overlay == {"Authorizing", "DBLocked", "Executing", "Opening", "Parsing", "Rendering", "ResolvingSession"}
Final == {"Corrupt", "DBError", "Denied", "Done", "ValidationFailed"}

TypeOK == st \in States /\ rc \in 0..MaxRetries
Init == st = "Parsing" /\ rc = 0

  \* T1: Parsing -always-> Opening
  \* T2: Parsing -always-> ValidationFailed
  \* T3: Opening -after:openTimeout-> DBError
  \* T4: Opening -onDone:openDatabase-> ResolvingSession
  \* T5: Opening -onError:openDatabase-> DBLocked
  \* T6: Opening -onError:openDatabase-> Corrupt
  \* T7: Opening -onError:openDatabase-> DBError
  \* T8: Opening -onError:openDatabase-> DBError
  \* T9: ResolvingSession -after:sessionResolveTimeout-> DBError
  \* T10: ResolvingSession -onDone:resolveSession-> Authorizing
  \* T11: ResolvingSession -onError:resolveSession-> Denied
  \* T12: ResolvingSession -onError:resolveSession-> Denied
  \* T13: ResolvingSession -onError:resolveSession-> DBLocked
  \* T14: ResolvingSession -onError:resolveSession-> DBError
  \* T15: Authorizing -always-> Executing
  \* T16: Authorizing -always-> Denied
  \* T17: Executing -after:queryTimeout-> DBError
  \* T18: Executing -onDone:executeInTx-> Rendering
  \* T19: Executing -onError:executeInTx-> ValidationFailed
  \* T20: Executing -onError:executeInTx-> DBLocked
  \* T21: Executing -onError:executeInTx-> DBLocked
  \* T22: Executing -onError:executeInTx-> DBError
  \* T23: Executing -onError:executeInTx-> DBError
  \* T24: Executing -onError:executeInTx-> DBError
  \* T25: Rendering -always-> Done

T1 == st = "Parsing" /\ st' = "Opening" /\ rc' = rc
T2 == st = "Parsing" /\ st' = "ValidationFailed" /\ rc' = 0
T3 == st = "Opening" /\ st' = "DBError" /\ rc' = 0
T4 == st = "Opening" /\ st' = "ResolvingSession" /\ rc' = rc
T5 == st = "Opening" /\ st' = "DBLocked" /\ rc' = rc
T6 == st = "Opening" /\ st' = "Corrupt" /\ rc' = 0
T7 == st = "Opening" /\ st' = "DBError" /\ rc' = 0
T8 == st = "Opening" /\ st' = "DBError" /\ rc' = 0
T9 == st = "ResolvingSession" /\ st' = "DBError" /\ rc' = 0
T10 == st = "ResolvingSession" /\ st' = "Authorizing" /\ rc' = rc
T11 == st = "ResolvingSession" /\ st' = "Denied" /\ rc' = 0
T12 == st = "ResolvingSession" /\ st' = "Denied" /\ rc' = 0
T13 == st = "ResolvingSession" /\ st' = "DBLocked" /\ rc' = rc
T14 == st = "ResolvingSession" /\ st' = "DBError" /\ rc' = 0
T15 == st = "Authorizing" /\ st' = "Executing" /\ rc' = rc
T16 == st = "Authorizing" /\ st' = "Denied" /\ rc' = 0
T17 == st = "Executing" /\ st' = "DBError" /\ rc' = 0
T18 == st = "Executing" /\ st' = "Rendering" /\ rc' = rc
T19 == st = "Executing" /\ st' = "ValidationFailed" /\ rc' = 0
T20 == st = "Executing" /\ st' = "DBLocked" /\ rc' = rc
T21 == st = "Executing" /\ st' = "DBLocked" /\ rc' = rc
T22 == st = "Executing" /\ st' = "DBError" /\ rc' = 0
T23 == st = "Executing" /\ st' = "DBError" /\ rc' = 0
T24 == st = "Executing" /\ st' = "DBError" /\ rc' = 0
T25 == st = "Rendering" /\ st' = "Done" /\ rc' = 0
RetryExhausted == st = "DBLocked" /\ rc >= MaxRetries /\ st' = "DBError" /\ rc' = rc
RetryAgain == st = "DBLocked" /\ rc < MaxRetries /\ st' = "Opening" /\ rc' = rc + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == FALSE
OverlayNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ RetryExhausted \/ RetryAgain
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====