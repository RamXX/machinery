---
id: MAC-84ur
title: "Twin rule residual: a residual mark resolving in both policy layer and model is a finding (confirm still relevant)"
status: open
priority: 4
type: feature
labels: [gy, consistency-layer, twins, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:50Z
content_hash: "sha256:f82f2c919cac6895b78d0648793bdbe345fe7e6f5e1d3c0044db7c062ca887a8"
---

## Description
Twin rule residual: a residual mark resolving in both the policy layer and the model is a finding

Problem: NEXT entry 37 (gap class D, twins that must agree). General rule: where the design states one fact in two artifacts by design, the second statement is either generated from the first or compared to it, and a prose restatement is a finding.

Delivered: payload cells vs the Architecture Contract (0.9.0 Gx, later `payload_twin` in Gy-rules 0.10.0); the BUILD.md milestone/oracle "bound at" table vs Gt-bound suites under `--impl` (0.10.0, `rules/consistency/bindings.dl`, commit 2db644d7, findings `milestone_binding_stale` and `milestone_binding_phantom`). The milestone table is compared, not generated, which satisfies the rule.

Residual (not delivered): a residual mark whose id resolves in the policy layer OR the model (two artifacts claiming one fact). No rule under rules/consistency/ covers residual marks. Note: 0.10.0 stopped reading design-private residual verb tables, so first confirm whether H2 still carries residual marks that machinery reads; if not, close this as moot.

Proposed fix: a Gy rule (for example `residual_twin`) that fires when a residual id declared in a relational layer's `residuals:` section also resolves as a model invariant/attribute with a different disposition, or resolves in neither; project the needed relations in projection 2.0.

Acceptance criteria:
1. A fixture whose residual id resolves in both the policy layer and the model with disagreeing dispositions fails with a finding naming both sources (path:line).
2. A residual id resolving in exactly one place passes.
3. Bundled examples stay green; `rules/README.md` lists the new finding.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
