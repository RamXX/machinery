---- MODULE ReferenceDataCommand ----
\* machinery-version: v0.6.6
EXTENDS Naturals

\* Generated from ReferenceDataCommand.machine.json by machinery tla. Control-flow model.
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
VARIABLES st
vars == << st >>

States == {"building", "failed", "idle", "refreshing", "succeeded", "upserting"}
Domain == {"failed", "idle", "succeeded"}
Overlay == {"building", "refreshing", "upserting"}
Final == {"failed", "succeeded"}

TypeOK == st \in States /\ TRUE
Init == st = "idle"

  \* T1: idle -on:refresh-> refreshing
  \* T2: idle -on:upsert-> upserting
  \* T3: idle -on:build-> building
  \* T4: refreshing -after:COMMAND_TIMEOUT-> failed
  \* T5: refreshing -onDone:selectEligibleConstituents-> succeeded
  \* T6: refreshing -onError:selectEligibleConstituents-> failed
  \* T7: upserting -after:COMMAND_TIMEOUT-> failed
  \* T8: upserting -onDone:upsertSecurityByTicker-> succeeded
  \* T9: upserting -onError:upsertSecurityByTicker-> failed
  \* T10: building -after:COMMAND_TIMEOUT-> failed
  \* T11: building -onDone:buildCandidateSet-> succeeded
  \* T12: building -onError:buildCandidateSet-> failed

T1 == st = "idle" /\ st' = "refreshing"
T2 == st = "idle" /\ st' = "upserting"
T3 == st = "idle" /\ st' = "building"
T4 == st = "refreshing" /\ st' = "failed"
T5 == st = "refreshing" /\ st' = "succeeded"
T6 == st = "refreshing" /\ st' = "failed"
T7 == st = "upserting" /\ st' = "failed"
T8 == st = "upserting" /\ st' = "succeeded"
T9 == st = "upserting" /\ st' = "failed"
T10 == st = "building" /\ st' = "failed"
T11 == st = "building" /\ st' = "succeeded"
T12 == st = "building" /\ st' = "failed"
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3
OverlayNext == T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
