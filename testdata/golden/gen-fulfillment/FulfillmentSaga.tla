---- MODULE FulfillmentSaga ----
\* machinery-version: v0.3.5-dev
EXTENDS Naturals

\* Generated from FulfillmentSaga.machine.json by machinery tla. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      (none here: every guarded branch list has an unguarded fallback)
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

States == {"Compensating", "Completed", "Failed", "FailedDirty", "Paying", "Reserving", "Shipping", "compensateRetry"}
Domain == {"Completed", "Failed", "FailedDirty"}
Overlay == {"Compensating", "Paying", "Reserving", "Shipping", "compensateRetry"}
Final == {"Completed", "Failed", "FailedDirty"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Reserving" /\ rc1 = 0

  \* T1: Reserving -after:reserveTimeout-> Failed
  \* T2: Reserving -onDone:reserveInventory-> Paying
  \* T3: Reserving -onError:reserveInventory-> Failed
  \* T4: Paying -after:payTimeout-> Compensating
  \* T5: Paying -onDone:capturePayment-> Shipping
  \* T6: Paying -onError:capturePayment-> Compensating
  \* T7: Shipping -after:shipTimeout-> Compensating
  \* T8: Shipping -onDone:dispatchShipment-> Completed
  \* T9: Shipping -onError:dispatchShipment-> Compensating
  \* T10: Compensating -after:compensateTimeout-> compensateRetry
  \* T11: Compensating -onDone:compensate-> Failed
  \* T12: Compensating -onError:compensate-> compensateRetry

T1 == st = "Reserving" /\ st' = "Failed" /\ rc1' = 0
T2 == st = "Reserving" /\ st' = "Paying" /\ rc1' = rc1
T3 == st = "Reserving" /\ st' = "Failed" /\ rc1' = 0
T4 == st = "Paying" /\ st' = "Compensating" /\ rc1' = rc1
T5 == st = "Paying" /\ st' = "Shipping" /\ rc1' = rc1
T6 == st = "Paying" /\ st' = "Compensating" /\ rc1' = rc1
T7 == st = "Shipping" /\ st' = "Compensating" /\ rc1' = rc1
T8 == st = "Shipping" /\ st' = "Completed" /\ rc1' = 0
T9 == st = "Shipping" /\ st' = "Compensating" /\ rc1' = rc1
T10 == st = "Compensating" /\ st' = "compensateRetry" /\ rc1' = rc1
T11 == st = "Compensating" /\ st' = "Failed" /\ rc1' = 0
T12 == st = "Compensating" /\ st' = "compensateRetry" /\ rc1' = rc1
RetryExhausted_compensateRetry == st = "compensateRetry" /\ rc1 >= MaxRetries /\ st' = "FailedDirty" /\ rc1' = rc1
RetryAgain_compensateRetry == st = "compensateRetry" /\ rc1 < MaxRetries /\ st' = "Compensating" /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == FALSE
OverlayNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ RetryExhausted_compensateRetry \/ RetryAgain_compensateRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
