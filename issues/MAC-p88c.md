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
updated_at: 2026-09-25T20:11:09Z
content_hash: "sha256:7ebd480a13e1a98078310ab96a10e5ec7198aa846605fcc271d1ed1f7ca94eaa"
blocked_by: [MAC-w44m]
was_blocked_by: [MAC-g7d4]
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
2026-09-25 Modelith decision input: 0.5.0 ships same-repo imports, shared vocabulary models and qualified scope.Name refs (rendered as links to per-model .md). That makes option (a) available upstream without filing anything, superseding (b) machinery-side assembly and (c) render-only sharding. Machinery work: multi-model discovery via a declared root/manifest (replace the exactly-one *.modelith.yaml rule at ~8 sites: checker.ModelPaths callers, projection_readers.go:290, gates.go:1741-1763, alloy.go:1242-1255, checkers.go:76, checker/project.go:83); resolve scope.Name to (file, Name) with a canonical context id (file stem) since scope is importer-local: entity:<context>.Name, bare entity:Name kept for the root model to avoid churn; global invariant id uniqueness across files; fact grammar disambiguation for dotted refs (vocab.X.y vs Entity.attr.member); Alloy/Gx/Gc across files; per-model renders in complete.go. Imports are non-recursive and reciprocity/invariants stop at the boundary, so machinery owns cross-model checks. Blocked on the pin bump.

## History
- 2026-09-25T19:39:58Z dep_added: blocked_by MAC-w44m
- 2026-09-25T20:11:08Z dep_added: blocked_by MAC-g7d4
- 2026-10-06T04:03:40Z dep_removed: was_blocked_by MAC-g7d4

## Links
- Parent: [[MAC-syos]]
- Blocked by: [[MAC-w44m]]
- Was blocked by: [[MAC-g7d4]]

## Comments
