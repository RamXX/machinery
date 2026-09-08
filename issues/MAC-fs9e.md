---
id: MAC-fs9e
title: "Ga-accept accepts an acceptance entry whose commit names no commit in the repository"
status: open
priority: 0
type: bug
labels: [gates, ga-accept, fail-open]
created_at: 2026-09-08T20:30:48Z
created_by: ramirosalas
updated_at: 2026-09-08T20:30:48Z
content_hash: "sha256:0f49ec365bdf694eafa46174848a1ab1f77743d393092cb57b4a472782ee9db8"
---

## Description
## Symptom
With design/acceptance/M1.yaml carrying commit: deadbeef... (40 hex digits that name no commit in the repository holding the design), machinery check design reports Ga-accept ok with no finding, on both v0.6.11 and the 0.7.0 candidate, and the checked: line names no binding mode. docs/acceptance-gate.md says the commit must resolve and be an ancestor of the design head. Found by probe on H2 (2026-09-08) while creating the M1 acceptance entry; the committed file carries a real sha.

## Why it matters
Ga-accept is the milestone-closure gate; accepting an unresolvable commit is fail-open on the seal. The nightly golden failure on 2026-09-07 showed the opposite side of the same binding (a real commit unresolvable in a shallow checkout was reported), so the resolution path exists but is not applied here, or is skipped when the milestone is open.

## Ask
Reproduce with a fixture (open milestone, acceptance entry with an unresolvable commit), make Ga-accept fail closed with the exact diagnostic (commit names no commit in the repository holding the design, or is not an ancestor), and print the binding mode on the checked: line. Cover the open and closed milestone states.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
