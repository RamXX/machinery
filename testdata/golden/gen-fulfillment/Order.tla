---- MODULE Order ----
EXTENDS Naturals

\* Generated from Order.machine.json by tools/tla_gen.py. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      - UNVERIFIED, state rolledBack: priorStatus is set by every setPending* action to the current domain state; only Pending, Confirmed, Reserved, Paid, and Shipped have transitions into the persist overlay (final states persist nothing), and all five priorIs* guards are present
\*   2. Every invoke resolves exactly once (onDone or onError; no lost or
\*      duplicated completion) and every after timer eventually fires.
\*   3. Single machine instance; no interleaving with other instances or
\*      machines, no message loss/duplication/reordering between machines.
\*   4. Context data, event payloads, action effects, and real time (the
\*      _delays values) are not modeled at this rung; the data-refined rung
\*      (refine_gen) and the implementation tests carry those.
CONSTANT MaxRetries
VARIABLES st, rc1
vars == << st, rc1 >>

States == {"Cancelled", "Confirmed", "Delivered", "Failed", "Paid", "Pending", "Reserved", "Shipped", "persistRetry", "persisting", "rolledBack"}
Domain == {"Cancelled", "Confirmed", "Delivered", "Failed", "Paid", "Pending", "Reserved", "Shipped"}
Overlay == {"persistRetry", "persisting", "rolledBack"}
Final == {"Cancelled", "Delivered", "Failed"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Pending" /\ rc1 = 0

  \* T1: Pending -on:confirm-> persisting
  \* T2: Pending -on:confirm-> Pending
  \* T3: Pending -on:cancel-> persisting
  \* T4: Pending -on:cancel-> Pending
  \* T5: Confirmed -on:markReserved-> persisting
  \* T6: Confirmed -on:cancel-> persisting
  \* T7: Confirmed -on:cancel-> Confirmed
  \* T8: Confirmed -on:fail-> persisting
  \* T9: Reserved -on:markPaid-> persisting
  \* T10: Reserved -on:fail-> persisting
  \* T11: Paid -on:markShipped-> persisting
  \* T12: Paid -on:fail-> persisting
  \* T13: Shipped -on:markDelivered-> persisting
  \* T14: Shipped -on:fail-> persisting
  \* T15: persisting -after:persistTimeout-> rolledBack
  \* T16: persisting -onDone:persistOrder-> Confirmed
  \* T17: persisting -onDone:persistOrder-> Reserved
  \* T18: persisting -onDone:persistOrder-> Paid
  \* T19: persisting -onDone:persistOrder-> Shipped
  \* T20: persisting -onDone:persistOrder-> Delivered
  \* T21: persisting -onDone:persistOrder-> Cancelled
  \* T22: persisting -onDone:persistOrder-> Failed
  \* T23: persisting -onDone:persistOrder-> rolledBack
  \* T24: persisting -onError:persistOrder-> persistRetry
  \* T25: persisting -onError:persistOrder-> persistRetry
  \* T26: persisting -onError:persistOrder-> rolledBack
  \* T27: rolledBack -always-> Pending
  \* T28: rolledBack -always-> Confirmed
  \* T29: rolledBack -always-> Reserved
  \* T30: rolledBack -always-> Paid
  \* T31: rolledBack -always-> Shipped

T1 == st = "Pending" /\ st' = "persisting" /\ rc1' = 0
T2 == st = "Pending" /\ st' = "Pending" /\ rc1' = 0
T3 == st = "Pending" /\ st' = "persisting" /\ rc1' = 0
T4 == st = "Pending" /\ st' = "Pending" /\ rc1' = 0
T5 == st = "Confirmed" /\ st' = "persisting" /\ rc1' = 0
T6 == st = "Confirmed" /\ st' = "persisting" /\ rc1' = 0
T7 == st = "Confirmed" /\ st' = "Confirmed" /\ rc1' = 0
T8 == st = "Confirmed" /\ st' = "persisting" /\ rc1' = 0
T9 == st = "Reserved" /\ st' = "persisting" /\ rc1' = 0
T10 == st = "Reserved" /\ st' = "persisting" /\ rc1' = 0
T11 == st = "Paid" /\ st' = "persisting" /\ rc1' = 0
T12 == st = "Paid" /\ st' = "persisting" /\ rc1' = 0
T13 == st = "Shipped" /\ st' = "persisting" /\ rc1' = 0
T14 == st = "Shipped" /\ st' = "persisting" /\ rc1' = 0
T15 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T16 == st = "persisting" /\ st' = "Confirmed" /\ rc1' = 0
T17 == st = "persisting" /\ st' = "Reserved" /\ rc1' = 0
T18 == st = "persisting" /\ st' = "Paid" /\ rc1' = 0
T19 == st = "persisting" /\ st' = "Shipped" /\ rc1' = 0
T20 == st = "persisting" /\ st' = "Delivered" /\ rc1' = 0
T21 == st = "persisting" /\ st' = "Cancelled" /\ rc1' = 0
T22 == st = "persisting" /\ st' = "Failed" /\ rc1' = 0
T23 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T24 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T25 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1
T26 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1
T27 == st = "rolledBack" /\ st' = "Pending" /\ rc1' = 0
T28 == st = "rolledBack" /\ st' = "Confirmed" /\ rc1' = 0
T29 == st = "rolledBack" /\ st' = "Reserved" /\ rc1' = 0
T30 == st = "rolledBack" /\ st' = "Paid" /\ rc1' = 0
T31 == st = "rolledBack" /\ st' = "Shipped" /\ rc1' = 0
RetryExhausted_persistRetry == st = "persistRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc1 < MaxRetries /\ st' = "persisting" /\ rc1' = rc1 + 1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14
OverlayNext == T15 \/ T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ T26 \/ T27 \/ T28 \/ T29 \/ T30 \/ T31 \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
