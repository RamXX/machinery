---- MODULE Deal ----
EXTENDS Naturals

\* Generated from Deal.machine.json by tools/tla_gen.py. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: sound for safety; for liveness the
\*      guard lists must be exhaustive. machine_lint enforces an unguarded
\*      fallback or an _exhaustive note on every fully guarded always-list.
\*      - rolledBack: prior is set by every setPending action to the stage the operation departed from; its codomain is exactly the six DealStage values, so one guarded branch per stage is total
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

States == {"Lost", "Negotiation", "Proposal", "Prospecting", "Qualification", "Won", "persistRetry", "persisting", "rolledBack"}
Domain == {"Lost", "Negotiation", "Proposal", "Prospecting", "Qualification", "Won"}
Overlay == {"persistRetry", "persisting", "rolledBack"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Prospecting" /\ rc1 = 0

  \* T1: Prospecting -on:advance-> persisting
  \* T2: Prospecting -on:win-> persisting
  \* T3: Prospecting -on:lose-> persisting
  \* T4: Qualification -on:advance-> persisting
  \* T5: Qualification -on:win-> persisting
  \* T6: Qualification -on:lose-> persisting
  \* T7: Proposal -on:advance-> persisting
  \* T8: Proposal -on:win-> persisting
  \* T9: Proposal -on:lose-> persisting
  \* T10: Negotiation -on:win-> persisting
  \* T11: Negotiation -on:lose-> persisting
  \* T12: Won -on:reopen-> persisting
  \* T13: Lost -on:reopen-> persisting
  \* T14: persisting -after:PERSIST_TIMEOUT-> persistRetry
  \* T15: persisting -onDone:persist-> Qualification
  \* T16: persisting -onDone:persist-> Proposal
  \* T17: persisting -onDone:persist-> Negotiation
  \* T18: persisting -onDone:persist-> Won
  \* T19: persisting -onDone:persist-> Lost
  \* T20: persisting -onError:persist-> persistRetry
  \* T21: persisting -onError:persist-> rolledBack
  \* T22: rolledBack -always-> Prospecting
  \* T23: rolledBack -always-> Qualification
  \* T24: rolledBack -always-> Proposal
  \* T25: rolledBack -always-> Negotiation
  \* T26: rolledBack -always-> Won
  \* T27: rolledBack -always-> Lost

T1 == st = "Prospecting" /\ st' = "persisting" /\ rc1' = 0
T2 == st = "Prospecting" /\ st' = "persisting" /\ rc1' = 0
T3 == st = "Prospecting" /\ st' = "persisting" /\ rc1' = 0
T4 == st = "Qualification" /\ st' = "persisting" /\ rc1' = 0
T5 == st = "Qualification" /\ st' = "persisting" /\ rc1' = 0
T6 == st = "Qualification" /\ st' = "persisting" /\ rc1' = 0
T7 == st = "Proposal" /\ st' = "persisting" /\ rc1' = 0
T8 == st = "Proposal" /\ st' = "persisting" /\ rc1' = 0
T9 == st = "Proposal" /\ st' = "persisting" /\ rc1' = 0
T10 == st = "Negotiation" /\ st' = "persisting" /\ rc1' = 0
T11 == st = "Negotiation" /\ st' = "persisting" /\ rc1' = 0
T12 == st = "Won" /\ st' = "persisting" /\ rc1' = 0
T13 == st = "Lost" /\ st' = "persisting" /\ rc1' = 0
T14 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T15 == st = "persisting" /\ st' = "Qualification" /\ rc1' = 0
T16 == st = "persisting" /\ st' = "Proposal" /\ rc1' = 0
T17 == st = "persisting" /\ st' = "Negotiation" /\ rc1' = 0
T18 == st = "persisting" /\ st' = "Won" /\ rc1' = 0
T19 == st = "persisting" /\ st' = "Lost" /\ rc1' = 0
T20 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T21 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T22 == st = "rolledBack" /\ st' = "Prospecting" /\ rc1' = 0
T23 == st = "rolledBack" /\ st' = "Qualification" /\ rc1' = 0
T24 == st = "rolledBack" /\ st' = "Proposal" /\ rc1' = 0
T25 == st = "rolledBack" /\ st' = "Negotiation" /\ rc1' = 0
T26 == st = "rolledBack" /\ st' = "Won" /\ rc1' = 0
T27 == st = "rolledBack" /\ st' = "Lost" /\ rc1' = 0
RetryExhausted_persistRetry == st = "persistRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc1 < MaxRetries /\ st' = "persisting" /\ rc1' = rc1 + 1

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13
OverlayNext == T14 \/ T15 \/ T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ T26 \/ T27 \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
