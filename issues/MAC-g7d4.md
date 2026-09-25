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
updated_at: 2026-09-25T20:11:07Z
content_hash: "sha256:7eb9d0be8525ac857da534faba2b4d87779bd236f386d4d7508252c9e47d9dc6"
blocked_by: [MAC-gscc]
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


## History
- 2026-09-25T20:11:08Z dep_added: blocked_by MAC-gscc

## Links
- Parent: [[MAC-0p5d]]
- Blocked by: [[MAC-gscc]]

## Comments
