---
id: MAC-p88c
title: "Third-party formats: split workspace.dsl via !include; decide Modelith model and render sharding"
status: open
priority: 2
type: feature
labels: [context-budget, modelith, structurizr]
parent: MAC-syos
created_at: 2026-09-25T19:39:57Z
created_by: ramirosalas
updated_at: 2026-09-25T19:39:57Z
content_hash: "sha256:f7288e43388373b3f62c53de6809df166af0623ee29bdd9eb92b00f7861f6e54"
blocked_by: [MAC-w44m]
---

## Description
Third-party formats in the context-bounded artifacts epic.
- Structurizr DSL (workspace.dsl): !include is already allowed inside the retained workspace (skills/machinery/references/c4-standalone.md:144). Make the guidance and the build-writer split workspace.dsl by container/boundary, and have the size gate cover included files.
- Modelith (domain.modelith.yaml 920 KB and its render domain.modelith.md 1.05 MB on H2): Modelith owns the format. `modelith help` shows lint/render/deps, and nothing evident for one model split across files. Decide: (a) upstream multi-file model support in Modelith, (b) machinery-side assembly from per-entity YAML shards into the single file Modelith lints, or (c) per-entity render sharding only (the .md is what agents read most). Record the decision with alternatives, then implement it.

Acceptance criteria:
1. A decision issue/record for Modelith with the three options weighed; if upstream, an issue filed upstream and linked.
2. workspace.dsl guidance and generator split by boundary; the size gate covers included files.
3. The rendered domain Markdown is readable per entity through an index on H2.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-25T19:39:58Z dep_added: blocked_by MAC-w44m

## Links
- Parent: [[MAC-syos]]
- Blocked by: [[MAC-w44m]]

## Comments
