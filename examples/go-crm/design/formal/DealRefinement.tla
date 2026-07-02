---- MODULE DealRefinement ----
\* Proves the concrete data-refined Deal model (DealData) REFINES the abstract
\* DealContract, under a refinement mapping. This is the composition glue for the
\* recursion: each subsystem is designed fresh at its own level, and is a correct
\* part of the whole exactly when it refines the contract the big picture assumed.
\* TLC checks that every DealData behavior is a DealContract behavior under the map.
EXTENDS DealData

\* refinement mapping: concrete state -> abstract contract state
phaseBar == IF st \in Domain THEN "resting" ELSE "busy"
kindBar == IF stage \in Terminal THEN "terminal" ELSE "open"

DC == INSTANCE DealContract WITH phase <- phaseBar, kind <- kindBar

\* named aliases so the TLC config can reference the instance's operators
RefTypeOK == DC!CTypeOK
RefSpec == DC!CSpec
RefTermination == DC!CTermination
====
