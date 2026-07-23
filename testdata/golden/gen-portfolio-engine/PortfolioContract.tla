---- MODULE PortfolioContract ----
\* machinery-version: v0.3.5-dev
\* GENERATED. The abstract contract the big picture assumes of the Portfolio
\* aggregate: resting or busy, atomic while busy, and every busy period terminates.
VARIABLES phase, kind
cvars == << phase, kind >>

Phases == {"resting", "busy"}
Kinds == {"open", "terminal"}

CTypeOK == phase \in Phases /\ kind \in Kinds
CInit == phase = "resting" /\ kind = "open"

Begin == phase = "resting" /\ phase' = "busy" /\ kind' = kind
Finish == phase = "busy" /\ phase' = "resting" /\ kind' \in Kinds
Churn == phase = "busy" /\ phase' = "busy" /\ kind' = kind
RestStutter == phase = "resting" /\ UNCHANGED cvars

CNext == Begin \/ Finish \/ Churn \/ RestStutter
CSpec == CInit /\ [][CNext]_cvars /\ WF_cvars(Finish)
CTermination == (phase = "busy") ~> (phase = "resting")
====
