---
id: MAC-a89e
title: "Keep regeneration advice from accepting new debt"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, red-approved, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T23:57:24Z
content_hash: "sha256:76edf1633ef0ce8b4fa54e3704cb1dbfc315af13b58bf6c2119072b1c3613e42"
blocks: [MAC-gcrr, MAC-ou97]
follows: [MAC-olrx, MAC-p8ce]
assignee: dev-MAC-a89e
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
NEXT9: version-skew advice includes machinery baseline when ratchet exists. Baseline snapshots tolerated import offenders and can widen accepted architecture debt; ratchet has no version stamp. Routine regeneration must not silently authorize debt.
Same-story AC3 defect confirmed during RED preparation: explicit baseline on an already-baselined edge with a newly added offender proposes zero dependency rules, prints "the contract already covers every observed edge; nothing new to baseline", then rewrites ratchet.json to accept the new offender. Current help's "after review" refers only to pasted dependency rules. Help and successful output must explain ratchet/offender debt acceptance and review even when no rule is proposed. Explicit deliberate baseline remains supported.

## Ownership
Own exactly four files: internal/gates/gates.go (routine regeneration advice), cmd/machinery/baseline.go (narrow help and successful-output debt-review messaging), internal/gates/regeneration_safety_test.go (new focused and real CLI regression coverage), and internal/gates/gates_test.go (only the independently authorized TestVersionSkewNoteNamesEveryApplicableCommand expectation repair). Preserve that existing test's ratchet-present fixture and oracle/Alloy/verify-formal/pack expectations; replace only baseline expectation with explicit baseline absence. No other existing test/assertion changes are authorized. You are not alone in this codebase; preserve other edits and coordinate shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- internal/gates/gates.go -> hardened behavior and regression proof
- internal/gates/regeneration_safety_test.go -> hardened behavior and real CLI regeneration/baseline regression proof
- cmd/machinery/baseline.go -> help and successful output explaining deliberate ratchet/offender debt acceptance and required review
- internal/gates/gates_test.go -> narrowly authorized baseline-absence expectation in TestVersionSkewNoteNamesEveryApplicableCommand
CONSUMES:
- Existing Machinery source interfaces.
  spec: regenCommands(design string) []string; VersionSkewNote(design string, gs []*Gate) string
- Existing internal/gates/baseline.go (read-only behavior consumer).
  spec: BuildBaseline(design, impl, date string) (*BaselineReport, error)
  source: already-baselined edges are re-snapshotted using current offender files; zero Proposed rules does not mean ratchet debt is unchanged.
- Existing cmd/machinery/baseline.go.
  spec: newBaselineCmd() *cobra.Command
  source: baseline <design-dir> --impl <dir>; --date controls deterministic snapshot date; successful command publishes ratchet.json.

### Story Acceptance Criteria
1. Version-skew regeneration instructions never include machinery baseline or any debt-accepting mutation. Existing oracle/Alloy/formal/pack regeneration remains accurate and deterministic.
2. A design with ratchet and newly introduced offender continues failing architecture checks after following all routine regeneration instructions; regenerated stamps cannot accept the offender.
3. Explicit machinery baseline remains a supported deliberate user-invoked debt-acceptance operation. Before invocation, baseline --help must explain that rerunning rewrites ratchet.json and may accept newly added offender files, and tell users to review ratchet/offender changes before adopting them. After successful publication, output must convey the same debt-change review guidance even when zero dependency rules were proposed. The zero-rule message must describe only the absence of new rule proposals and must not claim that no debt was accepted or nothing changed. Real CLI regression: start with an already-baselined edge, add a new offender file, prove the architecture check fails, deliberately run baseline successfully, verify the new offender is now recorded/accepted with zero new rule proposals, and assert help plus successful output require ratchet/offender-change review. This deliberate acceptance is allowed; routine regeneration must never perform it. No automatic migration, new confirmation protocol, version stamp or prohibition of explicit baseline.
4. Positive no-debt/version-skew fixture yields correct required generator commands; negative existing-ratchet/new-offender regression demonstrates failure before and after advised regeneration.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: go test -count=1 ./internal/gates -run 'VersionSkew|Regen|Ratchet'; real built CLI integration on an isolated ratchet design with a genuine new boundary violation. New named tests should match the scoped selection or record their exact supplemental command.
- AC3 CLI proof is required alongside the routine-regeneration negative: capture baseline --help; then exercise the already-baselined/zero-proposed-rule/new-offender success path and assert actual ratchet expansion plus explicit debt-review guidance. A vague "after review" referring only to dependency rules is insufficient. Pin the semantic requirement, not arbitrary exact prose. Preserve actual explicit-baseline functionality and passing controls; no mocks/stubs/skips or external service dependency is needed.
- Existing PM comment at 2026-09-05T21:14:38Z authorizes only the named gates_test.go expectation repair. That repair commit subject must include tdd-red and [test-edit-authorized]. This development coordination convention is never a product/runtime requirement. Freeze revised tests/fixtures after independent RED review; GREEN may not weaken them.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.
- Baseline algorithm, ratchet schema/date semantics and new approval/confirmation mechanisms: not needed for the identified advice/messaging defect; preserve existing deliberate baseline behavior.
- Machinery product must remain standalone; no Paivot/pvg/nd dependency, workflow labels or commit conventions in shipped behavior.

## DIFF BUDGET
- Exactly four owned files, under 300 total changed LOC. The file-count increase from 2-3 is justified by one bounded CLI help/output change and the independently authorized existing-test expectation repair alongside gates.go/new regression coverage. Keep the original tight LOC ceiling; report concrete real-CLI fixture needs before any material overrun instead of weakening proof.

## MANDATORY SKILLS
- developer for implementation; codebase-memory for discovery; pm_acceptor for independent acceptance.

## Delivery Requirements
Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands and outputs, test inventory, independent proof per AC, and any residual limits to shared nd. Do not use pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05; source signatures verified at assessment base 497419ab4512fcff765cd5feb27aed4c67b5608d.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.
RED-DISPUTE: paused before delivery at dispatcher direction. RED commit 9b85ef6 adds internal/gates/regeneration_safety_test.go only. Authorized superseded assertion repair commit f482c7683f5005e40722123e3a382e01b15219d2 changes only TestVersionSkewNoteNamesEveryApplicableCommand as permitted by independent PM comment (tdd-red and [test-edit-authorized] subject). Production unchanged; worktree clean.

PROOF (provisional, NOT RED approval): go test -count=1 -timeout=5m ./internal/gates -run "VersionSkewRegenerationOnlySafeGenerators|RegenRatchetRealCLI" at 9b85ef6 completed in 5.046s: 10 leaves, 7 pass, 3 fail, 0 skip. Unit ratchet-only and all-with-ratchet fail because advice includes baseline; five single-family controls pass. CLI no-debt-version-skew-control passes. CLI new-offender-survives-all-advice fails on exactly unsafe advice, changed ratchet bytes, and unexpected green G4 after executing every printed command: alpha/a.go snapshot expands to alpha/a.go plus alpha/b.go and G4 returns zero findings. CLI explicit-baseline-reviews-debt-change superficially passes, but its review substring matches the temporary subtest pathname; that AC3 assertion is invalid evidence and requires bounded repair authorization. Proposed repair checks actual prose guidance and adds help-before-invocation coverage when canonical scope is repaired. No compile/setup/runtime prerequisite failures. go test -count=1 -timeout=5m ./internal/gates -run "^TestVersionSkewNoteNamesEveryApplicableCommand$" at f482c7683f5005e40722123e3a382e01b15219d2: 0 pass, 1 fail, 0 skip, intended baseline-absence assertion. Full requested target with JSON inventory and coverage remains pending after repair.

pvg verify internal/gates/regeneration_safety_test.go --format text: PASS, 1 file, 0 issues. pvg verify internal/gates/gates_test.go internal/gates/regeneration_safety_test.go --format text: FAILED, 2 files, 1 stub at gates_test.go:125 return empty string; this is outside the narrowly authorized amendment and must be reviewed rather than silently edited.

AC mapping: AC1 precise generator unit fixtures and authorized existing expectation fail behaviorally; AC2 real CLI byte and G4 checks fail behaviorally with initial G4 green and genuine new-offender rejection before regeneration; AC3 explicit baseline functionality works but guidance assertion is disputed; AC4 passing real no-debt control plus negative CLI reproduced, complete inventory pending. Coverage percentage not yet measured; no delivery claim. Codebase-memory Verify inherited parent exact symbols/coverage; index_status confirmed ready, worktree excluded so exact source fallback used.

LEARNINGS:
- Output assertions can accidentally match temporary test directory names; test prose rather than unrestricted substring presence.
- Real baseline reruns expand an already-baselined edge even when output says nothing new to baseline.
- Executing all printed advice with placeholder substitution demonstrates debt mutation directly without mocks or external services.
## PM GREEN Final Review — MAC-a89e — 2026-09-05
ACCEPTED: candidate 4f339474e52ee9e788d0d5d071105fba385188b2 satisfies all four ACs after the authorized narrow documentation rework. Read current canonical description, complete delivered evidence, frozen RED review, both test-edit authorizations and previous GREEN rejection; reviewed committed source independently in detached /tmp/MAC-a89e-pm-final.HlPJ3V/checkout.

Rework and scope:
- git show 4f339474 confirms only docs/brownfield-team-guide.md former lines 451-452 changed: legacy YYYY-MM remains supported, ages from month start, retains its date without explicit date or SOURCE_DATE_EPOCH, and baseline remains deliberate reviewed debt acceptance. This exactly resolves the previous sole DOCS_STALE gap. No production/frozen-test change in rework.
- Full candidate range from merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890: 5 files, +267/-12 = 279 changed LOC. Fifth document and both narrow paragraphs explicitly authorized by dispatcher. Production changes only safe advice and baseline help/success prose; algorithm, schema, dates and confirmation behavior preserved.
- Reviewed baseline/restamp documentation occurrences and both updated paragraphs; no further stale routine-baseline recommendation found.

Hard-TDD and static:
- cd /tmp/MAC-a89e-pm-final.HlPJ3V/checkout && pvg story verify-tdd --base epic/MAC-ui8a: PASS, 6 commits, 0 merges skipped, no unauthorized test edits.
- git diff --exit-code e95be63019d89b93ae527f27ae1ce43326def038 HEAD -- internal/gates/gates_test.go internal/gates/regeneration_safety_test.go: PASS; git diff --check epic/MAC-ui8a: PASS.
- Frozen SHA256 gates_test.go e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; regeneration_safety_test.go 2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3, identical to approved RED.
- pvg verify internal/gates/gates.go cmd/machinery/baseline.go internal/gates/regeneration_safety_test.go internal/gates/gates_test.go docs/brownfield-team-guide.md --format text: reports exactly 4 return-empty findings in 4 scanned source files, unchanged output hash. Independently read each full function: gates.go:76 implements documented no-skew silence after collection; :219 returns empty only on read failure and actual bytes otherwise; :230 records non-missing read error then returns the error sentinel, actual bytes on success; gates_test.go:125 omits interface table only for no concrete rows and otherwise renders the full table. These are populated implementations, not stubs; precise prior dispositions confirmed, no blanket waiver or read-error-policy claim.
- Go signatures typed; no new API/config/cross-cutting concerns. No mocks, service dependency, skipped prerequisites, Paivot runtime coupling or frozen-test changes.

Independent test commands, both exit 0:
cd /tmp/MAC-a89e-pm-final.HlPJ3V/checkout && set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-pm-final.HlPJ3V/gates-coverage.out | tee /tmp/MAC-a89e-pm-final.HlPJ3V/gates-tests.json | tail -n 5
cd /tmp/MAC-a89e-pm-final.HlPJ3V/checkout && set -o pipefail && go test -count=1 -timeout=5m ./cmd/machinery -run '^(TestBaselineSourceDateEpochStampIsFullDate|TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting|TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree)$/^baseline$' -json -coverprofile=/tmp/MAC-a89e-pm-final.HlPJ3V/cli-coverage.out | tee /tmp/MAC-a89e-pm-final.HlPJ3V/cli-tests.json | tail -n 5
Native JSON leaf inventory: gates 37 PASS, 0 FAIL, 0 SKIP (5.966s); cmd 3 PASS, 0 FAIL, 0 SKIP (0.864s). Total 40/40. Names match delivered inventory modulo temp nonce. No compile/setup/deadline/test warnings or unexpected runtime failures. Existing diagnostics test exercised baseline subprocess and did not take symlink skip.
Coverage: gates total 9.9%, gates.go VersionSkewNote 93.8%, regenCommands 100%; cmd total 2.2%, newBaselineCmd 75%, resolveBaselineDate 84.2%. Both scoped coverage profiles match author hashes; separately built CLI subprocess is excluded from package coverage.
Independent retained logs /tmp/MAC-a89e-pm-final.HlPJ3V:
- gates-tests.json SHA256 4fce0fdee095587c46cab3ccb1030208570d4006fead6c0779b76328e585dbeb
- cli-tests.json SHA256 5051b37a77de95a6941de4deace5cce7c7fcc8833f772868cfe3da9b7f058d39
- gates-coverage.out SHA256 f1b45d80d3aeecd7157adbeda1a85594d452cf99a2905b1355715c40f3a6fbbb
- cli-coverage.out SHA256 8f85fe7958ee02927a94af73448d5b72dbbe90e699aeb33f05047575681b325e
- verify.txt SHA256 841ae4d70019ef05407b2c096de729ef9a86c6a39226ab76fee41a76e67b51c2
- tdd.txt native PASS output.
Latest author gates/CLI JSON and verify.txt hashes independently match shared proof.

AC review:
1. PASS: regenCommands removes only baseline; exact deterministic ordered family inventories, ratchet-only, individual and combined controls pass; existing generated-TLA selection preserved.
2. PASS: real built CLI starts with accepted alpha/a.go and green G4, adds alpha/b.go and requires exact G4 growth failure, executes every advised generator, verifies fresh oracle stamp, unchanged ratchet bytes and same offender rejection.
3. PASS: actual help before invocation and successful publication explain ratchet rewrite/possible new offender acceptance/review before adoption even at zero proposals. Zero-rule prose describes only proposal absence. Deliberate CLI baseline accepts both offenders with 0 new rules, unstamped ratchet and later G4 green. Five sensitivity controls pass. Both scoped docs paragraphs now require deliberate debt review; legacy-date statement matches independently passing date-reuse test.
4. PASS: real no-debt oracle regeneration control resolves skew, keeps g3/g4 green and creates no ratchet; negative debt growth remains blocked after routine advice.
User intent met: routine regeneration preserves architectural enforcement while explicit reviewed debt acceptance stays available. Prior discovered documentation issue resolved in this same story; no new unrelated bug found. LEARNINGS present in author delivery.
Limits: focused native suite only; executable regeneration fixture carries oracle, other generator families have command-selection controls, no Alloy/TLA runtime claim. Full preflight/final integration belong to root epic gate. No remote action, merge, installed artifact change or main move.

## nd_contract
status: accepted

### evidence
- Independently reviewed and tested 4f339474e52ee9e788d0d5d071105fba385188b2: 40 PASS/0 FAIL/0 SKIP; hard-TDD and frozen hashes PASS.
- Sole prior documentation rejection resolved by exact authorized rework; all four scanner findings independently confirmed nonstubs.

### proof
- [x] AC #1: safe deterministic complete regeneration advice.
- [x] AC #2: actual new offender remains rejected after every routine command.
- [x] AC #3: explicit baseline supported with truthful help/output/docs debt-review guidance and date semantics.
- [x] AC #4: real positive no-debt and negative debt-growth paths executed.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence (DELIVERED GREEN REWORK — MAC-a89e)

PROOF:

### Rework resolution and explicit authorization
- Prior GREEN de9745067382c5de25f05d69ea05c2808eb0b2e7 was independently reviewed with 40/40 PASS and rejected solely for DOCS_STALE: the legacy ratchet date bullet still recommended baseline merely to restamp. No production or frozen-test defect was found.
- PM rejection at 2026-09-05T23:46:44Z records dispatcher authorization for only that additional bullet in the already-scoped docs/brownfield-team-guide.md. Rework commit 4f339474e52ee9e788d0d5d071105fba385188b2 changes only those two former lines to five lines, preserving every other file and prior change.
- Corrected text: Legacy ratchet.json snapshots with YYYY-MM remain supported and age from the first of that month. Without an explicit date or SOURCE_DATE_EPOCH, a baseline rerun retains that date; no routine restamping is needed. Baseline remains a deliberate debt-acceptance operation: review ratchet and offender changes before adopting its result.
- This fixes the documented migration recommendation without introducing date behavior, new tests, confirmation mechanisms or additional source ownership. Existing date-reuse test passed again at the latest SHA.

### Commit and scope
- Frozen approved RED: e95be63019d89b93ae527f27ae1ce43326def038.
- GREEN production: bcba7900a042286b45ff46ae08bc04516ce34923.
- Final tested candidate: 4f339474e52ee9e788d0d5d071105fba385188b2 on story/MAC-a89e, worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e.
- Scope expansion explicitly authorized by dispatcher during GREEN: docs/brownfield-team-guide.md only the generated-file merge paragraph around 259–265; separate ratchet conflicts as explicit debt-policy review, preserve deliberate baseline, require review of accepted offender changes, no new merge algorithm/confirmation workflow. Five files, 279 total changed LOC against local merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890; original four production/test paths plus this one document. GREEN adds 21/removes 11 lines across three non-test files.
- Both frozen tests remain byte-identical to approved RED (git diff e95be63 HEAD -- test paths exits 0). SHA256 gates_test.go=e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; regeneration_safety_test.go=2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3. No authored tests or fixtures edited during GREEN.

### CI/Test Results
Commands run:
All shell commands use explicit cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e.
- set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-green-rework.roS7lB/gates-coverage.out | tee /tmp/MAC-a89e-green-rework.roS7lB/gates-tests.json
- set -o pipefail && go test -count=1 -timeout=5m ./cmd/machinery -run '^(TestBaselineSourceDateEpochStampIsFullDate|TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting|TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree)$/^baseline$' -json -coverprofile=/tmp/MAC-a89e-green-rework.roS7lB/cli-coverage.out | tee /tmp/MAC-a89e-green-rework.roS7lB/cli-tests.json
- go tool cover -func=/tmp/MAC-a89e-green-rework.roS7lB/gates-coverage.out
- go tool cover -func=/tmp/MAC-a89e-green-rework.roS7lB/cli-coverage.out
- pvg verify internal/gates/gates.go cmd/machinery/baseline.go internal/gates/regeneration_safety_test.go internal/gates/gates_test.go docs/brownfield-team-guide.md --format text
- pvg story verify-tdd --base epic/MAC-ui8a
- git diff --check epic/MAC-ui8a
- git diff --exit-code e95be63019d89b93ae527f27ae1ce43326def038 HEAD -- internal/gates/gates_test.go internal/gates/regeneration_safety_test.go
- shasum -a 256 internal/gates/gates_test.go internal/gates/regeneration_safety_test.go

Summary:
- Required gates: 37 native leaves PASS, 0 FAIL, 0 SKIP; package 4.899s (native package event 4.900s), exit 0.
- Downstream baseline date, deterministic-date sources, actual subprocess baseline diagnostics: 3 native leaves PASS, 0 FAIL, 0 SKIP; package 1.435s (native package event 1.439s), exit 0.
- Total 40/40 selected leaves PASS. No compiler/setup/runtime deadline failures or test warnings.
- Hard-TDD PASS: local merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890..HEAD, 6 commits checked, 0 merges skipped, no unauthorized test edits. git diff --check PASS.
- Coverage: internal/gates scoped total 9.9%, gates.VersionSkewNote 93.8%, regenCommands 100%; cmd/machinery scoped total 2.2%, newBaselineCmd 75.0%, resolveBaselineDate 84.2%. These are scoped package figures; independently built real CLI subprocess is not included in internal/gates coverage. Downstream private-path test intentionally selects baseline only.
- Test integration runs actual freshly built CLI and native filesystem unconditionally; no mocks/stubs/services/env-based omission. The existing downstream fixture has a symlink-platform fallback but this run created symlinks successfully and reports zero skips.

### Retained raw evidence
Directory /tmp/MAC-a89e-green-rework.roS7lB:
- gates-tests.json SHA256 11dbcc06a3365fca64c726f6bf450755e6ec8d02216e9d6bf594230b932a083b
- cli-tests.json SHA256 94f7c9e37a4e21228db98238ebc138b7e391bab45383ed64673ee2f7a7cd7889
- inventory.txt SHA256 61286daa417fba0978ca70a0408a4d21d6da06953b8e67e71dc03d8c49bb366c
- gates-coverage.out SHA256 f1b45d80d3aeecd7157adbeda1a85594d452cf99a2905b1355715c40f3a6fbbb
- cli-coverage.out SHA256 8f85fe7958ee02927a94af73448d5b72dbbe90e699aeb33f05047575681b325e
- verify.txt SHA256 841ae4d70019ef05407b2c096de729ef9a86c6a39226ab76fee41a76e67b51c2
- tdd.txt contains native hard-TDD PASS output.

### pvg verify
Native output: VERIFY: FAILED (4 files scanned, 4 issues), all [stub] return empty string. Documentation is not scanned by this source scanner.
- internal/gates/gates.go:76 is VersionSkewNote's documented no-skew empty result. Full implementation collects/sorts skew and emits applicable advice. Existing tests TestOracleMissingStampIsFreshAndSilent and TestOracleCurrentStampIsSilent assert empty result; real no-debt regeneration control asserts skew disappears.
- internal/gates/gates.go:219 is readDesignOrEmpty's conditional read-failure empty sentinel; success returns actual file text.
- internal/gates/gates.go:230 is readDesignFileOrErr's conditional error sentinel; non-missing errors append an explicit gate error, success returns actual file text.
- internal/gates/gates_test.go:125 is coveringInterfaceTable's conditional no-concrete-allow-rows result; otherwise full markdown table is rendered. Independently reviewed earlier as exact false positive.
All four implementations/branches are unchanged from approved RED; none is a placeholder. Dispatcher independently inspected all three production paths and confirmed populated, unchanged conditional-return functions rather than empty implementations; this is a specific scanner false-positive disposition, not a claim that every caller's read-error policy is safe. PM GREEN review at 2026-09-05T23:46:23Z independently confirmed all four findings are populated implementations, not stubs; these exact paths and scanner output remain unchanged during rework. No scanner changes or test-helper edits.

### AC Verification
| AC | Requirement and evidence | Status |
|---|---|---|
| 1 | gates.go regenCommands removes only baseline from the deterministic explicit generator list. Frozen VersionSkewRegenerationOnlySafeGenerators tests all individual oracle/Alloy/semantics/composition/pack families and all-with-ratchet ordered output; ratchet-only emits no command. Existing ratchet-present TestVersionSkewNoteNamesEveryApplicableCommand and generated-TLA selection remain intact. | PASS |
| 2 | RealCLI/new-offender-survives-all-advice starts with actual CLI ratchet alpha/a.go and green G4, adds real alpha/b.go import and observes exact G4 growth failure, executes EVERY advised generator, proves current oracle stamp, compares ratchet bytes unchanged and requires same offender failure afterward. | PASS |
| 3 | CLI help before invocation explains rerunning rewrites ratchet.json and may accept newly added offender files, tells users to review ratchet changes before adoption; identical substantive guidance prints only after successful publication, regardless of proposal count. Zero-rule line now says only no new baseline dependency rules proposed. RealCLI/explicit-baseline-reviews-debt-change begins already baselined, rejects alpha/b.go, deliberately invokes baseline, proves 0 need a baseline rule, two recorded offenders, unstamped ratchet and G4 green. All five semantic-guidance controls pass, including pathname bait rejection. Baseline algorithm/schema/date/confirmation semantics preserved; downstream date tests pass. Authorized legacy-date bullet now states supported month precision, month-start aging and default date retention without routine restamping. | PASS |
| 4 | RealCLI/no-debt-version-skew-control obtains correct oracle advice, executes it, proves fresh current stamp/no skew and green g3,g4 without creating ratchet; negative AC2 still fails after all routine advice. Advice family controls execute deterministically. | PASS |

### Exact native leaf inventory
PASS TestG4RatchetSnapshotNote
PASS TestG4BaselineWithoutRatchetFails
PASS TestG4RatchetGreenAtSnapshot
PASS TestG4RatchetGrowthFails
PASS TestG4RatchetShrinkAndStaleEdgesNote
PASS TestRatchetRoundTrip
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/unknown_root
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edge_name
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_number
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edges_array
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_value_string
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_entry_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/trailing_value
PASS TestWriteRatchetRejectsSymlinkTarget
PASS TestVersionSkewNoteNamesEveryApplicableCommand
PASS TestVersionSkewNoteFormalCommandFollowsGeneratedTLA
PASS TestRatchetSnapshotNoteBothFormatsIsClockIndependent
PASS TestVersionSkewRegenerationOnlySafeGenerators/ratchet-only
PASS TestVersionSkewRegenerationOnlySafeGenerators/oracle
PASS TestVersionSkewRegenerationOnlySafeGenerators/alloy
PASS TestVersionSkewRegenerationOnlySafeGenerators/semantics
PASS TestVersionSkewRegenerationOnlySafeGenerators/composition
PASS TestVersionSkewRegenerationOnlySafeGenerators/pack
PASS TestVersionSkewRegenerationOnlySafeGenerators/all-with-ratchet
PASS TestRegenBaselineGuidanceSensitivity/genuine
PASS TestRegenBaselineGuidanceSensitivity/equivalent-wrapped
PASS TestRegenBaselineGuidanceSensitivity/path-bait
PASS TestRegenBaselineGuidanceSensitivity/old-rule-review-help
PASS TestRegenBaselineGuidanceSensitivity/unrelated-review
PASS TestRegenRatchetRealCLI/no-debt-version-skew-control
PASS TestRegenRatchetRealCLI/new-offender-survives-all-advice
PASS TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change
PASS TestBaselineSourceDateEpochStampIsFullDate
PASS TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting
PASS TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree/baseline/--impl_/var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/machinery-diagnostic-user-input-293403372/TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree1739941386/003_--date_2026-01-01_/var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/machinery-diagnostic-user-input-293403372/TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree1739941386/001


### Wiring, discovery, and residual limits
- Existing machinery check -> gates.VersionSkewNote -> regenCommands remains wired; existing baseline Cobra command renders help and prints guidance after PublishExpectedRooted succeeds. No new interfaces/config/product dependencies.
- Used developer and codebase-memory skills; inherited parent Verify structural evidence then independently checked project readiness and exact coverage. Graph generation 2026-09-05T23:08:53Z, no recorded gaps for gates.go, gates_test.go, baseline algorithm, CLI baseline/check/diagnostic paths. New regression test is absent from root graph; .claude worktree is excluded, so reviewed exact worktree source and tests directly. No completeness claim.
- Documentation freshness scan found the same unsafe operation in generated-file conflict advice; authorized narrow paragraph correction now separates explicit ratchet debt review.
- No full preflight or unrelated runtime suite run, as explicitly constrained; final epic gate owns those checks. Real executable fixture has oracle only; other generator families are precise command-selection tests, not claims of Alloy solver/TLA execution.
- No remote fetch/pull/push/sync, branch integration, installed binary/skills/plugins/agents changes, Docker changes, Paivot runtime dependency, or baseline algorithm/schema/date modifications. Worktree retained, committed and clean.

LEARNINGS:
- Zero new dependency-rule proposals can still mean expanded accepted offender debt; help and successful output must distinguish them.
- Explicit generators form the safe regeneration contract. A ratchet is debt policy rather than a version-stamped derivative.
- Frozen RED history exposed pathname-bait guidance assertions; approved path-stripping semantic controls now reject accidental matches without weakening real explicit-baseline proof.
- Source scanners flag valid conditional empty sentinels; retain raw findings and secure specific independent disposition instead of altering working semantics.
- Initial documentation review corrected generated-file merge advice but missed the later legacy-date restamping instruction. PM found that omission; the authorized rework removes it and states existing date retention accurately. Future freshness review must cover every occurrence of the unsafe operation, including migration/date guidance.

## nd_contract
status: in_progress

### evidence
- Candidate 4f339474e52ee9e788d0d5d071105fba385188b2, 40 PASS/0 FAIL/0 SKIP, scoped coverage and raw evidence above.
- Frozen RED unchanged; hard-TDD PASS; five-file/279-LOC scope includes explicit dispatcher authorization.

### proof
- [x] AC #1: safe deterministic complete regeneration advice.
- [x] AC #2: real offender remains G4 failure after every advised generator.
- [x] AC #3: explicit baseline supported with truthful pre-invocation and successful debt-review guidance even at zero rules.
- [x] AC #4: real no-debt positive and debt-growth negative both exercised.


## nd_contract
status: rejected

### evidence
- PM rejection applied via pvg story reject on 2026-09-05.

### proof
- [ ] Story requires another developer delivery before it can be accepted.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence (GREEN — MAC-a89e)

PROOF:

### Commit and scope
- Frozen approved RED: e95be63019d89b93ae527f27ae1ce43326def038.
- GREEN production: bcba7900a042286b45ff46ae08bc04516ce34923.
- Final tested candidate: de9745067382c5de25f05d69ea05c2808eb0b2e7 on story/MAC-a89e, worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e.
- Scope expansion explicitly authorized by dispatcher during GREEN: docs/brownfield-team-guide.md only the generated-file merge paragraph around 259–265; separate ratchet conflicts as explicit debt-policy review, preserve deliberate baseline, require review of accepted offender changes, no new merge algorithm/confirmation workflow. Five files, 272 total changed LOC against local merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890; original four production/test paths plus this one document. GREEN adds 16/removes 9 lines across three non-test files.
- Both frozen tests remain byte-identical to approved RED (git diff e95be63 HEAD -- test paths exits 0). SHA256 gates_test.go=e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; regeneration_safety_test.go=2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3. No authored tests or fixtures edited during GREEN.

### CI/Test Results
Commands run:
All shell commands use explicit cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e.
- set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-green.xtIsDr/gates-coverage.out | tee /tmp/MAC-a89e-green.xtIsDr/gates-tests.json
- set -o pipefail && go test -count=1 -timeout=5m ./cmd/machinery -run '^(TestBaselineSourceDateEpochStampIsFullDate|TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting|TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree)$/^baseline$' -json -coverprofile=/tmp/MAC-a89e-green.xtIsDr/cli-coverage.out | tee /tmp/MAC-a89e-green.xtIsDr/cli-tests.json
- go tool cover -func=/tmp/MAC-a89e-green.xtIsDr/gates-coverage.out
- go tool cover -func=/tmp/MAC-a89e-green.xtIsDr/cli-coverage.out
- pvg verify internal/gates/gates.go cmd/machinery/baseline.go internal/gates/regeneration_safety_test.go internal/gates/gates_test.go docs/brownfield-team-guide.md --format text
- pvg story verify-tdd --base epic/MAC-ui8a
- git diff --check epic/MAC-ui8a
- git diff --exit-code e95be63019d89b93ae527f27ae1ce43326def038 HEAD -- internal/gates/gates_test.go internal/gates/regeneration_safety_test.go
- shasum -a 256 internal/gates/gates_test.go internal/gates/regeneration_safety_test.go

Summary:
- Required gates: 37 native leaves PASS, 0 FAIL, 0 SKIP; package 5.683s, exit 0.
- Downstream baseline date, deterministic-date sources, actual subprocess baseline diagnostics: 3 native leaves PASS, 0 FAIL, 0 SKIP; package 1.470s, exit 0.
- Total 40/40 selected leaves PASS. No compiler/setup/runtime deadline failures or test warnings.
- Hard-TDD PASS: local merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890..HEAD, 5 commits checked, 0 merges skipped, no unauthorized test edits. git diff --check PASS.
- Coverage: internal/gates scoped total 9.9%, gates.VersionSkewNote 93.8%, regenCommands 100%; cmd/machinery scoped total 2.2%, newBaselineCmd 75.0%, resolveBaselineDate 84.2%. These are scoped package figures; independently built real CLI subprocess is not included in internal/gates coverage. Downstream private-path test intentionally selects baseline only.
- Test integration runs actual freshly built CLI and native filesystem unconditionally; no mocks/stubs/services/env-based omission. The existing downstream fixture has a symlink-platform fallback but this run created symlinks successfully and reports zero skips.

### Retained raw evidence
Directory /tmp/MAC-a89e-green.xtIsDr:
- gates-tests.json SHA256 9f138ecab644843d7932420be54e1c2be78853b4bb558f512e33ed9890c15382
- cli-tests.json SHA256 5945d2913263d1a197d901fcf87e80121d614b2e9ee35bf40ac9e332f14419d5
- inventory.txt SHA256 67302005140ca8aab06f698508f6616dfc469d25d35e01493b1ed6312c609b99
- gates-coverage.out SHA256 f1b45d80d3aeecd7157adbeda1a85594d452cf99a2905b1355715c40f3a6fbbb
- cli-coverage.out SHA256 8f85fe7958ee02927a94af73448d5b72dbbe90e699aeb33f05047575681b325e
- verify.txt SHA256 841ae4d70019ef05407b2c096de729ef9a86c6a39226ab76fee41a76e67b51c2
- tdd.txt contains native hard-TDD PASS output.

### pvg verify
Native output: VERIFY: FAILED (4 files scanned, 4 issues), all [stub] return empty string. Documentation is not scanned by this source scanner.
- internal/gates/gates.go:76 is VersionSkewNote's documented no-skew empty result. Full implementation collects/sorts skew and emits applicable advice. Existing tests TestOracleMissingStampIsFreshAndSilent and TestOracleCurrentStampIsSilent assert empty result; real no-debt regeneration control asserts skew disappears.
- internal/gates/gates.go:219 is readDesignOrEmpty's conditional read-failure empty sentinel; success returns actual file text.
- internal/gates/gates.go:230 is readDesignFileOrErr's conditional error sentinel; non-missing errors append an explicit gate error, success returns actual file text.
- internal/gates/gates_test.go:125 is coveringInterfaceTable's conditional no-concrete-allow-rows result; otherwise full markdown table is rendered. Independently reviewed earlier as exact false positive.
All four implementations/branches are unchanged from approved RED; none is a placeholder. Three production scanner findings were explicitly escalated to dispatcher for independent disposition, not silently suppressed. No scanner changes or test-helper edits.

### AC Verification
| AC | Requirement and evidence | Status |
|---|---|---|
| 1 | gates.go regenCommands removes only baseline from the deterministic explicit generator list. Frozen VersionSkewRegenerationOnlySafeGenerators tests all individual oracle/Alloy/semantics/composition/pack families and all-with-ratchet ordered output; ratchet-only emits no command. Existing ratchet-present TestVersionSkewNoteNamesEveryApplicableCommand and generated-TLA selection remain intact. | PASS |
| 2 | RealCLI/new-offender-survives-all-advice starts with actual CLI ratchet alpha/a.go and green G4, adds real alpha/b.go import and observes exact G4 growth failure, executes EVERY advised generator, proves current oracle stamp, compares ratchet bytes unchanged and requires same offender failure afterward. | PASS |
| 3 | CLI help before invocation explains rerunning rewrites ratchet.json and may accept newly added offender files, tells users to review ratchet changes before adoption; identical substantive guidance prints only after successful publication, regardless of proposal count. Zero-rule line now says only no new baseline dependency rules proposed. RealCLI/explicit-baseline-reviews-debt-change begins already baselined, rejects alpha/b.go, deliberately invokes baseline, proves 0 need a baseline rule, two recorded offenders, unstamped ratchet and G4 green. All five semantic-guidance controls pass, including pathname bait rejection. Baseline algorithm/schema/date/confirmation semantics preserved; downstream date tests pass. | PASS |
| 4 | RealCLI/no-debt-version-skew-control obtains correct oracle advice, executes it, proves fresh current stamp/no skew and green g3,g4 without creating ratchet; negative AC2 still fails after all routine advice. Advice family controls execute deterministically. | PASS |

### Exact native leaf inventory
PASS TestG4RatchetSnapshotNote
PASS TestG4BaselineWithoutRatchetFails
PASS TestG4RatchetGreenAtSnapshot
PASS TestG4RatchetGrowthFails
PASS TestG4RatchetShrinkAndStaleEdgesNote
PASS TestRatchetRoundTrip
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/unknown_root
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edge_name
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_number
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edges_array
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_value_string
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_entry_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/trailing_value
PASS TestWriteRatchetRejectsSymlinkTarget
PASS TestVersionSkewNoteNamesEveryApplicableCommand
PASS TestVersionSkewNoteFormalCommandFollowsGeneratedTLA
PASS TestRatchetSnapshotNoteBothFormatsIsClockIndependent
PASS TestVersionSkewRegenerationOnlySafeGenerators/ratchet-only
PASS TestVersionSkewRegenerationOnlySafeGenerators/oracle
PASS TestVersionSkewRegenerationOnlySafeGenerators/alloy
PASS TestVersionSkewRegenerationOnlySafeGenerators/semantics
PASS TestVersionSkewRegenerationOnlySafeGenerators/composition
PASS TestVersionSkewRegenerationOnlySafeGenerators/pack
PASS TestVersionSkewRegenerationOnlySafeGenerators/all-with-ratchet
PASS TestRegenBaselineGuidanceSensitivity/genuine
PASS TestRegenBaselineGuidanceSensitivity/equivalent-wrapped
PASS TestRegenBaselineGuidanceSensitivity/path-bait
PASS TestRegenBaselineGuidanceSensitivity/old-rule-review-help
PASS TestRegenBaselineGuidanceSensitivity/unrelated-review
PASS TestRegenRatchetRealCLI/no-debt-version-skew-control
PASS TestRegenRatchetRealCLI/new-offender-survives-all-advice
PASS TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change
PASS TestBaselineSourceDateEpochStampIsFullDate
PASS TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting
PASS TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree/baseline/--impl_/var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/machinery-diagnostic-user-input-448099535/TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree3077046406/003_--date_2026-01-01_/var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/machinery-diagnostic-user-input-448099535/TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree3077046406/001


### Wiring, discovery, and residual limits
- Existing machinery check -> gates.VersionSkewNote -> regenCommands remains wired; existing baseline Cobra command renders help and prints guidance after PublishExpectedRooted succeeds. No new interfaces/config/product dependencies.
- Used developer and codebase-memory skills; inherited parent Verify structural evidence then independently checked project readiness and exact coverage. Graph generation 2026-09-05T23:08:53Z, no recorded gaps for gates.go, gates_test.go, baseline algorithm, CLI baseline/check/diagnostic paths. New regression test is absent from root graph; .claude worktree is excluded, so reviewed exact worktree source and tests directly. No completeness claim.
- Documentation freshness scan found the same unsafe operation in generated-file conflict advice; authorized narrow paragraph correction now separates explicit ratchet debt review.
- No full preflight or unrelated runtime suite run, as explicitly constrained; final epic gate owns those checks. Real executable fixture has oracle only; other generator families are precise command-selection tests, not claims of Alloy solver/TLA execution.
- No remote fetch/pull/push/sync, branch integration, installed binary/skills/plugins/agents changes, Docker changes, Paivot runtime dependency, or baseline algorithm/schema/date modifications. Worktree retained, committed and clean.

LEARNINGS:
- Zero new dependency-rule proposals can still mean expanded accepted offender debt; help and successful output must distinguish them.
- Explicit generators form the safe regeneration contract. A ratchet is debt policy rather than a version-stamped derivative.
- Frozen RED history exposed pathname-bait guidance assertions; approved path-stripping semantic controls now reject accidental matches without weakening real explicit-baseline proof.
- Source scanners flag valid conditional empty sentinels; retain raw findings and secure specific independent disposition instead of altering working semantics.
- User-facing merge documentation can retain unsafe command advice after the implementation is fixed; targeted freshness review found and corrected this instance under explicit scope authorization.

## nd_contract
status: in_progress

### evidence
- Candidate de9745067382c5de25f05d69ea05c2808eb0b2e7, 40 PASS/0 FAIL/0 SKIP, scoped coverage and raw evidence above.
- Frozen RED unchanged; hard-TDD PASS; five-file/272-LOC scope includes explicit dispatcher authorization.

### proof
- [x] AC #1: safe deterministic complete regeneration advice.
- [x] AC #2: real offender remains G4 failure after every advised generator.
- [x] AC #3: explicit baseline supported with truthful pre-invocation and successful debt-review guidance even at zero rules.
- [x] AC #4: real no-debt positive and debt-growth negative both exercised.


## nd_contract
status: red-approved

### evidence
- RED tests approved via pvg story approve-red on 2026-09-05.

### proof
- [ ] GREEN developer must implement against the approved RED tests without modifying them.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## RED Delivery Evidence

PROOF:

### Commits and immutable candidate
- RED: 9b85ef69ed426a337e4e2584867a698904676643.
- Narrow existing-test amendment: f482c7683f5005e40722123e3a382e01b15219d2, authorized 2026-09-05T21:14:38Z; retained ratchet fixture/all four safe generator expectations and asserted baseline absence only.
- Authorized guidance repair / tested candidate: e95be63019d89b93ae527f27ae1ce43326def038, authorized 2026-09-05T21:18:55Z; both literal tdd-red and [test-edit-authorized] markers included. Prior commits preserved without amend/rebase.
- Branch story/MAC-a89e; worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e. Production unchanged. Diff against epic/MAC-ui8a: regeneration_safety_test.go +243; gates_test.go +3/-1; 247 total changed LOC across 2 test files, leaving bounded room for 2 production files within canonical 4-file/under-300 budget.
- Test SHA256: internal/gates/gates_test.go e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; internal/gates/regeneration_safety_test.go 2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3.

### Exact commands / retained evidence
All commands ran with explicit cd to the story worktree.
- set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-red.wlHWaG/coverage.out | tee /tmp/MAC-a89e-red.wlHWaG/tests.json
- Exit 1: expected RED behavioral assertion failures. 37 leaf tests, 32 PASS, 5 FAIL, 0 SKIP; package elapsed 4.832s. No compile/setup/deadline/missing-prerequisite failures. All new tests are selected by the specified regexp; no supplemental selector needed.
- Raw JSON /tmp/MAC-a89e-red.wlHWaG/tests.json SHA256 405f556e1036a36547bf9231d38d3cdedca5d52acc6879b997f95839789584bb.
- Exact leaf inventory /tmp/MAC-a89e-red.wlHWaG/inventory.txt SHA256 7e20dff8a081de4734b40f68cf07da4d334362e37ed1654cea2b9c121c90e8af.
- Coverage profile /tmp/MAC-a89e-red.wlHWaG/coverage.out SHA256 1ff93989e5a6fb5a2c6c70dd42dc3214e241ff62a74f1429471dbc0f44e1eef0.
- go tool cover -func=/tmp/MAC-a89e-red.wlHWaG/coverage.out: internal/gates total 9.9%; gates.go regenCommands 100.0%; gates.go VersionSkewNote 93.8%. This is scoped package coverage; separately built real CLI subprocess is not included in Go package instrumentation.
- git diff --check epic/MAC-ui8a: PASS. Worktree clean.
- pvg verify internal/gates/regeneration_safety_test.go --format text: PASS, 1 file, 0 issues.
- pvg verify internal/gates/gates_test.go internal/gates/regeneration_safety_test.go --format text: reports one [stub] finding, gates_test.go:125 return "". Raw result /tmp/MAC-a89e-red.wlHWaG/verify.txt. Specific independent PM false-positive disposition at 2026-09-05T21:18:55Z: coveringInterfaceTable returns empty only for no concrete allow rows; otherwise renders the full interface table, consumed by c4GraphFixture and graph tests. Preserved helper unchanged. This is a recorded single finding, not blanket scanner waiver; earlier note's helper name fixtureInterfaceRows was incorrect and is superseded by coveringInterfaceTable.

### Exact test inventory
PASS TestG4RatchetSnapshotNote
PASS TestG4BaselineWithoutRatchetFails
PASS TestG4RatchetGreenAtSnapshot
PASS TestG4RatchetGrowthFails
PASS TestG4RatchetShrinkAndStaleEdgesNote
PASS TestRatchetRoundTrip
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/unknown_root
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edge_name
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_number
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edges_array
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_value_string
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_entry_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/trailing_value
PASS TestWriteRatchetRejectsSymlinkTarget
FAIL TestVersionSkewNoteNamesEveryApplicableCommand
PASS TestVersionSkewNoteFormalCommandFollowsGeneratedTLA
PASS TestRatchetSnapshotNoteBothFormatsIsClockIndependent
FAIL TestVersionSkewRegenerationOnlySafeGenerators/ratchet-only
PASS TestVersionSkewRegenerationOnlySafeGenerators/oracle
PASS TestVersionSkewRegenerationOnlySafeGenerators/alloy
PASS TestVersionSkewRegenerationOnlySafeGenerators/semantics
PASS TestVersionSkewRegenerationOnlySafeGenerators/composition
PASS TestVersionSkewRegenerationOnlySafeGenerators/pack
FAIL TestVersionSkewRegenerationOnlySafeGenerators/all-with-ratchet
PASS TestRegenBaselineGuidanceSensitivity/genuine
PASS TestRegenBaselineGuidanceSensitivity/equivalent-wrapped
PASS TestRegenBaselineGuidanceSensitivity/path-bait
PASS TestRegenBaselineGuidanceSensitivity/old-rule-review-help
PASS TestRegenBaselineGuidanceSensitivity/unrelated-review
PASS TestRegenRatchetRealCLI/no-debt-version-skew-control
FAIL TestRegenRatchetRealCLI/new-offender-survives-all-advice
FAIL TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change

### Independent AC-to-test evidence
| AC | Test and observable result | RED status |
|---|---|---|
| 1 | VersionSkewRegenerationOnlySafeGenerators pins exact ordered command list with no additional mutations for every family, individual controls and all-with-ratchet. Ratchet-only/all-with-ratchet plus authorized existing expectation fail because machinery baseline is printed. Unchanged repeated inputs produce identical advice; single-family positive controls pass. | Intended failure |
| 2 | RealCLI/new-offender-survives-all-advice starts with CLI-created ratchet containing alpha/a.go and green g3,g4, then adds alpha/b.go and gets exact G4 growth finding. Executes EVERY printed routine command (including unsafe baseline after substituting its documented impl placeholder), proves current oracle stamp generated, then compares exact ratchet bytes and requires G4 still reject alpha/b.go. Unchanged production expands ratchet to [alpha/a.go,alpha/b.go], followed by G4 zero blocking findings. | Intended failure on advice, bytes and G4 result |
| 3 | RealCLI/explicit-baseline-reviews-debt-change logs actual --help before baseline invocation; requires ratchet replacement, possible new-offender acceptance and review-before-adoption prose. Existing ratchet + new offender fails G4 first; deliberate real baseline succeeds with explicit "0 need a baseline rule", records both offenders, remains unstamped and subsequent G4 passes. Current help fails guidance; successful output fails guidance and "nothing new to baseline" falsely implies no debt change. | Intended failure for missing guidance / misleading zero-rule output; explicit operation remains successful |
| 4 | RealCLI/no-debt-version-skew-control runs oracle on a valid source, introduces stamp-only skew, obtains exactly the oracle advice, executes it, verifies current stamp, no skew after regeneration, g3,g4 green, and no new ratchet. All five sensitivity controls pass (two equivalent valid prose variants, unsafe output with review/reviews path bait, old dependency-rule-only review help, unrelated review words). | Controls PASS; negative AC2 demonstrably fails |

### Limits and scope
This delivery is RED ONLY, not product completion or permission to begin GREEN without independent RED approval. Integration builds ./cmd/machinery into t.TempDir and uses real CLI/native files unconditionally, no mocks, services, skips or env gates. Real executable fixture carries oracle generation, so all applicable printed commands are executed. Other safe generator families are pinned as exact advice-selection controls; no Alloy solver/TLA model-check execution is claimed. No full preflight, installed binary/plugin/skill modification, push/sync/GitHub write, main branch move, runtime schema change or Paivot product dependency.

LEARNINGS:
- Initial AC3 check falsely passed because the temporary pathname inherited "reviews" from its subtest. After independent authorization, matching now removes path tokens and relates review/change/ratchet/offender semantics within prose, with explicit passing sensitivity controls.
- Existing baseline rerun expands offender snapshots despite zero new dependency-rule proposals; rule proposal counts and accepted debt changes are distinct observations.
- Executing all printed commands exposes the exact harmful mutation: regenerated current stamps alone do not preserve architecture enforcement if the advice also re-baselines debt.
- A scanner can flag a legitimate conditional empty fixture section; preserve implemented behavior and record an exact reviewer disposition.
- Retained raw JSON and hashes make failure provenance independently replayable; initial summarized-only evidence was insufficient for that purpose.

## nd_contract
status: in_progress

### evidence
- Candidate e95be63019d89b93ae527f27ae1ce43326def038; complete scoped RED 32 pass / 5 intended fail / 0 skip, raw evidence above.
- Authorized repaired tests, preserved original commits and specific static-finding disposition.

### proof
- [x] AC #1: behavioral RED with exact safe advice and positive controls authored.
- [x] AC #2: real CLI harmful mutation independently observable in ratchet bytes and G4 output.
- [x] AC #3: missing help/output guidance now genuinely fails; explicit operation and matcher controls exercised.
- [x] AC #4: real no-debt positive plus genuine negative path executed; full inventory recorded.


## nd_contract
status: in_progress

### evidence
- Provisional RED and authorized amendment committed; paused for narrow test repair authorization and AC3 scope update.
- No delivery, claim release, production change, installed asset change, or remote mutation.

### proof
- [ ] AC #1: intended assertion failures established; independent RED approval pending.
- [ ] AC #2: real CLI regression established; independent RED approval pending.
- [ ] AC #3: RED-DISPUTE false-positive assertion must be repaired after authorization.
- [ ] AC #4: complete target inventory and coverage pending repair.

## Historical canonical description before MAC-a89e scope repair
Preserved verbatim as quoted history; current Description is authoritative.

> ## USER INTENT
> Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.
> 
> ## Context (Embedded)
> NEXT9: version-skew advice includes machinery baseline when ratchet exists. Baseline snapshots tolerated import offenders and can widen accepted architecture debt; ratchet has no version stamp. Routine regeneration must not silently authorize debt.
> 
> ## Ownership
> Own only these paths and directly associated tests: internal/gates/gates.go, internal/gates/regeneration_safety_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.
> 
> ## Boundary Map
> PRODUCES:
> - internal/gates/gates.go -> hardened behavior and regression proof
> - internal/gates/regeneration_safety_test.go -> hardened behavior and regression proof
> CONSUMES:
> - Existing Machinery source interfaces.
>   spec: regenCommands(design string) []string
> 
> ### Story Acceptance Criteria
> 1. Version-skew regeneration instructions never include machinery baseline or any debt-accepting mutation. Existing oracle/Alloy/formal/pack regeneration remains accurate and deterministic.
> 2. A design with ratchet and newly introduced offender continues failing architecture checks after following all routine regeneration instructions; regenerated stamps cannot accept the offender.
> 3. Explicit baseline remains a deliberate user-invoked operation with clear debt-change review guidance, not automatic migration. Do not add a version stamp as a substitute for preventing debt expansion.
> 4. Positive no-debt/version-skew fixture yields correct required generator commands; negative existing-ratchet/new-offender regression demonstrates failure before and after advised regeneration.
> 
> ## Testing Requirements
> - Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
> - Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
> - Commands: go test ./internal/gates -run 'VersionSkew|Regen|Ratchet'; CLI integration on isolated ratchet design with a genuine new boundary violation.
> - No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.
> 
> ## OUT OF SCOPE
> - Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
> - Global heavy preflight, main merge and local release binary belong to final epic gate.
> 
> ## DIFF BUDGET
> - ~2-3 files, under 300 changed LOC; material overrun requires PM investigation, not weakened requirements.
> 
> ## MANDATORY SKILLS
> - developer for implementation; codebase-memory for discovery; pm_acceptor for independent acceptance.
> 
> ## Delivery Requirements
> Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands and outputs, test inventory, independent proof per AC, and any residual limits to shared nd. Do not use pushing pvg story merge.
> 
> ## nd_contract
> status: new
> 
> ### evidence
> - Created 2026-09-05; source signatures verified at assessment base 497419ab4512fcff765cd5feb27aed4c67b5608d.
> 
> ### proof
> - [ ] AC #1: independently verified
> - [ ] AC #2: independently verified
> - [ ] AC #3: independently verified
> - [ ] AC #4: independently verified


## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97
- 2026-09-05T21:11:54Z status: open -> in_progress
- 2026-09-05T21:11:54Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-05T21:11:54Z claimed by dev-MAC-a89e
- 2026-09-05T21:23:23Z status: in_progress -> in_progress
- 2026-09-05T21:28:42Z status: in_progress -> open
- 2026-09-05T23:34:48Z status: open -> in_progress
- 2026-09-05T23:34:48Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-05T23:34:48Z claimed by dev-MAC-a89e
- 2026-09-05T23:41:17Z status: in_progress -> in_progress
- 2026-09-05T23:46:44Z status: in_progress -> open
- 2026-09-05T23:46:44Z released by ramirosalas
- 2026-09-05T23:47:31Z status: open -> in_progress
- 2026-09-05T23:47:31Z claimed by dev-MAC-a89e
- 2026-09-05T23:49:26Z status: in_progress -> in_progress

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Follows: [[MAC-olrx]], [[MAC-p8ce]]

## Comments

### 2026-09-05T21:14:38Z ramirosalas
TEST-EDIT AUTHORIZED: internal/gates/gates_test.go -- pre-RED repair limited to TestVersionSkewNoteNamesEveryApplicableCommand. Replace the expected "machinery baseline <design> --impl <dir>" regeneration command with an explicit assertion that the note does not contain "machinery baseline"; preserve the ratchet-present fixture and all existing oracle, Alloy, verify-formal, and pack generator expectations. Its present baseline expectation directly contradicts AC1 and encodes the unsafe behavior being removed. The repair commit subject must contain both tdd-red and [test-edit-authorized]. No other existing tests or assertions are authorized for modification by this decision.

Evidence: read pvg issues show MAC-a89e --json; inspected committed main 497419ab4512fcff765cd5feb27aed4c67b5608d and epic f24b2df3cb1e1521f97b406a7516f72bb7bc7890. The reviewed gates_test.go and cmd/machinery/baseline.go are identical between these refs. gates.go regenCommands currently appends baseline solely when RatchetFile exists. No uncommitted author work inspected, tests replayed, source files edited, or story status/labels changed.

AC3 SCOPE GAP: Current baseline help advises review of pasted dependency rules and rerunning after burning down debt, but does not explain that rerunning can accept newly added offender files by replacing ratchet.json. The zero-proposed-rules path prints "the contract already covers every observed edge; nothing new to baseline", then publishes the new ratchet and declares it armed, without debt-change review guidance. AC3 therefore cannot be proven by pinning the current help text.

Required bounded Sr PM ownership clarification: include cmd/machinery/baseline.go for help/output messaging and directly associated regression coverage. State explicitly that baseline is a deliberate debt-acceptance operation, rerunning may expand accepted offenders, and users must review ratchet.json/offender changes before adopting them. Successful output must convey this even when no dependency rule is proposed; help should explain it before invocation. Preserve explicit baseline functionality and do not add automatic migration, a version stamp, or a new confirmation protocol. A real CLI RED case that deliberately invokes baseline on an existing edge with a new offender, demonstrates the changed ratchet/accepted offender, and requires clear review guidance covers the actual missing behavior. This note does not authorize source edits outside current story ownership; Sr PM must repair that scope first.

This is narrow test-edit authorization and AC3 scope analysis only. It is NOT RED approval, delivery acceptance, rejection, or a status transition. Author must produce genuine assertion-failure RED proof and passing controls, followed by a full independent RED review before GREEN.

## nd_contract
status: in_progress

### evidence
- Independent pre-RED dispute review against committed main/epic source; narrowly authorized test expectation repair as recorded above.
- AC3 help/output ownership gap reported to dispatcher for Sr PM clarification.

### proof
- [x] AC #1: existing baseline-positive expectation is contradicted by the criterion; bounded replacement authorized.
- [ ] AC #2: new real CLI before/after regeneration proof awaits RED author and independent review.
- [ ] AC #3: explicit debt-change review guidance requires the bounded scope repair above.
- [ ] AC #4: full RED proof and passing controls await independent review.

## AUTHORITATIVE MAC-a89e SCOPE REPAIR — 2026-09-05
Current canonical Description includes the concrete AC3 baseline debt-review behavior and supersedes only conflicting earlier ownership/AC3/budget wording. Previous description, PM authorization, notes and history remain preserved.

Bug-triage disposition: absorb the newly confirmed explicit-baseline messaging gap into the existing P0 MAC-a89e under MAC-ui8a. It is already required by AC3 and concerns the same distinction between routine regeneration and deliberate debt acceptance. No new issue, dependency, claim or status/label transition.

Scope delta: cmd/machinery/baseline.go is now explicitly owned only for help and successful-output guidance. internal/gates/gates_test.go is owned only for the PM-authorized TestVersionSkewNoteNamesEveryApplicableCommand baseline-positive expectation replacement with baseline absence, preserving its ratchet fixture and all other generator assertions. Existing gates.go and new regeneration_safety_test.go remain owned. Total four files; original under-300-LOC ceiling retained.

AC3 observable contract: help explains baseline reruns rewrite ratchet.json and can accept new offender files; successful output tells users to review ratchet/offender changes before adoption even if no dependency rule was proposed. Zero proposed rules does not mean no debt change. The real isolated CLI test must demonstrate an already-baselined edge failing after a new offender, explicit baseline successfully accepting that offender with zero rule proposals, and the required help/output guidance. Deliberate baseline acceptance remains allowed. No confirmation prompt, stamp, algorithm/schema rewrite, or Paivot runtime coupling.

Evidence: full latest MAC-a89e story and independent PM comment read; graph-first exact source for regenCommands, VersionSkewNote, BuildBaseline, newBaselineCmd and TestVersionSkewNoteNamesEveryApplicableCommand. Source confirms baseline is included when RatchetFile exists, existing test requires it, BuildBaseline re-snapshots current offenders on baselined edges, and CLI zero-rule path claims nothing new before writing the expanded ratchet. Graph coverage 2026-09-05T20:28:41Z reported metadata_match/no_recorded_issue for four cited source/test files; exact snippets were read. No author worktree/source/frozen tests inspected or modified and no runtime test replay performed during this tracker-only scope repair.

The existing 2026-09-05T21:14:38Z TEST-EDIT AUTHORIZED comment remains the authority for that narrow existing-test change and its tdd-red/[test-edit-authorized] development commit markers. This scope repair is not RED approval, delivery or acceptance. Parent will lint and notify the author; full independent RED replay is still required before GREEN.

## nd_contract
status: in_progress

### evidence
- Canonical self-contained description repaired through supported pvg nd edit; append-only scope note written with pvg nd update --append-notes.
- P0 MAC-a89e retains dev-MAC-a89e, hard-tdd, parent MAC-ui8a and existing blocks MAC-gcrr/MAC-ou97.
- Four-file scope with under-300-LOC budget; all previous notes/history/comments/contracts preserved.

### proof
- [x] Scope repair: AC3 explicitly covers help and successful zero-rule output for ratchet/offender debt expansion.
- [x] Scope repair: narrow existing-test authorization embedded without enlarging its allowed assertion changes.
- [ ] AC #1-4: RED execution, independent approval, implementation and acceptance remain pending.


### 2026-09-05T21:18:55Z ramirosalas
TEST-EDIT AUTHORIZED: internal/gates/regeneration_safety_test.go -- bounded pre-approval AC3 assertion repair against committed 9b85ef6 and f482c7683f5005e40722123e3a382e01b15219d2. In TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change, the existing whole-output strings.Contains(lower, "review") assertion is invalid: t.TempDir incorporates the subtest name containing "reviews", and baseline prints the resulting design path in its wrote-ratchet line. Existing "ratchet", "debt", and "offender" words elsewhere complete the false positive without any actual review instruction. The author-reported AC3 pass is NOT valid evidence.

Authorized repair: strengthen only the AC3 guidance proof and any small directly supporting matcher/sensitivity tests in this new regression file. Read baseline --help before deliberate baseline invocation and assert the canonical AC3 semantics: rerunning rewrites ratchet.json, may accept new offender files, and users must review those ratchet/offender changes before adopting them. Assert successful output carries that debt-change review guidance for the existing-edge/new-offender/zero-new-rule case and that its zero-rule message does not imply no accepted debt or no change. Verify zero proposed rules explicitly. Preserve the real successful baseline call, actual offender expansion, preceding G4 failure, subsequent G4 success, and absence-of-version-stamp checks. Preserve all unrelated generator/advice/no-debt/regeneration assertions and fixtures. No production edits are authorized in this RED amendment.

Robustness bar: inspect actual prose, excluding fixture paths from evidence and relating review guidance to ratchet/offender changes and the possibility of accepting new debt. Unrelated mentions of words across help/output must not suffice. Add focused passing sensitivity controls: genuine guidance must be recognized; the existing unsafe zero-rule output with a temporary pathname containing review/reviews and ratchet/debt/offender bait must be rejected; existing help that reviews pasted dependency rules but never warns about accepting added offenders must be rejected. The matcher should allow natural equivalent wording and whitespace/wrapping, without prescribing one exact production sentence. These controls supplement, never replace, the real CLI assertions.

Commit the repair with both literal tdd-red and [test-edit-authorized] in its subject. Preserve 9b85ef6 and f482c7683f5005e40722123e3a382e01b15219d2 in history; no amend/rebase/squash. Replay and record the scoped target, sensitivity-control results, genuine guidance assertion failures on unchanged production, passing controls, inventory/skips, coverage and exact SHA before delivery. Full independent RED review still required; this decision is not approve-red, acceptance, rejection or a status transition.

Scope evidence: read latest canonical MAC-a89e Description and authoritative Sr PM scope repair; the four-file ownership and expanded observable AC3 are now recorded. Reviewed committed regression source at f482c7683f5005e40722123e3a382e01b15219d2 lines 179-195, baseline help/output source, and the prior narrow gates_test.go diff, which exactly matches the earlier authorization. Shared provisional proof records 10 leaves / 7 pass / 3 intended failures / 0 skip and invalid AC3 pass. No retained raw command-output path is supplied there; I have not independently replayed that run or inspected a raw transcript. The precise false-positive mechanism is independently established from committed source, sufficient for this bounded repair authorization; raw replay evidence remains required for full RED review.

STATIC SCAN FALSE POSITIVE DISPOSITION: pvg verify flag at internal/gates/gates_test.go:125 is legitimate fixture behavior, not a stub. The actual committed helper is coveringInterfaceTable (not fixtureInterfaceRows). Lines 108-122 parse concrete allow edges and build Markdown rows; lines 124-125 return an empty string only when no concrete rows exist, correctly omitting an unnecessary interface-contract section. Lines 127-128 render a full table otherwise. c4GraphFixture line 145 includes that section in ARCHITECTURE.md then executes CheckC4; TestG2AllowGraphAcyclicity and TestG2TransitivePairsCount exercise the generated coherent graph fixtures. The helper predates this story and is unchanged in the two reviewed commits. Preserve it without removal, rewriting, disabled checks or a blanket scan waiver. Record this one explained scanner finding alongside verification evidence; the narrow gates_test.go amendment remains limited to TestVersionSkewNoteNamesEveryApplicableCommand.

## nd_contract
status: in_progress

### evidence
- Independent committed-source review authorized the bounded AC3 test strengthening and scoped sensitivity controls.
- Canonical AC3 ownership repair read; prior gates_test.go amendment matches authorization.
- One precise static scan false positive explained; raw RED output/replay remains for full review.

### proof
- [x] Amendment review: pathname false positive established and robust repair bounded.
- [x] Static disposition: conditional fixture section omission is implemented behavior, not a stub.
- [ ] AC #1-4: complete RED delivery and independent approval remain pending.

### 2026-09-05T21:25:31Z ramirosalas
## Implementation Evidence

PROOF:

### CI/Test Results
Commands run:
- cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e && set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-red.wlHWaG/coverage.out | tee /tmp/MAC-a89e-red.wlHWaG/tests.json
- go tool cover -func=/tmp/MAC-a89e-red.wlHWaG/coverage.out
- git diff --check epic/MAC-ui8a
- pvg verify internal/gates/regeneration_safety_test.go --format text
- pvg verify internal/gates/gates_test.go internal/gates/regeneration_safety_test.go --format text

Summary: RED ONLY. 37 executed leaf tests, 32 PASS, 5 intended assertion FAIL, 0 SKIP; exit 1, package 4.832s. No compilation, setup, timeout, or missing-prerequisite failure. Five guidance matcher controls and the real no-debt CLI control pass. No test rerun during this proof-record repair.
Coverage: internal/gates scoped total 9.9%; gates.go regenCommands 100.0%; gates.go VersionSkewNote 93.8%. Real CLI subprocess is built separately and excluded from package coverage instrumentation.

### Commit
SHA: e95be63019d89b93ae527f27ae1ce43326def038
Branch: story/MAC-a89e
Initial RED SHA: 9b85ef69ed426a337e4e2584867a698904676643
Authorized existing expectation amendment SHA: f482c7683f5005e40722123e3a382e01b15219d2
Candidate e95be63019d89b93ae527f27ae1ce43326def038 is the authorized AC3 assertion repair. Prior commits remain intact; no production changes. Two test files total +246/-1 = 247 changed LOC against epic/MAC-ui8a.
Test hashes SHA256:
- internal/gates/gates_test.go: e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff
- internal/gates/regeneration_safety_test.go: 2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3

### Retained raw evidence
- /tmp/MAC-a89e-red.wlHWaG/tests.json SHA256 405f556e1036a36547bf9231d38d3cdedca5d52acc6879b997f95839789584bb
- /tmp/MAC-a89e-red.wlHWaG/coverage.out SHA256 1ff93989e5a6fb5a2c6c70dd42dc3214e241ff62a74f1429471dbc0f44e1eef0
- /tmp/MAC-a89e-red.wlHWaG/inventory.txt SHA256 7e20dff8a081de4734b40f68cf07da4d334362e37ed1654cea2b9c121c90e8af
- /tmp/MAC-a89e-red.wlHWaG/verify.txt contains the single reviewed static-scan finding.

### Test inventory
PASS TestG4RatchetSnapshotNote
PASS TestG4BaselineWithoutRatchetFails
PASS TestG4RatchetGreenAtSnapshot
PASS TestG4RatchetGrowthFails
PASS TestG4RatchetShrinkAndStaleEdgesNote
PASS TestRatchetRoundTrip
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/unknown_root
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edge_name
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_number
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edges_array
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_value_string
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_entry_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/trailing_value
PASS TestWriteRatchetRejectsSymlinkTarget
FAIL TestVersionSkewNoteNamesEveryApplicableCommand
PASS TestVersionSkewNoteFormalCommandFollowsGeneratedTLA
PASS TestRatchetSnapshotNoteBothFormatsIsClockIndependent
FAIL TestVersionSkewRegenerationOnlySafeGenerators/ratchet-only
PASS TestVersionSkewRegenerationOnlySafeGenerators/oracle
PASS TestVersionSkewRegenerationOnlySafeGenerators/alloy
PASS TestVersionSkewRegenerationOnlySafeGenerators/semantics
PASS TestVersionSkewRegenerationOnlySafeGenerators/composition
PASS TestVersionSkewRegenerationOnlySafeGenerators/pack
FAIL TestVersionSkewRegenerationOnlySafeGenerators/all-with-ratchet
PASS TestRegenBaselineGuidanceSensitivity/genuine
PASS TestRegenBaselineGuidanceSensitivity/equivalent-wrapped
PASS TestRegenBaselineGuidanceSensitivity/path-bait
PASS TestRegenBaselineGuidanceSensitivity/old-rule-review-help
PASS TestRegenBaselineGuidanceSensitivity/unrelated-review
PASS TestRegenRatchetRealCLI/no-debt-version-skew-control
FAIL TestRegenRatchetRealCLI/new-offender-survives-all-advice
FAIL TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change

### pvg verify
New test file: PASSED, 1 file scanned, 0 issues.
Both changed files: FAILED, 2 files scanned, one [stub] at gates_test.go:125 return "".
Specific PM disposition, 2026-09-05T21:18:55Z: actual helper coveringInterfaceTable conditionally omits the Markdown section when no concrete allow-edge rows exist; otherwise it renders the full table. c4GraphFixture uses it and graph tests exercise it. This is implemented fixture behavior, not a stub. Preserve unchanged; no blanket scan waiver. Earlier provisional note naming fixtureInterfaceRows was incorrect.

### AC Verification
| AC | Test and evidence | RED result |
|---|---|---|
| 1 | TestVersionSkewRegenerationOnlySafeGenerators asserts exact ordered oracle/Alloy/formal/pack advice and no extra mutation. Five single-family controls pass; ratchet-only/all-with-ratchet and authorized existing test fail because advice includes baseline. Repeated calls are deterministic. | Intended assertion failures |
| 2 | RealCLI/new-offender-survives-all-advice: CLI creates ratchet with alpha/a.go and initially green g3,g4; new alpha/b.go produces exact growth failure; every printed command runs, including unsafe baseline after impl placeholder substitution. Oracle current stamp appears, ratchet bytes change to include alpha/b.go, and subsequent G4 incorrectly returns zero findings. | Intended advice, byte-preservation and G4 assertion failures |
| 3 | RealCLI/explicit-baseline-reviews-debt-change reads actual help before any fixture baseline invocation, adds offender to an existing ratchet, proves G4 failure, deliberately invokes baseline successfully with explicit 0 need a baseline rule, records both offenders, verifies absent version stamp, and proves subsequent G4 success. Current help/output lack required guidance and zero-rule text misleadingly says nothing new to baseline. | Genuine guidance and misleading-message failures; explicit operation still succeeds |
| 4 | RealCLI/no-debt-version-skew-control regenerates a valid oracle through CLI, introduces stamp-only skew, follows exactly the oracle advice, proves current stamp, no residual skew, green g3,g4 and no ratchet creation. All five matcher sensitivity controls pass, including path bait and old dependency-rule help rejection. | Positive controls PASS; negative regression established |

### Limits
RED only, independent PM RED approval still required. Real CLI/native filesystem tests are unconditional, without mocks, stubs, skip/env gates or external services. The real executable fixture carries oracle generation and executes every applicable printed command; other generator families have exact advice-selection controls. No Alloy solver/TLA model checking is claimed. No full preflight, installed asset modification, push/sync/GitHub mutation, main branch movement or production Paivot dependency. GREEN remains pending.

LEARNINGS:
- Whole-output review substring initially matched a temporary pathname inherited from the test name. Independent authorization allowed path exclusion, related-prose matching and five passing sensitivity controls.
- Zero newly proposed dependency rules can coexist with newly accepted offender files; the CLI regression proves both separately.
- Executing every printed regeneration command exposes actual ratchet mutation and lost G4 enforcement.
- A legitimate empty fixture section can trigger a static scanner; preserve behavior and record an exact reviewer disposition.
- Raw JSON retention and fresh canonical readback are required; the earlier append returned success but final proof did not survive the subsequent story serialization. This comment is the supported append-only correction; it changes no tests or status.

## nd_contract
status: delivered

### evidence
- RED candidate e95be63019d89b93ae527f27ae1ce43326def038; 37 leaf tests, 32 pass / 5 intended fail / 0 skip; raw logs and hashes above.
- This terminal comment restores proof and authoritative RED-delivered contract after prior notes did not persist. Existing delivered label is preserved; story deliver is not rerun.

### proof
- [x] AC #1: exact safe generator advice specified with intended RED and passing family controls.
- [x] AC #2: actual CLI ratchet expansion and lost G4 failure reproduced.
- [x] AC #3: repaired help/output assertions genuinely fail; explicit baseline and sensitivity controls exercised.
- [x] AC #4: real positive no-debt and negative existing-ratchet paths executed with full inventory.


### 2026-09-05T21:28:38Z ramirosalas
## PM RED Review — MAC-a89e — 2026-09-05
RED REVIEW APPROVED for immutable candidate e95be63019d89b93ae527f27ae1ce43326def038, pending the supported approve-red transition below. This is RED-bar approval only, not acceptance or product completion.

Reviewed full current canonical Description, both bounded TEST-EDIT AUTHORIZED comments, latest terminal Implementation Evidence and delivered contract, committed files and diffs from epic f24b2df3cb1e1521f97b406a7516f72bb7bc7890. Re-ran the full verification ladder starting at static checks; independently replayed RED rather than relying on earlier dispute reviews.

Tier 1 / scope / integrity:
- Candidate contains only internal/gates/regeneration_safety_test.go (+243) and the precisely authorized gates_test.go expectation replacement (+3/-1); 247 changed LOC. Production and all other existing tests/fixtures unchanged.
- Initial 9b85ef69ed426a337e4e2584867a698904676643, f482c7683f5005e40722123e3a382e01b15219d2 and e95be63019d89b93ae527f27ae1ce43326def038 remain in order; tdd-red markers present and both repair subjects contain [test-edit-authorized].
- Frozen SHA256: gates_test.go e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; regeneration_safety_test.go 2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3. Both independently match delivery.
- git diff --check f24b2df..e95be63019d89b93ae527f27ae1ce43326def038 passed. No new TODO, skip, mock, environment-gated omission, panic or stub found. Go function signatures are typed; no config or cross-cutting integration changes apply.
- pvg verify internal/gates/gates_test.go internal/gates/regeneration_safety_test.go --format text reproduces exactly one finding at gates_test.go:125. Rechecked committed diff: coveringInterfaceTable is unchanged and its conditional empty return legitimately omits a section when no concrete allow rows exist, otherwise renders a complete table used by c4GraphFixture. The prior precise false-positive disposition stands; no blanket waiver, edits or disabled checks.
- Documentation freshness: no production behavior is claimed in this RED-only candidate; actual baseline --help and successful output are explicitly in frozen AC3 proof, so GREEN must update both.

Tier 2 / independent execution:
Created a dedicated detached PM checkout at /tmp/MAC-a89e-pm.oPU8eu/checkout pinned to e95be63019d89b93ae527f27ae1ce43326def038. Author worktree untouched.
Exact command:
cd /tmp/MAC-a89e-pm.oPU8eu/checkout && set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-pm.oPU8eu/coverage.out | tee /tmp/MAC-a89e-pm.oPU8eu/tests.json
Exit 1, package result 9.217s: 37 executed leaves, 32 PASS, 5 intended assertion FAIL, 0 SKIP. Leaf inventory exactly matches delivered names/statuses. No compile, build, setup, timeout, prerequisite, warning or unexpected failure occurred.
Five failing leaves:
- TestVersionSkewNoteNamesEveryApplicableCommand
- TestVersionSkewRegenerationOnlySafeGenerators/ratchet-only
- TestVersionSkewRegenerationOnlySafeGenerators/all-with-ratchet
- TestRegenRatchetRealCLI/new-offender-survives-all-advice
- TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change
All five guidance sensitivity leaves and real no-debt CLI control PASS. Parent failures were excluded from leaf counts.
go tool cover -func=/tmp/MAC-a89e-pm.oPU8eu/coverage.out: package 9.9%, gates.go regenCommands 100.0%, gates.go VersionSkewNote 93.8%. Separately built real CLI is not covered by package instrumentation.
Retained independent raw JSON SHA256 fc8d76f2fc90d376f912138094277a59b82099fb76fce68fd87103ee8aba64f6; coverage SHA256 1ff93989e5a6fb5a2c6c70dd42dc3214e241ff62a74f1429471dbc0f44e1eef0 (identical to author's profile). Author's raw JSON, inventory and coverage hashes also independently verified.

Tier 3 / AC and user-intent judgment:
- AC1: exact ordered command inventories cover ratchet-only, each generator source family and all families with ratchet; repeated inputs must produce identical notes. Baseline advice fails three precise assertions; all safe single-family controls pass. Existing generated-TLA formal-command control also passes. These assert actual selected advice, not function existence.
- AC2: real built CLI starts with valid fresh oracle generation, a CLI-created ratchet accepting alpha/a.go, and green g3/g4. Adding alpha/b.go requires the specific G4 new-offender failure. EVERY advised command is executed with the documented impl placeholder filled, including the unsafe baseline in current production. Tests require a freshly regenerated oracle stamp, byte-for-byte unchanged ratchet, and persistent exact G4 rejection. Replay proves current production expands ratchet to alpha/a.go plus alpha/b.go and then returns zero blocking findings; the negative remains decisively RED.
- AC3: actual baseline --help is read before fixture baseline invocation. Deliberate existing-edge baseline succeeds after genuine new-offender rejection, explicitly reports 0 proposed baseline rules, records both offenders, adds no version stamp, and subsequent G4 succeeds. Guidance assertions independently fail on missing help semantics, misleading zero-rule prose, and missing successful-output review guidance. Matcher strips path tokens and requires related replacement/possible-acceptance/review-before-adoption prose; two equivalent/wrapped valid examples pass and pathname bait, old rule-only help and unrelated review text are rejected. No exact production sentence is prescribed.
- AC4: real no-debt control regenerates its oracle, resolves version skew, keeps g3/g4 green, and creates no ratchet; negative ratchet path exercises failure before and after advice. Safe generator family selection is unit-covered; actual executable fixture carries oracle and runs every command applicable to that fixture. No claim of Alloy solving or TLA model checking.
- Standalone intent: real commands run the candidate Machinery executable directly without Paivot/nd/workflow runtime coupling, mocks, external services, skip-if-missing or environment gates. Deliberate baseline stays functional while routine advice must preserve architectural debt enforcement. Tests passing unchanged would prove all four bounded ACs; no shallow or implementation-only substitute identified.
No unrelated discovered bug, unauthorized scope change, unresolved review gap, or human-only decision found. Freeze the exact two test files and their fixture dependencies at this candidate for GREEN. Source implementation remains pending.

## nd_contract
status: delivered

### evidence
- Full independent RED review and isolated replay at e95be63019d89b93ae527f27ae1ce43326def038: 32 PASS / 5 intended FAIL / 0 SKIP.
- Reviewed exact authorization diffs, confirmed frozen hashes and all four AC proof obligations.
- Supported approve-red transition is the next operation; no acceptance/closure is authorized by this review.

### proof
- [x] AC #1: deterministic complete safe advice contract validated at RED.
- [x] AC #2: real offender/regeneration/ratchet/G4 regression independently reproduced.
- [x] AC #3: real deliberate baseline and guidance sensitivity requirements independently validated.
- [x] AC #4: real positive control and negative behavior independently exercised.

### 2026-09-05T21:29:03Z ramirosalas
## PM RED Transition Verified — 2026-09-05
pvg story approve-red MAC-a89e succeeded using the shared vault. Fresh canonical readback confirms Status open; Labels hard-tdd, red-approved; delivered removed. Full independent review comment and proof remain present. This terminal contract supersedes earlier rendered delivered contracts, because comments render after transition notes.

Frozen RED candidate: e95be63019d89b93ae527f27ae1ce43326def038. GREEN may implement the four-file canonical scope but must preserve the exact approved test and fixture bytes. No story acceptance or closure occurred.

## nd_contract
status: new

### evidence
- Supported approve-red transition completed and canonical status/labels independently verified.
- RED approved at e95be63019d89b93ae527f27ae1ce43326def038 after isolated 37-leaf replay: 32 PASS / 5 intended FAIL / 0 SKIP.
- Full PM proof, raw evidence locations, frozen hashes and specific scanner false-positive disposition are retained in the preceding review comment.

### proof
- [x] AC #1: approved RED advice-selection contract.
- [x] AC #2: approved RED real CLI debt-preservation regression.
- [x] AC #3: approved RED deliberate-baseline and guidance contract.
- [x] AC #4: approved RED real positive/negative controls.
- [ ] GREEN implementation and final acceptance remain pending.

### 2026-09-05T23:41:28Z ramirosalas
## Implementation Evidence (DELIVERED GREEN — MAC-a89e)

PROOF:

### Commit and scope
- Frozen approved RED: e95be63019d89b93ae527f27ae1ce43326def038.
- GREEN production: bcba7900a042286b45ff46ae08bc04516ce34923.
- Final tested candidate: de9745067382c5de25f05d69ea05c2808eb0b2e7 on story/MAC-a89e, worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e.
- Scope expansion explicitly authorized by dispatcher during GREEN: docs/brownfield-team-guide.md only the generated-file merge paragraph around 259–265; separate ratchet conflicts as explicit debt-policy review, preserve deliberate baseline, require review of accepted offender changes, no new merge algorithm/confirmation workflow. Five files, 272 total changed LOC against local merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890; original four production/test paths plus this one document. GREEN adds 16/removes 9 lines across three non-test files.
- Both frozen tests remain byte-identical to approved RED (git diff e95be63 HEAD -- test paths exits 0). SHA256 gates_test.go=e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; regeneration_safety_test.go=2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3. No authored tests or fixtures edited during GREEN.

### CI/Test Results
Commands run:
All shell commands use explicit cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e.
- set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-green.xtIsDr/gates-coverage.out | tee /tmp/MAC-a89e-green.xtIsDr/gates-tests.json
- set -o pipefail && go test -count=1 -timeout=5m ./cmd/machinery -run '^(TestBaselineSourceDateEpochStampIsFullDate|TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting|TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree)$/^baseline$' -json -coverprofile=/tmp/MAC-a89e-green.xtIsDr/cli-coverage.out | tee /tmp/MAC-a89e-green.xtIsDr/cli-tests.json
- go tool cover -func=/tmp/MAC-a89e-green.xtIsDr/gates-coverage.out
- go tool cover -func=/tmp/MAC-a89e-green.xtIsDr/cli-coverage.out
- pvg verify internal/gates/gates.go cmd/machinery/baseline.go internal/gates/regeneration_safety_test.go internal/gates/gates_test.go docs/brownfield-team-guide.md --format text
- pvg story verify-tdd --base epic/MAC-ui8a
- git diff --check epic/MAC-ui8a
- git diff --exit-code e95be63019d89b93ae527f27ae1ce43326def038 HEAD -- internal/gates/gates_test.go internal/gates/regeneration_safety_test.go
- shasum -a 256 internal/gates/gates_test.go internal/gates/regeneration_safety_test.go

Summary:
- Required gates: 37 native leaves PASS, 0 FAIL, 0 SKIP; package 5.683s, exit 0.
- Downstream baseline date, deterministic-date sources, actual subprocess baseline diagnostics: 3 native leaves PASS, 0 FAIL, 0 SKIP; package 1.470s, exit 0.
- Total 40/40 selected leaves PASS. No compiler/setup/runtime deadline failures or test warnings.
- Hard-TDD PASS: local merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890..HEAD, 5 commits checked, 0 merges skipped, no unauthorized test edits. git diff --check PASS.
- Coverage: internal/gates scoped total 9.9%, gates.VersionSkewNote 93.8%, regenCommands 100%; cmd/machinery scoped total 2.2%, newBaselineCmd 75.0%, resolveBaselineDate 84.2%. These are scoped package figures; independently built real CLI subprocess is not included in internal/gates coverage. Downstream private-path test intentionally selects baseline only.
- Test integration runs actual freshly built CLI and native filesystem unconditionally; no mocks/stubs/services/env-based omission. The existing downstream fixture has a symlink-platform fallback but this run created symlinks successfully and reports zero skips.

### Retained raw evidence
Directory /tmp/MAC-a89e-green.xtIsDr:
- gates-tests.json SHA256 9f138ecab644843d7932420be54e1c2be78853b4bb558f512e33ed9890c15382
- cli-tests.json SHA256 5945d2913263d1a197d901fcf87e80121d614b2e9ee35bf40ac9e332f14419d5
- inventory.txt SHA256 67302005140ca8aab06f698508f6616dfc469d25d35e01493b1ed6312c609b99
- gates-coverage.out SHA256 f1b45d80d3aeecd7157adbeda1a85594d452cf99a2905b1355715c40f3a6fbbb
- cli-coverage.out SHA256 8f85fe7958ee02927a94af73448d5b72dbbe90e699aeb33f05047575681b325e
- verify.txt SHA256 841ae4d70019ef05407b2c096de729ef9a86c6a39226ab76fee41a76e67b51c2
- tdd.txt contains native hard-TDD PASS output.

### pvg verify
Native output: VERIFY: FAILED (4 files scanned, 4 issues), all [stub] return empty string. Documentation is not scanned by this source scanner.
- internal/gates/gates.go:76 is VersionSkewNote's documented no-skew empty result. Full implementation collects/sorts skew and emits applicable advice. Existing tests TestOracleMissingStampIsFreshAndSilent and TestOracleCurrentStampIsSilent assert empty result; real no-debt regeneration control asserts skew disappears.
- internal/gates/gates.go:219 is readDesignOrEmpty's conditional read-failure empty sentinel; success returns actual file text.
- internal/gates/gates.go:230 is readDesignFileOrErr's conditional error sentinel; non-missing errors append an explicit gate error, success returns actual file text.
- internal/gates/gates_test.go:125 is coveringInterfaceTable's conditional no-concrete-allow-rows result; otherwise full markdown table is rendered. Independently reviewed earlier as exact false positive.
All four implementations/branches are unchanged from approved RED; none is a placeholder. Dispatcher independently inspected all three production paths and confirmed populated, unchanged conditional-return functions rather than empty implementations; this is a specific scanner false-positive disposition, not a claim that every caller's read-error policy is safe. Independent PM adjudication remains required before acceptance. No scanner changes or test-helper edits.

### AC Verification
| AC | Requirement and evidence | Status |
|---|---|---|
| 1 | gates.go regenCommands removes only baseline from the deterministic explicit generator list. Frozen VersionSkewRegenerationOnlySafeGenerators tests all individual oracle/Alloy/semantics/composition/pack families and all-with-ratchet ordered output; ratchet-only emits no command. Existing ratchet-present TestVersionSkewNoteNamesEveryApplicableCommand and generated-TLA selection remain intact. | PASS |
| 2 | RealCLI/new-offender-survives-all-advice starts with actual CLI ratchet alpha/a.go and green G4, adds real alpha/b.go import and observes exact G4 growth failure, executes EVERY advised generator, proves current oracle stamp, compares ratchet bytes unchanged and requires same offender failure afterward. | PASS |
| 3 | CLI help before invocation explains rerunning rewrites ratchet.json and may accept newly added offender files, tells users to review ratchet changes before adoption; identical substantive guidance prints only after successful publication, regardless of proposal count. Zero-rule line now says only no new baseline dependency rules proposed. RealCLI/explicit-baseline-reviews-debt-change begins already baselined, rejects alpha/b.go, deliberately invokes baseline, proves 0 need a baseline rule, two recorded offenders, unstamped ratchet and G4 green. All five semantic-guidance controls pass, including pathname bait rejection. Baseline algorithm/schema/date/confirmation semantics preserved; downstream date tests pass. | PASS |
| 4 | RealCLI/no-debt-version-skew-control obtains correct oracle advice, executes it, proves fresh current stamp/no skew and green g3,g4 without creating ratchet; negative AC2 still fails after all routine advice. Advice family controls execute deterministically. | PASS |

### Exact native leaf inventory
PASS TestG4RatchetSnapshotNote
PASS TestG4BaselineWithoutRatchetFails
PASS TestG4RatchetGreenAtSnapshot
PASS TestG4RatchetGrowthFails
PASS TestG4RatchetShrinkAndStaleEdgesNote
PASS TestRatchetRoundTrip
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/unknown_root
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edge_name
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_number
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edges_array
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_value_string
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_entry_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/trailing_value
PASS TestWriteRatchetRejectsSymlinkTarget
PASS TestVersionSkewNoteNamesEveryApplicableCommand
PASS TestVersionSkewNoteFormalCommandFollowsGeneratedTLA
PASS TestRatchetSnapshotNoteBothFormatsIsClockIndependent
PASS TestVersionSkewRegenerationOnlySafeGenerators/ratchet-only
PASS TestVersionSkewRegenerationOnlySafeGenerators/oracle
PASS TestVersionSkewRegenerationOnlySafeGenerators/alloy
PASS TestVersionSkewRegenerationOnlySafeGenerators/semantics
PASS TestVersionSkewRegenerationOnlySafeGenerators/composition
PASS TestVersionSkewRegenerationOnlySafeGenerators/pack
PASS TestVersionSkewRegenerationOnlySafeGenerators/all-with-ratchet
PASS TestRegenBaselineGuidanceSensitivity/genuine
PASS TestRegenBaselineGuidanceSensitivity/equivalent-wrapped
PASS TestRegenBaselineGuidanceSensitivity/path-bait
PASS TestRegenBaselineGuidanceSensitivity/old-rule-review-help
PASS TestRegenBaselineGuidanceSensitivity/unrelated-review
PASS TestRegenRatchetRealCLI/no-debt-version-skew-control
PASS TestRegenRatchetRealCLI/new-offender-survives-all-advice
PASS TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change
PASS TestBaselineSourceDateEpochStampIsFullDate
PASS TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting
PASS TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree/baseline/--impl_/var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/machinery-diagnostic-user-input-448099535/TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree3077046406/003_--date_2026-01-01_/var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/machinery-diagnostic-user-input-448099535/TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree3077046406/001


### Wiring, discovery, and residual limits
- Existing machinery check -> gates.VersionSkewNote -> regenCommands remains wired; existing baseline Cobra command renders help and prints guidance after PublishExpectedRooted succeeds. No new interfaces/config/product dependencies.
- Used developer and codebase-memory skills; inherited parent Verify structural evidence then independently checked project readiness and exact coverage. Graph generation 2026-09-05T23:08:53Z, no recorded gaps for gates.go, gates_test.go, baseline algorithm, CLI baseline/check/diagnostic paths. New regression test is absent from root graph; .claude worktree is excluded, so reviewed exact worktree source and tests directly. No completeness claim.
- Documentation freshness scan found the same unsafe operation in generated-file conflict advice; authorized narrow paragraph correction now separates explicit ratchet debt review.
- No full preflight or unrelated runtime suite run, as explicitly constrained; final epic gate owns those checks. Real executable fixture has oracle only; other generator families are precise command-selection tests, not claims of Alloy solver/TLA execution.
- No remote fetch/pull/push/sync, branch integration, installed binary/skills/plugins/agents changes, Docker changes, Paivot runtime dependency, or baseline algorithm/schema/date modifications. Worktree retained, committed and clean.

LEARNINGS:
- Zero new dependency-rule proposals can still mean expanded accepted offender debt; help and successful output must distinguish them.
- Explicit generators form the safe regeneration contract. A ratchet is debt policy rather than a version-stamped derivative.
- Frozen RED history exposed pathname-bait guidance assertions; approved path-stripping semantic controls now reject accidental matches without weakening real explicit-baseline proof.
- Source scanners flag valid conditional empty sentinels; retain raw findings and secure specific independent disposition instead of altering working semantics.
- User-facing merge documentation can retain unsafe command advice after the implementation is fixed; targeted freshness review found and corrected this instance under explicit scope authorization.

## nd_contract
status: delivered

### evidence
- Candidate de9745067382c5de25f05d69ea05c2808eb0b2e7, 40 PASS/0 FAIL/0 SKIP, scoped coverage and raw evidence above.
- Frozen RED unchanged; hard-TDD PASS; five-file/272-LOC scope includes explicit dispatcher authorization.

### proof
- [x] AC #1: safe deterministic complete regeneration advice.
- [x] AC #2: real offender remains G4 failure after every advised generator.
- [x] AC #3: explicit baseline supported with truthful pre-invocation and successful debt-review guidance even at zero rules.
- [x] AC #4: real no-debt positive and debt-growth negative both exercised.


### 2026-09-05T23:46:23Z ramirosalas
## PM GREEN Review — MAC-a89e — 2026-09-05
Reviewed delivered GREEN de9745067382c5de25f05d69ea05c2808eb0b2e7 against approved frozen RED e95be63019d89b93ae527f27ae1ce43326def038, current canonical four ACs, prior test-edit authorizations, dispatcher-authorized fifth documentation file and terminal Implementation Evidence. Full review restarted from TDD integrity and static checks; no earlier RED approval substituted for GREEN review.

Hard-TDD / frozen proof:
- In independent detached /tmp/MAC-a89e-pm-green.8WCpJz/checkout at de9745067382c5de25f05d69ea05c2808eb0b2e7: pvg story verify-tdd --base epic/MAC-ui8a PASS; range f24b2df3cb1e1521f97b406a7516f72bb7bc7890..HEAD, 5 commits, 0 merges skipped, no unauthorized test edits.
- git diff --exit-code e95be63019d89b93ae527f27ae1ce43326def038 HEAD -- internal/gates/gates_test.go internal/gates/regeneration_safety_test.go PASS. GREEN changes only gates.go, cmd/machinery/baseline.go and the authorized documentation paragraph; no frozen fixture or test changes.
- SHA256 gates_test.go e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; regeneration_safety_test.go 2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3, both identical to approved RED.
- git diff --check epic/MAC-ui8a PASS. Five files / 272 changed LOC within expanded documented scope and LOC ceiling.

Independent exact test commands (both exit 0):
cd /tmp/MAC-a89e-pm-green.8WCpJz/checkout && set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-pm-green.8WCpJz/gates-coverage.out | tee /tmp/MAC-a89e-pm-green.8WCpJz/gates-tests.json
cd /tmp/MAC-a89e-pm-green.8WCpJz/checkout && set -o pipefail && go test -count=1 -timeout=5m ./cmd/machinery -run '^(TestBaselineSourceDateEpochStampIsFullDate|TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting|TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree)$/^baseline$' -json -coverprofile=/tmp/MAC-a89e-pm-green.8WCpJz/cli-coverage.out | tee /tmp/MAC-a89e-pm-green.8WCpJz/cli-tests.json
Results: 37/37 gates leaves PASS, 0 FAIL, 0 SKIP, 5.285s; 3/3 downstream leaves PASS, 0 FAIL, 0 SKIP, 0.767s. Total 40/40. Exact gates inventory matches delivered names; downstream names match modulo temp nonce. No compiler, prerequisite, timeout or unexpected runtime warnings/errors.
Coverage independently reproduced: gates 9.9%, gates.VersionSkewNote 93.8%, regenCommands 100%; cmd 2.2%, newBaselineCmd 75%, resolveBaselineDate 84.2%. The separately built CLI subprocess is not in package coverage. Both profiles have identical hashes to author.
Independent raw SHA256: gates-tests.json 98982fe66cadfc781c54f813ad6d0fc1fa7f9f76b112054487b8596aa47222e1; cli-tests.json df89b4519018763f709e8897d6249b8923f4aa234415eea05bc6c6eee7a62951; gates-coverage.out f1b45d80d3aeecd7157adbeda1a85594d452cf99a2905b1355715c40f3a6fbbb; cli-coverage.out 8f85fe7958ee02927a94af73448d5b72dbbe90e699aeb33f05047575681b325e. Author's gates/CLI raw JSON and verify.txt hashes independently match delivery.

Static / implementation review:
- pvg verify over the 5 scoped paths reproduces 4 source files scanned, 4 return-empty findings. Independent precise disposition: VersionSkewNote:76 implements documented no-skew silence after collecting version skew and otherwise emits advice; readDesignOrEmpty:219 returns real read bytes on success and an empty sentinel on error; readDesignFileOrErr:230 records non-missing read errors on the Gate then returns its empty error sentinel, returning real bytes on success; coveringInterfaceTable:125 omits a Markdown table only without concrete allow rows and otherwise renders it. All four function bodies/branches unchanged from RED and populated implementations, not stubs. No scanner modification or blanket claim that every caller's read-error policy is safe.
- Production diff removes only baseline advice from regenCommands. Other family conditions, exact commands and ordering remain unchanged. Baseline changes only help, zero-proposal text and post-success guidance after PublishExpectedRooted; algorithm/schema/date/recovery behavior unchanged. No new API/config registration/security integration obligations; Go signatures are typed.
- Codebase-memory Verify used indexed project Users-ramirosalas-workspace-machinery, ready generation 2026-09-05T23:08:53Z. Exact symbol search and bidirectional trace confirm regenCommands -> VersionSkewNote -> SelectRunAndNote wiring; current command source and downstream tests were read. Coverage checked all cited paths plus docs scope. Root graph lacks new regression file and excludes .claude worktrees; exact detached committed source is authoritative for GREEN differences. No exhaustive graph-completeness claim.

AC review:
- AC1 PASS in implementation/frozen proof: no baseline/debt mutation in exact deterministic generator inventories; ratchet-only, individual oracle/Alloy/formal/pack and combined cases pass.
- AC2 PASS: real CLI begins with alpha/a.go ratchet and green G4; actual added alpha/b.go is rejected, every advised generator runs, current oracle stamp is checked, ratchet bytes remain unchanged and G4 still rejects that offender.
- AC3 runtime PASS: actual pre-invocation help warns rerun rewrites ratchet and may accept new offenders, requires review before adoption; successful publication emits the same guidance even at 0 proposals. Deliberate baseline accepts both offender files and G4 then passes without a version stamp or new confirmation mechanism. All five matcher controls pass. Documentation portion has one concrete gap below.
- AC4 PASS: no-debt real CLI control regenerates current oracle, resolves skew, keeps g3/g4 green, creates no ratchet; negative offender behavior remains blocked after advice.
- Standalone product behavior preserved: real Machinery CLI/filesystem paths with no mocks, service dependencies, skip-if-missing or Paivot runtime coupling. No full preflight, installed artifact changes, remote activity or root branch switch.
- Authorized merge-conflict paragraph correctly removes baseline from routine generation and explains separate ratchet debt review.

DOCS_STALE — sole remaining gap:
EXPECTED: AC3 calls explicit baseline a deliberate debt-acceptance operation and requires clear debt-change review guidance; routine regeneration must not cause debt acceptance. Documentation freshness is part of the deliverable.
DELIVERED: docs/brownfield-team-guide.md:451-452 still says old YYYY-MM ratchets age from the first of the month, then instructs "rerun machinery baseline to restamp" without qualifying the debt change.
GAP: this is still a stamp-only migration recommendation using a debt-accepting command. It also misstates current default behavior: resolveBaselineDate explicitly reuses the existing YYYY-MM date when no explicit date/epoch is supplied; the unchanged downstream test TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting verifies that exact behavior and passed independently. The nearby corrected merge paragraph does not repair this separate instruction.
FIX: narrowly replace this final date bullet in the already-scoped document. State legacy YYYY-MM remains supported and a default rerun retains the date; do not recommend routine restamping. If mentioning replacement, describe a deliberate reviewed debt-acceptance operation with an explicitly chosen deterministic date and review of ratchet/offender changes. No production, algorithm/schema/date, frozen test or fixture edits are required. Dispatcher must extend the existing paragraph-only authorization to this exact bullet before author editing. No new test needed for this prose correction; preserve the already passing frozen evidence.

DISCOVERED_BUG:
  title: Brownfield guide still recommends baseline as a date-restamping migration
  context: Same-story documentation freshness review found the final date-format bullet recommends rerunning baseline merely to restamp, although rerun can accept offender growth and without explicit date input retains an existing YYYY-MM date.
  affected_files: docs/brownfield-team-guide.md:451-452
  discovered_during: MAC-a89e
  triage: bounded same-story documentation correction recommended; no new issue created by this reviewer.

No other blocking gap found. Full GREEN proof above remains usable; final acceptance withheld only for the precise documentation correction.

### 2026-09-05T23:46:44Z ramirosalas
REJECTED [2026-09-05]: DOCS_STALE only; full independent GREEN review is recorded in the preceding PM comment.
EXPECTED: AC3 requires explicit baseline to remain deliberate debt acceptance with clear review guidance; routine artifact/date regeneration must not advise accepting new debt.
DELIVERED: docs/brownfield-team-guide.md:451-452 still recommends rerunning machinery baseline to restamp a legacy YYYY-MM ratchet.
GAP: this recommends a debt-accepting operation as stamp-only migration and misstates default date behavior, which reuses the existing YYYY-MM date. Independent TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting passed and confirms reuse.
FIX: dispatcher has explicitly authorized the exact additional legacy-date bullet in the already-scoped fifth document. Explain legacy YYYY-MM remains supported and ages from month start; remove routine baseline-for-restamping advice. Any deliberate baseline reference must preserve debt-review requirements and must not imply a default date upgrade. No new date behavior or broader docs ownership; no production or frozen test/fixture changes. All 40 selected tests independently pass, exact frozen hashes and verify-tdd pass; 4 scanner findings are precisely adjudicated nonstubs in the full review.
## nd_contract
status: rejected

### evidence
- GREEN de9745067382c5de25f05d69ea05c2808eb0b2e7: independent 37 gates + 3 downstream leaves PASS, 0 FAIL, 0 SKIP.
- Exact frozen RED e95be63019d89b93ae527f27ae1ce43326def038 test bytes unchanged; native verify-tdd PASS.
- Sole gap: stale legacy-date advice at docs/brownfield-team-guide.md:451-452; narrow same-file correction authorized by dispatcher.

### proof
- [x] AC #1: safe deterministic generator advice independently verified.
- [x] AC #2: real offender remains rejected after all routine advice.
- [ ] AC #3: runtime/help/output pass, but remaining legacy-date documentation must stop advising routine debt acceptance.
- [x] AC #4: positive no-debt and negative debt-growth controls pass.

### 2026-09-05T23:49:36Z ramirosalas
## Implementation Evidence (DELIVERED GREEN REWORK — MAC-a89e)

PROOF:

### Rework resolution and explicit authorization
- Prior GREEN de9745067382c5de25f05d69ea05c2808eb0b2e7 was independently reviewed with 40/40 PASS and rejected solely for DOCS_STALE: the legacy ratchet date bullet still recommended baseline merely to restamp. No production or frozen-test defect was found.
- PM rejection at 2026-09-05T23:46:44Z records dispatcher authorization for only that additional bullet in the already-scoped docs/brownfield-team-guide.md. Rework commit 4f339474e52ee9e788d0d5d071105fba385188b2 changes only those two former lines to five lines, preserving every other file and prior change.
- Corrected text: Legacy ratchet.json snapshots with YYYY-MM remain supported and age from the first of that month. Without an explicit date or SOURCE_DATE_EPOCH, a baseline rerun retains that date; no routine restamping is needed. Baseline remains a deliberate debt-acceptance operation: review ratchet and offender changes before adopting its result.
- This fixes the documented migration recommendation without introducing date behavior, new tests, confirmation mechanisms or additional source ownership. Existing date-reuse test passed again at the latest SHA.

### Commit and scope
- Frozen approved RED: e95be63019d89b93ae527f27ae1ce43326def038.
- GREEN production: bcba7900a042286b45ff46ae08bc04516ce34923.
- Final tested candidate: 4f339474e52ee9e788d0d5d071105fba385188b2 on story/MAC-a89e, worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e.
- Scope expansion explicitly authorized by dispatcher during GREEN: docs/brownfield-team-guide.md only the generated-file merge paragraph around 259–265; separate ratchet conflicts as explicit debt-policy review, preserve deliberate baseline, require review of accepted offender changes, no new merge algorithm/confirmation workflow. Five files, 279 total changed LOC against local merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890; original four production/test paths plus this one document. GREEN adds 21/removes 11 lines across three non-test files.
- Both frozen tests remain byte-identical to approved RED (git diff e95be63 HEAD -- test paths exits 0). SHA256 gates_test.go=e4f630ec2ff7684eea436881b8a55706496f348c0ae7cd8748a255a6b6d439ff; regeneration_safety_test.go=2e1801d96618aaf146621e822a342b4bfdf69cdace50b94d0585d7f1ad4f20c3. No authored tests or fixtures edited during GREEN.

### CI/Test Results
Commands run:
All shell commands use explicit cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-a89e.
- set -o pipefail && go test -count=1 -timeout=5m ./internal/gates -run 'VersionSkew|Regen|Ratchet' -json -coverprofile=/tmp/MAC-a89e-green-rework.roS7lB/gates-coverage.out | tee /tmp/MAC-a89e-green-rework.roS7lB/gates-tests.json
- set -o pipefail && go test -count=1 -timeout=5m ./cmd/machinery -run '^(TestBaselineSourceDateEpochStampIsFullDate|TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting|TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree)$/^baseline$' -json -coverprofile=/tmp/MAC-a89e-green-rework.roS7lB/cli-coverage.out | tee /tmp/MAC-a89e-green-rework.roS7lB/cli-tests.json
- go tool cover -func=/tmp/MAC-a89e-green-rework.roS7lB/gates-coverage.out
- go tool cover -func=/tmp/MAC-a89e-green-rework.roS7lB/cli-coverage.out
- pvg verify internal/gates/gates.go cmd/machinery/baseline.go internal/gates/regeneration_safety_test.go internal/gates/gates_test.go docs/brownfield-team-guide.md --format text
- pvg story verify-tdd --base epic/MAC-ui8a
- git diff --check epic/MAC-ui8a
- git diff --exit-code e95be63019d89b93ae527f27ae1ce43326def038 HEAD -- internal/gates/gates_test.go internal/gates/regeneration_safety_test.go
- shasum -a 256 internal/gates/gates_test.go internal/gates/regeneration_safety_test.go

Summary:
- Required gates: 37 native leaves PASS, 0 FAIL, 0 SKIP; package 4.899s (native package event 4.900s), exit 0.
- Downstream baseline date, deterministic-date sources, actual subprocess baseline diagnostics: 3 native leaves PASS, 0 FAIL, 0 SKIP; package 1.435s (native package event 1.439s), exit 0.
- Total 40/40 selected leaves PASS. No compiler/setup/runtime deadline failures or test warnings.
- Hard-TDD PASS: local merge-base f24b2df3cb1e1521f97b406a7516f72bb7bc7890..HEAD, 6 commits checked, 0 merges skipped, no unauthorized test edits. git diff --check PASS.
- Coverage: internal/gates scoped total 9.9%, gates.VersionSkewNote 93.8%, regenCommands 100%; cmd/machinery scoped total 2.2%, newBaselineCmd 75.0%, resolveBaselineDate 84.2%. These are scoped package figures; independently built real CLI subprocess is not included in internal/gates coverage. Downstream private-path test intentionally selects baseline only.
- Test integration runs actual freshly built CLI and native filesystem unconditionally; no mocks/stubs/services/env-based omission. The existing downstream fixture has a symlink-platform fallback but this run created symlinks successfully and reports zero skips.

### Retained raw evidence
Directory /tmp/MAC-a89e-green-rework.roS7lB:
- gates-tests.json SHA256 11dbcc06a3365fca64c726f6bf450755e6ec8d02216e9d6bf594230b932a083b
- cli-tests.json SHA256 94f7c9e37a4e21228db98238ebc138b7e391bab45383ed64673ee2f7a7cd7889
- inventory.txt SHA256 61286daa417fba0978ca70a0408a4d21d6da06953b8e67e71dc03d8c49bb366c
- gates-coverage.out SHA256 f1b45d80d3aeecd7157adbeda1a85594d452cf99a2905b1355715c40f3a6fbbb
- cli-coverage.out SHA256 8f85fe7958ee02927a94af73448d5b72dbbe90e699aeb33f05047575681b325e
- verify.txt SHA256 841ae4d70019ef05407b2c096de729ef9a86c6a39226ab76fee41a76e67b51c2
- tdd.txt contains native hard-TDD PASS output.

### pvg verify
Native output: VERIFY: FAILED (4 files scanned, 4 issues), all [stub] return empty string. Documentation is not scanned by this source scanner.
- internal/gates/gates.go:76 is VersionSkewNote's documented no-skew empty result. Full implementation collects/sorts skew and emits applicable advice. Existing tests TestOracleMissingStampIsFreshAndSilent and TestOracleCurrentStampIsSilent assert empty result; real no-debt regeneration control asserts skew disappears.
- internal/gates/gates.go:219 is readDesignOrEmpty's conditional read-failure empty sentinel; success returns actual file text.
- internal/gates/gates.go:230 is readDesignFileOrErr's conditional error sentinel; non-missing errors append an explicit gate error, success returns actual file text.
- internal/gates/gates_test.go:125 is coveringInterfaceTable's conditional no-concrete-allow-rows result; otherwise full markdown table is rendered. Independently reviewed earlier as exact false positive.
All four implementations/branches are unchanged from approved RED; none is a placeholder. Dispatcher independently inspected all three production paths and confirmed populated, unchanged conditional-return functions rather than empty implementations; this is a specific scanner false-positive disposition, not a claim that every caller's read-error policy is safe. PM GREEN review at 2026-09-05T23:46:23Z independently confirmed all four findings are populated implementations, not stubs; these exact paths and scanner output remain unchanged during rework. No scanner changes or test-helper edits.

### AC Verification
| AC | Requirement and evidence | Status |
|---|---|---|
| 1 | gates.go regenCommands removes only baseline from the deterministic explicit generator list. Frozen VersionSkewRegenerationOnlySafeGenerators tests all individual oracle/Alloy/semantics/composition/pack families and all-with-ratchet ordered output; ratchet-only emits no command. Existing ratchet-present TestVersionSkewNoteNamesEveryApplicableCommand and generated-TLA selection remain intact. | PASS |
| 2 | RealCLI/new-offender-survives-all-advice starts with actual CLI ratchet alpha/a.go and green G4, adds real alpha/b.go import and observes exact G4 growth failure, executes EVERY advised generator, proves current oracle stamp, compares ratchet bytes unchanged and requires same offender failure afterward. | PASS |
| 3 | CLI help before invocation explains rerunning rewrites ratchet.json and may accept newly added offender files, tells users to review ratchet changes before adoption; identical substantive guidance prints only after successful publication, regardless of proposal count. Zero-rule line now says only no new baseline dependency rules proposed. RealCLI/explicit-baseline-reviews-debt-change begins already baselined, rejects alpha/b.go, deliberately invokes baseline, proves 0 need a baseline rule, two recorded offenders, unstamped ratchet and G4 green. All five semantic-guidance controls pass, including pathname bait rejection. Baseline algorithm/schema/date/confirmation semantics preserved; downstream date tests pass. Authorized legacy-date bullet now states supported month precision, month-start aging and default date retention without routine restamping. | PASS |
| 4 | RealCLI/no-debt-version-skew-control obtains correct oracle advice, executes it, proves fresh current stamp/no skew and green g3,g4 without creating ratchet; negative AC2 still fails after all routine advice. Advice family controls execute deterministically. | PASS |

### Exact native leaf inventory
PASS TestG4RatchetSnapshotNote
PASS TestG4BaselineWithoutRatchetFails
PASS TestG4RatchetGreenAtSnapshot
PASS TestG4RatchetGrowthFails
PASS TestG4RatchetShrinkAndStaleEdgesNote
PASS TestRatchetRoundTrip
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/unknown_root
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/missing_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_date
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edges
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/duplicate_edge_name
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_number
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/date_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edges_array
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_value_string
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/edge_entry_null
PASS TestRatchetRejectsUnknownDuplicateAndMistypedJSON/trailing_value
PASS TestWriteRatchetRejectsSymlinkTarget
PASS TestVersionSkewNoteNamesEveryApplicableCommand
PASS TestVersionSkewNoteFormalCommandFollowsGeneratedTLA
PASS TestRatchetSnapshotNoteBothFormatsIsClockIndependent
PASS TestVersionSkewRegenerationOnlySafeGenerators/ratchet-only
PASS TestVersionSkewRegenerationOnlySafeGenerators/oracle
PASS TestVersionSkewRegenerationOnlySafeGenerators/alloy
PASS TestVersionSkewRegenerationOnlySafeGenerators/semantics
PASS TestVersionSkewRegenerationOnlySafeGenerators/composition
PASS TestVersionSkewRegenerationOnlySafeGenerators/pack
PASS TestVersionSkewRegenerationOnlySafeGenerators/all-with-ratchet
PASS TestRegenBaselineGuidanceSensitivity/genuine
PASS TestRegenBaselineGuidanceSensitivity/equivalent-wrapped
PASS TestRegenBaselineGuidanceSensitivity/path-bait
PASS TestRegenBaselineGuidanceSensitivity/old-rule-review-help
PASS TestRegenBaselineGuidanceSensitivity/unrelated-review
PASS TestRegenRatchetRealCLI/no-debt-version-skew-control
PASS TestRegenRatchetRealCLI/new-offender-survives-all-advice
PASS TestRegenRatchetRealCLI/explicit-baseline-reviews-debt-change
PASS TestBaselineSourceDateEpochStampIsFullDate
PASS TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting
PASS TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree/baseline/--impl_/var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/machinery-diagnostic-user-input-293403372/TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree1739941386/003_--date_2026-01-01_/var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/machinery-diagnostic-user-input-293403372/TestMigratedCommandDiagnosticsAreByteStableAndPrivatePathFree1739941386/001


### Wiring, discovery, and residual limits
- Existing machinery check -> gates.VersionSkewNote -> regenCommands remains wired; existing baseline Cobra command renders help and prints guidance after PublishExpectedRooted succeeds. No new interfaces/config/product dependencies.
- Used developer and codebase-memory skills; inherited parent Verify structural evidence then independently checked project readiness and exact coverage. Graph generation 2026-09-05T23:08:53Z, no recorded gaps for gates.go, gates_test.go, baseline algorithm, CLI baseline/check/diagnostic paths. New regression test is absent from root graph; .claude worktree is excluded, so reviewed exact worktree source and tests directly. No completeness claim.
- Documentation freshness scan found the same unsafe operation in generated-file conflict advice; authorized narrow paragraph correction now separates explicit ratchet debt review.
- No full preflight or unrelated runtime suite run, as explicitly constrained; final epic gate owns those checks. Real executable fixture has oracle only; other generator families are precise command-selection tests, not claims of Alloy solver/TLA execution.
- No remote fetch/pull/push/sync, branch integration, installed binary/skills/plugins/agents changes, Docker changes, Paivot runtime dependency, or baseline algorithm/schema/date modifications. Worktree retained, committed and clean.

LEARNINGS:
- Zero new dependency-rule proposals can still mean expanded accepted offender debt; help and successful output must distinguish them.
- Explicit generators form the safe regeneration contract. A ratchet is debt policy rather than a version-stamped derivative.
- Frozen RED history exposed pathname-bait guidance assertions; approved path-stripping semantic controls now reject accidental matches without weakening real explicit-baseline proof.
- Source scanners flag valid conditional empty sentinels; retain raw findings and secure specific independent disposition instead of altering working semantics.
- Initial documentation review corrected generated-file merge advice but missed the later legacy-date restamping instruction. PM found that omission; the authorized rework removes it and states existing date retention accurately. Future freshness review must cover every occurrence of the unsafe operation, including migration/date guidance.

## nd_contract
status: delivered

### evidence
- Candidate 4f339474e52ee9e788d0d5d071105fba385188b2, 40 PASS/0 FAIL/0 SKIP, scoped coverage and raw evidence above.
- Frozen RED unchanged; hard-TDD PASS; five-file/279-LOC scope includes explicit dispatcher authorization.

### proof
- [x] AC #1: safe deterministic complete regeneration advice.
- [x] AC #2: real offender remains G4 failure after every advised generator.
- [x] AC #3: explicit baseline supported with truthful pre-invocation and successful debt-review guidance even at zero rules.
- [x] AC #4: real no-debt positive and debt-growth negative both exercised.

