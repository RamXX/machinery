---- MODULE Portfolio ----
\* machinery-version: v0.6.3
EXTENDS Naturals

\* Generated from Portfolio.machine.json by machinery tla. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      (none here: every guarded always-list has an unguarded fallback)
\*      Handler refusal is permitted in this machine. A refused trigger leaves
\*      the state unchanged, so this rung checks safety only and makes no
\*      fairness or overlay-resolution liveness claim:
\*      - state Proposed, handler accept: canDecide false is an authorization refusal at the command boundary: pf.app returns AuthzError, the portfolio is untouched, and no transition is recorded (portfolio-accept-role). The absorption is deliberate, not a stall: the caller learns why.
\*      - state Proposed, handler reject: canDecide false is an authorization refusal at the command boundary: pf.app returns AuthzError, the portfolio is untouched, and no transition is recorded (portfolio-accept-role).
\*      - state UnderReview, handler accept: canDecide false is an authorization refusal at the command boundary: pf.app returns AuthzError, the portfolio is untouched, and no transition is recorded (portfolio-accept-role). The absorption is deliberate, not a stall: the caller learns why.
\*      - state UnderReview, handler reject: canDecide false is an authorization refusal at the command boundary: pf.app returns AuthzError, the portfolio is untouched, and no transition is recorded (portfolio-accept-role).
\*      - state Accepted, handler reopen: canReopen false is an authorization refusal at the command boundary: pf.app returns AuthzError and the decided portfolio is untouched (portfolio-reopen-role).
\*      - state Rejected, handler reopen: canReopen false is an authorization refusal at the command boundary: pf.app returns AuthzError and the decided portfolio is untouched (portfolio-reopen-role).
\*      - state committing, handler persistDecision.onDone: Unreachable by construction: context.pending is set by one of setPendingAdvance/Accept/Reject/Reopen before the invoke starts, and its codomain is exactly the three target stages the three guards cover. A fourth value would strand a committed write, so the guard set is total by the same argument the reverted always-list makes.
\*   2. Every invoke resolves exactly once (onDone or onError; no lost or
\*      duplicated completion) and every after timer eventually fires.
\*   3. Single machine instance; no interleaving with other instances or
\*      machines, no message loss/duplication/reordering between machines.
\*   4. Context data, event payloads, action effects, and real time (the
\*      _delays values) are not modeled at this rung; the data-refined rung
\*      (refine_gen) and the implementation tests carry those.
\*   5. Retry counters (rc*) reset to 0 on every transition that leaves from
\*      or lands on a domain state; a counter surviving a domain hop is not
\*      representable at this rung.
\*   6. Retry-shaped states (fully guarded always + after) are modeled as the
\*      concrete bounded loop: the guarded always list is replaced by the
\*      exhaustion test rc >= MaxRetries and the after timer by the retry step
\*      rc < MaxRetries; the guards themselves are erased (see 1).
CONSTANT MaxRetries
VARIABLES st, rc1
vars == << st, rc1 >>

States == {"Accepted", "Proposed", "Rejected", "UnderReview", "commitRetry", "committing", "reverted", "routingFault"}
Domain == {"Accepted", "Proposed", "Rejected", "UnderReview", "routingFault"}
Overlay == {"commitRetry", "committing", "reverted"}
Final == {"routingFault"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Proposed" /\ rc1 = 0

  \* T1: Proposed -on:advance-> committing
  \* T2: Proposed -on:accept-> committing
  \* T3: Proposed -on:reject-> committing
  \* T4: UnderReview -on:accept-> committing
  \* T5: UnderReview -on:reject-> committing
  \* T6: Accepted -on:reopen-> committing
  \* T7: Rejected -on:reopen-> committing
  \* T8: committing -after:COMMIT_TIMEOUT-> commitRetry
  \* T9: committing -onDone:persistDecision-> UnderReview
  \* T10: committing -onDone:persistDecision-> Accepted
  \* T11: committing -onDone:persistDecision-> Rejected
  \* T12: committing -onError:persistDecision-> commitRetry
  \* T13: committing -onError:persistDecision-> reverted
  \* T14: reverted -always-> Proposed
  \* T15: reverted -always-> UnderReview
  \* T16: reverted -always-> Accepted
  \* T17: reverted -always-> Rejected
  \* T18: reverted -always-> routingFault

T1 == st = "Proposed" /\ st' = "committing" /\ rc1' = 0
T2 == st = "Proposed" /\ st' = "committing" /\ rc1' = 0
T3 == st = "Proposed" /\ st' = "committing" /\ rc1' = 0
T4 == st = "UnderReview" /\ st' = "committing" /\ rc1' = 0
T5 == st = "UnderReview" /\ st' = "committing" /\ rc1' = 0
T6 == st = "Accepted" /\ st' = "committing" /\ rc1' = 0
T7 == st = "Rejected" /\ st' = "committing" /\ rc1' = 0
T8 == st = "committing" /\ st' = "commitRetry" /\ rc1' = rc1
T9 == st = "committing" /\ st' = "UnderReview" /\ rc1' = 0
T10 == st = "committing" /\ st' = "Accepted" /\ rc1' = 0
T11 == st = "committing" /\ st' = "Rejected" /\ rc1' = 0
T12 == st = "committing" /\ st' = "commitRetry" /\ rc1' = rc1
T13 == st = "committing" /\ st' = "reverted" /\ rc1' = rc1
T14 == st = "reverted" /\ st' = "Proposed" /\ rc1' = 0
T15 == st = "reverted" /\ st' = "UnderReview" /\ rc1' = 0
T16 == st = "reverted" /\ st' = "Accepted" /\ rc1' = 0
T17 == st = "reverted" /\ st' = "Rejected" /\ rc1' = 0
T18 == st = "reverted" /\ st' = "routingFault" /\ rc1' = 0
RetryExhausted_commitRetry == st = "commitRetry" /\ rc1 >= MaxRetries /\ st' = "reverted" /\ rc1' = rc1
RetryAgain_commitRetry == st = "commitRetry" /\ rc1 < MaxRetries /\ st' = "committing" /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7
OverlayNext == T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ T18 \/ RetryExhausted_commitRetry \/ RetryAgain_commitRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars

\* Live_OverlayResolves intentionally omitted: _refusal permits persistent stuttering.
====
