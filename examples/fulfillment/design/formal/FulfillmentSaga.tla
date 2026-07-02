---- MODULE FulfillmentSaga ----
EXTENDS Naturals

\* Generated from FulfillmentSaga.machine.json by tools/tla_gen.py. Control-flow model.
CONSTANT MaxRetries
VARIABLES st, rc
vars == << st, rc >>

States == {"Compensating", "Completed", "Failed", "FailedDirty", "Paying", "Reserving", "Shipping", "compensateRetry"}
Domain == {"Completed", "Failed", "FailedDirty"}
Overlay == {"Compensating", "Paying", "Reserving", "Shipping", "compensateRetry"}
Final == {"Completed", "Failed", "FailedDirty"}

TypeOK == st \in States /\ rc \in 0..MaxRetries
Init == st = "Reserving" /\ rc = 0

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

T1 == st = "Reserving" /\ st' = "Failed" /\ rc' = 0
T2 == st = "Reserving" /\ st' = "Paying" /\ rc' = rc
T3 == st = "Reserving" /\ st' = "Failed" /\ rc' = 0
T4 == st = "Paying" /\ st' = "Compensating" /\ rc' = rc
T5 == st = "Paying" /\ st' = "Shipping" /\ rc' = rc
T6 == st = "Paying" /\ st' = "Compensating" /\ rc' = rc
T7 == st = "Shipping" /\ st' = "Compensating" /\ rc' = rc
T8 == st = "Shipping" /\ st' = "Completed" /\ rc' = 0
T9 == st = "Shipping" /\ st' = "Compensating" /\ rc' = rc
T10 == st = "Compensating" /\ st' = "compensateRetry" /\ rc' = rc
T11 == st = "Compensating" /\ st' = "Failed" /\ rc' = 0
T12 == st = "Compensating" /\ st' = "compensateRetry" /\ rc' = rc
RetryExhausted == st = "compensateRetry" /\ rc >= MaxRetries /\ st' = "FailedDirty" /\ rc' = rc
RetryAgain == st = "compensateRetry" /\ rc < MaxRetries /\ st' = "Compensating" /\ rc' = rc + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == FALSE
OverlayNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ RetryExhausted \/ RetryAgain
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====