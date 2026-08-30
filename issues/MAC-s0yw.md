---
id: MAC-s0yw
title: "Zero-artifact checks B: edge gates (Ga, Gu, G5, Gs)"
status: in_progress
priority: 1
type: feature
created_at: 2026-08-30T18:54:10Z
created_by: ramirosalas
updated_at: 2026-08-30T18:54:50Z
content_hash: "sha256:4cdeec569245f5cabc7e444475a27ac09c8c9eb2b0258f34ce7dce8b7889ff38"
assignee: ramirosalas
---

## Description
# Zero-artifact engine checks, part B: the edge gates (Ga both-directions + default commit, Gu milestones, G5 direction vocab, Gs as_of)

## Context (from the 2026-08-30 determinism audit; line cites verified at f1dc685, may have drifted after 64bbd17/b235fc7/3b4fc6f landed; find by content)

User decision 2026-08-30: implement the eight zero-artifact checks. This story carries the four outside the G2/G3 cluster. A sibling story concurrently modifies internal/gates/gates.go, eventsource.go, and the pack-side event checks; do NOT touch gates.go or eventsource.go, and touch pack.go only in the checkBoundaryEvents switch (check 7). Shared docs (SKILL.md, README) will be edited by both stories in different gate sections; keep your hunks tight.

## Check 5: Ga acceptance evidence, both directions, plus default commit binding

Two parts, both in internal/gates/accept.go and the runner:
(a) checkDoDCoverage (near accept.go:389-415 at audit time) errors when a DoD-cited oracle id is missing from dod_ids, but a dod_ids entry that resolves to NO committed oracle id (typo, stale after regeneration) passes silently. Required: reverse resolution; every dod_ids entry must resolve to a committed oracle stable id, ERROR otherwise.
(b) Commit binding is only checked when --commit or MACHINERY_COMMIT is provided (near accept.go:163-165, suite wiring near suite.go:30); without it the binding degrades to a note. Required: when neither is provided and the design directory sits inside a git repository, default the commit to `git rev-parse HEAD` of that repository (resolve from the design dir, not the process cwd). Outside a git repo, or if git is unavailable, keep the current note-tier degradation unchanged. The explicit flag/env always wins. Make the provenance visible in the gate output (bound via flag/env vs derived from git). Mind test isolation: gate tests run inside the machinery repo itself, so fixture designs in temp dirs must not accidentally bind to the machinery repo's HEAD; resolve git from the design path and test both inside-a-repo and outside-a-repo behavior.

## Check 6: Gu milestone resolution (internal/gates/targetsurface.go)

acts[].milestone is only checked non-empty (near targetsurface.go:249-251). Required: when BUILD.md exists and declares milestones, each act's milestone must resolve to a declared milestone number/name; ERROR on no match. Reuse the existing plan-document parsing (the planDocuments/buildplan parse Gb uses) rather than re-parsing. When no BUILD.md exists yet, behavior unchanged (non-empty only): the ledger legitimately precedes the plan.

## Check 7: G5 direction vocabulary (internal/gates/pack.go)

checkBoundaryEvents' switch (near pack.go:452-468) has no default: a direction cell other than consumes/produces silently drops the row. Required: default case producing an ERROR naming the offending value and the accepted vocabulary. One-line semantics; do not restructure the switch beyond adding the default, since the sibling story may extract shared event helpers; if you hit a merge-sensitive area, keep the diff minimal.

## Check 8: Gs as_of shape (internal/gates/surface.go)

as_of is optional and accepts any non-empty string (near surface.go:129-132). Required: when present, it must parse as an ISO date (YYYY-MM-DD) or look like a VCS revision (hex of 7-40 chars, or a tag-like token; study what the surface-ledger docs say as_of holds before fixing the shape rule, docs/surface-ledger.md:33-38). Choose warn vs error and record the rationale; lean ERROR only if the docs pin the format, otherwise WARN with a message naming the expected shapes. Absent field stays legal.

## Hard constraints

- Fixture-based tests in the same pass, no mocks, following existing gate-test idioms (pass case + each failure class; for 5b, both git-present and git-absent environments).
- scripts/preflight.sh runs `machinery check` over the 8 bundled example designs and the golden corpus; new findings on examples mean the check caught a real defect: fix the EXAMPLE, never weaken the check. Note that 5b changes default-run output on every design inside a git repo (the examples are inside this repo), so golden corpus updates are likely: use `make golden-update`, review the diff line by line, record the reviewed summary.
- Update gate descriptions in lockstep (SKILL.md gate sections, README tables, docs/acceptance-gate.md for 5, docs/target-surfaces.md for 6, docs/surface-ledger.md for 8). SKILL.md frontmatter `version:` untouched.
- No em dashes, no emojis anywhere. Do not touch internal/hook/hook.go, internal/gates/gates.go, or eventsource.go.

## Acceptance criteria

1. All four checks implemented per spec with fixture tests for pass and every failure class; 5b tested for flag-wins, env-wins, git-derived, and no-git cases.
2. `go test ./... -count=1` green; `scripts/preflight.sh` fully green.
3. Example fixes and golden re-captures recorded and justified.
4. Docs in lockstep; no em dashes or emojis; SKILL.md version untouched.

## Proof required on delivery

Test and preflight summaries; per check, one induced-failure sample finding; the golden diff summary if re-captured.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-08-30T18:54:50Z status: open -> in_progress
- 2026-08-30T18:54:50Z claimed by ramirosalas

## Links


## Comments
