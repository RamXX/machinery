---- MODULE Deal ----
EXTENDS Naturals

\* Generated from Deal.machine.json by tools/tla_gen.py. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: sound for safety; for liveness the
\*      guard lists must be exhaustive. machine_lint enforces an unguarded
\*      fallback or an _exhaustive note on every fully guarded always-list.
\*      - rolledBack: priorStage is set by every setPending* action to the current domain state, which is one of the six DealStage values; the six priorIs* guards cover DealStage totally
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

States == {"Lead", "Lost", "Negotiation", "Proposal", "Qualified", "Won", "persistRetry", "persisting", "rolledBack"}
Domain == {"Lead", "Lost", "Negotiation", "Proposal", "Qualified", "Won"}
Overlay == {"persistRetry", "persisting", "rolledBack"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Lead" /\ rc1 = 0

  \* T1: Lead -on:advanceStage-> persisting
  \* T2: Lead -on:advanceStage-> Lead
  \* T3: Lead -on:win-> persisting
  \* T4: Lead -on:win-> Lead
  \* T5: Lead -on:lose-> persisting
  \* T6: Lead -on:lose-> Lead
  \* T7: Lead -on:reopen-> Lead
  \* T8: Qualified -on:advanceStage-> persisting
  \* T9: Qualified -on:advanceStage-> Qualified
  \* T10: Qualified -on:win-> persisting
  \* T11: Qualified -on:win-> Qualified
  \* T12: Qualified -on:lose-> persisting
  \* T13: Qualified -on:lose-> Qualified
  \* T14: Qualified -on:reopen-> Qualified
  \* T15: Proposal -on:advanceStage-> persisting
  \* T16: Proposal -on:advanceStage-> Proposal
  \* T17: Proposal -on:win-> persisting
  \* T18: Proposal -on:win-> Proposal
  \* T19: Proposal -on:lose-> persisting
  \* T20: Proposal -on:lose-> Proposal
  \* T21: Proposal -on:reopen-> Proposal
  \* T22: Negotiation -on:advanceStage-> Negotiation
  \* T23: Negotiation -on:win-> persisting
  \* T24: Negotiation -on:win-> Negotiation
  \* T25: Negotiation -on:lose-> persisting
  \* T26: Negotiation -on:lose-> Negotiation
  \* T27: Negotiation -on:reopen-> Negotiation
  \* T28: Won -on:reopen-> persisting
  \* T29: Won -on:reopen-> Won
  \* T30: Won -on:advanceStage-> Won
  \* T31: Won -on:win-> Won
  \* T32: Won -on:lose-> Won
  \* T33: Lost -on:reopen-> persisting
  \* T34: Lost -on:reopen-> Lost
  \* T35: Lost -on:advanceStage-> Lost
  \* T36: Lost -on:win-> Lost
  \* T37: Lost -on:lose-> Lost
  \* T38: persisting -after:persistTimeout-> rolledBack
  \* T39: persisting -onDone:saveDeal-> Qualified
  \* T40: persisting -onDone:saveDeal-> Proposal
  \* T41: persisting -onDone:saveDeal-> Negotiation
  \* T42: persisting -onDone:saveDeal-> Won
  \* T43: persisting -onDone:saveDeal-> Lost
  \* T44: persisting -onDone:saveDeal-> rolledBack
  \* T45: persisting -onError:saveDeal-> persistRetry
  \* T46: persisting -onError:saveDeal-> rolledBack
  \* T47: persisting -onError:saveDeal-> rolledBack
  \* T48: persisting -onError:saveDeal-> rolledBack
  \* T49: persisting -onError:saveDeal-> rolledBack
  \* T50: rolledBack -always-> Lead
  \* T51: rolledBack -always-> Qualified
  \* T52: rolledBack -always-> Proposal
  \* T53: rolledBack -always-> Negotiation
  \* T54: rolledBack -always-> Won
  \* T55: rolledBack -always-> Lost

T1 == st = "Lead" /\ st' = "persisting" /\ rc1' = 0
T2 == st = "Lead" /\ st' = "Lead" /\ rc1' = 0
T3 == st = "Lead" /\ st' = "persisting" /\ rc1' = 0
T4 == st = "Lead" /\ st' = "Lead" /\ rc1' = 0
T5 == st = "Lead" /\ st' = "persisting" /\ rc1' = 0
T6 == st = "Lead" /\ st' = "Lead" /\ rc1' = 0
T7 == st = "Lead" /\ st' = "Lead" /\ rc1' = 0
T8 == st = "Qualified" /\ st' = "persisting" /\ rc1' = 0
T9 == st = "Qualified" /\ st' = "Qualified" /\ rc1' = 0
T10 == st = "Qualified" /\ st' = "persisting" /\ rc1' = 0
T11 == st = "Qualified" /\ st' = "Qualified" /\ rc1' = 0
T12 == st = "Qualified" /\ st' = "persisting" /\ rc1' = 0
T13 == st = "Qualified" /\ st' = "Qualified" /\ rc1' = 0
T14 == st = "Qualified" /\ st' = "Qualified" /\ rc1' = 0
T15 == st = "Proposal" /\ st' = "persisting" /\ rc1' = 0
T16 == st = "Proposal" /\ st' = "Proposal" /\ rc1' = 0
T17 == st = "Proposal" /\ st' = "persisting" /\ rc1' = 0
T18 == st = "Proposal" /\ st' = "Proposal" /\ rc1' = 0
T19 == st = "Proposal" /\ st' = "persisting" /\ rc1' = 0
T20 == st = "Proposal" /\ st' = "Proposal" /\ rc1' = 0
T21 == st = "Proposal" /\ st' = "Proposal" /\ rc1' = 0
T22 == st = "Negotiation" /\ st' = "Negotiation" /\ rc1' = 0
T23 == st = "Negotiation" /\ st' = "persisting" /\ rc1' = 0
T24 == st = "Negotiation" /\ st' = "Negotiation" /\ rc1' = 0
T25 == st = "Negotiation" /\ st' = "persisting" /\ rc1' = 0
T26 == st = "Negotiation" /\ st' = "Negotiation" /\ rc1' = 0
T27 == st = "Negotiation" /\ st' = "Negotiation" /\ rc1' = 0
T28 == st = "Won" /\ st' = "persisting" /\ rc1' = 0
T29 == st = "Won" /\ st' = "Won" /\ rc1' = 0
T30 == st = "Won" /\ st' = "Won" /\ rc1' = 0
T31 == st = "Won" /\ st' = "Won" /\ rc1' = 0
T32 == st = "Won" /\ st' = "Won" /\ rc1' = 0
T33 == st = "Lost" /\ st' = "persisting" /\ rc1' = 0
T34 == st = "Lost" /\ st' = "Lost" /\ rc1' = 0
T35 == st = "Lost" /\ st' = "Lost" /\ rc1' = 0
T36 == st = "Lost" /\ st' = "Lost" /\ rc1' = 0
T37 == st = "Lost" /\ st' = "Lost" /\ rc1' = 0
T38 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T39 == st = "persisting" /\ st' = "Qualified" /\ rc1' = 0
T40 == st = "persisting" /\ st' = "Proposal" /\ rc1' = 0
T41 == st = "persisting" /\ st' = "Negotiation" /\ rc1' = 0
T42 == st = "persisting" /\ st' = "Won" /\ rc1' = 0
T43 == st = "persisting" /\ st' = "Lost" /\ rc1' = 0
T44 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T45 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T46 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T47 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T48 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T49 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T50 == st = "rolledBack" /\ st' = "Lead" /\ rc1' = 0
T51 == st = "rolledBack" /\ st' = "Qualified" /\ rc1' = 0
T52 == st = "rolledBack" /\ st' = "Proposal" /\ rc1' = 0
T53 == st = "rolledBack" /\ st' = "Negotiation" /\ rc1' = 0
T54 == st = "rolledBack" /\ st' = "Won" /\ rc1' = 0
T55 == st = "rolledBack" /\ st' = "Lost" /\ rc1' = 0
RetryExhausted_persistRetry == st = "persistRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc1 < MaxRetries /\ st' = "persisting" /\ rc1' = rc1 + 1

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ T26 \/ T27 \/ T28 \/ T29 \/ T30 \/ T31 \/ T32 \/ T33 \/ T34 \/ T35 \/ T36 \/ T37
OverlayNext == T38 \/ T39 \/ T40 \/ T41 \/ T42 \/ T43 \/ T44 \/ T45 \/ T46 \/ T47 \/ T48 \/ T49 \/ T50 \/ T51 \/ T52 \/ T53 \/ T54 \/ T55 \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
