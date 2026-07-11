---
description: Start or resume a machinery design run (four phases, gated)
argument-hint: "[greenfield|brownfield|rebuild|hybrid] what you want to design"
---

Run a machinery design session in this repository.

1. Invoke the `machinery` skill and follow it as the conductor: the
   four-phase interrogation (Modelith domain model, C4 architecture, XState
   machines, BUILD.md) with a deterministic gate between phases.
2. If `design/STATE.md` (or the design directory named in `.machinery.json`)
   exists, read it first and resume from the recorded phase instead of
   starting over.
3. If the request or the repository indicates an existing system (code,
   schema, deployments already present), run the skill's brownfield
   (archaeology) mode: describe the system as it is, do not invent.
4. If the user wants a new production foundation while preserving selected behavior, data, tests,
   or modules, run rebuild/hybrid mode instead: keep separate legacy and target domain models,
   author `migration.yaml`, and hold it with Gm-transition. Do not collapse current and intended
   truth into one model. Whenever a legacy system exists, also author the surface ledger
   (`design/legacy/surface.yaml`, held by Gs-surface): the opening sweep seeds it and the closing
   sweep after Gate 4 settles every row.
5. Treat the user's request below as the Phase 0 frame input. If it is
   empty, open Phase 0 by asking for the frame.

Request: $ARGUMENTS
