---
id: MAC-4nt7
title: "Decision: hierarchical isolation scoping (tenant > unit) deferred until an access-boundary need exists"
status: closed
priority: 4
type: decision
labels: [relational, isolation, deferred, from-next]
created_at: 2026-09-24T21:32:35Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:49Z
content_hash: "sha256:17def3b15ddc1fd77a4112d0d356629bb0bc7cd32f9b49ee122ce65de0b326cc"
closed_at: 2026-09-24T21:32:49Z
close_reason: "Deferred by owner ('do not pull forward'). Reopen when a design needs an access boundary below the tenant."
---

## Description
Decision: hierarchical isolation scoping (tenant > unit) is deferred until a concrete access-boundary need exists

Context: `isolation.relational.yaml` admits exactly one tenant entity with `lone` subject membership on the subject's own attribute (internal/alloy/isolation.go:87-88). H2's 2026-09-04 identity revision kept Principal as subject (Identity is tenantless and never acts), so it needed no change. The open case is BusinessUnit, a lens today that may become an access boundary post-v1.

Decision: do not pull forward. If reopened, the shape is a scoping hierarchy (tenant > unit) with references checked at the declared level, keeping the single-level form as the degenerate case so generated Isolation.als and oracle stay byte-identical for existing designs. MAC-ui8a and MAC-gcrr already recorded this as "explicitly deferred hierarchy"; do not infer security semantics for it.

Reopen trigger: a consumer design declares a second access boundary below the tenant.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:32:49Z status: open -> closed

## Links


## Comments
