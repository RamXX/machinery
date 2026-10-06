---
id: MAC-g7d4
title: "Bump Modelith pin to v0.5.0; refuse imports and qualified refs loudly until bounded contexts land"
status: open
priority: 1
type: feature
labels: [modelith, dependencies]
parent: MAC-0p5d
created_at: 2026-09-25T20:11:07Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:57Z
content_hash: "sha256:a9b696746fae91def620dd1f8cb82a44a72511da0e464a1391bf5796ddb85df1"
blocks: [MAC-r3b0, MAC-o3p2]
was_blocked_by: [MAC-gscc]
---

## Description
Bump the Modelith pin to v0.5.0 and make unsupported 0.5 features fail loudly instead of mis-projecting.

Pin sites: Makefile:18; cmd/machinery/diag.go:30 (exact-match preflight); scripts/modelith-render.sh:8; .dagger/main.go:441,616; README.md:991,993,1397; docs/brownfield-team-guide.md:62,342; test literals cmd/machinery/repository_contract_test.go:909,1060 (fake-0.4.0 fixtures in diag/install/modelithtx/run-safe tests need no change).

Loud refusal until bounded contexts land (MAC-p88c): the gates model reader rejects `imports:`, dotted entity/type refs (scope.Name), and dotted subtypeOf with "not yet supported by machinery" naming the file and line. Today they project a dangling relationship with zero findings.

Acceptance criteria:
1. Every pin site moved; preflight and make modelith-render-check pass with 0.5.0 and fail with 0.4.0 with a clear message.
2. The 8 changed example renders regenerated (Mermaid only), render check green in CI (Dagger).
3. A probe model with imports + qualified refs fails check with the new error; bounded cardinality on non-annotation edges still projects as stated.
4. CHANGELOG breaking notes for consumers: install 0.5.0; committed renders drift in Mermaid and must be re-rendered; bare `1` now means exactly one (flag models whose annotation says membership lone on such an edge); bounded and qualified forms refused where noted.
5. End-to-end proof: real modelith 0.5.0 + real machinery binary over every bundled example design, check green.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: partial. Remaining: Implement loud refusal of imports:, dotted entity/type refs and dotted subtypeOf in the gates model reader; CHANGELOG consumer notes (re-render drift, bare 1 meaning, refused forms); probe-model test; e2e over bundled examples with real modelith 0.5.0. Evidence: Pin sites all moved to v0.5.0 and examples re-rendered in 2bd125b0 (Makefile:18, diag.go:30, modelith-render.sh:8, .dagger/main.go, README:1004-1411, docs/brownfield-team-guide.md:62,367). No 'not yet supported by machinery' refusal of imports:/dotted refs exists (grep finds only projection layer 'scenarios' refusal, internal/checker/projection.go:129). 0.5 bare-1 semantic note not in CHANGELOG. Notes: Silent mis-projection of imports/qualified refs is a gate correctness hole (zero findings). Blocker gscc is not technical: n:n handling is independent of the refusal logic (only AC2 of gscc, bounded-form failure, overlaps and can be done in one pass with g7d4). Pin part could be closed out and the rest retitled.

## History
- 2026-09-25T20:11:08Z dep_added: blocked_by MAC-gscc
- 2026-09-25T20:11:08Z dep_added: blocks MAC-r3b0
- 2026-09-25T20:11:08Z dep_added: blocks MAC-o3p2
- 2026-09-25T20:11:08Z dep_added: blocks MAC-p88c
- 2026-09-25T20:11:08Z dep_added: blocks MAC-p46v
- 2026-10-06T04:03:39Z dep_removed: was_blocked_by MAC-gscc
- 2026-10-06T04:03:39Z dep_removed: no_longer_blocks MAC-p46v
- 2026-10-06T04:03:40Z dep_removed: no_longer_blocks MAC-p88c

## Links
- Parent: [[MAC-0p5d]]
- Blocks: [[MAC-r3b0]], [[MAC-o3p2]]
- Was blocked by: [[MAC-gscc]]

## Comments
