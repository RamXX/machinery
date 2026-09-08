---
id: MAC-y2l2
title: "Bug: governance hook-state grows unboundedly under heavy local test sweeps"
status: in_progress
priority: 1
type: bug
parent: MAC-ui8a
created_at: 2026-09-08T02:21:09Z
created_by: ramirosalas
updated_at: 2026-09-08T05:53:07Z
content_hash: "sha256:5dffe21385432e14d5566e010d593c0df7a533d4cc49f2968c546d5cb312892e"
assignee: dev-MAC-y2l2
follows: [MAC-3hzt]
---

## Description
Self-inflicted during release-candidate verification: go test -race ./... sweeps invoke the machinery CLI thousands of times; each hook invocation writes a route snapshot into ~/Library/Application Support/machinery-hook-state-<hash>/ with no retention. At 4096 entries the hook fails closed and bricks ALL shell/write tooling for every agent in the repo (including machinery doctor itself, which cannot remediate its own state). Remediation required a manual user prune. Needs: retention/compaction policy (e.g. keep newest N per route, size-bounded), and doctor must be able to repair its own state even at the limit. hwdb lineage. Evidence: session logs 2026-09-07, dir machinery-hook-state-288286948b3b49e6b991052d peaked >4096.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-08T05:53:07Z status: open -> in_progress
- 2026-09-08T05:53:07Z auto-follows: linked to predecessor MAC-3hzt
- 2026-09-08T05:53:07Z claimed by dev-MAC-y2l2

## Links
- Parent: [[MAC-ui8a]]
- Follows: [[MAC-3hzt]]

## Comments
