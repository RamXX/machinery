---- MODULE DealProps ----
\* Properties of the deal pipeline around the win. Hand-written; machinery
\* never overwrites this file.
EXTENDS Deal

\* A deal is won at most once: its win is audited at most once.
WinAuditedOnce == \A i \in Instances : fx[i]["auditWin"] <= 1

\* A won deal carries its win audit entry.
WonIsAudited == \A i \in Instances : st[i] = "Won" => fx[i]["auditWin"] = 1

\* A won deal never leaves Won.
WonStaysWon == [][\A i \in Instances : st[i] = "Won" => st'[i] = "Won"]_vars

\* The win write always ends: the deal leaves closing (won, back to
\* Qualified on an error, or back to Qualified on the timeout).
ClosingEnds == \A i \in Instances : (st[i] = "closing") ~> (st[i] # "closing")

\* The bound of the strict run. Every effect but the win attempt and the win
\* audit runs at most once per deal: each deal is denied, qualified, lost,
\* reopened, failed and timed out at most once, which keeps two deals with two
\* callers small enough to check. The win itself is never bounded. A history
\* the bound ends stutters, so the bound is not reported as a deadlock.
Bounded == \A i \in Instances : \A e \in Effects \ {"stampWinAttempt", "auditWin"} : fx[i][e] <= 1
BoundedNext == \/ Next /\ Bounded'
               \/ ~ENABLED (Next /\ Bounded') /\ UNCHANGED vars
BoundedSpec == Init /\ [][BoundedNext]_vars /\ Fairness

\* Reach predicates.
SomeWon == \E i \in Instances : st[i] = "Won"
SomeReopened == \E i \in Instances : st[i] = "Lead" /\ fx[i]["auditLose"] >= 1 /\ fx[i]["auditReopen"] >= 1
====
