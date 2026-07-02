---- MODULE FulfillmentSaga ----
EXTENDS Naturals

\* Generated from FulfillmentSaga.machine.json by tools/tla_gen.py. Control-flow model.
CONSTANT MaxRetries
VARIABLES st, rc
vars == << st, rc >>

States == {"CompensatingPay", "CompensatingReserve", "Completed", "Failed", "Paying", "Reserving", "Shipping", "releaseRetry"}
Domain == {"Completed", "Failed"}
Overlay == {"CompensatingPay", "CompensatingReserve", "Paying", "Reserving", "Shipping", "releaseRetry"}
Final == {"Completed", "Failed"}

TypeOK == st \in States /\ rc \in 0..MaxRetries
Init == st = "Reserving" /\ rc = 0

  \* T1: Reserving -after:reserveTimeout-> Failed
  \* T2: Reserving -onDone:reserveInventory-> Paying
  \* T3: Reserving -onError:reserveInventory-> Failed
  \* T4: Paying -after:payTimeout-> CompensatingReserve
  \* T5: Paying -onDone:capturePayment-> Shipping
  \* T6: Paying -onError:capturePayment-> CompensatingReserve
  \* T7: Shipping -after:shipTimeout-> CompensatingPay
  \* T8: Shipping -onDone:dispatchShipment-> Completed
  \* T9: Shipping -onError:dispatchShipment-> CompensatingPay
  \* T10: CompensatingPay -after:refundTimeout-> CompensatingReserve
  \* T11: CompensatingPay -onDone:refundPayment-> CompensatingReserve
  \* T12: CompensatingPay -onError:refundPayment-> CompensatingReserve
  \* T13: CompensatingReserve -after:releaseTimeout-> releaseRetry
  \* T14: CompensatingReserve -onDone:releaseReservations-> Failed
  \* T15: CompensatingReserve -onError:releaseReservations-> releaseRetry

T1 == st = "Reserving" /\ st' = "Failed" /\ rc' = 0
T2 == st = "Reserving" /\ st' = "Paying" /\ rc' = rc
T3 == st = "Reserving" /\ st' = "Failed" /\ rc' = 0
T4 == st = "Paying" /\ st' = "CompensatingReserve" /\ rc' = rc
T5 == st = "Paying" /\ st' = "Shipping" /\ rc' = rc
T6 == st = "Paying" /\ st' = "CompensatingReserve" /\ rc' = rc
T7 == st = "Shipping" /\ st' = "CompensatingPay" /\ rc' = rc
T8 == st = "Shipping" /\ st' = "Completed" /\ rc' = 0
T9 == st = "Shipping" /\ st' = "CompensatingPay" /\ rc' = rc
T10 == st = "CompensatingPay" /\ st' = "CompensatingReserve" /\ rc' = rc
T11 == st = "CompensatingPay" /\ st' = "CompensatingReserve" /\ rc' = rc
T12 == st = "CompensatingPay" /\ st' = "CompensatingReserve" /\ rc' = rc
T13 == st = "CompensatingReserve" /\ st' = "releaseRetry" /\ rc' = rc
T14 == st = "CompensatingReserve" /\ st' = "Failed" /\ rc' = 0
T15 == st = "CompensatingReserve" /\ st' = "releaseRetry" /\ rc' = rc
RetryExhausted == st = "releaseRetry" /\ rc >= MaxRetries /\ st' = "Failed" /\ rc' = rc
RetryAgain == st = "releaseRetry" /\ rc < MaxRetries /\ st' = "CompensatingReserve" /\ rc' = rc + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == FALSE
OverlayNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ RetryExhausted \/ RetryAgain
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====