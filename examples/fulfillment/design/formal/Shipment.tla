---- MODULE Shipment ----
\* machinery-version: v0.3.5-dev
EXTENDS Naturals

\* Generated from Shipment.machine.json by machinery tla. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      - UNVERIFIED, state rolledBack: priorStatus is set on every path into the overlay from a domain state; only Pending, Dispatched, and InTransit reach the overlay (Delivered and Lost are final), and all three priorIs* guards are present
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

States == {"Delivered", "Dispatched", "InTransit", "Lost", "Pending", "carrierRetry", "dispatching", "persistRetry", "persisting", "rolledBack"}
Domain == {"Delivered", "Dispatched", "InTransit", "Lost", "Pending"}
Overlay == {"carrierRetry", "dispatching", "persistRetry", "persisting", "rolledBack"}
Final == {"Delivered", "Lost"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries /\ rc2 \in 0..MaxRetries
Init == st = "Pending" /\ rc1 = 0 /\ rc2 = 0

  \* T1: Pending -on:dispatch-> dispatching
  \* T2: Dispatched -on:markInTransit-> persisting
  \* T3: Dispatched -on:deliver-> persisting
  \* T4: Dispatched -on:markLost-> persisting
  \* T5: InTransit -on:deliver-> persisting
  \* T6: InTransit -on:markLost-> persisting
  \* T7: dispatching -after:carrierTimeout-> carrierRetry
  \* T8: dispatching -onDone:carrierDispatch-> persisting
  \* T9: dispatching -onError:carrierDispatch-> carrierRetry
  \* T10: dispatching -onError:carrierDispatch-> rolledBack
  \* T11: persisting -after:persistTimeout-> rolledBack
  \* T12: persisting -onDone:persistShipment-> Dispatched
  \* T13: persisting -onDone:persistShipment-> InTransit
  \* T14: persisting -onDone:persistShipment-> Delivered
  \* T15: persisting -onDone:persistShipment-> Lost
  \* T16: persisting -onDone:persistShipment-> rolledBack
  \* T17: persisting -onError:persistShipment-> persistRetry
  \* T18: persisting -onError:persistShipment-> persistRetry
  \* T19: persisting -onError:persistShipment-> rolledBack
  \* T20: rolledBack -always-> Pending
  \* T21: rolledBack -always-> Dispatched
  \* T22: rolledBack -always-> InTransit

T1 == st = "Pending" /\ st' = "dispatching" /\ rc1' = 0 /\ rc2' = 0
T2 == st = "Dispatched" /\ st' = "persisting" /\ rc1' = 0 /\ rc2' = 0
T3 == st = "Dispatched" /\ st' = "persisting" /\ rc1' = 0 /\ rc2' = 0
T4 == st = "Dispatched" /\ st' = "persisting" /\ rc1' = 0 /\ rc2' = 0
T5 == st = "InTransit" /\ st' = "persisting" /\ rc1' = 0 /\ rc2' = 0
T6 == st = "InTransit" /\ st' = "persisting" /\ rc1' = 0 /\ rc2' = 0
T7 == st = "dispatching" /\ st' = "carrierRetry" /\ rc1' = rc1 /\ rc2' = rc2
T8 == st = "dispatching" /\ st' = "persisting" /\ rc1' = rc1 /\ rc2' = rc2
T9 == st = "dispatching" /\ st' = "carrierRetry" /\ rc1' = rc1 /\ rc2' = rc2
T10 == st = "dispatching" /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
T11 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
T12 == st = "persisting" /\ st' = "Dispatched" /\ rc1' = 0 /\ rc2' = 0
T13 == st = "persisting" /\ st' = "InTransit" /\ rc1' = 0 /\ rc2' = 0
T14 == st = "persisting" /\ st' = "Delivered" /\ rc1' = 0 /\ rc2' = 0
T15 == st = "persisting" /\ st' = "Lost" /\ rc1' = 0 /\ rc2' = 0
T16 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
T17 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1 /\ rc2' = rc2
T18 == st = "persisting" /\ st' = "persistRetry" /\ rc1' = rc1 /\ rc2' = rc2
T19 == st = "persisting" /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
T20 == st = "rolledBack" /\ st' = "Pending" /\ rc1' = 0 /\ rc2' = 0
T21 == st = "rolledBack" /\ st' = "Dispatched" /\ rc1' = 0 /\ rc2' = 0
T22 == st = "rolledBack" /\ st' = "InTransit" /\ rc1' = 0 /\ rc2' = 0
RetryExhausted_carrierRetry == st = "carrierRetry" /\ rc1 >= MaxRetries /\ st' = "rolledBack" /\ rc1' = rc1 /\ rc2' = rc2
RetryAgain_carrierRetry == st = "carrierRetry" /\ rc1 < MaxRetries /\ st' = "dispatching" /\ rc1' = rc1 + 1 /\ rc2' = rc2
RetryExhausted_persistRetry == st = "persistRetry" /\ rc2 >= MaxRetries /\ st' = "rolledBack" /\ rc2' = rc2 /\ rc1' = rc1
RetryAgain_persistRetry == st = "persistRetry" /\ rc2 < MaxRetries /\ st' = "persisting" /\ rc2' = rc2 + 1 /\ rc1' = rc1
Terminated == st \in Final /\ UNCHANGED vars

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T5 \/ T6
OverlayNext == T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ RetryExhausted_carrierRetry \/ RetryAgain_carrierRetry \/ RetryExhausted_persistRetry \/ RetryAgain_persistRetry
Next == DomainNext \/ OverlayNext \/ Terminated

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
