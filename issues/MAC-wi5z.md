---
id: MAC-wi5z
title: "Require active oracle discovery evidence — compliant successor to MAC-sh60"
status: in_progress
priority: 0
type: bug
parent: MAC-ui8a
created_at: 2026-09-06T16:33:51Z
created_by: ramirosalas
updated_at: 2026-09-06T23:26:04Z
content_hash: "sha256:91a2a677e3c97d7d07fb123cb1ae5907dcf533c964771e65586532af4fa97693"
labels: [hard-tdd, accepted]
follows: [MAC-sh60, MAC-hgz1]
blocks: [MAC-vx24, MAC-ou97]
was_blocked_by: [MAC-hgz1]
assignee: dev-MAC-wi5z
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape, and they need the replacement attempt to satisfy the hard-TDD audit without concealing MAC-sh60's failed history.

## Context (Embedded)
This P0 hard-TDD bug is the compliant successor to MAC-sh60 after its retained candidate a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940 failed `pvg story verify-tdd`. The raw audit at `/tmp/MAC-sh60-green-final.8U4Ho2/tdd-audit.txt` checked nine commits, skipped zero merges, and failed exactly 3d47b80ce59af1f521539e0b48e67be9ee4e9a71, 088ef8a6453df139aec6dcc9386f1021ab7ca6fc, and a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940 for existing-test edits without the required commit-subject marker. The old branch, retained worktree, commits, audit FAIL, notes, labels, claim, and evidence remain historical and must not be rewritten, accepted, merged, deleted, or used as implementation proof for this story.

The independently reviewed policy proposal is `/tmp/MAC-sh60-compliant-proposal-20260906-independent.md`, SHA256 b4dffe48f6f706460fe849a6d43c0024b25ad6cbda1567aec9960932602da8c2. It authorizes a genuinely new execution, not history repair or a private-tool exception. Test specifications and disclosed prior test provenance may inform new RED preparation; old GREEN production files, production diffs, patches, or implementation commits must not be copied or supplied to the fresh GREEN implementer.

Current read-only context at creation is main 497419ab4512fcff765cd5feb27aed4c67b5608d and epic/MAC-ui8a 2a73454d5f133a7b5fb4db0346232fd389810d28. Neither is the promised execution baseline. The actual baseline B is the then-current accepted epic SHA only after MAC-hgz1 is accepted and its shared Go CRM golden custody is released. Record B and accepted upstream SHAs before RED preparation. No story branch or worktree is created by this backlog operation.

## Ownership and serialization
This story depends on accepted MAC-hgz1 and follows MAC-sh60 historically. It must not add a reverse dependency to MAC-hgz1 or depend on MAC-lnu6, MAC-vx24, or MAC-ou97. MAC-vx24 and MAC-ou97 remain blocked by the retained MAC-sh60 hold and additionally depend on this successor until a canonical non-acceptance supersession policy is available; no old protection is removed by story creation.

Initial authorized candidate scope is exactly six paths and at most 1,000 changed LOC as an investigation bound, not permission to weaken the criteria or a guarantee that a fresh implementation will fit:
- internal/gates/oraclecov.go
- internal/gates/oraclecov_negative_test.go
- internal/gates/oraclecov_test.go
- cmd/machinery/oraclecov_negative_test.go
- internal/gates/oraclecov_scope_test.go
- testdata/golden/check-go-crm/stdout.txt

The stdout path transfers to this story only after MAC-hgz1 acceptance. No grant covers source examples, other golden files, MAC-hgz1 helpers/evidence, private pvg/nd tooling, installed assets, or any additional path. Preserve complete stdout/stderr/exit assertions. A necessary further path or semantic amendment must receive exact prospective independent PM review before writing it.

## Boundary Map
PRODUCES:
- internal/gates/oraclecov.go -> hardened static oracle discovery and conservative uncovered behavior.
- internal/gates/oraclecov_negative_test.go -> complete active-parser/literal positive controls and assessment-bypass negatives.
- internal/gates/oraclecov_test.go -> preserved stable-ID, orphan, missing-oracle, clause, conformance, formal-oracle, and boundary proof.
- cmd/machinery/oraclecov_negative_test.go -> actual standalone CLI integration over temporary real file trees.
- internal/gates/oraclecov_scope_test.go -> connected helper-return provenance, bounded-analysis, and ambiguity controls prepared in successor RED before production.
- testdata/golden/check-go-crm/stdout.txt -> independently reviewed migrated-baseline disclosure golden with complete prior stdout counts retained.
CONSUMES:
- MAC-hgz1: testdata/golden/check-go-crm/stdout.txt
  source: accepted migrated Go CRM v2 plan/current/historical classification, independently reviewed complete native stdout/stderr/exit inventory, current parser-backed subject, and released serialized custody; consume its exact accepted SHA and evidence, never its open proposal as fact.
- Existing Machinery source interface.
  spec: gates.CheckOracleCoverage(design, impl string) *Gate.
- MAC-sh60 retained audit record only.
  source: historical five-AC specification, approved test provenance, raw audit FAIL, and frozen hashes; no production source, patch, implementation commit, delivery, or acceptance is consumed.

### Story Acceptance Criteria
1. Comments, docstrings-only citations, unused filename/delimiter declarations, disabled Go build-tag files and commented Elixir IDs cannot establish oracle-row or wholesale table coverage.
2. Positive real literal-ID tests and genuine conformance table parsers remain discoverable across currently supported languages. Unsupported/ambiguous parser evidence is explicitly uncovered rather than silently credited; no filename-keyword heuristic can confer wholesale coverage.
3. Add full CheckOracleCoverage regression cases for every assessment bypass, zero-test directories, mixed legitimate/disabled tests, malformed oracle references and actual positive parser/literal fixtures.
4. Gate output accurately labels discovery versus actual test execution; this story never claims static references prove assertions ran. Later native execution protocol supplies execution evidence.
5. No loss of stable-ID boundary checks, orphan/missing-oracle diagnostics or clause coverage. Integration exercises actual CLI against temporary real file trees; no injected fake coverage result.

## Successor hard-TDD execution contract
1. After MAC-hgz1 acceptance, record B, the accepted upstream SHAs, a clean new candidate inventory, unchanged production hashes, and released Go CRM golden custody. No old GREEN source may be present in B or imported as the solution.
2. A separate RED author prepares tests honestly on B. Reuse the approved tests-only delta through 8f843256d30068816234d008a85239d662e8bb15 relative to 6cb2d974ea8aea211a5974f453cef2b5802bb11e with full provenance; do not cherry-pick 8f alone. Import the exact final 122-line supplemental test specification from a6ac10ebadc0e6b6a344d50aa10d0b810ce6b940 only as disclosed formerly-GREEN material newly prepared before this successor's production. Applicable commits carry `tdd-red` at creation and any separately authorized existing-test repair also carries `[test-edit-authorized]` contemporaneously.
3. Real RED on unchanged B must compile and reach intended acceptance assertions. Preserve the complete zero/mixed/disabled/malformed/positive, direct/assigned helper, uncalled/unused/constant/wrong-oracle/closure/receiver/path/return/depth/budget, stable-ID boundary, orphan/missing/clause/formal, native parser, and actual CLI/Go CRM matrices with named passing controls. Setup, import, infrastructure, unrelated Gv, missing-tool, changed-inventory, timeout, or skip failures are not RED. Cases already safely passing B are classified honestly; the combined suite must still demonstrate the unfixed semantic bug.
4. An independent RED reviewer replays and maps every criterion, reviews exact migrated golden bytes and accepted example inventory, checks provenance and unchanged B production, and freezes every successor test/fixture/config input through the ordinary RED gate. No past log alone approves new RED. If upstream semantics require a bounded change, hold and obtain exact prospective review, then rerun RED.
5. A fresh GREEN implementer receives this story, B, and the frozen successor RED/review only, never MAC-sh60 GREEN production files, production diffs, patches, or implementation commits. Implement independently, preserve frozen bytes, and return any necessary existing-test repair for prospective independent authorization and new RED proof before editing.
6. Same-SHA GREEN proof includes targeted oracle/conformance/coverage and CLI matrices, every frozen and supplemental case, TestCheckGreenSummaryLines, TestGoldenCheck/go-crm with complete migrated formal counts/output, full native internal/gates and cmd/machinery packages, actual unchanged Go CRM authz formal parsers, and relevant accepted MAC-uzxr native FSM suites/controls. Record exact commands, branch/SHA, raw native leaf names/counts, exits, hashes, AC mapping, coverage disposition, and learnings. Tooling proof-shape scans do not replace behavior.
7. Preserve finite bounds: targeted commands 120s and full packages 10m without coverage instrumentation under the recorded harness limit. Do not invent coverage or widen timeouts. Required infrastructure lanes and final scripts/preflight.sh remain owned by their existing stories/final gate; missing or skipped required integration is not success.
8. `pvg story verify-tdd` over the complete B..candidate ancestry must succeed with no omitted preparation commits, tip-only range, skipped-merge laundering, rewritten history, or exception. Normal canonical epic-base verification and the complete independent PM acceptance ladder also remain mandatory. Any verify-tdd failure rejects this successor. Only accepted/closed successor work may merge locally under the epic no-push rule.

Frozen historical reference hashes, to be preserved as provenance rather than treated as successor proof:
- internal/gates/oraclecov_negative_test.go: 42f1237819cfecc4a265fd5dfa4a90050d8f0f77f9aa0baf27b42995d8834e45
- internal/gates/oraclecov_test.go: dd68d778605397184630447bc09b712072fe604be77c496177d121690414b994
- cmd/machinery/oraclecov_negative_test.go: 6ea06399d8271691f7f40f30f067e0d3270184eaca4fc2548cc7a562c7775d08
- testdata/golden/check-go-crm/stdout.txt: d14ccc0e945d7bd21bbb681894682aa64240a8eb9c995a815f3bf4068b77b184

## Testing Requirements
- Hard TDD is explicitly authorized. Separate RED author, independent RED reviewer, and fresh GREEN implementer are mandatory; no self-approval or old implementation transfer.
- Unit plus integration tests: MANDATORY (no mocks). Real local Go/CLI/filesystem/parser execution, positive controls, exact causal failures, no stubs, no skip-if-missing, no env-gated dormant proof, and no fabricated or empty summary.
- Preserve active-test/comment/docstring/unused/disabled bypass controls and genuine literal/parser discovery across supported languages; ambiguous/unsupported evidence remains uncovered.
- Targeted commands are limited to 120s; full internal/gates and cmd/machinery packages are limited to 10m without coverage instrumentation. Heavy scripts/preflight.sh runs only at the final epic gate.
- No GitHub push/sync/remote mutation, installed binary/skill/plugin/agent replacement, dev-link, Docker/SSH/native execution during backlog preparation, or private workflow dependency in shipped Machinery.

## OUT OF SCOPE
- Repairing, rebasing, squashing, rewriting, accepting, merging, deleting, or relabeling MAC-sh60's failed history: retained permanently as audit evidence.
- Weakening golden counts, whole-CLI/formal assertions, parser obligations, supported-evidence behavior, or the five criteria to fit the old implementation: never.
- MAC-hgz1 evidence/helper ownership, MAC-lnu6 policy work, MAC-vx24 replay assurance, MAC-ou97 capstone, required infrastructure lanes, final preflight, main merge, installation, and publication: remain with their named owners.
- Private pvg/nd changes or a new runtime dependency from Machinery to Paivot coordination: prohibited.

## DIFF BUDGET
- Exactly six candidate paths, at most 1,000 changed LOC as a reviewed investigation limit. Any expansion requires a fresh exact PM amendment before writing; overrun never authorizes proof trimming.

## MANDATORY SKILLS
- developer for separate RED and fresh GREEN execution; codebase-memory for bounded source discovery; pm_acceptor for independent RED review and final acceptance; pvg and nd for canonical shared workflow only.

## Delivery Requirements
Use `pvg story deliver`, never close directly. Append B, all RED/review/GREEN SHAs, full-range audit output, frozen manifests, exact native evidence, and one proof item per unchanged criterion. Do not claim MAC-sh60 candidate logs as successor execution, and do not use pushing `pvg story merge`. Preserve Machinery as a standalone product with no pvg/nd dependency.

## nd_contract
status: new

### evidence
- Created as the user-approved compliant successor to failed-audit MAC-sh60; proposal SHA256 b4dffe48f6f706460fe849a6d43c0024b25ad6cbda1567aec9960932602da8c2.
- Baseline B, RED/review/GREEN SHAs, branch/worktree, and execution proof are intentionally unresolved until accepted MAC-hgz1; current main/epic SHAs are context only.

### proof
- [ ] AC #1: independently verified on fresh successor ancestry.
- [ ] AC #2: independently verified on fresh successor ancestry.
- [ ] AC #3: independently verified on fresh successor ancestry.
- [ ] AC #4: independently verified on fresh successor ancestry.
- [ ] AC #5: independently verified on fresh successor ancestry.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-06T16:34:00Z dep_added: blocked_by MAC-hgz1
- 2026-09-06T16:34:03Z dep_added: blocks MAC-vx24
- 2026-09-06T16:34:07Z dep_added: blocks MAC-ou97
- 2026-09-06T20:22:41Z dep_removed: was_blocked_by MAC-hgz1
- 2026-09-06T20:23:32Z status: open -> in_progress
- 2026-09-06T20:23:32Z auto-follows: linked to predecessor MAC-hgz1

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]
- Was blocked by: [[MAC-hgz1]]
- Follows: [[MAC-sh60]], [[MAC-hgz1]]

## Comments

### 2026-09-06T16:34:21Z ramirosalas
SUCCESSOR BOOKKEEPING 2026-09-06: User-approved compliant successor created from independently reviewed proposal SHA256 b4dffe48f6f706460fe849a6d43c0024b25ad6cbda1567aec9960932602da8c2. Canonical links: follows MAC-sh60; blocked by MAC-hgz1; additionally blocks MAC-vx24 and MAC-ou97. MAC-sh60 remains in_progress, claimed, hard-tdd/red-approved, and remains a blocker of both consumers because installed pvg 1.64.0/nd exposes no supported cancelled/superseded terminal transition or contract status; closed is documented as completed and auto-unblocks dependents. No acceptance, custom status, settings change, force close, old-edge removal, branch/worktree/source/test/native/private-tool/remote mutation occurred. Actual baseline B remains unresolved until MAC-hgz1 acceptance. All five AC remain pending; root must review this story before source dispatch.

### 2026-09-06T16:35:05Z ramirosalas
STRUCTURAL CHECKPOINT 2026-09-06: pvg nd dep cycles PASS (none); scoped RTM PASS (43 stories, 0 extracted/covered/uncovered). Scoped backlog lint scanned 58 issues and returned exit 1 with exactly two errors: produces-collision for internal/gates/oraclecov.go and internal/gates/oraclecov_negative_test.go, each simultaneously claimed by retained MAC-sh60 and successor MAC-wi5z without a dependency chain. Nine unrelated pre-existing vertical-slice review findings were also reported. The two errors are the expected consequence of unavailable canonical cancellation/supersession bookkeeping. No artificial successor-to-old completion dependency, old scope edit, terminal close, label/status change, edge removal, or queue bypass is authorized; MAC-wi5z remains open and blocked by MAC-hgz1, and must remain held/not dispatched pending root review plus a supported disposition policy. This lint result is structural evidence only, not AC/test/native proof.

### 2026-09-06T23:26:04Z ramirosalas
ACCEPTED 2026-09-06 (user decision 1a — production LOC overage accepted, proof not trimmed) — 6 commits 7dc13a4..0e0ea3d over baseline B df8dce0. Successor contract honored: no old GREEN production imported (verified not ancestors of B); tests-only delta 6cb2d97..8f84325 + disclosed 122-line spec from a6ac10e; audit markers correct at creation (3 tdd-red, 1 tdd-red [test-edit-authorized], fix, refactor, [test-edit-authorized] golden capture). RED: intended semantic failures incl. 26 bypass subtests; GREEN all targeted suites ok; golden one-line truthful reclassification (177->137 literal-covered) with same-SHA capture; G4 ERROR retained unmasked. 1,800 changed LOC total. Coordinator merged + targeted suites ok on epic. Record: .git/machinery-evidence-20260906.TEFZ7D/wi5z-record.md
