---- MODULE DealRefinement ----
\* machinery-version: v0.6.5
\* GENERATED. Proof that DealData refines DealContract under a refinement mapping.
EXTENDS DealData

phaseBar == IF st \in Resting THEN "resting" ELSE "busy"
kindBar == IF st \in Fault \/ stage \in Terminal THEN "terminal" ELSE "open"

DC == INSTANCE DealContract WITH phase <- phaseBar, kind <- kindBar

RefTypeOK == DC!CTypeOK
RefSpec == DC!CSpec
RefTermination == DC!CTermination
====
