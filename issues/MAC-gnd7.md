---
id: MAC-gnd7
title: "Integrity layer: tenant-scoped uniqueness via optional unique scope"
status: open
priority: 3
type: feature
labels: [relational, formal, h2-ask, unapproved-scope, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:34Z
content_hash: "sha256:e68308ff9358413e141b7e6e419f308710eff19924e42169381e395e5fcdf0ed"
---

## Description
Integrity layer: tenant-scoped uniqueness via optional unique `scope:`

Problem: `integrity.relational.yaml` unique rows are (entity, attribute, invariant) only (internal/alloy/integrity.go:129 `integrityUniqueKeys`), so a per-tenant uniqueness invariant such as H2's re-scoped `principal-email-unique` (unique within one Tenant, one Identity holding one principal per tenant) cannot be stated. It had to leave the layer (H2 DECISIONS 2026-09-04) and is carried by a composite index plus a matrix contract.

Proposed fix: add an optional `scope:` on a unique row naming the entity the uniqueness is keyed under, resolved through the record's tenancy attribution (reuse the isolation layer's tenant attribution where present), compiled to an Alloy fact and to the integrity oracle. The unscoped row stays the degenerate case. Unapproved feature scope until authorized.

Acceptance criteria:
1. A scoped unique row compiles; Alloy finds no counterexample for two records with equal attribute values in different scopes and does find one inside the same scope.
2. The oracle carries scoped rows and a test can bind them.
3. An unknown scope entity, a scope with no tenancy attribution path, or a scope on a singleton fails with a named diagnostic.
4. Unscoped designs (bundled examples, golden corpus) generate byte-identical output.
5. Docs and CHANGELOG record the form and bounded-scope proof limits.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
