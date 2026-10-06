---
id: MAC-qax4
title: "Docs: e2e proof is the definition of done; hard TDD is a rigor option"
status: open
priority: 2
type: task
labels: [e2e, docs, dod]
parent: MAC-9dai
created_at: 2026-09-25T19:53:32Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:56Z
content_hash: "sha256:e5ce568fd27e28c224fa4c7ac8dcc461cb66af0126cd3157e1c5b9cae4d1f8d4"
blocked_by: [MAC-p3pj, MAC-lioz, MAC-4ah8]
---

## Description
Reposition the methodology docs so e2e proof is the mandatory definition of done and hard TDD is a rigor option, not the headline.

Scope: README (proof ladder, "What machinery does not verify"), docs/acceptance-gate.md, docs/test-assurance-contract.md intro, skills/machinery SKILL.md and references/build-md-template.md, agents/machinery-build-writer.md. State plainly: unit and oracle tests prove functional correctness of parts; e2e proves the system works; done requires the latter. Hard TDD stays supported where a design opts in.

Acceptance criteria:
1. Every document that defines done names e2e evidence as required and points at the E2E block, postures and Ga evidence.
2. No document implies oracle coverage alone means done.
3. No em dashes; example designs' BUILD plans updated to the new format.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Rewrite README, acceptance-gate, test-assurance-contract, SKILL, build-md-template and build-writer to name e2e proof as done. Evidence: Docs still headline hard TDD and oracle coverage; no e2e DoD text. Depends on gates that do not exist (p3pj, lioz, 4ah8 all unimplemented). Notes: Blockers are real: docs must describe shipped behavior. Do last in the epic.

## History
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-p3pj
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-lioz
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-4ah8

## Links
- Parent: [[MAC-9dai]]
- Blocked by: [[MAC-p3pj]], [[MAC-lioz]], [[MAC-4ah8]]

## Comments
