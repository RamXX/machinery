---- MODULE PortfolioRefinement ----
\* machinery-version: v0.6.9
\* GENERATED. Proof that PortfolioData refines PortfolioContract under a refinement mapping.
EXTENDS PortfolioData

phaseBar == IF st \in Resting THEN "resting" ELSE "busy"
kindBar == IF st \in Fault \/ stage \in Terminal THEN "terminal" ELSE "open"

DC == INSTANCE PortfolioContract WITH phase <- phaseBar, kind <- kindBar

RefTypeOK == DC!CTypeOK
RefSpec == DC!CSpec
RefTermination == DC!CTermination
====
