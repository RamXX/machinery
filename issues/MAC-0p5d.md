---
id: MAC-0p5d
title: "Adopt Modelith 0.5.0 and simplify with its new features"
status: open
priority: 1
type: epic
labels: [modelith, dependencies]
created_at: 2026-09-25T20:11:07Z
created_by: ramirosalas
updated_at: 2026-09-25T20:11:07Z
content_hash: "sha256:d1c015861265799ce7e65cf38b28864f1218c51fba7b490ece76f55f53655bc1"
---

## Description
Adopt Modelith 0.5.0 (released 2026-09-24; machinery pins v0.4.0) and use its new features to simplify machinery.

Assessment 2026-09-25 (all 15 bundled models lint clean under 0.5.0; internal tests pass with 0.5.0 on PATH; no test runs real modelith, the live exposures are preflight and make modelith-render-check):
- Schema change is additive: imports/scope (same-repo models, qualified scope.Name refs), shared vocabulary models, modelith deps (vendoring with provenance), bounded/exact cardinalities (0..1, 1..n, 2), symmetric, subtypeOf, derived entities.
- Renders: 3 byte-identical, 8 change in Mermaid edges/labels only, prose unchanged except stray blank lines. 0.5 still emits em dashes, so machinery's normalization stays.
- Hazard: imports + qualified refs are silently mis-projected today (dangling rel:Account->vocab.Region, no enum_member facts, zero findings). Bounded cardinality on annotation edges fails loudly; symmetric/subtypeOf/derived are ignored harmlessly.
- Semantic shift: 0.5 defines bare `1` as exactly one; go-crm's `Team 1:n User` + `membership: lone` now disagrees with the Modelith doc silently.
- Not worth adopting: symmetric for Gy parallel-relationship naming (0.5 lint still silent there, keep Gy); modelith deps as pack replacement (provenance header breaks pack byte-match); subtypeOf vs SUPERSEDES (unrelated concepts).
Order: n:n bug, then the pin bump with loud refusal of unsupported forms, then simplifications, then bounded contexts (MAC-p88c).

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
