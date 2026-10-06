---
id: MAC-0p5d
title: "Adopt Modelith 0.5.0 and simplify with its new features"
status: open
priority: 2
type: epic
labels: [modelith, dependencies]
created_at: 2026-09-25T20:11:07Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:57Z
content_hash: "sha256:1279e0c1448d1ff73f20c53295a860ff63f5a4280fca29710baf222ec2a2485b"
related: [MAC-syos]
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
Revalidated 2026-10-06 against v0.11.0: partial. Remaining: Children remaining: gscc (n:n), g7d4 (loud refusal and breaking notes, e2e proof), o3p2, r3b0; bounded contexts via MAC-p88c. Epic still makes sense; its description of 'machinery pins v0.4.0' is stale. Evidence: Pin bump to v0.5.0 landed in 2bd125b0 (Makefile:18, cmd/machinery/diag.go:30, scripts/modelith-render.sh:8, .dagger/main.go:441,616, README, brownfield guide; examples re-rendered); CHANGELOG 0.11.0 lines 116,130 note pin. n:n (MAC-gscc) still open: internal/checker/model.go:199-203. No loud refusal of imports/qualified refs found. Notes: Children: MAC-gscc, MAC-g7d4, MAC-o3p2, MAC-r3b0 (and p88c/p46v are blocked by g7d4 but belong to other epics). Update epic description to note pin done.

## History


## Links
- Related: [[MAC-syos]]

## Comments
