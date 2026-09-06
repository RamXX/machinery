---
id: MAC-lnu6
title: "Remove unsafe frozen-test formatting exemptions"
status: open
priority: 0
type: bug
labels: [hard-tdd, red-approved]
parent: MAC-ui8a
created_at: 2026-09-05T19:36:13Z
created_by: ramirosalas
updated_at: 2026-09-06T02:18:00Z
content_hash: "sha256:2ef19fb4634cec9ef1344bfb519bcc2fc3f63456000a0d88a4dcf47777dae25e"
blocks: [MAC-vx24, MAC-ou97]
assignee: dev-MAC-lnu6
follows: [MAC-a89e, MAC-p8ce]
---

## Description
## USER INTENT
Frozen tests must not be weakened under a misleading formatting exemption.

## Context (Embedded)
cmd/machinery/tokensequal.go uses strings.Fields. Different indentation or spaces within string literals can change program behavior while this comparison reports equal. The utility can remain an honest whitespace-token comparison, never semantic or hard-TDD authority. Source inspection at epic 6cb2d974 found active formatting/token-identity exemptions on all nine guidance surfaces below, not just the original template. Six example BUILD files are also subjects of adjacent attestation covers. Removing every shipped authorization is existing AC1 coverage, not a new feature.

## Ownership
Exactly 17 paths are forecast; preserve other agents' accepted changes and unrelated bytes.
- cmd/machinery/tokensequal.go: honest help/comments/equal-output wording only; preserve comparison, exit behavior and rooted read/revalidation/close custody.
- cmd/machinery/tokensequal_semantics_test.go (new): all new real CLI, semantic counterexample, utility-control and multi-surface guidance tests.
Nine guidance surfaces, restricted to unsafe frozen-test authorization and its replacement exact-byte/inventory/evidence-revision/replay policy:
- skills/machinery/references/build-md-template.md
- agents/machinery-build-writer.md
- docs/brownfield-team-guide.md
- examples/checkout-split/orders/design/BUILD.md
- examples/checkout-split/payments/design/BUILD.md
- examples/fulfillment/design/BUILD.md
- examples/go-crm/design/BUILD.md
- examples/portfolio-engine/design/BUILD.md
- examples/surreal-crm/design/BUILD.md
Six conditional evidence paths, restricted to covers hashes whose exact subject path is BUILD.md:
- examples/checkout-split/orders/design/attestations.yaml
- examples/checkout-split/payments/design/attestations.yaml
- examples/fulfillment/design/attestations.yaml
- examples/go-crm/design/attestations.yaml
- examples/portfolio-engine/design/attestations.yaml
- examples/surreal-crm/design/attestations.yaml
Canonical ownership is NOT independent PM authorization to amend frozen fixtures or evidence. Before those writes, the reviewer must explicitly authorize the exact affected guidance/evidence amendment. Refresh only the actual reviewed BUILD subject hashes AFTER independent review of each prose delta; preserve every other subject/hash, claim, schema, attestor/date/note, acceptance and historical fact. Hash recomputation does not fabricate a new review or execution; no blanket rehash or renewed acceptance. Record the reviewer decision and exact old/new BUILD hashes.

## Boundary Map
PRODUCES:
- cmd/machinery/tokensequal.go -> honest whitespace-token CLI; equality is not semantic equivalence or frozen-edit permission.
- cmd/machinery/tokensequal_semantics_test.go -> real binary semantic-risk and utility controls plus all-nine-surface policy coverage.
- Nine explicitly owned guidance files -> no formatting/token-identity exception; frozen identity is exact bytes plus inventory, amendments require explicit new evidence revision and replay.
- Six explicitly owned attestations.yaml files -> only independently reviewed changed BUILD subject hashes, preserving all other evidence.
CONSUMES:
- newTokensEqualCmd() *cobra.Command; tokensEqualRunTo(oldPath, newPath string, stdoutW, stderrW io.Writer) error; strings.Fields and existing stable-file custody in cmd/machinery/tokensequal.go. Do not add a language parser to the product.
  source: cmd/machinery/tokensequal.go at epic 6cb2d974, functions newTokensEqualCmd and tokensEqualRunTo.
- goldenBin(t *testing.T) string; runBin(t *testing.T, args ...string) (string, string, int); repoRootDir(t *testing.T) string in cmd/machinery/golden_test.go, available unchanged for building/running the actual CLI with isolated configuration.
  source: cmd/machinery/golden_test.go at epic 6cb2d974, functions goldenBin/runBin/repoRootDir.
- Existing TestTokensEqual and TestTokensEqualRejectsMutationDuringComparison in cmd/machinery/oracle_test.go remain byte-for-byte unchanged. The former asserts the token-identical prefix and exit codes; its failure-message wording alone does not require amendment.
  source: cmd/machinery/oracle_test.go at epic 6cb2d974:757-813.
- Existing dependency MAC-lnu6 -> MAC-vx24 already orders the later process story after this hardened guidance. MAC-vx24 owns replay implementation and consumes the template/agent afterward; it must not be used to defer current unsafe shipped authorizations. MAC-ou97 remains downstream. No dependency changes.
  source: shared nd MAC-vx24 BlockedBy includes MAC-lnu6; existing MAC-lnu6 Blocks includes MAC-vx24 and MAC-ou97.

### Story Acceptance Criteria
1. Remove all authorization to edit frozen RED tests based on tokens-equal; exact bytes and inventory define identity, and any amendment needs explicit new evidence revision plus replay.
2. CLI help/output/docs accurately describe whitespace-token comparison, not preserved program meaning or proof of formatting-only change.
3. Negative tests use spacing inside quoted literals and indentation-sensitive source that compare token-equal but have changed semantics; no guidance/gate treats that as approved frozen-test edit.
4. Positive genuine whitespace-only comparison utility retains documented exit behavior without claiming hard-TDD approval.
5. Real CLI calls and shipped template contract test establish user-facing behavior; no Paivot product dependency.

## Testing Requirements
Hard TDD: author only the new owned test file for RED; no existing test/golden/helper edits are pre-authorized. Independently replay and freeze expected behavioral failures and passing controls before GREEN. Explicit PM fixture/evidence authorization remains required as above.
- Actual built CLI, real temporary files, no mocks/stubs or installed binary. On unchanged production, quoted-literal spacing and indentation-sensitive input pairs must genuinely differ in meaning while yielding equal whitespace tokens; make that semantic distinction executable using service-free Go facilities/already-pinned dependencies, not a label or an arbitrary nonzero error. CLI help/equal output must not claim preserved meaning, formatting-only proof or frozen-edit approval.
- Positive genuine whitespace-only controls retain exit 0 and the token-identical: N tokens prefix; token-value/count differences retain exit 1 and NOT token-identical diagnostics; ordinary read failure remains exit 1. Preserve existing real mutation/custody tests and all old bytes, including oracle_test.go and golden_test.go.
- Table-driven contract tests read ALL nine actual shipped guidance paths (missing paths fail, no skips). Each baseline case must fail because its actual unsafe authorization remains; GREEN requires affirmative exact-byte/inventory identity and explicit new evidence revision plus replay, and absence of token-identity/formatting exemptions. Include the agent, brownfield guide and six example BUILD files, not only the template. Record the tracked-source policy scan and review any remaining hits rather than hiding them with an overbroad filter.
- Keep semantic-risk negatives separate from passing utility controls: token equality remains useful, but never authorizes a frozen edit. No false RED from a missing future API, unknown CLI flag or missing fixture.
- After independently authorized BUILD/evidence edits, run the actual built CLI check <design-dir> --gate gv for all six owned example design directories with isolated config. Show changed BUILD is stale before its exact authorized hash refresh and that afterward no stale/invalid attestation is introduced; record baseline and final outcomes. Inspect the evidence diff to prove all non-BUILD fields/hashes and acceptance/history unchanged. Preserve existing golden bytes and rerun affected real example check tests; any pre-existing unrelated failure must be attributed explicitly, not concealed by evidence rewriting.
- Focused command: go test -count=1 ./cmd/machinery -run 'TokensEqual|Frozen' -json
- Compatibility command: go test -count=1 ./cmd/machinery -run 'TestGoldenCheck' -json
Runtime classification: service-free Go/native filesystem/local CLI only; no Docker, Python, Java or Node requirement and no Paivot product dependency. Record exact selected leaves, baseline/GREEN SHAs, terminal counts, duration and raw logs. No skip-if-missing, full preflight, remote mutations or installed binary replacement.

## OUT OF SCOPE
- Full language parser or semantic equivalence proof.
- Replay protocol implementation (MAC-vx24); no weakening or delayed removal of current policy.
- Shared helper rewrites, existing test/golden edits, wholesale BUILD rewrites, arbitrary attestation rehash, new claims/acceptance/history, generated oracle changes or unrelated guidance changes.

## DIFF BUDGET
- Revised forecast: 17 files, approximately 500 changed LOC combined, superseding the original 3 files/under 300 forecast because AC1 spans nine shipped surfaces and six adjacent evidence files.
- Approximate allocation: CLI 20-50; new tests 250-330; nine narrow guidance edits 60-100; six narrow evidence deltas 20-40. Report actual additions/deletions, files and runtime; this is an estimate, not automatic permission to exceed scope or trim proof. Escalate material growth with evidence.
- One native CLI build plus bounded real CLI calls and six Gv checks; actual runtime remains to be measured.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Acceptance Criteria


## Design


## Notes
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.
SCOPE REPAIR / SOURCE-VERIFIED COMPLETE SHIPPED GUIDANCE
EVIDENCE-TRUTH HOLD FOR INDEPENDENT PM (does not block new RED authoring)
Source at epic 6cb2d974, examples/go-crm/design/attestations.yaml:99-110, records gt.conformance-test-shape and g4.zero-context under attestor Codex CRM design review, date 2026-09-03, both covering BUILD.md. Rebinding changed BUILD bytes while mechanically preserving that identity/date/note may misrepresent historical review. The canonical conditional hash-only allowance is NOT authorization where it would imply those historical reviewers covered new bytes.
Unresolved reviewer question before ANY evidence write: is the exact prose delta a truthful mechanical no-new-claim refresh, or does it require an explicitly scoped new re-attestation with truthful reviewer/date/context? PM must decide from the actual delta and preserve historical evidence; if re-attestation requires field/schema/ownership changes beyond the current hash-only boundary, return the exact amendment for canonical review first. No blanket identity renewal, forged review or automatic rehash. No product protocol change is implied. New RED tests and unchanged existing tests may proceed through normal independent authorization while this evidence decision remains held.


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


## RED phase delivery — MAC-lnu6

PROOF:
- Scope: RED ONLY. Source baseline 6cb2d974ea8aea211a5974f453cef2b5802bb11e; committed test baseline 0ea1fdc730aadac15cecc8de33bb95898ad91d60 on story/MAC-lnu6. Exactly one new file, cmd/machinery/tokensequal_semantics_test.go, 229 additions/0 deletions. Production, old tests/helpers, nine guidance surfaces, all fixtures/goldens, and six attestation files unchanged. Worktree clean.
- Command from /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-lnu6: timeout 150s go test -count=1 -timeout=120s ./cmd/machinery -run 'TokensEqual|Frozen' -json. Terminal exit 1 is intended RED. Package elapsed 1.887 seconds; run start 2026-09-05T19:05:58.112992-07:00, terminal 19:05:59.999786-07:00. Actual local CLI built by unchanged goldenBin and invoked by unchanged runBin under private configuration. No installed binary, Python, Docker, service, mock, future flag/API, environment gate or Paivot product dependency.
- Raw complete JSON log: /tmp/MAC-lnu6-red-0ea1fdc.jsonl; SHA256 51195a39cdff28688605ff9e6d7e899a919db71e6f1d6dc6763d49a356fc2d82. Test-file SHA256 1138fc41a835abf3769210bdbc2dbdb1d9d2ad6db9752b14c90bf41b9945c51f.
- Exact selected leaf inventory: 26 executed; 12 PASS, 14 expected FAIL, 0 SKIP. Native terminal test events, including enclosing parents: 13 PASS, 19 FAIL, 0 SKIP. Test package terminal FAIL is separate. Five canonical AC mapped below (100% AC mapping); Go statement coverage was not measured by this focused command and is not claimed. Full suite, TestGoldenCheck and six Gv runs not executed in this RED-only phase; example/evidence deltas do not yet exist. GREEN still requires authorized stale-before/refreshed-after Gv and compatibility evidence.

PASS leaf inventory:
- TestTokensEqual
- TestTokensEqualRejectsMutationDuringComparison
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/executable_semantic_control
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/whitespace_token_utility
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/executable_semantic_control
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/whitespace_token_utility
- TestTokensEqualRealCLIUtilityControls/whitespace_reflow
- TestTokensEqualRealCLIUtilityControls/empty_whitespace
- TestTokensEqualRealCLIUtilityControls/token_value_changed
- TestTokensEqualRealCLIUtilityControls/token_added
- TestTokensEqualRealCLIUtilityControls/token_removed
- TestTokensEqualRealCLIUtilityControls/missing_input

Expected FAIL leaf inventory and causal assertions:
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/no_semantic_or_frozen_edit_assurance: actual equality stdout says "; the change is formatting-only".
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/no_semantic_or_frozen_edit_assurance: same actual false stdout assertion.
- TestTokensEqualHelpDescribesOnlyWhitespaceTokens/tokens-equal_--help: actual help says "prove two files are formatting-only".
- TestTokensEqualHelpDescribesOnlyWhitespaceTokens/--help: actual command listing says "tokens-equal prove two files are formatting-only".
- TestTokensEqualWhitespaceSuccessDoesNotAuthorizeFrozenEdits: genuine whitespace-only equality still emits the misleading "; the change is formatting-only".
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/skills/machinery/references/build-md-template.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/agents/machinery-build-writer.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/docs/brownfield-team-guide.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/checkout-split/orders/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/checkout-split/payments/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/fulfillment/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/go-crm/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/portfolio-engine/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/surreal-crm/design/BUILD.md
  All nine guidance leaves fail FIRST on the observed active owner-sanctioned formatting-only/token-identity authorization, not missing files or merely missing replacement words. GREEN assertions additionally require exact bytes plus inventory for frozen identity, an amendment requiring explicit new evidence revision plus replay, and explicit denial of token/formatting editing exemptions.

AC-to-test mapping:
| AC | RED evidence |
| 1 | TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay: nine real surface failures and affirmative identity/revision/replay/no-exemption assertions. |
| 2 | TestTokensEqualHelpDescribesOnlyWhitespaceTokens, semantic assurance leaves, genuine whitespace assurance test, all nine docs. |
| 3 | Go types.Eval evaluates len("pay  now")=8 versus len("pay now")=7; pinned yaml.v3 successfully parses both indentation variants and pins nested user deletion permission true versus false (and opposite root permission). Both input pairs have equal whitespace fields and actual CLI equality exit0. Both assurance leaves RED. |
| 4 | RealCLIUtilityControls: reflow and empty equality exit0/token-identical count prefix; token value/add/remove exit1/NOT token-identical; actual missing file exit1/named input diagnostic. Existing real mutation/custody and utility tests PASS unchanged. Separate successful-whitespace assurance remains RED. |
| 5 | Unmodified harness builds actual local CLI and invokes help/comparison under private roots; nine actual shipped guidance paths required, missing path fatal, service-free stdlib/pinned YAML only. |

Quality verification:
- First attempted skill-prescribed pvg verify ... --format=text returned "unknown flag". pvg verify --help documents separate --format text and default test exclusion.
- Corrected command: timeout 30s pvg verify cmd/machinery/tokensequal_semantics_test.go --include-tests --format text => VERIFY: PASSED (1 files scanned, 0 issues).
- git diff --check HEAD^ HEAD => clean; git status --short => empty.
- Tracked-source scan: git grep -n -E 'tokens-equal|token-identity proof|formatting-only amendment' -- ':!cmd/machinery/tokensequal_semantics_test.go' ':!go.sum'. Reviewed every returned hit: nine guidance authorizations and production tokensequal comment remain intentionally for GREEN; README is command inventory; oracle_test references are mutation failure diagnostics; snapshot_diagnostics_contract_test references are command wiring/missing-input controls. No further authorization found by this bounded literal scan; not a claim of semantic exhaustiveness.
- codebase-memory Verify discovery: project Users-ramirosalas-workspace-machinery, generation 2026-09-05T23:58:53Z, exact 5 symbol results no more pages. Coverage checked all 13 evidence paths plus worktree scope; worktree excluded and new file missing from graph. Exact source reads supplied current worktree evidence; no graph completeness claim.
- No remote fetch/pull/push/sync/GH, installed asset mutation, or Docker operation performed.

LEARNINGS:
- Go constant evaluation and valid YAML ownership changes demonstrate semantic loss without adding runtime dependencies or relying on parser failures.
- Separating utility leaves from assurance leaves preserves comparison behavior while exposing false user-facing claims.
- Nine actual policy cases must reject the old affirmative authorization before checking replacement wording, so RED remains causally attributable.
- pvg verify accepts --format text, and --include-tests is necessary to scan this new file.

## nd_contract
status: delivered

### evidence
- RED ONLY commit 0ea1fdc730aadac15cecc8de33bb95898ad91d60, baseline source 6cb2d974ea8aea211a5974f453cef2b5802bb11e; one new test file, +229/-0.
- Focused actual CLI suite: 26 leaves, 12 PASS/14 intended FAIL/0 SKIP; package elapsed1.887s; full raw log and SHA above; pvg verify1file0issues; clean worktree.
- Canonical evidence-truth hold remains untouched: independent reviewer must decide exact prose/evidence amendment and truthful historical handling before GREEN evidence writes.

### proof
- [x] AC #1 RED: nine actual unsafe authorizations fail; exact frozen identity/evidence revision/replay assertions committed.
- [x] AC #2 RED: help, equality output and all nine actual docs expose false assurance.
- [x] AC #3 RED: executable Go quoted-space and YAML indentation counterexamples pass semantic controls and equal-token utility; assurances fail.
- [x] AC #4 RED: six actual CLI compatibility controls and two existing utility/custody tests pass; separate assurance fails.
- [x] AC #5 RED: actual locally built standalone CLI and all nine shipped paths exercised without skips.
- [ ] GREEN implementation, independent RED approval/freeze, authorized evidence amendment and six Gv/compatibility runs remain pending; no final story acceptance claimed.


## nd_contract
status: new

### evidence
- Bounded Sr PM scope repair; all five user AC preserved; status open, hard-tdd, unclaimed and existing downstream dependencies unchanged.
- Exact epic 6cb2d974 source scan found nine active shipped exemptions and six adjacent BUILD subject hash records. Root approved 17-path/~500-LOC forecast. MAC-vx24 already depends on this story; no deferral or new feature.
- Inspected cmd/machinery/tokensequal.go, oracle_test.go:757-813, golden_test.go and check.go; existing oracle tests/helper bytes remain unchanged. Codebase graph lookup/coverage was best effort, exact committed source used for findings.
- Canonical scope is not PM frozen-fixture/evidence amendment authorization. Independent reviewer approval of exact prose/evidence deltas is required before writes; only exact reviewed BUILD subject hashes may refresh afterward, never other claims, schemas, acceptance or history.
- Additional source-verified evidence-truth hold: no hash-only rebinding if it misstates historical review; independent PM must decide truthful scoped re-attestation versus mechanical no-new-claim refresh before evidence writes. This does not block new RED authoring. Canonical broader identity/schema changes remain unauthorized.
- Scoped lint PASS; cycles none; RTM 0 extracted requirements is structural only, not AC proof.
- No source/test/docs/installed assets or worktree edits; no runtime proof performed during backlog repair. Required real CLI/guidance RED controls and post-refresh gate checks remain pending.

### proof
- [ ] AC #1: all nine shipped guidance surfaces remove frozen-edit exemptions; exact bytes/inventory and evidence revision/replay required.
- [ ] AC #2: honest CLI help/output/docs without semantic or formatting-only proof claims.
- [ ] AC #3: executable quoted-literal and indentation-sensitive semantic counterexamples plus real CLI negatives.
- [ ] AC #4: genuine whitespace utility controls and prior custody/exit behavior preserved.
- [ ] AC #5: actual built CLI, all-nine-surface tests and six post-refresh Gv checks, service-free and standalone.


## nd_contract
status: new

### evidence
- Bounded Sr PM scope repair; all five user AC preserved; status open, hard-tdd, unclaimed and existing downstream dependencies unchanged.
- Exact epic 6cb2d974 source scan found nine active shipped exemptions and six adjacent BUILD subject hash records. Root approved 17-path/~500-LOC forecast. MAC-vx24 already depends on this story; no deferral or new feature.
- Inspected cmd/machinery/tokensequal.go, oracle_test.go:757-813, golden_test.go and check.go; existing oracle tests/helper bytes remain unchanged. Codebase graph lookup/coverage was best effort, exact committed source used for findings.
- Canonical scope is not PM frozen-fixture/evidence amendment authorization. Independent reviewer approval of exact prose/evidence deltas is required before writes; only exact reviewed BUILD subject hashes may refresh afterward, never other claims, schemas, acceptance or history.
- Scoped backlog lint PASS (33 issues, 0 errors/review findings), dependency cycles none; RTM PASS with 0 extracted requirements/18 stories is structural only, not AC verification. Initial CONSUMES formatting lint errors corrected; all five AC preserved.
- No source/test/docs/installed assets or worktree edits; no runtime proof performed during backlog repair. Required real CLI/guidance RED controls and post-refresh gate checks remain pending.

### proof
- [ ] AC #1: all nine shipped guidance surfaces remove frozen-edit exemptions; exact bytes/inventory and evidence revision/replay required.
- [ ] AC #2: honest CLI help/output/docs without semantic or formatting-only proof claims.
- [ ] AC #3: executable quoted-literal and indentation-sensitive semantic counterexamples plus real CLI negatives.
- [ ] AC #4: genuine whitespace utility controls and prior custody/exit behavior preserved.
- [ ] AC #5: actual built CLI, all-nine-surface tests and six post-refresh Gv checks, service-free and standalone.


## nd_contract
status: new

### evidence
- Bounded Sr PM scope repair; all five user AC preserved; status open, hard-tdd, unclaimed and existing downstream dependencies unchanged.
- Exact epic 6cb2d974 source scan found nine active shipped exemptions and six adjacent BUILD subject hash records. Root approved 17-path/~500-LOC forecast. MAC-vx24 already depends on this story; no deferral or new feature.
- Inspected cmd/machinery/tokensequal.go, oracle_test.go:757-813, golden_test.go and check.go; existing oracle tests/helper bytes remain unchanged. Codebase graph lookup/coverage was best effort, exact committed source used for findings.
- Canonical scope is not PM frozen-fixture/evidence amendment authorization. Independent reviewer approval of exact prose/evidence deltas is required before writes; only exact reviewed BUILD subject hashes may refresh afterward, never other claims, schemas, acceptance or history.
- No source/test/docs/installed assets or worktree edits; no runtime proof performed during backlog repair. Required real CLI/guidance RED controls and post-refresh gate checks remain pending.

### proof
- [ ] AC #1: all nine shipped guidance surfaces remove frozen-edit exemptions; exact bytes/inventory and evidence revision/replay required.
- [ ] AC #2: honest CLI help/output/docs without semantic or formatting-only proof claims.
- [ ] AC #3: executable quoted-literal and indentation-sensitive semantic counterexamples plus real CLI negatives.
- [ ] AC #4: genuine whitespace utility controls and prior custody/exit behavior preserved.
- [ ] AC #5: actual built CLI, all-nine-surface tests and six post-refresh Gv checks, service-free and standalone.


Historical original canonical Description (superseded only as active scope; preserved verbatim below):
> ## USER INTENT
> Frozen tests must not be weakened under a misleading formatting exemption.
> 
> ## Context (Embedded)
> cmd/machinery/tokensequal.go uses strings.Fields. Different indentation or spaces within string literals can change program behavior while this comparison reports equal. BUILD guidance currently permits frozen-test edits using tokens-equal. The utility can remain as an honest whitespace-token comparison, never semantic or hard-TDD authority.
> 
> ## Ownership
> Own cmd/machinery/tokensequal.go, cmd/machinery/tokensequal_semantics_test.go, skills/machinery/references/build-md-template.md. Preserve other agents' edits; process story consumes template afterward.
> 
> ## Boundary Map
> PRODUCES:
> - cmd/machinery/tokensequal.go -> honest whitespace-token comparison CLI
> - cmd/machinery/tokensequal_semantics_test.go -> semantic-risk regression cases
> - skills/machinery/references/build-md-template.md -> exact-byte frozen-test guidance
> CONSUMES:
> - Existing CLI implementation.
>   source: strings.Fields comparison in cmd/machinery/tokensequal.go; no semantic parser.
> 
> ### Story Acceptance Criteria
> 1. Remove all authorization to edit frozen RED tests based on tokens-equal; exact bytes and inventory define identity, and any amendment needs explicit new evidence revision plus replay.
> 2. CLI help/output/docs accurately describe whitespace-token comparison, not preserved program meaning or proof of formatting-only change.
> 3. Negative tests use spacing inside quoted literals and indentation-sensitive source that compare token-equal but have changed semantics; no guidance/gate treats that as approved frozen-test edit.
> 4. Positive genuine whitespace-only comparison utility retains documented exit behavior without claiming hard-TDD approval.
> 5. Real CLI calls and shipped template contract test establish user-facing behavior; no Paivot product dependency.
> 
> ## Testing Requirements
> Hard TDD RED author updates obsolete tests/guidance assertions before GREEN; expected behavioral failures and passing control, independent replay. Integration tests: MANDATORY (no mocks); real temporary files and actual binary. Run go test ./cmd/machinery -run 'TokensEqual|Frozen'. No skip-if-missing. No full preflight, pushes, remote mutations or installed binary replacement.
> 
> ## OUT OF SCOPE
> - Full language parser or semantic equivalence proof: not promised by this utility.
> - Replay protocol implementation belongs to the process story.
> 
> ## DIFF BUDGET
> - ~3 files, under 300 changed LOC.
> 
> ## MANDATORY SKILLS
> - developer; codebase-memory; pm_acceptor.
> 
> ## nd_contract
> status: new
> 
> ### evidence
> - Created 2026-09-05 from architecture source finding.
> 
> ### proof
> - [ ] AC #1: frozen guidance rejects exemption.
> - [ ] AC #2: honest help/output.
> - [ ] AC #3: semantic counterexamples.
> - [ ] AC #4: utility compatibility.
> - [ ] AC #5: standalone real CLI.
> 

## History
- 2026-09-05T19:36:14Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:17Z dep_added: blocks MAC-ou97
- 2026-09-06T02:01:23Z status: open -> in_progress
- 2026-09-06T02:01:23Z auto-follows: linked to predecessor MAC-a89e
- 2026-09-06T02:01:23Z claimed by dev-MAC-lnu6
- 2026-09-06T02:07:57Z status: in_progress -> in_progress
- 2026-09-06T02:07:57Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-06T02:16:36Z status: in_progress -> open

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]
- Follows: [[MAC-a89e]], [[MAC-p8ce]]

## Comments

### 2026-09-06T01:12:51Z ramirosalas
## nd_contract
status: new

### evidence
- Bounded Sr PM scope repair; all five user AC preserved; status open, hard-tdd, unclaimed and existing downstream dependencies unchanged.
- Exact epic 6cb2d974 source scan found nine active shipped exemptions and six adjacent BUILD subject hash records. Root approved 17-path/~500-LOC forecast. MAC-vx24 already depends on this story; no deferral or new feature.
- Inspected cmd/machinery/tokensequal.go, oracle_test.go:757-813, golden_test.go and check.go; existing oracle tests/helper bytes remain unchanged. Codebase graph lookup/coverage was best effort, exact committed source used for findings.
- Canonical scope is not PM frozen-fixture/evidence amendment authorization. Independent reviewer approval of exact prose/evidence deltas is required before writes; only exact reviewed BUILD subject hashes may refresh afterward, never other claims, schemas, acceptance or history.
- No source/test/docs/installed assets or worktree edits; no runtime proof performed during backlog repair. Required real CLI/guidance RED controls and post-refresh gate checks remain pending.

### proof
- [ ] AC #1: all nine shipped guidance surfaces remove frozen-edit exemptions; exact bytes/inventory and evidence revision/replay required.
- [ ] AC #2: honest CLI help/output/docs without semantic or formatting-only proof claims.
- [ ] AC #3: executable quoted-literal and indentation-sensitive semantic counterexamples plus real CLI negatives.
- [ ] AC #4: genuine whitespace utility controls and prior custody/exit behavior preserved.
- [ ] AC #5: actual built CLI, all-nine-surface tests and six post-refresh Gv checks, service-free and standalone.


### 2026-09-06T01:14:10Z ramirosalas
## nd_contract
status: new

### evidence
- Bounded Sr PM scope repair; all five user AC preserved; status open, hard-tdd, unclaimed and existing downstream dependencies unchanged.
- Exact epic 6cb2d974 source scan found nine active shipped exemptions and six adjacent BUILD subject hash records. Root approved 17-path/~500-LOC forecast. MAC-vx24 already depends on this story; no deferral or new feature.
- Inspected cmd/machinery/tokensequal.go, oracle_test.go:757-813, golden_test.go and check.go; existing oracle tests/helper bytes remain unchanged. Codebase graph lookup/coverage was best effort, exact committed source used for findings.
- Canonical scope is not PM frozen-fixture/evidence amendment authorization. Independent reviewer approval of exact prose/evidence deltas is required before writes; only exact reviewed BUILD subject hashes may refresh afterward, never other claims, schemas, acceptance or history.
- Scoped backlog lint PASS (33 issues, 0 errors/review findings), dependency cycles none; RTM PASS with 0 extracted requirements/18 stories is structural only, not AC verification. Initial CONSUMES formatting lint errors corrected; all five AC preserved.
- No source/test/docs/installed assets or worktree edits; no runtime proof performed during backlog repair. Required real CLI/guidance RED controls and post-refresh gate checks remain pending.

### proof
- [ ] AC #1: all nine shipped guidance surfaces remove frozen-edit exemptions; exact bytes/inventory and evidence revision/replay required.
- [ ] AC #2: honest CLI help/output/docs without semantic or formatting-only proof claims.
- [ ] AC #3: executable quoted-literal and indentation-sensitive semantic counterexamples plus real CLI negatives.
- [ ] AC #4: genuine whitespace utility controls and prior custody/exit behavior preserved.
- [ ] AC #5: actual built CLI, all-nine-surface tests and six post-refresh Gv checks, service-free and standalone.


### 2026-09-06T01:18:00Z ramirosalas
EVIDENCE-TRUTH HOLD FOR INDEPENDENT PM (does not block new RED authoring)
Source at epic 6cb2d974, examples/go-crm/design/attestations.yaml:99-110, records gt.conformance-test-shape and g4.zero-context under attestor Codex CRM design review, date 2026-09-03, both covering BUILD.md. Rebinding changed BUILD bytes while mechanically preserving that identity/date/note may misrepresent historical review. The canonical conditional hash-only allowance is NOT authorization where it would imply those historical reviewers covered new bytes.
Unresolved reviewer question before ANY evidence write: is the exact prose delta a truthful mechanical no-new-claim refresh, or does it require an explicitly scoped new re-attestation with truthful reviewer/date/context? PM must decide from the actual delta and preserve historical evidence; if re-attestation requires field/schema/ownership changes beyond the current hash-only boundary, return the exact amendment for canonical review first. No blanket identity renewal, forged review or automatic rehash. No product protocol change is implied. New RED tests and unchanged existing tests may proceed through normal independent authorization while this evidence decision remains held.

## nd_contract
status: new

### evidence
- Bounded Sr PM scope repair; all five user AC preserved; status open, hard-tdd, unclaimed and existing downstream dependencies unchanged.
- Exact epic 6cb2d974 source scan found nine active shipped exemptions and six adjacent BUILD subject hash records. Root approved 17-path/~500-LOC forecast. MAC-vx24 already depends on this story; no deferral or new feature.
- Inspected cmd/machinery/tokensequal.go, oracle_test.go:757-813, golden_test.go and check.go; existing oracle tests/helper bytes remain unchanged. Codebase graph lookup/coverage was best effort, exact committed source used for findings.
- Canonical scope is not PM frozen-fixture/evidence amendment authorization. Independent reviewer approval of exact prose/evidence deltas is required before writes; only exact reviewed BUILD subject hashes may refresh afterward, never other claims, schemas, acceptance or history.
- Additional source-verified evidence-truth hold: no hash-only rebinding if it misstates historical review; independent PM must decide truthful scoped re-attestation versus mechanical no-new-claim refresh before evidence writes. This does not block new RED authoring. Canonical broader identity/schema changes remain unauthorized.
- Scoped lint PASS; cycles none; RTM 0 extracted requirements is structural only, not AC proof.
- No source/test/docs/installed assets or worktree edits; no runtime proof performed during backlog repair. Required real CLI/guidance RED controls and post-refresh gate checks remain pending.

### proof
- [ ] AC #1: all nine shipped guidance surfaces remove frozen-edit exemptions; exact bytes/inventory and evidence revision/replay required.
- [ ] AC #2: honest CLI help/output/docs without semantic or formatting-only proof claims.
- [ ] AC #3: executable quoted-literal and indentation-sensitive semantic counterexamples plus real CLI negatives.
- [ ] AC #4: genuine whitespace utility controls and prior custody/exit behavior preserved.
- [ ] AC #5: actual built CLI, all-nine-surface tests and six post-refresh Gv checks, service-free and standalone.


### 2026-09-06T02:09:38Z ramirosalas
## Implementation Evidence — MAC-lnu6 RED ONLY

PROOF:

### CI/Test Results
Commands run:
- cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-lnu6 && timeout 150s go test -count=1 -timeout=120s ./cmd/machinery -run 'TokensEqual|Frozen' -json
- timeout 30s pvg verify cmd/machinery/tokensequal_semantics_test.go --include-tests --format text
- git diff --check HEAD^ HEAD
- git status --short
Summary: 26 executed leaves: 12 PASS, 14 intended RED FAIL, 0 SKIP; native terminal test events including enclosing parents: 13 PASS, 19 FAIL, 0 SKIP. Package terminal exit1 elapsed1.887s. Actual local CLI builds successfully via unchanged goldenBin; genuine utility and existing mutation/custody controls PASS. All failures are intended public assurance/policy assertions, not infrastructure or compile errors.
Coverage: five of five canonical AC mapped (100% AC mapping); Go statement coverage not measured. Full suite, TestGoldenCheck and six Gv checks not run in RED; required GREEN stale-before/refreshed-after evidence and compatibility remain pending.

### Commit
- Branch: story/MAC-lnu6
- SHA: 0ea1fdc730aadac15cecc8de33bb95898ad91d60
- Baseline source SHA: 6cb2d974ea8aea211a5974f453cef2b5802bb11e
- Exactly one added file cmd/machinery/tokensequal_semantics_test.go; +229/-0. No production, prior test/helper, documentation, fixture, golden or attestation changes.
- New test SHA256: 1138fc41a835abf3769210bdbc2dbdb1d9d2ad6db9752b14c90bf41b9945c51f
- Raw log: /tmp/MAC-lnu6-red-0ea1fdc.jsonl; SHA256 51195a39cdff28688605ff9e6d7e899a919db71e6f1d6dc6763d49a356fc2d82
- Log start2026-09-05T19:05:58.112992-07:00; terminal19:05:59.999786-07:00.

### Exact native leaf inventory
PASS:
- TestTokensEqual
- TestTokensEqualRejectsMutationDuringComparison
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/executable_semantic_control
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/whitespace_token_utility
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/executable_semantic_control
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/whitespace_token_utility
- TestTokensEqualRealCLIUtilityControls/whitespace_reflow
- TestTokensEqualRealCLIUtilityControls/empty_whitespace
- TestTokensEqualRealCLIUtilityControls/token_value_changed
- TestTokensEqualRealCLIUtilityControls/token_added
- TestTokensEqualRealCLIUtilityControls/token_removed
- TestTokensEqualRealCLIUtilityControls/missing_input

Expected RED FAIL:
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/no_semantic_or_frozen_edit_assurance
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/no_semantic_or_frozen_edit_assurance
- TestTokensEqualWhitespaceSuccessDoesNotAuthorizeFrozenEdits
  Cause for these three: actual equality stdout claims "; the change is formatting-only".
- TestTokensEqualHelpDescribesOnlyWhitespaceTokens/tokens-equal_--help
- TestTokensEqualHelpDescribesOnlyWhitespaceTokens/--help
  Cause: actual help and command listing claim "prove two files are formatting-only".
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/skills/machinery/references/build-md-template.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/agents/machinery-build-writer.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/docs/brownfield-team-guide.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/checkout-split/orders/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/checkout-split/payments/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/fulfillment/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/go-crm/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/portfolio-engine/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/surreal-crm/design/BUILD.md
  Cause on each of these nine: actual active owner-sanctioned formatting-only/token-identity amendment authorization. Each fails on that real authorization before replacement wording assertions; all files exist/read. GREEN must affirm exact bytes/inventory identity, explicit new evidence revision/replay, and no formatting/token exemption.

### AC Verification
| AC # | Test location and outcome |
| 1 | TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay: all nine actual unsafe authorizations RED; affirmative exact identity/revision/replay/no-exemption requirements committed. |
| 2 | TestTokensEqualHelpDescribesOnlyWhitespaceTokens, semantic assurance leaves, whitespace success assurance, all nine policy cases expose actual false help/output/docs. |
| 3 | Go types.Eval executes len("pay  now")=8 versus len("pay now")=7. Pinned yaml.v3 successfully decodes valid indentation variants with nested user deletion true versus false and opposite root permission. Both pairs have equal whitespace tokens and actual CLI equality exit0. Semantic and utility controls PASS; assurance leaves RED. |
| 4 | Six real CLI controls pin reflow/empty equality exit0 and exact token-identical count prefix, different token/add/remove exit1 with NOT token-identical, real missing input exit1/named diagnostic. Existing utility and mutation tests PASS untouched. Separate assurance test RED. |
| 5 | Actual locally built CLI and all nine real shipped paths exercised using unchanged harness/private configuration; no mocks/skips/future API/installed binary/Paivot product dependency. |

### pvg verify
- Skill-prescribed --format=text was rejected as unknown flag. Read pvg verify --help; corrected --format text and explicit --include-tests.
- VERIFY: PASSED (1 files scanned, 0 issues).
- git diff --check clean, worktree clean.
- First verify-delivery failed metadata shape: earlier Comments contracts followed append-notes and remained authoritative. This terminal full-proof comment preserves history and corrects the final contract; delivered label already set by successful canonical pvg story deliver MAC-lnu6.
- Bounded tracked-source scan reviewed all hits from git grep -n -E 'tokens-equal|token-identity proof|formatting-only amendment'. Nine guidance authorizations and production header await GREEN; remaining README command inventory and existing diagnostic test references are not authorizations. No semantic-exhaustiveness claim.
- Graph Verify discovery: project Users-ramirosalas-workspace-machinery generation2026-09-05T23:58:53Z; five exact symbol results, no more pages. Coverage of13 paths plus worktree identified excluded worktree/new missing test. Exact source fallback used; graph completeness not claimed.

LEARNINGS:
- Go constant evaluation and valid YAML ownership changes prove semantic differences using existing dependencies.
- Independent utility and assurance leaves retain useful comparison behavior while exposing misleading claims.
- Nine actual guidance cases report affirmative exemptions first, keeping baseline failures causal.
- pvg verify requires --format text and --include-tests; old terminal Comments can outrank append-notes contracts.

## nd_contract
status: delivered

### evidence
- RED-only SHA 0ea1fdc730aadac15cecc8de33bb95898ad91d60; baseline6cb2d974ea8aea211a5974f453cef2b5802bb11e; only new test+229/-0.
- Focused suite26 leaves12PASS/14 intendedFAIL/0SKIP; package1.887s; raw log and hashes above; pvg verify1file0issues; no code/guidance/evidence edits beyond new tests.
- Independent RED review/freeze and GREEN remain pending. Canonical historical-evidence hold unchanged; PM must authorize exact guidance/evidence deltas before later evidence writes.

### proof
- [x] AC #1 RED: nine real policy failures; exact bytes/inventory and explicit evidence revision/replay requirements committed.
- [x] AC #2 RED: actual help/output/docs assurance failures.
- [x] AC #3 RED: executable quoted-spacing and indentation semantic differences; passing equality utility and failing assurance.
- [x] AC #4 RED: real comparison/exit/read-error controls plus old mutation/utility tests pass; assurance remains RED.
- [x] AC #5 RED: actual local standalone CLI, all nine real guidance paths, no skips.
- [ ] GREEN production/guidance fix, independent acceptance, authorized evidence amendments and six Gv/compatibility proofs not yet performed.


### 2026-09-06T02:09:58Z ramirosalas
## Implementation Evidence


PROOF:

### CI/Test Results
Commands run:
- cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-lnu6 && timeout 150s go test -count=1 -timeout=120s ./cmd/machinery -run 'TokensEqual|Frozen' -json
- timeout 30s pvg verify cmd/machinery/tokensequal_semantics_test.go --include-tests --format text
- git diff --check HEAD^ HEAD
- git status --short
Summary: 26 executed leaves: 12 PASS, 14 intended RED FAIL, 0 SKIP; native terminal test events including enclosing parents: 13 PASS, 19 FAIL, 0 SKIP. Package terminal exit1 elapsed1.887s. Actual local CLI builds successfully via unchanged goldenBin; genuine utility and existing mutation/custody controls PASS. All failures are intended public assurance/policy assertions, not infrastructure or compile errors.
Coverage: five of five canonical AC mapped (100% AC mapping); Go statement coverage not measured. Full suite, TestGoldenCheck and six Gv checks not run in RED; required GREEN stale-before/refreshed-after evidence and compatibility remain pending.

### Commit
- Branch: story/MAC-lnu6
- SHA: 0ea1fdc730aadac15cecc8de33bb95898ad91d60
- Baseline source SHA: 6cb2d974ea8aea211a5974f453cef2b5802bb11e
- Exactly one added file cmd/machinery/tokensequal_semantics_test.go; +229/-0. No production, prior test/helper, documentation, fixture, golden or attestation changes.
- New test SHA256: 1138fc41a835abf3769210bdbc2dbdb1d9d2ad6db9752b14c90bf41b9945c51f
- Raw log: /tmp/MAC-lnu6-red-0ea1fdc.jsonl; SHA256 51195a39cdff28688605ff9e6d7e899a919db71e6f1d6dc6763d49a356fc2d82
- Log start2026-09-05T19:05:58.112992-07:00; terminal19:05:59.999786-07:00.

### Exact native leaf inventory
PASS:
- TestTokensEqual
- TestTokensEqualRejectsMutationDuringComparison
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/executable_semantic_control
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/whitespace_token_utility
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/executable_semantic_control
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/whitespace_token_utility
- TestTokensEqualRealCLIUtilityControls/whitespace_reflow
- TestTokensEqualRealCLIUtilityControls/empty_whitespace
- TestTokensEqualRealCLIUtilityControls/token_value_changed
- TestTokensEqualRealCLIUtilityControls/token_added
- TestTokensEqualRealCLIUtilityControls/token_removed
- TestTokensEqualRealCLIUtilityControls/missing_input

Expected RED FAIL:
- TestTokensEqualSemanticCounterexamples/quoted_literal_spacing/no_semantic_or_frozen_edit_assurance
- TestTokensEqualSemanticCounterexamples/indentation_changes_permission_owner/no_semantic_or_frozen_edit_assurance
- TestTokensEqualWhitespaceSuccessDoesNotAuthorizeFrozenEdits
  Cause for these three: actual equality stdout claims "; the change is formatting-only".
- TestTokensEqualHelpDescribesOnlyWhitespaceTokens/tokens-equal_--help
- TestTokensEqualHelpDescribesOnlyWhitespaceTokens/--help
  Cause: actual help and command listing claim "prove two files are formatting-only".
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/skills/machinery/references/build-md-template.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/agents/machinery-build-writer.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/docs/brownfield-team-guide.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/checkout-split/orders/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/checkout-split/payments/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/fulfillment/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/go-crm/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/portfolio-engine/design/BUILD.md
- TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay/examples/surreal-crm/design/BUILD.md
  Cause on each of these nine: actual active owner-sanctioned formatting-only/token-identity amendment authorization. Each fails on that real authorization before replacement wording assertions; all files exist/read. GREEN must affirm exact bytes/inventory identity, explicit new evidence revision/replay, and no formatting/token exemption.

### AC Verification
| AC # | Test location and outcome |
| 1 | TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay: all nine actual unsafe authorizations RED; affirmative exact identity/revision/replay/no-exemption requirements committed. |
| 2 | TestTokensEqualHelpDescribesOnlyWhitespaceTokens, semantic assurance leaves, whitespace success assurance, all nine policy cases expose actual false help/output/docs. |
| 3 | Go types.Eval executes len("pay  now")=8 versus len("pay now")=7. Pinned yaml.v3 successfully decodes valid indentation variants with nested user deletion true versus false and opposite root permission. Both pairs have equal whitespace tokens and actual CLI equality exit0. Semantic and utility controls PASS; assurance leaves RED. |
| 4 | Six real CLI controls pin reflow/empty equality exit0 and exact token-identical count prefix, different token/add/remove exit1 with NOT token-identical, real missing input exit1/named diagnostic. Existing utility and mutation tests PASS untouched. Separate assurance test RED. |
| 5 | Actual locally built CLI and all nine real shipped paths exercised using unchanged harness/private configuration; no mocks/skips/future API/installed binary/Paivot product dependency. |

### pvg verify
- Skill-prescribed --format=text was rejected as unknown flag. Read pvg verify --help; corrected --format text and explicit --include-tests.
- VERIFY: PASSED (1 files scanned, 0 issues).
- git diff --check clean, worktree clean.
- First verify-delivery failed metadata shape: earlier Comments contracts followed append-notes and remained authoritative. This terminal full-proof comment preserves history and corrects the final contract; delivered label already set by successful canonical pvg story deliver MAC-lnu6.
- Bounded tracked-source scan reviewed all hits from git grep -n -E 'tokens-equal|token-identity proof|formatting-only amendment'. Nine guidance authorizations and production header await GREEN; remaining README command inventory and existing diagnostic test references are not authorizations. No semantic-exhaustiveness claim.
- Graph Verify discovery: project Users-ramirosalas-workspace-machinery generation2026-09-05T23:58:53Z; five exact symbol results, no more pages. Coverage of13 paths plus worktree identified excluded worktree/new missing test. Exact source fallback used; graph completeness not claimed.

LEARNINGS:
- Go constant evaluation and valid YAML ownership changes prove semantic differences using existing dependencies.
- Independent utility and assurance leaves retain useful comparison behavior while exposing misleading claims.
- Nine actual guidance cases report affirmative exemptions first, keeping baseline failures causal.
- pvg verify requires --format text and --include-tests; old terminal Comments can outrank append-notes contracts.

## nd_contract
status: delivered

### evidence
- RED-only SHA 0ea1fdc730aadac15cecc8de33bb95898ad91d60; baseline6cb2d974ea8aea211a5974f453cef2b5802bb11e; only new test+229/-0.
- Focused suite26 leaves12PASS/14 intendedFAIL/0SKIP; package1.887s; raw log and hashes above; pvg verify1file0issues; no code/guidance/evidence edits beyond new tests.
- Independent RED review/freeze and GREEN remain pending. Canonical historical-evidence hold unchanged; PM must authorize exact guidance/evidence deltas before later evidence writes.

### proof
- [x] AC #1 RED: nine real policy failures; exact bytes/inventory and explicit evidence revision/replay requirements committed.
- [x] AC #2 RED: actual help/output/docs assurance failures.
- [x] AC #3 RED: executable quoted-spacing and indentation semantic differences; passing equality utility and failing assurance.
- [x] AC #4 RED: real comparison/exit/read-error controls plus old mutation/utility tests pass; assurance remains RED.
- [x] AC #5 RED: actual local standalone CLI, all nine real guidance paths, no skips.
- [ ] GREEN production/guidance fix, independent acceptance, authorized evidence amendments and six Gv/compatibility proofs not yet performed.

### 2026-09-06T02:18:00Z ramirosalas
## PM Decision — independent RED review MAC-lnu6
RED APPROVED [2026-09-05]. Conditional review question answered YES: the unchanged committed tests, together with all explicitly required GREEN source/evidence review and runtime proof, establish all five canonical AC. This approves RED only, not implementation acceptance, closure, fixture/evidence writes, or historical hash rebinding.

### Independent evidence
- Read complete shared canonical MAC-lnu6 with pvg issues show --json; active Description and five Story Acceptance Criteria govern, historical scope retained only as history. Delivered label and final developer contract were present. pvg story verify-delivery passed 9/9 shape checks; that is metadata evidence, not code proof.
- Reviewed local story SHA 0ea1fdc730aadac15cecc8de33bb95898ad91d60 relative to epic 6cb2d974ea8aea211a5974f453cef2b5802bb11e. Commit subject carries tdd-red. Diff is exactly one new cmd/machinery/tokensequal_semantics_test.go, +229/-0. SHA256 1138fc41a835abf3769210bdbc2dbdb1d9d2ad6db9752b14c90bf41b9945c51f. Existing tests/helpers, product, guidance, golden and attestation bytes unchanged; git diff --check clean.
- Graph Verify lookup: Users-ramirosalas-workspace-machinery generation 2026-09-05T23:58:53Z; five relevant symbol results, no remaining pages. Coverage checked thirteen evidence paths and detached worktree scope; new test missing from graph, .claude excluded. Exact committed source supplied current review; no graph completeness claim.
- PM-owned DETACHED proof checkout retained: /Users/ramirosalas/workspace/machinery/.claude/worktrees/pm-MAC-lnu6-red at 0ea1fdc730aadac15cecc8de33bb95898ad91d60. Clean after replay.
- Independent native command: cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/pm-MAC-lnu6-red && timeout 150s go test -count=1 -timeout=120s ./cmd/machinery -run 'TokensEqual|Frozen' -json
- Terminal exit 1, package duration 1.611s, command wall 3.380s; terminal timestamp 2026-09-05T19:15:47.682475-07:00. Raw /tmp/MAC-lnu6-pm-red-0ea1fdc.jsonl SHA256 70c0718f4fd5dde578ab4c541a1f4bdca1ff2930f54a24b3f5798a5629d03d61. Author raw SHA256 independently matched 51195a39cdff28688605ff9e6d7e899a919db71e6f1d6dc6763d49a356fc2d82.
- Exact leaf inventory matches the developer terminal proof: 26 leaves, 12 PASS / 14 intended FAIL / 0 SKIP. Native parent-inclusive test terminal events 13 PASS / 19 FAIL / 0 SKIP. PASS: old TestTokensEqual and TestTokensEqualRejectsMutationDuringComparison; both semantic-counterexample executable_semantic_control and whitespace_token_utility leaves; six RealCLIUtilityControls (whitespace_reflow, empty_whitespace, token_value_changed, token_added, token_removed, missing_input). FAIL: both semantic no_semantic_or_frozen_edit_assurance leaves; both HelpDescribesOnlyWhitespaceTokens command/root help leaves; WhitespaceSuccessDoesNotAuthorizeFrozenEdits; all nine FrozenGuidanceRequiresExactIdentityAndEvidenceReplay path leaves.
- Five CLI/help failures are causal actual false formatting-only assurances. Nine guidance failures identify each actual owner-sanctioned token-identity formatting amendment, before testing new wording. No build/infrastructure failure or skip caused RED. Go statement coverage not measured and not claimed; AC mapping is five/five.
- Static review found concrete typed Go tests/helpers, no stub, no skip, no environment-gated omission, no mocks, future API or missing fixture workaround. Actual local goldenBin/runBin builds current checkout and runs the binary with existing private-config harness. Existing genuine read-mutation/custody test remains immutable.
- Bounded exact-source scan: git grep -n -E 'tokens-equal|token-identity proof|formatting-only amendment' HEAD -- ':!cmd/machinery/tokensequal_semantics_test.go' ':!go.sum'. Reviewed all returned hits: nine unsafe guidance surfaces plus tokensequal.go comment/help remain for GREEN; README command inventory and oracle/snapshot diagnostic test references are not authorizations. This literal scan is not exhaustive natural-language validation.

### Assertion adequacy and limits
- AC1: all nine real paths are required/read with missing-file fatal, unsafe shipped phrases rejected first, and separate affirmative exact-byte/inventory identity, explicit new evidence revision/replay, and no formatting/token exemption patterns required. Both removal and replacement obligations are present.
- AC2: both real command and root help require whitespace/token description and reject current false assurances; equal output checked on both semantic-risk pairs and a benign utility pair. Source comments and all changed user-facing prose must still be reviewed directly at GREEN.
- AC3: Go types.Eval executes valid quoted expressions with len 8 -> 7; pinned yaml.v3 decodes valid indentation variants and verifies complete permission ownership distinction (nested user true -> false with opposite root permission). Both actual CLI calls still return equal-token exit 0, proving equality does not imply semantic preservation.
- AC4: real whitespace reflow and empty/whitespace retain exit 0 and count prefix; token value/add/remove retain exit 1 and NOT token-identical; real missing input retains exit 1 and named diagnostic without equality. Existing mutation/custody behavior passes unchanged.
- AC5: actual built standalone CLI, real temporary files and all nine shipped guidance cases execute service-free using Go/native facilities and already-pinned YAML; no Paivot product dependency introduced.
- Prose regular expressions are bounded lexical checks, not natural-language semantic validators: alternate assurances could evade finite forbidden phrases, and contradictory surrounding prose could coexist with affirmative matches. They do not independently certify arbitrary rewritten documentation. The canonical independent exact-delta review and tracked-source scan remain mandatory GREEN evidence and must reject such bypasses, contradictions or misleading claims. No AC is waived. This combined bar is adequate for this narrowly scoped wording repair; passing regexes alone cannot justify GREEN acceptance.

### GREEN requirements still held
- Preserve the frozen RED file and every pre-existing test/helper/golden byte. No TEST-EDIT AUTHORIZED marker or general test-repair permission is granted.
- BEFORE writing the nine guidance/six evidence paths, present exact proposed prose/evidence deltas for independent PM authorization. For all six actual BUILD covers, explicitly decide truthful mechanical no-new-claim refresh versus scoped new re-attestation; RED approval supplies no such decision.
- Never hash-rebind revised BUILD bytes under old historical reviewer/date/claims where doing so misstates what was reviewed. If truthful re-attestation needs broader identity/date/schema/ownership edits, return the exact amendment for canonical review first. Current broader changes remain unauthorized.
- Record exact old/new BUILD hashes and authorized scope. Run six actual local CLI check <design-dir> --gate gv with isolated config for baseline, changed-BUILD stale-before-refresh, and final after any authorized refresh; attribute actual outcomes. Review evidence diff for all non-BUILD subjects/hashes and every other field/history unchanged.
- Run the immutable focused suite GREEN, affected TestGoldenCheck with unchanged goldens, review all remaining policy scan hits and exact CLI/docs changes. Required GREEN evidence is pending, not inferred from current RED or metadata gates.
- No remote, main/epic/source/history/installed-assets mutation or other agent cleanup performed. Root remains 497419ab4512fcff765cd5feb27aed4c67b5608d.

## nd_contract
status: new

### evidence
- Independent RED approval at frozen SHA 0ea1fdc730aadac15cecc8de33bb95898ad91d60, replay 26 leaves 12 PASS / 14 intended FAIL / 0 SKIP, 1.611s package, raw log/hash above.
- Canonical pvg story approve-red MAC-lnu6 succeeded against shared nd vault. Read-back status open, labels hard-tdd and red-approved, delivered removed, assignee dev-MAC-lnu6 retained. This final comment corrects the older delivered contract still appearing last after transition.
- Exact prose/evidence amendment and historical binding decisions remain pending; no write or test-edit authorization from this RED decision.

### proof
- [x] AC #1 RED bar: all nine actual unsafe authorizations fail, replacement policy required; direct GREEN policy review remains mandatory.
- [x] AC #2 RED bar: actual CLI help/output false assurances fail with bounded lexical checks plus required GREEN source review.
- [x] AC #3 RED bar: executable semantic distinction and real equal-token calls independently verified.
- [x] AC #4 RED bar: utility/exit/read-failure and original mutation controls pass unchanged.
- [x] AC #5 RED bar: standalone actual CLI and all nine real guidance paths executed, no skips.
- [ ] GREEN implementation, exact amendment/historical truth authorization, six baseline/stale/final Gv outcomes, unchanged-golden compatibility and final PM acceptance remain outstanding.

