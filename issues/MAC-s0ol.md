---
id: MAC-s0ol
title: "Decision: --skip-plugins skips plugin management only; ownership discovery stays fail-closed"
status: open
priority: 4
type: decision
labels: [installer, policy, from-next]
created_at: 2026-09-24T21:32:35Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:35Z
content_hash: "sha256:b8645c7461b9583ecd4ba53352f1572a39ad81030a37d1920140585e2762b02d"
---

## Description
Decision: --skip-plugins skips host plugin management only; plugin ownership discovery stays fail-closed

Recorded policy: `--skip-plugins` skips only host plugin management. Plugin-ownership discovery runs before the skip is honored, because discovery decides which homes the plugin serves, and its errors abort rather than degrade to a warning. No unsafe fallback to defaults. Receipt and ownership inspection still run under --skip-plugins. Documented in docs/agent-portability.md:153; delivered and tested under MAC-2u36 (AC4) and MAC-gcrr. No further change unless the policy is reopened.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
