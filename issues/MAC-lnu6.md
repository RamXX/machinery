---
id: MAC-lnu6
title: "Remove unsafe frozen-test formatting exemptions"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:36:13Z
created_by: ramirosalas
updated_at: 2026-09-06T01:13:52Z
content_hash: "sha256:e48e60d55dc1597833c7aba1e728f01b1e18e6553b995579152ce6a582a2a13a"
blocks: [MAC-vx24, MAC-ou97]
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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]]

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

