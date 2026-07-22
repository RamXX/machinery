# Generated tenant-scoping oracle: isolation

Generated from `domain.modelith.yaml` + `isolation.relational.yaml` by `machinery alloy`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.4-dev -->
Single source of truth for the link-authorization test: one row is one decision
case for the pure tenant-scoping function that decides whether a reference may
be established. Key tests on the STABLE id, not the row number; row numbers
renumber when the design changes, stable ids do not.

A link From -> To is allowed exactly when the source owner and the target owner
share a tenant. tenant(record) = owner's team; a teamless owner has no tenant, so
a link touching one is refused. This table is the COMPLETE semantics: the pure
function decides every case from whether the two owners share a tenant.

## Case vocabulary

Tenant cases (relationship between the source owner and the target owner):

- `same-tenant`: both owners hold a tenant and it is the same one.
- `cross-tenant`: both owners hold a tenant and they differ.
- `source-teamless`: the source record's owner holds no tenant.
- `target-teamless`: the target record's owner holds no tenant.

## Decisions

| test id | stable id | reference | tenant case | expectation | invariants |
|---|---|---|---|---|---|
| O-TENANT-01 | TENANT-97e9c6 | Task.deal -> Deal | same-tenant | allow | task-deal-same-tenant |
| O-TENANT-02 | TENANT-ab139b | Task.deal -> Deal | cross-tenant | deny | task-deal-same-tenant |
| O-TENANT-03 | TENANT-3ff52e | Task.deal -> Deal | source-teamless | deny | task-deal-same-tenant |
| O-TENANT-04 | TENANT-fd4ae5 | Task.deal -> Deal | target-teamless | deny | task-deal-same-tenant |
| O-TENANT-05 | TENANT-f6e72d | Activity.contact -> Contact | same-tenant | allow | activity-contact-same-tenant |
| O-TENANT-06 | TENANT-49174b | Activity.contact -> Contact | cross-tenant | deny | activity-contact-same-tenant |
| O-TENANT-07 | TENANT-0b87ae | Activity.contact -> Contact | source-teamless | deny | activity-contact-same-tenant |
| O-TENANT-08 | TENANT-54ea9e | Activity.contact -> Contact | target-teamless | deny | activity-contact-same-tenant |
