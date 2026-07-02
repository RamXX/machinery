---- MODULE System ----
\* Assume-guarantee composition, the rule that makes the recursion scale. A caller
\* (the command envelope) drives a deal aggregate that is known ONLY through its
\* contract: the aggregate's steps below are INSTANCED from DealContract, not
\* hand-mirrored, so the assumption the caller makes is the same module that
\* DealRefinement proves DealData provides. If refine_gen regenerates a different
\* contract, this module checks against the new one or fails to parse; the
\* assumption and the proven guarantee cannot drift apart silently.
\*
\* Checked here: under the contract's fairness (busy work finishes), every command
\* completes and observes a resting aggregate; and the composed system's aggregate
\* steps themselves satisfy the contract spec (AggregateContractHolds), which is
\* what justifies substituting the real, refinement-checked aggregate.
EXTENDS Naturals

CmdStates == {"idle", "waiting", "done"}

VARIABLES cmd, phase, kind
vars == << cmd, phase, kind >>

DC == INSTANCE DealContract WITH phase <- phase, kind <- kind

TypeOK == cmd \in CmdStates /\ DC!CTypeOK
Init == cmd = "idle" /\ DC!CInit

\* the caller starts the aggregate: exactly a Begin step of the contract
Invoke == cmd = "idle" /\ phase = "resting" /\ cmd' = "waiting" /\ DC!Begin

\* the aggregate as its contract allows: churn atomically, or finish
AggChurn == DC!Churn /\ cmd' = cmd
AggFinish == DC!Finish /\ cmd' = cmd

\* the caller observes completion
Complete == cmd = "waiting" /\ phase = "resting" /\ cmd' = "done" /\ UNCHANGED << phase, kind >>
DoneStutter == cmd = "done" /\ UNCHANGED vars

Next == Invoke \/ AggChurn \/ AggFinish \/ Complete \/ DoneStutter

\* the aggregate's contract guarantees it finishes (assumed); the caller acts on it
Spec == Init /\ [][Next]_vars /\ WF_vars(AggFinish) /\ WF_vars(Complete)

Safe_DoneImpliesResting == (cmd = "done") => (phase = "resting")
Live_CommandCompletes == (cmd = "waiting") ~> (cmd = "done")

\* the composition's aggregate behavior satisfies the contract spec, fairness
\* included: the assumption made of the aggregate is discharged, not just named
AggregateContractHolds == DC!CSpec
====
