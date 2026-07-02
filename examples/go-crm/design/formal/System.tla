---- MODULE System ----
\* Assume-guarantee composition, the rule that makes the recursion scale. A caller
\* (the command envelope) drives a deal aggregate that is known ONLY through its
\* contract (DealContract): the caller ASSUMES the aggregate is atomic while busy and
\* always terminates. Under that assumption the caller GUARANTEES that the command
\* completes and leaves the deal resting. Because DealData refines DealContract
\* (checked in DealRefinement), substituting the real Deal aggregate preserves these
\* guarantees without re-checking the whole system: the parts are verified against
\* small contracts, never against the flattened composition.
EXTENDS Naturals

Phases == {"resting", "busy"}
Kinds == {"open", "terminal"}
CmdStates == {"idle", "waiting", "done"}

VARIABLES cmd, phase, kind
vars == << cmd, phase, kind >>

TypeOK == cmd \in CmdStates /\ phase \in Phases /\ kind \in Kinds
Init == cmd = "idle" /\ phase = "resting" /\ kind = "open"

\* the caller starts the aggregate (a Begin step of the contract)
Invoke ==
  /\ cmd = "idle" /\ phase = "resting"
  /\ cmd' = "waiting" /\ phase' = "busy" /\ kind' = kind

\* the aggregate as its contract allows: churn atomically, or finish
AggChurn == phase = "busy" /\ phase' = "busy" /\ kind' = kind /\ cmd' = cmd
AggFinish == phase = "busy" /\ phase' = "resting" /\ kind' \in Kinds /\ cmd' = cmd

\* the caller observes completion
Complete == cmd = "waiting" /\ phase = "resting" /\ cmd' = "done" /\ UNCHANGED << phase, kind >>
DoneStutter == cmd = "done" /\ UNCHANGED vars

Next == Invoke \/ AggChurn \/ AggFinish \/ Complete \/ DoneStutter

\* the aggregate's contract guarantees it finishes (assumed); the caller acts on it
Spec == Init /\ [][Next]_vars /\ WF_vars(AggFinish) /\ WF_vars(Complete)

Safe_DoneImpliesResting == (cmd = "done") => (phase = "resting")
Live_CommandCompletes == (cmd = "waiting") ~> (cmd = "done")
====
