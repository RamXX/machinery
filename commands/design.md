---
description: Start or resume a machinery design run (four phases, gated)
argument-hint: "[greenfield|brownfield|rebuild|hybrid] what you want to design"
---

Run a machinery design session in this repository. Everything below is
conductor instruction, never language to repeat to the user: speak in the
skill's dialog register (plain step names, findings translated to their
meaning, no gate ids, phase numbers, or CLI invocations unless the user
uses them first).

1. Invoke the `machinery` skill and follow it as the conductor. Internally
   you will run the four-phase interrogation (Modelith domain model, C4
   architecture, XState machines, BUILD.md) with a deterministic gate
   between phases; to the user these are simply the domain model, the
   architecture, the behavior model, and the build plan.
2. If `design/STATE.md` (or the design directory named in `.machinery.json`)
   exists, read it first and resume from the recorded phase instead of
   starting over.
3. If the request or the repository indicates an existing system (code,
   schema, deployments already present), run the skill's brownfield
   (archaeology) mode: describe the system as it is, do not invent.
4. If the user wants a new production foundation while preserving selected behavior, data, tests,
   or modules, run rebuild/hybrid mode instead. Internally: keep separate legacy and target domain
   models, author `migration.yaml`, and hold it with Gm-transition; never collapse current and
   intended truth into one model. Whenever a legacy system exists, also author the surface ledger
   (`design/legacy/surface.yaml`, held by Gs-surface): the opening sweep seeds it and the closing
   sweep after Gate 4 settles every row. In every mode, author the target surface ledger
   (`design/surfaces.yaml`, held by Gu-surfaces) during Phase 2: walk each human persona's complete
   action list into named surfaces before Gate 2. To the user this is one conversation about what
   exists, what carries over, and where each person goes to do each thing.
5. Treat the user's request below as the opening frame. If it is empty,
   open by asking for the frame in plain language, for example: "What are
   we building, who uses it, and what should it be written in?"

Request: $ARGUMENTS
