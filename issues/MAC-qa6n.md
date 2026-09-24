---
id: MAC-qa6n
title: "Policy layer: set-valued subject holdings keyed by capability (composable roles)"
status: open
priority: 3
type: feature
labels: [relational, formal, h2-ask, unapproved-scope, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:34Z
content_hash: "sha256:d879026d49b961939a7e749aca2c4302475fc6754fd6d43da88ab501c198f654"
blocked_by: [MAC-p8f1]
---

## Description
Policy layer: set-valued subject holdings keyed by capability (H2 composable roles)

Problem: `policy.relational.yaml` admits exactly one role attribute per subject (`subjects.role_attr`, required; internal/alloy/alloy.go:269 and 432-434). H2 models composable TenantRoles, where a principal's held capability set is the union of its assigned roles, so its capability re-base is staged "behind a machinery enhancement" (H2 DECISIONS 2026-08-31).

Proposed fix: extend the policy annotation to a set-valued holding (the subject holds a set of register keys; grants are keyed by capability, not by role), generate the Alloy model and the policy oracle over capability rows, and keep the `role_attr: preset` form as the degenerate case so existing designs compile unchanged. Unapproved feature scope until the owner authorizes it.

Acceptance criteria:
1. A synthetic design with a set-valued holding compiles to Alloy and the oracle; a grant reachable only through the union of two roles is admitted and one outside every held role is denied.
2. Existing `role_attr` designs (bundled examples and golden corpus) generate byte-identical Alloy and oracle output.
3. Malformed holdings (unknown register key, capability not declared, both forms at once) fail with a named diagnostic.
4. Documentation and CHANGELOG (Compatibility, Generated output, Proof scope) describe the new form and its bounded-scope limits.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:33:57Z dep_added: blocked_by MAC-p8f1

## Links
- Blocked by: [[MAC-p8f1]]

## Comments
