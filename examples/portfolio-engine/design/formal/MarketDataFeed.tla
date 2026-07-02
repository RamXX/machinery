---- MODULE MarketDataFeed ----
EXTENDS Naturals

\* Generated from MarketDataFeed.machine.json by tools/tla_gen.py. Control-flow model.
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
CONSTANT MaxRetries
VARIABLES st
vars == << st >>

States == {"closed", "halfOpen", "open"}
Domain == {"closed", "halfOpen"}
Overlay == {"open"}

TypeOK == st \in States /\ TRUE
Init == st = "closed"

  \* T1: closed -on:failure-> open
  \* T2: closed -on:failure-> closed
  \* T3: closed -on:success-> closed
  \* T4: open -after:COOLDOWN-> halfOpen
  \* T5: halfOpen -on:probeResult-> closed
  \* T6: halfOpen -on:probeResult-> open

T1 == st = "closed" /\ st' = "open"
T2 == st = "closed" /\ st' = "closed"
T3 == st = "closed" /\ st' = "closed"
T4 == st = "open" /\ st' = "halfOpen"
T5 == st = "halfOpen" /\ st' = "closed"
T6 == st = "halfOpen" /\ st' = "open"

DomainNext == T1 \/ T2 \/ T3 \/ T5 \/ T6
OverlayNext == T4
Next == DomainNext \/ OverlayNext

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
