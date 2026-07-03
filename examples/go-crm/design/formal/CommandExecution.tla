---- MODULE CommandExecution ----
EXTENDS Naturals

\* Generated from CommandExecution.machine.json by machinery tla. Control-flow model.
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
VARIABLES st, rc1
vars == << st, rc1 >>

States == {"Authorizing", "Corrupt", "DBError", "DBLocked", "Denied", "Done", "Executing", "Opening", "Parsing", "Rendering", "ResolvingSession", "ValidationFailed"}
Domain == {"Corrupt", "DBError", "Denied", "Done", "ValidationFailed"}
Overlay == {"Authorizing", "DBLocked", "Executing", "Opening", "Parsing", "Rendering", "ResolvingSession"}
Final == {"Corrupt", "DBError", "Denied", "Done", "ValidationFailed"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Parsing" /\ rc1 = 0

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

T1 == st = "Parsing" /\ st' = "Opening" /\ rc1' = rc1
T2 == st = "Parsing" /\ st' = "ValidationFailed" /\ rc1' = 0
T3 == st = "Opening" /\ st' = "DBError" /\ rc1' = 0
T4 == st = "Opening" /\ st' = "ResolvingSession" /\ rc1' = rc1
T5 == st = "Opening" /\ st' = "DBLocked" /\ rc1' = rc1
T6 == st = "Opening" /\ st' = "Corrupt" /\ rc1' = 0
T7 == st = "Opening" /\ st' = "DBError" /\ rc1' = 0
T8 == st = "Opening" /\ st' = "DBError" /\ rc1' = 0
T9 == st = "ResolvingSession" /\ st' = "DBError" /\ rc1' = 0
T10 == st = "ResolvingSession" /\ st' = "Authorizing" /\ rc1' = rc1
T11 == st = "ResolvingSession" /\ st' = "Denied" /\ rc1' = 0
T12 == st = "ResolvingSession" /\ st' = "Denied" /\ rc1' = 0
T13 == st = "ResolvingSession" /\ st' = "DBLocked" /\ rc1' = rc1
T14 == st = "ResolvingSession" /\ st' = "DBError" /\ rc1' = 0
T15 == st = "Authorizing" /\ st' = "Executing" /\ rc1' = rc1
T16 == st = "Authorizing" /\ st' = "Denied" /\ rc1' = 0
T17 == st = "Executing" /\ st' = "DBError" /\ rc1' = 0
T18 == st = "Executing" /\ st' = "Rendering" /\ rc1' = rc1
T19 == st = "Executing" /\ st' = "ValidationFailed" /\ rc1' = 0
T20 == st = "Executing" /\ st' = "DBLocked" /\ rc1' = rc1
T21 == st = "Executing" /\ st' = "DBLocked" /\ rc1' = rc1
T22 == st = "Executing" /\ st' = "DBError" /\ rc1' = 0
T23 == st = "Executing" /\ st' = "DBError" /\ rc1' = 0
T24 == st = "Executing" /\ st' = "DBError" /\ rc1' = 0
T25 == st = "Rendering" /\ st' = "Done" /\ rc1' = 0
RetryExhausted_DBLocked == st = "DBLocked" /\ rc1 >= MaxRetries /\ st' = "DBError" /\ rc1' = rc1
RetryAgain_DBLocked == st = "DBLocked" /\ rc1 < MaxRetries /\ st' \in {"Executing", "Opening"} /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == FALSE
OverlayNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ RetryExhausted_DBLocked \/ RetryAgain_DBLocked
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
