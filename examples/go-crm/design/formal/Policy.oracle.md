# Generated authorization oracle: policy

Generated from `domain.modelith.yaml` + `policy.relational.yaml` by `machinery alloy`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.4-dev -->
Single source of truth for the authorization tests: one row is one decision case
for the pure authz function. Key tests on the STABLE id, not the row number; row
numbers renumber when the design changes, stable ids do not. A design revision
flips an expectation under an unchanged stable id, so the test diff names exactly
which cases changed behavior.

The scope algebra decides every case from two booleans (does the actor own the
record; do the actor and the owner share a team), so this table is the COMPLETE
semantics of the policy: two configurations that agree on the case columns are
policy-equivalent, and a correct implementation agrees with every reachable row.

## Case vocabulary

Owner cases (relationship between the actor and the record owner):

- `own-teamed`: the actor owns the record and holds a team.
- `own-teamless`: the actor owns the record and holds no team.
- `teammate`: the record is owned by another member of the actor's team.
- `outsider`: the owner is neither the actor nor a teammate. Concrete variants
  (all policy-equivalent, test every one): owner in a different team; owner
  teamless; actor teamless with any other owner.

Target cases (reassign only; where the record would go):

- `target-teammate`: the new owner is a member of the actor's team (the actor
  itself, when teamed, is its own teammate).
- `target-outsider`: the new owner is not a member of the actor's team.
- `any`: the rule places no constraint on the target for this role.

Rows marked `unreachable` are configurations the named invariants forbid; the
write discipline refuses to construct them, and authorization behavior on them
is unspecified. Tests skip them (or assert the construction is refused).

## Decisions

| test id | stable id | verb | role | owner case | target | expectation | invariants |
|---|---|---|---|---|---|---|---|
| O-AUTHZ-01 | AUTHZ-21866c | create | Admin | - | - | allow | rbac-crud-verbs |
| O-AUTHZ-02 | AUTHZ-63b862 | create | Manager | - | - | allow | rbac-crud-verbs |
| O-AUTHZ-03 | AUTHZ-06b8a0 | create | Rep | - | - | allow | rbac-crud-verbs |
| O-AUTHZ-04 | AUTHZ-d6443f | create | ReadOnly | - | - | deny | rbac-crud-verbs |
| O-AUTHZ-05 | AUTHZ-b7c2db | read | Admin | own-teamed | - | allow | rbac-read-visibility |
| O-AUTHZ-06 | AUTHZ-24780c | read | Admin | own-teamless | - | allow | rbac-read-visibility |
| O-AUTHZ-07 | AUTHZ-76c225 | read | Admin | teammate | - | allow | rbac-read-visibility |
| O-AUTHZ-08 | AUTHZ-4ea0cc | read | Admin | outsider | - | allow | rbac-read-visibility |
| O-AUTHZ-09 | AUTHZ-3748b1 | read | Manager | own-teamed | - | allow | rbac-read-visibility |
| O-AUTHZ-10 | AUTHZ-0048e5 | read | Manager | own-teamless | - | unreachable | single-team, manager-has-team |
| O-AUTHZ-11 | AUTHZ-950c3c | read | Manager | teammate | - | allow | rbac-read-visibility |
| O-AUTHZ-12 | AUTHZ-374aae | read | Manager | outsider | - | deny | rbac-read-visibility |
| O-AUTHZ-13 | AUTHZ-fc14f2 | read | Rep | own-teamed | - | allow | rbac-read-visibility |
| O-AUTHZ-14 | AUTHZ-3ba6fd | read | Rep | own-teamless | - | allow | rbac-read-visibility |
| O-AUTHZ-15 | AUTHZ-582aba | read | Rep | teammate | - | allow | rbac-read-visibility |
| O-AUTHZ-16 | AUTHZ-f5d436 | read | Rep | outsider | - | deny | rbac-read-visibility |
| O-AUTHZ-17 | AUTHZ-54760d | read | ReadOnly | own-teamed | - | allow | rbac-read-visibility |
| O-AUTHZ-18 | AUTHZ-610903 | read | ReadOnly | own-teamless | - | allow | rbac-read-visibility |
| O-AUTHZ-19 | AUTHZ-36c864 | read | ReadOnly | teammate | - | allow | rbac-read-visibility |
| O-AUTHZ-20 | AUTHZ-6d9aec | read | ReadOnly | outsider | - | deny | rbac-read-visibility |
| O-AUTHZ-21 | AUTHZ-afac0d | update | Admin | own-teamed | - | allow | rbac-write-scope |
| O-AUTHZ-22 | AUTHZ-f83379 | update | Admin | own-teamless | - | allow | rbac-write-scope |
| O-AUTHZ-23 | AUTHZ-8884ab | update | Admin | teammate | - | allow | rbac-write-scope |
| O-AUTHZ-24 | AUTHZ-ce036c | update | Admin | outsider | - | allow | rbac-write-scope |
| O-AUTHZ-25 | AUTHZ-b13aef | update | Manager | own-teamed | - | allow | rbac-write-scope |
| O-AUTHZ-26 | AUTHZ-20b826 | update | Manager | own-teamless | - | unreachable | single-team, manager-has-team |
| O-AUTHZ-27 | AUTHZ-99df64 | update | Manager | teammate | - | allow | rbac-write-scope |
| O-AUTHZ-28 | AUTHZ-a5226e | update | Manager | outsider | - | deny | rbac-write-scope |
| O-AUTHZ-29 | AUTHZ-8193df | update | Rep | own-teamed | - | allow | rbac-write-scope |
| O-AUTHZ-30 | AUTHZ-f03de6 | update | Rep | own-teamless | - | allow | rbac-write-scope |
| O-AUTHZ-31 | AUTHZ-8acd04 | update | Rep | teammate | - | deny | rbac-write-scope |
| O-AUTHZ-32 | AUTHZ-f71187 | update | Rep | outsider | - | deny | rbac-write-scope |
| O-AUTHZ-33 | AUTHZ-02ae72 | update | ReadOnly | own-teamed | - | deny | rbac-crud-verbs |
| O-AUTHZ-34 | AUTHZ-f7e5e1 | update | ReadOnly | own-teamless | - | deny | rbac-crud-verbs |
| O-AUTHZ-35 | AUTHZ-df5dcf | update | ReadOnly | teammate | - | deny | rbac-crud-verbs |
| O-AUTHZ-36 | AUTHZ-545bba | update | ReadOnly | outsider | - | deny | rbac-crud-verbs |
| O-AUTHZ-37 | AUTHZ-8bb7e6 | delete | Admin | own-teamed | - | allow | rbac-write-scope |
| O-AUTHZ-38 | AUTHZ-92d11c | delete | Admin | own-teamless | - | allow | rbac-write-scope |
| O-AUTHZ-39 | AUTHZ-acd4a6 | delete | Admin | teammate | - | allow | rbac-write-scope |
| O-AUTHZ-40 | AUTHZ-fe22fc | delete | Admin | outsider | - | allow | rbac-write-scope |
| O-AUTHZ-41 | AUTHZ-9f8279 | delete | Manager | own-teamed | - | allow | rbac-write-scope |
| O-AUTHZ-42 | AUTHZ-d33642 | delete | Manager | own-teamless | - | unreachable | single-team, manager-has-team |
| O-AUTHZ-43 | AUTHZ-9b3466 | delete | Manager | teammate | - | allow | rbac-write-scope |
| O-AUTHZ-44 | AUTHZ-017d70 | delete | Manager | outsider | - | deny | rbac-write-scope |
| O-AUTHZ-45 | AUTHZ-b9cbb0 | delete | Rep | own-teamed | - | allow | rbac-write-scope |
| O-AUTHZ-46 | AUTHZ-a65f6e | delete | Rep | own-teamless | - | allow | rbac-write-scope |
| O-AUTHZ-47 | AUTHZ-e394f9 | delete | Rep | teammate | - | deny | rbac-write-scope |
| O-AUTHZ-48 | AUTHZ-1da44b | delete | Rep | outsider | - | deny | rbac-write-scope |
| O-AUTHZ-49 | AUTHZ-e1caaf | delete | ReadOnly | own-teamed | - | deny | rbac-crud-verbs |
| O-AUTHZ-50 | AUTHZ-55a039 | delete | ReadOnly | own-teamless | - | deny | rbac-crud-verbs |
| O-AUTHZ-51 | AUTHZ-0d7332 | delete | ReadOnly | teammate | - | deny | rbac-crud-verbs |
| O-AUTHZ-52 | AUTHZ-0a62af | delete | ReadOnly | outsider | - | deny | rbac-crud-verbs |
| O-AUTHZ-53 | AUTHZ-e2bc72 | reassign | Admin | own-teamed | any | allow | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-54 | AUTHZ-5c6425 | reassign | Admin | own-teamless | any | allow | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-55 | AUTHZ-c1e494 | reassign | Admin | teammate | any | allow | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-56 | AUTHZ-8775f2 | reassign | Admin | outsider | any | allow | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-57 | AUTHZ-fbc244 | reassign | Manager | own-teamed | target-teammate | allow | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-58 | AUTHZ-3cb8e2 | reassign | Manager | own-teamed | target-outsider | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-59 | AUTHZ-f50d28 | reassign | Manager | own-teamless | - | unreachable | single-team, manager-has-team |
| O-AUTHZ-60 | AUTHZ-13a179 | reassign | Manager | teammate | target-teammate | allow | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-61 | AUTHZ-20d68e | reassign | Manager | teammate | target-outsider | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-62 | AUTHZ-d87990 | reassign | Manager | outsider | - | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-63 | AUTHZ-b4b911 | reassign | Rep | own-teamed | - | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-64 | AUTHZ-d08d5d | reassign | Rep | own-teamless | - | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-65 | AUTHZ-6f6cbf | reassign | Rep | teammate | - | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-66 | AUTHZ-e5bf1a | reassign | Rep | outsider | - | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-67 | AUTHZ-977c09 | reassign | ReadOnly | own-teamed | - | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-68 | AUTHZ-b82fd0 | reassign | ReadOnly | own-teamless | - | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-69 | AUTHZ-21b434 | reassign | ReadOnly | teammate | - | deny | rbac-reassign-authority, task-assignee-visible |
| O-AUTHZ-70 | AUTHZ-9d609b | reassign | ReadOnly | outsider | - | deny | rbac-reassign-authority, task-assignee-visible |
