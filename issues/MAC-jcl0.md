---
id: MAC-jcl0
title: "Fix go-crm G4 boundary mapping for internal/testoracle"
status: open
priority: 1
type: bug
assignee: dev-g4
parent: MAC-ui8a
created_at: 2026-09-06T23:26:40Z
created_by: ramirosalas
updated_at: 2026-09-06T23:26:40Z
content_hash: "sha256:ba72d469f7b5035f989e52fcad842ac2acacdb3f0272370ce2dc3cd884d28705"
---

## Description
Pre-existing G4 finding surfaced truthfully by MAC-hgz1 goldens: internal/testoracle/fsm.go maps to no boundary in examples/go-crm/design/workspace.dsl, causing check-go-crm exit 1 and breaking frozen p7 TestAttestImplementationCLI/C-complete-sole-current-warning at baseline. Fix the design boundary mapping truthfully (testoracle is part of the go-crm example implementation), refresh affected golden with same-SHA capture, verify TestGoldenCheck + the two pre-existing failing tests flip honestly if G4 was their cause. Discovered during hgz1/wi5z verification; proven pre-existing at baseline.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
