---
id: MAC-mbh7
title: "Gz-threat detection precision: audit noise on large designs"
status: open
priority: 2
type: feature
labels: [gates, threat, from-0.11.0]
created_at: 2026-10-06T04:04:25Z
created_by: ramirosalas
updated_at: 2026-10-06T04:04:25Z
content_hash: "sha256:0860c9951158c1ae55ae26b295d5e456acb767f490d7758a8e8dacd93c4fda21"
---

## Description
The 0.11.0 candidate detection is a broad word match. The consumer-corpus diff produced 15 to 107 Gz audit notes per design (231 over five designs); one design flagged an entity because its definition mentioned mfa. Audit notes block nothing, but this volume makes enforce-mode adoption costly. Improve precision without losing recall on real verifiers: weight names and actions over definition prose; aggregate per entity (one note per entity, not per word); allow not_security_relevant to exempt a word or pattern for a whole design; report a ranked list. Acceptance: the same five-design diff shows at least a 50% note reduction with every currently classified example subject (go-crm User) still detected; a fixture with a verifier entity whose only signal is an action name is still detected.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
