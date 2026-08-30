---
id: MAC-s0yw
title: "Zero-artifact checks B: edge gates (Ga, Gu, G5, Gs)"
status: in_progress
priority: 1
type: feature
created_at: 2026-08-30T18:54:10Z
created_by: ramirosalas
updated_at: 2026-08-30T19:17:17Z
content_hash: "sha256:f8865f979d8add9172d046581f23d0a2c39b8de144b5ce49dc539ad3340a1dde"
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
DELIVERED at d9017d4 on branch story/MAC-s0yw-edge-gates (worktree scratchpad/wt-s0yw, base main 3b4fc6f).

PROOF
Commands run in the worktree, all from d9017d4:
- go build ./...            clean
- go vet ./...              clean
- gofmt -l cmd/ internal/   empty
- go test ./... -count=1    16/16 packages ok, 0 failures
- go test ./... -count=1 -v 1349 PASS, 0 FAIL, 1 SKIP (TestStableIDPrefixCollisionIsExtended in
  internal/oracle, pre-existing probabilistic collision hunt, "no collision found in budget";
  untouched by this story)
- scripts/preflight.sh      "preflight OK: local gates match ci.yml, safe to push"
  (whitespace, gofmt, vet, golangci-lint, go mod tidy, docs gate incl. em-dash scan, build,
  go test -race ./..., golden corpus + adversarial experiments, all 8 example check suites,
  go-crm impl module suite)
- .bin/machinery check on all 8 bundled examples: 0 blocking findings each, exit 0.

Acceptance criteria:
AC1 four checks + fixture tests for pass and every failure class - MET.
  5a  TestCheckAcceptanceDoDIDsMustResolve, TestCheckAcceptanceDoDIDTypoFailsBothWays,
      TestCheckAcceptanceDoDIDCoverage (forward, pre-existing).
  5b  TestResolveReviewCommit{flag or env wins over git / derived from the design's repository /
      no repository leaves it unresolved}, TestCheckAcceptanceDefaultsToGitHead,
      TestCheckAcceptanceDerivedCommitBlocksStaleEvidence,
      TestCheckAcceptanceExplicitCommitWinsInsideRepo,
      TestCheckAcceptanceOutsideRepoKeepsTheNote,
      TestCheckAcceptanceNoClosedMilestoneResolvesNoCommit,
      cmd: TestCheckDefaultsCommitToGitHeadOfTheDesignRepo (env cleared explicitly) and the
      pre-existing TestCheckCommitFlagAndEnvironmentReachGa (flag-wins, env-wins, wrong-commit).
      Isolation: git is resolved with `git -C <design>`, never process cwd; the derived-commit
      test asserts the resolved sha differs from gitHeadAt(".") (the machinery repo the tests run
      inside), and the no-repo test asserts gitHeadAt(design)=="" before running the gate.
  6   TestCheckTargetSurfacesMilestone{Resolves / DoesNotResolve / Spellings / PaddedNumber /
      WithoutPlan / FromManifestShard}, TestCheckTargetSurfacesEmptyMilestoneWithPlan.
  7   TestBoundaryEvent{UnknownDirection / EmptyDirection / DirectionVocabularyIsExact /
      LegalDirectionsOnly} (new file internal/gates/packdirection_test.go, kept separate from
      pack.go to minimize merge surface with the sibling story).
  8   TestCheckSurfaceAsOfShape (7 accepted shapes, prose warn, impossible-date warn,
      short-token lane) plus the updated TestCheckSurfaceAsOfAnchor.
AC2 go test ./... -count=1 green, preflight fully green - MET.
AC3 example fix and golden re-capture recorded and justified - MET (below).
AC4 docs in lockstep, no em dashes or emojis, SKILL.md version untouched - MET
  (SKILL.md metadata.version still "0.4.1"; the preflight docs gate greps the whole doc surface
  for em dashes and passed).

Induced-failure samples, one per check, from .bin/machinery on throwaway designs:
5a  ERROR acceptance/M0.yaml: dod_ids names 'CMD-999999', which no committed oracle declares
    (neither a test id nor a stable id of machines/*.oracle.md or the relational decision
    oracles); a typo, or an id a regeneration left behind, binds the evidence to no obligation
    at all
5b  ERROR acceptance/M0.yaml: commit '9f3c1a2b...12345' does not name the commit under review
    ('6a3238d93584d6f6372e8ef9b7510d5893ee8422'); ...
    (no --commit, no MACHINERY_COMMIT; the sha is the throwaway repo's HEAD, derived from the
    design path, and the checked: line reads "commit under review derived from git HEAD of the
    repository holding the design")
6   ERROR acts[0] (Deal.approve) names milestone 'M7', which the build plan does not declare;
    the declared milestones are M0 - Walking skeleton; M2 - Approvals slice
7   ERROR boundary event 'refunded' has direction 'emits'; the direction vocabulary is exactly
    consumes or produces, and a row with any other value is held to neither the consumption nor
    the emission rule
    (unit-level: end to end the child pack's content-hash check returns before the event checks,
    so a hand-edited events.md never reaches the switch; the reachable path is a pack the parent
    generated with a bad direction, which is what the unit test exercises)
8   warn as_of 'swept some time last quarter against the running prototype' is none of the shapes
    an anchor takes: an ISO date (YYYY-MM-DD), a VCS revision (7 to 40 hex characters), or a
    tag-like token; ... put the narrative in _comment

as_of tier decision (check 8): WARN, not ERROR.
  docs/surface-ledger.md:88 documents as_of as "the legacy commit or date the surface was
  enumerated against; non-empty string". The MEANING is pinned; the FORMAT is explicitly typed as
  a plain string, so the docs do not pin it and the story's rule ("lean ERROR only if the docs pin
  the format") lands on WARN. A design must not be blocked for a spelling the documentation never
  demanded. The warning still does the work the field exists for (P-F3: a stale ledger visible in
  every run), because what it catches is prose where an anchor belongs, which a reviewer cannot
  compare against the legacy system. Accepted shapes: ISO date (calendar-validated), 7 to 40 hex
  characters, or a single revision-like token wide enough for v1.4.2, release/2026-06,
  legacy@a1b2c3, svn:41207, HEAD@{2}. What none of them admits is whitespace.

Example fix (one):
  examples/go-crm/design/legacy/surface.yaml, as_of changed from
    "2026-07-22, enumerated from the committed legacy model and the migration.yaml inventory
     (retrofit; the prototype binary is not in this repo)"
  to
    "2026-07-22".
  The check caught a real defect: that value is a date plus a paragraph of provenance. No
  information was lost: the ledger's existing _comment already says "Retrofit ledger, authored
  2026-07-22: ... derived from the two committed legacy sources rather than from a running
  prototype", which is the same statement. The check was not weakened.
  Also updated the internal/gates/surface_test.go as_of fixture from "legacy@a1b2c3 (2026-06-30)"
  to "legacy@a1b2c3" so the "accepted and surfaced" case is a well-formed anchor, and tightened
  that test to assert zero warns as well as zero errors; the prose case it used to stand in for is
  now covered explicitly by TestCheckSurfaceAsOfShape/prose_warns_and_stays_surfaced, which also
  pins that a warned anchor is still printed verbatim on the checked line.

Golden re-capture (make golden-update), reviewed line by line: ONE line in ONE file.
  testdata/golden/check-go-crm/stdout.txt line 5, the Gs-surface checked: line:
  -  ... 18 surface items, as_of 2026-07-22, enumerated from the committed legacy model and the
     migration.yaml inventory (retrofit; the prototype binary is not in this repo)
  +  ... 18 surface items, as_of 2026-07-22
  That is the example fix above surfacing in the gate output. Nothing else in the corpus moved:
  git diff --stat testdata/ = 1 file changed, 1 insertion(+), 1 deletion(-).
  Note the story predicted 5b would move the golden output on every design inside a git repo. It
  did not, and correctly so: Ga only activates on a design with design/acceptance/ or a milestone
  marked "Status: closed", and no bundled example has either (verified: find examples -name
  acceptance -type d and grep -rn "Status: closed" examples/ are both empty, and no golden file
  mentions Ga-accept). The commit default therefore changed no example output.

Docs updated in lockstep:
  README.md gate table (Gs-surface, Gu-surfaces, G5-pack rows)
  skills/machinery/SKILL.md (Gu paragraph in Phase 2, Milestone acceptance section, the surface
    ledger section, the GATE 5 child-side list); metadata.version untouched
  skills/machinery/references/surface-ledger.md and references/target-surfaces.md (the skill-side
    mirrors of the two guides)
  docs/acceptance-gate.md (protocol step 4, both dod_ids directions, the commit-binding bullet)
  docs/target-surfaces.md (milestone row in the acts table, new rule 7, checked-line example)
  docs/surface-ledger.md (as_of root-key row, the gate section)
  commands/check.md (--commit is now optional locally)

File-ownership boundaries respected: internal/gates/gates.go, eventsource.go, and
internal/hook/hook.go untouched; internal/gates/pack.go touched only inside the
checkBoundaryEvents switch (4 added lines, a default case). Check 7's tests live in a new file
rather than in pack.go's vicinity, to keep the sibling story's merge clean.

RISK FOR REVIEW (not a defect in this change; reported per the Own All Errors rule).
  Ga's commit binding is an IDENTITY, not an ancestry: commitBinds requires an exact match or a
  >=7-character prefix relationship. That was already true of --commit, and CI passing
  $(git rev-parse HEAD) has the same property. Defaulting the commit to HEAD makes that property
  fire on every plain local run and on the stop-time hook (internal/hook passes RunOptions{}).
  The consequence: for a design with an accepted closed milestone, the commit that ADDS the
  evidence file already has a different sha than the commit the evidence names, so Ga goes red on
  the next commit and stays red until the closure is re-reviewed or --commit is passed. No bundled
  example exercises this (none has acceptance evidence), so nothing in this repo regresses, but a
  real adopting design would feel it immediately. This is implemented exactly as the story
  specifies; whether the derived lane (or both lanes) should accept an ANCESTOR of the commit
  under review, e.g. `git merge-base --is-ancestor`, is a semantics decision above my pay grade
  and I did not invent it. Flagging for Sr PM triage.

## History
- 2026-08-30T18:54:50Z status: open -> in_progress
- 2026-08-30T18:54:50Z claimed by ramirosalas

## Links


## Comments
