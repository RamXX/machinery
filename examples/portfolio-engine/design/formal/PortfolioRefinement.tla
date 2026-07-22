---- MODULE PortfolioRefinement ----
\* machinery-version: v0.3.4-dev
\* GENERATED. Proof that PortfolioData refines PortfolioContract under a refinement mapping.
EXTENDS PortfolioData

phaseBar == IF st \in Domain THEN "resting" ELSE "busy"
kindBar == IF stage \in Terminal THEN "terminal" ELSE "open"

DC == INSTANCE PortfolioContract WITH phase <- phaseBar, kind <- kindBar

RefTypeOK == DC!CTypeOK
RefSpec == DC!CSpec
RefTermination == DC!CTermination
====
