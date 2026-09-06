---
id: MAC-hgz1
title: "Migrate repaired example claims to truthful plan and current evidence"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-06T03:16:50Z
created_by: ramirosalas
updated_at: 2026-09-06T05:46:14Z
content_hash: "sha256:83540326da1fac515f297ad127f62e9a3d5075f7cb02f5659418faf935c24f13"
blocked_by: [MAC-uzxr, MAC-lhu5, MAC-p7jd]
blocks: [MAC-lnu6, MAC-ou97]
---

## Description
## USER INTENT
Shipped examples must distinguish planned tests from actual current conformance review, and their BUILD handoffs and evidence must accurately describe the repaired subjects rather than silently carrying old unsupported claims forward.

## Context (Embedded)
P0 consumer repair discovered in MAC-lnu6's independent substantive review: the original six reviewed legacy gt.conformance-test-shape records overstate current wholesale FSM oracle linkage. Go CRM's actual218 passing transition leaves and197 named stable IDs were not a parser-backed complete current claim; MAC-uzxr repairs that test support. The five other examples in that original review have no target implementation; fulfillment has a sound prospective obligation, but a plan is not a currently executing suite. MAC-lhu5 repairs six portfolio packet-alone deficiencies, including M3 objective/units and M4/M5 conformance obligations.
Existing approved MAC-p7jd R2 already defines closed v2 plan/current/historical classification, complete rooted implementation inventory and honest non-authenticated hash limits. This story consumes its ACCEPTED implementation; it does not redesign that core contract. MAC-l7m0/MAC-vx24 remain owners of future execution authentication/replay enforcement.
Diagnostic authority: /tmp/MAC-lnu6-PM-AMENDMENT-713184d.md SHA2566fc5223d3e5d48170f89ff9e29b634b256bfe2454b710378f08cb60d875aadf1. All14 affected existing BUILD records remain HELD today; no actual new reviewer/date/note/hash or migration write is authorized by this backlog creation.

## Ownership and serialization
Exactly41 forecast paths: six BUILD files, eight evidence files, twenty-four conditionally affected golden outputs, one new native regression test and two existing test files limited to the serialized fixture-helper adaptations below. The committed bundled inventory correction below adds two evidence documents and six conditional golden files; it does not add BUILD repair scope.
- examples/checkout-split/orders/design/BUILD.md
- examples/checkout-split/orders/design/attestations.yaml
- examples/checkout-split/payments/design/BUILD.md
- examples/checkout-split/payments/design/attestations.yaml
- examples/fulfillment/design/BUILD.md
- examples/fulfillment/design/attestations.yaml
- examples/go-crm/design/BUILD.md
- examples/go-crm/design/attestations.yaml
- examples/portfolio-engine/design/BUILD.md
- examples/portfolio-engine/design/attestations.yaml
- examples/surreal-crm/design/BUILD.md
- examples/surreal-crm/design/attestations.yaml
- testdata/golden/check-checkout-split-orders/stdout.txt
- testdata/golden/check-checkout-split-orders/stderr.txt
- testdata/golden/check-checkout-split-orders/exitcode.txt
- testdata/golden/check-checkout-split-payments/stdout.txt
- testdata/golden/check-checkout-split-payments/stderr.txt
- testdata/golden/check-checkout-split-payments/exitcode.txt
- testdata/golden/check-fulfillment/stdout.txt
- testdata/golden/check-fulfillment/stderr.txt
- testdata/golden/check-fulfillment/exitcode.txt
- testdata/golden/check-go-crm/stdout.txt
- testdata/golden/check-go-crm/stderr.txt
- testdata/golden/check-go-crm/exitcode.txt
- testdata/golden/check-portfolio/stdout.txt
- testdata/golden/check-portfolio/stderr.txt
- testdata/golden/check-portfolio/exitcode.txt
- testdata/golden/check-surreal-crm/stdout.txt
- testdata/golden/check-surreal-crm/stderr.txt
- testdata/golden/check-surreal-crm/exitcode.txt
- examples/checkout-split/parent/design/attestations.yaml
- examples/pii-flow/design/attestations.yaml
- testdata/golden/check-checkout-split-parent/stdout.txt
- testdata/golden/check-checkout-split-parent/stderr.txt
- testdata/golden/check-checkout-split-parent/exitcode.txt
- testdata/golden/check-pii-flow/stdout.txt
- testdata/golden/check-pii-flow/stderr.txt
- testdata/golden/check-pii-flow/exitcode.txt
- cmd/machinery/example_attestation_migration_test.go (new).
- internal/hook/hook_test.go (only copyTree temporary-fixture adaptation, exact independent PM review before edits).
- internal/gates/obligation_ownership_test.go (only obligationParentFixture temporary-fixture adaptation, exact independent PM review before edits).
Golden scope is conditional: change only the exact files/lines independently PM-authorized BEFORE edits for genuine approved migration diagnostics/counts; retain every historical baseline, command and old expected output. No wholesale golden recapture, acceptance of arbitrary failure, cmd/machinery/golden_test.go changes, or mutation of MAC-lnu6/MAC-p7jd frozen tests outside the two later independently exact-before-edit reviewed fixture-helper bodies explicitly scoped below. All original assertions remain frozen.
Exclusive serial custody: this story runs only AFTER MAC-uzxr, MAC-lhu5 and MAC-p7jd are accepted, then releases the six BUILD/evidence paths to dependent MAC-lnu6 for its final policy/evidence revalidation. No dependency on MAC-lnu6, MAC-vx24 or MAC-l7m0; this avoids a cycle and does not wait for future runner authentication. MAC-lnu6's healthy claim/worktree stays retained, but its shared-file writes are paused until this story is accepted.
MAC-lnu6 alone owns its nine exact policy prose replacements, frozen RED0ea1fdc730aadac15cecc8de33bb95898ad91d60 and CLI713184db16a12b8c3763b4aa8f5bf025721abf22. This consumer MUST preserve the exact existing old replacement blocks in all six BUILD files byte-for-byte and uniquely located; it must NOT apply, paraphrase, delete or move their ownership, nor change the three non-BUILD policy files. Use /tmp/MAC-lnu6-proposal.NZsERt/guidance-deltas.json SHA256d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011 to identify those exact old/new blocks. If a concrete correction overlaps a preserved block, stop for exact canonical/PM review rather than silently widening permission.
Read-only: core gates/CLI/designlock/hook implementations, packet files owned by MAC-lhu5, Go CRM tests/implementation owned by MAC-uzxr, all generated oracles/machines, acceptance files, go.mod/go.sum and other examples/goldens. No settings/install/remote/preflight changes. You are not alone; preserve accepted upstream work.

## Complete bundled inventory and frozen-regression separation
Exact committed70652b948bf090008b1965c85daf36ea374daea4 and paused51454e069ebe4039f02d6d9108acf9354c7ad6c8 tree inventories contain EIGHT examples/**/attestations.yaml documents, all v1: checkout-split/orders14 rows, checkout-split/parent8, checkout-split/payments14, fulfillment12, go-crm13, pii-flow12, portfolio-engine12, surreal-crm12 =97 rows. Each has one gt row; orders/payments each add pack-event-discipline and standin-coverage, for12 behavioral rows total. Only go-crm has ga.review-quality (one historical row). The six-file original review was not an exhaustive bundled inventory.
Checkout parent and pii-flow are added to this same migration owner, not to MAC-lnu6. Their committed BUILD.md files remain READ-ONLY: parent is a recursive manifest delegating child conformance/assembly, pii-flow already prospectively requires runtime parsing with next state/actions. Preserve those sources, generated inputs, Gk evidence and external-checker residuals; no target application implementation exists in the bounded committed-tree inventory for these two. Review their complete8/12-row documents substantively under the same v2/real-review/provenance rules. The parent gt becomes an explicitly reviewed plan for delegated future tests, never proof that the synthetic parent test fixture or deployed children provide current conformance. Pii-flow behavioral gt is prospective plan; Gk projection/checker evidence is not DataSubject application execution. If broader content changes prove necessary, return exact gaps before extending BUILD or checker scope.
Only the two standard check golden triplets above are added. The separate check-pii-flow-gk corpus explicitly selects --gate gk in cmd/machinery/checker_example_golden_test.go and remains read-only; no evidence that Gv migration should change it. Existing TestGoldenCheck and TestGoldenCheckCheckoutSplit remain exact and keep both parent/default and child selection coverage.
Serialization remains accepted MAC-p7jd + MAC-uzxr + MAC-lhu5 -> MAC-hgz1 -> MAC-lnu6, with final MAC-ou97 downstream. No reverse p7 dependency. MAC-lnu6 still owns exactly its original six BUILD/evidence pairs, fourteen conditional rows, nine policy blocks and17 total paths; the two added evidence documents never transfer to it. Updated eight-example proof is hgz1-owned; lnu6 still owes its six baseline/stale/final stages plus unchanged-golden compatibility.
The four paused p7 legacy regression leaves are repaired first under that story only if its exact65-line temporary-fixture proposal receives independent PM authorization. This story consumes ACCEPTED p7 before changing bundled examples. Both accepted constructors deliberately guard attestation_version:1; migrating GoCRM and checkout-parent source documents to v2 WILL break those guards across13 current helper-caller leaves. This source-confirmed consequence gives hgz1 exclusive LATER ownership inside exactly copyTree and obligationParentFixture, not authority to edit them now or unblock p7 through a reverse dependency. Exact before/after hunks must be independently PM-authorized BEFORE EDITS at the actual accepted hgz1 baseline, preserving all original assertions and independently frozen semantic intent. Full current review requires real complete scope and substantive evidence; renewed hashes are not renewed reviews.

## Serialized temporary-fixture adaptation contract
Own only the later adaptation within func copyTree(t *testing.T,src,dst string) in internal/hook/hook_test.go and func obligationParentFixture(t *testing.T)(string,string) in internal/gates/obligation_ownership_test.go. All13 current affected native leaves remain owed: six Stop/ledger/drift/wave/staged-impl/reaped callers; parent selection Policy/Isolation, CLI obligation-free/Policy/Isolation, deleted-relational gp/gn. The latter obligation_ownership_green_test.go remains read-only. No broader old-test/golden scope, new helper framework, selector/seam/production change, or arbitrary fixture replacement is authorized.
The future v2 fixture construction is deliberately not chosen before the actual migrated baseline and independent exact-text review. GoCRM shipped gt will be current over its reviewed implementation, whereas these design/ledger fixtures must retain explicit prospective intent, missing-current warning and current-review count0; parent decision-ID coverage is not substantive implementation review. Do not automatically strip a current manifest/relabel a current review as plan, fabricate a review, reuse old attribution as a fresh judgment, or refresh hashes to make tests pass. The proposal must explicitly distinguish immutable source-example evidence from independently reviewed temporary fixture semantics, preserving original reviewer/date/note provenance and all unaffected covers/acceptance bytes/input inventory. Any unavoidable representation or semantic conflict returns for a bounded technical decision BEFORE editing; no automatic downgrade rule.
Preserve exact original silent Stop then ledger clearing, default-versus-explicit Gt selection and zero-obligation counts, real covered controls and missing-ID oracle/ID/Gt negatives, Ga ancestry/history and production no-grandfathering. Review full before/after design+implementation+acceptance inventories: baseline GoCRM62files/6acceptance files and parent73files control/75files decision cases are historical reference inventories, not ceilings; enumerate the complete actual migrated baseline and every authorized difference. No deletion/exclusion/ancestry rewrite or new currentimplementation claim is justified by ordinary gate success.
Fresh exact PM review at this story baseline is required even if p7's initial patch was authorized. Preserve p7 initial patch/replay history and accepted source hashes, then run all13 unchanged caller leaves on the accepted migrated candidate, retaining raw native names/outcomes, current0/plan-warning/Ga evidence, silent/ledger observation, and default/explicit controls plus intended negative causes. Setup/guard failures or unrelated errors do not satisfy this proof. These13 replay cases add no new test paths and cannot be reduced to four previously failing leaves.

## Boundary Map
PRODUCES:
- Six exact BUILD.md files -> complete truthful prospective conformance/context obligations, preserving MAC-lnu6's exact pending policy blocks.
- Eight exact attestations.yaml files -> explicitly reviewed v2 classification and subject binding with old provenance preserved, no invented reviewer or execution.
- Twenty-four exact conditional golden files -> only individually PM-authorized expected output changes from accepted schema/claim semantics, preserving historical baseline.
- cmd/machinery/example_attestation_migration_test.go -> real eight-example migration and current/plan/history positive/negative proof.
- internal/hook/hook_test.go -> only serialized copyTree temporary-fixture adaptation under later exact independent PM authorization.
- internal/gates/obligation_ownership_test.go -> only serialized obligationParentFixture temporary-fixture adaptation under later exact independent PM authorization.
CONSUMES:
- MAC-p7jd: internal/hook/hook_test.go and internal/gates/obligation_ownership_test.go -> accepted conditional fixture amendment.
  spec: copyTree(t *testing.T,src,dst string); obligationParentFixture(t *testing.T)(string,string). Consume exact accepted helperconstruction/commit, independent amendment disposition, full13-caller same-SHA blast proof and complete input/acceptance/provenance inventory; current external patch65add/0delete ae6e9241ea5d7b0c4556b2ee02c6e2910b2663c2452761e7984f1c6663252971 is a pending proposal, not accepted source.
- MAC-p7jd accepted closed v2 implementation and docs/attestation-evidence.md.
  spec: gates.RenderAttestation(design, impl string, review AttestationReview) ([]byte,error); AttestationReview{Claim,Kind,Attestor,Date,Note string}; gates.CheckAttestationsWithImplementation(design, impl string) *Gate are approved outputs, unavailable on current70652b948; invoke only after upstream acceptance.
  endpoint: machinery attest --design <design> --claim <claim> --kind <plan|current|historical> [--impl <root>] --attestor <actual-reviewed-attribution> --date <actual-review-date> [--note <verified-note>]; machinery check <design> --gate gv [--impl <root>] [--warnings-as-errors].
- MAC-uzxr accepted Go CRM conformance repair.
  source: five real parser-backed FSM suites plus internal/testoracle support; consume exact accepted source/test SHA,197-row-baseline reconciliation and actual safe/unsafe evidence, not the old218-pass name inventory as sufficient proof.
- MAC-lhu5 accepted six portfolio packets.
  source: complete local packet obligations including optimizer pure properties, exact units/allocation and M4/M5 next-state/action requirements; consume exact old/new packet hashes and independent handoff proof.
- Existing unchanged CLI test harness.
  spec: goldenBin(t *testing.T) string; runBin(t *testing.T,args ...string)(string,string,int); repoRootDir(t *testing.T) string in cmd/machinery/golden_test.go; actual built binary/isolated config.
- MAC-lnu6 preserved policy handoff.
  source: exact nine old/new prose blocks identified above; frozen RED and CLI remain downstream owner assets, not this story's editable files.

### Story Acceptance Criteria
1. Correct six BUILD conformance/context statements using actual accepted subjects, without applying MAC-lnu6 policy edits. Orders/payments/surreal must explicitly require wholesale committed-FSM parsing, actual row/guard-input reconciliation, next state and expected actions including legitimate entry/exit semantics; fulfillment's already sound prospective obligations remain intact. Portfolio root must consistently describe the accepted repaired packets without weakening packet-alone handoff or inventing an M3 FSM. Go CRM accurately describes the accepted actual parser-backed suite and its bounded evidence, not authenticated/full-implementation acceptance.
2. Resolve Go CRM's concrete zero-context wording drift against local authority: distinguish declared rebuild/prototype migration and disposable seed/nonproduction context from the erroneous whole-design greenfield assertion; align the stale x/crypto v0.53.0 prose with authoritative impl/go.mod v0.55.0 at the reviewed source, preserving the latter as authority. No dependency upgrade, claim that production migration ran, or invented historical customer-data fact. If necessary facts remain ambiguous, record exact missing technical clarification before current review.
3. Migrate each of the eight entire owned YAML documents to accepted p7 v2, adding a valid kind to EVERY existing row under its exact closed classification. Preserve all claim IDs/row order/covers membership and historical acceptance facts. Design g2/g3/g4.zero-context are plan; ga.review-quality is historical; behavior gt/pack/standin are current only with substantive current review and complete implementation subject, otherwise explicitly reviewed prospective plan where the design-only status is true. For seven design-only targets (the original five plus checkout-split/parent and pii-flow) the affected behavioral rows are plan and visibly cannot discharge current implementation obligations. Go CRM gt requires actual independent review of the accepted parser-backed implementation/test scope before current generation; do not recast it merely to evade required verification. Any other unsupported purported current claim is held and reported, never silently grandfathered.
4. Independent PM substantively reviews complete current covers and intended claim statements BEFORE authorizing exact evidence edits or providing actual reviewer/date/specific findings. Use actual accepted CLI generation for authoritative covers/full-root-v1 inventory/hash; merge its single-row documents under closed schema, do not hand-fabricate inventories or exclusions. Preserve original reviewer/date/note/hash provenance and immutable old records/history; no blanket metadata renewal or automatic use of this triage/review author's identity. Only needed BUILD hashes and accepted changed portfolio packet hashes may refresh; every other nonchanged subject/hash remains exact. The six packet hash updates are this consumer's explicitly reviewed exception to MAC-lnu6's later non-BUILD immutability boundary, not continuing permission.
5. Actual isolated CLI tests on all eight migrated examples establish ordinary valid plan/current/historical outcomes and exact warnings/counts. Design-only behavioral plans have missing-current warnings and zero current-review credit; warning promotion blocks them for that intended reason. Go CRM's reviewed unchanged current scope succeeds with --impl, then real test/handler/config mutation and add/remove/rename invalidate that scope for the accepted p7 categories, with matching safe control. Historical acceptance remains historical; no ordinary plan Gv success is claimed as complete or execution-authenticated assurance. Wrong kind, incomplete inventory and stale BUILD/packet negatives reach their intended categories, not earlier missing fixtures/unsupported future flags.
6. Only independently exact-before-edit authorized golden changes encode truthful migrated outcomes. Each diff must be linked to actual same-SHA stdout/stderr/exit evidence and reviewed expected category; no failure masking or empty/missing-row positive. Rerun unchanged GoldenCheck test code against all affected expected outputs and retain historical original outputs. Hand off accepted schema/subject/golden baseline, complete proposal/hash/claim inventory, actual review evidence and uniquely preserved nine policy blocks to MAC-lnu6. It must rerun all six baseline/stale/final stages and independently re-review prose/evidence after its own edits; this story does not complete any MAC-lnu6 AC or authorize downstream writes.

## Testing Requirements
Hard TDD scoped to consumer fixtures/behavior AFTER accepted upstream interfaces: new test file uses existing real CLI, filesystem and local Git, not missing symbols/flags as RED. Baseline on accepted p7 with the old example fixtures must fail because claims lack correct migration/subjects or BUILD obligations, with separately valid generated plan/current controls. Independent PM reviews exact new tests, evidence proposal and golden amendment before respective writes; frozen test/fixture bytes thereafter unchanged. No edits to other stories' RED files except the two later exactly independently PM-authorized fixture-helper adaptations explicitly scoped here; no authority beyond those helper bodies.
Unit plus Integration tests: MANDATORY (no mocks/stubs of CLI, parser, process output or review). Tests may copy actual example inputs into isolated temporary trees and introduce reviewed malformed/stale variants. Full substantive review is separate from machine hash checks; no test fabricates reviewer truth.
Commands from Machinery root after upstream acceptance:
- go test -count=1 -timeout=180s ./cmd/machinery -run 'ExampleAttestationMigration' -json
- go test -count=1 -timeout=180s ./cmd/machinery -run 'TestGoldenCheck' -json
- go test -count=1 -timeout=5m -json ./internal/hook -run 'TestStopGreenDesignClearsStateSilently|TestPreEditObligationSurvivesLostPostAndReplacementSession|TestWaveDeferralSurvivesCrashAndOutOfBandClose|TestStopDriftBlocks|TestStopWarnsWhenStagedImplGatesLackImpl|TestReapedStopStillGatesTheTree'
- go test -count=1 -timeout=5m -json ./internal/gates -run 'TestObligationParent'
Use actual built CLI check for all eight paths; Go CRM current calls supply examples/go-crm/impl, seven design-only cases omit invented implementation roots. Exact --warnings-as-errors controls distinguish missing-current plan warnings without relying on unrelated incomplete --complete prerequisites. No all-green complete claim for unimplemented examples.
Record all selected leaves, raw logs, same-SHA source/binary/input hashes, claim classification before/after, independent review identities supplied only after review, and per-file golden before/after justifications. Missing/skip/setup failure is not RED/proof. Service-free Go/local CLI/local Git; this is not a new external runtime lane or Docker/Paivot product dependency. Full preflight remains final epic gate.

## Dependencies
After accepted MAC-uzxr, MAC-lhu5 and MAC-p7jd; blocks MAC-lnu6 and MAC-ou97. MAC-vx24 already follows MAC-lnu6 and retains future runner/replay migration ownership; no reverse dependency or core contract changes. Serialized shared-file custody is mandatory even though MAC-lnu6 keeps its healthy claim.

## OUT OF SCOPE
- Core schema/custody/API redesign, p7 frozen/seam changes outside the two expressly scoped later fixture-helper adaptations, future authenticated execution/replay.
- Go CRM application/test implementation and portfolio packet implementation owned by upstream repairs.
- MAC-lnu6 nine policy replacements, CLI/frozen tests, arbitrary golden updates or historical acceptance renewal.
- Global settings, library upgrades, new target implementations, generated-source changes, installations/remotes/full preflight.

## DIFF BUDGET
-41 explicit paths maximum forecast. Existing consumer1100-1900 changedLOC planning estimate now has a separate conditional helper-adaptation increment:65-160 changedLOC across exactly2existing helper bodies, using the proposed65-line p7 construction as a rough scale reference only. Combined rounded planning range approximately1170-2060 changedLOC, not measured implementation, arbitrary cap or permission to skip proof. Exact v2 construction remains undecided, so independently review and report actual additions/deletions before edits; escalate unsupported growth rather than inventing a generic helper. Six BUILD and eight evidence subjects, up to24 exact affected golden files, one new migration test, plus two narrow fixture helpers. This bounded planning allowance supersedes31/900-1600 because the inventory includes20 previously omitted YAML rows and two additional positive/negative example controls plus six conditional golden outputs. It is not measured implementation or exact edit authority. Change only necessary golden files; report actual count and full per-file cost.
- Replay/review cost additionally includes all13 existing helper-caller leaves. Initial external p7 observations6hookPASS in4.704s plus7parentPASS in9.320s are historical diagnostics, not an hgz1 timing guarantee or candidate proof. Retain existing5m command bounds, measure actual candidate duration and full inventories; do not trim13cases or manufacture scope claims to fit forecast.
- Evidence cost includes full v2 all-row kind conversion and real Go CRM complete-root manifest, not just14 hash edits. Baseline Go CRM impl has35 tracked files before accepted new helper additions, plus directories/ignored/untracked inputs included by the actual full-root policy. Never trim inventory to fit cost. Tests/substantive review/golden evidence and provenance may dominate; report actual LOC/time and investigate overrun without weaker claims.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## nd_contract
status: new

### evidence
- Created P0 consumer migration under MAC-ui8a by explicit dispatcher serialization direction; core p7 contract unchanged and not yet implemented proof.
- PM identified unsupported14 current claims; all current evidence writes remain held. Existing reports/old prose authorization remain historical; source/claim/provenance decisions require fresh exact independent approval.
- No source/test/docs/evidence/golden/claim/status mutation by this triage; only new tracker story/dependencies and scoped downstream contract reconciliation.

### proof
- [ ] AC #1: truthful complete conformance/context BUILD contracts with policy blocks preserved.
- [ ] AC #2: local-authority Go CRM migration/toolchain wording resolved.
- [ ] AC #3: complete closed v2 row classification and actual current versus planned distinction.
- [ ] AC #4: substantive independent review, real generated scope and exact provenance/hash changes.
- [ ] AC #5: eight real CLI positive/negative outcomes without overclaim.
- [ ] AC #6: exact reviewed golden updates and accepted downstream baseline handoff.


## Acceptance Criteria


## Design


## Notes
TERMINAL SERIALIZATION SELF-REVIEW
SR PM COMPLETE BUNDLED INVENTORY CORRECTION. Canonical39 possible paths now comprise sixBUILD/eightattestations/24conditionalstandardgoldens/one new test;97 total rows across8documents,12 behavioral rows. Added checkout-split/parent and pii-flow evidence plus their6standard golden outputs. Their BUILD files and check-pii-flow-gk remain read-only. Six AC retained with migration proof expanded to8examples; exact evidence/golden amendments still require prior independent substantive/PM review, true reviewer/date/note, complete covers/scope and old provenance. No hash-only review renewal. Serial accepted p7+uzxr+lhu5 -> hgz1 -> lnu6 unchanged; lnu6 still exactly17paths/6examples/14conditionalrows/9policyblocks. Guarded ndedit external trial and exact live full-byte comparison passed; bodyb5160c7cdec3320cce415261f01a5eef963eadb0db2887e3fd59c4848046e900 verified, all prior Notes/History/Links/Comments and metadata preserved except expected hash/time. Approx1100-1900 changedLOC is a forecast, not evidence/test-edit permission. No reverse p7 dependency. Scoped structural verification after exact canonical repair: pvg lint --backlog --epic MAC-ui8a scanned37 issues,0 errors/0 review findings; pvg rtm check37 stories/19closed/0 extracted requirements; pvg nd dep cycles found none. These are structure checks, not AC proof. Claims/status/labels/dependencies retained; no source/test/example/toolchain/remote/preflight mutation or Paivot product dependency.

## nd_contract
status: new

### evidence
- Terminal tracker readback verified P0 open/unclaimed consumer,31 explicit forecast paths and six pending AC; depends on MAC-uzxr,MAC-lhu5,MAC-p7jd and blocks MAC-lnu6 plus final MAC-ou97. MAC-lnu6 keeps its original five AC,17 paths,healthy claim and frozen RED/CLI. Existing story claims/status/labels preserved.
- Scoped backlog lint:37 issues,0 errors,0 review findings; dependency cycles:none. RTM:22 stories checked,4 closed,0 extracted requirements; structural result only, NOT acceptance-criterion proof.
- Actual canonical headings, commands and AC read back semantically; original MAC-lnu6 AC compared exact. Root remains clean main497419ab4512fcff765cd5feb27aed4c67b5608d. No runtime/source/test/docs/evidence/golden changes occurred. All14 evidence amendments remain held for substantive review; current observed historical execution is not future parser/migration proof.
- Created P0 consumer migration under MAC-ui8a by explicit dispatcher serialization direction; core p7 contract unchanged and not yet implemented proof.
- PM identified unsupported14 current claims; all current evidence writes remain held. Existing reports/old prose authorization remain historical; source/claim/provenance decisions require fresh exact independent approval.
- No source/test/docs/evidence/golden/claim/status mutation by this triage; only new tracker story/dependencies and scoped downstream contract reconciliation.

### proof
- [ ] AC #1: truthful complete conformance/context BUILD contracts with policy blocks preserved.
- [ ] AC #2: local-authority Go CRM migration/toolchain wording resolved.
- [ ] AC #3: complete closed v2 row classification and actual current versus planned distinction.
- [ ] AC #4: substantive independent review, real generated scope and exact provenance/hash changes.
- [ ] AC #5: six real CLI positive/negative outcomes without overclaim.
- [ ] AC #6: exact reviewed golden updates and accepted downstream baseline handoff.

## History
- 2026-09-06T03:17:08Z dep_added: blocked_by MAC-uzxr
- 2026-09-06T03:17:08Z dep_added: blocked_by MAC-lhu5
- 2026-09-06T03:17:09Z dep_added: blocked_by MAC-p7jd
- 2026-09-06T03:17:09Z dep_added: blocks MAC-lnu6
- 2026-09-06T03:17:09Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-lnu6]], [[MAC-ou97]]
- Blocked by: [[MAC-uzxr]], [[MAC-lhu5]], [[MAC-p7jd]]

## Comments

### 2026-09-06T03:24:58Z ramirosalas
## nd_contract
status: new

### evidence
- Terminal tracker readback verified P0 open/unclaimed consumer,31 explicit forecast paths and six pending AC; depends on MAC-uzxr,MAC-lhu5,MAC-p7jd and blocks MAC-lnu6 plus final MAC-ou97. MAC-lnu6 keeps its original five AC,17 paths,healthy claim and frozen RED/CLI. Existing story claims/status/labels preserved.
- Scoped backlog lint:37 issues,0 errors,0 review findings; dependency cycles:none. RTM:22 stories checked,4 closed,0 extracted requirements; structural result only, NOT acceptance-criterion proof.
- Actual canonical headings, commands and AC read back semantically; original MAC-lnu6 AC compared exact. Root remains clean main497419ab4512fcff765cd5feb27aed4c67b5608d. No runtime/source/test/docs/evidence/golden changes occurred. All14 evidence amendments remain held for substantive review; current observed historical execution is not future parser/migration proof.
- Created P0 consumer migration under MAC-ui8a by explicit dispatcher serialization direction; core p7 contract unchanged and not yet implemented proof.
- PM identified unsupported14 current claims; all current evidence writes remain held. Existing reports/old prose authorization remain historical; source/claim/provenance decisions require fresh exact independent approval.
- No source/test/docs/evidence/golden/claim/status mutation by this triage; only new tracker story/dependencies and scoped downstream contract reconciliation.

### proof
- [ ] AC #1: truthful complete conformance/context BUILD contracts with policy blocks preserved.
- [ ] AC #2: local-authority Go CRM migration/toolchain wording resolved.
- [ ] AC #3: complete closed v2 row classification and actual current versus planned distinction.
- [ ] AC #4: substantive independent review, real generated scope and exact provenance/hash changes.
- [ ] AC #5: six real CLI positive/negative outcomes without overclaim.
- [ ] AC #6: exact reviewed golden updates and accepted downstream baseline handoff.

### 2026-09-06T05:26:05Z ramirosalas
## nd_contract
status: new

### evidence
- SR PM COMPLETE BUNDLED INVENTORY CORRECTION. Canonical39 possible paths now comprise sixBUILD/eightattestations/24conditionalstandardgoldens/one new test;97 total rows across8documents,12 behavioral rows. Added checkout-split/parent and pii-flow evidence plus their6standard golden outputs. Their BUILD files and check-pii-flow-gk remain read-only. Six AC retained with migration proof expanded to8examples; exact evidence/golden amendments still require prior independent substantive/PM review, true reviewer/date/note, complete covers/scope and old provenance. No hash-only review renewal. Serial accepted p7+uzxr+lhu5 -> hgz1 -> lnu6 unchanged; lnu6 still exactly17paths/6examples/14conditionalrows/9policyblocks. Guarded ndedit external trial and exact live full-byte comparison passed; bodyb5160c7cdec3320cce415261f01a5eef963eadb0db2887e3fd59c4848046e900 verified, all prior Notes/History/Links/Comments and metadata preserved except expected hash/time. Approx1100-1900 changedLOC is a forecast, not evidence/test-edit permission. No reverse p7 dependency. Scoped structural verification after exact canonical repair: pvg lint --backlog --epic MAC-ui8a scanned37 issues,0 errors/0 review findings; pvg rtm check37 stories/19closed/0 extracted requirements; pvg nd dep cycles found none. These are structure checks, not AC proof. Claims/status/labels/dependencies retained; no source/test/example/toolchain/remote/preflight mutation or Paivot product dependency.

### proof
- [ ] AC #1: Six owned BUILD conformance/context corrections pending with lnu policy blocks preserved.
- [ ] AC #2: Go CRM local-authority migration/toolchain wording correction pending.
- [ ] AC #3: All8 YAML documents require complete closed v2 classification; no migration performed.
- [ ] AC #4: Substantive independent review, actual generated scope and exact provenance/hash edits pending.
- [ ] AC #5: Eight real example positive/negative CLI matrices pending; design-only plans never current proof.
- [ ] AC #6: Exact reviewed golden amendments and downstream six-example lnu handoff pending.

### 2026-09-06T05:28:44Z ramirosalas
## nd_contract
status: new

### evidence
- Final root-detected canonical checklist count corrected ONLY from six to eight for AC #5 through the same full-byte guarded ndedit route. External trial and exact readback passed; body9ae7d694f2a8b69c255ac6841ddee3401880aa88fc4779bc7baa24500aff7c37 before this append. All historical Notes/Comments/state/claims/dependencies unchanged; no criterion or scope expansion.
- SR PM COMPLETE BUNDLED INVENTORY CORRECTION. Canonical39 possible paths now comprise sixBUILD/eightattestations/24conditionalstandardgoldens/one new test;97 total rows across8documents,12 behavioral rows. Added checkout-split/parent and pii-flow evidence plus their6standard golden outputs. Their BUILD files and check-pii-flow-gk remain read-only. Six AC retained with migration proof expanded to8examples; exact evidence/golden amendments still require prior independent substantive/PM review, true reviewer/date/note, complete covers/scope and old provenance. No hash-only review renewal. Serial accepted p7+uzxr+lhu5 -> hgz1 -> lnu6 unchanged; lnu6 still exactly17paths/6examples/14conditionalrows/9policyblocks. Guarded ndedit external trial and exact live full-byte comparison passed; bodyb5160c7cdec3320cce415261f01a5eef963eadb0db2887e3fd59c4848046e900 verified, all prior Notes/History/Links/Comments and metadata preserved except expected hash/time. Approx1100-1900 changedLOC is a forecast, not evidence/test-edit permission. No reverse p7 dependency. Scoped structural verification after exact canonical repair: pvg lint --backlog --epic MAC-ui8a scanned37 issues,0 errors/0 review findings; pvg rtm check37 stories/19closed/0 extracted requirements; pvg nd dep cycles found none. These are structure checks, not AC proof. Claims/status/labels/dependencies retained; no source/test/example/toolchain/remote/preflight mutation or Paivot product dependency.

### proof
- [ ] AC #1: Six owned BUILD conformance/context corrections pending with lnu policy blocks preserved.
- [ ] AC #2: Go CRM local-authority migration/toolchain wording correction pending.
- [ ] AC #3: All8 YAML documents require complete closed v2 classification; no migration performed.
- [ ] AC #4: Substantive independent review, actual generated scope and exact provenance/hash edits pending.
- [ ] AC #5: Eight real example positive/negative CLI matrices pending; design-only plans never current proof.
- [ ] AC #6: Exact reviewed golden amendments and downstream six-example lnu handoff pending.

### 2026-09-06T05:46:14Z ramirosalas
## nd_contract
status: new

### evidence
- Sr PM canonical future migration ownership now41possiblepaths (prior39 plus only existing copyTree/obligationParentFixture bodies). hgz1 consumes accepted p7 exact construction/commit, independent disposition, full13-caller same-SHA blast and complete inventories. Both v1 guards WILL fail when these source YAML docs migrate to v2; later helper adaptation requires new exact BEFORE-EDIT PM review at actual accepted hgz1 baseline. Future v2 construction not selected; no automatic current-to-plan downgrade, invented review/hash/attribution or current implementation credit. Preserve every original assertion/Ga ancestry/default-versus-explicit Gt and complete provenance/acceptance/input inventory. All13 caller leaves remain mandatory, not just four failures. Separate rough65-160changedLOC helper increment and measured historical13-caller runtime added; combined rounded1170-2060forecast is conditional, never proof/cap or scope expansion. lnu6 remains exactly17paths/6examples/14conditionalrows/9policyblocks; serial dependencies unchanged.
- Proven guarded pvg nd edit used exclusive-lock editor and apply_patch only, fullraw expected header/body + external exacttext trial + exactpostreadback; all prior Notes/History/Links/Comments and metadata retained except normal hash/time. Original story AC compared byte-exact. Source/proposal read complete; no product/test/example mutation or Paivot runtime/build/test dependency.
- Scoped lint37issues/0errors/0review; cyclesnone; globalRTM37stories19closed0extracted requirements. Structural results only, not epic completion or AC proof. Claims/status/labels/dependencies unchanged. External preservation manifests /tmp/machinery-private-fixture-serialization.clhWCS.

### proof
- [ ] AC #1: Six BUILD conformance/context corrections pending.
- [ ] AC #2: Go CRM local-authority wording correction pending.
- [ ] AC #3: Eight-document closed-v2 migration pending.
- [ ] AC #4: Substantive independent evidence and exact fixture amendment review pending; no automatic current downgrade.
- [ ] AC #5: Eight migration matrices plus all13 unchanged helper-caller leaves and inventory/provenance proof pending.
- [ ] AC #6: Exact reviewed goldens/accepted handoff to unchanged six-example lnu scope pending.
