---- MODULE Payment ----
\* machinery-version: v0.3.4-dev
EXTENDS Naturals

\* Generated from Payment.machine.json by machinery tla. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      - UNVERIFIED, state gatewayResume: gatewayOp is set by setGatewayAuthorize/setGatewayCapture/setGatewayRefund before entering the three gateway-call states, the only states that reach gatewayRetry; the three gatewayFor* guards cover {authorize, capture, refund} totally
\*      - UNVERIFIED, state rolledBack: priorStatus is set on every path into the overlay from a domain state; only Pending, Authorized, and Captured reach the overlay (Failed and Refunded are final), and all three priorIs* guards are present
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
VARIABLES st, rc1, rc2
vars == << st, rc1, rc2 >>

States == {"Authorized", "Captured", "Failed", "Pending", "Refunded", "authorizing", "capturing", "gatewayResume", "gatewayRetry", "persistRetry", "persisting", "refunding", "rolledBack"}
Domain == {"Authorized", "Captured", "Failed", "Pending", "Refunded"}
Overlay == {"authorizing", "capturing", "gatewayResume", "gatewayRetry", "persistRetry", "persisting", "refunding", "rolledBack"}
Final == {"Failed", "Refunded"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries /\ rc2 \in 0..MaxRetries
Init == st = "Pending" /\ rc1 = 0 /\ rc2 = 0

  \* T1: Pending -on:authorize-> authorizing
  \* T2: Pending -on:authorize-> Pending
  \* T3: Pending -on:fail-> persisting
  \* T4: Authorized -on:capture-> capturing
  \* T5: Authorized -on:fail-> persisting
  \* T6: Captured -on:refund-> refunding
  \* T7: authorizing -after:gatewayTimeout-> gatewayRetry
  \* T8: authorizing -onDone:gatewayAuthorize-> persisting
  \* T9: authorizing -onError:gatewayAuthorize-> gatewayRetry
  \* T10: authorizing -onError:gatewayAuthorize-> persisting
  \* T11: capturing -after:gatewayTimeout-> gatewayRetry
  \* T12: capturing -onDone:gatewayCapture-> persisting
  \* T13: capturing -onError:gatewayCapture-> gatewayRetry
  \* T14: capturing -onError:gatewayCapture-> persisting
  \* T15: refunding -after:gatewayTimeout-> gatewayRetry
  \* T16: refunding -onDone:gatewayRefund-> persisting
  \* T17: refunding -onError:gatewayRefund-> gatewayRetry
  \* T18: refunding -onError:gatewayRefund-> rolledBack
  \* T19: gatewayResume -always-> authorizing
  \* T20: gatewayResume -always-> capturing
  \* T21: gatewayResume -always-> refunding
  \* T22: persisting -after:persistTimeout-> rolledBack
  \* T23: persisting -onDone:persistPayment-> Authorized
  \* T24: persisting -onDone:persistPayment-> Captured
  \* T25: persisting -onDone:persistPayment-> Refunded
  \* T26: persisting -onDone:persistPayment-> Failed
  \* T27: persisting -onDone:persistPayment-> rolledBack
  \* T28: persisting -onError:persistPayment-> persistRetry
  \* T29: persisting -onError:persistPayment-> persistRetry
  \* T30: persisting -onError:persistPayment-> rolledBack
  \* T31: rolledBack -always-> Pending
  \* T32: rolledBack -always-> Authorized
  \* T33: rolledBack -always-> Captured

T1 == st = "Pending" /\ st' = "authorizing" /\ rc1' = 0 /\ rc2' = 0
T2 == st = "Pending" /\ st' = "Pending" /\ rc1' = 0 /\ rc2' = 0
T3 == st = "Pending" /\ st' = "persisting" /\ rc1' = 0 /\ rc2' = 0
T4 == st = "Authorized" /\ st' = "capturing" /\ rc1' = 0 /\ rc2' = 0
T5 == st = "Authorized" /\ st' = "persisting" /\ rc1' = 0 /\ rc2' = 0
T6 == st = "Captured" /\ st' = "refunding" /\ rc1' = 0 /\ rc2' = 0
T7 == st = "authorizing" /\ st' = "gatewayRetry" /\ rc1' = rc1 /\ rc2' = rc2
T8 == st = "authorizing" /\ st' = "persisting" /\ rc1' = rc1 /\ rc2' = rc2
T9 == st = "authorizing" /\ st' = "gatewayRetry" /\ rc1' = rc1 /\ rc2' = rc2
T10 == st = "authorizing" /\ st' = "persisting" /\ rc1' = rc1 /\ rc2' = rc2
T11 == st = "capturing" /\ st' = "gatewayRetry" /\ rc1' = rc1 /\ rc2' = rc2
T12 == st = "capturing" /\ st' = "persisting" /\ rc1' = rc1 /\ rc2' = rc2
T13 == st = "capturing" /\ st' = "gatewayRetry" /\ rc1' = rc1 /\ rc2' = rc2
T14 == st = "capturing" /\ st' = "persisting" /\ rc1' = rc1 /\ rc2' = rc2
T15 == st = "refunding" /\ st' = "gatewayRetry" /\ rc1' = rc1 /\ rc2' = rc2
T16 == st = "refunding" /\ st' = "persisting" /\ rc1' = rc1 /\ rc2' = rc2
T17 == st = "refunding" /\ st' = "gatewayRetry" /\ rc1' = rc1 /\ rc2' = rc2
T18 == st = "refunding" /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
T19 == st = "gatewayResume" /\ st' = "authorizing" /\ rc1' = rc1 /\ rc2' = rc2
T20 == st = "gatewayResume" /\ st' = "capturing" /\ rc1' = rc1 /\ rc2' = rc2
T21 == st = "gatewayResume" /\ st' = "refunding" /\ rc1' = rc1 /\ rc2' = rc2
T22 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
T23 == st = "persisting" /\ st' = "Authorized" /\ rc1' = 0 /\ rc2' = 0
T24 == st = "persisting" /\ st' = "Captured" /\ rc1' = 0 /\ rc2' = 0
T25 == st = "persisting" /\ st' = "Refunded" /\ rc1' = 0 /\ rc2' = 0
T26 == st = "persisting" /\ st' = "Failed" /\ rc1' = 0 /\ rc2' = 0
T27 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
T28 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1 /\ rc2' = rc2
T29 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1 /\ rc2' = rc2
T30 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
T31 == st = "rolledBack" /\ st' = "Pending" /\ rc1' = 0 /\ rc2' = 0
T32 == st = "rolledBack" /\ st' = "Authorized" /\ rc1' = 0 /\ rc2' = 0
T33 == st = "rolledBack" /\ st' = "Captured" /\ rc1' = 0 /\ rc2' = 0
RetryExhausted_gatewayRetry == st = "gatewayRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
RetryAgain_gatewayRetry == st = "gatewayRetry" /\ rc1 < MaxRetries /\ st' = "gatewayResume" /\ rc1' = rc1 + 1 /\ rc2' = rc2
RetryExhausted_persistRetry == st = "persistRetry" /\ rc2 >= MaxRetries /\ st' = "rolledBack" /\ rc2' = rc2 /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc2 < MaxRetries /\ st' = "persisting" /\ rc2' = rc2 + 1 /\ rc1' = rc1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6
OverlayNext == T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ T26 \/ T27 \/ T28 \/ T29 \/ T30 \/ T31 \/ T32 \/ T33 \/ RetryExhausted_gatewayRetry \/ RetryAgain_gatewayRetry \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
