---- MODULE RecommendationRun ----
EXTENDS Naturals

\* Generated from RecommendationRun.machine.json by tools/tla_gen.py. Control-flow model.
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

States == {"Collecting", "Failed", "Optimizing", "Ready", "collectRetry"}
Domain == {"Failed", "Ready"}
Overlay == {"Collecting", "Optimizing", "collectRetry"}
Final == {"Failed", "Ready"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Collecting" /\ rc1 = 0

  \* T1: Collecting -after:FETCH_TIMEOUT-> collectRetry
  \* T2: Collecting -onDone:fetchPrices-> Optimizing
  \* T3: Collecting -onError:fetchPrices-> collectRetry
  \* T4: Optimizing -after:OPTIMIZE_TIMEOUT-> Failed
  \* T5: Optimizing -onDone:optimize-> Ready
  \* T6: Optimizing -onError:optimize-> Failed

T1 == st = "Collecting" /\ st' = "collectRetry" /\ rc1' = rc1
T2 == st = "Collecting" /\ st' = "Optimizing" /\ rc1' = rc1
T3 == st = "Collecting" /\ st' = "collectRetry" /\ rc1' = rc1
T4 == st = "Optimizing" /\ st' = "Failed" /\ rc1' = 0
T5 == st = "Optimizing" /\ st' = "Ready" /\ rc1' = 0
T6 == st = "Optimizing" /\ st' = "Failed" /\ rc1' = 0
RetryExhausted_collectRetry == st = "collectRetry" /\ rc1 >= MaxRetries /\ st' = "Failed" /\ rc1' = rc1
RetryAgain_collectRetry == st = "collectRetry" /\ rc1 < MaxRetries /\ st' = "Collecting" /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == FALSE
OverlayNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ RetryExhausted_collectRetry \/ RetryAgain_collectRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
