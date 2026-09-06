---
id: MAC-lnu6
title: "Remove unsafe frozen-test formatting exemptions"
status: closed
priority: 0
type: bug
labels: [hard-tdd, red-approved, accepted]
parent: MAC-ui8a
created_at: 2026-09-05T19:36:13Z
created_by: ramirosalas
updated_at: 2026-09-06T23:26:04Z
content_hash: "sha256:578813f17c1649d35e534c22da063f7d2576dee660459300596cc3f08008a7b1"
assignee: dev-MAC-lnu6
follows: [MAC-a89e, MAC-p8ce, MAC-olrx, MAC-hgz1]
was_blocked_by: [MAC-hgz1]
closed_at: 2026-09-06T23:26:04Z
close_reason: "Accepted: unsafe frozen-test formatting exemptions removed; nine policy surfaces truthful; merged to local epic"
---

## Description
## USER INTENT
Frozen tests must not be weakened under a misleading formatting exemption.

## Context (Embedded)
cmd/machinery/tokensequal.go uses strings.Fields. Different indentation or spaces within string literals can change program behavior while this comparison reports equal. The utility can remain an honest whitespace-token comparison, never semantic or hard-TDD authority. Source inspection at epic 6cb2d974 found active formatting/token-identity exemptions on all nine guidance surfaces below, not just the original template. Six example BUILD files are also subjects of adjacent attestation covers. Removing every shipped authorization is existing AC1 coverage, not a new feature.

## Ownership
Exactly 17 paths remain this story's post-migration scope; preserve other agents' accepted changes and unrelated bytes. MAC-hgz1 exclusively holds the six BUILD/evidence paths first, after MAC-uzxr/MAC-lhu5/MAC-p7jd acceptance. This story retains its healthy claim but performs no shared-file writes until MAC-hgz1 is accepted. Its frozen RED0ea1fdc730aadac15cecc8de33bb95898ad91d60, CLI713184db16a12b8c3763b4aa8f5bf025721abf22 and all five AC remain intact.
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
Six conditional evidence paths: after accepted MAC-hgz1 migration, only the fourteen named rows below may receive independently reviewed policy-related attestor/date/note and BUILD-hash amendments. Preserve accepted upstream schema/kinds/full-root inventories/packet hashes and all other fields; no broader migration ownership:
- examples/checkout-split/orders/design/attestations.yaml
- examples/checkout-split/payments/design/attestations.yaml
- examples/fulfillment/design/attestations.yaml
- examples/go-crm/design/attestations.yaml
- examples/portfolio-engine/design/attestations.yaml
- examples/surreal-crm/design/attestations.yaml
Canonical scope is NOT evidence write authorization. Independent PM previously authorized only the nine exact prose deltas at source713184d; none of the fourteen evidence rows is currently authorized. Because upstream migration changes subjects/full-file hashes, prose and evidence require fresh exact post-migration independent PM review before this author writes them. No automatic reuse of old proposal hashes or reviewer/date.

## Serialized upstream baseline and policy ownership
MAC-hgz1 consumes accepted MAC-uzxr conformance, MAC-lhu5 packets and MAC-p7jd v2; it owns six BUILD/evidence corrections and only exact-before-edit PM-authorized affected golden updates. It MUST preserve the nine old policy replacement blocks for this story, and must not apply the policy replacements or edit this story's CLI/frozen tests. Its accepted source/schema/subject/golden state is the baseline for this story's remaining proof, not an assertion that this story changed those upstream files.
MAC-lnu6 alone applies its nine policy replacements after new PM review, then obtains truthful narrowly scoped evidence review and completes all original frozen tests and six-example checks. It performs ZERO old-test/golden changes. Existing oracle_test.go/golden_test.go and frozen tokensequal_semantics_test.go bytes stay exact; new upstream golden expectations must have their own MAC-hgz1 PM authorization and historical before/after provenance.
Preserve old713184d baseline raw logs and all original RED/proposal/full-file hashes as history. Do not compare the final tree to obsolete full-file hashes and call changed accepted upstream subjects unauthorized; instead separately prove (a) upstream accepted diffs and exact permissions, (b) unchanged lnu CLI/frozen files, and (c) only this story's new exact approved policy/evidence delta. No evidence or AC waiver and no claim that changed golden text proves execution.

## Conditional post-migration policy evidence amendment
Original proposal /tmp/MAC-lnu6-proposal.NZsERt/REVIEW.md SHA25648889c0b61afae030244dcd90f394515a183f9e90d9126dffe0602ffbe31b21f; exact policy blocks guidance-deltas.json SHA256d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011; original subject-inventory.json SHA256e08121e72aa07e230276a1e1227f761a4d7b193c52af1f5be22e5f43dfc9c0e5. Full PM report /tmp/MAC-lnu6-PM-AMENDMENT-713184d.md SHA2566fc5223d3e5d48170f89ff9e29b634b256bfe2454b710378f08cb60d875aadf1 authorized only prose and HELD all evidence. These are historical authorities, not a current approved post-migration file/evidence proposal.
Exactly fourteen affected claim rows, regardless of their accepted upstream plan/current kind:
- examples/checkout-split/orders/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context, g4.standin-coverage (3).
- examples/checkout-split/payments/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context, g4.standin-coverage (3).
- examples/fulfillment/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context (2).
- examples/go-crm/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context (2).
- examples/portfolio-engine/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context (2).
- examples/surreal-crm/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context (2).
Fresh independent PM review must verify all nine exact proposed prose deltas against accepted MAC-hgz1 subjects and every affected claim's FULL current covers and accepted kind. Only then may it authorize precise metadata/BUILD-hash edits and provide actual reviewer/date/verified specific note; no prefilling or auto-reusing historical attribution. Plan remains plan, current requires supported actual implementation review, historical acceptance remains historical. Narrow prose or Gv checks do not prove execution, reviewer identity or complete implementation acceptance.
Allowed lnu evidence delta only: those fourteen records' attestor/date/note and existing BUILD.md cover hash as exact independently authorized. Relative to accepted upstream migration, preserve claim IDs, schema/version, kinds, row/covers membership/order, implementation root/policy/entries/hash, all non-BUILD subject hashes (including newly accepted portfolio packet hashes), all other records, acceptance and git history. Schema/all-row-kind migration and packet-hash refresh occurred upstream under separate authority; they are not a continuing exception for this author.
The actual note must distinguish the new specific reviewed finding from original history and accepted migration, with truthful inline prior attribution/date/BUILD hash/note or explicit note absence as independently approved. Preserve full old records at immutable RED0ea1fdc and accepted upstream commit, and hash the new complete proposal/inventory. No new archive/schema/duplicate-claim mechanism or history rewrite. If the changed subjects require wider representation or inventory changes, return for precise canonical and PM review; do not silently change them.
For portfolio-engine both affected records cover BUILD plus all six accepted packets; all seven subjects must support the new plan review, not just changed prose. For Go CRM the current gt claim must retain actual accepted parser-backed conformance/guard/action proof and full scope; no reinstatement of unsupported handwritten-table claims. Missing verification holds evidence and delivery; it is not permission to weaken claims or disguise stale results.

## Boundary Map
PRODUCES:
- cmd/machinery/tokensequal.go -> honest whitespace-token CLI; equality is not semantic equivalence or frozen-edit permission.
- cmd/machinery/tokensequal_semantics_test.go -> real binary semantic-risk and utility controls plus all-nine-surface policy coverage.
- Nine explicitly owned guidance files -> no formatting/token-identity exception; frozen identity is exact bytes plus inventory, amendments require explicit new evidence revision and replay.
- Six explicitly owned attestations.yaml files -> conditional post-migration policy-related review amendment of exactly fourteen named rows, preserving accepted kind/schema/implementation scope and all non-BUILD hashes; no current amendment claimed.
CONSUMES:
- MAC-hgz1 accepted six-example consumer migration.
  source: accepted source/schema/subject/golden baseline following MAC-uzxr/MAC-lhu5/MAC-p7jd; exact policy blocks preserved, all-row v2 and packet-hash changes separately reviewed upstream. Current Go CRM Gv requires --impl examples/go-crm/impl; design-only plans intentionally report missing-current warnings and never current credit.
- newTokensEqualCmd() *cobra.Command; tokensEqualRunTo(oldPath, newPath string, stdoutW, stderrW io.Writer) error; strings.Fields and existing stable-file custody in cmd/machinery/tokensequal.go. Do not add a language parser to the product.
  source: cmd/machinery/tokensequal.go at epic 6cb2d974, functions newTokensEqualCmd and tokensEqualRunTo.
- goldenBin(t *testing.T) string; runBin(t *testing.T, args ...string) (string, string, int); repoRootDir(t *testing.T) string in cmd/machinery/golden_test.go, available unchanged for building/running the actual CLI with isolated configuration.
  source: cmd/machinery/golden_test.go at epic 6cb2d974, functions goldenBin/runBin/repoRootDir.
- Existing TestTokensEqual and TestTokensEqualRejectsMutationDuringComparison in cmd/machinery/oracle_test.go remain byte-for-byte unchanged. The former asserts the token-identical prefix and exit codes; its failure-message wording alone does not require amendment.
  source: cmd/machinery/oracle_test.go at epic 6cb2d974:757-813.
- Existing dependency MAC-lnu6 -> MAC-vx24 already orders the later process story after this hardened guidance. MAC-vx24 owns replay implementation and consumes the template/agent afterward; it must not be used to defer current unsafe shipped authorizations. MAC-ou97 remains downstream. This story now depends on MAC-hgz1; no reverse dependency, and the existing future replay owner remains unchanged.
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
- On accepted upstream MAC-hgz1 plus unchanged lnu CLI/frozen tests, rerun six ACTUAL baseline Gv checks before policy changes. Historical713184d baselines remain preserved but do not substitute for this new baseline. Use current accepted CLI check <design> --gate gv, supplying --impl examples/go-crm/impl for its current row; do not invent implementation roots for the other five designs. Baseline/final success means accepted ordinary plan/current behavior: design-only warnings are required and never current/full-complete credit.
- After fresh exact prose authorization, apply only the nine policy deltas and run all six stale-before-evidence checks, requiring intended changed-BUILD stale diagnostics, not any failure. After fresh substantive evidence authorization, run all six final Gv checks and require the same honest kind/coverage limits as baseline with no stale/invalid records. Record all eighteen outcomes, exact commands/input/SHA/logs and every intervening authorized delta.
- Prove only approved fourteen-row metadata/BUILD-hash changes relative to accepted upstream, unchanged kinds/schema/implementation inventory/non-BUILD hashes/other records/acceptance/history and exact preserved inline provenance. Rerun unchanged GoldenCheck test code against accepted upstream golden bytes; this story never edits those expectations. Any unexpected new mismatch returns for review, not auto-recapture or waived compatibility. Frozen RED0ea1fdc and CLI713184d content remain unchanged by the upstream migration and final lnu work except already approved lnu implementation authority; no additional test/golden change is authorized.
- Focused command: go test -count=1 ./cmd/machinery -run 'TokensEqual|Frozen' -json
- Compatibility command: go test -count=1 ./cmd/machinery -run 'TestGoldenCheck' -json
Runtime classification: service-free Go/native filesystem/local CLI only; no Docker, Python, Java or Node requirement and no Paivot product dependency. Record exact selected leaves, baseline/GREEN SHAs, terminal counts, duration and raw logs. No skip-if-missing, full preflight, remote mutations or installed binary replacement.

## OUT OF SCOPE
- Full language parser or semantic equivalence proof.
- Replay protocol implementation (MAC-vx24); no weakening or delayed removal of current policy.
- Shared helper rewrites, existing test/golden edits, wholesale BUILD rewrites, arbitrary attestation rehash, new claim IDs/renewed implementation acceptance/history rewriting or evidence changes outside the fourteen-record conditional boundary, generated oracle changes or unrelated guidance changes.

## DIFF BUDGET
- Revised forecast: 17 files, approximately 500 changed LOC combined, superseding the original 3 files/under 300 forecast because AC1 spans nine shipped surfaces and six adjacent evidence files.
- Existing measured/proposed lnu-only allocation remains337 changed lines before evidence (229 frozen RED,20 CLI,88 prose). Post-migration fourteen-row policy evidence forecast60-100 lines gives approximately397-437/~500 total over17 paths. Upstream MAC-hgz1 migration/packet/golden work has its own budget and must not be double-counted as lnu implementation. Revalidate actual post-migration evidence size before writes; this estimate does not grant broader fields or trim full-subject review. Report actual additions/deletions, files and runtime; this is an estimate, not automatic permission to exceed scope or trim proof. Escalate material growth with evidence.
- Preserve real CLI, all six baseline/stale/final Gv stages and unchanged-golden compatibility. Substantive full-claim review cost is additional and must be reported; the existing partial 26-leaf CLI focus and six baselines are not evidence that this pending review or final compatibility completed.

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
BOUNDED CURRENT-REVIEW SCOPE — CONDITIONAL ONLY, NOT GUIDANCE/EVIDENCE WRITE AUTHORIZATION
SCOPE-OWNER TRIAGE OF INDEPENDENT SUBSTANTIVE CLAIM FAILURES
Created P0 MAC-uzxr for actual Go CRM FSM parser/action-adequacy repair (five transition tests plus bounded test support; no BUILD/evidence writes). Created P0 MAC-lhu5 for source-established portfolio packet-alone omissions (six packets plus semantic/handoff test; root BUILD/evidence read-only). Both are contained in MAC-ui8a and explicitly block final MAC-ou97. Neither establishes repaired behavior today.
Core plan/current/historical distinction and exact v2 schema/CLI remain owned by active MAC-p7jd; no edits to its scope/frozen bar. MAC-l7m0/MAC-vx24 own future execution-authentication/replay, not proof of current examples.
All fourteen MAC-lnu6 evidence records remain held. The PM authorized only the nine exact prose deltas d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011; no broader claims/metadata/hashes or tests are authorized. New packet changes would alter currently protected non-BUILD hashes, so they require a serialized consumer migration/proposal revalidation, not an automatic hash refresh.
Pending dispatcher ownership decision: a consumer migration after MAC-p7jd/MAC-uzxr/MAC-lhu5 must reconcile six BUILD/YAML files, truthful design-only plan versus reviewed actual Go CRM current scope, exact historical provenance and affected golden expectations. V2 requires kind on every row, so this cannot be smuggled through the current fourteen-record metadata-only boundary. Explicit ownership/dependency/verification reconciliation is required before shared-file writers dispatch; no lnu dependency or ownership transfer added in this note.
Exact Go CRM BUILD concerns also need that bounded consumer clarification: migration.yaml declares mode rebuild/prototype phases but section8 calls the whole design greenfield; BUILD declares impl/go.mod authoritative yet lists x/crypto v0.53.0 versus actual v0.55.0. Correct wording against local authoritative files, not a library upgrade, migration implementation or external factual claim. Existing review cannot certify these away.
SERIALIZED CONSUMER MIGRATION AND POST-MIGRATION PROOF BASELINE

## nd_contract
status: in_progress

### evidence
- Dispatcher-authorized serial graph: MAC-uzxr + MAC-lhu5 + MAC-p7jd -> MAC-hgz1 -> MAC-lnu6; MAC-ou97 also directly depends on the consumer. Existing lnu claim/dev-MAC-lnu6, status and hard-tdd/red-approved labels retained. This is shared-file sequencing, not new core architecture or proof.
- MAC-hgz1 exclusively owns six BUILD/evidence migration/context corrections and only exact-before-edit PM-authorized affected golden updates. It preserves all nine old policy blocks; MAC-lnu6 alone owns policy replacements afterward. No concurrent shared-file writers.
- Original five AC, frozen RED0ea1fdc730aadac15cecc8de33bb95898ad91d60 and CLI713184db16a12b8c3763b4aa8f5bf025721abf22 remain intact. No old-test/golden edits by lnu; upstream accepted golden changes require separately recorded authority/provenance.
- All14 evidence amendments remain HELD today. Prior nine-block prose approval at713184d is historical; fresh exact post-migration prose AND full-claim/evidence review is required before new writes. No reviewer/date prefill or automatic schema/kind/scope/non-BUILD-hash edits.
- Accepted upstream source/schema/subject/golden baseline is consumed explicitly; retain historical original logs and rerun all6 actual baseline,6 stale-before-amendment and6 final Gv outcomes with honest plan/current warning limits. Current Go CRM uses actual --impl; no invented implementation for design-only examples. Unchanged-golden compatibility means the accepted upstream expected bytes, not silent mutation of old baselines.
- Scope stays17 paths/~500 lnu-only forecast; upstream migration budget is separately reported. No source/test/docs/evidence/worktree/installed asset changes by this triage.

### proof
- [ ] AC #1: original nine policy amendments and actual frozen-suite replay after accepted migration/fresh review.
- [ ] AC #2: retained CLI honesty with final truthful documentation.
- [ ] AC #3: frozen semantic/utility controls and final actual guidance checks.
- [x] AC #4: prior reported CLI utility/exit/custody controls remain unchanged; final compatibility replay still owed.
- [ ] AC #5: actual CLI/all-nine contract plus new six baseline/stale/final checks and accepted upstream golden compatibility; no Paivot dependency.


Prior canonical Description (historical pre-migration field/baseline boundary):
> ## USER INTENT
> Frozen tests must not be weakened under a misleading formatting exemption.
> 
> ## Context (Embedded)
> cmd/machinery/tokensequal.go uses strings.Fields. Different indentation or spaces within string literals can change program behavior while this comparison reports equal. The utility can remain an honest whitespace-token comparison, never semantic or hard-TDD authority. Source inspection at epic 6cb2d974 found active formatting/token-identity exemptions on all nine guidance surfaces below, not just the original template. Six example BUILD files are also subjects of adjacent attestation covers. Removing every shipped authorization is existing AC1 coverage, not a new feature.
> 
> ## Ownership
> Exactly 17 paths are forecast; preserve other agents' accepted changes and unrelated bytes.
> - cmd/machinery/tokensequal.go: honest help/comments/equal-output wording only; preserve comparison, exit behavior and rooted read/revalidation/close custody.
> - cmd/machinery/tokensequal_semantics_test.go (new): all new real CLI, semantic counterexample, utility-control and multi-surface guidance tests.
> Nine guidance surfaces, restricted to unsafe frozen-test authorization and its replacement exact-byte/inventory/evidence-revision/replay policy:
> - skills/machinery/references/build-md-template.md
> - agents/machinery-build-writer.md
> - docs/brownfield-team-guide.md
> - examples/checkout-split/orders/design/BUILD.md
> - examples/checkout-split/payments/design/BUILD.md
> - examples/fulfillment/design/BUILD.md
> - examples/go-crm/design/BUILD.md
> - examples/portfolio-engine/design/BUILD.md
> - examples/surreal-crm/design/BUILD.md
> Six conditional evidence paths: only the fourteen records explicitly listed below may amend attestor/date/note and the exact BUILD.md cover hash, and only after independent substantive review and exact PM authorization:
> - examples/checkout-split/orders/design/attestations.yaml
> - examples/checkout-split/payments/design/attestations.yaml
> - examples/fulfillment/design/attestations.yaml
> - examples/go-crm/design/attestations.yaml
> - examples/portfolio-engine/design/attestations.yaml
> - examples/surreal-crm/design/attestations.yaml
> Canonical ownership is NOT independent PM authorization to amend guidance, frozen fixtures or evidence. All nine guidance and six evidence writes remain held. This bounded current-review scope supersedes the earlier hash-only boundary ONLY as a conditional scope extension, not an actual re-attestation or grant to write.
> 
> ## Conditional current-review amendment boundary
> The fully read proposal is /tmp/MAC-lnu6-proposal.NZsERt/REVIEW.md SHA256 48889c0b61afae030244dcd90f394515a183f9e90d9126dffe0602ffbe31b21f; exact nine prose deltas are guidance-deltas.json SHA256 d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011; full old records/subject hashes are subject-inventory.json SHA256 e08121e72aa07e230276a1e1227f761a4d7b193c52af1f5be22e5f43dfc9c0e5 in the same directory. Exact committed-source comparison at 713184db16a12b8c3763b4aa8f5bf025721abf22 matched all nine old/proposed prose hashes and all six current attestation-file hashes. The proposal remains pending, not approved prose/evidence.
> Exactly fourteen affected records in the six owned attestation paths:
> - examples/checkout-split/orders/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context, g4.standin-coverage (3).
> - examples/checkout-split/payments/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context, g4.standin-coverage (3).
> - examples/fulfillment/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context (2).
> - examples/go-crm/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context (2).
> - examples/portfolio-engine/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context (2).
> - examples/surreal-crm/design/attestations.yaml: gt.conformance-test-shape, g4.zero-context (2).
> All fourteen records carry historical reviewer/date attribution; mechanically rebinding new policy bytes to those old reviews is not authorized.
> Before any write, independent PM must substantively review the nine exact prose deltas AND every affected claim against its FULL current covers set, then explicitly authorize the exact guidance/evidence edits, actual new reviewer identity/date and verified claim-specific note. Do not prefill an attestor or date, imply review is complete, or infer evidence permission from prose approval. Where applicable, inspect/replay the relevant evidence needed to support the claim; narrow CLI wording tests and Gv only establish their bounded outcomes.
> Only those fourteen records may change attestor, date, note (including adding a note if originally absent), and the hash of their existing BUILD.md cover. Preserve claim IDs, schema/attestation_version, record order, covers membership/order, every non-BUILD subject/hash, all other records, acceptance files and git history. No broader claim renewal, schema/archive/duplicate-claim mechanism or new evidence file.
> The actual reviewer/date must identify the actual new independent review. A reviewer-approved note must distinguish the specific current verified finding from history and explicitly disclaim renewed implementation acceptance or implementation replay not performed. Preserve exact old attestor/date/BUILD hash and original note verbatim (or explicitly absent) inline, identifying immutable RED 0ea1fdc730aadac15cecc8de33bb95898ad91d60. Full old records remain intact in that commit and the hashed subject inventory. This replacement/provenance representation itself requires PM confirmation of truthfulness; if insufficient, keep evidence held and return the exact alternative for explicit review, rather than inventing an archive or duplicate record.
> For portfolio-engine, BOTH records cover BUILD.md plus BUILD/M0-walking-skeleton.md, M1-run-pipeline.md, M2-feed-breaker.md, M3-optimizer.md, M4-portfolio-review.md and M5-reference-operations.md: all seven subjects must support each new review, with the six packet hashes unchanged. Go CRM's gt note asserts that real Go tests key executable table cases on every stable oracle ID and assert next state plus ordered actions; a prose-only/Gv check cannot establish that execution-related claim. No automatic carry-forward of historical implementation acceptance.
> If any complete claim or provenance representation cannot be truthfully supported in this scope, report the exact missing verification and keep evidence held; do not hide stale evidence, narrow the claim to obtain green, or deliver incomplete GREEN. This scope preparation changes no product protocol and grants no test/assertion/golden changes. Existing independently approved RED and its review bar remain intact.
> 
> ## Boundary Map
> PRODUCES:
> - cmd/machinery/tokensequal.go -> honest whitespace-token CLI; equality is not semantic equivalence or frozen-edit permission.
> - cmd/machinery/tokensequal_semantics_test.go -> real binary semantic-risk and utility controls plus all-nine-surface policy coverage.
> - Nine explicitly owned guidance files -> no formatting/token-identity exception; frozen identity is exact bytes plus inventory, amendments require explicit new evidence revision and replay.
> - Six explicitly owned attestations.yaml files -> conditional truthful current-review amendment of exactly fourteen named records (attestor/date/note + existing BUILD hash only), after substantive independent authorization and preserved provenance; no current amendment claimed.
> CONSUMES:
> - newTokensEqualCmd() *cobra.Command; tokensEqualRunTo(oldPath, newPath string, stdoutW, stderrW io.Writer) error; strings.Fields and existing stable-file custody in cmd/machinery/tokensequal.go. Do not add a language parser to the product.
>   source: cmd/machinery/tokensequal.go at epic 6cb2d974, functions newTokensEqualCmd and tokensEqualRunTo.
> - goldenBin(t *testing.T) string; runBin(t *testing.T, args ...string) (string, string, int); repoRootDir(t *testing.T) string in cmd/machinery/golden_test.go, available unchanged for building/running the actual CLI with isolated configuration.
>   source: cmd/machinery/golden_test.go at epic 6cb2d974, functions goldenBin/runBin/repoRootDir.
> - Existing TestTokensEqual and TestTokensEqualRejectsMutationDuringComparison in cmd/machinery/oracle_test.go remain byte-for-byte unchanged. The former asserts the token-identical prefix and exit codes; its failure-message wording alone does not require amendment.
>   source: cmd/machinery/oracle_test.go at epic 6cb2d974:757-813.
> - Existing dependency MAC-lnu6 -> MAC-vx24 already orders the later process story after this hardened guidance. MAC-vx24 owns replay implementation and consumes the template/agent afterward; it must not be used to defer current unsafe shipped authorizations. MAC-ou97 remains downstream. No dependency changes.
>   source: shared nd MAC-vx24 BlockedBy includes MAC-lnu6; existing MAC-lnu6 Blocks includes MAC-vx24 and MAC-ou97.
> 
> ### Story Acceptance Criteria
> 1. Remove all authorization to edit frozen RED tests based on tokens-equal; exact bytes and inventory define identity, and any amendment needs explicit new evidence revision plus replay.
> 2. CLI help/output/docs accurately describe whitespace-token comparison, not preserved program meaning or proof of formatting-only change.
> 3. Negative tests use spacing inside quoted literals and indentation-sensitive source that compare token-equal but have changed semantics; no guidance/gate treats that as approved frozen-test edit.
> 4. Positive genuine whitespace-only comparison utility retains documented exit behavior without claiming hard-TDD approval.
> 5. Real CLI calls and shipped template contract test establish user-facing behavior; no Paivot product dependency.
> 
> ## Testing Requirements
> Hard TDD: author only the new owned test file for RED; no existing test/golden/helper edits are pre-authorized. Independently replay and freeze expected behavioral failures and passing controls before GREEN. Explicit PM fixture/evidence authorization remains required as above.
> - Actual built CLI, real temporary files, no mocks/stubs or installed binary. On unchanged production, quoted-literal spacing and indentation-sensitive input pairs must genuinely differ in meaning while yielding equal whitespace tokens; make that semantic distinction executable using service-free Go facilities/already-pinned dependencies, not a label or an arbitrary nonzero error. CLI help/equal output must not claim preserved meaning, formatting-only proof or frozen-edit approval.
> - Positive genuine whitespace-only controls retain exit 0 and the token-identical: N tokens prefix; token-value/count differences retain exit 1 and NOT token-identical diagnostics; ordinary read failure remains exit 1. Preserve existing real mutation/custody tests and all old bytes, including oracle_test.go and golden_test.go.
> - Table-driven contract tests read ALL nine actual shipped guidance paths (missing paths fail, no skips). Each baseline case must fail because its actual unsafe authorization remains; GREEN requires affirmative exact-byte/inventory identity and explicit new evidence revision plus replay, and absence of token-identity/formatting exemptions. Include the agent, brownfield guide and six example BUILD files, not only the template. Record the tracked-source policy scan and review any remaining hits rather than hiding them with an overbroad filter.
> - Keep semantic-risk negatives separate from passing utility controls: token equality remains useful, but never authorizes a frozen edit. No false RED from a missing future API, unknown CLI flag or missing fixture.
> - After independently authorized BUILD/evidence edits, run the actual built CLI check <design-dir> --gate gv for all six owned example design directories with isolated config. Show changed BUILD is stale before its exact authorized evidence amendment and that afterward no stale/invalid attestation is introduced; record all six baseline, six stale-before-amendment and six final outcomes. Inspect the exact evidence diff to prove only the fourteen approved attestor/date/note/BUILD-hash deltas, exact inline provenance, unchanged claim IDs/schema/covers membership/order, all non-BUILD hashes/other records and acceptance/history. Six actual baselines at 713184d already passed (reported structural evidence, not substantive review); stale and final stages remain pending. Preserve existing golden bytes and rerun affected real example check tests; any pre-existing unrelated failure must be attributed explicitly, not concealed by evidence rewriting.
> - Focused command: go test -count=1 ./cmd/machinery -run 'TokensEqual|Frozen' -json
> - Compatibility command: go test -count=1 ./cmd/machinery -run 'TestGoldenCheck' -json
> Runtime classification: service-free Go/native filesystem/local CLI only; no Docker, Python, Java or Node requirement and no Paivot product dependency. Record exact selected leaves, baseline/GREEN SHAs, terminal counts, duration and raw logs. No skip-if-missing, full preflight, remote mutations or installed binary replacement.
> 
> ## OUT OF SCOPE
> - Full language parser or semantic equivalence proof.
> - Replay protocol implementation (MAC-vx24); no weakening or delayed removal of current policy.
> - Shared helper rewrites, existing test/golden edits, wholesale BUILD rewrites, arbitrary attestation rehash, new claim IDs/renewed implementation acceptance/history rewriting or evidence changes outside the fourteen-record conditional boundary, generated oracle changes or unrelated guidance changes.
> 
> ## DIFF BUDGET
> - Revised forecast: 17 files, approximately 500 changed LOC combined, superseding the original 3 files/under 300 forecast because AC1 spans nine shipped surfaces and six adjacent evidence files.
> - Updated measured/proposed allocation: frozen RED + CLI + nine proposed prose deltas total 337 changed lines before evidence; the prose alone is +57/-31 (88). Fourteen provenance-preserving current-review amendments forecast 60-100 evidence changed LOC rather than the old 20-40 hash-only allocation, for approximately 397-437 combined, retaining the approximately 500 forecast and 17 paths. Report actual additions/deletions, files and runtime; this is an estimate, not automatic permission to exceed scope or trim proof. Escalate material growth with evidence.
> - Preserve real CLI, all six baseline/stale/final Gv stages and unchanged-golden compatibility. Substantive full-claim review cost is additional and must be reported; the existing partial 26-leaf CLI focus and six baselines are not evidence that this pending review or final compatibility completed.
> 
> ## MANDATORY SKILLS
> - developer; codebase-memory; pm_acceptor.
> 

## nd_contract
status: in_progress

### evidence
- Sr PM triage created MAC-uzxr and MAC-lhu5 as separate P0 capstone prerequisites; source-established defects are not current repairs. Six BUILD/evidence consumer migration, serial ownership and future v2/golden contract reconciliation remain a dispatcher decision. No change to this story's claim/status/AC/frozen tests/dependencies and no evidence-write authorization.
- Independent bounded amendment review completed at 713184db16a12b8c3763b4aa8f5bf025721abf22, immutable RED 0ea1fdc730aadac15cecc8de33bb95898ad91d60 unchanged.
- Exactly nine prose replacements authorized by guidance-deltas SHA256 d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011 and full-file hashes above; none applied during PM review.
- All fourteen records reviewed against full covers, including all seven portfolio subjects. All six evidence files remain held; no attestor/date/note/BUILD-hash write authorization. Missing current wholesale FSM parsing and action-completeness proof, design-only/current-claim mismatch, and packet/context concerns recorded explicitly above.
- Independent native Go CRM transition replay: 218 PASS, 0 FAIL, 0 SKIP; all 197 committed FSM IDs present in executed names; three package durations 0.590/0.361/0.917s, command wall1.503486s; raw log and SHA256 above. This is bounded transition proof, not wholesale parser or full implementation acceptance.
- Detached PM checkout retained clean; source/tests/goldens/evidence/main/epic/installed assets/user services unchanged. No state transition, release, new dependency or backlog expansion; existing in_progress/hard-tdd/red-approved/dev-MAC-lnu6 claim preserved.

### proof
- [ ] AC #1: nine exact policy changes authorized; actual application and immutable-suite replay pending, evidence amendments held.
- [ ] AC #2: CLI-only wording reviewed; authorized documentation still unapplied.
- [ ] AC #3: existing executable semantic counterexamples remain frozen; actual shipped guidance replay pending.
- [x] AC #4: prior recorded CLI utility/exit/custody controls remain unchanged; no comparison implementation change found.
- [ ] AC #5: final all-nine contract, six stale and six final Gv outcomes, unchanged-golden TestGoldenCheck and GREEN PM acceptance remain owed.


## nd_contract
status: in_progress

### evidence
- Sr PM conditional canonical scope extension only, root-authorized: fourteen named records in the existing six evidence files may be considered for actual reviewer/date/note + BUILD-hash amendment AFTER independent substantive review and exact PM authorization. Nine prose and six evidence writes remain held. No current re-attestation or implementation acceptance/replay claimed.
- All five original AC, independently approved RED 0ea1fdc730aadac15cecc8de33bb95898ad91d60, tests/goldens, state/claim/dependencies unchanged. Scope remains seventeen paths/~500 LOC; measured/proposed 337 pre-evidence plus forecast60-100 evidence LOC.
- Fully read REVIEW.md (319 lines), guidance-deltas.json and subject-inventory.json; exact stated SHA256 values match. Source at713184d matches all9 original/proposed prose hashes and all6 evidence-file hashes. Graph coverage metadata_match/no recorded gaps is best effort, not semantic completeness.
- Independent reviewer must inspect nine exact prose deltas and FULL covers/claim semantics of all14 records, including portfolio7-subject records and Go CRM executable-test assertion, provide actual identity/date/specific findings and approve provenance representation. Missing verification remains an evidence hold, never automatic rehash. All claim IDs/schema/membership/order/non-BUILD hashes/other records/acceptance/history preserved.
- Six baseline Gv passes are structural only; six stale-before-amendment, six final Gv and unchanged-golden compatibility remain mandatory. No source/test/docs/evidence/installed/worktree changes during this repair.
- CLI commit 713184db16a12b8c3763b4aa8f5bf025721abf22; frozen RED unchanged; focused 26 leaves: 17 PASS / 9 expected guidance FAIL / 0 SKIP; six actual Gv baselines PASS.
- Exact review artifact and SHA256 above. Nine guidance files and six evidence files remain untouched; independent amendment/historical-truth authorization pending.
- Healthy hold retains the atomic claim and clean worktree. No delivery, release, closure, renewed acceptance or current re-attestation claimed.

### proof
- [ ] AC #1: exact nine guidance amendments proposed; application and replay await authorization.
- [ ] AC #2: actual CLI help/output honesty passes; documentation changes pending.
- [ ] AC #3: executable quoted-space and valid indentation semantic controls, real equal-token calls and CLI honesty pass; guidance condition pending.
- [x] AC #4: genuine whitespace/empty utility, count prefix, exit/read-error and original mutation/custody controls pass unchanged.
- [ ] AC #5: actual standalone CLI and all nine real guidance cases execute without skips; full shipped contract, post-amendment Gv and unchanged-golden compatibility remain pending.



Prior canonical Description (historical hash-only boundary superseded only as explicitly conditional scope):
> ## USER INTENT
> Frozen tests must not be weakened under a misleading formatting exemption.
> 
> ## Context (Embedded)
> cmd/machinery/tokensequal.go uses strings.Fields. Different indentation or spaces within string literals can change program behavior while this comparison reports equal. The utility can remain an honest whitespace-token comparison, never semantic or hard-TDD authority. Source inspection at epic 6cb2d974 found active formatting/token-identity exemptions on all nine guidance surfaces below, not just the original template. Six example BUILD files are also subjects of adjacent attestation covers. Removing every shipped authorization is existing AC1 coverage, not a new feature.
> 
> ## Ownership
> Exactly 17 paths are forecast; preserve other agents' accepted changes and unrelated bytes.
> - cmd/machinery/tokensequal.go: honest help/comments/equal-output wording only; preserve comparison, exit behavior and rooted read/revalidation/close custody.
> - cmd/machinery/tokensequal_semantics_test.go (new): all new real CLI, semantic counterexample, utility-control and multi-surface guidance tests.
> Nine guidance surfaces, restricted to unsafe frozen-test authorization and its replacement exact-byte/inventory/evidence-revision/replay policy:
> - skills/machinery/references/build-md-template.md
> - agents/machinery-build-writer.md
> - docs/brownfield-team-guide.md
> - examples/checkout-split/orders/design/BUILD.md
> - examples/checkout-split/payments/design/BUILD.md
> - examples/fulfillment/design/BUILD.md
> - examples/go-crm/design/BUILD.md
> - examples/portfolio-engine/design/BUILD.md
> - examples/surreal-crm/design/BUILD.md
> Six conditional evidence paths, restricted to covers hashes whose exact subject path is BUILD.md:
> - examples/checkout-split/orders/design/attestations.yaml
> - examples/checkout-split/payments/design/attestations.yaml
> - examples/fulfillment/design/attestations.yaml
> - examples/go-crm/design/attestations.yaml
> - examples/portfolio-engine/design/attestations.yaml
> - examples/surreal-crm/design/attestations.yaml
> Canonical ownership is NOT independent PM authorization to amend frozen fixtures or evidence. Before those writes, the reviewer must explicitly authorize the exact affected guidance/evidence amendment. Refresh only the actual reviewed BUILD subject hashes AFTER independent review of each prose delta; preserve every other subject/hash, claim, schema, attestor/date/note, acceptance and historical fact. Hash recomputation does not fabricate a new review or execution; no blanket rehash or renewed acceptance. Record the reviewer decision and exact old/new BUILD hashes.
> 
> ## Boundary Map
> PRODUCES:
> - cmd/machinery/tokensequal.go -> honest whitespace-token CLI; equality is not semantic equivalence or frozen-edit permission.
> - cmd/machinery/tokensequal_semantics_test.go -> real binary semantic-risk and utility controls plus all-nine-surface policy coverage.
> - Nine explicitly owned guidance files -> no formatting/token-identity exception; frozen identity is exact bytes plus inventory, amendments require explicit new evidence revision and replay.
> - Six explicitly owned attestations.yaml files -> only independently reviewed changed BUILD subject hashes, preserving all other evidence.
> CONSUMES:
> - newTokensEqualCmd() *cobra.Command; tokensEqualRunTo(oldPath, newPath string, stdoutW, stderrW io.Writer) error; strings.Fields and existing stable-file custody in cmd/machinery/tokensequal.go. Do not add a language parser to the product.
>   source: cmd/machinery/tokensequal.go at epic 6cb2d974, functions newTokensEqualCmd and tokensEqualRunTo.
> - goldenBin(t *testing.T) string; runBin(t *testing.T, args ...string) (string, string, int); repoRootDir(t *testing.T) string in cmd/machinery/golden_test.go, available unchanged for building/running the actual CLI with isolated configuration.
>   source: cmd/machinery/golden_test.go at epic 6cb2d974, functions goldenBin/runBin/repoRootDir.
> - Existing TestTokensEqual and TestTokensEqualRejectsMutationDuringComparison in cmd/machinery/oracle_test.go remain byte-for-byte unchanged. The former asserts the token-identical prefix and exit codes; its failure-message wording alone does not require amendment.
>   source: cmd/machinery/oracle_test.go at epic 6cb2d974:757-813.
> - Existing dependency MAC-lnu6 -> MAC-vx24 already orders the later process story after this hardened guidance. MAC-vx24 owns replay implementation and consumes the template/agent afterward; it must not be used to defer current unsafe shipped authorizations. MAC-ou97 remains downstream. No dependency changes.
>   source: shared nd MAC-vx24 BlockedBy includes MAC-lnu6; existing MAC-lnu6 Blocks includes MAC-vx24 and MAC-ou97.
> 
> ### Story Acceptance Criteria
> 1. Remove all authorization to edit frozen RED tests based on tokens-equal; exact bytes and inventory define identity, and any amendment needs explicit new evidence revision plus replay.
> 2. CLI help/output/docs accurately describe whitespace-token comparison, not preserved program meaning or proof of formatting-only change.
> 3. Negative tests use spacing inside quoted literals and indentation-sensitive source that compare token-equal but have changed semantics; no guidance/gate treats that as approved frozen-test edit.
> 4. Positive genuine whitespace-only comparison utility retains documented exit behavior without claiming hard-TDD approval.
> 5. Real CLI calls and shipped template contract test establish user-facing behavior; no Paivot product dependency.
> 
> ## Testing Requirements
> Hard TDD: author only the new owned test file for RED; no existing test/golden/helper edits are pre-authorized. Independently replay and freeze expected behavioral failures and passing controls before GREEN. Explicit PM fixture/evidence authorization remains required as above.
> - Actual built CLI, real temporary files, no mocks/stubs or installed binary. On unchanged production, quoted-literal spacing and indentation-sensitive input pairs must genuinely differ in meaning while yielding equal whitespace tokens; make that semantic distinction executable using service-free Go facilities/already-pinned dependencies, not a label or an arbitrary nonzero error. CLI help/equal output must not claim preserved meaning, formatting-only proof or frozen-edit approval.
> - Positive genuine whitespace-only controls retain exit 0 and the token-identical: N tokens prefix; token-value/count differences retain exit 1 and NOT token-identical diagnostics; ordinary read failure remains exit 1. Preserve existing real mutation/custody tests and all old bytes, including oracle_test.go and golden_test.go.
> - Table-driven contract tests read ALL nine actual shipped guidance paths (missing paths fail, no skips). Each baseline case must fail because its actual unsafe authorization remains; GREEN requires affirmative exact-byte/inventory identity and explicit new evidence revision plus replay, and absence of token-identity/formatting exemptions. Include the agent, brownfield guide and six example BUILD files, not only the template. Record the tracked-source policy scan and review any remaining hits rather than hiding them with an overbroad filter.
> - Keep semantic-risk negatives separate from passing utility controls: token equality remains useful, but never authorizes a frozen edit. No false RED from a missing future API, unknown CLI flag or missing fixture.
> - After independently authorized BUILD/evidence edits, run the actual built CLI check <design-dir> --gate gv for all six owned example design directories with isolated config. Show changed BUILD is stale before its exact authorized hash refresh and that afterward no stale/invalid attestation is introduced; record baseline and final outcomes. Inspect the evidence diff to prove all non-BUILD fields/hashes and acceptance/history unchanged. Preserve existing golden bytes and rerun affected real example check tests; any pre-existing unrelated failure must be attributed explicitly, not concealed by evidence rewriting.
> - Focused command: go test -count=1 ./cmd/machinery -run 'TokensEqual|Frozen' -json
> - Compatibility command: go test -count=1 ./cmd/machinery -run 'TestGoldenCheck' -json
> Runtime classification: service-free Go/native filesystem/local CLI only; no Docker, Python, Java or Node requirement and no Paivot product dependency. Record exact selected leaves, baseline/GREEN SHAs, terminal counts, duration and raw logs. No skip-if-missing, full preflight, remote mutations or installed binary replacement.
> 
> ## OUT OF SCOPE
> - Full language parser or semantic equivalence proof.
> - Replay protocol implementation (MAC-vx24); no weakening or delayed removal of current policy.
> - Shared helper rewrites, existing test/golden edits, wholesale BUILD rewrites, arbitrary attestation rehash, new claims/acceptance/history, generated oracle changes or unrelated guidance changes.
> 
> ## DIFF BUDGET
> - Revised forecast: 17 files, approximately 500 changed LOC combined, superseding the original 3 files/under 300 forecast because AC1 spans nine shipped surfaces and six adjacent evidence files.
> - Approximate allocation: CLI 20-50; new tests 250-330; nine narrow guidance edits 60-100; six narrow evidence deltas 20-40. Report actual additions/deletions, files and runtime; this is an estimate, not automatic permission to exceed scope or trim proof. Escalate material growth with evidence.
> - One native CLI build plus bounded real CLI calls and six Gv checks; actual runtime remains to be measured.
> 
> ## MANDATORY SKILLS
> - developer; codebase-memory; pm_acceptor.
> 

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
- 2026-09-06T02:22:34Z status: open -> in_progress
- 2026-09-06T02:22:34Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-06T02:22:34Z claimed by dev-MAC-lnu6
- 2026-09-06T03:17:09Z dep_added: blocked_by MAC-hgz1
- 2026-09-06T09:10:09Z dep_added: blocks MAC-wbxq
- 2026-09-06T20:22:41Z dep_removed: was_blocked_by MAC-hgz1
- 2026-09-06T20:23:32Z status: in_progress -> in_progress
- 2026-09-06T20:23:32Z auto-follows: linked to predecessor MAC-hgz1
- 2026-09-06T23:26:04Z status: in_progress -> closed
- 2026-09-06T23:26:04Z dep_removed: no_longer_blocks MAC-vx24
- 2026-09-06T23:26:04Z dep_removed: no_longer_blocks MAC-ou97
- 2026-09-06T23:26:04Z dep_removed: no_longer_blocks MAC-wbxq

## Links
- Parent: [[MAC-ui8a]]
- Was blocked by: [[MAC-hgz1]]
- Follows: [[MAC-a89e]], [[MAC-p8ce]], [[MAC-olrx]], [[MAC-hgz1]]

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


### 2026-09-06T02:30:55Z ramirosalas
## MAC-lnu6 bounded GREEN stage — healthy amendment-review hold

PROOF:
- Canonical Description and last independent PM RED decision read in full. Fresh GREEN author; retained story/MAC-lnu6 worktree /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-lnu6, frozen RED0ea1fdc730aadac15cecc8de33bb95898ad91d60 on epic6cb2d974ea8aea211a5974f453cef2b5802bb11e.
- CLI-only commit713184db16a12b8c3763b4aa8f5bf025721abf22: cmd/machinery/tokensequal.go +11/-9. Honest comments, Short/Long help and equality-output wording only; token comparison/count prefix/exit and rooted stable open/read/revalidate/close behavior unchanged. All other tracked bytes unchanged after RED; clean worktree. Overall epic diff currently2files +240/-9 including immutable RED229additions.
- Focused actual CLI command: timeout150s go test -count=1 -timeout=120s ./cmd/machinery -run 'TokensEqual|Frozen' -json. Terminal exit1, partial GREEN26leaves17PASS/9expected guidanceFAIL/0SKIP, package1.614s, terminal2026-09-05T19:25:14.840016-07:00. Raw /tmp/MAC-lnu6-cli-713184d.jsonl SHA256ca09b37821c5217b498c5f9403823a1cd3ae8e8fcddf3a5c149dba3bc129a2b6.
- PASS: TestTokensEqual; TestTokensEqualRejectsMutationDuringComparison; both semantic cases' executable_semantic_control, whitespace_token_utility and no_semantic_or_frozen_edit_assurance leaves; both command/root HelpDescribesOnlyWhitespaceTokens leaves; all6RealCLIUtilityControls (whitespace_reflow, empty_whitespace, token_value_changed, token_added, token_removed, missing_input); WhitespaceSuccessDoesNotAuthorizeFrozenEdits.
- Expected FAIL: each of the9canonical paths in TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay. Each actual old owner-sanctioned formatting/token-identity exemption fails at test176 before replacement-policy assertions. Complete exact leaf inventory is in review artifact. These failures remain held by explicit independent amendment requirement; no GREEN delivery is claimed.
- Frozen new test SHA2561138fc41a835abf3769210bdbc2dbdb1d9d2ad6db9752b14c90bf41b9945c51f; git diff frozenRED HEAD -- '*_test.go' '**/testdata/**' empty. Existing tests/helpers/goldens immutable. pvg verify cmd/machinery/tokensequal.go --format text PASS1file0issues; git diff --check clean.
- Exact9prose old/new blocks, prospective whole-file hashes, sixattestation/14record handling proposal: /tmp/MAC-lnu6-proposal.NZsERt/REVIEW.md SHA25648889c0b61afae030244dcd90f394515a183f9e90d9126dffe0602ffbe31b21f. Proposal only; none applied.
- Machine-readable exact deltas: /tmp/MAC-lnu6-proposal.NZsERt/guidance-deltas.json SHA256d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011. Complete old affected records and old/proposed hashes: /tmp/MAC-lnu6-proposal.NZsERt/subject-inventory.json SHA256e08121e72aa07e230276a1e1227f761a4d7b193c52af1f5be22e5f43dfc9c0e5.
- Proposed9prose changes+57/-31; combinedCLI+RED+prose337changedLOC before evidence. No guidance/evidence/test write authorization inferred. For all6files recommend NO mechanical hash-only historical rebinding. Proposed current re-attestation requires canonical scope extension for exactly14affected records' attestor/date/note plus BUILDhash, actual independent claim verification, and exact original provenance retention; all claimIDs/schema/otherrecords/nonBUILDcovers/hash/acceptance/history unchanged. This is an explicitly UNAUTHORIZED proposal, forecast60–100 evidencechangedLOC rather than20–40, still17paths/about500total; canonical review is required before these broader changes. Actual reviewer/date/verified findings must come from independent review, not fabricated placeholders.
- Portfolio's2affected records each coverBUILD plus6milestonepackets: a current attestor would need full7subject review. GoCRM gt note asserts executable Go test behavior; narrow prose review/Gv cannot renew it. Historical Sept2/3reviews cannot silently claim to have examined newly amended BUILD bytes.
- Local CLI built with timeout120s go build -o /tmp/MAC-lnu6-proposal.NZsERt/machinery ./cmd/machinery. BinarySHA2563c519c5e609f694527a40255cae932a2bd683dba17ca6826c3234ea897845e96. Each actual baseline used MACHINERY_CONFIG_DIR=/tmp/MAC-lnu6-proposal.NZsERt/config timeout30s <local-binary> check <design-dir> --gate gv. All6baselineexit0, stderr empty,0blocking. Currentartifacts/attested/owed: orders23/14/14, payments23/14/14, fulfillment56/12/12, goCRM54/13/13, portfolio52/12/12, surrealCRM48/12/12. Durations0.217831/0.208802/0.091902/0.106182/0.086750/0.099941s. Initial exploratory6passed, repeated once for individual raw exit/duration records.
- Raw6baseline log /tmp/MAC-lnu6-proposal.NZsERt/gv-baseline.jsonl SHA2569400c61ce661483cc5019394f788dcca7098a22a5002ed820048308da1c5769e. This only proves current structural validity. Six changedBUILD stale-before-refresh and sixfinalGvchecks remain pending actualauthorizededits.
- TestGoldenCheck/fullsuite notrun in boundedCLI stage; mandatory unchanged-golden compatibility remains after authorization. Go statementcoverage notmeasured. All5AC accounted below; no coveragepercentage falselyclaimed.
- Bounded tracked scan git grep -n -E 'tokens-equal|token-identity proof|formatting-only amendment' -- ':!go.sum': allreturnedhits reviewed. Nineguidanceexemptionsawaitproposal, READMEcommandinventory, oldcustody/snapshotdiagnostics, frozenREDtestpatterns and currentCLIhonestwording. No natural-language completenessclaim; finalscan+directprose reviewmandatory.
- Codebase-memory Verify graphgeneration2026-09-05T23:58:53Z,7tokensEqualresults/noextra pages,19evidencepaths+worktreescopecoveragechecked; worktreeexcluded/newtestmissing, exactsourcefallbackused. Vaultsearch no relevantreturnednotes. No remote/main/epic/history/installedasset/service/otherworktreemutation and no Paivot productdependency.

LEARNINGS:
- All six evidence files, not only GoCRM, attribute BUILD claims to historical reviewers/dates; policy changes cannot be safely represented by silent old-record hash substitution.
- Separating CLI implementation from held fixture/evidence changes gives honest partialGREEN: fivepreviousCLIassurancefailures nowPASS while nineactualguidancefailures remain visible.
- Full covers sets matter for re-attestation: portfolio's unchanged packets and GoCRM's execution claim need substantive independent review, not mechanical hash refresh.

## nd_contract
status: in_progress

### evidence
- Claimed retained story/MAC-lnu6; CLI713184db16a12b8c3763b4aa8f5bf025721abf22 committed; cleanworktree;26leaves17PASS/9expectedguidanceFAIL/0SKIP,1.614s; frozenRED/newandoldtestsuntouched.
- Exactreviewartifact/hashand6baselineGvlogsabove. Nineguidance/sixevidencefilesuntouched. Independentprose/evidencedecisionpending; no implicitapproval, delivery, release or closure.
- Healthy hold preservesclaim/worktree; broaderre-attestationmetadataextensionrequirescanonicalreview beforewrites. FrozenREDapprovaldoesnotauthorizeit.

### proof
- [ ] AC #1: exactnineguidanceamendmentsproposed; apply/replayawaitindependentauthorization.
- [ ] AC #2: actualCLIhelp/outputhonestyPASS; docschangespending.
- [ ] AC #3: executableGoquotedspace8to7andvalidYAMLpermissionownership1to0plusrealCLIequalandhonestyPASS; guidanceconditionpending.
- [x] AC #4: genuinewhitespaceutility/empty/countprefix/exit/readerrorandoriginalmutationcustodyPASSunchanged.
- [ ] AC #5: actualstandaloneCLIandallnineguidancecasesexecuted0skip; fullshippedcontractandpostamendmentGv/unchangedgoldencompatibilitypending.

### 2026-09-06T02:31:41Z ramirosalas
## MAC-lnu6 current bounded-stage contract (readable terminal summary)

The preceding detailed evidence and its immutable raw logs remain applicable. This is a healthy in_progress hold for independent amendment review, not a delivery.

PROOF:
- Retained branch story/MAC-lnu6 at 713184db16a12b8c3763b4aa8f5bf025721abf22. Only cmd/machinery/tokensequal.go changed after frozen RED 0ea1fdc730aadac15cecc8de33bb95898ad91d60: +11/-9, limited to comments/help/equal-output wording. Comparison, count prefix, exit codes and stable read/revalidate/close custody remain unchanged.
- Focused command: timeout 150s go test -count=1 -timeout=120s ./cmd/machinery -run 'TokensEqual|Frozen' -json. Result: exit 1, 26 leaves, 17 PASS / 9 expected guidance FAIL / 0 SKIP, package elapsed 1.614s. All five earlier CLI assurance failures now pass. All nine guidance cases still fail on their actual old formatting/token-identity exemption, as expected while amendment authorization is held. Raw log and exact leaf inventory are in the review artifact.
- Review artifact: /tmp/MAC-lnu6-proposal.NZsERt/REVIEW.md, SHA256 48889c0b61afae030244dcd90f394515a183f9e90d9126dffe0602ffbe31b21f. Includes exact nine old/new prose blocks, prospective BUILD hashes, complete six-file / 14-record evidence inventory, exact runtime logs and decisions needed. Companion guidance-deltas.json and subject-inventory.json in the same directory are immutable proposals.
- All six actual baseline Gv commands passed with exit 0, empty stderr and zero blocking findings. Binary built locally from the CLI commit; private MACHINERY_CONFIG_DIR, finite timeouts, no installed binary used. Baseline raw log: /tmp/MAC-lnu6-proposal.NZsERt/gv-baseline.jsonl, SHA256 9400c61ce661483cc5019394f788dcca7098a22a5002ed820048308da1c5769e.
- Frozen test SHA256 remains 1138fc41a835abf3769210bdbc2dbdb1d9d2ad6db9752b14c90bf41b9945c51f. All existing tests/helpers/goldens and all guidance/evidence files are untouched. Worktree clean; diff --check clean; pvg verify passed (1 file, 0 issues). No skips, mocks, Paivot product dependency, remote mutation or user-service/installed-asset changes.
- Exact proposed guidance: +57/-31. Combined CLI, frozen RED and proposed prose: 337 changed lines before evidence. Recommendation: reject silent hash-only historical rebinding for all six evidence files. A current re-attestation requires explicit canonical extension of the 14 affected records' attestor/date/note plus BUILD hash, actual independent verification of each complete claim/coverage set, and preserved original provenance. No broader write is authorized or performed. Estimate 60–100 evidence changed lines instead of 20–40, retaining 17 paths and approximately 500 total; this is a scope-review request, not inferred permission.
- All other claims, covers membership/order, non-BUILD subjects/hashes, schemas, acceptance and git history must remain unchanged. Portfolio's two records each cover seven subjects. Go CRM's gt note asserts executable Go test behavior. Neither the narrow CLI test run nor Gv establishes those full current review claims.
- Pending: independent exact-prose and truthful evidence decision; any necessary canonical scope authorization; approved guidance/evidence writes; six stale-before-refresh and six final Gv checks; unchanged-golden TestGoldenCheck; final focused GREEN, source scan and PM acceptance. Statement coverage was not measured; full suite and compatibility were not run in this bounded stage.

LEARNINGS:
- Historical reviewer/date binding affects all six examples; mechanically matching hashes does not establish a new substantive review.
- Unchanged utility/custody checks and separately failing guidance cases provide honest evidence for the bounded CLI stage.
- Re-attestation must address the full covers set and the actual claim, including unchanged portfolio packets and Go CRM's execution statement.

## nd_contract
status: in_progress

### evidence
- CLI commit 713184db16a12b8c3763b4aa8f5bf025721abf22; frozen RED unchanged; focused 26 leaves: 17 PASS / 9 expected guidance FAIL / 0 SKIP; six actual Gv baselines PASS.
- Exact review artifact and SHA256 above. Nine guidance files and six evidence files remain untouched; independent amendment/historical-truth authorization pending.
- Healthy hold retains the atomic claim and clean worktree. No delivery, release, closure, renewed acceptance or current re-attestation claimed.

### proof
- [ ] AC #1: exact nine guidance amendments proposed; application and replay await authorization.
- [ ] AC #2: actual CLI help/output honesty passes; documentation changes pending.
- [ ] AC #3: executable quoted-space and valid indentation semantic controls, real equal-token calls and CLI honesty pass; guidance condition pending.
- [x] AC #4: genuine whitespace/empty utility, count prefix, exit/read-error and original mutation/custody controls pass unchanged.
- [ ] AC #5: actual standalone CLI and all nine real guidance cases execute without skips; full shipped contract, post-amendment Gv and unchanged-golden compatibility remain pending.


### 2026-09-06T02:38:47Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Sr PM conditional canonical scope extension only, root-authorized: fourteen named records in the existing six evidence files may be considered for actual reviewer/date/note + BUILD-hash amendment AFTER independent substantive review and exact PM authorization. Nine prose and six evidence writes remain held. No current re-attestation or implementation acceptance/replay claimed.
- All five original AC, independently approved RED 0ea1fdc730aadac15cecc8de33bb95898ad91d60, tests/goldens, state/claim/dependencies unchanged. Scope remains seventeen paths/~500 LOC; measured/proposed 337 pre-evidence plus forecast60-100 evidence LOC.
- Fully read REVIEW.md (319 lines), guidance-deltas.json and subject-inventory.json; exact stated SHA256 values match. Source at713184d matches all9 original/proposed prose hashes and all6 evidence-file hashes. Graph coverage metadata_match/no recorded gaps is best effort, not semantic completeness.
- Independent reviewer must inspect nine exact prose deltas and FULL covers/claim semantics of all14 records, including portfolio7-subject records and Go CRM executable-test assertion, provide actual identity/date/specific findings and approve provenance representation. Missing verification remains an evidence hold, never automatic rehash. All claim IDs/schema/membership/order/non-BUILD hashes/other records/acceptance/history preserved.
- Six baseline Gv passes are structural only; six stale-before-amendment, six final Gv and unchanged-golden compatibility remain mandatory. No source/test/docs/evidence/installed/worktree changes during this repair.
- CLI commit 713184db16a12b8c3763b4aa8f5bf025721abf22; frozen RED unchanged; focused 26 leaves: 17 PASS / 9 expected guidance FAIL / 0 SKIP; six actual Gv baselines PASS.
- Exact review artifact and SHA256 above. Nine guidance files and six evidence files remain untouched; independent amendment/historical-truth authorization pending.
- Healthy hold retains the atomic claim and clean worktree. No delivery, release, closure, renewed acceptance or current re-attestation claimed.

### proof
- [ ] AC #1: exact nine guidance amendments proposed; application and replay await authorization.
- [ ] AC #2: actual CLI help/output honesty passes; documentation changes pending.
- [ ] AC #3: executable quoted-space and valid indentation semantic controls, real equal-token calls and CLI honesty pass; guidance condition pending.
- [x] AC #4: genuine whitespace/empty utility, count prefix, exit/read-error and original mutation/custody controls pass unchanged.
- [ ] AC #5: actual standalone CLI and all nine real guidance cases execute without skips; full shipped contract, post-amendment Gv and unchanged-golden compatibility remain pending.



### 2026-09-06T02:53:50Z ramirosalas
# MAC-lnu6 independent amendment review

Reviewer: Codex independent PM reviewer, agent /root/pm_lnu6_red.
Review date: 2026-09-06 UTC (2026-09-05 America/Los_Angeles).
Review source: 713184db16a12b8c3763b4aa8f5bf025721abf22.
Immutable RED: 0ea1fdc730aadac15cecc8de33bb95898ad91d60.

## Decision

AUTHORIZE ONLY the nine exact old/new guidance replacements in
`/tmp/MAC-lnu6-proposal.NZsERt/guidance-deltas.json`, SHA256
`d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011`.
This authorizes those precise prose edits, +57/-31, with every other byte
preserved. No source, guidance, evidence or test was changed during this review.

HOLD ALL evidence amendments. No new attestor/date/note or BUILD cover hash is
authorized in any of the fourteen records. Complete current conformance claims
cannot be truthfully established from the present evidence. Some other claims
have positive design-review support, enumerated below; this does not authorize
partial evidence writes or certify the unsupported claims.

The story remains in_progress, hard-tdd/red-approved, claimed by dev-MAC-lnu6.
This is neither a GREEN review nor a deliver/accept/reject transition. The
independent RED approval remains intact. No test-edit authorization is granted.

## Exact authorized prose and provenance

Read all 319 lines of REVIEW.md, all guidance-deltas.json, and all
subject-inventory.json. Their SHA256 values independently match:

- REVIEW.md: 48889c0b61afae030244dcd90f394515a183f9e90d9126dffe0602ffbe31b21f.
- subject-inventory.json: e08121e72aa07e230276a1e1227f761a4d7b193c52af1f5be22e5f43dfc9c0e5.

Reviewed the nine replacements in their surrounding hard-TDD/adjudication
protocols, the full six covered BUILD documents, and all six portfolio packets.
For Surreal CRM, complete Go CRM source plus the complete zero-context diff
between the two documents established the unchanged text and every difference.
The replacements require exact bytes AND file inventory, explicit owner
authorization, a new evidence revision, RED plus all applicable gate replay,
and an immutable original revision. They remove the existing formatting/token
exception without allowing an implementer to edit tests to pass. Brownfield's
adjudication remains required; its new replay paragraph does not waive it.
The retained pre-lock “no legal remedy” language is a prohibition on editing the
locked revision, read with the new explicit new-revision process, not a surviving
token exception. No additional prose correction is authorized or required for
this particular formatting-exemption change.

Each old block occurs exactly once. Replacements computed in memory from the
independent checkout yield these exact prospective full-file hashes:

| Guidance path | Old SHA256 | Authorized proposed SHA256 |
|---|---|---|
| skills/machinery/references/build-md-template.md | 3de68ee2e0fccd5167b218d957095b7d45be5f6cfbad99efc3214923f74ce144 | bd00c17071f39b6956b9e0f9468349726c033606818db69c7c4034379262c6e6 |
| agents/machinery-build-writer.md | dede53d1088ee90d853bdca15d4dadcf92858eb134b7116a63708810ee973674 | 23bb84add3d714bc8d794c9246bd45c12b54d5055c05361966884b9ea2d9114e |
| docs/brownfield-team-guide.md | b70193e00f2767856d6e444c24901dd88b91398a0db44c4fac267c5fff0ff176 | ffca5be6bfe2d6debf114740ef299f8a636a7612cd34f3f95f19db43e35cb298 |
| examples/checkout-split/orders/design/BUILD.md | 04822dd9ab8e3e3386a99082f3bd720cd43d5830fea27c614cfaddd70442fb5a | 9cbed7b78871a1a4632867a63849d126177888abc89282dae36efa4e490fbd06 |
| examples/checkout-split/payments/design/BUILD.md | 5fc797d0edafbed7fe24d7429f55f6b257613737a557b13e2e420d4fbf861869 | 04f3980416e75e0bf185f349ad088118e45ee64221248ce85339609cc147eb47 |
| examples/fulfillment/design/BUILD.md | 24a8fd8a71a7504a6bd614520ee6ca5c6b72a0bd4386d2439c59042d91853baf | 3a2a2b9649c6a0d39e114c3b3236f589be618bf50a71ce039a9e2304f75cf61f |
| examples/go-crm/design/BUILD.md | c58124cf07f7f8cdf8f4e99b2cb484aec5d4ca110543e25b2ed0711863d44c1d | 4555d4da472129153a6650c48ac9b760fad7583fff5867516a0c1d69bc249ac4 |
| examples/portfolio-engine/design/BUILD.md | 2c1240dfd86c84c253cf4da5b08cd0f6c04d62c8cf29ea6267ea2f2e636aa38a | abd0d3c8a23fbefaa945128ad2ce763f08c8835443cb09847332da27f06c2178 |
| examples/surreal-crm/design/BUILD.md | 426bf956676e9bcf47c7559339c1f5fb9467397f90c183a828cbca124a0d298e | 1fa87c359b95f5ca10a820ebbec932a81068c0dee6175ce34c1431d3c4743e52 |

These are prose authorization hashes, NOT authorized attestation hash changes.
All six current attestation-file hashes independently match subject-inventory.
All six portfolio packet hashes match the complete covers lists in both records.

The proposed inline provenance representation is capable of being truthful:
the actual new reviewer/date and specific new judgment must be distinguished
from a verbatim historical attestor/date/BUILD hash/note (or explicit absence),
with the complete old record still available at immutable RED and in the hashed
inventory. A new reviewer cannot merely repeat the old claim. This representation
is not the blocker; missing support for current complete claims is. Therefore I
do not authorize instantiating that representation for the fourteen records now.
Do not prefill this report's reviewer/date as an attestor for a claim held below.

## Complete fourteen-record claim review

The current claim vocabulary at internal/gates/attest.go:151 describes
gt.conformance-test-shape as: a wholesale test parses the committed oracle table
and asserts each row's next state AND expected actions. The build-writer's
attestation instruction repeats this. It is not merely a stable-ID citation
claim, a future-plan claim, or proof supplied by Gv's hash checks.

| Design / record | Full covers reviewed | Finding |
|---|---|---|
| orders / gt.conformance-test-shape | BUILD.md | Unsupported. Section 7 requires every stable row, but neither a wholesale committed-table parser nor next-state AND expected-action assertions. No implementation/test tree exists in this child. |
| orders / g4.zero-context | BUILD.md | Positive design support: domain dictionary/invariants, command and event contracts, row-lock/outbox realization, failures, test obligations and ordered milestones are present. Conformance gap above remains a separate unresolved obligation; no renewed implementation acceptance. |
| orders / g4.standin-coverage | BUILD.md | Positive design support: its single payments neighbor is covered by a named PaymentsContract stand-in, stable-row oracle obligations, duplicate/reordered delivery, and an isolated Postgres/NATS/stand-in/fixtures recipe. Referenced parent contract exists. No compose or stand-in runtime was executed. |
| payments / gt.conformance-test-shape | BUILD.md | Unsupported for the same exact missing parser/next-state/action obligations as orders; no implementation/test tree exists. |
| payments / g4.zero-context | BUILD.md | Positive design support: Payment dictionary, settlement/error contracts, row-lock/outbox realization, dedupe, failures and ordered milestones are present. No implementation or production acceptance claimed. |
| payments / g4.standin-coverage | BUILD.md | Positive design support: its single orders neighbor is covered by an OrdersContract stand-in, stable-row oracle obligations and duplicate/reordering cases; isolated Postgres/NATS/stand-in/fixtures recipe is present. Parent contract exists. No runtime executed. |
| fulfillment / gt.conformance-test-shape | BUILD.md | Its section 8 lines 231-234 explicitly requires a wholesale parser, target state and complete ordered actions for all rows. This supports a prospective design obligation. The example is explicitly design-only with no impl, so it cannot establish that the current test actually exists and performs the claim. No plan/current-claim narrowing authorized. |
| fulfillment / g4.zero-context | BUILD.md | Positive design support: six machine/oracle/matrix sources, twenty-five-invariant traceability, concrete service milestones, ExUnit/OTP/Elixir and real-boundary requirements, migration and failure recovery are laid out. References supply full source detail as expressly declared by the handoff. No Elixir, service or TLC replay performed. |
| go-crm / gt.conformance-test-shape | BUILD.md plus inspected current implementation | Unsupported as the complete current claim. All five handwritten FSM tables run and cover all 197 current stable IDs by executed name, asserting state and ordered action containment. They do not parse the committed FSM oracles and do not reject arbitrary extra actions. See actual replay and assertion analysis below. |
| go-crm / g4.zero-context | BUILD.md | Source locations, dictionary, interface signatures/errors, five lifecycles, test obligations, migration, toolchain and milestone ordering are present. Caveat: the text mixes a prototype migration plan with a greenfield/no-production-data statement and has a stale x/crypto v0.53 pin while go.mod is v0.55 with the latter declared authoritative. These are existing clarification/drift concerns; this review does not certify complete zero-context correctness or renew its record while they remain unresolved. |
| portfolio-engine / gt.conformance-test-shape | BUILD.md and all six M0-M5 packets | Unsupported as a current executable claim: no implementation exists. Even the prospective claim is incomplete: M4 says parse every row and assert ordered actions but omits explicit target-state assertion; M5 names twelve parsed rows and timeout/error recording but does not specify next state and complete ordered actions. Root return signatures describe implementation output, not the required assertion. M3 is intentionally a pure transform, so it needs properties, not an invented machine. |
| portfolio-engine / g4.zero-context | BUILD.md and all six M0-M5 packets | Unsupported under the stated packet-alone claim. M3 does not state minimization of maximum drawdown, exact sum 10000 bps or a concrete numeric tolerance/representation; it instead refers to a chosen representation/domain tolerance. These facts live in the root or elsewhere. Other packets reference domain schema/ports without exact local forms. This review does not silently substitute root-plus-packet execution for the explicit independent packet-alone promise. |
| surreal-crm / gt.conformance-test-shape | BUILD.md | Unsupported. Section 7 specifies stable-row derivation and policy/tenant decision parsers, but no wholesale FSM-table parser/assertions. The legacy Go CRM test reference does not close that gap; it is handwritten and accepts extra actions. This example has no target implementation. |
| surreal-crm / g4.zero-context | BUILD.md | Positive design support: target repository interface, typed store/container failures, unchanged domain lifecycles, migration phases, contracts, toolchain, milestones and runbook/deferred command distinction are present. This does not establish runtime migration, container behavior or acceptance. |

“Positive design support” records the substance found, not a narrowed claim or a
current attestation authorization. The full fourteen-record request is held.

## Go CRM runtime and assertion evidence

PM-owned independent detached checkout:
`/Users/ramirosalas/workspace/machinery/.claude/worktrees/pm-MAC-lnu6-amendment`.
It remains clean at 713184d. Frozen RED and all existing tests are unchanged.

Exact synchronous command, from that checkout's examples/go-crm/impl:

```sh
timeout 150s go test -count=1 -timeout=120s ./internal/domain ./internal/session ./internal/cli -run '^(TestDealTransitions|TestTaskTransitions|TestUserTransitions|TestSessionTransitions|TestCommandExecutionTransitions)$' -json
```

Terminal exit 0, wall 1.503486s. 218 executed leaves, 218 PASS, 0 FAIL, 0 SKIP:
Deal 75, Task 35, User 20, Session 60, CommandExecution 28.
Package durations: domain 0.590s; session 0.361s; cli 0.917s.
Raw log preserves all test names:
`/tmp/MAC-lnu6-pm-go-crm-transitions-713184d.jsonl`, SHA256
`8a88436b9d6212a45eb422c4f4d609902679a15343ba796b1c72174aff03a662`.

Independent text-set comparison of the five committed FSM oracle files to
PASS subtest names found 197 distinct committed IDs and zero missing names.
This is executed-name coverage, not parser linkage or independently recomputed
assertion correctness. Compound guards account for additional cases. This run
did not select terminal-exit supplements, authz/tenant, repository, end-to-end,
full suite, coverage instrumentation or external services.

Concrete assertions are in domain/deal_test.go:160-168, task_test.go:110-118,
user_test.go:73-81, session/machine_test.go:153-161 and cli/command_test.go:118-126.
They call the actual Fire methods and compare the resulting State.
The domain firedInOrder at deal_test.go:213 and analogous session/CLI helpers
test whether wanted actions are an ordered SUBSEQUENCE of actual actions.
Thus actual [unexpected, commitStage] passes expected [commitStage], and every
actual list passes an empty expected list. This conclusion follows directly
from the reviewed loop; it is not a claimed mutant runtime run.

The local comments expressly choose containment because state-entry actions
are not in the transition-action column. Legitimate entry/exit actions must be
accounted for using the committed machine/oracle semantics. That explanation
does not establish a license for arbitrary additional effects or prove the
complete expected action sequence. The claim vocabulary says expected actions;
it does not expressly say exact-list equality. Therefore the definite claim
failure is absent wholesale committed-table parsing. Action completeness is an
additional unresolved semantic adequacy gap, not an invented quotation from
the registry. The proposal/historical phrase “ordered actions” alone does not
turn containment into exact equality.

The only implementation oracle parsers found by bounded source search are
authz/oracle_test.go and tenant_oracle_test.go for Policy and Isolation decision
tables. They do not prove the five FSM tables. Earlier authz evidence on another
Machinery revision was not used as proof here.

## Scope-owner findings and minimum next proof

DISCOVERED_BUG:
  title: Existing BUILD conformance attestations overstate current FSM oracle linkage
  discovered_during: MAC-lnu6 substantive amendment review
  affected_claims: gt.conformance-test-shape in all six owned example attestations.yaml files
  affected_files: six owned BUILD.md files; portfolio BUILD/M4-portfolio-review.md and M5-reference-operations.md; go-crm impl/internal/domain/deal_test.go, task_test.go, user_test.go, internal/session/machine_test.go, internal/cli/command_test.go; existing claim vocabulary internal/gates/attest.go
  expected: a current wholesale conformance test parses committed oracle rows and checks next state and expected actions; a design obligation must not masquerade as existing executable proof
  present: 218 native Go FSM leaves PASS with all 197 current IDs named, but handwritten inputs/expectations and ordered-subsequence action checks; five other examples have no implementation, with missing or incomplete prospective obligations except fulfillment
  gap: ID presence and Gv hash currency do not bind expected state/actions to current committed tables; plan-only examples do not establish present test execution
  minimum_proof: review the exact current claim semantics; for executable claims produce parser-backed row/guard/input/next-state/action reconciliation and complete executed row inventory, explicitly account for entry/exit actions and reject unexpected effects; for design-only claims a separately reviewed truthful plan/current distinction is needed before a current attestation can be represented
  ownership_overlap: MAC-p7jd may address plan/current/historical semantics; concrete Go CRM/example test-shape and portfolio packet edits exceed this MAC-lnu6 wording-only boundary and require scope-owner triage; no dependency, future delivery or test edit is authorized by this report

DISCOVERED_BUG:
  title: Portfolio packet-alone zero-context claim lacks concrete optimizer obligation
  discovered_during: MAC-lnu6 full portfolio covers review
  affected_claims: portfolio-engine g4.zero-context
  affected_files: examples/portfolio-engine/design/BUILD.md and BUILD/M3-optimizer.md (reviewed all M0-M5 packets)
  expected: each milestone can be executed from its packet alone under the explicit root promise
  present: M3 states selection/properties but omits minimum maximum-drawdown objective and exact 10000-bps allocation, referring to a chosen representation or domain tolerance
  minimum_proof: explicitly review and supply the missing exact obligations within the packet or separately authorize a truthful root-plus-packet contract; review all packet subjects under the chosen semantics
  ownership_overlap: example/packet scope-owner work outside the nine exact approved policy replacements; do not amend the packet or relabel the claim under MAC-lnu6 without canonical review

No automatic backlog expansion, dependency, test repair, or future MAC-p7jd /
MAC-vx24 assurance is inferred. The new policy is not the cause of these old
claim deficiencies; it exposed the need to repeat their substantive judgments.

## Cost, limits and next execution

Review read the complete proposal and full covered BUILD/packet content,
current registry and relevant current implementation assertions; one native
218-leaf transition replay was performed. Graph discovery was attempted first;
the graph returned the claim definitions but no matching narrowed implementation
symbols. Coverage later reported generation 2026-09-06T02:42:16Z, metadata matches
for material paths, excluded detached .claude worktree and unrelated partial
formal config ranges. Exact source and tracked-file inventory supplied bounded
negative evidence; no graph completeness claim or formal proof is made.

The CLI-only 713184d diff was directly reviewed: comments/help/equality wording
only, comparison/custody untouched. Its prior 26-leaf 17-pass/9-held-guidance-fail
run and six baseline Gv passes are supporting historical stage evidence, not
repeated or substituted for this review. No installed binary was used/replaced.

Forecast remains 337 changed lines before evidence (229 RED, 20 CLI, 88 prose),
17 paths if the evidence problem is subsequently resolved. No evidence-line
forecast can be treated as an approved implementation now. The broader test or
packet/claim corrections above need independent scope review and costing.

After applying the authorized prose, capture all six actual stale Gv outcomes
before any later authorized evidence amendment. Complete final Gv outcomes,
focused unchanged RED GREEN, unchanged-golden TestGoldenCheck and final direct
source/remaining-policy scan remain owed. Do not deliver incomplete GREEN or
conceal the resulting stale evidence. Preserve all record IDs/schema/order,
covers membership/order, non-BUILD hashes, other records, acceptance files and
history. Main/epic, installed assets, user services and other worktrees were not
modified. No process remains running.

## nd_contract
status: in_progress

### evidence
- Independent bounded amendment review completed at 713184db16a12b8c3763b4aa8f5bf025721abf22, immutable RED 0ea1fdc730aadac15cecc8de33bb95898ad91d60 unchanged.
- Exactly nine prose replacements authorized by guidance-deltas SHA256 d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011 and full-file hashes above; none applied during PM review.
- All fourteen records reviewed against full covers, including all seven portfolio subjects. All six evidence files remain held; no attestor/date/note/BUILD-hash write authorization. Missing current wholesale FSM parsing and action-completeness proof, design-only/current-claim mismatch, and packet/context concerns recorded explicitly above.
- Independent native Go CRM transition replay: 218 PASS, 0 FAIL, 0 SKIP; all 197 committed FSM IDs present in executed names; three package durations 0.590/0.361/0.917s, command wall1.503486s; raw log and SHA256 above. This is bounded transition proof, not wholesale parser or full implementation acceptance.
- Detached PM checkout retained clean; source/tests/goldens/evidence/main/epic/installed assets/user services unchanged. No state transition, release, new dependency or backlog expansion; existing in_progress/hard-tdd/red-approved/dev-MAC-lnu6 claim preserved.

### proof
- [ ] AC #1: nine exact policy changes authorized; actual application and immutable-suite replay pending, evidence amendments held.
- [ ] AC #2: CLI-only wording reviewed; authorized documentation still unapplied.
- [ ] AC #3: existing executable semantic counterexamples remain frozen; actual shipped guidance replay pending.
- [x] AC #4: prior recorded CLI utility/exit/custody controls remain unchanged; no comparison implementation change found.
- [ ] AC #5: final all-nine contract, six stale and six final Gv outcomes, unchanged-golden TestGoldenCheck and GREEN PM acceptance remain owed.

### 2026-09-06T03:05:28Z ramirosalas
SCOPE-OWNER TRIAGE OF INDEPENDENT SUBSTANTIVE CLAIM FAILURES
Created P0 MAC-uzxr for actual Go CRM FSM parser/action-adequacy repair (five transition tests plus bounded test support; no BUILD/evidence writes). Created P0 MAC-lhu5 for source-established portfolio packet-alone omissions (six packets plus semantic/handoff test; root BUILD/evidence read-only). Both are contained in MAC-ui8a and explicitly block final MAC-ou97. Neither establishes repaired behavior today.
Core plan/current/historical distinction and exact v2 schema/CLI remain owned by active MAC-p7jd; no edits to its scope/frozen bar. MAC-l7m0/MAC-vx24 own future execution-authentication/replay, not proof of current examples.
All fourteen MAC-lnu6 evidence records remain held. The PM authorized only the nine exact prose deltas d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011; no broader claims/metadata/hashes or tests are authorized. New packet changes would alter currently protected non-BUILD hashes, so they require a serialized consumer migration/proposal revalidation, not an automatic hash refresh.
Pending dispatcher ownership decision: a consumer migration after MAC-p7jd/MAC-uzxr/MAC-lhu5 must reconcile six BUILD/YAML files, truthful design-only plan versus reviewed actual Go CRM current scope, exact historical provenance and affected golden expectations. V2 requires kind on every row, so this cannot be smuggled through the current fourteen-record metadata-only boundary. Explicit ownership/dependency/verification reconciliation is required before shared-file writers dispatch; no lnu dependency or ownership transfer added in this note.
Exact Go CRM BUILD concerns also need that bounded consumer clarification: migration.yaml declares mode rebuild/prototype phases but section8 calls the whole design greenfield; BUILD declares impl/go.mod authoritative yet lists x/crypto v0.53.0 versus actual v0.55.0. Correct wording against local authoritative files, not a library upgrade, migration implementation or external factual claim. Existing review cannot certify these away.


## nd_contract
status: in_progress

### evidence
- Sr PM triage created MAC-uzxr and MAC-lhu5 as separate P0 capstone prerequisites; source-established defects are not current repairs. Six BUILD/evidence consumer migration, serial ownership and future v2/golden contract reconciliation remain a dispatcher decision. No change to this story's claim/status/AC/frozen tests/dependencies and no evidence-write authorization.
- Independent bounded amendment review completed at 713184db16a12b8c3763b4aa8f5bf025721abf22, immutable RED 0ea1fdc730aadac15cecc8de33bb95898ad91d60 unchanged.
- Exactly nine prose replacements authorized by guidance-deltas SHA256 d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011 and full-file hashes above; none applied during PM review.
- All fourteen records reviewed against full covers, including all seven portfolio subjects. All six evidence files remain held; no attestor/date/note/BUILD-hash write authorization. Missing current wholesale FSM parsing and action-completeness proof, design-only/current-claim mismatch, and packet/context concerns recorded explicitly above.
- Independent native Go CRM transition replay: 218 PASS, 0 FAIL, 0 SKIP; all 197 committed FSM IDs present in executed names; three package durations 0.590/0.361/0.917s, command wall1.503486s; raw log and SHA256 above. This is bounded transition proof, not wholesale parser or full implementation acceptance.
- Detached PM checkout retained clean; source/tests/goldens/evidence/main/epic/installed assets/user services unchanged. No state transition, release, new dependency or backlog expansion; existing in_progress/hard-tdd/red-approved/dev-MAC-lnu6 claim preserved.

### proof
- [ ] AC #1: nine exact policy changes authorized; actual application and immutable-suite replay pending, evidence amendments held.
- [ ] AC #2: CLI-only wording reviewed; authorized documentation still unapplied.
- [ ] AC #3: existing executable semantic counterexamples remain frozen; actual shipped guidance replay pending.
- [x] AC #4: prior recorded CLI utility/exit/custody controls remain unchanged; no comparison implementation change found.
- [ ] AC #5: final all-nine contract, six stale and six final Gv outcomes, unchanged-golden TestGoldenCheck and GREEN PM acceptance remain owed.


### 2026-09-06T03:21:31Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Dispatcher-authorized serial graph: MAC-uzxr + MAC-lhu5 + MAC-p7jd -> MAC-hgz1 -> MAC-lnu6; MAC-ou97 also directly depends on the consumer. Existing lnu claim/dev-MAC-lnu6, status and hard-tdd/red-approved labels retained. This is shared-file sequencing, not new core architecture or proof.
- MAC-hgz1 exclusively owns six BUILD/evidence migration/context corrections and only exact-before-edit PM-authorized affected golden updates. It preserves all nine old policy blocks; MAC-lnu6 alone owns policy replacements afterward. No concurrent shared-file writers.
- Original five AC, frozen RED0ea1fdc730aadac15cecc8de33bb95898ad91d60 and CLI713184db16a12b8c3763b4aa8f5bf025721abf22 remain intact. No old-test/golden edits by lnu; upstream accepted golden changes require separately recorded authority/provenance.
- All14 evidence amendments remain HELD today. Prior nine-block prose approval at713184d is historical; fresh exact post-migration prose AND full-claim/evidence review is required before new writes. No reviewer/date prefill or automatic schema/kind/scope/non-BUILD-hash edits.
- Accepted upstream source/schema/subject/golden baseline is consumed explicitly; retain historical original logs and rerun all6 actual baseline,6 stale-before-amendment and6 final Gv outcomes with honest plan/current warning limits. Current Go CRM uses actual --impl; no invented implementation for design-only examples. Unchanged-golden compatibility means the accepted upstream expected bytes, not silent mutation of old baselines.
- Scope stays17 paths/~500 lnu-only forecast; upstream migration budget is separately reported. No source/test/docs/evidence/worktree/installed asset changes by this triage.

### proof
- [ ] AC #1: original nine policy amendments and actual frozen-suite replay after accepted migration/fresh review.
- [ ] AC #2: retained CLI honesty with final truthful documentation.
- [ ] AC #3: frozen semantic/utility controls and final actual guidance checks.
- [x] AC #4: prior reported CLI utility/exit/custody controls remain unchanged; final compatibility replay still owed.
- [ ] AC #5: actual CLI/all-nine contract plus new six baseline/stale/final checks and accepted upstream golden compatibility; no Paivot dependency.


### 2026-09-06T23:26:03Z ramirosalas
ACCEPTED 2026-09-06 (user decision 2a) — 3 commits af4f1ae..274e53a over df8dce0. Frozen RED 0ea1fdc retained byte-exact; nine byte-exact policy replacements applied (4 match PM prospective hashes, 5 differ only by intervening hgz1 edits); fourteen evidence rows: BUILD-hash refresh with pinned attribution preserved (user accepted — substantive review is this acceptance); v2 schema/kinds/inventories byte-preserved. Upstream suites ok; isolated CLI checks green on five design-only examples; go-crm Gv green with pre-existing G4 ERROR (proven at baseline). 15 files +71/-45. Coordinator merged to epic + targeted suites ok. Record: .git/machinery-evidence-20260906.TEFZ7D/lnu6-record.md
