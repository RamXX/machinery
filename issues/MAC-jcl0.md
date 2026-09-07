---
id: MAC-jcl0
title: "Fix go-crm G4 boundary mapping for internal/testoracle"
status: in_progress
priority: 1
type: bug
assignee: dev-g4
parent: MAC-ui8a
created_at: 2026-09-06T23:26:40Z
created_by: ramirosalas
updated_at: 2026-09-07T00:29:09Z
content_hash: "sha256:e8bdb28d4bfc021dc19631bf49a20a889dedbc81a3a3a22776170d30c423ae3e"
follows: [MAC-wi5z]
labels: [accepted]
---

## Description
Pre-existing G4 finding surfaced truthfully by MAC-hgz1 goldens: internal/testoracle/fsm.go maps to no boundary in examples/go-crm/design/workspace.dsl, causing check-go-crm exit 1 and breaking frozen p7 TestAttestImplementationCLI/C-complete-sole-current-warning at baseline. Fix the design boundary mapping truthfully (testoracle is part of the go-crm example implementation), refresh affected golden with same-SHA capture, verify TestGoldenCheck + the two pre-existing failing tests flip honestly if G4 was their cause. Discovered during hgz1/wi5z verification; proven pre-existing at baseline.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-06T23:26:40Z status: open -> in_progress
- 2026-09-06T23:26:40Z auto-follows: linked to predecessor MAC-wi5z

## Links
- Parent: [[MAC-ui8a]]
- Follows: [[MAC-wi5z]]

## Comments

### 2026-09-07T00:29:09Z ramirosalas
ACCEPTED 2026-09-06 — root cause: internal/testoracle is test-only code (imported exclusively by _test.go); correct remedy is the Architecture Contract ignore list alongside testsupport/arch (ARCHITECTURE.md:118), exactly what the G4 error names. Golden refreshed via sanctioned -update; exitcode 1->0; Gt accounting re-classification 137->197 literals (classifier shared by design; zero missing-id errors); three pre-existing baseline failures flipped PASS (TestAttestImplementationCLI incl. frozen C-complete-sole-current-warning, TestCheckGreenSummaryLines, TestCheckImportsCleanOnGoCRM). 2 commits, 4 files +25/-23. Six g2.* claims hash-rebound with amendment notes. Coordinator verified + merged; targeted suites ok on epic.
