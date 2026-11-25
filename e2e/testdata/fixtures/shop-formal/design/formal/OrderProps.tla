---- MODULE OrderProps ----
\* Properties of the order lifecycle. Hand-written.
EXTENDS Order

\* An order is charged at most once.
ChargedOnce == \A i \in Instances : fx[i]["recordCharge"] <= 1

\* An order ships only after its payment was captured.
ShippedOnlyAfterCharge == \A i \in Instances : st[i] = "Shipped" => fx[i]["recordCharge"] >= 1

\* Money is refunded only when it was taken.
RefundOnlyIfCharged == \A i \in Instances : fx[i]["refundPayment"] >= 1 => fx[i]["recordCharge"] >= 1

\* The payment call always ends: a result, an error or the timeout.
PayingSettles == \A i \in Instances : (st[i] = "paying") ~> (st[i] # "paying")

\* Reach predicates.
SomeShipped == \E i \in Instances : st[i] = "Shipped"
SomeRefunded == \E i \in Instances : st[i] = "Cancelled" /\ fx[i]["refundPayment"] = 1
====
