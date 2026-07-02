---- MODULE CommandExecution ----
EXTENDS Naturals

\* Generated from CommandExecution.machine.json by tools/tla_gen.py. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: sound for safety; for liveness the
\*      guard lists must be exhaustive. machine_lint enforces an unguarded
\*      fallback or an _exhaustive note on every fully guarded always-list.
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

States == {"authenticating", "authorizing", "done", "executing", "failedCorrupt", "failedError", "opening", "refused", "rejected"}
Domain == {"done", "failedCorrupt", "failedError", "refused", "rejected"}
Overlay == {"authenticating", "authorizing", "executing", "opening"}
Final == {"done", "failedCorrupt", "failedError", "refused", "rejected"}

TypeOK == st \in States /\ TRUE
Init == st = "opening"

  \* T1: opening -after:OPEN_TIMEOUT-> failedError
  \* T2: opening -onDone:openDb-> authenticating
  \* T3: opening -onError:openDb-> failedCorrupt
  \* T4: opening -onError:openDb-> failedError
  \* T5: authenticating -after:AUTH_TIMEOUT-> failedError
  \* T6: authenticating -onDone:loadSession-> authorizing
  \* T7: authenticating -onDone:loadSession-> rejected
  \* T8: authenticating -onError:loadSession-> failedError
  \* T9: authorizing -after:AUTHZ_TIMEOUT-> failedError
  \* T10: authorizing -onDone:checkScope-> executing
  \* T11: authorizing -onDone:checkScope-> rejected
  \* T12: authorizing -onError:checkScope-> failedError
  \* T13: executing -after:EXEC_TIMEOUT-> refused
  \* T14: executing -onDone:execute-> done
  \* T15: executing -onError:execute-> refused
  \* T16: executing -onError:execute-> rejected
  \* T17: executing -onError:execute-> failedError

T1 == st = "opening" /\ st' = "failedError"
T2 == st = "opening" /\ st' = "authenticating"
T3 == st = "opening" /\ st' = "failedCorrupt"
T4 == st = "opening" /\ st' = "failedError"
T5 == st = "authenticating" /\ st' = "failedError"
T6 == st = "authenticating" /\ st' = "authorizing"
T7 == st = "authenticating" /\ st' = "rejected"
T8 == st = "authenticating" /\ st' = "failedError"
T9 == st = "authorizing" /\ st' = "failedError"
T10 == st = "authorizing" /\ st' = "executing"
T11 == st = "authorizing" /\ st' = "rejected"
T12 == st = "authorizing" /\ st' = "failedError"
T13 == st = "executing" /\ st' = "refused"
T14 == st = "executing" /\ st' = "done"
T15 == st = "executing" /\ st' = "refused"
T16 == st = "executing" /\ st' = "rejected"
T17 == st = "executing" /\ st' = "failedError"
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == FALSE
OverlayNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
