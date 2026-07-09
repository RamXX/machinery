---
description: Report machinery design status (phase ledger plus gate health)
allowed-tools: Bash(machinery:*), Read, Grep, Glob
---

Report where this repository's machinery design stands.

1. Read `.machinery.json` (if present) for the design directory, staged
   gates, impl dir, and strict mode; state which are in effect.
2. Read `<design>/STATE.md` if present and report per-phase status and open
   questions.
3. Inventory which artifacts exist (target domain model, optional legacy domain model and
   `migration.yaml`, the relational annotations
   `formal/{policy,integrity,isolation}.relational.yaml` with their generated
   models and oracles, workspace.dsl and Architecture Contract, machines with
   matrices and oracles, formal/, BUILD.md, decomposition/packs) so the
   current phase is visible even without STATE.md.
4. Run `machinery check <design>` with the configured gates and impl, and
   summarize: which gates are green, ERROR and DRIFT counts, and the single
   most important next action.
