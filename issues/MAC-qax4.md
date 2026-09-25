---
id: MAC-qax4
title: "Docs: e2e proof is the definition of done; hard TDD is a rigor option"
status: open
priority: 1
type: task
labels: [e2e, docs, dod]
parent: MAC-9dai
created_at: 2026-09-25T19:53:32Z
created_by: ramirosalas
updated_at: 2026-09-25T19:53:32Z
content_hash: "sha256:9fcbb27c08d51929c1c83c2e86420285f1f11e6938af62ac1a897a7e6f8ef8b3"
blocked_by: [MAC-p3pj, MAC-lioz]
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


## History
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-p3pj
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-lioz

## Links
- Parent: [[MAC-9dai]]
- Blocked by: [[MAC-p3pj]], [[MAC-lioz]]

## Comments
