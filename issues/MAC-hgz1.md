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
updated_at: 2026-09-06T03:24:58Z
content_hash: "sha256:14e16f2e29ac68271527b297dd1f5cea9c19bd7c64a2180c68ce7e6eaceb4360"
blocked_by: [MAC-uzxr, MAC-lhu5, MAC-p7jd]
blocks: [MAC-lnu6, MAC-ou97]
---

## Description
## USER INTENT
Shipped examples must distinguish planned tests from actual current conformance review, and their BUILD handoffs and evidence must accurately describe the repaired subjects rather than silently carrying old unsupported claims forward.

## Context (Embedded)
P0 consumer repair discovered in MAC-lnu6's independent substantive review: all six legacy gt.conformance-test-shape records overstate current wholesale FSM oracle linkage. Go CRM's actual218 passing transition leaves and197 named stable IDs were not a parser-backed complete current claim; MAC-uzxr repairs that test support. Five other examples have no target implementation; fulfillment has a sound prospective obligation, but a plan is not a currently executing suite. MAC-lhu5 repairs six portfolio packet-alone deficiencies, including M3 objective/units and M4/M5 conformance obligations.
Existing approved MAC-p7jd R2 already defines closed v2 plan/current/historical classification, complete rooted implementation inventory and honest non-authenticated hash limits. This story consumes its ACCEPTED implementation; it does not redesign that core contract. MAC-l7m0/MAC-vx24 remain owners of future execution authentication/replay enforcement.
Diagnostic authority: /tmp/MAC-lnu6-PM-AMENDMENT-713184d.md SHA2566fc5223d3e5d48170f89ff9e29b634b256bfe2454b710378f08cb60d875aadf1. All14 affected existing BUILD records remain HELD today; no actual new reviewer/date/note/hash or migration write is authorized by this backlog creation.

## Ownership and serialization
Exactly31 forecast paths: six BUILD files, six evidence files, eighteen conditionally affected golden outputs and one new native regression test.
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
- cmd/machinery/example_attestation_migration_test.go (new).
Golden scope is conditional: change only the exact files/lines independently PM-authorized BEFORE edits for genuine approved migration diagnostics/counts; retain every historical baseline, command and old expected output. No wholesale golden recapture, acceptance of arbitrary failure, cmd/machinery/golden_test.go changes, or mutation of MAC-lnu6/MAC-p7jd frozen tests.
Exclusive serial custody: this story runs only AFTER MAC-uzxr, MAC-lhu5 and MAC-p7jd are accepted, then releases the six BUILD/evidence paths to dependent MAC-lnu6 for its final policy/evidence revalidation. No dependency on MAC-lnu6, MAC-vx24 or MAC-l7m0; this avoids a cycle and does not wait for future runner authentication. MAC-lnu6's healthy claim/worktree stays retained, but its shared-file writes are paused until this story is accepted.
MAC-lnu6 alone owns its nine exact policy prose replacements, frozen RED0ea1fdc730aadac15cecc8de33bb95898ad91d60 and CLI713184db16a12b8c3763b4aa8f5bf025721abf22. This consumer MUST preserve the exact existing old replacement blocks in all six BUILD files byte-for-byte and uniquely located; it must NOT apply, paraphrase, delete or move their ownership, nor change the three non-BUILD policy files. Use /tmp/MAC-lnu6-proposal.NZsERt/guidance-deltas.json SHA256d21d80e66e18356110be1a4c1753adebf6a3440846c8286b3664da945deac011 to identify those exact old/new blocks. If a concrete correction overlaps a preserved block, stop for exact canonical/PM review rather than silently widening permission.
Read-only: core gates/CLI/designlock/hook implementations, packet files owned by MAC-lhu5, Go CRM tests/implementation owned by MAC-uzxr, all generated oracles/machines, acceptance files, go.mod/go.sum and other examples/goldens. No settings/install/remote/preflight changes. You are not alone; preserve accepted upstream work.

## Boundary Map
PRODUCES:
- Six exact BUILD.md files -> complete truthful prospective conformance/context obligations, preserving MAC-lnu6's exact pending policy blocks.
- Six exact attestations.yaml files -> explicitly reviewed v2 classification and subject binding with old provenance preserved, no invented reviewer or execution.
- Eighteen exact conditional golden files -> only individually PM-authorized expected output changes from accepted schema/claim semantics, preserving historical baseline.
- cmd/machinery/example_attestation_migration_test.go -> real six-example migration and current/plan/history positive/negative proof.
CONSUMES:
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
3. Migrate each entire owned YAML document to accepted p7 v2, adding a valid kind to EVERY existing row under its exact closed classification. Preserve all claim IDs/row order/covers membership and historical acceptance facts. Design g2/g3/g4.zero-context are plan; ga.review-quality is historical; behavior gt/pack/standin are current only with substantive current review and complete implementation subject, otherwise explicitly reviewed prospective plan where the design-only status is true. For five unimplemented targets the affected behavioral rows are plan and visibly cannot discharge current implementation obligations. Go CRM gt requires actual independent review of the accepted parser-backed implementation/test scope before current generation; do not recast it merely to evade required verification. Any other unsupported purported current claim is held and reported, never silently grandfathered.
4. Independent PM substantively reviews complete current covers and intended claim statements BEFORE authorizing exact evidence edits or providing actual reviewer/date/specific findings. Use actual accepted CLI generation for authoritative covers/full-root-v1 inventory/hash; merge its single-row documents under closed schema, do not hand-fabricate inventories or exclusions. Preserve original reviewer/date/note/hash provenance and immutable old records/history; no blanket metadata renewal or automatic use of this triage/review author's identity. Only needed BUILD hashes and accepted changed portfolio packet hashes may refresh; every other nonchanged subject/hash remains exact. The six packet hash updates are this consumer's explicitly reviewed exception to MAC-lnu6's later non-BUILD immutability boundary, not continuing permission.
5. Actual isolated CLI tests on all six migrated examples establish ordinary valid plan/current/historical outcomes and exact warnings/counts. Design-only behavioral plans have missing-current warnings and zero current-review credit; warning promotion blocks them for that intended reason. Go CRM's reviewed unchanged current scope succeeds with --impl, then real test/handler/config mutation and add/remove/rename invalidate that scope for the accepted p7 categories, with matching safe control. Historical acceptance remains historical; no ordinary plan Gv success is claimed as complete or execution-authenticated assurance. Wrong kind, incomplete inventory and stale BUILD/packet negatives reach their intended categories, not earlier missing fixtures/unsupported future flags.
6. Only independently exact-before-edit authorized golden changes encode truthful migrated outcomes. Each diff must be linked to actual same-SHA stdout/stderr/exit evidence and reviewed expected category; no failure masking or empty/missing-row positive. Rerun unchanged GoldenCheck test code against all affected expected outputs and retain historical original outputs. Hand off accepted schema/subject/golden baseline, complete proposal/hash/claim inventory, actual review evidence and uniquely preserved nine policy blocks to MAC-lnu6. It must rerun all six baseline/stale/final stages and independently re-review prose/evidence after its own edits; this story does not complete any MAC-lnu6 AC or authorize downstream writes.

## Testing Requirements
Hard TDD scoped to consumer fixtures/behavior AFTER accepted upstream interfaces: new test file uses existing real CLI, filesystem and local Git, not missing symbols/flags as RED. Baseline on accepted p7 with the old example fixtures must fail because claims lack correct migration/subjects or BUILD obligations, with separately valid generated plan/current controls. Independent PM reviews exact new tests, evidence proposal and golden amendment before respective writes; frozen test/fixture bytes thereafter unchanged. No edits to other stories' RED files.
Unit plus Integration tests: MANDATORY (no mocks/stubs of CLI, parser, process output or review). Tests may copy actual example inputs into isolated temporary trees and introduce reviewed malformed/stale variants. Full substantive review is separate from machine hash checks; no test fabricates reviewer truth.
Commands from Machinery root after upstream acceptance:
- go test -count=1 -timeout=180s ./cmd/machinery -run 'ExampleAttestationMigration' -json
- go test -count=1 -timeout=180s ./cmd/machinery -run 'TestGoldenCheck' -json
Use actual built CLI check for all six paths; Go CRM current calls supply examples/go-crm/impl, five design-only cases omit invented implementation roots. Exact --warnings-as-errors controls distinguish missing-current plan warnings without relying on unrelated incomplete --complete prerequisites. No all-green complete claim for unimplemented examples.
Record all selected leaves, raw logs, same-SHA source/binary/input hashes, claim classification before/after, independent review identities supplied only after review, and per-file golden before/after justifications. Missing/skip/setup failure is not RED/proof. Service-free Go/local CLI/local Git; this is not a new external runtime lane or Docker/Paivot product dependency. Full preflight remains final epic gate.

## Dependencies
After accepted MAC-uzxr, MAC-lhu5 and MAC-p7jd; blocks MAC-lnu6 and MAC-ou97. MAC-vx24 already follows MAC-lnu6 and retains future runner/replay migration ownership; no reverse dependency or core contract changes. Serialized shared-file custody is mandatory even though MAC-lnu6 keeps its healthy claim.

## OUT OF SCOPE
- Core schema/custody/API redesign, p7 frozen/seam changes, future authenticated execution/replay.
- Go CRM application/test implementation and portfolio packet implementation owned by upstream repairs.
- MAC-lnu6 nine policy replacements, CLI/frozen tests, arbitrary golden updates or historical acceptance renewal.
- Global settings, library upgrades, new target implementations, generated-source changes, installations/remotes/full preflight.

## DIFF BUDGET
-31 explicit paths maximum forecast, approximately900-1600 changed LOC: twelve BUILD/evidence subjects, up to18 exact affected golden files, one new migration test. Change only necessary golden files; report actual count.
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
- [ ] AC #5: six real CLI positive/negative outcomes without overclaim.
- [ ] AC #6: exact reviewed golden updates and accepted downstream baseline handoff.


## Acceptance Criteria


## Design


## Notes
TERMINAL SERIALIZATION SELF-REVIEW

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
