---
id: MAC-wwv6
title: "Four-state checker evidence (pass/fail/unknown/not_applicable) with fail-closed hook reduction"
status: open
priority: 3
type: feature
labels: [external-checkers, evidence, hooks, unapproved-scope, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:53Z
content_hash: "sha256:3effa2239034fdf9e688782b284e06e7d17105047f18032f1aa7eabfb362b72f"
---

## Description
Four-state external-checker evidence (pass, fail, unknown, not_applicable) with fail-closed hook reduction

Problem: evidence_schema 1.0 collapses checker results to pass/fail (internal/checker/checker.go:29 `SchemaVersion = "1.0"`, evidence.go:137 rejects any other version). An undecided element must be omitted, so it is indistinguishable from a coverage gap, and `not_applicable` can only be approximated by a static manifest residual, although applicability can vary by subject, facts and evaluation time and is not a waiver. docs/consistency-layer-proposal.md section 7 confirms this is still open Gk-side schema work.

Proposed fix (engine-neutral; names no checker implementation):
- New evidence schema version with top-level and per-element outcomes `pass`, `fail`, `unknown`, `not_applicable`; split evidence and projection schema-version constants so projection stays 1.0/2.0 unchanged.
- Keep reading evidence 1.0 during migration; committed and freshly reproduced evidence must use the same schema and preserve the complete outcome rows byte for byte.
- Every explicit outcome counts as coverage; only an absent claimed element is a coverage gap. Manifest residuals remain solely for declared, reasoned exclusions from the obligation set.
- Deterministic aggregation: any fail dominates; else any unknown yields unknown; else any applicable pass yields pass; all not_applicable yields not_applicable. A pass may not hide a failed or unknown row; unknown surfaces as REVIEW REQUIRED, never compliance.
- Preserve the four states in `machinery check`, evidence, findings, replay and `verify-checkers`. At a host hook or other binary policy boundary collapse fail-closed: pass and not_applicable permit, fail and unknown deny/block, with the detailed reason kept for the agent and audit trail.
- Stay checker-agnostic: schemas, Go types, CLI text, docs, tests and shipped fixtures use generic vocabulary and a synthetic checker; vendor, rule-language, corpus or engine identity stays in the private adapter and git-ignored local registry; committed evidence carries hashes and opaque attestations only.

Acceptance criteria:
1. Tests distinguish all four outcomes and distinguish unknown from a missing coverage row.
2. A fact- or time-dependent not_applicable is proven without a manifest edit.
3. Inconsistent aggregate/row combinations are rejected.
4. `verify-checkers` reproduces all outcome rows exactly; evidence 1.0 still verifies.
5. The Claude, Codex and OpenCode hook adapters emit only their existing allow/deny or block/no-output protocols, with fail-closed binary reduction proven for unknown.
6. No committed file names a specific checker engine; CHANGELOG records compatibility, migration and proof scope.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Introduce a new evidence schema with four outcomes, deterministic aggregation, 1.0 read compatibility, verify-checkers exact replay, and fail-closed collapse at the host hook boundary, with checker-agnostic docs and fixtures. Evidence: internal/checker/checker.go:29 SchemaVersion = "1.0" and internal/checker/evidence.go:137-138 reject any other evidence_schema; no unknown/not_applicable outcome anywhere in internal/checker or docs/external-checkers.md. CHANGELOG has no four-state work. Notes: Schema change plus hook adapter work across Claude, Codex and OpenCode; large but self-contained.

## History


## Links


## Comments
