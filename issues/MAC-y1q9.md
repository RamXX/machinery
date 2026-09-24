---
id: MAC-y1q9
title: "Decision: checker registry inputs stay registry-relative; repo-root --registry is the workaround"
status: open
priority: 4
type: decision
labels: [external-checkers, policy, from-next]
created_at: 2026-09-24T21:32:35Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:35Z
content_hash: "sha256:771e83288ef73555867b17ce3d7c8944dc0d4be306ebf67e69c9eb50ff9d40c6"
---

## Description
Decision: checker registry inputs stay registry-relative; repo-root --registry is the supported workaround

Recorded policy: registry `inputs` resolve against the directory containing the registry file, with no `..`, so the default `.machinery/checkers.local.yaml` forces a copy of a committed adapter under `.machinery/`. Supported workaround (docs/external-checkers.md:316, 349): place the registry beside the committed inputs (for example repo root) and pass `--registry`; the derived closure is unchanged. Allowing repo-root-relative sources from the default location is an unapproved feature extension. Documented under MAC-gcrr AC2.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
