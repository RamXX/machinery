---- MODULE OutboxMessage ----
\* machinery-version: v0.3.5-dev
EXTENDS Naturals

\* Generated from OutboxMessage.machine.json by machinery tla. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      - UNVERIFIED, state rolledBack: priorStatus is set on every path into the overlay from a domain state; only Pending and Published reach the overlay (Consumed is final), and both priorIs* guards are present; a rollback to Pending is precisely the at-least-once re-drive point for the next poller sweep
\*   2. Every invoke resolves exactly once (onDone or onError; no lost or
\*      duplicated completion) and every after timer eventually fires.
\*   3. Single machine instance; no interleaving with other instances or
\*      machines, no message loss/duplication/reordering between machines.
\*   4. Context data, event payloads, action effects, and real time (the
\*      _delays values) are not modeled at this rung; the data-refined rung
\*      (refine_gen) and the implementation tests carry those.
\*   5. Retry counters (rc*) reset to 0 on every transition that leaves from
\*      or lands on a domain state; a counter surviving a domain hop is not
\*      representable at this rung.
\*   6. Retry-shaped states (fully guarded always + after) are modeled as the
\*      concrete bounded loop: the guarded always list is replaced by the
\*      exhaustion test rc >= MaxRetries and the after timer by the retry step
\*      rc < MaxRetries; the guards themselves are erased (see 1).
CONSTANT MaxRetries
VARIABLES st, rc1
vars == << st, rc1 >>

States == {"Consumed", "Pending", "Published", "persistRetry", "persisting", "publishing", "rolledBack"}
Domain == {"Consumed", "Pending", "Published"}
Overlay == {"persistRetry", "persisting", "publishing", "rolledBack"}
Final == {"Consumed"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Pending" /\ rc1 = 0

  \* T1: Pending -on:publish-> publishing
  \* T2: Published -on:markConsumed-> persisting
  \* T3: publishing -after:busTimeout-> rolledBack
  \* T4: publishing -onDone:publishToBus-> persisting
  \* T5: publishing -onError:publishToBus-> rolledBack
  \* T6: persisting -after:persistTimeout-> rolledBack
  \* T7: persisting -onDone:persistOutboxRow-> Published
  \* T8: persisting -onDone:persistOutboxRow-> Consumed
  \* T9: persisting -onDone:persistOutboxRow-> rolledBack
  \* T10: persisting -onError:persistOutboxRow-> persistRetry
  \* T11: persisting -onError:persistOutboxRow-> persistRetry
  \* T12: persisting -onError:persistOutboxRow-> rolledBack
  \* T13: rolledBack -always-> Pending
  \* T14: rolledBack -always-> Published

T1 == st = "Pending" /\ st' = "publishing" /\ rc1' = 0
T2 == st = "Published" /\ st' = "persisting" /\ rc1' = 0
T3 == st = "publishing" /\ st' = "rolledBack" /\ rc1' = rc1
T4 == st = "publishing" /\ st' = "persisting" /\ rc1' = rc1
T5 == st = "publishing" /\ st' = "rolledBack" /\ rc1' = rc1
T6 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T7 == st = "persisting" /\ st' = "Published" /\ rc1' = 0
T8 == st = "persisting" /\ st' = "Consumed" /\ rc1' = 0
T9 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T10 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T11 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T12 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T13 == st = "rolledBack" /\ st' = "Pending" /\ rc1' = 0
T14 == st = "rolledBack" /\ st' = "Published" /\ rc1' = 0
RetryExhausted_persistRetry == st = "persistRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc1 < MaxRetries /\ st' = "persisting" /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2
OverlayNext == T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
