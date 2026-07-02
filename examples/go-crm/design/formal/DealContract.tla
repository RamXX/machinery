---- MODULE DealContract ----
\* The ABSTRACT contract the big picture assumes of a deal aggregate. It hides the
\* six stages and the persist mechanism and keeps only what a caller depends on: a
\* deal is either resting (a committed outcome) or busy (an operation in flight);
\* while busy the committed outcome does not change (atomicity); and every busy
\* period ends resting (the operation terminates). A subsystem is a correct
\* refinement of its place in the big picture exactly when its detailed model
\* implements this contract, which is what DealRefinement checks with TLC.
VARIABLES phase, kind
cvars == << phase, kind >>

Phases == {"resting", "busy"}
Kinds == {"open", "terminal"}

CTypeOK == phase \in Phases /\ kind \in Kinds
CInit == phase = "resting" /\ kind = "open"

Begin == phase = "resting" /\ phase' = "busy" /\ kind' = kind          \* start an operation
Finish == phase = "busy" /\ phase' = "resting" /\ kind' \in Kinds       \* commit or roll back, atomically
Churn == phase = "busy" /\ phase' = "busy" /\ kind' = kind              \* internal step; outcome unchanged
RestStutter == phase = "resting" /\ UNCHANGED cvars

CNext == Begin \/ Finish \/ Churn \/ RestStutter
CSpec == CInit /\ [][CNext]_cvars /\ WF_cvars(Finish)

CTermination == (phase = "busy") ~> (phase = "resting")
====
