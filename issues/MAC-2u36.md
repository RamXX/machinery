---
id: MAC-2u36
title: "Converge installer reruns on recorded targets"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T21:04:08Z
content_hash: "sha256:1bd3b597386f5bb503e9defd29457d11030b3d11c469f4439e6d71e56f310834"
blocks: [MAC-gcrr, MAC-ou97]
assignee: dev-MAC-2u36
follows: [MAC-olrx]
---

## Description
## USER INTENT
A user rerunning the installer or ordinary machinery update must get a complete, safely repaired installation at the requested release across all recorded placements, including a missing owned native artifact. Success must mean current content, complete receipt evidence and preserved topology; failure must preserve recoverable pre-run state.

## Context (Embedded)
Two production defects share this full-plan update boundary:
- NEXT1: install.sh passes --bootstrap-defaults, but updatePlan chooses default homes before the complete recorded home/native plan.
- Newly reproduced in MAC-2u36: ordinary update retains the missing Codex target in its plan, but refreshDirectInstalls runs home groups first; each child install calls recordHomeInstallLocked -> saveReceipt -> refreshReceiptArtifacts over ALL recorded placements. A missing later Codex file aborts the first child's receipt inventory before native repair is reached.
Actual isolated CLI diagnostic on frozen v1 99956740ae5559262790a9473b5597c1775928f2 failed in 35.820s after verified binary replacement and first home-group mutation: "inventory installed artifacts for receipt: digest .../.codex/agents/machinery-fsm-author.toml: lstat ...: no such file or directory". It did not assert final rollback state. This disproves the earlier assumption that the exact missing-file ordinary run would already pass; it does NOT change the required successful repair behavior.
The existing ordinary intact convergence and edited-regular-file repair controls pass. Metadata validity, path/type safety and plugin ownership are distinct from repairable installed content drift or absence.

## Scope decision and Ownership
This existing P0 bug owns both defects because both prevent the same recorded-plan convergence and use the same direct-refresh transaction. No separate issue or artificial dependency is created.
Production ownership expands explicitly from internal/install/update.go to:
- internal/install/update.go: receipt-aware bootstrap planning, full-plan refresh coordination and final commit/publication sequencing.
- internal/install/receipt.go: bounded receipt accumulation/finalization necessary for this transaction, preserving normal receipt validation/persistence.
- internal/install/install.go: child placement/receipt interaction needed for authenticated parent-owned full-plan refresh; preserve normal standalone install recording.
Tests: internal/install/bootstrap_receipt_test.go; directly associated regression additions in internal/install/receipt_test.go and internal/install/install_test.go only when required by the changed receipt/standalone-install boundary. Frozen existing assertions cannot be changed without explicit independent reviewer authorization.
Read-only consumers unless a concrete need is separately reviewed: internal/install/targets.go, internal/install/transaction.go, internal/install/lock.go, cmd/machinery/install.go and cmd/machinery/update.go. No blanket CLI/target/lock/journal rewrite is included. You are not alone in the codebase; preserve others' edits and coordinate shared paths with dispatcher.

## Boundary Map
PRODUCES:
- internal/install/update.go -> complete recorded-plan convergence, including safely missing owned native artifacts, under the existing update transaction.
- internal/install/receipt.go -> validated complete receipt publication coordinated with successful full direct refresh.
- internal/install/install.go -> bounded child-refresh participation while ordinary install continues to persist validated receipts.
- internal/install/bootstrap_receipt_test.go -> real CLI RED/GREEN matrix for both defects and preserved controls.
- internal/install/receipt_test.go -> focused receipt-publication regression proof if needed.
- internal/install/install_test.go -> focused standalone-install regression proof if needed.
CONSUMES:
- Existing internal/install/update.go.
  spec: updatePlan(opts UpdateOptions) (refreshPlan, error); refreshDirectInstalls(binary, source string, plan refreshPlan, run commandRunner, out io.Writer) error.
- Existing internal/install/receipt.go.
  spec: loadReceipt() (installReceipt, bool, error); saveReceipt(receipt installReceipt) (retErr error); refreshReceiptArtifacts(receipt *installReceipt) error; recordHomeInstallLocked(homes []string, copyAll bool) error; recordTargetInstallLocked(names []string, copyAll bool) error.
- Existing internal/install/install.go.
  spec: Install(opts Options) error; installLocked(opts Options) (retErr error).
  fields: Options.Homes []string; Targets []string; From string; Copy bool; Record bool. Current CLI passes Record: true.
- Existing internal/install/transaction.go.
  spec: beginArtifactTransaction(paths []string) (*artifactTransaction, error); delegatedInstallOperation() bool; delegatedArtifactTransaction(paths []string) (*artifactTransaction, error).
  source: authenticated delegated children require the parent's prepared journal and coverage of their exact artifact paths; parent transaction snapshots binary, direct placements and receipt, and owns final commit/rollback.
- Existing internal/install/targets.go.
  spec: installTargets(names []string, src string, copyAll bool, out io.Writer, before func(string) error) error.
  source: validates target source before placement; receipt recording is in installLocked, not this function.

### Story Acceptance Criteria
1. Valid supported receipt plus --bootstrap-defaults uses the same complete recorded home/native/plugin plan and relevant discovery as ordinary machinery update without selectors. Preserve per-group copy/symlink modes. No-receipt first bootstrap retains plugin-aware defaults; explicit homes/targets remain incompatible with bootstrap.
2. Fail closed with actionable diagnostics and no silent defaults for malformed/duplicate/unknown/trailing JSON, unsupported schemas, invalid topology/inventory/digest encoding, unsafe receipt permissions/type, unsafe artifact or parent path/type substitutions, and uncertain plugin ownership, including --skip-plugins. Supported schema 1 is not rejected merely because it is older. Valid receipt plus edited/missing owned regular artifact with unchanged safe parents/topology is repairable; absence alone must not be classified as unsafe ownership. Existing safety validation must not be relaxed.
3. Actual ordinary and bootstrap full-plan update succeeds for intact installs, an edited regular owned Codex artifact, and a missing regular owned native artifact, restoring exact current-release content and binary, complete receipt topology/digests/plugin obligations, both mixed-mode home groups and Codex/OpenCode targets, unrelated-file preservation and same-release idempotence. Exact missing-file fixture: remove only temporary HOME/.codex/agents/machinery-fsm-author.toml after valid receipt recording; no unsafe parent/type substitution. Test both native copy-mode groups missing an owned file in the same run as a bounded guard against a fix that merely repairs one hardcoded target first. During RED the ordinary missing-file case is EXPECTED TO FAIL behaviorally on unchanged production; intact ordinary and edited-file ordinary cases remain passing controls.
4. Retain actual later-target failure after binary, home and earlier native mutations with complete pre-run restoration of binary, homes/native artifacts, receipt and unrelated sentinels. Also exercise that rollback with a safely missing owned artifact in pre-state: absence must be restored as absence, not replaced by a half-repaired result. Retain real interrupted journal recovery at CLI startup and explicitly separate its pre-mutation interruption boundary from post-mutation rollback proof. Preserve independent malformed/schema/path/type/plugin-ownership negatives and mixed-copy-mode checks.
5. Integration uses actual built release/CLI install/update subprocesses, real checksummed archives and temporary home/config/target/binary roots. No mocks, stubs, skip-if-missing or env-gated omissions. No live installation mutations. Machinery product behavior/tests remain standalone with no Paivot/pvg/nd, label or commit-convention dependency.
6. A full direct-refresh operation must not require complete final artifact inventory before remaining selected placements can repair safely missing owned files. Keep one parent-owned bounded transaction covering the binary, full selected direct plan and receipt. Publish a complete normalized receipt with real refreshed digests only after all selected direct placements and source checks succeed, before final direct transaction commit; any final inventory/publication failure must fail and roll back the whole direct update. Do not drop missing entries, retain fabricated/stale digest evidence, invent success for an incomplete plan, or bypass final validation. Standalone install continues to record validated successful topology. Any internal receipt deferral/coordination must be restricted to authenticated parent-owned journal scope and must not become a user/env-only skip-receipt backdoor; forged/absent parent authority, unprepared journal or out-of-scope paths remain rejected. Host-managed plugin refresh retains its existing post-direct-commit ownership/obligation semantics; no new claim of atomic host-plugin rollback.

## Testing Requirements
- Hard TDD remains explicitly authorized. After independent PM reviews this scope/contract and authorizes the revised tests, a RED-only author revises the disputed edited/missing cases, adds the bounded multi-missing and missing-pre-state rollback proof, and any focused new boundary tests needed for AC6. No production edits before independently approved RED.
- Preserve original candidate 99956740ae5559262790a9473b5597c1775928f2, test SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329 and /tmp/machinery-MAC-2u36-red.qRk5DC. Existing 36-leaf replay: 32 pass, 4 fail, 0 skip, 271.751s; only three original failures are valid bootstrap defect proof. The fourth stale_artifact rejection is superseded. Missing-file ordinary FAIL 35.820s is new genuine defect evidence, separate from earlier edited-file ordinary PASS 36.508s.
- Preserve complete-recorded-plan, defaults/selectors, intact normal/bootstrap convergence, plugin obligations, real later-target rollback, interrupted recovery, and all legitimate safety negatives. Rename obsolete stale_artifact/disappeared_target rejection cases as repair-success requirements. Only specifically reviewer-authorized frozen-test changes are allowed.
- Fresh replay: go test -count=1 -timeout=10m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json. Timeout increased from 5m because the existing real replay already consumed 271.751s and the newly required paired repair/rollback cases add CLI executions; do not relax assertions or skip cases to fit the old timeout.
- Focused receipt/standalone-install/delegated-authority tests selected by actual test names introduced or affected. GREEN runs go test -count=1 -timeout=15m ./internal/install ./cmd/machinery for ordinary native package regression coverage. No full scripts/preflight.sh here; final epic gate owns heavy preflight.
- Record SHA, exact command, named leaf inventory, zero skips, assertion failure causes and passing controls. Compile/import/fixture/infra/timeout errors are not behavioral RED. Unexpected missing-file receipt-inventory failure is a valid success-contract assertion failure, not an acceptable permanent rejection.
- If internal deferred receipt behavior is introduced, add proof that normal standalone install still publishes correct receipts, missing/forged/out-of-scope parent authority cannot enable it, and final publication failure rolls back real prior mutations. Reuse existing authenticated delegation mechanisms; implementation details require evidence, not an assumed new public API.
- Separate independent PM replays revised RED before approval/freezing. GREEN preserves exact approved test/fixture/config bytes; any later repair requires explicit reviewer authorization and re-RED.

## OUT OF SCOPE
- Arbitrary missing-root recreation, changed unsafe parents, and ownership uncertainty: this story preserves their safety boundary; it does not authorize blind repair.
- Redesign of host-managed plugin transactions, lock/journal format or public install selectors: outside the direct receipt-publication defect; report a concrete need for review before expanding.
- Other assessment subsystems stay in sibling stories. No pushes, sync, remote mutation, installed binary/plugin/skill/agent replacement, dev-link or healthy worktree cleanup. NIL uses the installed product.
- Final heavy preflight, main merge and isolated release candidate belong to epic gate MAC-ou97.

## DIFF BUDGET
Expected 4-6 files, under 1100 total changed LOC: approximately 750 test LOC including the existing 469-line real fixture, and 350 production LOC across the three named files. This replaces the disproven 450-LOC estimate after newly reproduced receipt-publication scope. Do not pad to the budget; material overrun or another production file requires PM investigation and explicit scope review, not weaker tests.

## Dependencies
Parent MAC-ui8a; remains blocking MAC-gcrr and MAC-ou97. No new inter-story dependency: one developer owns both coupled installer defects and one independent PM reviews the full outcome. No other sibling implementation declares the three owned installer source seams. Consumer guidance and epic acceptance remain blocked until this story is accepted.

## MANDATORY SKILLS
- developer for RED/GREEN roles; codebase-memory for source discovery; pm_acceptor for independent scope/test authorization, RED replay and acceptance.

## Delivery Requirements
This repaired scope requires independent PM review before author re-RED; the previous 20:55:32Z test-edit authorization did not authorize implementing this newly reproduced defect. Preserve that review as history and obtain explicit expanded test authorization. Do not deliver or approve v1 unchanged.
Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands/results, inventory, changed-file rationale, per-AC proof and residual limits. Product has no dependency on these development coordination tools.

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.
RED-DISPUTE ADDENDUM — disappeared_target classification:
The frozen disappeared_target fixture removes only the regular owned file <temporary HOME>/.codex/agents/machinery-fsm-author.toml after a valid receipt was recorded. It does not substitute a symlink, change a parent path/type, or introduce unrelated ownership. The current bootstrap run rejects and preserves pre-run state (PASS, 18.65s), but no ordinary-update control for this exact missing-file input was run. Its acceptance assertion therefore needs the same AC1/AC2 contract review: supported ordinary forced repair may recreate a missing owned artifact. Do not infer unsafe ownership from absence alone. The passing negative demonstrates current bootstrap behavior, not proof that this is the intended permanent contract. Root requested no additional expensive CLI runs before Sr PM clarification.
## DISCOVERED_BUG — exact ordinary missing-file repair control

title: Ordinary multi-placement update aborts receipt inventory before repairing a missing native artifact
discovered_during: MAC-2u36
affected_files: internal/install/update.go (refreshDirectInstalls home-before-native order), internal/install/install.go (recordHomeInstallLocked after child placement), internal/install/receipt.go (saveReceipt -> refreshReceiptArtifacts)

Context: The exact safe missing-file control required by the authoritative clarification and TEST-EDIT AUTHORIZED comment was executed BEFORE editing any frozen test. Real install seeded two mixed copy/symlink home groups and Codex/OpenCode targets with a valid receipt. Only temporary HOME/.codex/agents/machinery-fsm-author.toml was removed; safe parents, other role, receipt metadata, and ownership remained unchanged. Actual ordinary update with no selectors unexpectedly exits 1 after downloading/checksum-verifying the actual built release, replacing the binary and installing the first copy home group. Its real child then aborts receipt recording:
"inventory installed artifacts for receipt: digest .../.codex/agents/machinery-fsm-author.toml: lstat .../.codex/agents/machinery-fsm-author.toml: no such file or directory"

Source chain: refreshDirectInstalls processes home groups before native groups. Each child install saves the complete receipt; saveReceipt calls refreshReceiptArtifacts over every recorded placement, including the native artifact that has not been repaired yet. Thus the current ordinary path does not satisfy the clarified missing-file repair expectation, despite buildRefreshPlan retaining its target. Scope/ordering/receipt persistence must be reviewed before expanding production work. No stricter rejection expectation was inferred.

Commands run:
- go test -count=1 -timeout=3m ./internal/install -run '^TestBootstrapOrdinaryMissingArtifactDiagnostic$' -v
Summary: Exact ordinary missing-file control FAIL, 35.820s; 1 leaf failure, zero skips. This is an unexpected production behavior, not a passing RED control. The diagnostic stops on the CLI error and does not independently verify final rollback state.
Evidence: /tmp/machinery-MAC-2u36-red.qRk5DC/bootstrap_missing_diagnostic_test.go and ordinary-missing-diagnostic.txt. Diagnostic source moved out of the retained worktree; frozen candidate 99956740ae5559262790a9473b5597c1775928f2 and SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329 remain unchanged. Previous edited-artifact ordinary repair succeeds; these are different observed outcomes.

Paused per the explicit reviewer condition to report this exact unexpected failure before expanding ownership or changing expected repair semantics. No frozen revision, repair commit, delivery, release, acceptance, or closure performed. Root informed for scoped Sr PM/independent review.

## nd_contract
status: in_progress

### evidence
- Read the complete authoritative replacement AC/testing block and exact TEST-EDIT AUTHORIZED comment.
- Executed required first ordinary missing-file control: actual CLI fails during first home group's full receipt inventory, before native repair.
- V1 candidate and prior logs preserved; diagnostic retained outside clean worktree; no production or frozen test changes.

### proof
- [x] Revision prerequisite: exact missing-file control executed and unexpected failure reported before scope expansion.
- [ ] Revised AC #1-5: blocked on scoped review of newly observed ordinary repair failure; prior valid bootstrap and rollback evidence remains preserved.


## nd_contract
status: in_progress

### evidence
- Independent PM bounded RED-DISPUTE review read the complete current story, pm_acceptor skill, frozen bootstrap_receipt_test.go, updatePlan, receipt loading/validation and discovery, target digest health validation, CLI forced-refresh help, and preserved edited-file diagnostic source/result.
- Verified retained worktree HEAD 99956740ae5559262790a9473b5597c1775928f2, one new 469-line test file, SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329, clean worktree; no tests rerun and no source edits for this semantic authorization.
- Appended TEST-EDIT AUTHORIZED comment limited to stale_artifact/disappeared_target repair-success cases and minimum required shared controls. Reviewed clarification is consistent with original ordinary-update parity and source repair semantics.

### proof
- [x] Bounded contract review: edited/missing owned regular-file repair distinguished from invalid receipt metadata and unsafe path/type/ownership states.
- [x] Explicit reviewer authorization recorded before frozen candidate revisions; tdd-red and [test-edit-authorized] required on repair commits.
- [ ] Exact missing-file ordinary CLI control remains required in revised candidate; source reasoning is not execution proof.
- [ ] AC #1-5: revised RED replay/approval and later implementation/acceptance remain pending. Candidate v1 is not RED-approved. Story remains in_progress/hard-tdd and undelivered.

## AUTHORITATIVE CONTRACT CLARIFICATION — MAC-2u36 — 2026-09-05
Scope: bounded Sr. PM repair of this existing story. The complete AC and testing contract below supersede the earlier ambiguous AC2/AC4 wording and the disputed v1 rejection expectations only. Earlier notes, evidence, candidate bytes and history remain preserved. This is NOT RED approval, delivery, acceptance, a production change, or authorization for the RED author to edit the frozen candidate. Independent PM review and explicit RED-revision authorization remain the next gate.

### Before / after rationale
- Prior AC2: "Malformed/stale/unsafe receipt fails closed with actionable diagnostic, never silently selects defaults. Explicit homes/targets remain incompatible with bootstrap as before."
- Repaired AC2 defines invalid receipt metadata and unsafe ownership/path state precisely; "stale" does not mean every mismatch between recorded digests and current owned artifact bytes. Supported schema 1 is not an unsupported-schema failure merely because it is older; current loadReceipt accepts schema 1 and receiptSchema.
- Prior AC4: "Negative tests cover corrupt receipt, plugin ownership discovery failure even under --skip-plugins, disappeared target, mixed copy modes and interrupted update rollback without altering unrelated host files."
- Repaired AC4 separates ordinary repair inputs (edited or missing owned regular artifacts) from safety rejection inputs and transaction failures. Missing-file absence alone is not proof of uncertain ownership. Mixed copy modes remain topology preservation coverage.

### Story Acceptance Criteria — complete authoritative replacement
1. When a valid supported receipt exists, --bootstrap-defaults uses the same complete recorded home/native/plugin plan and relevant discovery as ordinary machinery update with no selectors. Installer rerun must converge identically, including per-group copy/symlink modes and supported repair of owned artifacts. No-receipt first bootstrap retains plugin-aware default homes.
2. A receipt that cannot be safely loaded/validated fails closed with an actionable diagnostic, never silently selecting defaults: malformed/duplicate/unknown/trailing JSON, unsupported schema, invalid or inconsistent recorded topology/inventory/digest encoding, unsafe receipt file permissions/type, or unsafe path/type substitutions. Plugin discovery that cannot establish ownership remains fatal, including under --skip-plugins. Preserve existing safety validation and transaction protections; do not relax them to obtain parity. A structurally valid, safely owned receipt is not invalid solely because an owned regular artifact's bytes differ from its recorded digest, or because the specific owned regular artifact is missing at its recorded path with unchanged safe parents/topology. Those are forced-refresh repair inputs governed by AC1/AC3. Explicit homes/targets remain incompatible with bootstrap.
3. Real isolated install followed by installer-equivalent rerun updates the binary and every recorded home/native target, verifies release bytes and refreshed receipt digests/topology, and is idempotent. Cover two home groups with mixed copy/symlink modes plus Codex/OpenCode native targets. For both an edited regular Codex artifact and the exact missing-file fixture defined below, ordinary update and bootstrap rerun must restore the current release artifact, retain the complete receipt plan, refresh recorded placements, and preserve unrelated files. The ordinary path is a passing control, not permission to redefine unexpected failures as acceptable behavior.
4. Negative and transaction coverage retains corrupt/unsafe/unsupported-schema receipt rejection, metadata/path integrity protections, and plugin ownership discovery failure even under --skip-plugins. Retain real later-target failure after actual binary, home and earlier native-target mutations and prove complete pre-run restoration of binary, all recorded homes/native targets, receipt and unrelated sentinels. Retain real interrupted journal recovery on CLI startup; identify its pre-mutation interruption boundary separately from post-mutation rollback proof. The v1 disappeared_target fixture is a repair-success/parity case, not a mandatory rejection case. Preserve unsafe symlink/type/parent-path and ownership rejection coverage independently; do not use missing-file absence as a substitute for those hazards.
5. Integration uses temporary home/config/target/binary roots and the actual built CLI update/install flow, not only updatePlan. No mocks, stubs, skip-if-missing, env-gated omissions or live user installation mutations. Machinery product behavior and tests must remain standalone and must not require Paivot/pvg/nd, workflow labels or commit conventions; Paivot is development coordination only.

### Exact disputed-input classification
- v1 stale_artifact writes "externally modified target" into the existing regular temporary HOME/.codex/agents/machinery-fsm-author.toml. The receipt and safe topology remain valid. Expected behavior: successful forced repair and full recorded-plan convergence for ordinary and bootstrap update; refreshed digest must match restored release content, and edited content must be gone.
- v1 disappeared_target removes only that same regular file after valid receipt recording. It does not replace it or a parent with a symlink/non-directory, mutate receipt metadata, or introduce unrelated ownership. Expected behavior: recreate the recorded artifact through ordinary and bootstrap forced repair. Source evidence supports this classification: buildRefreshPlan copies receipt.Targets before discovery; fileExists returns false, nil on os.IsNotExist; targetInstalled retains discovery of other roles and does not remove receipt targets.
- The v1 bootstrap disappeared_target rejection PASS is historical behavior, not approved desired behavior. No exact ordinary-update missing-file control was executed before this clarification. Revised RED must execute that control and record the result. If ordinary update unexpectedly fails for this exact safe missing-file input, report the concrete production failure to the dispatcher/Sr. PM before expanding ownership or changing expectations; do not infer a stricter rejection contract.
- No unresolved user product choice was identified: existing AC1 parity, documented forced refresh and the edited-file control resolve the ambiguity. This does not authorize recreating arbitrary absent roots or following changed/unsafe paths.

### Revised RED candidate instructions — conditional on subsequent independent PM authorization
1. Preserve v1 commit 99956740ae5559262790a9473b5597c1775928f2, test SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329, and all logs under /tmp/machinery-MAC-2u36-red.qRk5DC. Preserve the 36-leaf result: 32 PASS, 4 FAIL, 0 SKIP, 271.751s. Only three v1 failures are valid bootstrap defect evidence; stale_artifact is disputed and cannot qualify RED unchanged.
2. After independent PM authorizes revision, amend only the two disputed expected-outcome cases and minimal shared test structure needed to run ordinary/bootstrap repair controls. Rename them clearly as edited-owned-artifact and missing-owned-artifact repair cases. Verify restored current release bytes, complete recorded topology/copy modes, refreshed digests and unrelated-state preservation; cover idempotence consistently with AC3. Any further frozen assertion/fixture/config change requires specific reviewer authorization.
3. Preserve complete_recorded_mixed_plan, ordinary/bootstrap convergence, no-receipt defaults and explicit-selector controls, plugin obligations, later_target_failure_rolls_back ordinary/bootstrap cases, interrupted recovery and all existing legitimate malformed/schema/path/ownership negatives. Do not alter production, bypass the real archive/CLI path, remove requirements, or claim a passing safety case is a RED defect.
4. Run fresh real CLI controls and behavioral RED against unchanged production; use go test -count=1 -timeout=5m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json, increasing only the timeout if the enlarged real matrix demonstrably needs more time and recording why. Capture exact command, leaf inventory, failures by assertion, passing controls, zero skips, SHA and file hashes. Compile/import/fixture/timeout failures are not behavioral RED.
5. Return revised candidate for independent PM replay/approval; only after approval freeze exact test/fixture/config bytes for a separate GREEN implementer. No automatic RED approval is granted by this clarification. Keep source ownership internal/install/update.go and test ownership internal/install/bootstrap_receipt_test.go; the 469-line v1 file exceeded the rough 450-line budget by 19 lines for real archive, rollback and negative fixtures. Record and justify any revision delta for PM investigation.
6. No full preflight, remote/git-host mutation, installed binary/plugin/skill/agent replacement, dev-link, worktree cleanup, acceptance or closing is authorized here.

### Evidence reviewed
- Full pvg nd show MAC-2u36 including both RED disputes and v1 proof; the complete frozen test file read without modification.
- Graph-first discovery and exact source: internal/install/update.go updatePlan (248); internal/install/receipt.go loadReceipt (99), buildRefreshPlan (559), targetInstalled (666), fileExists (697), validateReceipt (779); internal/install/targets.go validateReceiptArtifactDigest (157); cmd/machinery/update.go newUpdateCmd (12). Coverage generation 2026-09-05T20:28:41Z reports metadata_match/no_recorded_issue for these four source paths (best-effort, supplemented by exact source reads).
- loadReceipt supports schema 1/current schema; validateReceipt validates digest shape and inventory topology. Artifact health digest validation is distinct from update plan construction. CLI describes forced refresh of recorded placements even at the same version.
- Preserved actual ordinary edited-artifact diagnostic: exit 0, both home groups and both native targets refreshed, binary matches next release and edited content removed; PASS 36.508s, ordinary-update-diagnostic.txt.
- Root and retained story worktree git status --short were clean during this read-only repair inspection. No new production defect established and no new issue created.

## nd_contract
status: in_progress

### evidence
- Sr. PM appended this authoritative complete AC/testing clarification via shared pvg nd; existing story state, ownership, labels, history and v1 candidate preserved.
- Bounded semantic review only. Independent PM review of clarification and RED-revision authorization pending; no RED/GREEN approval or delivery.

### proof
- [x] Contract repair: AC2 distinguishes invalid receipt/unsafe ownership from supported regular-artifact repair; AC4 preserves independent safety and transaction negatives.
- [x] Contract repair: exact disappeared_target mutation classified from source, with missing ordinary CLI control explicitly required.
- [ ] AC #1-5: implementation and independent acceptance remain pending; v1 evidence is preserved, not promoted to acceptance.


## nd_contract
status: in_progress

### evidence
- Frozen candidate 99956740ae5559262790a9473b5597c1775928f2 preserved unchanged.
- Paused before delivery for review of stale_artifact and disappeared_target repair semantics; all existing proof retained.
- Ephemeral diagnostic source moved outside worktree to /tmp/machinery-MAC-2u36-red.qRk5DC; no production changes.

### proof
- [x] AC #1: receipt-plan divergence reproduced; repair-input parity requires clarification.
- [ ] AC #2: stale receipt versus content drift and missing owned artifacts requires Sr PM/reviewer clarification.
- [x] AC #3: real normal convergence and idempotence pass; bootstrap recorded-target staleness fails.
- [ ] AC #4: post-mutation rollback/interruption controls pass; missing-artifact rejection expectation remains under review.
- [x] AC #5: actual built CLI and isolated roots verified; live installation SHA unchanged.


## Implementation Evidence (RED CANDIDATE — PAUSED BEFORE DELIVERY)

PROOF:

### CI/Test Results
Commands run:
- go test -count=1 -timeout=5m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json
- go test -count=1 -timeout=3m ./internal/install -run '^TestBootstrapOrdinaryEditedArtifactDiagnostic$' -v (ephemeral diagnostic companion; retained outside candidate worktree)
- pvg verify internal/install/bootstrap_receipt_test.go --format text --include-tests
- git diff --check
- shasum -a 256 /Users/ramirosalas/.local/bin/machinery

Summary: Frozen broad replay completed in 271.751s with 36 leaf tests: 32 PASS, 4 FAIL, 0 SKIP. New candidate inventory: 13 leaves, 9 PASS and 4 FAIL; 23 existing regression leaves PASS. Three failures reproduce bootstrap receipt-plan loss and skipped native rollback. The fourth stale_artifact failure is a disputed test expectation, not approved RED evidence. Diagnostic ordinary update PASS in 36.508s establishes successful repair of the same edited artifact. pvg verify PASS (1 file, 0 issues); git diff --check PASS.
Coverage: AC mapping below covers 5/5 requested areas; statement-coverage percentage not collected for this tests-only RED candidate.

### Commit
- Branch: story/MAC-2u36
- RED SHA: 99956740ae5559262790a9473b5597c1775928f2
- Frozen file SHA256: 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329
- Change: one new test file, 469 LOC. The 19-line budget overrun supports real release packaging, actual mutation rollback, and required negative fixtures. No production or existing test changes.
- Raw proof: /tmp/machinery-MAC-2u36-red.qRk5DC/frozen-red-test-output.jsonl
- Diagnostic source and result: /tmp/machinery-MAC-2u36-red.qRk5DC/bootstrap_semantics_diagnostic_test.go and ordinary-update-diagnostic.txt

### Real boundary proof
Actual Go-built Cobra CLI binaries report versions v9.9.1/v9.9.2. Existing endpoint string variables are linked to a loopback HTTP artifact server; the repository's real release-archive command supplies checksummed source bytes. The historical source fixture adds a visible old-asset marker. No replacement executable, command runner mock, skip gate, Docker/Java/Node dependency, or live installer invocation is used. Every actual install/update uses temporary home/config/target/binary roots and explicit --install-dir.

Ordinary update convergence and idempotence PASS (48.41s). A checksummed archive missing only the later OpenCode plugin triggers failure after actual binary replacement, home refresh, and Codex refresh; exact binary/home/native/receipt pre-state restoration PASS (32.20s). A killed update during a held source download leaves a real journal; actual CLI startup recovers it and preserves pre-state (16.54s). This interruption proves pre-mutation recovery; the separate later-target failure proves restoration after real mutations.

Live installed Machinery SHA remains 5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849. User plugins/skills/agents were not changed.

### AC Verification
| AC | Candidate proof | Status |
|---|---|---|
| 1 | TestBootstrapReceiptPlan complete_recorded_mixed_plan pins real recorded mixed homes/native targets and both plugin obligations against normal update; first bootstrap defaults and explicit-selector checks retained | Receipt parity RED; defaults/selectors PASS |
| 2 | CLI corrupt_receipt, unsafe_receipt, stale_schema all fail closed without changing state | PASS; stale_artifact interpretation disputed below |
| 3 | Real CLI converges_bootstrap_false/true validates binary bytes, source freshness, receipt topology/digests, and second-run idempotence across two home groups and two native targets | Normal PASS; bootstrap RED |
| 4 | Mixed copy/symlink topology, disappeared target, plugin discovery failure under --skip-plugins, later-target rollback, interrupted startup recovery, unrelated sentinel checks | Controls PASS; bootstrap later-target behavior RED |
| 5 | Temporary roots and real compiled update/install subprocess chain; no live user installation operations | PASS fixture containment |

### RED-DISPUTE
TestBootstrapReceiptCLI/stale_artifact currently expects rejection when an existing regular Codex artifact's bytes differ from its receipt digest. Independent actual ordinary update on precisely this input exits successfully, refreshes both home groups and native targets, replaces the binary, and removes the edited content. Its receipt is syntactically/topologically valid. buildRefreshPlan -> targetInstalled -> fileExists checks regular-file type, not receipt content digests; validateReceiptArtifactDigest is used by artifact health validation and recommends update as remediation.

The candidate rejection assertion conflicts with AC1 ordinary-update parity and supported forced-repair semantics. AC2's "stale receipt" needs a precise distinction between invalid/unsafe receipt ownership/topology versus repairable edited installed content. Root directed pause before delivery and Sr PM/independent review. Frozen candidate bytes remain unchanged; do not approve RED or implement GREEN against this disputed assertion until reviewed authorization and re-RED.

LEARNINGS:
- The real release archive builder is required: raw git archive emits a global PAX header rejected by source extraction. On macOS its output directory must use canonical /private/var ancestry.
- A late missing source asset gives deterministic actual post-mutation rollback coverage without a simulated command runner.
- Content drift and invalid receipt ownership are different updater contracts; ordinary forced repair must remain viable.

## nd_contract
status: in_progress

### evidence
- Frozen RED candidate 99956740ae5559262790a9473b5597c1775928f2; 36 leaves, 32 pass, 4 fail, zero skip, 271.751s.
- Three valid behavioral RED failures; stale_artifact expectation disputed after actual normal-update repair succeeded.
- Paused before delivery at root request; no story deliver/accept/close transition performed for MAC-2u36.

### proof
- [x] AC #1: receipt/default/selector behavior represented and actual lost-plan failure reproduced.
- [ ] AC #2: pending reviewed clarification of stale receipt versus supported edited-artifact repair.
- [x] AC #3: real normal control passes; bootstrap staleness reproduced with digests and idempotence.
- [x] AC #4: real mutation rollback and interrupted recovery controls pass; bootstrap skipped-target failure reproduced.
- [x] AC #5: real built CLI, temporary scopes, unchanged live installation digest.


## Historical canonical description before receipt-ordering triage R2
The following original description is preserved verbatim as quoted history; the current Description is authoritative.

> ## USER INTENT
> Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.
> 
> ## Context (Embedded)
> NEXT1: installer always passes --bootstrap-defaults; updatePlan branches to default homes before receipt planning, so recorded native targets stay stale. Receipt-based normal update already exists. NEXT2 plugin discovery is ownership-critical; do not degrade uncertain ownership to blind fallback.
> 
> ## Ownership
> Own only these paths and directly associated tests: internal/install/update.go, internal/install/bootstrap_receipt_test.go. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.
> 
> ## Boundary Map
> PRODUCES:
> - internal/install/update.go -> hardened behavior and regression proof
> - internal/install/bootstrap_receipt_test.go -> hardened behavior and regression proof
> CONSUMES:
> - Existing Machinery source interfaces.
>   spec: updatePlan(opts UpdateOptions) (refreshPlan, error)
> 
> ### Story Acceptance Criteria
> 1. When a valid receipt exists, --bootstrap-defaults uses its complete recorded home/native/plugin target plan so installer rerun converges identically to machinery update; no-receipt first bootstrap retains defaults.
> 2. Malformed/stale/unsafe receipt fails closed with actionable diagnostic, never silently selects defaults. Explicit homes/targets remain incompatible with bootstrap as before.
> 3. Positive real isolated install followed by installer-equivalent rerun updates binary and all recorded targets, verifies receipts/digests, and is idempotent; test multiple homes plus native targets.
> 4. Negative tests cover corrupt receipt, plugin ownership discovery failure even under --skip-plugins, disappeared target, mixed copy modes and interrupted update rollback without altering unrelated host files.
> 5. Integration uses temporary home/target directories and actual built CLI update/install flow, not only updatePlan. No live user installation mutations in story tests.
> 
> ## Testing Requirements
> - Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
> - Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
> - Commands: go test ./internal/install -run 'Bootstrap|Receipt|UpdatePlan'; scoped cmd install/update integration using temporary configured roots.
> - No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.
> 
> ## OUT OF SCOPE
> - Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
> - Global heavy preflight, main merge and local release binary belong to final epic gate.
> 
> ## DIFF BUDGET
> - ~2-4 files, under 450 changed LOC; material overrun requires PM investigation, not weakened requirements.
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
> - [ ] AC #5: independently verified


## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97
- 2026-09-05T20:30:19Z status: open -> in_progress
- 2026-09-05T20:30:19Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-05T20:30:19Z claimed by dev-MAC-2u36

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Follows: [[MAC-olrx]]

## Comments

### 2026-09-05T20:55:32Z ramirosalas
TEST-EDIT AUTHORIZED: internal/install/bootstrap_receipt_test.go -- Bounded independent RED-DISPUTE review validates the 2026-09-05 Sr. PM clarification against original AC1 parity, cmd/machinery/update.go forced-refresh help, loadReceipt/validateReceipt, buildRefreshPlan/targetInstalled/fileExists, the exact frozen test mutations, and the preserved ordinary edited-artifact diagnostic. Authorize revising only TestBootstrapReceiptCLI/stale_artifact and disappeared_target from rejection expectations into clearly named edited-owned-artifact and missing-owned-artifact repair-success cases. Both mutate only HOME/.codex/agents/machinery-fsm-author.toml with safe unchanged parents/topology and valid receipt metadata. Add actual isolated ordinary-update and bootstrap CLI controls for each, with restored current-release bytes (edited content gone / missing file recreated), complete recorded home/native/plugin plan and copy/symlink modes retained, refreshed receipt digests/topology, binary release verification, unrelated-file preservation, and same-release idempotence. Only minimal shared test structure necessary for those cases is authorized; existing assertions must remain equally strong. The missing-file ordinary control has NOT yet run: execute it and report any unexpected production failure to dispatcher/Sr. PM before expanding scope or changing expected repair semantics. Preserve all legitimate metadata/schema/path/type/plugin-ownership rejection cases, complete recorded mixed-plan and normal/bootstrap convergence tests, defaults/selectors, real post-mutation rollback and interrupted journal recovery. No other frozen assertion/fixture/config changes or production edits are authorized. Candidate v1 99956740ae5559262790a9473b5597c1775928f2 (file SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329) is NOT approved; preserve its original evidence and /tmp/machinery-MAC-2u36-red.qRk5DC logs. Every repair commit subject must carry tdd-red and [test-edit-authorized]. Re-freeze revised bytes and replay the fresh real CLI matrix against unchanged production, recording exact command, SHA/hashes, leaf counts, assertion failures, passing controls, zero skips, and justified LOC/timeout deltas, then obtain a separate independent full RED review. This is test-edit authorization only, not RED approval, delivery, GREEN waiver, acceptance, rejection, or closure.
