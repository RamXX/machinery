---- MODULE DealData ----
\* Data-refined model of the Deal machine. Hand-annotated on top of the generated
\* control-flow skeleton (Deal.tla): it adds the committed stage and the persist
\* context so the real domain invariants are model-checked, not just reachability.
\*
\* Proven here (see DealData.cfg):
\*   Inv_StageValid          stage is always a valid committed domain stage
\*   Inv_Atomic              during a persist attempt the committed stage is unchanged
\*                           (it flips only on the atomic commit; a failure rolls back)
\*   Inv_DomainConsistent    at rest, the machine state and the committed stage agree
\*   Inv_WonHasCloseDate     a Won deal always has a close date        (deal-won-has-closedate)
\*   StageForward            the committed stage only moves to a later stage, to Won/Lost,
\*                           or back via an explicit reopen             (deal-stage-forward)
\*   Live_OverlayResolves    every persist attempt terminates (commit or rollback)
EXTENDS Naturals

CONSTANT MaxRetries

Open == {"Lead", "Qualified", "Proposal", "Negotiation"}
Terminal == {"Won", "Lost"}
Domain == Open \cup Terminal
Overlay == {"persisting", "persistRetry", "rolledBack"}
None == "none"

Rank == [Lead |-> 0, Qualified |-> 1, Proposal |-> 2, Negotiation |-> 3, Won |-> 4, Lost |-> 4]
NextStage == [Lead |-> "Qualified", Qualified |-> "Proposal", Proposal |-> "Negotiation"]

VARIABLES st, rc, stage, pending, prior, closeSet
vars == << st, rc, stage, pending, prior, closeSet >>

TypeOK ==
  /\ st \in (Domain \cup Overlay)
  /\ rc \in 0..MaxRetries
  /\ stage \in Domain
  /\ pending \in (Domain \cup {None})
  /\ prior \in (Domain \cup {None})
  /\ closeSet \in BOOLEAN

Init ==
  /\ st = "Lead" /\ stage = "Lead"
  /\ rc = 0 /\ pending = None /\ prior = None /\ closeSet = FALSE

\* ---- domain-initiated operations (from a resting stage) ----
StartAdvance ==
  /\ st \in {"Lead", "Qualified", "Proposal"}
  /\ st' = "persisting" /\ pending' = NextStage[st] /\ prior' = st
  /\ rc' = 0 /\ stage' = stage /\ closeSet' = closeSet

StartWin ==
  /\ st \in Open
  /\ st' = "persisting" /\ pending' = "Won" /\ prior' = st
  /\ rc' = 0 /\ stage' = stage /\ closeSet' = closeSet

StartLose ==
  /\ st \in Open
  /\ st' = "persisting" /\ pending' = "Lost" /\ prior' = st
  /\ rc' = 0 /\ stage' = stage /\ closeSet' = closeSet

StartReopen ==
  /\ st \in Terminal
  /\ st' = "persisting" /\ pending' = "Negotiation" /\ prior' = st
  /\ rc' = 0 /\ stage' = stage /\ closeSet' = closeSet

\* ---- persist overlay ----
SaveDone ==
  /\ st = "persisting"
  /\ st' = pending /\ stage' = pending
  /\ closeSet' = (closeSet \/ (pending = "Won"))
  /\ pending' = None /\ prior' = None /\ rc' = 0

SaveLocked ==
  /\ st = "persisting" /\ st' = "persistRetry"
  /\ UNCHANGED << rc, stage, pending, prior, closeSet >>

SaveFail ==
  /\ st = "persisting" /\ st' = "rolledBack"
  /\ UNCHANGED << rc, stage, pending, prior, closeSet >>

RetryExhausted ==
  /\ st = "persistRetry" /\ rc >= MaxRetries /\ st' = "rolledBack"
  /\ UNCHANGED << rc, stage, pending, prior, closeSet >>

RetryAgain ==
  /\ st = "persistRetry" /\ rc < MaxRetries /\ st' = "persisting" /\ rc' = rc + 1
  /\ UNCHANGED << stage, pending, prior, closeSet >>

RolledBack ==
  /\ st = "rolledBack"
  /\ st' = prior /\ stage' = prior
  /\ pending' = None /\ prior' = None /\ rc' = 0 /\ closeSet' = closeSet

Domain_Next == StartAdvance \/ StartWin \/ StartLose \/ StartReopen
Overlay_Next == SaveDone \/ SaveLocked \/ SaveFail \/ RetryExhausted \/ RetryAgain \/ RolledBack
Next == Domain_Next \/ Overlay_Next

Spec == Init /\ [][Next]_vars /\ WF_vars(Overlay_Next)

\* ---- invariants ----
Inv_StageValid == stage \in Domain
Inv_Atomic == (st \in Overlay) => (stage = prior)
Inv_DomainConsistent == (st \in Domain) => (st = stage /\ pending = None /\ prior = None)
Inv_WonHasCloseDate == (stage = "Won") => closeSet

StageForward ==
  [][ (stage' # stage) =>
        \/ Rank[stage'] > Rank[stage]
        \/ (stage \in Terminal /\ stage' = "Negotiation") ]_stage

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
