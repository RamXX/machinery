---
id: MAC-y2l2
title: "Bug: governance hook-state grows unboundedly under heavy local test sweeps"
status: open
priority: 1
type: bug
parent: MAC-ui8a
created_at: 2026-09-08T02:21:09Z
created_by: ramirosalas
updated_at: 2026-09-08T02:21:09Z
content_hash: "sha256:770a733e7e009846be5b97852df851d7fe197d3f55f38a1152bf150651ea47e0"
---

## Description
Self-inflicted during release-candidate verification: go test -race ./... sweeps invoke the machinery CLI thousands of times; each hook invocation writes a route snapshot into ~/Library/Application Support/machinery-hook-state-<hash>/ with no retention. At 4096 entries the hook fails closed and bricks ALL shell/write tooling for every agent in the repo (including machinery doctor itself, which cannot remediate its own state). Remediation required a manual user prune. Needs: retention/compaction policy (e.g. keep newest N per route, size-bounded), and doctor must be able to repair its own state even at the limit. hwdb lineage. Evidence: session logs 2026-09-07, dir machinery-hook-state-288286948b3b49e6b991052d peaked >4096.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
