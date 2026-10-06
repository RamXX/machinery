---
id: MAC-r0ey
title: "Locked-suite gate: a LOCKED test file changed without an AMEND- tag blocks under --impl"
status: open
priority: 1
type: feature
labels: [gates, hard-tdd, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:53Z
content_hash: "sha256:ac6d49ee600802a64bdfed1a32280dc5de3c40f90ae81125a0e5e477ab302b7e"
---

## Description
Locked-suite gate: a LOCKED test file changed without an AMEND- tag is a blocking finding under --impl

Problem (H2): hard TDD says a LOCKED suite changes only under a recorded `AMEND-` tag. A lane's wip commit (89436e6b) changed one line in two LOCKED suites (a fixture tuple shape) and landed through a green wall and a green sixteen-gate check; a review found it nine days later. Locking lives in the suite's moduledoc and in the project's red ledger, and nothing in machinery reads either.

Evidence: no gate reads a locked inventory (no `locked`/`AMEND` handling in internal/gates; CHANGELOG 0.7.0 to 0.10.1 has none).

Proposed fix:
- A `locked:` inventory the design names (paths/globs, or a moduledoc/header marker the design declares).
- A gate under `--impl` with `--commit <anchor>`: a locked file whose bytes differ from the anchor commit's, with no `AMEND-` token in the commit messages of the diff range and no `DECISIONS.md` entry dated after the anchor naming the tag, is a blocking finding naming the file and the commits.

Acceptance criteria:
1. A fixture reproducing the H2 case (one-line change to a locked suite, no tag) fails with the file and offending commit named.
2. The same change with an `AMEND-<id>` token in a commit message in the range passes.
3. The same change with a dated `DECISIONS.md` entry after the anchor naming the tag passes.
4. Without a `locked:` inventory the gate is inactive and existing output is byte-identical.
5. A locked file that is deleted or renamed is also a finding unless tagged.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add the `locked:` inventory declaration and a gate under --impl --commit <anchor> that blocks a locked file changed vs the anchor without an AMEND- token in range commit messages or a dated DECISIONS.md entry; include deleted/renamed files and byte-identical output when no inventory exists. Evidence: No locked/AMEND handling exists: grep -rl AMEND over *.go and *.md (excluding vault/plans/tmp) returns nothing; internal/gates has no locked-inventory reader (internal/gates/threat.go:118 'locked' refers to negative tests named in the Gz threat ledger, not hard-TDD locked suites). CHANGELOG 0.7.0-0.11.0 has no such gate. Notes: Safety hole in hard-TDD enforcement (H2 evidence). Hard TDD is a rigor option, so P2 is defensible; kept P1 as filed.

## History


## Links


## Comments
