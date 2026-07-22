---- MODULE DealRefinement ----
\* machinery-version: v0.3.4-dev
\* GENERATED. Proof that DealData refines DealContract under a refinement mapping.
EXTENDS DealData

phaseBar == IF st \in Domain THEN "resting" ELSE "busy"
kindBar == IF stage \in Terminal THEN "terminal" ELSE "open"

DC == INSTANCE DealContract WITH phase <- phaseBar, kind <- kindBar

RefTypeOK == DC!CTypeOK
RefSpec == DC!CSpec
RefTermination == DC!CTermination
====
