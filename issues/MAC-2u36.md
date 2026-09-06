---
id: MAC-2u36
title: "Converge installer reruns on recorded targets"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, red-approved]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-06T00:27:45Z
content_hash: "sha256:6c372f5e3ad5d23fe4329dad0beca5b2e501106f9732606e2c36b1005ca855ea"
blocks: [MAC-gcrr, MAC-ou97]
follows: [MAC-olrx, MAC-p8ce, MAC-a89e]
assignee: dev-MAC-2u36
---

## Description
## USER INTENT
A user rerunning the installer or ordinary machinery update must get a complete, safely repaired installation at the requested release across all recorded placements, including a missing owned native artifact. Success must mean current content, complete receipt evidence and preserved topology; failure must preserve recoverable pre-run state.

## Context (Embedded)
Three production defects share this receipt-publication and recorded-plan boundary:
- NEXT1: install.sh passes --bootstrap-defaults, but updatePlan chooses default homes before the complete recorded home/native plan.
- Newly reproduced in MAC-2u36: ordinary update retains the missing Codex target in its plan, but refreshDirectInstalls runs home groups first; each child install calls recordHomeInstallLocked -> saveReceipt -> refreshReceiptArtifacts over ALL recorded placements. A missing later Codex file aborts the first child's receipt inventory before native repair is reached.
Actual isolated CLI diagnostic on frozen v1 99956740ae5559262790a9473b5597c1775928f2 failed in 35.820s after verified binary replacement and first home-group mutation: "inventory installed artifacts for receipt: digest .../.codex/agents/machinery-fsm-author.toml: lstat ...: no such file or directory". It did not assert final rollback state. This disproves the earlier assumption that the exact missing-file ordinary run would already pass; it does NOT change the required successful repair behavior.
The existing ordinary intact convergence and edited-regular-file repair controls pass. Metadata validity, path/type safety and plugin ownership are distinct from repairable installed content drift or absence.
- Newly reproduced standalone writer defect: an existing receipt records [custom-a, custom-b] Copy:false and [copy-a, copy-b] Copy:true, plus Codex Copy:false/OpenCode Copy:true. An explicit all-copy install of [custom-a, custom-b, copy-a, copy-b] returns success, but the next native install rejects the emitted receipt with "home install paths overlap or repeat". recordHomeInstallLocked replaces only a matching first/canonical home group, leaving the second overlapping group; normalizeReceipt sorts but does not reconcile groups, and saveReceipt currently publishes without validateReceipt. Actual built-CLI evidence is preserved at /tmp/machinery-MAC-2u36-red.qRk5DC/e1fd0a3-parent-finalization.jsonl (48.882s). This is product corruption of its own receipt, separate from the legitimate absent-reference fixture repair at d9d53b980a393c527eaf856b0bba5d8bc70867ec.

## Scope decision and Ownership
This existing P0 bug owns all three defects: the added standalone writer defect violates the already-owned validated-publication guarantee in AC6 and touches the same receipt.go/install.go seams. A sibling would duplicate that ownership. No separate issue or artificial dependency is created; bounded AC7 specifies the writer correctness guarantee without adding automatic regrouping semantics.
Production ownership expands explicitly from internal/install/update.go to:
- internal/install/update.go: receipt-aware bootstrap planning, full-plan refresh coordination and final commit/publication sequencing.
- internal/install/receipt.go: bounded receipt accumulation/finalization, validation before publication of the final complete candidate receipt, and rejection of conflicting standalone home-group recording; preserve normal receipt validation/persistence.
- internal/install/install.go: child placement/receipt interaction needed for authenticated parent-owned full-plan refresh; preserve normal standalone recording and transactional restoration when conflicting recording is rejected.
Tests: internal/install/bootstrap_receipt_test.go; directly associated regression additions in internal/install/receipt_test.go and internal/install/install_test.go only when required by the changed receipt/standalone-install boundary. The AC2 proof repair below additionally scopes narrowly defined amendments to three existing receipt_test.go tests and their minimal receipt-local fixture/assertion support; this is not blanket permission to edit existing tests. Shared install_test.go write helper remains unchanged. Frozen existing assertions cannot be changed without explicit independent reviewer authorization; previous additive-only receipt/install-test authorizations do not authorize these amendments.

Additional verification-only ownership pending independent PM TEST-EDIT AUTHORIZATION: internal/install/update_test.go only the three named legacy fixture tests below; NEW internal/install/update_receipt_fixture_test.go only their actual-placement/release-archive/child-execution support; bootstrap_receipt_test.go only bootstrapFinalizationCase writer-close discrimination; NEW internal/install/bootstrap_receipt_writer_unix_test.go for Darwin/Linux descriptor query and real sensitivity controls; NEW internal/install/bootstrap_receipt_writer_other_test.go for compile-safe explicit unsupported-platform error. This is not permission to edit these files before PM authorization. All other frozen bootstrap/receipt/install-test bytes, global helpers and existing safety assertions remain unchanged.
Read-only consumers unless a concrete need is separately reviewed: internal/install/targets.go, internal/install/transaction.go, internal/install/lock.go, cmd/machinery/install.go and cmd/machinery/update.go. No blanket CLI/target/lock/journal rewrite is included. You are not alone in the codebase; preserve others' edits and coordinate shared paths with dispatcher.

## Boundary Map
PRODUCES:
- internal/install/update.go -> complete recorded-plan convergence, including safely missing owned native artifacts, under the existing update transaction.
- internal/install/receipt.go -> validated complete receipt publication coordinated with successful full direct refresh.
- internal/install/install.go -> bounded child-refresh participation while ordinary install continues to persist validated receipts.
- internal/install/bootstrap_receipt_test.go -> real CLI RED/GREEN matrix for recorded-plan, receipt-publication and standalone overlap defects with preserved controls.
- internal/install/receipt_test.go -> focused receipt-publication regression proof plus accurately reached AC2 parser/topology/inventory/digest rejection and matched private-fixture controls.
- internal/install/install_test.go -> focused standalone-install regression proof if needed.
- internal/install/update_test.go -> only three named legacy command/execution-fixture amendments after PM authorization; original intent/assertions preserved.
- internal/install/update_receipt_fixture_test.go -> NEW test-only actual downloaded-source placement, archive/server and authenticated child helper support.
- internal/install/bootstrap_receipt_writer_unix_test.go -> NEW Darwin/Linux writable-descriptor classifier and real sensitivity controls, no product code.
- internal/install/bootstrap_receipt_writer_other_test.go -> NEW complementary compile-safe explicit unsupported-platform classifier error, not a passing/skipped runtime claim.
CONSUMES:
- Existing internal/install/update.go.
  spec: updatePlan(opts UpdateOptions) (refreshPlan, error); refreshDirectInstalls(binary, source string, plan refreshPlan, run commandRunner, out io.Writer) error.
- Existing internal/install/receipt.go.
  spec: loadReceipt() (installReceipt, bool, error); saveReceipt(receipt installReceipt) (retErr error); refreshReceiptArtifacts(receipt *installReceipt) error; recordHomeInstallLocked(homes []string, copyAll bool) error; recordTargetInstallLocked(names []string, copyAll bool) error; validateReceipt(receipt installReceipt) error; normalizeReceipt(receipt *installReceipt).
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
6. A full direct-refresh operation must not require complete final artifact inventory before remaining selected placements can repair safely missing owned files. The Update parent PROCESS that holds the operation lock and owns the prepared transaction must perform final complete receipt inventory/publication itself, after every authenticated selected placement child has successfully returned and all direct placement/source checks have succeeded, and before that parent's direct transaction commit. During the direct-refresh phase, delegated placement children do not publish intermediate or final persisted receipts: the receipt's pre-run bytes, existence, file type and mode remain unchanged until parent finalization (if no receipt existed before the run, it remains absent until parent finalization). In-memory or journal-owned private coordination may accumulate selected topology within existing authority; it must not advertise an intermediate installation as final. The parent publishes a complete normalized receipt with real refreshed digests for the full recorded/selected plan, preserving recorded topology/copy modes/plugin obligations and existing supported-schema validation. Do not drop missing entries, retain fabricated/stale digest evidence, invent success for an incomplete plan, or bypass final validation. Any final inventory/publication failure before direct commit fails the operation and rolls back the whole direct update when transaction-written post-images are unchanged; preserve the existing refusal to overwrite concurrent foreign changes and its recoverable diagnostic/journal behavior. Standalone nondelegated install still records validated successful topology. Any receipt deferral/coordination remains restricted to authenticated parent-owned prepared journal scope; forged/absent parent authority, unprepared journal and out-of-scope paths remain rejected. No public or env-only skip-receipt switch. Host-managed plugin refresh retains its existing post-direct-commit ownership/obligation semantics, with no new atomic host-plugin rollback or schema-migration claim.

7. Every successful recording operation must publish a complete candidate receipt that passes the product's existing supported-schema/topology/inventory validation on the next load. Validate final normalized real inventory before receipt publication, for standalone recording as well as parent finalization; normalization alone is not validation. For the reproduced cross-group home request, and other requests whose candidate under existing same-canonical replacement semantics leaves repeated/nested homes in another recorded group, standalone install must return a nonzero actionable conflict diagnostic identifying the conflicting paths and advising a nonconflicting recorded-group retry or explicit topology reconfiguration. It must preserve the exact prior valid receipt and installation (content, symlink destinations, file types/modes and unrelated state), rolling back any owned placement writes when transaction post-images are unchanged. Do not silently delete conflicting receipt groups, forget still-installed ownership, rewrite unselected link/copy relationships, or report success then rely on next-operation validation. Preserve refusal to overwrite concurrent foreign changes and existing recoverable journal/diagnostic semantics. Fresh installs, disjoint additional groups and same recorded-group reruns (including a supported copy-mode change within that same ordered home set) remain successful and record exact valid topology/real digests; native installs still work after a rejected conflict. Parent AC6 process/unchanged-receipt sequencing remains mandatory. Automatic merging/splitting of overlapping recorded groups is not required: README.md promises exact topology after success and cmd/machinery/install.go defines first-home/copy placement, but neither defines cross-group ownership migration. The chosen conflict policy preserves this guarantee without inventing migration of unselected symlink dependents.

## Testing Requirements
- Hard TDD remains explicitly authorized. After independent PM reviews this scope/contract and authorizes the revised tests, a RED-only author revises the disputed edited/missing cases, adds the bounded multi-missing and missing-pre-state rollback proof, and any focused new boundary tests needed for AC6. No production edits before independently approved RED.
- Preserve original candidate 99956740ae5559262790a9473b5597c1775928f2, test SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329 and /tmp/machinery-MAC-2u36-red.qRk5DC. Existing 36-leaf replay: 32 pass, 4 fail, 0 skip, 271.751s; only three original failures are valid bootstrap defect proof. The fourth stale_artifact rejection is superseded. Missing-file ordinary FAIL 35.820s is new genuine defect evidence, separate from earlier edited-file ordinary PASS 36.508s.
- Preserve complete-recorded-plan, defaults/selectors, intact normal/bootstrap convergence, plugin obligations, real later-target rollback, interrupted recovery, and all legitimate safety negatives. Rename obsolete stale_artifact/disappeared_target rejection cases as repair-success requirements. Only specifically reviewer-authorized frozen-test changes are allowed.
- Fresh replay: go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json. This package deadline replaces 10m: the compiled 5a94840 broad matrix took 536.345s for 49 leaves (34 pass, 15 fail, 0 skip), including four invalid early authority-fixture failures. Corrected real authority controls take approximately 60s, bringing the conservative replay budget near 596.345s before the new AC6 positive/fault real updates and ordinary scheduling variance; 600s is inadequate for the complete expanded matrix. The 900s package bound provides room for those measured/required cases. Preserve every individual operation timeout, frozen test selection and assertion; no skipped cases, retries masking hangs or relaxed operation bounds. This is not permission for a hung individual update. Keep the existing GREEN 15m package command unchanged pending actual measurement; full preflight remains final-epic-only.
- Focused receipt/standalone-install/delegated-authority tests selected by actual test names introduced or affected. GREEN runs go test -count=1 -timeout=15m ./internal/install ./cmd/machinery for ordinary native package regression coverage. No full scripts/preflight.sh here; final epic gate owns heavy preflight.
- Record SHA, exact command, named leaf inventory, zero skips, assertion failure causes and passing controls. Compile/import/fixture/infra/timeout errors are not behavioral RED. Unexpected missing-file receipt-inventory failure is a valid success-contract assertion failure, not an acceptable permanent rejection.
- If internal deferred receipt behavior is introduced, add proof that normal standalone install still publishes correct receipts, missing/forged/out-of-scope parent authority cannot enable it, and final publication failure rolls back real prior mutations. Reuse existing authenticated delegation mechanisms; implementation details require evidence, not an assumed new public API.
- Finalization boundary proof is mandatory: use real Update with its default command runner, actual checksummed built release and real placement children, plus a successful no-fault counterpart. Independently observe current release binary/home/native content, full prepared-journal path coverage and unchanged persisted receipt through the final completed child. Then the parent's real final receipt scratch-file persistence must encounter a deterministic failure before receipt rename/commit. Use the EXISTING process-local closeInstallFile seam only after independent PM approves the concrete test: narrowly target the final receipt-* scratch file in the current prepared journal, actually close the real file and surface its OS close failure, prove the hook fired on that exact final publication operation, and preserve normal file behavior otherwise. Do not substitute a fabricated command result, new production hook/API, modified child executable or user/env-only bypass.
- Required late-failure assertions: no direct commit; exact binary/home/native/receipt pre-state restored, including absence where applicable; unrelated sentinels preserved; journal cleanup and reusable/released operation lock. The no-fault counterpart must publish complete current digests/topology and commit successfully. Compile/fixture errors, early missing-file inventory, or a never-fired hook are not this boundary proof. Existing concurrent-change rejection remains a distinct safety case; deleting a live updated artifact with os.Remove or durableRemove after a child is not a receipt-publication fault and cannot be used to demand unsafe overwrite during rollback.
- This parent-process/unchanged-receipt refinement and any proposed boundary test must receive independent PM review and explicit test-edit authorization before the RED author uses it. Existing authorized matrix work retains its prior authorization; no source edits are authorized during RED. Parent-finalization ownership remains unchanged; the current combined estimate is stated under DIFF BUDGET below, and the paused author resumes only after the pending independent scope/oracle review.
- AC7 regression after independent PM test-edit authorization: use an actual built temporary CLI and real local source/roots, seed the exact two mixed home groups plus the two native targets above through successful standalone installs, snapshot the complete real state and receipt, then issue install --from SOURCE --copy --home HOME/custom-a --home HOME/custom-b --home HOME/copy-a --home HOME/copy-b. Assert the intended overlap diagnostic/nonzero result and complete exact restoration, then run a real standalone native install --from SOURCE --target codex --target opencode --copy and require success with valid complete topology/inventory/digests. The original bug must fail on its false-success/invalid-receipt behavior, not a fixture error. Add bounded positive controls for a fresh/disjoint group and a same-ordered-group rerun with copy-mode change, checking the other group's ownership and content remain correct. Reuse the built CLI/fixture and existing snapshots; retain every safety assertion. Place additions in bootstrap_receipt_test.go or the already-owned install_test.go/receipt_test.go, selected by names containing Receipt or explicitly included in the focused command. No new API, mock runner, runtime service or production edit during RED. Runtime/package deadline changes require measurement and independent review; the existing 15m broad replay remains unchanged.
- The two paused d9d53b9 boundary-oracle defects (non-prefix macOS path normalization and desired digests computed only inside the close observer) are test defects, not extra product scope. Independent PM must review and explicitly authorize any repair and the concrete AC7 tests before RED resumes; this triage does not authorize test edits or supersede the 21:42:11Z closeInstallFile authorization.
- Separate independent PM replays revised RED before approval/freezing. GREEN preserves exact approved test/fixture/config bytes; any later repair requires explicit reviewer authorization and re-RED.

### AC2 proof repair: permissions must not mask parser/topology validation
This is a same-story verification defect, not a newly demonstrated production parser bypass or a fourth production defect. In current source, TestCorruptReceiptFailsLoudly (one leaf), TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology (eight leaves), and TestSemanticallyInvalidReceiptFailsBeforeUpdate (one leaf) create fresh receipt files through shared write(...,0644). Under the reviewed umask022 these remain0644; readReceiptFile rejects permissions before JSON parsing, while tests assert only err!=nil. Those ten existing passing leaves therefore cannot be claimed as parser/topology proof. The new bootstrap corrupt-receipt case overwrites an existing0600 file and does not share this fixture defect. Independent PM's final verdict remains separate from this scope clarification.

After independent explicit TEST-EDIT AUTHORIZED, repair those THREE existing receipt_test.go tests in place, including only necessary table fields/imports and a minimal receipt-local private-fixture/assertion helper. Preserve their intended negative cases; do not leave misnamed false-positive tests intact and merely add replacements. Use an explicitly private0700 config root and0600 regular receipt, assert actual mode/type/existence before loading and establish matched valid schema1/schema2 fixture controls through the same loadReceipt read path. Keep the global install_test.go write helper and unrelated test fixtures/assertions unchanged; no production edits or validation bypass. If the existing semantic-test name is retained, its proof must remain accurately described as loader rejection ahead of update planning; no actual binary-replacement ordering claim follows from loadReceipt alone. Any renaming requires explicit reviewer authorization and an old-to-new leaf mapping.

Every negative must change only its intended defect from an otherwise valid matched fixture and assert exists=true, nonnil error, and the specific diagnostic category/detail: malformed JSON syntax; unknown root/home/target field with that field's name; duplicate root/nested field with that key; trailing JSON value; wrong homes/copy JSON type; semantic unknown target with the rejected target value. For the unknown-home-field case, the matched home group must itself have valid nonempty safe absolute homes, so removing the unknown field yields success. Match typed JSON errors where available or remove the exact known receipt-path wrapper before checking the diagnostic detail; whole-error substring matches must not pass merely because a test directory/path contains the expected words. A private-file, directory/path, inventory or other earlier error cannot stand in for the intended parser/semantic reason. Preserve independent malformed/schema/path/type/plugin safety cases and supported-schema1 success.

Add bounded real-filesystem schema2 validation cases in receipt_test.go using a valid private receipt with actual seeded topology, complete independently known artifact inventory and real digests as the positive control. Mutate one field/category at a time: missing or extra artifact inventory entries (count mismatch); duplicated/substituted inventory path at unchanged count; relative/non-clean/unexpected absolute inventory path; digest wrong prefix, wrong length, and correct-length nonhex encoding. Require the corresponding inventory-count/path or malformed-digest detail after successful private read/parsing, with every other field valid. Retain normalized-order compatibility: reversed valid input ordering that the loader normalizes must not be invented as invalid. These fast loader/file tests complement existing real CLI behavior and do not need an additional expensive release/CLI fixture per metadata case.

Reuse existing TestBootstrapReceiptCLI/unsafe_receipt UNCHANGED as the separate actual permission negative: it starts with a successfully recorded valid0600 receipt, changes only mode to0666, requires nonzero failure and preserves exact full state. Source/readback must confirm that this is a mode-only mutation of the valid private seed; do not recast it as parser evidence. Keep all bootstrap98d3b57 bytes frozen, including the other five negative cases and every rollback assertion. The matched private schema controls establish that valid bytes are accepted when permissions are safe; do not corrupt both bytes and mode in one claimed parser test. Precise intended parser/topology/inventory/digest diagnoses belong in the repaired receipt-local cases. No new CLI case or new platform-specific source file is needed for this proof repair.

Verification sequence after all repairs are authorized and frozen: run the focused named receipt tests first, including all repaired/new cases and matched controls, with real private files and zero skips; report exact leaf identities and intended diagnostic causes. Corrected validation tests may PASS unchanged production and must be reported as passing safety controls, not manufactured behavioral RED. Any actual validator bypass discovered is a new finding requiring triage. Preserve all intended existing product RED failures and AC1-7 tests. Then perform ONE complete latest-SHA raw JSON replay with the existing go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json selection, plus explicitly named tests outside it if necessary; no costly full replay until repairs are stable. Author98d3b57 replay was66leaves49PASS17FAIL0SKIP748.254s; retain its raw history but qualify the ten misleading passes. No operation/package bound increase, frozen input weakening, test omission or retry-based masking is authorized.

This repair supplements the existing seven ACs without changing their product behavior, prior parent-process/close-fault authorization, AC7 conflict semantics, ownership-safety rules or current review status. Independent PM must first finish its current verdict and issue concrete scope/test-edit authorization before RED amendments; Sr PM triage does not reject/approve/deliver or alter claim/status/labels. All earlier authorization/evidence remains history.

### Verification repair: legacy command fixtures and actual writer-close oracle
Disposition: same-story verification-contract repair, not a fourth production bug or a new prerequisite. Retain GREEN production checkpoint 0ad71eb (+74/-21 in the same three source files) and approved RED 496963fb7f4842d706d308dafcbd145908a6e395. No production rollback defect is established by the contaminated fault oracle described below. Seven ACs, parent-process authority, actual complete final inventory, standalone validated recording and foreign-postimage protection remain unchanged. Independent PM must explicitly authorize the exact test amendments and separate RED rework before any test edit; this scope text is not TEST-EDIT AUTHORIZED, approve-red, reject, delivery, or a state/claim transition.

Source/evidence:
- Three untouched internal/install/update_test.go tests fail because their custom runners or downloaded shell only report/log success and never place artifacts. Exact GREEN focused command selected TestUpdateVerifiesReleaseAndRefreshesRecordedHarnesses, TestUpdateExecutesDownloadedBinaryForHarnessRefresh and TestUpdateRefreshesExistingDefaultInstallInPlace. All three fail final real receipt inventory for missing machinery-build-writer.md; 5.684s. Raw /tmp/machinery-MAC-2u36-green.G05gWz/legacy-runner-focused.jsonl, SHA256 da0548ffac4b9cce7dd51a7ac9d75958c88484662b97f4e26c793c78a3129cf9. Do not disable real receipt validation, fabricate digests, or accept incomplete successful updates to accommodate these fixtures.
- Frozen actual focus at 0ad71eb: go test -count=1 -timeout=7m ./internal/install -run '^TestBootstrapReceiptCLI$/(standalone_receipt|parent_finalization)' -json. Nine leaves: 5 PASS (all standalone), 4 FAIL (all parent), 0 SKIP; all 13 run events terminal; 269.131s. Raw /tmp/machinery-MAC-2u36-green.G05gWz/parent-standalone-focused.jsonl SHA256 e05ffbcc615cb5c9dee66e731607da43adf1e70cdec81c7a924c7268a67bcf5d.
- bootstrapFinalizationCase currently matches only journal scratch parent and receipt-* basename. The two no-fault updates return nil and publish the expected complete receipt, but the observer counts seven closes as publications. Each fault case matches/injects three times, contaminating subsequent rollback inventory reads. saveReceipt closes its actual os.CreateTemp writer; transaction.go digestArtifactEntry opens the same scratch path with os.Open and routes that READ handle through closeInstallFile too. A pathname is not descriptor access mode. Preserve the old raw failures honestly; do not claim clean late-fault rollback has been demonstrated.

Exact legacy amendments for independent PM authorization:
1. TestUpdateVerifiesReleaseAndRefreshesRecordedHarnesses and TestUpdateRefreshesExistingDefaultInstallInPlace retain their existing real Update invocation, legacy schema1 topology, recorded target/copy groups, release checksum/download/version behavior, exact candidate bytes, result counts, argument groups, executable identity and existing journal assertions. The third retains its old-role drift fixture and default .agents/.claude plus Claude/Codex/OpenCode plan. Their custom runners remain explicitly COMPONENT command observers, not actual released-CLI integration proof. After collecting the actual name/args, perform real filesystem placements from the actual received downloaded --from source using the existing postimage-aware placeReal/placeLinks/installTargets operations; verify/cleanup the held source via resolveInstallSource and its verify/cleanup functions. Add substantive current placement/content/link-copy checks without changing original success results or command assertions. Do not add a new final-receipt equality assertion to these compatibility controls: the unchanged old parent does not publish a final receipt; frozen AC6 tests already own that distinct behavior.
2. The first/third opts.run observers must NOT spawn a real CLI or call Install inside the locked parent. updateLocked grants scoped child capability only for its default runner (opts.run == nil). No production capability change, fabricated parent environment or reentrant lock workaround is authorized. Lower-level placement support is test-only and uses the existing active parent transaction/post-image operations; it may not save/patch a receipt or fabricate installed inventory.
3. TestUpdateExecutesDownloadedBinaryForHarnessRefresh retains the downloaded shell candidate/version branch, actual checksum-verified execution under DEFAULT runner, existing argument log and exact log assertions. After logging install args, the shell execs the existing Go test executable os.Args[0] with a narrowly named TestUpdatePlacementChildHelper and the unchanged install args after '--'. That actual helper process calls EnsureActivationConsistency then Install(Options{From: received source, Homes/Targets/Copy: received selectors, Record: true}), propagating actual failures/exit status. The existing parent-issued scoped capability and prepared journal must authenticate normally. A private MACHINERY_INTERNAL_TEST_LOCK_ROOT may align parent/helper test cache using the already-existing test-only behavior; no product/env authorization bypass. Preserve genuine downloaded-executable wiring; label this an execution-wiring fixture, NOT an actual released Machinery CLI. The frozen built-release CLI matrix remains intact and supplies AC5 end-to-end proof.
4. Confine new shared support to update_receipt_fixture_test.go: proposed test-only updatePlacementOptions(args []string) (Options, error), runUpdatePlacement(opts Options) error, TestUpdatePlacementChildHelper(t *testing.T), and updateNativeReleaseServer(t *testing.T, tag string, candidate []byte, badChecksum bool) *httptest.Server, plus only their minimal local archive/assertion support. Strictly parse the expected install --from/--home/--target/--copy vocabulary and reject malformed/unexpected input; no success-on-unknown arguments. Archive the complete real fakeSource fixture (canonical role bodies plus required OpenCode adapters) into the actual checksummed downloaded source archive. Existing updateReleaseServer/sourceTarball provide insufficient native fixture content; keep them and fakeSource/shared write unchanged, do not place from an unrelated off-download source. Close/verify real sources and server/archive resources normally. Child helper selection is scaffolding, not a required-case environment skip: the parent must actually invoke it and assert its outcome; an uninvoked helper return earns no behavioral credit.
5. No other update_test.go function/assertion changes or global helper rewrite. Keep checksum-mismatch/version-rejection/rollback and all other existing safety tests unchanged. Corrections to explanatory comments are limited to these exact three functions so their component/execution boundaries are honest; no rename or deletion is authorized by this triage.

Exact parent writer-oracle amendment for independent PM authorization:
- In bootstrapFinalizationCase only, retain its existing exact canonical journal scratch path and receipt-* prefix predicate, then classify the OPEN descriptor's access mode using the already-pinned golang.org/x/sys/unix v0.47.0 FcntlInt(fd uintptr, cmd, arg int) (int, error), command F_GETFL and mask O_ACCMODE. Query under file.SyscallConn().Control; propagate both Control and FcntlInt errors as a test failure, never silently treat unknown access as writer/nonwriter. O_RDONLY delegates unchanged to oldClose. Only O_WRONLY/O_RDWR enter the publication observer. F_GETFL is a read-only descriptor query: no bytes, offset, mode, topology, journal, lock or production code modifications.
- Count EVERY matching writable close. Keep publications == 1 in each finalization case; do not choose the first matching filename and ignore later writers. In fault mode, close the first matching writer through the original real closer, then call Close on that same already-closed file and return its real os.ErrClosed; record an explicit injected-writer count == 1. Later matching writers still increment publication count and fail the one-publication invariant, but must not receive repeated injected faults. All read closes and unrelated paths use original behavior, including scratch/journal rollback readers.
- Preserve unchanged receipt bytes/existence/type/mode through all completed actual children, exact prepared-journal coverage, complete actual binary/home/native placement checks, independently expected normalized scratch payload/current digests, no direct commit on fault, exact full rollback including prior absence and unrelated sentinels, journal cleanup, and real reusable/released operation lock. No observer-removal, weakened success/rollback assertion, fabricated fault/result, new production hook or close/durability suppression.
- New bootstrap_receipt_writer_unix_test.go has darwin || linux build constraint and only the minimal classifier plus mandatory real-temp-file sensitivity tests. Classifier signature may be bootstrapReceiptWritable(file *os.File) (bool, error). Open the SAME path as O_RDONLY, O_WRONLY and O_RDWR and prove only writable handles match; distinct writable handles on that path must each match. Check bytes, file mode and current offsets unchanged by query, handles still usable, and closed/invalid descriptor query errors are not success. Include a matched normal-close control. No skip-if-missing or mocked descriptor flags.
- New bootstrap_receipt_writer_other_test.go has complementary !darwin && !linux constraint and the same signature returning an explicit unsupported-platform error, never false/nil, success, or a skip. This preserves test compilation without inventing supported Windows runtime behavior. The required runtime lane remains Darwin/Linux; Windows product cross-build is distinct. No dependency/toolchain update or new production platform file.

Verification/staging after independent PM authorization:
- Separate RED author applies only the named test amendments/additions. Preserve production 0ad71eb and original frozen 496963f; do not rewrite either history or fold test repair into the GREEN implementation commit. Every authorized amendment commit uses tdd-red and [test-edit-authorized].
- First run new descriptor controls and corrected three legacy tests on unchanged-production baseline 496963f with ONLY the authorized revised test files, then on candidate 0ad71eb with those same files. They are compatibility/oracle controls expected to PASS both, not manufactured product RED. Explicit exact command: go test -count=1 -timeout=2m ./internal/install -run '^(TestUpdateVerifiesReleaseAndRefreshesRecordedHarnesses|TestUpdateExecutesDownloadedBinaryForHarnessRefresh|TestUpdateRefreshesExistingDefaultInstallInPlace|TestBootstrapReceiptWriter.*)- Automatic cross-group merge/split/recanonicalization: no supported ownership-migration contract has been established; AC7 requires safe conflict rejection, while any future regrouping feature needs a separately reviewed contract. Do not misclassify validation before publishing this story's own receipt as out of scope.
- Arbitrary missing-root recreation, changed unsafe parents, and ownership uncertainty: this story preserves their safety boundary; it does not authorize blind repair.
- Redesign of host-managed plugin transactions, lock/journal format or public install selectors: outside the direct receipt-publication defect; report a concrete need for review before expanding.
- Other assessment subsystems stay in sibling stories. No pushes, sync, remote mutation, installed binary/plugin/skill/agent replacement, dev-link or healthy worktree cleanup. NIL uses the installed product.
- Final heavy preflight, main merge and isolated release candidate belong to epic gate MAC-ou97.

## DIFF BUDGET
Revised verification forecast: 9 changed files (3 owned production files; existing bootstrap/receipt tests; the three-function update_test amendment; new update_receipt_fixture_test.go and two platform-separated writer-query test files), approximately1,850 combined changed LOC. Measured approved test delta was1322 and retained production is95 changed lines (+74/-21), totaling1417. Author forecasts160–220 new legacy helper lines plus15–30 replacements (~190–280 additions+deletions), and65–105 writer-query/hook/sensitivity lines plus modest complementary-platform/error-control overhead (~75–120). Projected combined: 1417+190–280+75–120 = approximately1682–1817;1850 allows modest honest rounding, not padding or an automatic acceptance cap. This supersedes the4–6-file/~1700 estimate because actual downloaded-native fixtures and a descriptor-specific oracle require explicit local support. Report actual per-file additions/deletions, helper reuse and focused/full elapsed costs; PM investigates material overrun, new files or changed ownership rather than trimming safety proof. Shared write/fakeSource/sourceTarball/updateReleaseServer remain unchanged. Preserve all operation bounds and existing15m package command; prior measurements/authorizations remain historical.

## Dependencies
Parent MAC-ui8a; remains blocking MAC-gcrr and MAC-ou97. No new inter-story dependency: one developer owns all three coupled installer defects and one independent PM reviews the full outcome. No other sibling implementation declares the three owned installer source seams. Consumer guidance and epic acceptance remain blocked until this story is accepted.

## MANDATORY SKILLS
- developer for RED/GREEN roles; codebase-memory for source discovery; pm_acceptor for independent scope/test authorization, RED replay and acceptance.

## Delivery Requirements
This AC7 overlap repair, revised measured budget and concrete new tests require independent PM review before author re-RED; no prior authorization is silently broadened. Preserve all existing authorization history, including parent-process finalization and the exact existing closeInstallFile fault/control authorization at21:42:11Z. The paused oracle repairs require their own explicit reviewer authorization. Triage grants no test edits, production edits, RED approval, delivery or status/claim transition. Do not deliver or approve v1 unchanged.
Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands/results, inventory, changed-file rationale, per-AC proof and residual limits. Product has no dependency on these development coordination tools.
 -json. Name every new leaf and classify child helper scaffolding separately. Setup/compilation/lock mismatch is not RED.
- Replay the four corrected actual parent cases on the same unchanged-production baseline: the missing parent-owned finalization/child receipt mutation assertions must still fail for their intended AC6 reasons, with no claim that the writer fault fired if it did not. Then replay them against candidate with identical revised tests: expected no-fault one real writer publication plus successful complete receipt, and one actual writer close fault plus full rollback. Any residual failure requires diagnosis/triage, not weakening or asserting that the new oracle guarantees the production fix.
- Preserve all other frozen bootstrap, receipt and shared install_test bytes. After focused stability and independent review, refreeze exact SHA/hashes and run ONE full current-SHA existing 15m Bootstrap|Receipt|UpdatePlan matrix on the appropriate re-RED base; keep the three legacy tests separately explicitly selected because that regex omits them. Final GREEN full matrix/native package regression remains required after tests are frozen. No extra costly broad replay before fixture stability, no per-operation/package deadline increase, missing cases, retry masking or skip.
- Runtime forecast: corrected three legacy fixtures roughly 10–25s per focused run versus observed broken 5.684s; descriptor controls negligible but measure. Four parent-case focus observed 160.19s including its reference setup (about 180s conservative estimate), combined standalone+parent observed 269.131s. Budget baseline and candidate focused runs separately. Preserve 90s parent operation bound, all existing child bounds, 7m selected focus and 15m broad/native bounds; forecast does not relax them.

Scanner attribution (independent final PM disposition pending): five reported return-empty findings lie in unchanged branches, not new stubs: pluginStderrDiagnostic returns empty only for empty stderr; installFileChangeID returns empty when platform stat identity is unavailable (nil info/Sys, nil pointer, nonstruct, unsupported fields). Exact 496963f..0ad71eb diff does not change these branches. No suppression, whitespace workaround, assertion waiver or new product defect is inferred. Independent final reviewer still evaluates their legitimate fallback semantics.

## OUT OF SCOPE
- Automatic cross-group merge/split/recanonicalization: no supported ownership-migration contract has been established; AC7 requires safe conflict rejection, while any future regrouping feature needs a separately reviewed contract. Do not misclassify validation before publishing this story's own receipt as out of scope.
- Arbitrary missing-root recreation, changed unsafe parents, and ownership uncertainty: this story preserves their safety boundary; it does not authorize blind repair.
- Redesign of host-managed plugin transactions, lock/journal format or public install selectors: outside the direct receipt-publication defect; report a concrete need for review before expanding.
- Other assessment subsystems stay in sibling stories. No pushes, sync, remote mutation, installed binary/plugin/skill/agent replacement, dev-link or healthy worktree cleanup. NIL uses the installed product.
- Final heavy preflight, main merge and isolated release candidate belong to epic gate MAC-ou97.

## DIFF BUDGET
Expected 4-6 files, approximately1700 combined changed LOC: roughly1350 tests and350 production across the same three owned source files. Measured98d3b57 bootstrap suite is1111 lines, already above the earlier1050 test forecast; bounded AC2 private-fixture/precise-diagnostic repair and schema2 cases are estimated at another150-200 changed test lines, retaining the prior350 production allowance and modest rounding headroom. This supersedes the under1400 forecast transparently, not an automatic acceptance cap or permission to compress away proof. Keep shared write unchanged and reuse the existing real CLI permission negative; no duplicate costly CLI fixtures for loader cases. Report actual per-file additions/deletions, helper reuse and elapsed focused/full replay costs; independent PM investigates material overrun or any new file/scope before authorization. Preserve required safety tests, individual operation bounds and the15m package command. Earlier budget measurements and authorizations remain historical.

## Dependencies
Parent MAC-ui8a; remains blocking MAC-gcrr and MAC-ou97. No new inter-story dependency: one developer owns all three coupled installer defects and one independent PM reviews the full outcome. No other sibling implementation declares the three owned installer source seams. Consumer guidance and epic acceptance remain blocked until this story is accepted.

## MANDATORY SKILLS
- developer for RED/GREEN roles; codebase-memory for source discovery; pm_acceptor for independent scope/test authorization, RED replay and acceptance.

## Delivery Requirements
This AC7 overlap repair, revised measured budget and concrete new tests require independent PM review before author re-RED; no prior authorization is silently broadened. Preserve all existing authorization history, including parent-process finalization and the exact existing closeInstallFile fault/control authorization at21:42:11Z. The paused oracle repairs require their own explicit reviewer authorization. Triage grants no test edits, production edits, RED approval, delivery or status/claim transition. Do not deliver or approve v1 unchanged.
Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands/results, inventory, changed-file rationale, per-AC proof and residual limits. Product has no dependency on these development coordination tools.

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: This story's required current cases are service-free Go/native filesystem/local-process tests and real local CLI paths; no Docker/Java/Node dependency is implicit in ordinary native suites. Preserve actual non-mocked path tests. If implementation introduces any external runtime-backed case, it MUST add a dedicated closed fragment under testdata/integration-lanes via the shared required lane and declare ownership/dependency before delivery; no skip-if-missing, env-gated omission or reliance on later incidental execution. Missing service cannot silently convert required coverage to success.
RED-DISPUTE ADDENDUM — disappeared_target classification:
The frozen disappeared_target fixture removes only the regular owned file <temporary HOME>/.codex/agents/machinery-fsm-author.toml after a valid receipt was recorded. It does not substitute a symlink, change a parent path/type, or introduce unrelated ownership. The current bootstrap run rejects and preserves pre-run state (PASS, 18.65s), but no ordinary-update control for this exact missing-file input was run. Its acceptance assertion therefore needs the same AC1/AC2 contract review: supported ordinary forced repair may recreate a missing owned artifact. Do not infer unsafe ownership from absence alone. The passing negative demonstrates current bootstrap behavior, not proof that this is the intended permanent contract. Root requested no additional expensive CLI runs before Sr PM clarification.
SR PM VERIFICATION-CONTRACT REPAIR — scope canonicalized; independent PM TEST-EDIT AUTHORIZATION pending.
Same existing MAC-2u36/P0 under MAC-ui8a, no new product bug or prerequisite. Canonical readback verified exact scoped repair and all seven AC byte-for-byte; all previous notes/history/comments and claim/status/labels/dependencies preserved. GREEN0ad71eb retained, approved RED496963f retained; no source/tests/docs/worktree edits or state transition.
Required independent PM decision is the exact joint test scope: three named update_test.go legacy fixtures with local update_receipt_fixture_test.go real placements/downloaded native source/actual authenticated helper; bootstrapFinalizationCase only writable-descriptor matching and one real writer fault, plus two platform-separated writer query/control test files. No global helper rewrite. First/third remain component command observers, second execution-wiring fixture; frozen actual released CLI matrix still owns end-to-end AC5. opts.run must not gain/spoof parent authority or reenter Install under parent lock.
Descriptor proposal verified in local pinned x/sys/unix v0.47.0 source: FcntlInt(fd uintptr, cmd,arg int)(int,error), F_GETFL and O_ACCMODE for Darwin/Linux; query via SyscallConn.Control. Count every real writable receipt close, exclude read-only closes, real first-writer double-close error only once; retain every child sequencing/complete payload/final inventory/rollback/foreign-change/lock assertion. Complementary unsupported test-platform file returns explicit error, not passing/skip. New helper/control runtime proof remains pending PM authorization and RED author.
Corrected legacy/descriptor controls must pass both unchanged-production496963f and candidate0ad71eb; corrected parent cases must retain intended missing-finalization RED on unchanged production and then execute one actual writer fault/full rollback on candidate. A GREEN-only run is not re-RED evidence. No fabricated receipt/digest, durability bypass, source changes to stop reader closes, retry or timeout relaxation.
Verified raw SHA256s: legacy-runner-focused.jsonl da0548ffac4b9cce7dd51a7ac9d75958c88484662b97f4e26c793c78a3129cf9 (3 failures,5.684s); parent-standalone-focused.jsonl e05ffbcc615cb5c9dee66e731607da43adf1e70cdec81c7a924c7268a67bcf5d (9leaves5PASS4FAIL0SKIP269.131s/all13runterminal). The newly reached path-only matcher observes7closes in no-fault and injects3times in fault, including real read handles; full clean fault rollback not proven, no new production rollback defect claimed.
Budget revised to9files/~1850combined LOC: measured1322test+95source=1417, plus190–280legacy fixture amendments and~75–120writer/platform controls =>1682–1817projected; actual reporting required. Four parent focus160.19s measured (~180forecast); legacy repaired10–25s forecast per run. Preserve existing90s operation/7mfocus/15mbroad bounds. Full preflight remains epic-final.
Five scanner returns source-attributed to unchanged empty-stderr/unavailable-stat fallback branches; no suppressions or final acceptance claim. Parent/PM disposition remains independent.
Initial instruction/context/source checks only, not scripts/preflight.sh. Graph source discovery and exact candidate/base diff used; graph generation2026-09-05T23:58:53Z best-effort metadata_match for existing installer paths; new bootstrap file absent from main index was read directly at retained story checkpoint. An earlier guessed log filename was a reported lookup error, parent authorized exact supplied path continuation; not a product finding.

Prior canonical budget preserved verbatim:
SR PM terminal structural checks for the joint verification repair: pvg lint --backlog --epic MAC-ui8a PASSED, 33 issues, 0 errors/0 review findings; pvg nd dep cycles none; pvg rtm check --epic MAC-ui8a PASSED with 18 stories/3 closed, 0 tagged requirements extracted/0 uncovered (structural result, not product AC proof). Independent PM exact test-edit authorization and separate RED rework remain pending. Root main497419a; no state transition or production/test edits.

## DIFF BUDGET
Expected 4-6 files, approximately1700 combined changed LOC: roughly1350 tests and350 production across the same three owned source files. Measured98d3b57 bootstrap suite is1111 lines, already above the earlier1050 test forecast; bounded AC2 private-fixture/precise-diagnostic repair and schema2 cases are estimated at another150-200 changed test lines, retaining the prior350 production allowance and modest rounding headroom. This supersedes the under1400 forecast transparently, not an automatic acceptance cap or permission to compress away proof. Keep shared write unchanged and reuse the existing real CLI permission negative; no duplicate costly CLI fixtures for loader cases. Report actual per-file additions/deletions, helper reuse and elapsed focused/full replay costs; independent PM investigates material overrun or any new file/scope before authorization. Preserve required safety tests, individual operation bounds and the15m package command. Earlier budget measurements and authorizations remain historical.


## nd_contract
status: in_progress

### evidence
- Guarded supported single-executable EDITOR canonical edit plus full shared readback preserved seven AC, prior history and live metadata.
- Exact same-story verification repair ready for independent PM test-edit and separate RED rework authorization; GREEN production retained undelivered.
- No authorizations, delivery/approval/rejection/status/claim changes or source/test/installed-asset mutations performed by triage.

### proof
- [ ] AC #1-3: retained frozen proof and complete updated regression replay pending
- [ ] AC #4: corrected uncontaminated late-writer-fault rollback and complete regression pending
- [ ] AC #5: retained actual released-CLI integration; component fixtures are not substitutes
- [ ] AC #6: scoped descriptor-specific oracle repair and real fault/normal control pending independent PM authorization
- [ ] AC #7: retained standalone actual5PASS checkpoint; final full acceptance pending


## MAC-2u36 GREEN checkpoint — intentional verification-review hold

Implementation committed 0ad71eb on retained story/MAC-2u36 at /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-2u36. Only update.go, receipt.go, install.go changed (+74/-21,95 changed production LOC). Bootstrap uses full recorded/discovered plan when receipt exists; authenticated delegated tx children defer receipt writes; parent merges completed selections into prior ownership and publishes complete normalized real inventory before commit; standalone conflicting home groups reject with actionable paths and validated receipts remain required. No frozen tests, journal/lock/target/CLI implementation, public bypass or installed assets changed.

PROOF (checkpoint, not delivery):
- Fast focused canonical planning/private validators: go test -count=1 -timeout=2m ./internal/install -run '^(TestCorruptReceiptFailsLoudly|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology|TestSemanticallyInvalidReceiptFailsBeforeUpdate|TestReceiptPrivateSchemaControls|TestReceiptSchemaTwoInventoryValidation|TestBootstrapReceiptPlan|TestBootstrapDefaultPlanIsPluginAware)$' -json. Exit0,27 leaves PASS,12.020s; tool display only, no complete raw artifact claimed.
- Three legacy runner checks: go test -count=1 -timeout=2m ./internal/install -run '^(TestUpdateVerifiesReleaseAndRefreshesRecordedHarnesses|TestUpdateExecutesDownloadedBinaryForHarnessRefresh|TestUpdateRefreshesExistingDefaultInstallInPlace)$' -json. Exit1,3FAIL0SKIP. Complete raw /tmp/machinery-MAC-2u36-green.G05gWz/legacy-runner-focused.jsonl SHA256da0548ffac4b9cce7dd51a7ac9d75958c88484662b97f4e26c793c78a3129cf9. Their untouched mocked runners/shell only report or log success without materializing recorded artifacts; final real receipt inventory correctly detects missing machinery-build-writer.md. RED-DISPUTE reported; no production inventory bypass or test amendment.
- Actual frozen focus: go test -count=1 -timeout=7m ./internal/install -run '^TestBootstrapReceiptCLI$/(standalone_receipt|parent_finalization)' -json. Exit1,9leaves5PASS4FAIL0SKIP269.131s; all13run events terminal. Raw /tmp/machinery-MAC-2u36-green.G05gWz/parent-standalone-focused.jsonl SHA256e05ffbcc615cb5c9dee66e731607da43adf1e70cdec81c7a924c7268a67bcf5d.
- All5 standalone supported/conflict/next-native leaves PASS, including exact restoration. All4 parent leaves reach parent finalization but frozen pathname-only close matcher also captures subsequent READ descriptors: no-fault7matches each though real Update succeeds and complete receipt matches; fault3matches/injections each, contaminating cleanup/rollback. Source: saveReceipt303 actual writer; digestArtifactEntry in transaction.go1743/1755 opens same scratch with os.Open and routes read Close through same seam. Full clean fault rollback not yet proven. Separate RED-DISPUTE and actual-handle F_GETFL discriminator proposal sent to dispatcher/Sr PM; no edits authorized or made.
- pvg verify reports5 heuristic stub findings in unchanged legitimate return-empty branches: update.go993 pluginStderrDiagnostic empty stderr; install.go1157/1162/1167/1184 installFileChangeID unavailable platform-stat identity. No new stub/suppression/whitespace workaround. Independent reviewer disposition requested.
- pvg story verify-tdd --base f24b2df PASS11commits0unauthorized edits; git diff --check PASS. Frozen bootstrap SHA256ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23, receipt SHA256a56a3dc07bd447a0e4d4469f0d027509e5149eb26b494d118d394089cbfb5c48, shared install SHA256d5039079dd032f8b86f86ad6bd0ac411b192983b10ee194e599d6e59bcba97ba unchanged.
- Required full latest-SHA matrix/native regression remains pending. Coverage not instrumented. No broad replay or full preflight; all test sessions terminated normally with recorded exit codes.

LEARNINGS:
- Parent real inventory exposes legacy command-only success fixtures; materialize actual downloaded-source placements without pretending those component tests are real CLI evidence.
- A scratch pathname identifies a file, not its publication writer handle; read-close fault injection can break independent rollback inventory and invalidate the oracle.
- Preserve correct real inventory/durability behavior; independently review exact frozen-fixture repairs before changing tests.

## nd_contract
status: in_progress

### evidence
- Clean committed GREEN checkpoint0ad71eb; root-directed intentional hold after completed focus, retaining claim/worktree.
- No story deliver/release/approve/close, test edits, remote operations or installed product changes.
- Exact raw evidence and narrow repair proposals supplied for independent same-story review.

### proof
- [x] AC #1: focused recorded/default/schema1 plan checks pass.
- [x] AC #2:23private receipt parser/topology/inventory/digest controls pass; broad safety replay pending.
- [ ] AC #3: full real convergence/missing repair/idempotence replay pending.
- [ ] AC #4: full rollback/interruption replay pending.
- [x] AC #5: actual frozen focused CLI/children executed; no skip/live install mutations.
- [ ] AC #6: actual parent finalization reached; overbroad frozen close oracle requires independent repair, clean late-fault proof pending.
- [x] AC #7:5actual standalone supported/conflict/restoration/follow-on cases pass.
- [ ] Final unchanged/reapproved tests, complete package regression and independent acceptance pending.


## nd_contract
status: red-approved

### evidence
- RED tests approved via pvg story approve-red on 2026-09-05.

### proof
- [ ] GREEN developer must implement against the approved RED tests without modifying them.


# FULL INDEPENDENT RED REVIEW — MAC-2u36

Decision: APPROVE-RED for frozen 496963fb7f4842d706d308dafcbd145908a6e395. The repaired tests close the prior material AC2 proof gap. This approves the RED acceptance bar only; the story is not accepted/closed, and GREEN implementation and actual final-close fault execution remain pending.

## Scope, authorization and static review

Reviewed current canonical seven ACs, scope, testing requirements, measured budget, latest delivery and all applicable authorizations through shared `pvg issues show MAC-2u36 --json`. Preserved distinctions among historical six-AC scope, later parent-PROCESS refinement, the retrospectively reviewed cdb8036 diagnostic amendment, AC7 overlap behavior, prior oracle repairs, and the final narrow AC2 authorization at 23:04:15Z. No prior blanket test-edit authorization was inferred.

The entire 98d3b57..496963f delta is receipt_test.go +186/-25. It amends only TestCorruptReceiptFailsLoudly, TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology, and TestSemanticallyInvalidReceiptFailsBeforeUpdate, with required imports/table fields, three private receipt helpers and two new bounded test functions. Names and all intended negatives remain. Shared install_test.go and every bootstrap byte are unchanged. No production/config/API/hook/CLI/shipped asset edits occurred.

Fresh deterministic checks: git diff --check f24b2df 496963f clean; gofmt -l both test files empty; pvg verify both test files --format text --include-tests PASS, 2 files/0 issues; pvg story verify-tdd --base f24b2df PASS, 10 commits/0 unauthorized edits/0 skipped merges. Every amendment subject carries tdd-red and [test-edit-authorized]. No new stub, skip, environment gate or parallel mutable fixture. Existing receipt-test platform skips remain unchanged and none executed in either relevant native replay. Go signatures provide explicit types; no new public product API/config/cross-cutting integration exists to audit. This RED-only test repair changes no documented product behavior, so no DOCS_STALE finding.

Verified SHA256:
- bootstrap_receipt_test.go: ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23
- receipt_test.go: a56a3dc07bd447a0e4d4469f0d027509e5149eb26b494d118d394089cbfb5c48
- install_test.go: d5039079dd032f8b86f86ad6bd0ac411b192983b10ee194e599d6e59bcba97ba

## Evidence freshness and exact counts

Fresh independent focused replay in detached /tmp/machinery-pm-MAC-2u36-final.EeD9Ho/checkout at 496963f:

`go test -count=1 -timeout=2m ./internal/install -run '^(TestCorruptReceiptFailsLoudly|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology|TestSemanticallyInvalidReceiptFailsBeforeUpdate|TestReceiptPrivateSchemaControls|TestReceiptSchemaTwoInventoryValidation)$' -json`

Native stdout/stderr directly captured to /tmp/machinery-pm-MAC-2u36-final.EeD9Ho/496963f-independent-focused.jsonl. Exit 0; 23 leaves, 23 PASS/0 FAIL/0 SKIP, 0.441s. All 26 run events have terminal outcomes. Raw SHA256 4ce4f6378c140208a34a06eaecbbd40ddea7712bde0fbc26ffb538e0c96e13de. Twenty intended negative diagnostics and three named positive controls executed.

Primary current-SHA full integration evidence is the author's complete native replay, independently parsed and audited here, NOT a fresh full PM replay:

`go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json`

/tmp/machinery-MAC-2u36-red.qRk5DC/496963f-final-matrix.jsonl, independently verified SHA256 a42f96c5d36651fbd835d9626057eb3f659ae6b2503668d4dd6375407dce7b03. Exit 1 intended RED; 79 leaves, 62 PASS/17 FAIL/0 SKIP, 739.525s. All 89 run events have terminal outcomes. Bootstrap subset: 43 leaves, 26 PASS/17 FAIL; other selected tests: 36 PASS. The complete named inventory is preserved in this raw log and the canonical delivery. All 23 changed/new checks also pass in this full invocation. No compiler, fixture, package/operation timeout, race warning or unexpected failure class was found. No statement/branch coverage was instrumented; seven of seven AC outcome contracts were independently assessed, not a numeric code-coverage claim.

Carried-forward corroboration: prior independent 98d3b57 full replay, 66 leaves/49 PASS/17 FAIL/0 SKIP, 758.895s; /tmp/machinery-pm-MAC-2u36.DDIEBx/98d3b57-independent-matrix.jsonl, independently reverified SHA256 9457a41c68ebdc041dc4ffcfccc71a9026dd595b858349f8429e48b2cefba778. Every prior run was terminal. Exact 17 failing leaf identities equal the current replay. The old ten permission-masked PASS results are historical execution results ONLY, not old valid AC2 parser/topology proof.

Why no redundant full PM replay: the complete current author evidence is trustworthy and consistent; the only changed file contains the locally isolated repaired/new loader tests, freshly independently executed above. Bootstrap test bytes, all production source, archive-building code, shipped installation content and shared helpers remain identical to the prior independent replay. The real archive includes the changed test file and built binaries may carry different revision metadata, so byte-identical archives/binaries across SHAs are NOT claimed. Current author execution supplies that current-SHA proof; prior independent integration execution corroborates the unchanged behavior/oracle closure. Test environment cleanup and absence of new package initialization/global mutations were inspected. No specific unresolved doubt warrants another approximately twelve-minute broad run. This follows the PM evidence discipline and user's targeted-verification preference; it is a complete independent review, not a claim of a second fresh full current-SHA execution.

## AC-by-AC behavioral review

AC1: complete-recorded-plan ordinary/bootstrap comparison includes both home groups with preserved modes, Codex/OpenCode modes and plugin obligations. Defaults and supported schema1 controls pass; selectors remain rejected. The actual plugin-aware default test passes. Recorded-plan parity fails as intended because bootstrap selects defaults. The real convergence tests prevent an apparent success that refreshes only default homes.

AC2: repaired helpers call the SAME real loader with a matched valid receipt before each negative; explicitly private 0700 config and existing regular 0600 receipt modes/types are checked before each read. `exists=true`, nonnil error, exact parse/invalid wrapper removal and intended detail are mandatory. Earlier permissions/path/read failures have the wrong wrapper and cannot pass. Directory/test names cannot satisfy details. Malformed input additionally requires json.SyntaxError; wrong homes/copy require json.UnmarshalTypeError, string value, correct field suffix and []string/bool type. Unknown root/home/target fields assert their exact names; nested/root duplicates assert keys; trailing value and cursor target assert their specific diagnoses. Unknown-home fixture has nonempty safe absolute homes and passes when the field is removed. The semantic test accurately claims loader rejection before planning, not observed binary replacement.

Schema2 controls seed two actual home trees, independently enumerate six known artifact roots and compute real digests. Count defects are 5/7 versus 6; unchanged-count duplicate/substituted/relative/non-clean/unexpected-absolute paths require the inventory-path mismatch category. Sorting may expose another shifted inventory entry first; this is the same intended path-membership category, not a false parser/permission attribution. Digest prefix/length/correct-length nonhex defects keep all other metadata valid and require malformed-digest diagnosis for the mutated artifact. Reversed valid ordering succeeds and normalizes. Named schema1/schema2 positives compare exact loaded metadata. The existing CLI unsafe_receipt starts from successfully recorded private data and changes only mode to0666; it passes independently of parser checks (current author 13.47s). Bootstrap malformed/schema/plugin/path/type controls remain frozen; unsafe native symlink is intended bootstrap RED. No new validator bypass was found.

AC3: actual ordinary intact (51.37s) and edited regular Codex artifact repair (49.65s) pass with exact verified-release binary/version, independent real-release filesystem reference, complete topology/path inventory/current digests/plugin obligations, mode/link destinations, unrelated sentinel and same-release idempotence. Paired bootstrap and missing-native cases preserve desired successful repair assertions. The exact missing Codex file and bounded simultaneous Codex/OpenCode file absences follow valid receipt creation with safe parents. Ordinary missing currently fails during early child inventory, a genuine success-contract RED rather than unsafe-ownership success. Missing entries cannot simply be forgotten because expected content and receipt membership are independently retained.

AC4: ordinary intact later OpenCode source failure passes (32.60s). The fixture removes only the archive's actual OpenCode adapter; binary/home/Codex progress witnesses, exact adapter path plus no-such-file diagnosis, nonzero result and complete pre-state snapshot restoration are mandatory. Missing-prestate cases separately require restored absence and actual later boundary, so early Codex receipt-inventory errors fail. Bootstrap later-source cases remain intended RED. Real killed held-download startup recovery passes (14.67s), with journal presence, CLI startup recovery, exact pre-state and journal disappearance. This is PRE-mutation interruption evidence; it is not substituted for post-mutation rollback.

AC5: reviewed all 1111 bootstrap-test lines and relevant source seams. Tests build actual Cobra/release executables, generate checksummed real source archives, use private loopback endpoints and real subprocess/default command runners with temporary HOME/config/target/binary roots. Endpoint/version linking is fixture routing; no fake command result, modified child, production hook or public/environment-only receipt bypass. New metadata tests use real filesystem loader operations. All mandatory new integration leaves executed, zero skips. No Paivot/pvg/nd product dependency was added; coordination remains private tracker use.

AC6: full parent prepared journal covers binary, complete selected artifacts and receipt. Parent observer sees genuine buffered child completion, checks persisted receipt bytes/existence/type/mode after every child, independently verifies all current content, and builds desired current normalized receipt outside the close callback. Exact component-bounded alias matching preserves foreign siblings/interior aliases and exact scratch parent identity. The existing closeInstallFile hook targets only receipt-* directly under the current journal scratch, checks prepared phase, expected-ready content/payload and unchanged receipt, successfully closes the real file, then returns actual second-close os.ErrClosed in fault cases. Required child/publication counts, retained error cause, no completion/commit, exact rollback/absence, journal cleanup and actual lock reacquisition remain strict. Both receipt-present and legitimate explicit absent-plan cases have matched no-fault controls. Current four leaves observe premature child persistence and zero parent publications: missing-boundary RED, with NO real late close injection/rollback claim. All four absent/forged/unprepared/out-of-scope authority controls pass and require bounded failure and unchanged content/receipt; no unsafe forged authority can enable success. Existing concurrent foreign-postimage refusal remains unchanged and must be preserved in GREEN.

AC7: actual four-home and repeated cross-group requests require actionable path-specific conflict, exact prior installation/receipt restoration, followed immediately by successful actual native install without manual receipt repair. Their current false success, changed state and next native rejection are intended writer-validation RED. Fresh/disjoint/same-ordered-group-copy-change controls all pass with exact fixed topology, source content, link destinations and real digests; unrelated/unselected state is checked. The conflict fixture permits safe rejection before placement, and does not authorize deleting groups or inventing ownership migration. Parent finalization requirements remain separate and mandatory.

## Budget, limits and learnings

Total delivery from f24b2df: bootstrap +1111/-0, receipt +186/-25 =1322 changed test lines across two files. Adding the scoped approximately350 production estimate gives1672 within approximately1700 combined forecast. Receipt change is11 lines above the upper200-line repair estimate, justified by explicit actual modes, matched controls, typed/detail assertions and independent six-root inventory. Shared fixtures avoid extra CLI cost. No material scope overrun or new file is hidden; do not weaken assertions to meet an estimate.

Native darwin/arm64 evidence only. No Linux/Windows runtime, full package/race suite or final preflight claim. Prior POSIX alias-literal limitation remains an out-of-lane observation, not a new blocker under the current runtime contract. No new DISCOVERED_BUG found.

LEARNINGS:
- Verify the reached validation layer, not only nonnil errors or named PASS counts. Actual private modes and matched positives prevent permission-masked proof.
- Strip exact known wrappers before matching diagnostics; typed parser errors and real independent inventories strengthen attribution without expensive duplicate CLI fixtures.
- Separate a RED assertion for a missing finalization boundary from actual fault execution. GREEN must prove exact hook occurrence and rollback; zero injections cannot count as late-fault coverage.
- Carry forward only byte-closed source/oracle evidence, explicitly label historical and author-owned results, and independently rerun the changed proof when warranted.

## nd_contract
status: in_progress

### evidence
- Full independent seven-AC RED review at496963f; fresh23/23receipt checks, audited current79-leaf62PASS17FAIL0SKIP full author evidence and prior independent66-leaf replay with corrected historical attribution.
- Fresh static/format/TDD checks passed; frozen hashes and exact authorized scope verified.
- Root remained clean main497419ab4512fcff765cd5feb27aed4c67b5608d; review used only its own detached checkout and temp fixtures. No author checkout, installed product, external services/resources, remote refs or healthy user resources modified. Installed machinery SHA2565205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849 unchanged.

### proof
- [x] AC #1: full recorded-plan/default/plugin/schema1/selector RED contract reviewed.
- [x] AC #2: prior blocking proof gap closed with fresh attributable23-leaf private receipt proof and frozen safety controls.
- [x] AC #3: substantive actual full-release repair/convergence and idempotence RED contract reviewed.
- [x] AC #4: actual later-source rollback and separately classified interruption; missing-prestate desired contract retained.
- [x] AC #5: actual release/CLI/children/private roots and no executed skips verified.
- [x] AC #6: strict authenticated parent publication/fault contract verified; actual late close execution remains required in GREEN.
- [x] AC #7: conflict/restoration/next-native and supported recording controls reviewed.
- [x] RED acceptance bar independently approved; canonical approve-red transition follows.
- [ ] GREEN implementation, unchanged frozen tests passing, actual late-close fault rollback and final product acceptance remain pending.


## Reworked RED delivery readback
At frozen496963fb7f4842d706d308dafcbd145908a6e395, pvg story verify-delivery MAC-2u36 returned Passed:9, Failed:0 after one canonical deliver for this rework and the append-only terminal delivered comment. Fresh readback confirms in_progress plus delivered label, authoritative delivered contract, current79leaf62PASS17FAIL0SKIP proof, currentSHA and explicit historical AC2 correction. No second broad replay or repeat deliver in this attempt.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence
AC2 RED REWORK DELIVERY — MAC-2u36 — frozen496963fb7f4842d706d308dafcbd145908a6e395. Production remains unchanged; this is RED proof for independent review, not GREEN completion or RED approval.

PROOF:

### CI/Test Results
Commands run:
1. cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-2u36 && go test -count=1 -timeout=2m ./internal/install -run '^(TestCorruptReceiptFailsLoudly|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology|TestSemanticallyInvalidReceiptFailsBeforeUpdate|TestReceiptPrivateSchemaControls|TestReceiptSchemaTwoInventoryValidation)$' -json > /tmp/machinery-MAC-2u36-red.qRk5DC/ac2-private-focused.jsonl 2>&1
2. cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-2u36 && go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json > /tmp/machinery-MAC-2u36-red.qRk5DC/496963f-final-matrix.jsonl 2>&1
3. pvg verify internal/install/receipt_test.go internal/install/bootstrap_receipt_test.go --format text --include-tests
4. pvg story verify-tdd --base f24b2df
5. git diff f24b2df --numstat; git status --short; shasum -a256 both test files, frozen install_test.go, both raw logs and installed NIL binary (all read-only in the explicit story worktree).
6. Raw JSON parsing enumerated every run/terminal event and compared the entire failure-name set with preserved98d3b57 raw output. No raw log was reconstructed from tool display.

Summary:
Focused:23 named leaves23PASS0FAIL0SKIP,0.461s, exit0. It ran only after correcting an uncommitted extra-closing-brace syntax typo caught by gofmt; that initial formatting failure ran no tests and is not RED evidence. Stable tested bytes were then committed unchanged as496963f.
ONE complete latest-SHA broad replay after focus stabilized:79 leaves62PASS17FAIL0SKIP,739.525s, exit1 intended RED. Every run event has a terminal outcome. The exact failure set (including parent test outcomes) equals98d3b57; no new product/fixture/compiler/timeout failure. All23 repaired/new receipt-validation leavesPASS in this same broad invocation. Frozen bootstrap suite remains43leaves26PASS17FAIL; remaining selected36leavesallPASS.
VERIFY PASSED2files0issues. Hard-TDD PASS10commits checked,0merges skipped,no unauthorized test edits. Worktree clean.
Coverage:100% of7ACs represented by authored outcome contracts; statement/branch coverage was not instrumented and no numeric code-coverage claim is made. The focused cases are real-filesystem loader/unit proof, not additional CLI integration. Frozen bootstrap fixtures continue to exercise actual built CLI/binaries/checksummed source and real children. Actual native darwin/arm64 execution has zero skips; no new skip/env gate or runtime dependency added. No Linux/Windows runtime result, full package/race suite or final scripts/preflight.sh result is claimed. No timeout was increased.

### Commit, frozen bytes and raw artifacts
Branch story/MAC-2u36; RED rework SHA496963fb7f4842d706d308dafcbd145908a6e395; previous delivered98d3b573ac493c4d7252bffb03c83f92dc99f82b and all earlier commits retained.
internal/install/receipt_test.go SHA256 a56a3dc07bd447a0e4d4469f0d027509e5149eb26b494d118d394089cbfb5c48.
FROZEN internal/install/bootstrap_receipt_test.go SHA256 ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23 (unchanged).
FROZEN internal/install/install_test.go SHA256 d5039079dd032f8b86f86ad6bd0ac411b192983b10ee194e599d6e59bcba97ba (shared write helper unchanged).
Final full raw JSON: /tmp/machinery-MAC-2u36-red.qRk5DC/496963f-final-matrix.jsonl, SHA256 a42f96c5d36651fbd835d9626057eb3f659ae6b2503668d4dd6375407dce7b03.
Focused raw JSON: /tmp/machinery-MAC-2u36-red.qRk5DC/ac2-private-focused.jsonl, SHA25649ca8ce18f9b63d16c4be5ee3ad8f00e9ddfb1fb3785b3510a9611fe5daeae9c.
Complete79-leaf inventory: /tmp/machinery-MAC-2u36-red.qRk5DC/496963f-leaf-inventory.md, reproduced below.
Both raw files directly capture native stdout/stderr completely; final content parses successfully. No further broad run was performed in this rework.

### Exact authorized repair and budget
Only three existing receipt_test.go functions were amended in place, retaining names and all intended negatives: TestCorruptReceiptFailsLoudly, TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology, TestSemanticallyInvalidReceiptFailsBeforeUpdate. Added necessary errors/slices imports, three receipt-local helpers, and TestReceiptPrivateSchemaControls/TestReceiptSchemaTwoInventoryValidation. No other existing assertion, shared fixture or production file changed.
Actual receipt-test delta +186/-25 (211 changed lines; net161), plus frozen1111-line bootstrap file. Overall repository delta from epic:1297insertions25deletions across2test files,1322changed test lines. Adding the estimated350future production lines yields1672, within approximately1700 reviewed combined forecast. The receipt delta is slightly above the150-200 estimate because it includes actual config/file mode/type assertions, matched-valid reads, exact wrapper removal, typed errors, real six-root digest fixture and normalization compatibility; no unrelated scope added.
No new production parser bypass was found. No source/API/hook, instrumented child, fake runner/output, env-only bypass or regrouping implementation was introduced.

### AC2 diagnostic attribution and valid controls
Every negative first loads an otherwise valid matched receipt through the SAME loader and read path. The receipt-local helper verifies an actual0700 directory and an existing0600 regular file before each load. Each negative requires exists=true and nonnil error, then removes the exact known parse/invalid installation-receipt path wrapper before checking diagnostic details. Test directory names cannot satisfy the assertion; permission/path errors have the wrong wrapper and fail it.
- Malformed original {not-json reaches typed json.SyntaxError; actual diagnostic invalid character in literal null. It no longer passes on0644 permission rejection.
- Unknown root/home/target fields reach exact unknown-field names home_installz/copies/copied. The unknown-home matched fixture has a nonempty safe absolute home and loads successfully without the unknown field.
- Duplicate root/nested fields reach duplicate JSON field schema_version/target.
- Wrong homes/copy JSON types reach typed json.UnmarshalTypeError with Value string, correct field suffix and expected []string/bool type.
- Trailing JSON reaches trailing JSON value.
- Semantic cursor target reaches invalid-receipt unknown target "cursor". This is loader rejection ahead of planning, NOT observed binary-replacement ordering; the retained test name is now explained accurately.
- Matched named schema1/schema2 controls use actual seeded topology and verify exact normalized loaded metadata.
- Schema2 fixture independently enumerates all6artifact roots for2real homes (skill tree and both shipped role docs per home) with actual artifactTreeDigest values and valid copy/plugin topology. Missing/extra entries require exact5/7-versus6 inventory-count detail; duplicate/substituted/relative/non-clean/unexpected-absolute paths at unchanged count require inventory-path mismatch detail; digest prefix/length/correct-length nonhex mutations require malformed-digest detail for the intended artifact. Only the intended category is changed per case.
- Reversed valid home-group/plugin/artifact input order is accepted and normalized to the original complete expected receipt. It is not invented as invalid.
The full log records20intended negative diagnostics and3named positive controls; all23PASS. These are safety controls, not manufactured behavioral RED.

### Independent permission proof and preserved real behavior
Frozen TestBootstrapReceiptCLI/unsafe_receipt at bootstrap_receipt_test.go584-587 changes only os.Chmod(path,0666) after a valid0600 receipt was successfully recorded. Its complete-state/nonzero rejection remains unchanged andPASS13.47s. The bootstrap corrupt-receipt case overwrites an existing0600 file and alsoPASS; it never shared the old fresh0644 fixture bug.
Actual ordinary complete convergencePASS51.37s and edited-file repairPASS49.65s with exact binary/current source content, full topology/modes/inventory/digests/plugin obligations/unrelated preservation and same-release idempotence.
Actual ordinary later OpenCode-source failure after binary/home/Codex mutation rolls the entire pre-state back:PASS32.60s. Real interrupted held-download startup recoveryPASS14.67s is explicitly PRE-mutation interruption proof only.
All4real authority/cleanup controlsPASS. Four parent-finalization leaves remain premature-child-receipt/missing-parent-publication RED; NO real final-close fault was injected. Strict scratch/coverage/content/count/error/rollback/cleanup/lock assertions remain frozen and mandatory for GREEN. Missing-native early inventory failures do not count as later-target or final-close boundary execution.
Three real standalone supported-recording controlsPASS; two real overlap/next-native flows remain desired conflict/restoration failures with no manual repair. Exact prepared scope and concurrent foreign-postimage protections remain unchanged.

### Unchanged17 product RED leaves
1complete-recorded-plan unit;2bootstrap intact/edited convergence;4single/multiple missing-native repair success requirements;1intact bootstrap later-source requirement;2missing-prestate later-source boundary requirements;1unsafe native-symlink bootstrap;4parent-finalization present/absent no-fault/fault requirements;2standalone cross-group conflict/next-native flows. All failure identities equal the preserved prior full replay. No AC2 safety control was artificially made to fail.

### AC Verification
|AC|Current proof|
|---|---|
|1|Frozen complete recorded-plan parity/default/plugin/schema1/selector and real convergence tests replayed; desired bootstrap defects remainRED, positive controlsPASS.|
|2|Ten old false-positive tests now genuinely reach typed/exact parser or semantic diagnostics after matched private reads;13additional schema/inventory/digest/order leavesPASS. Frozen mode-only CLI permission and other safety controls independently replayed.|
|3|Frozen actual release content/binary/mode/topology/plugin/digest/idempotence proof; ordinary intact/editedPASS, bootstrap and missing-repair defectsRED.|
|4|Frozen real ordinary post-mutation rollbackPASS32.60s; missing/bootstrap later-boundary defectsRED; pre-mutation interrupted recovery separatelyPASS14.67s.|
|5|Frozen actual built CLI/checksummed loopback release/source/unmodified children/temp destinations; added metadata tests use real filesystem only. No mocks/skips/live installation writes.|
|6|Frozen parent-process sequencing/fault contract remainsRED on missing publication, not claimed fault execution;4real authority/cleanup controlsPASS.|
|7|Frozen actual conflict/next-native desired outcomes remainRED; fresh/disjoint/same-ordered-group-copy controlsPASS with independent complete content/topology checks.|

### Historical proof correction and retained evidence
The formal independent PM rejection at2026-09-05T23:04:15Z is accepted. Ten old passes in author98d3b57 (66leaves49PASS17FAIL0SKIP748.254s) and independent replay (same outcomes758.895s) stopped at unsafe0644permissions, not their named parser/topology reason. Those logs remain accurate runtime history, but those ten passes are NOT valid old AC2 proof. Prior blanket claims of complete attributed AC2 coverage are superseded by this correction. The digest-computation test alone likewise did not establish malformed schema2 receipt metadata rejection.
Current repaired functions retain the same10leaf identities, with genuine private matched-valid/precise-negative proof. Thirteen added leaves supply supported schema controls, actual schema2 inventory/digest encoding and valid normalization proof. No fourth production defect or parser bypass is claimed.
All prior commits/raw logs, the formal PM report /tmp/machinery-pm-MAC-2u36.DDIEBx/REVIEW.md and prior oracle/fixture disputes remain intact. The prior delivery was formally rejected and root atomically reclaimed this rework; the canonical deliver transition below is one new delivery for THIS rework, not a repeated non-idempotent call within the prior attempt.

LEARNINGS:
- A negative err!=nil assertion can silently prove an earlier permission check instead of its advertised validation boundary; inspect reached diagnostics, not test names.
- Matched valid/private controls and actual mode/type checks make input-validation tests attributable; typed errors and exact wrapper removal prevent path-name false positives.
- Schema2 inventory/digest encoding needs direct receipt-input mutations, distinct from digest-function content-change tests; reversed valid order is compatibility.
- Preserve independent unchanged CLI permission/rollback proof rather than modifying a shared write helper or duplicating expensive release fixtures.
- Separate historical runtime pass counts from valid coverage claims; this rework fixes proof attribution while retaining all17genuine product RED failures.

### Containment and remaining gates
Installed /Users/ramirosalas/.local/bin/machinery SHA256 remains5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849. No installed skill/plugin/agent/config, user Dagger resource, root branch, remote resource or unrelated edit touched. No full preflight, push/sync/setup/recover/dev-link/live update, product implementation or subagent used.
Independent full RED re-review/approval and GREEN remain pending. Actual final-close fault/rollback proof will only be claimable when the real parent publication operation is reached. Worktree retained clean with exact frozen inputs; final shell restored to repository root.

### Complete named leaf inventory
|Leaf|Outcome|Seconds|
|---|---|---|
|TestBootstrapReceiptRootAlias/root|PASS|0|
|TestBootstrapReceiptRootAlias/descendant|PASS|0|
|TestBootstrapReceiptRootAlias/canonical_root|PASS|0|
|TestBootstrapReceiptRootAlias/canonical_descendant|PASS|0|
|TestBootstrapReceiptRootAlias/similar_sibling|PASS|0|
|TestBootstrapReceiptRootAlias/interior_alias|PASS|0|
|TestBootstrapReceiptRootAlias/repeated_suffix|PASS|0|
|TestBootstrapReceiptRootAlias/sibling_journal|PASS|0|
|TestBootstrapReceiptPlan/first_bootstrap_defaults_control|PASS|0|
|TestBootstrapReceiptPlan/supported_schema_one_control|PASS|0|
|TestBootstrapReceiptPlan/complete_recorded_mixed_plan|FAIL|9.98|
|TestBootstrapReceiptCLI/standalone_receipt/supported_recording/fresh_mixed_groups|PASS|0.02|
|TestBootstrapReceiptCLI/standalone_receipt/supported_recording/disjoint_additional_group|PASS|2.85|
|TestBootstrapReceiptCLI/standalone_receipt/supported_recording/same_ordered_group_copy_change|PASS|3.94|
|TestBootstrapReceiptCLI/standalone_receipt/reject_cross_group_repeated_false_then_native|FAIL|27.45|
|TestBootstrapReceiptCLI/standalone_receipt/reject_cross_group_repeated_true_then_native|FAIL|24.98|
|TestBootstrapReceiptCLI/parent_finalization/absent_false_close_fault_false|FAIL|31.07|
|TestBootstrapReceiptCLI/parent_finalization/absent_false_close_fault_true|FAIL|30.86|
|TestBootstrapReceiptCLI/parent_finalization/absent_true_close_fault_false|FAIL|29.53|
|TestBootstrapReceiptCLI/parent_finalization/absent_true_close_fault_true|FAIL|30.61|
|TestBootstrapReceiptCLI/receipt_authority_absent|PASS|14.22|
|TestBootstrapReceiptCLI/receipt_authority_forged|PASS|3.87|
|TestBootstrapReceiptCLI/receipt_authority_unprepared|PASS|3.83|
|TestBootstrapReceiptCLI/receipt_authority_out_of_scope|PASS|3.99|
|TestBootstrapReceiptCLI/converges_bootstrap_false|PASS|51.37|
|TestBootstrapReceiptCLI/converges_bootstrap_true|FAIL|27.52|
|TestBootstrapReceiptCLI/repair_edited_owned_artifact_bootstrap_false|PASS|49.65|
|TestBootstrapReceiptCLI/repair_edited_owned_artifact_bootstrap_true|FAIL|27.5|
|TestBootstrapReceiptCLI/repair_missing_owned_artifact_bootstrap_false|FAIL|30.33|
|TestBootstrapReceiptCLI/repair_missing_owned_artifact_bootstrap_true|FAIL|21.78|
|TestBootstrapReceiptCLI/repair_both_native_groups_missing_bootstrap_false|FAIL|28.91|
|TestBootstrapReceiptCLI/repair_both_native_groups_missing_bootstrap_true|FAIL|21.03|
|TestBootstrapReceiptCLI/later_target_failure_rolls_back_bootstrap_false|PASS|32.6|
|TestBootstrapReceiptCLI/later_target_failure_rolls_back_bootstrap_true|FAIL|18.45|
|TestBootstrapReceiptCLI/missing_prestate_later_target_failure_rolls_back_bootstrap_false|FAIL|25.68|
|TestBootstrapReceiptCLI/missing_prestate_later_target_failure_rolls_back_bootstrap_true|FAIL|19.49|
|TestBootstrapReceiptCLI/corrupt_receipt|PASS|12.57|
|TestBootstrapReceiptCLI/unsafe_receipt|PASS|13.47|
|TestBootstrapReceiptCLI/stale_schema|PASS|13.33|
|TestBootstrapReceiptCLI/plugin_discovery_failure|PASS|13.08|
|TestBootstrapReceiptCLI/symlink_native_artifact|FAIL|19.73|
|TestBootstrapReceiptCLI/non-directory_native_parent|PASS|11.43|
|TestBootstrapReceiptCLI/interrupted_download_recovers_real_transaction|PASS|14.67|
|TestReceiptArtifactDigestIgnoresInstallTimeButDetectsModeAndContent|PASS|2.53|
|TestUninstallDeletionFailureRollsBackArtifactsAndReceipt|PASS|2.46|
|TestAbandonedLegacyReceiptLockDirectoryDoesNotBlock|PASS|0.25|
|TestLoadReceiptRejectsFIFOWithoutOpening|PASS|0|
|TestLoadReceiptRejectsNonPrivateLeafConfigDirectory|PASS|0|
|TestReceiptReadModifyWriteIsSerialized|PASS|5.86|
|TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap/symlink|PASS|0|
|TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap/oversize|PASS|0|
|TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap/config_directory_swap|PASS|0|
|TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap/entry_swap|PASS|0|
|TestForgetReceiptUsesNativeCaseAliasIdentity|PASS|0.48|
|TestCorruptReceiptFailsLoudly|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/unknown_root|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/unknown_home_field|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/unknown_target_field|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/duplicate_root|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/duplicate_nested|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/wrong_homes_type|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/wrong_copy_type|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/trailing_value|PASS|0|
|TestSemanticallyInvalidReceiptFailsBeforeUpdate|PASS|0|
|TestReceiptPrivateSchemaControls/schema_1|PASS|0.01|
|TestReceiptPrivateSchemaControls/schema_2|PASS|0.01|
|TestReceiptSchemaTwoInventoryValidation/missing_entry|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/extra_entry|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/duplicate_path|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/substituted_path|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/relative_path|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/non-clean_path|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/unexpected_absolute_path|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/digest_prefix|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/digest_length|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/digest_nonhex|PASS|0|
|TestReceiptSchemaTwoInventoryValidation/valid_reversed_ordering|PASS|0|
|TestUpdateRollsBackAllHomesBinaryAndReceiptOnLaterFailure|PASS|2.03|
|TestBootstrapDefaultPlanIsPluginAware|PASS|0.06|


## nd_contract
status: rejected

### evidence
- PM rejection applied via pvg story reject on 2026-09-05.

### proof
- [ ] Story requires another developer delivery before it can be accepted.


## AUTHORITATIVE AC2 PROOF-REPAIR SCOPE — MAC-2u36 — 2026-09-05

Scope-only Sr PM disposition: keep this under existing P0 MAC-2u36/AC2. All seven product ACs are preserved verbatim. No new issue, dependency, source ownership, production bug claim or status/label/claim transition. Independent PM is completing its RED verdict and retains authority for rejection/approval and explicit test-edit authorization. This note does not replace its delivery/review nd_contract.

Verified source chain: install_test.go write uses os.WriteFile0644; the three receipt_test.go functions named in canonical AC2 proof repair use it on fresh receipts. readReceiptFile verifies private regular-file permissions before reading JSON; Unix privateFilePermissionsOK is mode&0077==0. On the reviewed umask022, the one corrupt/eight parser/one semantic leaves can pass on permissions failure alone. The source-established evidence gap does not establish a production parser bypass. Existing bootstrap corrupt receipt preserves prior0600 mode; its unsafe receipt case explicitly changes a valid receipt to0666 and is legitimate distinct permission evidence.

Required narrow repair after explicit independent PM authorization: amend those three receipt tests and minimal receipt-local fixture/assertion code, leaving shared write unchanged. Same-path private0700/0600 setup and matched valid schema1/schema2 positives must reach loadReceipt successfully. Each malformed/unknown/duplicate/trailing/wrong-type/semantic case must reach its intended detail category, protected against pathname false positives. Add actual-file schema2 missing/extra inventory, same-count duplicate/substituted/invalid paths and digest prefix/length/nonhex cases, each based on valid real topology/digests. Preserve normalization of valid ordering. Reuse the existing actual CLI unsafe-receipt leaf unchanged, verifying its isolated0600-to0666 mutation of valid bytes; bootstrap98d3b57 stays frozen. Do not add expensive CLI tests for each loader mutation or weaken any safety checks.

Budget now records measured1111 current test LOC plus estimated150-200 focused repair/case LOC and prior350 production allowance, approximately1700 combined/4-6files. The older1400 forecast is superseded, with actual changed-LOC/helper/time reporting and PM overrun investigation required; neither old nor new estimate permits dropping required proof. Existing15m full command and individual bounds are unchanged. Run focused repaired tests first, then one stable/frozen full latest-SHA raw replay; no additional costly broad run during this triage.

Evidence classification: author98d3b57 full replay66leaves49PASS17FAIL0SKIP748.254s remains historical raw evidence. The ten permissions-masked passes cannot be advertised as parser/topology coverage. Repaired validators may pass unchanged production and are legitimate safety controls; do not manufacture RED or interpret a passing check count as proof of its advertised cause. Independent PM subsequently reports its replay completed with matching66leaves49PASS17FAIL0SKIP758.895s; its formal verdict and exact authorization remain PM-owned and are not preempted here.

Source discovery used graph Verify project Users-ramirosalas-workspace-machinery generation2026-09-05T20:28:41Z; receipt.go/receipt_test.go/install_test.go/durability_unix.go metadata_match/no_recorded_issue, best-effort only. Bootstrap candidate is missing from root graph and was inspected through git show98d3b57. Direct bounded test-source scan found no dedicated schema2 malformed-inventory/digest category tests in the selected source; no broader exhaustive coverage claim. No runtime test, source/test/worktree edit, installed binary/plugin/skill/agent change, preflight or remote operation performed.

Self-review verdict: clean after narrowing scope to the three defective existing tests and reusing the existing real permission case. All prior parent-process/close-seam/AC7 authorizations remain intact; the new amendments need their own explicit TEST-EDIT AUTHORIZED from independent PM before use.


## Historical canonical description before AC2 proof repair
Preserved as quoted history; current Description is authoritative.

> ## Description
> ## USER INTENT
> A user rerunning the installer or ordinary machinery update must get a complete, safely repaired installation at the requested release across all recorded placements, including a missing owned native artifact. Success must mean current content, complete receipt evidence and preserved topology; failure must preserve recoverable pre-run state.
> 
> ## Context (Embedded)
> Three production defects share this receipt-publication and recorded-plan boundary:
> - NEXT1: install.sh passes --bootstrap-defaults, but updatePlan chooses default homes before the complete recorded home/native plan.
> - Newly reproduced in MAC-2u36: ordinary update retains the missing Codex target in its plan, but refreshDirectInstalls runs home groups first; each child install calls recordHomeInstallLocked -> saveReceipt -> refreshReceiptArtifacts over ALL recorded placements. A missing later Codex file aborts the first child's receipt inventory before native repair is reached.
> Actual isolated CLI diagnostic on frozen v1 99956740ae5559262790a9473b5597c1775928f2 failed in 35.820s after verified binary replacement and first home-group mutation: "inventory installed artifacts for receipt: digest .../.codex/agents/machinery-fsm-author.toml: lstat ...: no such file or directory". It did not assert final rollback state. This disproves the earlier assumption that the exact missing-file ordinary run would already pass; it does NOT change the required successful repair behavior.
> The existing ordinary intact convergence and edited-regular-file repair controls pass. Metadata validity, path/type safety and plugin ownership are distinct from repairable installed content drift or absence.
> - Newly reproduced standalone writer defect: an existing receipt records [custom-a, custom-b] Copy:false and [copy-a, copy-b] Copy:true, plus Codex Copy:false/OpenCode Copy:true. An explicit all-copy install of [custom-a, custom-b, copy-a, copy-b] returns success, but the next native install rejects the emitted receipt with "home install paths overlap or repeat". recordHomeInstallLocked replaces only a matching first/canonical home group, leaving the second overlapping group; normalizeReceipt sorts but does not reconcile groups, and saveReceipt currently publishes without validateReceipt. Actual built-CLI evidence is preserved at /tmp/machinery-MAC-2u36-red.qRk5DC/e1fd0a3-parent-finalization.jsonl (48.882s). This is product corruption of its own receipt, separate from the legitimate absent-reference fixture repair at d9d53b980a393c527eaf856b0bba5d8bc70867ec.
> 
> ## Scope decision and Ownership
> This existing P0 bug owns all three defects: the added standalone writer defect violates the already-owned validated-publication guarantee in AC6 and touches the same receipt.go/install.go seams. A sibling would duplicate that ownership. No separate issue or artificial dependency is created; bounded AC7 specifies the writer correctness guarantee without adding automatic regrouping semantics.
> Production ownership expands explicitly from internal/install/update.go to:
> - internal/install/update.go: receipt-aware bootstrap planning, full-plan refresh coordination and final commit/publication sequencing.
> - internal/install/receipt.go: bounded receipt accumulation/finalization, validation before publication of the final complete candidate receipt, and rejection of conflicting standalone home-group recording; preserve normal receipt validation/persistence.
> - internal/install/install.go: child placement/receipt interaction needed for authenticated parent-owned full-plan refresh; preserve normal standalone recording and transactional restoration when conflicting recording is rejected.
> Tests: internal/install/bootstrap_receipt_test.go; directly associated regression additions in internal/install/receipt_test.go and internal/install/install_test.go only when required by the changed receipt/standalone-install boundary. Frozen existing assertions cannot be changed without explicit independent reviewer authorization.
> Read-only consumers unless a concrete need is separately reviewed: internal/install/targets.go, internal/install/transaction.go, internal/install/lock.go, cmd/machinery/install.go and cmd/machinery/update.go. No blanket CLI/target/lock/journal rewrite is included. You are not alone in the codebase; preserve others' edits and coordinate shared paths with dispatcher.
> 
> ## Boundary Map
> PRODUCES:
> - internal/install/update.go -> complete recorded-plan convergence, including safely missing owned native artifacts, under the existing update transaction.
> - internal/install/receipt.go -> validated complete receipt publication coordinated with successful full direct refresh.
> - internal/install/install.go -> bounded child-refresh participation while ordinary install continues to persist validated receipts.
> - internal/install/bootstrap_receipt_test.go -> real CLI RED/GREEN matrix for recorded-plan, receipt-publication and standalone overlap defects with preserved controls.
> - internal/install/receipt_test.go -> focused receipt-publication regression proof if needed.
> - internal/install/install_test.go -> focused standalone-install regression proof if needed.
> CONSUMES:
> - Existing internal/install/update.go.
>   spec: updatePlan(opts UpdateOptions) (refreshPlan, error); refreshDirectInstalls(binary, source string, plan refreshPlan, run commandRunner, out io.Writer) error.
> - Existing internal/install/receipt.go.
>   spec: loadReceipt() (installReceipt, bool, error); saveReceipt(receipt installReceipt) (retErr error); refreshReceiptArtifacts(receipt *installReceipt) error; recordHomeInstallLocked(homes []string, copyAll bool) error; recordTargetInstallLocked(names []string, copyAll bool) error; validateReceipt(receipt installReceipt) error; normalizeReceipt(receipt *installReceipt).
> - Existing internal/install/install.go.
>   spec: Install(opts Options) error; installLocked(opts Options) (retErr error).
>   fields: Options.Homes []string; Targets []string; From string; Copy bool; Record bool. Current CLI passes Record: true.
> - Existing internal/install/transaction.go.
>   spec: beginArtifactTransaction(paths []string) (*artifactTransaction, error); delegatedInstallOperation() bool; delegatedArtifactTransaction(paths []string) (*artifactTransaction, error).
>   source: authenticated delegated children require the parent's prepared journal and coverage of their exact artifact paths; parent transaction snapshots binary, direct placements and receipt, and owns final commit/rollback.
> - Existing internal/install/targets.go.
>   spec: installTargets(names []string, src string, copyAll bool, out io.Writer, before func(string) error) error.
>   source: validates target source before placement; receipt recording is in installLocked, not this function.
> 
> ### Story Acceptance Criteria
> 1. Valid supported receipt plus --bootstrap-defaults uses the same complete recorded home/native/plugin plan and relevant discovery as ordinary machinery update without selectors. Preserve per-group copy/symlink modes. No-receipt first bootstrap retains plugin-aware defaults; explicit homes/targets remain incompatible with bootstrap.
> 2. Fail closed with actionable diagnostics and no silent defaults for malformed/duplicate/unknown/trailing JSON, unsupported schemas, invalid topology/inventory/digest encoding, unsafe receipt permissions/type, unsafe artifact or parent path/type substitutions, and uncertain plugin ownership, including --skip-plugins. Supported schema 1 is not rejected merely because it is older. Valid receipt plus edited/missing owned regular artifact with unchanged safe parents/topology is repairable; absence alone must not be classified as unsafe ownership. Existing safety validation must not be relaxed.
> 3. Actual ordinary and bootstrap full-plan update succeeds for intact installs, an edited regular owned Codex artifact, and a missing regular owned native artifact, restoring exact current-release content and binary, complete receipt topology/digests/plugin obligations, both mixed-mode home groups and Codex/OpenCode targets, unrelated-file preservation and same-release idempotence. Exact missing-file fixture: remove only temporary HOME/.codex/agents/machinery-fsm-author.toml after valid receipt recording; no unsafe parent/type substitution. Test both native copy-mode groups missing an owned file in the same run as a bounded guard against a fix that merely repairs one hardcoded target first. During RED the ordinary missing-file case is EXPECTED TO FAIL behaviorally on unchanged production; intact ordinary and edited-file ordinary cases remain passing controls.
> 4. Retain actual later-target failure after binary, home and earlier native mutations with complete pre-run restoration of binary, homes/native artifacts, receipt and unrelated sentinels. Also exercise that rollback with a safely missing owned artifact in pre-state: absence must be restored as absence, not replaced by a half-repaired result. Retain real interrupted journal recovery at CLI startup and explicitly separate its pre-mutation interruption boundary from post-mutation rollback proof. Preserve independent malformed/schema/path/type/plugin-ownership negatives and mixed-copy-mode checks.
> 5. Integration uses actual built release/CLI install/update subprocesses, real checksummed archives and temporary home/config/target/binary roots. No mocks, stubs, skip-if-missing or env-gated omissions. No live installation mutations. Machinery product behavior/tests remain standalone with no Paivot/pvg/nd, label or commit-convention dependency.
> 6. A full direct-refresh operation must not require complete final artifact inventory before remaining selected placements can repair safely missing owned files. The Update parent PROCESS that holds the operation lock and owns the prepared transaction must perform final complete receipt inventory/publication itself, after every authenticated selected placement child has successfully returned and all direct placement/source checks have succeeded, and before that parent's direct transaction commit. During the direct-refresh phase, delegated placement children do not publish intermediate or final persisted receipts: the receipt's pre-run bytes, existence, file type and mode remain unchanged until parent finalization (if no receipt existed before the run, it remains absent until parent finalization). In-memory or journal-owned private coordination may accumulate selected topology within existing authority; it must not advertise an intermediate installation as final. The parent publishes a complete normalized receipt with real refreshed digests for the full recorded/selected plan, preserving recorded topology/copy modes/plugin obligations and existing supported-schema validation. Do not drop missing entries, retain fabricated/stale digest evidence, invent success for an incomplete plan, or bypass final validation. Any final inventory/publication failure before direct commit fails the operation and rolls back the whole direct update when transaction-written post-images are unchanged; preserve the existing refusal to overwrite concurrent foreign changes and its recoverable diagnostic/journal behavior. Standalone nondelegated install still records validated successful topology. Any receipt deferral/coordination remains restricted to authenticated parent-owned prepared journal scope; forged/absent parent authority, unprepared journal and out-of-scope paths remain rejected. No public or env-only skip-receipt switch. Host-managed plugin refresh retains its existing post-direct-commit ownership/obligation semantics, with no new atomic host-plugin rollback or schema-migration claim.
> 
> 7. Every successful recording operation must publish a complete candidate receipt that passes the product's existing supported-schema/topology/inventory validation on the next load. Validate final normalized real inventory before receipt publication, for standalone recording as well as parent finalization; normalization alone is not validation. For the reproduced cross-group home request, and other requests whose candidate under existing same-canonical replacement semantics leaves repeated/nested homes in another recorded group, standalone install must return a nonzero actionable conflict diagnostic identifying the conflicting paths and advising a nonconflicting recorded-group retry or explicit topology reconfiguration. It must preserve the exact prior valid receipt and installation (content, symlink destinations, file types/modes and unrelated state), rolling back any owned placement writes when transaction post-images are unchanged. Do not silently delete conflicting receipt groups, forget still-installed ownership, rewrite unselected link/copy relationships, or report success then rely on next-operation validation. Preserve refusal to overwrite concurrent foreign changes and existing recoverable journal/diagnostic semantics. Fresh installs, disjoint additional groups and same recorded-group reruns (including a supported copy-mode change within that same ordered home set) remain successful and record exact valid topology/real digests; native installs still work after a rejected conflict. Parent AC6 process/unchanged-receipt sequencing remains mandatory. Automatic merging/splitting of overlapping recorded groups is not required: README.md promises exact topology after success and cmd/machinery/install.go defines first-home/copy placement, but neither defines cross-group ownership migration. The chosen conflict policy preserves this guarantee without inventing migration of unselected symlink dependents.
> 
> ## Testing Requirements
> - Hard TDD remains explicitly authorized. After independent PM reviews this scope/contract and authorizes the revised tests, a RED-only author revises the disputed edited/missing cases, adds the bounded multi-missing and missing-pre-state rollback proof, and any focused new boundary tests needed for AC6. No production edits before independently approved RED.
> - Preserve original candidate 99956740ae5559262790a9473b5597c1775928f2, test SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329 and /tmp/machinery-MAC-2u36-red.qRk5DC. Existing 36-leaf replay: 32 pass, 4 fail, 0 skip, 271.751s; only three original failures are valid bootstrap defect proof. The fourth stale_artifact rejection is superseded. Missing-file ordinary FAIL 35.820s is new genuine defect evidence, separate from earlier edited-file ordinary PASS 36.508s.
> - Preserve complete-recorded-plan, defaults/selectors, intact normal/bootstrap convergence, plugin obligations, real later-target rollback, interrupted recovery, and all legitimate safety negatives. Rename obsolete stale_artifact/disappeared_target rejection cases as repair-success requirements. Only specifically reviewer-authorized frozen-test changes are allowed.
> - Fresh replay: go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json. This package deadline replaces 10m: the compiled 5a94840 broad matrix took 536.345s for 49 leaves (34 pass, 15 fail, 0 skip), including four invalid early authority-fixture failures. Corrected real authority controls take approximately 60s, bringing the conservative replay budget near 596.345s before the new AC6 positive/fault real updates and ordinary scheduling variance; 600s is inadequate for the complete expanded matrix. The 900s package bound provides room for those measured/required cases. Preserve every individual operation timeout, frozen test selection and assertion; no skipped cases, retries masking hangs or relaxed operation bounds. This is not permission for a hung individual update. Keep the existing GREEN 15m package command unchanged pending actual measurement; full preflight remains final-epic-only.
> - Focused receipt/standalone-install/delegated-authority tests selected by actual test names introduced or affected. GREEN runs go test -count=1 -timeout=15m ./internal/install ./cmd/machinery for ordinary native package regression coverage. No full scripts/preflight.sh here; final epic gate owns heavy preflight.
> - Record SHA, exact command, named leaf inventory, zero skips, assertion failure causes and passing controls. Compile/import/fixture/infra/timeout errors are not behavioral RED. Unexpected missing-file receipt-inventory failure is a valid success-contract assertion failure, not an acceptable permanent rejection.
> - If internal deferred receipt behavior is introduced, add proof that normal standalone install still publishes correct receipts, missing/forged/out-of-scope parent authority cannot enable it, and final publication failure rolls back real prior mutations. Reuse existing authenticated delegation mechanisms; implementation details require evidence, not an assumed new public API.
> - Finalization boundary proof is mandatory: use real Update with its default command runner, actual checksummed built release and real placement children, plus a successful no-fault counterpart. Independently observe current release binary/home/native content, full prepared-journal path coverage and unchanged persisted receipt through the final completed child. Then the parent's real final receipt scratch-file persistence must encounter a deterministic failure before receipt rename/commit. Use the EXISTING process-local closeInstallFile seam only after independent PM approves the concrete test: narrowly target the final receipt-* scratch file in the current prepared journal, actually close the real file and surface its OS close failure, prove the hook fired on that exact final publication operation, and preserve normal file behavior otherwise. Do not substitute a fabricated command result, new production hook/API, modified child executable or user/env-only bypass.
> - Required late-failure assertions: no direct commit; exact binary/home/native/receipt pre-state restored, including absence where applicable; unrelated sentinels preserved; journal cleanup and reusable/released operation lock. The no-fault counterpart must publish complete current digests/topology and commit successfully. Compile/fixture errors, early missing-file inventory, or a never-fired hook are not this boundary proof. Existing concurrent-change rejection remains a distinct safety case; deleting a live updated artifact with os.Remove or durableRemove after a child is not a receipt-publication fault and cannot be used to demand unsafe overwrite during rollback.
> - This parent-process/unchanged-receipt refinement and any proposed boundary test must receive independent PM review and explicit test-edit authorization before the RED author uses it. Existing authorized matrix work retains its prior authorization; no source edits are authorized during RED. Parent-finalization ownership remains unchanged; the current combined estimate is stated under DIFF BUDGET below, and the paused author resumes only after the pending independent scope/oracle review.
> - AC7 regression after independent PM test-edit authorization: use an actual built temporary CLI and real local source/roots, seed the exact two mixed home groups plus the two native targets above through successful standalone installs, snapshot the complete real state and receipt, then issue install --from SOURCE --copy --home HOME/custom-a --home HOME/custom-b --home HOME/copy-a --home HOME/copy-b. Assert the intended overlap diagnostic/nonzero result and complete exact restoration, then run a real standalone native install --from SOURCE --target codex --target opencode --copy and require success with valid complete topology/inventory/digests. The original bug must fail on its false-success/invalid-receipt behavior, not a fixture error. Add bounded positive controls for a fresh/disjoint group and a same-ordered-group rerun with copy-mode change, checking the other group's ownership and content remain correct. Reuse the built CLI/fixture and existing snapshots; retain every safety assertion. Place additions in bootstrap_receipt_test.go or the already-owned install_test.go/receipt_test.go, selected by names containing Receipt or explicitly included in the focused command. No new API, mock runner, runtime service or production edit during RED. Runtime/package deadline changes require measurement and independent review; the existing 15m broad replay remains unchanged.
> - The two paused d9d53b9 boundary-oracle defects (non-prefix macOS path normalization and desired digests computed only inside the close observer) are test defects, not extra product scope. Independent PM must review and explicitly authorize any repair and the concrete AC7 tests before RED resumes; this triage does not authorize test edits or supersede the 21:42:11Z closeInstallFile authorization.
> - Separate independent PM replays revised RED before approval/freezing. GREEN preserves exact approved test/fixture/config bytes; any later repair requires explicit reviewer authorization and re-RED.
> 
> ## OUT OF SCOPE
> - Automatic cross-group merge/split/recanonicalization: no supported ownership-migration contract has been established; AC7 requires safe conflict rejection, while any future regrouping feature needs a separately reviewed contract. Do not misclassify validation before publishing this story's own receipt as out of scope.
> - Arbitrary missing-root recreation, changed unsafe parents, and ownership uncertainty: this story preserves their safety boundary; it does not authorize blind repair.
> - Redesign of host-managed plugin transactions, lock/journal format or public install selectors: outside the direct receipt-publication defect; report a concrete need for review before expanding.
> - Other assessment subsystems stay in sibling stories. No pushes, sync, remote mutation, installed binary/plugin/skill/agent replacement, dev-link or healthy worktree cleanup. NIL uses the installed product.
> - Final heavy preflight, main merge and isolated release candidate belong to epic gate MAC-ou97.
> 
> ## DIFF BUDGET
> Expected 4-6 files, under 1400 total changed LOC: approximately 1050 test LOC and 350 production LOC across the same three named source files. Investigation: the retained d9d53b9 candidate already has 903 test LOC, including about 194 lines for the independently required four parent-finalization no-fault/close-fault present/absent cases; the former 750-test/1100-total estimate leaves only197 production LOC and is disproven. Allocate about147 additional test LOC for bounded AC7 real-CLI overlap/rollback/next-operation and supported-rerun controls plus the independently reviewed oracle repairs, retaining the prior350 production allowance and modest rounding headroom. This is a realistic estimate for the current verified scope, not permission to trim required safety proof or expand regrouping. Independent PM must inspect the concrete added-test/helper cost before authorization; actual material overrun or another production file requires explicit scope review.
> 
> ## Dependencies
> Parent MAC-ui8a; remains blocking MAC-gcrr and MAC-ou97. No new inter-story dependency: one developer owns all three coupled installer defects and one independent PM reviews the full outcome. No other sibling implementation declares the three owned installer source seams. Consumer guidance and epic acceptance remain blocked until this story is accepted.
> 
> ## MANDATORY SKILLS
> - developer for RED/GREEN roles; codebase-memory for source discovery; pm_acceptor for independent scope/test authorization, RED replay and acceptance.
> 
> ## Delivery Requirements
> This AC7 overlap repair, revised measured budget and concrete new tests require independent PM review before author re-RED; no prior authorization is silently broadened. Preserve all existing authorization history, including parent-process finalization and the exact existing closeInstallFile fault/control authorization at21:42:11Z. The paused oracle repairs require their own explicit reviewer authorization. Triage grants no test edits, production edits, RED approval, delivery or status/claim transition. Do not deliver or approve v1 unchanged.
> Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands/results, inventory, changed-file rationale, per-AC proof and residual limits. Product has no dependency on these development coordination tools.


## RED delivery readback verification
After the append-only terminal delivered comment, pvg story verify-delivery MAC-2u36 returned Passed:9, Failed:0; all label/authoritative-contract/EOF/evidence/commands/summary/SHA/AC checks OK. Canonical deliver was invoked once only. Frozen source SHA98d3b573ac493c4d7252bffb03c83f92dc99f82b and full raw final66leaf49PASS17FAIL0SKIP748.254s evidence remain unchanged.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## Implementation Evidence
RED ONLY — MAC-2u36. Tests frozen for independent PM review; production intentionally unchanged. This is author delivery of the seven-AC RED contract, not GREEN completion or independent RED approval.

PROOF:

### CI/Test Results
Commands run:
1. cd /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-2u36 && go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json > /tmp/machinery-MAC-2u36-red.qRk5DC/98d3b57-final-matrix.jsonl 2>&1
2. go test -count=1 -timeout=7m ./internal/install -run '^TestBootstrapReceiptCLI$/(standalone_receipt|parent_finalization)' -json > /tmp/machinery-MAC-2u36-red.qRk5DC/ac7-parent-focused.jsonl 2>&1 (same explicit worktree prefix, same frozen SHA).
3. go test -count=1 -timeout=7m ./internal/install -run '^TestBootstrapReceipt(RootAlias|CLI/(standalone_receipt|parent_finalization))' -json > /tmp/machinery-MAC-2u36-red.qRk5DC/ac7-oracles-focused.jsonl 2>&1 (same worktree/SHA). This earlier selector actually selected only the eight alias leaves; it is oracle-only proof, not CLI coverage. Corrected command2 supplies the intended CLI focus.
4. pvg verify internal/install/bootstrap_receipt_test.go --format text --include-tests
5. pvg story verify-tdd --base f24b2df
6. shasum -a 256 internal/install/bootstrap_receipt_test.go /tmp/machinery-MAC-2u36-red.qRk5DC/98d3b57-final-matrix.jsonl /Users/ramirosalas/.local/bin/machinery; git status --short (read-only).

Summary:
Final complete latest-SHA replay exit1 (intended RED), 66 named leaf outcomes: 49 PASS, 17 FAIL, 0 SKIP, 748.254 seconds. Every run event has a terminal outcome. New file contributes43leaves26PASS17FAIL; existing selected23leavesallPASS. No compiler/import/setup/runtime-infrastructure/timeout failure, missing terminal event, skip/env gate or remaining known oracle error. The raw file was captured directly from native Go stdout/stderr before tool-display truncation and parses completely. A later overlarge diagnostic-summary display was truncated; that does not affect the complete original raw artifact or counts derived from it.
Focused command2:9leaves3PASS6FAIL0SKIP232.004s. Command3:8aliasleaves8PASS0FAIL0SKIP0.409s. These are separate from final matrix.
VERIFY: PASSED (1 files scanned,0issues). Hard-TDD check:9commits checked,0merges skipped,PASS no unauthorized test edits. Worktree clean.
Coverage: 100% of seven ACs represented (7/7 authored outcome contracts); statement/branch coverage percentage was not instrumented and is not claimed. Native darwin/arm64 actual CLI lane; no Docker/Java/Node runtime dependency in tests. No full install/cmd package regression suite, race suite or scripts/preflight.sh was run here; those belong to GREEN/final epic gates. Actual targeted runtime748.254s should inform later full/race budget measurement; no additional timeout increase was made. Existing individual30/90-second operation bounds retained.

### Frozen commit and artifacts
Branch story/MAC-2u36.
RED SHA:98d3b573ac493c4d7252bffb03c83f92dc99f82b, based on local epic f24b2df3cb1e1521f97b406a7516f72bb7bc7890.
Only production-repository change: internal/install/bootstrap_receipt_test.go,1111 added lines. File SHA256 ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23.
Raw final JSON /tmp/machinery-MAC-2u36-red.qRk5DC/98d3b57-final-matrix.jsonl SHA256 f2b15c4ef66ccd9b6f82c6d65795df4220364b767a4f71f92f265a7ab3a0947e.
Named leaf inventory /tmp/machinery-MAC-2u36-red.qRk5DC/98d3b57-leaf-inventory.md (also reproduced below).
Focused CLI raw SHA256509a3d2e5a384a9f437dad52d3c8cbdb92ac105e5e85148371ccf21627dfb37b; alias raw SHA2563a384604d50ba2ced4ea9877ae09d6a9fec699987c7ee35cbd1456063c7f750e.
Budget investigation: current1111 test lines are61 above approximate1050 test allocation; plus the estimated350 future production would total1461 versus under1400 estimate. Latest authorized amendment +220/-12 over903 lines includes8alias sensitivity leaves,3supported recording controls,2conflict→next-native flows, full expected-source bytes/inventory/canonical-link checks, unselected-state comparisons, and moved expected receipt oracle. This modest ~4.4% combined estimate overrun retains required proof; PM must assess actual GREEN sizing. No additional file, production/API/hook or regrouping scope added.

### Real integration boundary and containment
bootstrapReleaseFixture builds actual cmd/machinery v9.9.1/v9.9.2 binaries from the worktree and produces real source bytes with cmd/release-archive. Only existing link-time version/release endpoint strings point at a private loopback server; binary and source downloads have real checksums. No command output, command runner or replacement CLI is mocked.
Actual standalone install and update execute the built temporary binaries with process-local HOME/USERPROFILE/config/XDG/temp roots and explicit temporary binary destination. Reference installations independently establish full current content/inventory, not whatever receipt the tested updater happens to emit. Parent-finalization cases call real Update with the DEFAULT runner and unmodified actual placement children. Output observation never supplies fake child results.
The existing closeInstallFile seam is scoped only to parent receipt-* directly in the current prepared journal scratch directory, after all real child completions, exact complete content and full journal coverage. The desired normalized real-digest receipt is constructed independently after those observations, not inside the close callback. Fault would close a real file twice and preserve os.ErrClosed. On unchanged production this hook never fires: AC6 tests are missing-parent-publication RED, NOT exercised late-close rollback proof. No artifact post-image, authority directory or journal record is changed to manufacture that fault.
The existing MACHINERY_INTERNAL_TEST_LOCK_ROOT override is honored only in Go test (test.v flag registered). In-process parent tests set it to the real CLI's temporary HOME cache; actual CLI ignores the override and chooses the same temporary cache itself. It is not a product bypass and selects no live cache. Parent authority/finalization tests verify owned journal cleanup and actual bounded operation-lock reacquisition.
Installed /Users/ramirosalas/.local/bin/machinery remains SHA2565205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849. No installed skills/plugins/agents, user Dagger resources, root branch, remote repository or live config were modified. No preflight/push/sync/setup/recover/dev-link/live update was run.

### Exact failure classification (17 leaves)
-1 unit: complete_recorded_mixed_plan — bootstrap chooses defaults instead of the full ordinary recorded plan.
-2 real bootstrap intact/edited convergence — topology changes and recorded homes/native content remain stale; ordinary counterpartsPASS47.17s/43.59s with exact release bytes/version/content/digests/plugins and same-release idempotence.
-4 missing-regular repair leaves (single Codex and simultaneous Codex/OpenCode, ordinary/bootstrap) — valid repair request aborts when early child receipt inventory sees a still-missing later target. Actual binary/source download and first placements occur; missing-file failure is the intended success-contract RED.
-1 intact bootstrap later-source-failure leaf — omits native targets and reports success instead of reaching missing OpenCode source. Ordinary counterpartPASS38.79s: actual binary/home/Codex mutations, then real acquireInstallSourceSnapshot failure at adapters/opencode/plugins/machinery.js, full exact pre-state restored.
-2 missing-prestate later-source leaves — early complete receipt inventory prevents reaching repaired Codex then later OpenCode failure. Exact original absence/state restoration checks retained; these do NOT prove the later boundary occurred.
-1 unsafe native symlink bootstrap — succeeds despite unsafe artifact substitution.
-4 parent-finalization leaves — receipt-present/absent, no-fault/fault: each child changes the persisted receipt; parent final publication count0. All selected children, full independent release placements, desired receipt readiness and prepared coverage observed; no stale desired-digest/alias errors remain. Journal cleanup/lock reuse pass. No-fault final receipt comparison succeeds on actual bytes; overall leaf correctly remains sequencing RED. No real close fault/late publication rollback was exercised.
-2 standalone conflict leaves — exact all-four-home and compact repeated-other-group requests wrongly return success and alter valid state; subsequent real Codex+OpenCode install, with NO manual receipt repair, rejects the invalid overlap receipt. Fresh, disjoint and same-ordered-group copy-change controlsPASS, with full source contents/topology/digests and unselected ownership preserved.

### AC Verification
| AC | Authored proof and actual result |
|---|---|
|1|TestBootstrapReceiptPlan plus intact ordinary/bootstrap CLI convergence; mixed homes/native Copy:false/true and plugin obligations. Recorded bootstrap parityRED; no-receipt/default/plugin-aware/schema1/selector controlsPASS.|
|2|Real corrupt/unsafe/schema/plugin-discovery/non-directory-parent controlsPASS; native-symlink bootstrapRED. Existing bounded receipt FIFO/symlink/oversize/swap/unknown/duplicate/trailing/type/semantic/digest validation testsPASS. Edited/missing safe files are repair-success requests, not unsafe ownership.|
|3|Intact and edited ordinary full convergence/idempotencePASS; bootstrap and single/multi-missing repairRED. Complete independently expected release bytes/inventory/digests/topology/plugin/sentinel/binary checks retained.|
|4|Real ordinary intact later-target post-mutation rollbackPASS38.79s; bootstrap and missing-prestate boundary gapsRED. Real killed held-download startup recoveryPASS20.22s, explicitly PRE-mutation interruption proof only.|
|5|Actual built binaries/checksummed loopback downloads/source/real CLI processes/temp roots execute without skips/mocks. Helper-only plan assertions are distinguished from real CLI evidence; no external runtime lane needed.|
|6|All4parent finalization leavesRED on actual premature child publication and missing parent close. Correct oracle/coverage/content checks execute; actual final-close error and rollback remain futureGREEN proof. All4absent/forged/unprepared/out-of-scope authority/cleanup controlsPASS; real standalone valid recording remainsPASS.|
|7|Exact overlap+bounded repeated conflict/next-native flowsRED on false success and unusable emitted receipt. Three named fresh/disjoint/same-group-copy positivesPASS. No automatic merge/split policy or manual repair between conflict/next operation.|

### Preserved history and review timing
Original v1 99956740ae5559262790a9473b5597c1775928f2/file8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329 and36leaf32PASS4FAIL0SKIP271.751s log remain history; stale_artifact rejection was disputed and superseded, not valid RED. Actual ordinary edited repairPASS and missing repairFAIL drove reviewed R2.
Historical5a94840 broad49leaf34PASS15FAIL0SKIP536.345s includes4invalid authority fixture errors and truncated tool diagnostics; it is not final proof. Corrected93f8e6c focus4PASS4FAIL135.883s includes the source-diagnostic oracle mismatch.
cdb8036 diagnostic amendment committed21:38:23Z before dispatcher hold arrived; independent21:42:11Z review explicitly authorized retention AFTER the edit. Preserve chronology.
e1fd0a3 actual standalone writer corruption is valid AC7 product evidence but invalid parent-boundary setup evidence. d9d53b9 reference-only absent-receipt workaround remains distinct from conflict tests, where no workaround occurs. Its faulty root alias/close-dependent desired digest were test-oracle defects, separately reviewed and repaired under22:03:52Z authorization in98d3b57. All original logs/commits retained; no failures relabeled as production merely to obtain RED.

LEARNINGS:
- Installed content drift/regular-file absence is not automatically unsafe ownership; actual ordinary repair controls prevented freezing an incorrect rejection contract.
- Real delegated children expose early whole-receipt inventory and standalone publication corruption that mocked runners miss.
- Parent-process final publication needed an explicit product sequencing contract before a process-local close seam could fairly test it.
- Test/CLI lock roots and macOS alias normalization require real contention and sensitivity controls; incorrect fixture/oracle failures are not RED.
- Capture full native JSON directly before display truncation, preserve every leaf and historical mismatch, and freeze the exact replayed SHA.

### Complete named leaf inventory
| Leaf | Outcome | Seconds |
|---|---|---|
|TestBootstrapReceiptRootAlias/root|PASS|0|
|TestBootstrapReceiptRootAlias/descendant|PASS|0|
|TestBootstrapReceiptRootAlias/canonical_root|PASS|0|
|TestBootstrapReceiptRootAlias/canonical_descendant|PASS|0|
|TestBootstrapReceiptRootAlias/similar_sibling|PASS|0|
|TestBootstrapReceiptRootAlias/interior_alias|PASS|0|
|TestBootstrapReceiptRootAlias/repeated_suffix|PASS|0|
|TestBootstrapReceiptRootAlias/sibling_journal|PASS|0|
|TestBootstrapReceiptPlan/first_bootstrap_defaults_control|PASS|0|
|TestBootstrapReceiptPlan/supported_schema_one_control|PASS|0|
|TestBootstrapReceiptPlan/complete_recorded_mixed_plan|FAIL|10.51|
|TestBootstrapReceiptCLI/standalone_receipt/supported_recording/fresh_mixed_groups|PASS|0.02|
|TestBootstrapReceiptCLI/standalone_receipt/supported_recording/disjoint_additional_group|PASS|2.9|
|TestBootstrapReceiptCLI/standalone_receipt/supported_recording/same_ordered_group_copy_change|PASS|4.1|
|TestBootstrapReceiptCLI/standalone_receipt/reject_cross_group_repeated_false_then_native|FAIL|28.08|
|TestBootstrapReceiptCLI/standalone_receipt/reject_cross_group_repeated_true_then_native|FAIL|23.8|
|TestBootstrapReceiptCLI/parent_finalization/absent_false_close_fault_false|FAIL|31.35|
|TestBootstrapReceiptCLI/parent_finalization/absent_false_close_fault_true|FAIL|31.49|
|TestBootstrapReceiptCLI/parent_finalization/absent_true_close_fault_false|FAIL|28.23|
|TestBootstrapReceiptCLI/parent_finalization/absent_true_close_fault_true|FAIL|33.17|
|TestBootstrapReceiptCLI/receipt_authority_absent|PASS|13.6|
|TestBootstrapReceiptCLI/receipt_authority_forged|PASS|3.3|
|TestBootstrapReceiptCLI/receipt_authority_unprepared|PASS|3.11|
|TestBootstrapReceiptCLI/receipt_authority_out_of_scope|PASS|3.1|
|TestBootstrapReceiptCLI/converges_bootstrap_false|PASS|47.17|
|TestBootstrapReceiptCLI/converges_bootstrap_true|FAIL|29.88|
|TestBootstrapReceiptCLI/repair_edited_owned_artifact_bootstrap_false|PASS|43.59|
|TestBootstrapReceiptCLI/repair_edited_owned_artifact_bootstrap_true|FAIL|26.19|
|TestBootstrapReceiptCLI/repair_missing_owned_artifact_bootstrap_false|FAIL|26.04|
|TestBootstrapReceiptCLI/repair_missing_owned_artifact_bootstrap_true|FAIL|19.13|
|TestBootstrapReceiptCLI/repair_both_native_groups_missing_bootstrap_false|FAIL|28.19|
|TestBootstrapReceiptCLI/repair_both_native_groups_missing_bootstrap_true|FAIL|21.4|
|TestBootstrapReceiptCLI/later_target_failure_rolls_back_bootstrap_false|PASS|38.79|
|TestBootstrapReceiptCLI/later_target_failure_rolls_back_bootstrap_true|FAIL|19.14|
|TestBootstrapReceiptCLI/missing_prestate_later_target_failure_rolls_back_bootstrap_false|FAIL|30.3|
|TestBootstrapReceiptCLI/missing_prestate_later_target_failure_rolls_back_bootstrap_true|FAIL|21.96|
|TestBootstrapReceiptCLI/corrupt_receipt|PASS|13.55|
|TestBootstrapReceiptCLI/unsafe_receipt|PASS|12.71|
|TestBootstrapReceiptCLI/stale_schema|PASS|13.26|
|TestBootstrapReceiptCLI/plugin_discovery_failure|PASS|13.55|
|TestBootstrapReceiptCLI/symlink_native_artifact|FAIL|20.18|
|TestBootstrapReceiptCLI/non-directory_native_parent|PASS|11.34|
|TestBootstrapReceiptCLI/interrupted_download_recovers_real_transaction|PASS|20.22|
|TestReceiptArtifactDigestIgnoresInstallTimeButDetectsModeAndContent|PASS|3.91|
|TestUninstallDeletionFailureRollsBackArtifactsAndReceipt|PASS|3.76|
|TestAbandonedLegacyReceiptLockDirectoryDoesNotBlock|PASS|0.36|
|TestLoadReceiptRejectsFIFOWithoutOpening|PASS|0|
|TestLoadReceiptRejectsNonPrivateLeafConfigDirectory|PASS|0|
|TestReceiptReadModifyWriteIsSerialized|PASS|6.64|
|TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap/symlink|PASS|0|
|TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap/oversize|PASS|0|
|TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap/config_directory_swap|PASS|0|
|TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap/entry_swap|PASS|0|
|TestForgetReceiptUsesNativeCaseAliasIdentity|PASS|0.51|
|TestCorruptReceiptFailsLoudly|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/unknown_root|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/unknown_home_field|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/unknown_target_field|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/duplicate_root|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/duplicate_nested|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/wrong_homes_type|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/wrong_copy_type|PASS|0|
|TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology/trailing_value|PASS|0|
|TestSemanticallyInvalidReceiptFailsBeforeUpdate|PASS|0|
|TestUpdateRollsBackAllHomesBinaryAndReceiptOnLaterFailure|PASS|1.92|
|TestBootstrapDefaultPlanIsPluginAware|PASS|0.06|


## AUTHORITATIVE STANDALONE RECEIPT OVERLAP TRIAGE — MAC-2u36 — 2026-09-05

Disposition: absorb the reproduced standalone writer defect into existing P0 MAC-2u36 under MAC-ui8a. Canonical Description now contains seven ACs. Source ownership remains update.go/receipt.go/install.go and the already named tests; no sibling or new dependency is created. The existing MAC-gcrr/MAC-ou97 blocks remain intact. Scope collision inspection found no sibling implementation owning these three source files; capstone MAC-ou97 consumes update.go downstream.

Product decision: successful standalone recording must never publish a receipt rejected by its own loader. Final complete normalized inventory receives existing validation before publication. A cross-group request that leaves conflicting homes under current same-canonical replacement semantics is rejected with paths and actionable retry guidance; preserve the prior valid receipt and full installation, rolling back any owned writes when transaction post-images remain unchanged. Preserve concurrent foreign-change protection and recoverable journal behavior. Fresh/disjoint installs and same-ordered-group reruns/copy-mode changes remain supported. Automatic merge/split/recanonicalization is not introduced.

Rationale verified in committed source: recordHomeInstallLocked replaces only a group sharing Homes[0], normalizeReceipt only sorts, saveReceipt refreshes inventory but omits validateReceipt before rename, and validateReceipt rejects repeated/nested homes on the next load. installLocked already routes recording failures through its artifact transaction rollback. README.md 592-595 promises exact successful topology; cmd/machinery/install.go defines canonical-home and copy placement, without cross-group ownership migration semantics. Reconciliation would need decisions about unselected symlink dependents and preserved copy ownership; choosing fail-closed does not require that new feature. This is a correction to the owned publication guarantee, not a separate subsystem.

Required new proof: real built temporary CLI, exact mixed-group seed and all-copy cross-group reproduction, intended nonzero conflict diagnosis, exact state/receipt restoration, then real native-install success and complete valid inventory. Retain fresh/disjoint/same-group controls and all existing safety/finalization tests. AC6 parent-PROCESS final publication, unchanged receipt through every completed child, complete real digests, late close-fault rollback and existing authenticated delegation remain unchanged. No new seam/API or production edit in RED.

Budget investigation: d9d53b9 already has903 test LOC; its roughly194-line parent-boundary helper/addition implements four independently required fault/control and present/absent cases. Old1100 total would leave197 production LOC versus prior350. New estimate remains4-6files, under1400 total (~1050tests/~350production), allowing about147 additional test lines for overlap/control and independently reviewed oracle repairs. Independent PM must inspect the concrete proof/helper cost; no weakening safety tests to fit an estimate. All individual operation bounds and15m broad replay remain unchanged pending measured evidence.

Evidence preserved: d9d53b980a393c527eaf856b0bba5d8bc70867ec clean retained RED worktree; original v1 and all raw logs remain history. The e1fd0a3 actual standalone corruption is product evidence but invalid parent-boundary fixture evidence. d9d53b9 only removes the independent task-owned reference receipt first to establish legitimate absence; it does not fix production. The two paused oracle defects (macOS non-prefix normalization, expected digests depending on the close hook) remain independent PM review items, not new production defects. Prior21:42:11Z explicit closeInstallFile authorization remains intact; this triage itself grants no test-edit authorization, RED approval, delivery or implementation permission.

Tooling record: first supported nd edit attempt failed before mutation because EDITOR contained an executable plus argument, while this nd expects one executable path. sr_pm stopped and reported it; dispatcher explicitly authorized a task-owned executable Ruby editor correction. Corrected supported pvg nd edit succeeded. This is an nd invocation constraint, not another Machinery bug. Optional absence of repository AGENTS.md/convention files was found during initial instruction/context checks, not product preflight or a product defect.

## nd_contract
status: in_progress

### evidence
- Read full canonical issue, author pause/reproduction and prior authorization history through pvg issues show MAC-2u36 --json; independently read committed receipt.go/install.go, public CLI help and docs, and retained d9d53b9 boundary source. No runtime test replay performed by triage.
- Canonical repaired through supported pvg nd edit; guard verified original AC6 verbatim. Read back complete canonical seven-AC Description, preserving original sections and authorization history. Prior canonical description retained as quoted history below.
- pvg lint --backlog --epic MAC-ui8a --json: []; pvg rtm check --epic MAC-ui8a: passed (0 tagged requirements,18 checked stories,1 closed; not substantive coverage proof); pvg nd dep cycles: no cycles.
- Existing P0/hard-tdd/in_progress, assignee dev-MAC-2u36, parent MAC-ui8a and downstream blocks preserved. No status/claim/label/dependency transition; author remains paused undelivered.
- Root git status --short --branch: clean main...origin/main. No source/test/worktree or installed asset edits; no branch switch, sync/push/GitHub mutation or scripts/preflight.sh execution.

### proof
- [x] Triage assigned concrete standalone overlap publication bug to existing coherent owner and specified bounded fail-closed behavior with supported-rerun compatibility.
- [x] Parent-process AC6, prior authorizations, safe missing-file repair and no-unsafe-parent/concurrent-change boundaries retained.
- [x] New AC7 verification and measured scope/budget rationale are explicit; structural backlog checks pass.
- [ ] Independent PM scope/budget/concrete-test review and explicit oracle/test-edit authorization pending.
- [ ] AC1-7 final-SHA complete RED replay, separate RED approval, GREEN and independent acceptance pending.


## Historical canonical description before standalone overlap triage
Preserved as quoted history; current Description is authoritative.

> ## Description
> ## USER INTENT
> A user rerunning the installer or ordinary machinery update must get a complete, safely repaired installation at the requested release across all recorded placements, including a missing owned native artifact. Success must mean current content, complete receipt evidence and preserved topology; failure must preserve recoverable pre-run state.
> 
> ## Context (Embedded)
> Two production defects share this full-plan update boundary:
> - NEXT1: install.sh passes --bootstrap-defaults, but updatePlan chooses default homes before the complete recorded home/native plan.
> - Newly reproduced in MAC-2u36: ordinary update retains the missing Codex target in its plan, but refreshDirectInstalls runs home groups first; each child install calls recordHomeInstallLocked -> saveReceipt -> refreshReceiptArtifacts over ALL recorded placements. A missing later Codex file aborts the first child's receipt inventory before native repair is reached.
> Actual isolated CLI diagnostic on frozen v1 99956740ae5559262790a9473b5597c1775928f2 failed in 35.820s after verified binary replacement and first home-group mutation: "inventory installed artifacts for receipt: digest .../.codex/agents/machinery-fsm-author.toml: lstat ...: no such file or directory". It did not assert final rollback state. This disproves the earlier assumption that the exact missing-file ordinary run would already pass; it does NOT change the required successful repair behavior.
> The existing ordinary intact convergence and edited-regular-file repair controls pass. Metadata validity, path/type safety and plugin ownership are distinct from repairable installed content drift or absence.
> 
> ## Scope decision and Ownership
> This existing P0 bug owns both defects because both prevent the same recorded-plan convergence and use the same direct-refresh transaction. No separate issue or artificial dependency is created.
> Production ownership expands explicitly from internal/install/update.go to:
> - internal/install/update.go: receipt-aware bootstrap planning, full-plan refresh coordination and final commit/publication sequencing.
> - internal/install/receipt.go: bounded receipt accumulation/finalization necessary for this transaction, preserving normal receipt validation/persistence.
> - internal/install/install.go: child placement/receipt interaction needed for authenticated parent-owned full-plan refresh; preserve normal standalone install recording.
> Tests: internal/install/bootstrap_receipt_test.go; directly associated regression additions in internal/install/receipt_test.go and internal/install/install_test.go only when required by the changed receipt/standalone-install boundary. Frozen existing assertions cannot be changed without explicit independent reviewer authorization.
> Read-only consumers unless a concrete need is separately reviewed: internal/install/targets.go, internal/install/transaction.go, internal/install/lock.go, cmd/machinery/install.go and cmd/machinery/update.go. No blanket CLI/target/lock/journal rewrite is included. You are not alone in the codebase; preserve others' edits and coordinate shared paths with dispatcher.
> 
> ## Boundary Map
> PRODUCES:
> - internal/install/update.go -> complete recorded-plan convergence, including safely missing owned native artifacts, under the existing update transaction.
> - internal/install/receipt.go -> validated complete receipt publication coordinated with successful full direct refresh.
> - internal/install/install.go -> bounded child-refresh participation while ordinary install continues to persist validated receipts.
> - internal/install/bootstrap_receipt_test.go -> real CLI RED/GREEN matrix for both defects and preserved controls.
> - internal/install/receipt_test.go -> focused receipt-publication regression proof if needed.
> - internal/install/install_test.go -> focused standalone-install regression proof if needed.
> CONSUMES:
> - Existing internal/install/update.go.
>   spec: updatePlan(opts UpdateOptions) (refreshPlan, error); refreshDirectInstalls(binary, source string, plan refreshPlan, run commandRunner, out io.Writer) error.
> - Existing internal/install/receipt.go.
>   spec: loadReceipt() (installReceipt, bool, error); saveReceipt(receipt installReceipt) (retErr error); refreshReceiptArtifacts(receipt *installReceipt) error; recordHomeInstallLocked(homes []string, copyAll bool) error; recordTargetInstallLocked(names []string, copyAll bool) error.
> - Existing internal/install/install.go.
>   spec: Install(opts Options) error; installLocked(opts Options) (retErr error).
>   fields: Options.Homes []string; Targets []string; From string; Copy bool; Record bool. Current CLI passes Record: true.
> - Existing internal/install/transaction.go.
>   spec: beginArtifactTransaction(paths []string) (*artifactTransaction, error); delegatedInstallOperation() bool; delegatedArtifactTransaction(paths []string) (*artifactTransaction, error).
>   source: authenticated delegated children require the parent's prepared journal and coverage of their exact artifact paths; parent transaction snapshots binary, direct placements and receipt, and owns final commit/rollback.
> - Existing internal/install/targets.go.
>   spec: installTargets(names []string, src string, copyAll bool, out io.Writer, before func(string) error) error.
>   source: validates target source before placement; receipt recording is in installLocked, not this function.
> 
> ### Story Acceptance Criteria
> 1. Valid supported receipt plus --bootstrap-defaults uses the same complete recorded home/native/plugin plan and relevant discovery as ordinary machinery update without selectors. Preserve per-group copy/symlink modes. No-receipt first bootstrap retains plugin-aware defaults; explicit homes/targets remain incompatible with bootstrap.
> 2. Fail closed with actionable diagnostics and no silent defaults for malformed/duplicate/unknown/trailing JSON, unsupported schemas, invalid topology/inventory/digest encoding, unsafe receipt permissions/type, unsafe artifact or parent path/type substitutions, and uncertain plugin ownership, including --skip-plugins. Supported schema 1 is not rejected merely because it is older. Valid receipt plus edited/missing owned regular artifact with unchanged safe parents/topology is repairable; absence alone must not be classified as unsafe ownership. Existing safety validation must not be relaxed.
> 3. Actual ordinary and bootstrap full-plan update succeeds for intact installs, an edited regular owned Codex artifact, and a missing regular owned native artifact, restoring exact current-release content and binary, complete receipt topology/digests/plugin obligations, both mixed-mode home groups and Codex/OpenCode targets, unrelated-file preservation and same-release idempotence. Exact missing-file fixture: remove only temporary HOME/.codex/agents/machinery-fsm-author.toml after valid receipt recording; no unsafe parent/type substitution. Test both native copy-mode groups missing an owned file in the same run as a bounded guard against a fix that merely repairs one hardcoded target first. During RED the ordinary missing-file case is EXPECTED TO FAIL behaviorally on unchanged production; intact ordinary and edited-file ordinary cases remain passing controls.
> 4. Retain actual later-target failure after binary, home and earlier native mutations with complete pre-run restoration of binary, homes/native artifacts, receipt and unrelated sentinels. Also exercise that rollback with a safely missing owned artifact in pre-state: absence must be restored as absence, not replaced by a half-repaired result. Retain real interrupted journal recovery at CLI startup and explicitly separate its pre-mutation interruption boundary from post-mutation rollback proof. Preserve independent malformed/schema/path/type/plugin-ownership negatives and mixed-copy-mode checks.
> 5. Integration uses actual built release/CLI install/update subprocesses, real checksummed archives and temporary home/config/target/binary roots. No mocks, stubs, skip-if-missing or env-gated omissions. No live installation mutations. Machinery product behavior/tests remain standalone with no Paivot/pvg/nd, label or commit-convention dependency.
> 6. A full direct-refresh operation must not require complete final artifact inventory before remaining selected placements can repair safely missing owned files. The Update parent PROCESS that holds the operation lock and owns the prepared transaction must perform final complete receipt inventory/publication itself, after every authenticated selected placement child has successfully returned and all direct placement/source checks have succeeded, and before that parent's direct transaction commit. During the direct-refresh phase, delegated placement children do not publish intermediate or final persisted receipts: the receipt's pre-run bytes, existence, file type and mode remain unchanged until parent finalization (if no receipt existed before the run, it remains absent until parent finalization). In-memory or journal-owned private coordination may accumulate selected topology within existing authority; it must not advertise an intermediate installation as final. The parent publishes a complete normalized receipt with real refreshed digests for the full recorded/selected plan, preserving recorded topology/copy modes/plugin obligations and existing supported-schema validation. Do not drop missing entries, retain fabricated/stale digest evidence, invent success for an incomplete plan, or bypass final validation. Any final inventory/publication failure before direct commit fails the operation and rolls back the whole direct update when transaction-written post-images are unchanged; preserve the existing refusal to overwrite concurrent foreign changes and its recoverable diagnostic/journal behavior. Standalone nondelegated install still records validated successful topology. Any receipt deferral/coordination remains restricted to authenticated parent-owned prepared journal scope; forged/absent parent authority, unprepared journal and out-of-scope paths remain rejected. No public or env-only skip-receipt switch. Host-managed plugin refresh retains its existing post-direct-commit ownership/obligation semantics, with no new atomic host-plugin rollback or schema-migration claim.
> 
> ## Testing Requirements
> - Hard TDD remains explicitly authorized. After independent PM reviews this scope/contract and authorizes the revised tests, a RED-only author revises the disputed edited/missing cases, adds the bounded multi-missing and missing-pre-state rollback proof, and any focused new boundary tests needed for AC6. No production edits before independently approved RED.
> - Preserve original candidate 99956740ae5559262790a9473b5597c1775928f2, test SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329 and /tmp/machinery-MAC-2u36-red.qRk5DC. Existing 36-leaf replay: 32 pass, 4 fail, 0 skip, 271.751s; only three original failures are valid bootstrap defect proof. The fourth stale_artifact rejection is superseded. Missing-file ordinary FAIL 35.820s is new genuine defect evidence, separate from earlier edited-file ordinary PASS 36.508s.
> - Preserve complete-recorded-plan, defaults/selectors, intact normal/bootstrap convergence, plugin obligations, real later-target rollback, interrupted recovery, and all legitimate safety negatives. Rename obsolete stale_artifact/disappeared_target rejection cases as repair-success requirements. Only specifically reviewer-authorized frozen-test changes are allowed.
> - Fresh replay: go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json. This package deadline replaces 10m: the compiled 5a94840 broad matrix took 536.345s for 49 leaves (34 pass, 15 fail, 0 skip), including four invalid early authority-fixture failures. Corrected real authority controls take approximately 60s, bringing the conservative replay budget near 596.345s before the new AC6 positive/fault real updates and ordinary scheduling variance; 600s is inadequate for the complete expanded matrix. The 900s package bound provides room for those measured/required cases. Preserve every individual operation timeout, frozen test selection and assertion; no skipped cases, retries masking hangs or relaxed operation bounds. This is not permission for a hung individual update. Keep the existing GREEN 15m package command unchanged pending actual measurement; full preflight remains final-epic-only.
> - Focused receipt/standalone-install/delegated-authority tests selected by actual test names introduced or affected. GREEN runs go test -count=1 -timeout=15m ./internal/install ./cmd/machinery for ordinary native package regression coverage. No full scripts/preflight.sh here; final epic gate owns heavy preflight.
> - Record SHA, exact command, named leaf inventory, zero skips, assertion failure causes and passing controls. Compile/import/fixture/infra/timeout errors are not behavioral RED. Unexpected missing-file receipt-inventory failure is a valid success-contract assertion failure, not an acceptable permanent rejection.
> - If internal deferred receipt behavior is introduced, add proof that normal standalone install still publishes correct receipts, missing/forged/out-of-scope parent authority cannot enable it, and final publication failure rolls back real prior mutations. Reuse existing authenticated delegation mechanisms; implementation details require evidence, not an assumed new public API.
> - Finalization boundary proof is mandatory: use real Update with its default command runner, actual checksummed built release and real placement children, plus a successful no-fault counterpart. Independently observe current release binary/home/native content, full prepared-journal path coverage and unchanged persisted receipt through the final completed child. Then the parent's real final receipt scratch-file persistence must encounter a deterministic failure before receipt rename/commit. Use the EXISTING process-local closeInstallFile seam only after independent PM approves the concrete test: narrowly target the final receipt-* scratch file in the current prepared journal, actually close the real file and surface its OS close failure, prove the hook fired on that exact final publication operation, and preserve normal file behavior otherwise. Do not substitute a fabricated command result, new production hook/API, modified child executable or user/env-only bypass.
> - Required late-failure assertions: no direct commit; exact binary/home/native/receipt pre-state restored, including absence where applicable; unrelated sentinels preserved; journal cleanup and reusable/released operation lock. The no-fault counterpart must publish complete current digests/topology and commit successfully. Compile/fixture errors, early missing-file inventory, or a never-fired hook are not this boundary proof. Existing concurrent-change rejection remains a distinct safety case; deleting a live updated artifact with os.Remove or durableRemove after a child is not a receipt-publication fault and cannot be used to demand unsafe overwrite during rollback.
> - This parent-process/unchanged-receipt refinement and any proposed boundary test must receive independent PM review and explicit test-edit authorization before the RED author uses it. Existing authorized matrix work may continue under its prior authorization; no source edits are authorized during RED. Ownership and budget are unchanged.
> - Separate independent PM replays revised RED before approval/freezing. GREEN preserves exact approved test/fixture/config bytes; any later repair requires explicit reviewer authorization and re-RED.
> 
> ## OUT OF SCOPE
> - Arbitrary missing-root recreation, changed unsafe parents, and ownership uncertainty: this story preserves their safety boundary; it does not authorize blind repair.
> - Redesign of host-managed plugin transactions, lock/journal format or public install selectors: outside the direct receipt-publication defect; report a concrete need for review before expanding.
> - Other assessment subsystems stay in sibling stories. No pushes, sync, remote mutation, installed binary/plugin/skill/agent replacement, dev-link or healthy worktree cleanup. NIL uses the installed product.
> - Final heavy preflight, main merge and isolated release candidate belong to epic gate MAC-ou97.
> 
> ## DIFF BUDGET
> Expected 4-6 files, under 1100 total changed LOC: approximately 750 test LOC including the existing 469-line real fixture, and 350 production LOC across the three named files. This replaces the disproven 450-LOC estimate after newly reproduced receipt-publication scope. Do not pad to the budget; material overrun or another production file requires PM investigation and explicit scope review, not weaker tests.
> 
> ## Dependencies
> Parent MAC-ui8a; remains blocking MAC-gcrr and MAC-ou97. No new inter-story dependency: one developer owns both coupled installer defects and one independent PM reviews the full outcome. No other sibling implementation declares the three owned installer source seams. Consumer guidance and epic acceptance remain blocked until this story is accepted.
> 
> ## MANDATORY SKILLS
> - developer for RED/GREEN roles; codebase-memory for source discovery; pm_acceptor for independent scope/test authorization, RED replay and acceptance.
> 
> ## Delivery Requirements
> This repaired scope requires independent PM review before author re-RED; the previous 20:55:32Z test-edit authorization did not authorize implementing this newly reproduced defect. Preserve that review as history and obtain explicit expanded test authorization. Do not deliver or approve v1 unchanged.
> Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands/results, inventory, changed-file rationale, per-AC proof and residual limits. Product has no dependency on these development coordination tools.


## MAC-2u36 RED AUTHOR PAUSE — overlap triage and boundary-oracle review
No delivery or claim release. Dispatcher requested a clean pause after the already-running focused replay; final full latest-SHA matrix is deferred until triage and reviewed oracle repairs settle.

### Frozen candidate and history
Current SHA d9d53b980a393c527eaf856b0bba5d8bc70867ec; branch story/MAC-2u36; internal/install/bootstrap_receipt_test.go SHA256 401a837755e1bb001680d33dd5516c18584c907cd1557fb00025dcc54e8cabe1; 903 added test LOC, no production edits. Boundary addition e1fd0a3 cost +194/-1, followed by three-line isolated-reference receipt removal d9d53b9. Combined budget under1100 now leaves197 LOC for eventual production versus earlier350 estimate; required four-case proof/helper cost must be investigated, not trimmed.
Keep v1 99956740ae5559262790a9473b5597c1775928f2 and all evidence. cdb80369490db3c33586735934111f85796f9389 diagnostic amendment committed21:38:23Z before dispatcher hold reached author; explicit independent authorization at21:42:11Z followed the edit, not preceded it.

### Focused commands and exact outcomes
- e1fd0a3 compile-only: go test -count=1 -timeout=30s ./internal/install -run '^$' passed0.396s; zero selected tests, compile evidence only.
- e1fd0a3: go test -count=1 -timeout=5m ./internal/install -run '^TestBootstrapReceiptCLI/parent_finalization' -json redirected stdout/stderr directly to /tmp/machinery-MAC-2u36-red.qRk5DC/e1fd0a3-parent-finalization.jsonl. Exit1,48.882s. Independent reference setup hit the separately reported real standalone overlapping-home receipt bug; no four boundary leaves executed. Log SHA256 bc36698fb00d9fe169db2db208ff0d7c5b5d7c20a77a19c022b216f0e7ac5ae3.
- d9d53b9 same focused command redirected stdout/stderr directly to /tmp/machinery-MAC-2u36-red.qRk5DC/absent-reference-parent-finalization.jsonl. Exit1,166.511s. Four leaves: absent_false_close_fault_false FAIL31.18s; absent_false_close_fault_true FAIL31.48s; absent_true_close_fault_false FAIL29.36s; absent_true_close_fault_true FAIL29.13s. Leaf counts0pass/4fail/0skip. Full raw JSON parses; SHA256 0a073d3e1c2c7ed334130021b82436ea11d4efd3586ebbfd5b0063bb5ff17b63.
- All four actual Update operations returned nil and observed all selected children (4/4 recorded mixed mode,2/2 explicit absent all-copy). Actual complete independent content comparison did not fail. Persisted receipt changed after each child: genuine AC6 sequencing defect. Parent close publication count0 on unchanged code is not late injected-fault rollback proof. Journal cleanup and bounded real lock reacquisition executed without failure.
- Two test-oracle limitations below invalidate claiming the whole four-failure result as clean approved RED. No final matrix run.
- pvg verify internal/install/bootstrap_receipt_test.go --format text --include-tests: PASSED1file0issues at d9d53b9. Worktree clean.
- Live installed binary read-only SHA256 remains5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849; no live install/plugin/agent mutation, no preflight/push/sync/remote operations.

### Pending narrow oracle review (no repairs made after pause)
1. bootstrapFinalizationCase line673 canonical normalizer uses strings.ReplaceAll(path,f.root,canonicalRoot). A /private/var journal path contains the /var fixture-root substring and becomes /private/private/var. Lines679-689 therefore falsely report missing prepared coverage; line719 may fail to match a real scratch path. Not a production journal gap. Smallest proposed repair: replace only an actual prefix (HasPrefix + canonicalRoot + TrimPrefix), leave already-canonical paths unchanged, preserve exact path equality and full coverage.
2. bootstrapFinalizationCase lines731-739 compute desired current digests and normalize wantReceipt only inside the close observer. No-fault comparison lines774-776 then sees stale desired digests if the hook never fires. That secondary receipt mismatch is an oracle dependency, not an independent production defect. Smallest proposed repair: compute independent desired normalized receipt once at the final completed-child observation after actual placements match the independent release reference; both scratch and final persisted-receipt comparisons consume that desired object without depending on the close observer firing. Keep all publication/count/scope/error/rollback expectations unchanged.
Neither repair has been made; review/authorization required. Existing true ordinary intact later-target rollback and intact/edited convergence passing history remain separate from these new sequencing RED cases.

## nd_contract
status: in_progress

### evidence
- Retained committed tests-only d9d53b980a393c527eaf856b0bba5d8bc70867ec and complete direct raw focused logs; historical broad5a94840 has49leaves34pass15fail0skip536.345s with4invalid authority fixture failures.
- Corrected93f8e6c focused authority/rollback history:8leaves4pass4fail0skip135.883s; ordinary intact failure was the reviewed diagnostic-oracle mismatch, not production RED. Tool diagnostics were truncated, so not final full proof.
- No active test process remains. No delivery/approve-red/claim/status/label mutation.

### proof
- [x] Real standalone overlap defect reported with exact temporary CLI repro for Sr PM triage.
- [x] Four real parent boundary cases attempted with all children and complete release placements observed; premature child receipt writes identified.
- [ ] Two narrow boundary-oracle repairs pending independent review.
- [ ] AC6 actual final publication/real close-fault rollback remains unexercised on unchanged production.
- [ ] Full coherent latest-SHA raw15m replay, independent RED review/approval, GREEN and acceptance remain pending.


## DISCOVERED_BUG — standalone overlapping home receipt groups (RED author, 2026-09-05)
title: Standalone install regrouping can publish overlapping home groups rejected by the next operation.
context: While constructing the independent initially-absent-receipt AC6 reference, an actual built v9.9.2 CLI first recorded custom-a/custom-b with Copy:false and copy-a/copy-b with Copy:true, plus Codex Copy:false and OpenCode Copy:true. All roots were temporary. Running another actual standalone install from current source with --copy and --home custom-a --home custom-b --home copy-a --home copy-b succeeded, but the following standalone install --from current-source --target codex --target opencode --copy failed after real placement output with: invalid installation receipt .../config/install.json: home install paths overlap or repeat: .../home/copy-a and .../home/copy-a.
repro: bootstrapSeed current-release reference, then machinery install --from SOURCE --copy --home HOME/custom-a --home HOME/custom-b --home HOME/copy-a --home HOME/copy-b, then machinery install --from SOURCE --target codex --target opencode --copy. Every machinery command used the built temporary reference binary, not the installed host binary.
affected_files: internal/install/receipt.go (recordHomeInstallLocked, saveReceipt, normalizeReceipt, validateReceipt); internal/install/install.go (standalone recording call).
discovered_during: MAC-2u36.
Evidence: compiled e1fd0a3e072182fd21e80f6a7f4b7cddfd384565; go test -count=1 -timeout=5m ./internal/install -run '^TestBootstrapReceiptCLI/parent_finalization' -json, direct raw stdout/stderr at /tmp/machinery-MAC-2u36-red.qRk5DC/e1fd0a3-parent-finalization.jsonl, exit1 in48.882s. Failure is real standalone production behavior but invalid AC6 fixture evidence: no four boundary leaves executed, no late receipt fault proof.
Fixture-only workaround committed d9d53b9: remove exactly the task-owned current-release reference config/install.json before the explicit all-copy install, so the independent reference legitimately starts with an absent receipt. No production change, no acceptance predicate change, no live artifact change. Preserve this bug for Sr PM triage; whether it is coupled to AC6 or separate is not decided by the author.
Parent instructed finish the already-running focused replay only, then pause undelivered with claim retained; no final full matrix or production expansion before triage.


## nd_contract
status: in_progress

### evidence
- Independent PM R2 pre-revision scope/commit-boundary review read canonical pvg issues show MAC-2u36 --json, complete current Description, appended R2 triage and relevant prior history; read preserved missing-file diagnostic source/result and frozen test via git show.
- Local source inspection confirmed updateLocked transaction/commit sequencing, refreshDirectInstalls grouping, installLocked per-child receipt writes/source verification, saveReceipt full inventory, delegated prepared-journal exact-path coverage, authenticated lock capability and absent pre-state snapshot support.
- Retained worktree remains clean at 99956740ae5559262790a9473b5597c1775928f2; bootstrap_receipt_test.go SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329. Story diff remains one 469-line test file; root remains main.
- Appended expanded TEST-EDIT AUTHORIZED R2 comment specifying allowed files/cases, preserved assertions, complete final publication/rollback boundary and both required commit tags tdd-red and [test-edit-authorized]. No runtime tests rerun and no source/test/live installation edits.

### proof
- [x] R2 scope: both confirmed defects belong to the same recorded-plan direct update transaction; update.go/receipt.go/install.go ownership and bounded budget reviewed.
- [x] R2 revision authorization: exact paired edited/missing repair cases, native multi-missing, missing-prestate rollback, non-weakening convergence helper changes and bounded AC6 additions authorized.
- [ ] AC #1-6: revised candidate author replay, full independent RED replay/approval, GREEN implementation and acceptance remain pending.
- [ ] Final receipt failure/authority controls must prove their actual boundary; early missing-file receipt failure is not substitute evidence.


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


## AUTHORITATIVE BUG TRIAGE R2 — MAC-2u36 — 2026-09-05
The current canonical Description (USER INTENT through Delivery Requirements) is the authoritative complete scope, AC and testing contract. It supersedes conflicting earlier descriptions/addenda, including the assumption that ordinary missing-file update was already a passing control. Historical text and all v1 evidence are retained.

Disposition: absorb the newly discovered ordinary receipt-publication defect into existing P0 MAC-2u36 under MAC-ui8a. No new bug/dependency is needed: it directly blocks the same recorded-plan repair outcome and uses the same update transaction/test fixture. Existing blocks MAC-gcrr and MAC-ou97 remain intact. Bounded epic ownership review found no sibling implementation claiming update.go, receipt.go or install.go.

Before: ownership only update.go/bootstrap_receipt_test.go; the prior clarification expected ordinary missing-file success as a control and scoped revisions to two disputed cases.
After: production ownership explicitly includes update.go, receipt.go and install.go, with conditional focused tests in receipt_test.go/install_test.go. Ordinary missing-file success is now an intended RED assertion failure against unchanged production. Intact ordinary and edited-artifact ordinary flows remain passing controls. targets.go, CLI, lock.go and transaction.go are read-only consumers unless separately justified/reviewed. Revised estimate is 4-6 files/1100 changed LOC (750 tests including existing 469; 350 production), with overrun investigation.

Required direct-refresh commit boundary: parent owns complete selected placement plan, binary and receipt under the prepared bounded transaction. Remaining selected placements must be able to repair owned missing native files before complete final inventory. Only after successful direct placements and source validation may the complete normalized receipt with real digests publish before parent commit. Final inventory/publication failure rolls back the entire direct update. No omitted inventory members, stale/fabricated digest acceptance, or unauthenticated skip-receipt switch. Any internal coordination preserves authenticated delegated authority, prepared journal coverage and normal standalone install recording. Plugin refresh retains existing post-direct-commit semantics.

Added proof: exact missing Codex file in ordinary/bootstrap modes; both native copy-mode groups missing an owned file together to avoid a target-order special case; later-target failure from missing-file pre-state must restore absence as absence after actual prior mutations. Retain original safety, full-plan/defaults/selectors, intact convergence, edited repair, mixed topology, actual later-target rollback and pre-mutation interrupted startup recovery. Any receipt-coordination mechanism also needs focused legitimate standalone-install and rejected untrusted-delegation proof. User product remains standalone.

Independent review gate: root must obtain PM review of this expanded ownership/AC/commit contract and specific RED edit/addition authorization before author resumes re-RED. The prior 20:55:32Z comment remains historical authorization for its exact scope, not approval of this new production defect scope or v1 RED. This Sr. PM action grants no RED approval, delivery, acceptance or GREEN waiver. V1 remains unchanged and undelivered.

### Reviewed evidence and limits
- Full latest pvg nd show MAC-2u36 plus exact missing-file diagnostic source/result in /tmp/machinery-MAC-2u36-red.qRk5DC; FAIL 35.820s after first home-group mutation. That diagnostic stops at CLI failure and supplies no independent rollback conclusion.
- Graph exact source: refreshDirectInstalls/updateLocked; recordHomeInstallLocked/recordTargetInstallLocked/saveReceipt/refreshReceiptArtifacts; installLocked/Options; newInstallCmd; beginArtifactTransaction/delegatedInstallOperation/delegatedArtifactTransaction; installTargets; existing lock capability.
- Source confirms per-child Record: true and full receipt inventory before subsequent native groups, even though the plan retains the missing target. Existing parent journal supports absent pre-state and exact delegated path coverage. These capabilities identify a bounded repair seam, not an approved concrete implementation.
- Coverage generation 2026-09-05T20:28:41Z: metadata_match/no_recorded_issue for update.go, receipt.go, install.go, targets.go, transaction.go, lock.go and cmd/machinery/install.go (best-effort only; exact snippets read).
- Earlier "No new production defect established" finding is superseded by the executed missing-file diagnostic. No unresolved user product choice is needed to repair this failure.
- No source/tests/fixtures edits, runtime tests, installed product writes, GitHub/remote operations or heavy preflight during this triage.

## nd_contract
status: in_progress

### evidence
- Sr. PM repaired canonical Description via supported pvg nd edit and preserved the original as quoted historical description; appended R2 triage through pvg nd update --append-notes.
- P0 existing issue MAC-2u36 retains owner dev-MAC-2u36, hard-tdd, parent MAC-ui8a and downstream blocks. No new issue or status transition.
- Frozen v1 99956740ae5559262790a9473b5597c1775928f2 and prior proof remain preserved. Independent PM scope review and expanded RED authorization pending.

### proof
- [x] Triage: confirmed concrete ordinary missing-file receipt-ordering defect and assigned coherent ownership within MAC-2u36.
- [x] Contract: corrected disproven passing-control assumption; preserved desired repair success, safety boundaries and explicit final receipt/rollback requirements.
- [ ] AC #1-6: revised RED replay, implementation and independent acceptance pending.


## Historical canonical description before parent-process finalization refinement
Preserved as quoted history; current Description is authoritative.

> ## USER INTENT
> A user rerunning the installer or ordinary machinery update must get a complete, safely repaired installation at the requested release across all recorded placements, including a missing owned native artifact. Success must mean current content, complete receipt evidence and preserved topology; failure must preserve recoverable pre-run state.
> 
> ## Context (Embedded)
> Two production defects share this full-plan update boundary:
> - NEXT1: install.sh passes --bootstrap-defaults, but updatePlan chooses default homes before the complete recorded home/native plan.
> - Newly reproduced in MAC-2u36: ordinary update retains the missing Codex target in its plan, but refreshDirectInstalls runs home groups first; each child install calls recordHomeInstallLocked -> saveReceipt -> refreshReceiptArtifacts over ALL recorded placements. A missing later Codex file aborts the first child's receipt inventory before native repair is reached.
> Actual isolated CLI diagnostic on frozen v1 99956740ae5559262790a9473b5597c1775928f2 failed in 35.820s after verified binary replacement and first home-group mutation: "inventory installed artifacts for receipt: digest .../.codex/agents/machinery-fsm-author.toml: lstat ...: no such file or directory". It did not assert final rollback state. This disproves the earlier assumption that the exact missing-file ordinary run would already pass; it does NOT change the required successful repair behavior.
> The existing ordinary intact convergence and edited-regular-file repair controls pass. Metadata validity, path/type safety and plugin ownership are distinct from repairable installed content drift or absence.
> 
> ## Scope decision and Ownership
> This existing P0 bug owns both defects because both prevent the same recorded-plan convergence and use the same direct-refresh transaction. No separate issue or artificial dependency is created.
> Production ownership expands explicitly from internal/install/update.go to:
> - internal/install/update.go: receipt-aware bootstrap planning, full-plan refresh coordination and final commit/publication sequencing.
> - internal/install/receipt.go: bounded receipt accumulation/finalization necessary for this transaction, preserving normal receipt validation/persistence.
> - internal/install/install.go: child placement/receipt interaction needed for authenticated parent-owned full-plan refresh; preserve normal standalone install recording.
> Tests: internal/install/bootstrap_receipt_test.go; directly associated regression additions in internal/install/receipt_test.go and internal/install/install_test.go only when required by the changed receipt/standalone-install boundary. Frozen existing assertions cannot be changed without explicit independent reviewer authorization.
> Read-only consumers unless a concrete need is separately reviewed: internal/install/targets.go, internal/install/transaction.go, internal/install/lock.go, cmd/machinery/install.go and cmd/machinery/update.go. No blanket CLI/target/lock/journal rewrite is included. You are not alone in the codebase; preserve others' edits and coordinate shared paths with dispatcher.
> 
> ## Boundary Map
> PRODUCES:
> - internal/install/update.go -> complete recorded-plan convergence, including safely missing owned native artifacts, under the existing update transaction.
> - internal/install/receipt.go -> validated complete receipt publication coordinated with successful full direct refresh.
> - internal/install/install.go -> bounded child-refresh participation while ordinary install continues to persist validated receipts.
> - internal/install/bootstrap_receipt_test.go -> real CLI RED/GREEN matrix for both defects and preserved controls.
> - internal/install/receipt_test.go -> focused receipt-publication regression proof if needed.
> - internal/install/install_test.go -> focused standalone-install regression proof if needed.
> CONSUMES:
> - Existing internal/install/update.go.
>   spec: updatePlan(opts UpdateOptions) (refreshPlan, error); refreshDirectInstalls(binary, source string, plan refreshPlan, run commandRunner, out io.Writer) error.
> - Existing internal/install/receipt.go.
>   spec: loadReceipt() (installReceipt, bool, error); saveReceipt(receipt installReceipt) (retErr error); refreshReceiptArtifacts(receipt *installReceipt) error; recordHomeInstallLocked(homes []string, copyAll bool) error; recordTargetInstallLocked(names []string, copyAll bool) error.
> - Existing internal/install/install.go.
>   spec: Install(opts Options) error; installLocked(opts Options) (retErr error).
>   fields: Options.Homes []string; Targets []string; From string; Copy bool; Record bool. Current CLI passes Record: true.
> - Existing internal/install/transaction.go.
>   spec: beginArtifactTransaction(paths []string) (*artifactTransaction, error); delegatedInstallOperation() bool; delegatedArtifactTransaction(paths []string) (*artifactTransaction, error).
>   source: authenticated delegated children require the parent's prepared journal and coverage of their exact artifact paths; parent transaction snapshots binary, direct placements and receipt, and owns final commit/rollback.
> - Existing internal/install/targets.go.
>   spec: installTargets(names []string, src string, copyAll bool, out io.Writer, before func(string) error) error.
>   source: validates target source before placement; receipt recording is in installLocked, not this function.
> 
> ### Story Acceptance Criteria
> 1. Valid supported receipt plus --bootstrap-defaults uses the same complete recorded home/native/plugin plan and relevant discovery as ordinary machinery update without selectors. Preserve per-group copy/symlink modes. No-receipt first bootstrap retains plugin-aware defaults; explicit homes/targets remain incompatible with bootstrap.
> 2. Fail closed with actionable diagnostics and no silent defaults for malformed/duplicate/unknown/trailing JSON, unsupported schemas, invalid topology/inventory/digest encoding, unsafe receipt permissions/type, unsafe artifact or parent path/type substitutions, and uncertain plugin ownership, including --skip-plugins. Supported schema 1 is not rejected merely because it is older. Valid receipt plus edited/missing owned regular artifact with unchanged safe parents/topology is repairable; absence alone must not be classified as unsafe ownership. Existing safety validation must not be relaxed.
> 3. Actual ordinary and bootstrap full-plan update succeeds for intact installs, an edited regular owned Codex artifact, and a missing regular owned native artifact, restoring exact current-release content and binary, complete receipt topology/digests/plugin obligations, both mixed-mode home groups and Codex/OpenCode targets, unrelated-file preservation and same-release idempotence. Exact missing-file fixture: remove only temporary HOME/.codex/agents/machinery-fsm-author.toml after valid receipt recording; no unsafe parent/type substitution. Test both native copy-mode groups missing an owned file in the same run as a bounded guard against a fix that merely repairs one hardcoded target first. During RED the ordinary missing-file case is EXPECTED TO FAIL behaviorally on unchanged production; intact ordinary and edited-file ordinary cases remain passing controls.
> 4. Retain actual later-target failure after binary, home and earlier native mutations with complete pre-run restoration of binary, homes/native artifacts, receipt and unrelated sentinels. Also exercise that rollback with a safely missing owned artifact in pre-state: absence must be restored as absence, not replaced by a half-repaired result. Retain real interrupted journal recovery at CLI startup and explicitly separate its pre-mutation interruption boundary from post-mutation rollback proof. Preserve independent malformed/schema/path/type/plugin-ownership negatives and mixed-copy-mode checks.
> 5. Integration uses actual built release/CLI install/update subprocesses, real checksummed archives and temporary home/config/target/binary roots. No mocks, stubs, skip-if-missing or env-gated omissions. No live installation mutations. Machinery product behavior/tests remain standalone with no Paivot/pvg/nd, label or commit-convention dependency.
> 6. A full direct-refresh operation must not require complete final artifact inventory before remaining selected placements can repair safely missing owned files. Keep one parent-owned bounded transaction covering the binary, full selected direct plan and receipt. Publish a complete normalized receipt with real refreshed digests only after all selected direct placements and source checks succeed, before final direct transaction commit; any final inventory/publication failure must fail and roll back the whole direct update. Do not drop missing entries, retain fabricated/stale digest evidence, invent success for an incomplete plan, or bypass final validation. Standalone install continues to record validated successful topology. Any internal receipt deferral/coordination must be restricted to authenticated parent-owned journal scope and must not become a user/env-only skip-receipt backdoor; forged/absent parent authority, unprepared journal or out-of-scope paths remain rejected. Host-managed plugin refresh retains its existing post-direct-commit ownership/obligation semantics; no new claim of atomic host-plugin rollback.
> 
> ## Testing Requirements
> - Hard TDD remains explicitly authorized. After independent PM reviews this scope/contract and authorizes the revised tests, a RED-only author revises the disputed edited/missing cases, adds the bounded multi-missing and missing-pre-state rollback proof, and any focused new boundary tests needed for AC6. No production edits before independently approved RED.
> - Preserve original candidate 99956740ae5559262790a9473b5597c1775928f2, test SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329 and /tmp/machinery-MAC-2u36-red.qRk5DC. Existing 36-leaf replay: 32 pass, 4 fail, 0 skip, 271.751s; only three original failures are valid bootstrap defect proof. The fourth stale_artifact rejection is superseded. Missing-file ordinary FAIL 35.820s is new genuine defect evidence, separate from earlier edited-file ordinary PASS 36.508s.
> - Preserve complete-recorded-plan, defaults/selectors, intact normal/bootstrap convergence, plugin obligations, real later-target rollback, interrupted recovery, and all legitimate safety negatives. Rename obsolete stale_artifact/disappeared_target rejection cases as repair-success requirements. Only specifically reviewer-authorized frozen-test changes are allowed.
> - Fresh replay: go test -count=1 -timeout=10m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json. Timeout increased from 5m because the existing real replay already consumed 271.751s and the newly required paired repair/rollback cases add CLI executions; do not relax assertions or skip cases to fit the old timeout.
> - Focused receipt/standalone-install/delegated-authority tests selected by actual test names introduced or affected. GREEN runs go test -count=1 -timeout=15m ./internal/install ./cmd/machinery for ordinary native package regression coverage. No full scripts/preflight.sh here; final epic gate owns heavy preflight.
> - Record SHA, exact command, named leaf inventory, zero skips, assertion failure causes and passing controls. Compile/import/fixture/infra/timeout errors are not behavioral RED. Unexpected missing-file receipt-inventory failure is a valid success-contract assertion failure, not an acceptable permanent rejection.
> - If internal deferred receipt behavior is introduced, add proof that normal standalone install still publishes correct receipts, missing/forged/out-of-scope parent authority cannot enable it, and final publication failure rolls back real prior mutations. Reuse existing authenticated delegation mechanisms; implementation details require evidence, not an assumed new public API.
> - Separate independent PM replays revised RED before approval/freezing. GREEN preserves exact approved test/fixture/config bytes; any later repair requires explicit reviewer authorization and re-RED.
> 
> ## OUT OF SCOPE
> - Arbitrary missing-root recreation, changed unsafe parents, and ownership uncertainty: this story preserves their safety boundary; it does not authorize blind repair.
> - Redesign of host-managed plugin transactions, lock/journal format or public install selectors: outside the direct receipt-publication defect; report a concrete need for review before expanding.
> - Other assessment subsystems stay in sibling stories. No pushes, sync, remote mutation, installed binary/plugin/skill/agent replacement, dev-link or healthy worktree cleanup. NIL uses the installed product.
> - Final heavy preflight, main merge and isolated release candidate belong to epic gate MAC-ou97.
> 
> ## DIFF BUDGET
> Expected 4-6 files, under 1100 total changed LOC: approximately 750 test LOC including the existing 469-line real fixture, and 350 production LOC across the three named files. This replaces the disproven 450-LOC estimate after newly reproduced receipt-publication scope. Do not pad to the budget; material overrun or another production file requires PM investigation and explicit scope review, not weaker tests.
> 
> ## Dependencies
> Parent MAC-ui8a; remains blocking MAC-gcrr and MAC-ou97. No new inter-story dependency: one developer owns both coupled installer defects and one independent PM reviews the full outcome. No other sibling implementation declares the three owned installer source seams. Consumer guidance and epic acceptance remain blocked until this story is accepted.
> 
> ## MANDATORY SKILLS
> - developer for RED/GREEN roles; codebase-memory for source discovery; pm_acceptor for independent scope/test authorization, RED replay and acceptance.
> 
> ## Delivery Requirements
> This repaired scope requires independent PM review before author re-RED; the previous 20:55:32Z test-edit authorization did not authorize implementing this newly reproduced defect. Preserve that review as history and obtain explicit expanded test authorization. Do not deliver or approve v1 unchanged.
> Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands/results, inventory, changed-file rationale, per-AC proof and residual limits. Product has no dependency on these development coordination tools.


## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97
- 2026-09-05T20:30:19Z status: open -> in_progress
- 2026-09-05T20:30:19Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-05T20:30:19Z claimed by dev-MAC-2u36
- 2026-09-05T22:30:04Z status: in_progress -> in_progress
- 2026-09-05T23:04:15Z status: in_progress -> open
- 2026-09-05T23:04:15Z released by ramirosalas
- 2026-09-05T23:04:49Z status: open -> in_progress
- 2026-09-05T23:04:49Z claimed by dev-MAC-2u36
- 2026-09-05T23:25:59Z status: in_progress -> in_progress
- 2026-09-05T23:25:59Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-05T23:33:04Z status: in_progress -> open
- 2026-09-06T00:09:30Z status: open -> in_progress
- 2026-09-06T00:09:30Z auto-follows: linked to predecessor MAC-a89e
- 2026-09-06T00:09:30Z claimed by dev-MAC-2u36

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Follows: [[MAC-olrx]], [[MAC-p8ce]], [[MAC-a89e]]

## Comments

### 2026-09-05T20:55:32Z ramirosalas
TEST-EDIT AUTHORIZED: internal/install/bootstrap_receipt_test.go -- Bounded independent RED-DISPUTE review validates the 2026-09-05 Sr. PM clarification against original AC1 parity, cmd/machinery/update.go forced-refresh help, loadReceipt/validateReceipt, buildRefreshPlan/targetInstalled/fileExists, the exact frozen test mutations, and the preserved ordinary edited-artifact diagnostic. Authorize revising only TestBootstrapReceiptCLI/stale_artifact and disappeared_target from rejection expectations into clearly named edited-owned-artifact and missing-owned-artifact repair-success cases. Both mutate only HOME/.codex/agents/machinery-fsm-author.toml with safe unchanged parents/topology and valid receipt metadata. Add actual isolated ordinary-update and bootstrap CLI controls for each, with restored current-release bytes (edited content gone / missing file recreated), complete recorded home/native/plugin plan and copy/symlink modes retained, refreshed receipt digests/topology, binary release verification, unrelated-file preservation, and same-release idempotence. Only minimal shared test structure necessary for those cases is authorized; existing assertions must remain equally strong. The missing-file ordinary control has NOT yet run: execute it and report any unexpected production failure to dispatcher/Sr. PM before expanding scope or changing expected repair semantics. Preserve all legitimate metadata/schema/path/type/plugin-ownership rejection cases, complete recorded mixed-plan and normal/bootstrap convergence tests, defaults/selectors, real post-mutation rollback and interrupted journal recovery. No other frozen assertion/fixture/config changes or production edits are authorized. Candidate v1 99956740ae5559262790a9473b5597c1775928f2 (file SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329) is NOT approved; preserve its original evidence and /tmp/machinery-MAC-2u36-red.qRk5DC logs. Every repair commit subject must carry tdd-red and [test-edit-authorized]. Re-freeze revised bytes and replay the fresh real CLI matrix against unchanged production, recording exact command, SHA/hashes, leaf counts, assertion failures, passing controls, zero skips, and justified LOC/timeout deltas, then obtain a separate independent full RED review. This is test-edit authorization only, not RED approval, delivery, GREEN waiver, acceptance, rejection, or closure.

### 2026-09-05T21:12:43Z ramirosalas
TEST-EDIT AUTHORIZED: R2 independent pre-revision scope/commit-boundary review for MAC-2u36. The canonical current Description and AUTHORITATIVE BUG TRIAGE R2 form a coherent six-AC contract for the same full recorded-plan direct update. The preserved exact ordinary missing-file diagnostic and local source confirm that child recordHomeInstallLocked -> saveReceipt -> refreshReceiptArtifacts inventories a later missing native artifact before repair. This supersedes the earlier passing-control assumption; the prior 20:55:32Z authorization remains historical.

AUTHORIZED RED-ONLY FILES AND CASES:
1. internal/install/bootstrap_receipt_test.go: replace only the obsolete TestBootstrapReceiptCLI/stale_artifact and disappeared_target rejection cases with clearly named paired ordinary/bootstrap successful-repair cases. Preserve their exact safe Codex regular-file mutations after valid receipt creation at temporary HOME/.codex/agents/machinery-fsm-author.toml. Ordinary missing-file success is an intended behavioral RED failure on unchanged production; intact ordinary and edited-file ordinary remain passing controls.
2. In that file add paired ordinary/bootstrap multi-missing success cases removing the exact Codex file plus temporary HOME/.config/opencode/plugins/machinery.js in the same run. Preserve the seeded two native Copy-mode groups (Codex Copy:false and OpenCode Copy:true); "both groups" does not authorize changing them both to Copy:true. Preserve both mixed home groups, safe unchanged parents/topology and unrelated sentinels.
3. In that file add paired ordinary/bootstrap later-OpenCode-source failure cases with the exact Codex file already missing in pre-state. Keep existing intact-prestate rollback cases. Prove actual binary, home and earlier Codex mutation before the intended later source failure, then complete restoration including os.IsNotExist for the initially missing Codex artifact, original receipt bytes and unrelated files. Early receipt-inventory rejection alone cannot pass the post-mutation/later-boundary assertions. Keep interrupted-startup recovery as separately identified pre-mutation proof.
4. Minimal shared fixture/helper refactoring needed for these cases is authorized, including extracting and strengthening existing convergence assertions without dropping them: exact current-release installed content (including rendered native content and symlink destination content), expected complete artifact path inventory with real digests, normalized topology/copy modes, preserved plugin obligations, exact verified binary bytes, unrelated-state preservation and same-release idempotence. Expected inventory/content must not be derived solely from whatever incomplete receipt/result the updater happens to emit. Preserve existing complete-recorded-plan, plugin-aware defaults, selectors, intact convergence, plugin obligations and legitimate safety negatives. Add only directly required AC2 missing safety/schema-1-positive controls; existing safety expectations may be strengthened but not inverted or removed.
5. internal/install/receipt_test.go and internal/install/install_test.go: additions only, limited to AC6 final receipt/publication and standalone/delegated installation boundary regressions. Cover validated normal standalone receipt recording; absent/forged parent authority, unprepared journal and paths outside parent coverage cannot enable receipt deferral; complete real inventory/publication after successful direct placements/source verification and before parent commit; final inventory/publication failure after real prior mutations causes full parent rollback. Bootstrap-specific real CLI cases and necessary test helpers may instead remain in bootstrap_receipt_test.go. No alteration of existing assertions in these two regression files is authorized. Reuse actual public/existing internal behavior for compilable RED; do not introduce production stubs or assume an unimplemented API. Focused filesystem fault/control fixtures must exercise real operations, preserve journal authority, and cannot substitute a mock runner for required CLI integration. Any inability to deterministically exercise the final boundary must be reported for review, not omitted or satisfied by the earlier receipt-inventory failure.

REVIEWED COMMIT BOUNDARY: one authenticated parent-owned prepared transaction covers binary, the complete selected direct plan and receipt. Remaining selected placements can repair owned absence before final inventory. Complete normalized receipt publication with real digests follows all direct placements and source checks and precedes final direct commit. Final inventory/publication failure rolls the whole direct transaction back. Normal standalone validated persistence and existing post-direct-commit host-plugin semantics remain intact. The future GREEN ownership of update.go, receipt.go and install.go is coherent within R2, but this authorization permits no production edits before separate RED approval. targets.go, transaction.go, lock.go and CLI files remain read-only absent further scope review.

Every RED amendment/addition commit subject MUST carry both literal tags tdd-red and [test-edit-authorized]. Create new commits preserving ancestor v1 99956740ae5559262790a9473b5597c1775928f2, original file SHA256 8ed8636856305d50b9517c4087ecfff3582dac4fb467f92d8f49f3c836916329 and /tmp/machinery-MAC-2u36-red.qRk5DC evidence. Preserve the 36-leaf v1 32-pass/4-fail/0-skip 271.751s result as history; it is not approved RED. No other frozen-test/fixture/config edits, removals, assertion weakening or production changes are authorized.

Re-freeze revised candidate bytes and obtain a separate full independent RED review/replay before GREEN. Author replay: go test -count=1 -timeout=10m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json, plus exact named focused boundary tests outside that selector. Record revised SHA/file hashes, exact commands, named leaf counts, zero skips, intended assertion failure causes, passing controls and per-AC coverage. Build/import/fixture/infra/timeout failures are not behavioral RED. The reviewed aggregate budget is 4-6 files/under 1100 changed LOC across eventual tests and production; investigate material overruns or additional files. No full preflight, live installation changes, sync/push/remote mutation, delivery, approve-red, GREEN waiver, acceptance/rejection status, claim release or closure is granted by this authorization.

### 2026-09-05T21:32:28Z ramirosalas
BOUNDARY PROPOSAL REVIEW: GAPS_FOUND — independent PM pre-RED test-contract review for MAC-2u36, 2026-09-05. This is a bounded proposal verdict only: no delivery/RED verdict, approve-red, acceptance/rejection, status/claim change, or production-edit authorization.

EXPECTED: Canonical R2 AC6 and TEST-EDIT AUTHORIZED 2026-09-05T21:12:43Z require an authenticated parent-owned prepared transaction covering binary, complete selected direct plan and receipt; repair may precede complete inventory; complete normalized real-digest publication follows all direct placements/source checks and precedes direct commit; late inventory/publication failure after actual prior mutations restores the whole update. Existing concurrent-artifact protection and standalone/delegation controls remain intact.

PROPOSAL A: real Update/default runner/real loopback release; observe completed final OpenCode child via UpdateOptions.Out; verify complete actual release placements; arm existing closeInstallFile only for receipt-* scratch under the current prepared journal; genuinely close twice to obtain OS ErrClosed before renameReplace; demand full rollback.
VERDICT: Not authorized as mandatory outcome RED under the existing contract. Parent-owned transaction does not explicitly require parent-PROCESS receipt publication. An authenticated final child can finish all placements/source checks, publish the complete receipt and return while the parent still owns final commit. runAndRelay emits buffered output only after run returns, and closeInstallFile is a process-local function variable. The proposed hook therefore misses an otherwise contract-compatible final-child publication. A never-fired hook would reject an implementation choice rather than establish an AC violation. This does not dispute the hook's usefulness if parent-process finalization is explicitly selected in the contract.

PROPOSAL B: assert unchanged receipt bytes after nonfinal completed children; after final child verify all release placements, then os.Remove one owned native artifact and demand final rejection/full rollback.
VERDICT: Not authorized as substitute AC6 receipt-failure proof. In a final-child publication implementation this injection occurs after receipt publication and exercises commit post-image validation, not final receipt inventory/publication failure. More decisively, rollbackInstallJournal intentionally refuses to overwrite live artifacts that do not match their durable transaction-written post-image. Removing an updated native artifact triggers that concurrent-change boundary: complete restoration is not the existing safe outcome. Requiring it would conflict with preserved safety and read-only transaction.go scope. The byte-identical receipt assertion also needs clarification: the contract specifies final publication ordering; it does not separately define every permissible intermediate validated receipt representation. Do not freeze exact byte immutability as a proxy for that unresolved distinction.

BOUNDED ALTERNATIVES CHECKED: Existing close/stage/authority hooks are process-local; default child output is buffered until child completion. Receipt directory permission/type changes may affect journal authority, publication and rollback together, can fail too early, and have platform/privilege complications. No deterministic process-neutral existing-seam fixture was established in this review. This is a scoped finding, not an exhaustive claim that no technique exists.

MINIMAL NEXT CLARIFICATION FOR SR PM: Choose and record whether final normalized receipt publication is specifically a parent-process responsibility after all real children return. If yes, authorize that narrow sequencing contract explicitly, including whether the persisted receipt must remain untouched before finalization; proposal A then becomes eligible for test-edit authorization after verifying its positive control and fault sensitivity. If process placement is intentionally unconstrained, retain that outcome and separately scope/review a deterministic mechanism that observes and injects at final receipt publication in whichever actual process performs it. Do not silently add a production stub, public API, env-only bypass, instrumented fake CLI, concurrency-safety weakening, or omit the boundary proof. This note grants neither option.

REQUIRED EVIDENCE ON ANY APPROVED NEXT PROPOSAL: actual built release and real children; independent expected binary/home/native content and full prepared journal coverage observed before the injected late failure; successful no-fault counterpart; proof that the intended publication/inventory operation actually encountered the fault and no commit occurred; exact full pre-state restoration, unrelated sentinels, journal cleanup and released operation lock. Earlier missing-file inventory failure, compile/fixture errors, and a hook that never runs do not satisfy the required boundary. All eventual authorized RED additions/amendments retain tdd-red and [test-edit-authorized] commit tags and require separate full independent RED review.

## nd_contract
status: in_progress

### evidence
- Read canonical current six-AC Description and exact 21:12:43Z authorization through pvg issues show MAC-2u36 --json; inspected committed test references only (a3282ac), no author uncommitted files.
- Graph Verify tier, project Users-ramirosalas-workspace-machinery, generation 2026-09-05T20:28:41Z; exact saveReceipt/installScratchFile/delegatedArtifactTransaction/renameReplace symbols and saveReceipt both-direction trace. Relevant update.go/receipt.go/install.go/transaction.go coverage is metadata_match/no_recorded_issue, best-effort only.
- Read local exact source: saveReceipt 263-304; installLocked 124-258; refreshDirectInstalls 472-505/runAndRelay 507-519; transaction commit 588 onward and rollbackInstallJournal 855 onward. Parent owns commit; delegated commit closes anchors; rollback preserves concurrent post-image changes.
- No runtime tests or mutable fixtures run; no source, test, installation, branch, queue, status or claim mutations.

### proof
- [x] Bounded proposed fault mechanisms assessed against actual sequencing and safety contracts.
- [ ] AC6 late receipt inventory/publication rollback proof remains unresolved pending the narrow contract/testability clarification above.
- [ ] AC1-6 candidate replay, independent RED approval, implementation and acceptance remain pending.

## AUTHORITATIVE FINALIZATION BOUNDARY REFINEMENT — MAC-2u36 — 2026-09-05
Scope: resolve the specific AC6 process/receipt-publication testability gap identified by BOUNDARY PROPOSAL REVIEW: GAPS_FOUND at 2026-09-05T21:32:28Z. Current canonical AC6 and finalization testing bullets are authoritative; all earlier authorizations, RED evidence, comments and contracts remain history. Ownership remains update.go/receipt.go/install.go and the already scoped tests; transaction.go/lock.go/CLI remain read-only. No new production seam/API or RED production edit is authorized.

### Decision and technical rationale
Select parent-PROCESS finalization, not merely abstract parent ownership. The Update process holding the operation lock and complete prepared journal waits for every authenticated selected placement child, then inventories/publishes the normalized receipt before its own direct commit. Persisted receipt bytes/existence/type/mode stay unchanged through all children; an initially absent receipt stays absent. Normal standalone nondelegated install still records a receipt.

This is a bounded refinement of the existing coordinator, not a new host-trust architecture: updateLocked already constructs the entire plan and journal, controls the new-release children, waits synchronously in refreshDirectInstalls and owns direct rollback/commit. Delegated child commit only closes its anchors. The parent already loads/validates receipts and has an existing parent-side saveReceipt path for changed plugin obligations. Making that same coordinator own the direct phase's final publication supplies a single durable commit boundary and avoids transient persisted receipts that cannot represent still-unrepaired placements. The old receipt may be stale while files are refreshed; the prepared journal/operation lock already governs that window and startup recovery. Do not advertise the old receipt as a newly finalized installation.

The product benefit is complete receipt evidence coupled to completion of the entire direct plan, not convenience of a specific hook. Supported schema behavior and complete target/topology validation remain requirements; this story introduces no new cross-version schema protocol. New-release children still perform actual placement/source checks. Host plugin work remains outside the committed direct transaction exactly as before.

### Alternatives considered
- Authenticated final-child publication: compatible with previous abstract wording, but leaves a different serializer/publisher lifecycle and makes the existing parent-process failure seam unable to observe publication. Not selected for this bounded story; no cross-process final-publisher protocol is needed once the existing coordinator finalizes.
- Process-neutral finalization instrumentation: would require a new reviewed observation/fault mechanism or production seam absent from current authorization; adds surface without improving the chosen complete-plan commit ownership.
- Post-child os.Remove/durableRemove or directory authority corruption: rejected. Those can exercise commit post-image/concurrent-change or authority failure rather than receipt publication and may correctly prevent rollback. Never weaken transaction.go's concurrent-artifact protections to make such a test pass.

### Required proof and pending authorization
The existing closeInstallFile seam is explicitly documented for deterministic durability fault tests. A concrete Proposal-A-style test is now eligible for independent PM review: real Update/default runner/built release/real children; observe full actual mutations and prepared-journal coverage; verify persisted receipt unchanged through final child; narrowly arm actual receipt-scratch Close failure in the parent before rename; prove the exact operation encountered it, no commit, complete pre-state rollback, unrelated preservation, journal cleanup and released/reusable lock. Include a passing no-fault counterpart that observes final complete publication. The fixture must not edit live installed artifact post-images to create this fault. A hook that never fires establishes no publication-failure evidence.

This decision is a contract repair, not authorization to change the frozen tests or approve RED. Independent PM must review this selected boundary plus the concrete test and record explicit test-edit authorization before author use. Earlier authorized actual-CLI matrix work may continue. Preserve current authorized commits including 5a948405..., a3282ac and 4c5e0b4, original v1 evidence and all subsequent fixture corrections; earlier fixture/lock-root errors are not behavioral RED. No source/worktree or installed product mutation occurred in this review.

### Evidence
- Read full current canonical R2 and latest PM boundary verdict through pvg nd show MAC-2u36 --json.
- Exact source: updateLocked parent journal/commit and post-commit plugin obligation save; refreshDirectInstalls/runAndRelay synchronous buffered child completion; saveReceipt inventory/temp write/sync/Close/rename; receiptArtifactPaths; transaction commit and rollbackInstallJournal concurrent post-image checks; closeInstallFile documented existing test seam.
- Graph Verify project Users-ramirosalas-workspace-machinery, generation 2026-09-05T20:28:41Z; four relevant update/receipt/install/transaction source files metadata_match/no_recorded_issue (best-effort), supplemented by exact source reads.
- No new public behavior, host-trust scope, dependency, label/status/claim transition or budget expansion.

## nd_contract
status: in_progress

### evidence
- Sr. PM refined canonical AC6/testing using supported pvg nd edit and preserved prior canonical text as quoted history.
- Appended finalization decision/options/evidence via pvg nd update --append-notes; terminal contract preserves the current in_progress claim.
- Independent PM boundary review and concrete RED authorization remain required before use.

### proof
- [x] Contract: parent-process final publication and unchanged persisted receipt through children are explicit.
- [x] Safety: standalone recording, authenticated delegation, supported schema validation and concurrent post-image protections remain intact.
- [ ] AC6: concrete late real-file publication-failure test authorization and independent execution proof pending.
- [ ] AC1-6: independent RED approval, implementation and acceptance remain pending.

## MAC-2u36 TARGETED RED PACKAGE DEADLINE REFINEMENT — 2026-09-05
Canonical targeted RED command is now:
go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json

Numeric rationale supplied by the dispatcher: compiled 5a94840 broad replay consumed 536.345s across 49 leaves (34 pass, 15 fail, 0 skip); four authority failures were invalid early fixture failures, not behavioral RED. Corrected actual authority controls take approximately 60s; a conservative runtime estimate is already near 596.345s before the new AC6 real positive/fault updates. A 600s package deadline therefore risks truncating valid complete execution. Set the package deadline to 900s.

This changes only the package runtime allowance in the tracker. Retain existing individual operation bounds, all final frozen cases, exact assertions, zero-skip rules, and failure classification. No permission for a hung individual update, reduced inventory or relaxed per-operation timeout. GREEN remains go test -count=1 -timeout=15m ./internal/install ./cmd/machinery until actual evidence warrants review; no speculative increase. Full preflight remains final-epic-only.

Historical canonical wording preserved:
> - Fresh replay: go test -count=1 -timeout=10m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json. Timeout increased from 5m because the existing real replay already consumed 271.751s and the newly required paired repair/rollback cases add CLI executions; do not relax assertions or skip cases to fit the old timeout.

No production/tests/worktree edits, execution/replay, source claim, approval or status/label/dependency change. Parent will lint and submit boundary plus timeout refinement for independent PM review.

## nd_contract
status: in_progress

### evidence
- Canonical RED package deadline changed via supported pvg nd edit from 10m to 15m using measured 536.345s replay, approximately 60s corrected authority controls and required added AC6 executions.
- Existing GREEN deadline and all individual operation limits remain unchanged; no runtime test performed in this tracker-only refinement.

### proof
- [x] Verification budget: targeted RED deadline accounts for complete expanded real matrix without dropping cases or weakening assertions.
- [ ] AC1-6: boundary/test authorization, complete RED replay, implementation and independent acceptance remain pending.


### 2026-09-05T21:42:11Z ramirosalas
TEST-EDIT AUTHORIZED: MAC-2u36 independent bounded follow-up review of parent-PROCESS AC6 refinement and exact later-source diagnostic repair. Effective with this comment; no RED approval, delivery, GREEN waiver, acceptance/rejection, claim release, or status/label change.

CONTRACT REVIEW: Read current canonical six ACs, AUTHORITATIVE FINALIZATION BOUNDARY REFINEMENT and TARGETED RED PACKAGE DEADLINE REFINEMENT. The selected Update parent now explicitly owns final complete receipt inventory/publication after every authenticated placement child returns and before parent direct commit. Persisted receipt pre-run bytes/existence/type/mode remain unchanged through all children; absence stays absent. This resolves the process-choice/byte-immutability gaps in my 21:32:28Z review. Concurrent foreign-postimage protection remains unchanged. Ownership stays update.go/receipt.go/install.go for future GREEN only; no production edits during this RED phase.

AUTHORIZED AC6 TEST ADDITIONS: internal/install/bootstrap_receipt_test.go only, plus minimal directly necessary helpers in that file. Add clearly named parent-finalization no-fault and close-fault cases for pre-existing and initially absent receipts, retaining an intact-artifact late-boundary fixture. Use real Update with its DEFAULT command runner, real checksummed built release/loopback archive, actual unmodified child CLI processes and isolated temporary roots. For absence cases choose a legitimate supported explicit/discovered/default plan and independently specify that plan; do not pretend a deleted receipt still supplies its lost copy/group metadata. Preserve the complete recorded mixed home/native plan in the receipt-present fixture.

OBSERVATION AND FAULT CONTRACT:
- Observe genuine relayed child completions; runAndRelay buffers output until the real runner returns. The output observer is observation only, not a fabricated command result or replacement runner. Check persisted receipt bytes/existence/type/mode after each selected child's completion, through the final completed child.
- Before arming the fault, independently verify actual current-release binary, both relevant home/native placements and exact expected content/topology, plus the live prepared journal's coverage of the binary, complete selected direct plan and receipt. Expected content/inventory must not be derived solely from the emitted receipt. Require all selected child completions and unchanged receipt. Retain full pre-run snapshot and unrelated sentinels.
- Use ONLY the existing closeInstallFile variable, scoped to the parent's receipt-* file directly under the current prepared journal scratch directory after that observation. Match the exact current transaction, not every receipt-like file. Verify the journal is still prepared, the persisted receipt is still pre-state, and the real scratch payload is complete/normalized with expected real digests before provoking failure. Actually close the real file successfully, then close it again and capture/return its actual OS closed-file error (errors.Is with os.ErrClosed where appropriate), before renameReplace. Record the exact operation/path and injection count. For all other files and in the no-fault counterpart call the original closer normally. Restore the global seam with cleanup; tests run sequentially with isolated environment and no t.Parallel. Do not mutate live artifact post-images, directory authority, or journal records to manufacture this failure.
- Fault assertions: the intended parent final publication was reached and encountered the real close error; returned error retains that cause; no direct commit; exact prior binary/home/native/receipt state including required absences and modes restored; unrelated sentinels preserved; journal cleaned and operation lock released/reusable. A bounded real reacquisition control must actually establish lock reuse.
- No-fault counterpart: same placement/finalization observations, complete normalized current receipt/digests/topology/plugin obligations, actual successful commit and clean/reusable lock. The close seam must not accidentally affect this control. Current missing-native early failure remains separate expected repair RED and cannot substitute for intact late-fault sensitivity.

RED CLASSIFICATION: On unchanged production, child receipt writes and absent parent finalization can cause these desired-contract tests to fail; report their exact assertions honestly. A never-fired hook is an unmet boundary assertion, not evidence that publication-failure rollback was exercised. No claim of actual late-fault restoration or passing parent-sequencing control until that path executes. Preserve existing actual ordinary intact/edited success controls; no assertion weakening or production stubs to make the new boundary reachable during RED.

EXACT DIAGNOSTIC AMENDMENT REVIEW: Author reports commit cdb80369490db3c33586735934111f85796f9389 was made at 2026-09-05T21:38:23Z before dispatcher's hold reached the author, relying on the earlier expanded R2 authorization; this explicit independent review occurs AFTER that amendment. Preserve this timing and commit; do not represent authorization as predating it. I read the committed one-line diff from 93f8e6c: replace the literal "source is missing OpenCode governance adapter" condition with both exact "adapters/opencode/plugins/machinery.js" and "no such file". Authorize retaining that exact amendment. In this fixed archive fixture only that source adapter is omitted; acquireInstallSourceSnapshot validates its explicit OpenCode entry before installTargets/validateTargetSource, so the real missing-source wrapper is legitimate. The unchanged mandatory actual "installed Codex agents ->", binary/home witnesses, nonzero failure, existing operation deadline, complete pre-state equality and missing-prestate os.IsNotExist check preserve the later source-failure boundary. Do not broaden to arbitrary missing paths/errors or permit early Codex receipt-inventory rejection. Preserve historical overly specific predicate failure as test-oracle mismatch, not a new production defect. Any further predicate change needs review under the existing frozen-test rules.

REPLAY AND HISTORY: Every new amendment/addition commit subject carries BOTH tdd-red and [test-edit-authorized]. Preserve all ancestor/history and original v1 evidence. Re-freeze final test bytes and run one complete latest-SHA raw JSON replay using go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json, plus exact named boundary tests outside that selector if any. Keep all individual operation bounds unchanged. Record the untruncated raw JSON artifact, final SHA/hashes, exact command, every named leaf/outcome, zero skips, assertion causes and passing controls. Prior 5a94840 536.345s broad run is historical, including four invalid authority-fixture failures; tool-truncated diagnostics cannot serve as full final evidence. Corrected controls on 93f8e6c do not replace final-SHA replay. This authorization is not the separate full independent RED review.

## nd_contract
status: in_progress

### evidence
- Canonical contract/decision/deadline read through pvg issues show MAC-2u36 --json; reviewed exact committed 93f8e6c test lines and cdb80369490db3c33586735934111f85796f9389 diff, no uncommitted author source.
- Exact graph/source review: acquireInstallSourceSnapshot, validateTargetSource, source traversal wrapper; prior saveReceipt, runAndRelay, prepared journal/commit/rollback review remains applicable. Six relevant source paths metadata_match/no_recorded_issue at generation 2026-09-05T20:28:41Z; best-effort caveat retained.
- No runtime replay, author fixture interference, source/test/installation edits, branch/status/claim changes or remote operations during this review.

### proof
- [x] Parent-process/unchanged-receipt refinement resolves the earlier bounded contract gaps.
- [x] Exact existing-seam no-fault/fault RED test contract authorized; cdb8036 semantic diagnostic repair independently reviewed after its recorded commit time.
- [ ] AC6 actual finalization/fault execution and full restoration evidence pending.
- [ ] AC1-6 complete final-SHA author replay, separate independent RED approval, GREEN and acceptance pending.

### 2026-09-05T22:03:52Z ramirosalas
TEST-EDIT AUTHORIZED: MAC-2u36 AC7 scope and paused d9d53b9 oracle repairs — bounded independent PM review, 2026-09-05. This authorizes only the test additions/repairs specified below. No full RED verdict, approve-red, delivery, GREEN production edits, acceptance/rejection, status/label/claim change, or queue action.

SCOPE REVIEW: Read the entire current canonical Description (seven ACs, testing, scope, dependencies and budget), AUTHORITATIVE STANDALONE RECEIPT OVERLAP TRIAGE, the full author pause/reproduction notes, and committed d9d53b980a393c527eaf856b0bba5d8bc70867ec helpers. AC7 is coherent within the existing receipt.go/install.go ownership: recordHomeInstallLocked replaces only the group whose first home matches, normalizeReceipt sorts without reconciling overlap, and saveReceipt currently does not validate the complete candidate before publication. installLocked already routes recording failure into transaction rollback. Safe conflict rejection preserves supported successful topology without inventing regrouping or ownership migration. All AC1-6 and the 21:42:11Z parent-process/real-close-fault contract remain mandatory, including concurrent foreign-postimage protection.

AUTHORIZED FILE: internal/install/bootstrap_receipt_test.go, with only minimal test helpers/cases required below. No other existing assertion/fixture/config changes are authorized by this note. No production stubs, new production APIs/hooks, mock runner, modified child executable or env-only bypass.

ORACLE REPAIR 1 — exact bounded fixture-root alias normalization:
Replace the local bootstrapFinalizationCase strings.ReplaceAll path mapping. Map only path == fixtureRoot or a true descendant beginning fixtureRoot + directory separator to canonicalRoot with the same suffix. Leave already-canonical and unrelated paths unchanged. A bare strings.HasPrefix(path, fixtureRoot) is insufficient: similar-prefix siblings are foreign. Keep exact normalized equality for every expected prepared-journal path and for the receipt scratch parent directory; do not replace the coverage check with prefix/subset sampling, drop artifacts or broaden the fault target.
A minimal extracted test-only helper and deterministic path sensitivity cases are authorized: fixture root/descendant map correctly; already canonical root/descendant remain unchanged and idempotent; similar-prefix sibling and a foreign path containing the alias only as an interior substring remain unchanged; a sibling journal/scratch path does not become the authorized transaction path. Include a suffix containing a repeated alias-like substring if needed to distinguish prefix replacement from ReplaceAll. These small oracle checks must run without dependence on a macOS-only alias or skip condition. Resolve the actual fixture root alias once; no need to resolve every artifact path, including intentionally absent paths. Do not normalize arbitrary foreign paths into authority.

ORACLE REPAIR 2 — expected receipt independent of close observer:
Move complete desired schema/digest refresh and normalization out of the closeInstallFile callback to the final genuine completed-child observation, after independently expected full installed content/topology and exact release binary have been compared. Build a separate desired receipt from the fixed seeded/explicit selected plan and full independently known path inventory; copy mutable slices as necessary so expectation construction does not mutate the pre-state fixture. Compute each real digest from those verified artifact paths, normalize once, and explicitly record that the expected receipt is ready. Both actual scratch payload and final persisted-receipt comparisons consume this same desired object. A missing expected-ready observation is a failing assertion, never fallback to stale digests or a fabricated expected receipt. Never derive topology/path membership/digests from the updater's emitted receipt alone.
Keep all child-count/order and unchanged-receipt assertions, actual expected-content comparison, prepared full coverage, exact scratch payload comparison, publication count/hook-fired check, actual double-close OS error/cause assertion, rollback/no-commit, journal cleanup and lock reuse checks. Do not omit no-fault receipt comparison merely because the hook did not run. Source/content mismatch remains a failure even if current-file digests could be computed.

AUTHORIZED AC7 REAL CLI CASES:
1. Reuse the actual built temporary CLI/source and existing real seed to establish [custom-a,custom-b] Copy:false, [copy-a,copy-b] Copy:true, Codex Copy:false and OpenCode Copy:true. Verify the starting receipt is valid and represents those exact groups/modes/full real inventory; snapshot full installation/receipt bytes, existence, types/modes, link destinations and unrelated state. Execute the exact all-copy four-home standalone install in canonical testing requirements. Require bounded nonzero conflict rejection with the actual conflicting paths and actionable nonconflicting-group retry or explicit reconfiguration guidance. Match the semantic diagnosis, not an arbitrary incidental error. Assert complete pre-state restoration before the next operation. Do not demand placement output before rejection: safe validation before any writes is permitted; any owned writes that occur must roll back.
2. Without deleting/rewriting the receipt or repairing the installation after that request, run actual standalone install --from SOURCE --target codex --target opencode --copy. Require success, valid complete home/native/plugin topology and real digest inventory, both target modes as explicitly requested, and preservation of unselected home relationships/content and unrelated files. Keep the conflict rejection assertion even if this next operation succeeds. On unchanged production, report false success/invalid emitted receipt as the intended AC7 behavioral failure; do not convert it to a setup error or silently repair it.
3. Bounded positive controls using the same built CLI/helpers: fresh valid installation, adding a disjoint home group, and rerunning the same ordered recorded home set with a supported copy-mode change. Check exact expected group membership/order/mode and real installed topology/content/digests after each success, plus the other group's ownership/content and native/plugin/unrelated state. Existing bootstrapSeed operations may supply fresh-install setup evidence, but name and assert the positive outcomes so their execution is auditable. The tests must distinguish same-group mode change from unsupported cross-group merge.
4. A compact additional repeated/nested-against-another-recorded-group negative is authorized to exercise AC7's general conflict rule, using the same real CLI/snapshot pattern. Keep it bounded and do not introduce a regrouping feature or unsafe-parent repair. The exact all-four-home reproduced case remains mandatory. Cases must be selected by Bootstrap/Receipt names or explicitly included in the recorded focused command.

BUDGET REVIEW: Confirmed committed file length 903 lines and e1fd0a3 +194/-1 boundary addition; inspected the four-case shared finalization helper and reusable seed/state/content/CLI helpers. The old 750-test/1100-total estimate no longer accommodates required proof and the existing 350 production estimate. The revised under1400 combined estimate (~1050 test/~350 production, same 4-6 files) is justified for this scope: one shared conflict/next-operation flow, compact positive/negative cases, small path sensitivity table and relocation of the existing digest loop can reuse existing infrastructure. Approximately147 additional test lines is an estimate, not permission to compress away proof or a rigid per-helper quota. Record actual changed-file/LOC cost and explain material overrun before further scope expansion. Do not spend the budget on automatic group migration. No runtime-bound increase is granted.

EVIDENCE CLASSIFICATION/PRESERVATION: Retain d9d53b9 and all ancestors/raw logs. Read-only aggregation of absent-reference-parent-finalization.jsonl confirms four failing leaves, zero pass/skip, 166.511s package result; four reported complete child counts (4/4,4/4,2/2,2/2), publications=0 and nil Update results. Premature child receipt writes are genuine sequencing RED. The 104 prepared-coverage diagnostics from faulty alias mapping and two stale desired-receipt mismatch diagnostics are oracle defects, not additional production proof. No actual late close fault was exercised. e1fd0a3 standalone writer corruption is AC7 product evidence but invalid parent-boundary fixture evidence; d9d53b9's isolated reference receipt removal only establishes legitimate explicit absence and remains distinct from the conflict test, where such a workaround is forbidden.

Every amendment/addition commit subject must contain BOTH tdd-red and [test-edit-authorized]. Preserve exact prior history; re-freeze amended bytes and record SHA/file hashes. Final author evidence must include one complete untruncated latest-SHA raw JSON replay of go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json plus exact named cases outside that selector, all named leaves/counts, zero skips, operation bounds unchanged, intended assertion causes and positive controls. Measure runtime; do not drop cases or silently relax the existing15m package bound. Separate full independent RED review/approval remains required. No late-fault rollback claim until the intended real parent close operation and restoration actually execute.

## nd_contract
status: in_progress

### evidence
- Current seven-AC canonical description, AC7 rationale/budget and paused oracle proposals read through supported pvg issues show MAC-2u36 --json.
- Read committed d9d53b9 finalization/seed/state/reference helpers, e1fd0a3 change statistics and preserved raw focused log aggregate; no author uncommitted work or mutable fixture touched.
- Graph Verify: project Users-ramirosalas-workspace-machinery ready, generation2026-09-05T20:28:41Z; exact recordHomeInstallLocked source and saveReceipt both-direction trace, plus local validateReceipt/saveReceipt/installLocked source. Relevant four source paths metadata_match/no_recorded_issue, best-effort caveat preserved.
- No runtime tests, product/source/test edits, installed assets, remote operations or branch/status/claim transitions in this review.

### proof
- [x] AC7 validated-publication/conflict policy and revised measured estimate reviewed within existing ownership.
- [x] Two precise non-weakening oracle repairs and bounded real CLI AC7 regression/positive controls authorized.
- [ ] Corrected final-SHA RED replay and full independent RED approval pending.
- [ ] AC1-7 implementation/acceptance and actual late close-fault rollback proof pending.

### 2026-09-05T22:30:44Z ramirosalas
## nd_contract
status: delivered

### evidence
- RED-only frozen98d3b573ac493c4d7252bffb03c83f92dc99f82b; internal/install/bootstrap_receipt_test.go SHA256ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23; production unchanged.
- Exact complete command: go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json, executed in /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-2u36 with stdout/stderr directly captured to /tmp/machinery-MAC-2u36-red.qRk5DC/98d3b57-final-matrix.jsonl.
-66leaf49PASS17FAIL0SKIP748.254s; every run has terminal outcome, no fixture/compiler/timeout/remaining oracle errors. Raw SHA256f2b15c4ef66ccd9b6f82c6d65795df4220364b767a4f71f92f265a7ab3a0947e.
- Full Implementation Evidence, focused commands, exact named leaf inventory, failure classification, real/helper boundaries, AC mapping,1111test-LOC budget investigation and LEARNINGS persisted in Notes.
- pvg verify PASS1file0issues; verify-tdd PASS9commits. Installed binary hash unchanged, no live product mutation. Canonical pvg story deliver was invoked ONCE successfully.
- This terminal comment preserves prior history and supersedes older in_progress comments that rendered after the deliver-generated Notes contract. First structural readback was8/9 solely for that ordering; no second deliver invocation.
- Independent full RED approval, GREEN implementation and actual late-parent-close fault/rollback execution remain pending; never-fired close hook is a missing-boundary RED assertion only.

### proof
- [x] AC #1: complete recorded-plan bootstrap parity specified; default/plugin/schema1/selector controls pass, parity defects fail.
- [x] AC #2: valid repair vs unsafe ownership distinguished; existing receipt validation controls pass, unsafe bootstrap symlink fails.
- [x] AC #3: actual complete release content/binary/topology/digests/plugin/idempotence assertions; ordinary intact/edited controls pass and missing/bootstrap defects fail.
- [x] AC #4: real post-mutation ordinary rollback passes, missing/bootstrap boundary failures retained, pre-mutation interrupted recovery separately passes.
- [x] AC #5: actual built CLI/releases/checksums/children/private loopback and temporary roots, no mocks/skips/live installation writes.
- [x] AC #6: parent publication sequencing/fault outcome contract specified with corrected independent oracles; authority/cleanup controls pass; parent publication remains RED and actual late close-fault execution is not claimed.
- [x] AC #7: exact and repeated cross-group rejection/rollback/next-native flow fails as intended; fresh/disjoint/same-ordered-group copy-change controls pass.


### 2026-09-05T23:04:15Z ramirosalas
FULL INDEPENDENT RED REVIEW: REJECTED — MAC-2u36, frozen 98d3b573ac493c4d7252bffb03c83f92dc99f82b. This is rejection of RED proof, not product acceptance or a GREEN implementation authorization.

EXPECTED: AC2 needs independently reached fail-closed proof for malformed/unknown/duplicate/trailing JSON, wrong types, invalid topology/inventory/digest encoding, supported-schema positives, and separate permission/type safety.

DELIVERED: Independent exact-SHA synchronous replay completed: go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json. Exit 1; 66 leaves, 49 PASS, 17 FAIL, 0 SKIP, 758.895s; every started leaf terminal. All outcomes match author replay exactly. New bootstrap suite has 43 leaves (26 PASS, 17 intended FAIL); selected existing tests have 23 PASS. Ten of those existing passes are not valid proof of their named parser/topology boundary: TestCorruptReceiptFailsLoudly (1), TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology (8), TestSemanticallyInvalidReceiptFailsBeforeUpdate (1). Their fresh write() fixtures use 0644 under actual umask 022; readReceiptFile rejects unsafe permissions before parsing/semantic validation, while each only requires err != nil.

GAP: Removing the intended parser/semantic validation would not make those ten tests fail because earlier permission rejection still satisfies them. The digest-change test tests digest computation, not malformed schema2 receipt inventory/digest-encoding rejection; attributable input validation coverage for those required categories was not established. Passing current tests therefore would not prove all AC2 outcomes. This is a pre-existing test-oracle defect, not a newly demonstrated production parser bypass or a fourth product defect. Prior logs remain accurate runtime results but the ten misleading passes must be qualified.

FIX: Apply ONLY the explicitly authorized scope below, establish genuine private matched-valid fixtures and exact intended negative reasons, then re-freeze and redeliver complete latest-SHA evidence for a separate independent RED review.

INDEPENDENT EVIDENCE:
- Detached checkout /tmp/machinery-pm-MAC-2u36.DDIEBx/checkout at 98d3b573ac493c4d7252bffb03c83f92dc99f82b; root remained main.
- Complete raw /tmp/machinery-pm-MAC-2u36.DDIEBx/98d3b57-independent-matrix.jsonl; SHA256 9457a41c68ebdc041dc4ffcfccc71a9026dd595b858349f8429e48b2cefba778.
- Full review and all 66 named leaf outcomes: /tmp/machinery-pm-MAC-2u36.DDIEBx/REVIEW.md.
- Frozen bootstrap test SHA256 ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23 verified before/after; checkout clean. Author raw SHA256 f2b15c4ef66ccd9b6f82c6d65795df4220364b767a4f71f92f265a7ab3a0947e independently verified.
- gofmt/diff check clean; pvg verify PASS 1 file/0 issues; verify-tdd against local epic f24b2df3cb1e1521f97b406a7516f72bb7bc7890 PASS 9 commits, 0 skipped merges. No statement/branch coverage instrumentation or percentage claim.
- Exact source, all 1111 candidate test lines, seven ACs, terminal delivery proof and latest canonical AC2 repair/budget read. Graph Verify project Users-ramirosalas-workspace-machinery ready, generation 2026-09-05T20:28:41Z, relevant paths metadata_match/no_recorded_issue; best-effort caveat retained.

OTHER AC REVIEW:
AC1/3 independent complete release content, binary, topology/digests/plugin obligations, preservation/idempotence assertions are substantive. Actual ordinary intact and edited controls passed (47.45s/47.06s); bootstrap/missing-file failures are desired behavioral RED. AC4 actual ordinary later-source post-mutation rollback passed 37.26s with binary/home/Codex progress and full prior state; interrupted recovery passed 17.09s and remains explicitly pre-mutation interruption proof. AC5 uses actual built CLI/releases/checksums/children and isolated temporary roots, no fake runner/output.
AC6 corrected component-bounded alias and independently prepared desired-receipt oracle are sound in the supported POSIX execution lane; no old alias/stale-expectation diagnostics recurred. Four authority controls passed. Four parent-finalization cases remain missing-boundary/child-receipt-sequencing RED: ZERO actual final-close faults fired. Do not claim late-close rollback execution. The exact prepared scope, parent-owned scratch payload, hook count/error, no-commit, restoration/absence, journal cleanup and reusable lock assertions must remain mandatory for GREEN. Preserve concurrency/foreign-postimage protection.
AC7 three real supported-recording controls passed. Two actual cross-group conflict flows fail desired rejection/restoration and subsequent native installs reject the emitted invalid receipt without manual repair. Independent exact topology/content/path/digest expectations and permitted early safe rejection are appropriate.
Budget: measured 1111 bootstrap-test lines and required shared oracle/AC7 proof justify the earlier 61-line test estimate overrun, not weakening assertions. Reviewed canonical approximately 1700 combined changed LOC (~1350 tests/~350 production, same 4-6 files), with bounded ~150-200 receipt-test additions. Report actual per-file additions/deletions, helper reuse and focused/full elapsed costs; material new overrun/scope needs review. No automatic acceptance cap.
Nonblocking out-of-lane observation: alias table literals are POSIX paths; Windows runtime would need portability work, but current contributor contract only cross-builds Windows and supports Linux/Darwin execution. No Windows runtime guarantee or result is claimed.

TEST-EDIT AUTHORIZED — narrowly scoped AC2 RED proof repair, after independent review of the latest canonical seven-AC-preserving repair:
1. Amend IN PLACE only internal/install/receipt_test.go functions TestCorruptReceiptFailsLoudly, TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology, and TestSemanticallyInvalidReceiptFailsBeforeUpdate, plus necessary local table fields/imports and minimal receipt-local private-fixture/diagnostic helper. Preserve every intended negative. No renaming is authorized; describe the semantic test accurately as loader rejection before planning, not observed binary-replacement ordering.
2. Explicitly create private 0700 config roots and 0600 regular receipts; assert actual existence/type/mode before loadReceipt. Establish matched valid schema1 and schema2 controls through the same loader/read path. Keep global install_test.go write helper and unrelated fixtures/assertions unchanged.
3. Each negative changes only the intended defect from an otherwise valid matched fixture and requires exists=true, a nonnil error and intended diagnostic category/detail: malformed JSON syntax; unknown root/home/target fields plus exact field names; duplicate root/nested keys plus exact keys; trailing JSON; wrong homes/copy types; semantic unknown target plus value. Unknown-home-field control must have valid nonempty safe absolute homes. Use typed errors where available or strip the exact known receipt-path wrapper before diagnostic matching: test directory/path words must not satisfy an assertion. Earlier permission/path/inventory errors cannot substitute.
4. Add bounded fast real-filesystem schema2 cases in receipt_test.go from a valid private matched receipt with seeded real topology, independently complete known inventory and real digests. Single-category cases: missing/extra inventory count; duplicate/substituted path at unchanged count; relative/non-clean/unexpected absolute path; digest wrong prefix/length/correct-length nonhex. Require corresponding inventory-count/path/malformed-digest detail with other fields valid. Preserve valid reversed input ordering as normalization compatibility, not invented rejection.
5. ALL internal/install/bootstrap_receipt_test.go bytes stay frozen at SHA256 ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23. Reuse its actual valid-0600-to-0666 unsafe_receipt negative UNCHANGED, separately from parser proof; its malformed-receipt case overwrites an existing 0600 file and is not the old fixture bug. No bootstrap/finalization/AC7 edits, shared-write edits, unrelated assertion weakening, new production API/hook/stub, instrumented child, environment bypass or production implementation is authorized.
6. Every amendment/addition commit subject contains BOTH tdd-red and [test-edit-authorized]. Preserve all commits/raw history and transparently reclassify ten old passes. Run focused named repaired/new receipt cases first, zero skips and exact diagnostic causes/matched controls. Correct safety controls may all PASS unchanged production; do not manufacture RED. Report any actual validator bypass for triage.
7. After focused repairs are stable, capture ONE complete latest-SHA raw JSON replay with the existing 15m package command plus explicitly named outside-selector tests if needed. Preserve individual operation limits, every required case and all intended product RED assertions. No deadline increase, omissions, retry masking or extra broad run before focused stability. Record SHA/hashes/all named outcomes/counts/costs. Separate independent full RED review and approval remain mandatory; this is NOT approve-red or GREEN permission.

## nd_contract
status: rejected

### evidence
- Independent complete exact-SHA replay: 66 leaves, 49 PASS, 17 FAIL, 0 SKIP, 758.895s; raw/report paths and verified hashes above.
- Blocking AC2 early-permission false-positive proof gap in ten existing leaves; current canonical repair and approximately 1700 LOC estimate independently reviewed.
- No source/test/installed-asset edits, remote operations, root branch switches or overlapping mutable fixtures during review.

### proof
- [x] AC #1: desired recorded-plan parity and positive defaults/schema1/selectors assessed.
- [ ] AC #2: parser/topology/inventory/digest preservation proof incomplete; exact locked-test repair authorized above.
- [x] AC #3: substantive actual release convergence and repair-success RED assertions, with intact/edited positive controls.
- [x] AC #4: actual ordinary post-mutation rollback and separately classified pre-mutation interrupted recovery verified; desired missing/bootstrap assertions retained.
- [x] AC #5: actual CLI/release/children and isolated no-skip execution verified.
- [x] AC #6: parent-sequencing/finalization desired contract and strict fault sensitivity reviewed; missing-boundary RED only, actual late-close fault rollback NOT exercised.
- [x] AC #7: real conflict/restoration/follow-on desired outcomes and supported positive controls reviewed.
- [ ] Revised RED proof approved; GREEN implementation and final product acceptance remain pending.

### 2026-09-05T23:26:33Z ramirosalas
## nd_contract
status: delivered

### evidence
- RED rework frozen496963fb7f4842d706d308dafcbd145908a6e395; only authorized receipt_test.go amendment +186/-25 since98d3b57. Receipt test SHA256a56a3dc07bd447a0e4d4469f0d027509e5149eb26b494d118d394089cbfb5c48.
- Bootstrap hash ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23 and shared install_test.go hash d5039079dd032f8b86f86ad6bd0ac411b192983b10ee194e599d6e59bcba97ba unchanged; no production edits.
- Focused23receipt leaves23PASS0FAIL0SKIP0.461s; intended private-read/parser/semantic/inventory/digest diagnoses logged with matched positives.
- ONE final latest-SHA command: go test -count=1 -timeout=15m ./internal/install -run 'Bootstrap|Receipt|UpdatePlan' -json in /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-2u36, native stdout/stderr directly captured to /tmp/machinery-MAC-2u36-red.qRk5DC/496963f-final-matrix.jsonl.
-79leaves62PASS17FAIL0SKIP739.525s, no missing terminals, exact17product failure set unchanged. Raw SHA256a42f96c5d36651fbd835d9626057eb3f659ae6b2503668d4dd6375407dce7b03.
- Full current Implementation Evidence/Commands run/Summary/SHA/budget/AC mapping/79leaf inventory/LEARNINGS persisted in Notes. Ten old permission-masked passes explicitly reclassified as historical runtime results, not valid old AC2 proof.
- pvg verify PASS2files0issues; hard-TDD PASS10commits0unauthorized edits. No runtime bypass discovered. Installed NIL binary remains5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849.
- Canonical pvg story deliver invoked once for this formally rejected/reclaimed rework. This append-only terminal comment supersedes older rejected comments after Notes; no repeat delivery invocation.
- Independent RED approval, GREEN and real late-parent-close fault/rollback execution remain pending.

### proof
- [x] AC #1: frozen complete recorded-plan/default/plugin/schema1/selector proof replayed; intended bootstrap parityRED.
- [x] AC #2: all23repaired/new private matched-valid and precisely attributed parser/topology/inventory/digest/order leavesPASS; unchanged CLI mode-only rejection separatelyPASS. Semantic test proves loader rejection before planning, not binary swap ordering.
- [x] AC #3: frozen real release content/binary/topology/digest/plugin/idempotence checks; intact/edited ordinaryPASS, desired bootstrap/missing casesRED.
- [x] AC #4: actual ordinary post-mutation rollbackPASS32.60s; missing/bootstrap boundary failures retained; pre-mutation interrupted recovery separatelyPASS14.67s.
- [x] AC #5: actual CLI/release/checksum/children/private roots, no mocks or executed skips; metadata additions use real filesystem.
- [x] AC #6: strict frozen parent publication/fault contract and4authority/cleanup controls retained; missing-parent publicationRED, no actual close-fault rollback claim.
- [x] AC #7: frozen real overlap/next-operation desired outcomesRED with3supported recording controlsPASS.


### 2026-09-05T23:33:52Z ramirosalas
FULL INDEPENDENT RED REVIEW: APPROVED — MAC-2u36 at496963fb7f4842d706d308dafcbd145908a6e395.

Canonical pvg story approve-red MAC-2u36 completed once. Fresh readback: open with hard-tdd, red-approved; delivered removed; no accepted label or closure. This terminal append corrects rendered ordering where historical delivered comments follow Notes; no repeat transition.

## nd_contract
status: new

### evidence
- Complete independent current seven-AC review and exact authorization/history audit preserved in appended Notes and /tmp/machinery-pm-MAC-2u36-final.EeD9Ho/REVIEW.md, SHA2565524442c86ab0d60a67b16215ae09a93dda85435f5ee23755257b8f0d71e2549.
- Fresh independent496963f focused23leaves23PASS0FAIL0SKIP0.441s, every26runterminal; raw /tmp/machinery-pm-MAC-2u36-final.EeD9Ho/496963f-independent-focused.jsonl SHA2564ce4f6378c140208a34a06eaecbbd40ddea7712bde0fbc26ffb538e0c96e13de. Private modes/type, matched positives, typed/exact diagnostic attribution and independent real inventory verified.
- Current complete author79leaves62PASS17FAIL0SKIP739.525s raw independently hashed/parsed: a42f96c5d36651fbd835d9626057eb3f659ae6b2503668d4dd6375407dce7b03; all89runterminal, exact17failure set unchanged. This is audited author-owned current integration evidence, not fresh full PM replay.
- Carried-forward independent98d3b57 full66leaves49PASS17FAIL0SKIP758.895s hash9457a41c68ebdc041dc4ffcfccc71a9026dd595b858349f8429e48b2cefba778 corroborates unchanged bootstrap/product/source/helper closure. Ten old permission-masked passes expressly NOT old valid AC2 proof. Revision/archive-byte identity is not claimed; current author full run covers current artifacts. No unresolved inconsistency warrants another broad run.
- Fresh verify2files0issues, hard-TDD10commits0unauthorized edits; frozen bootstrap ee786dee86c951a5a3a4a53d9344aeb16979df19319ea77dcd14fbd7afefca23 and shared install_test d5039079dd032f8b86f86ad6bd0ac411b192983b10ee194e599d6e59bcba97ba unchanged. Only authorized receipt_test+186/-25; total1322test changed lines/projected1672with350production within~1700.
- Real ordinary later-source postmutation rollbackPASS32.60s; interrupted recoveryPASS14.67s is premutation interruption. Parent final-close hook still not exercised: strict missing-boundary RED only. Actual close/fault/full rollback remains mandatory GREEN.
- No production edits, new bugs, installed asset changes, remote operations, author checkout mutations or root branch switch. Root main497419a and installed binary5205883aaa4276d7eb6edb25b6ad43ac39a04bcb9a8b5ee55498127b04950849 unchanged.

### proof
- [x] AC #1: complete plan/default/plugin/schema1/selector RED bar reviewed.
- [x] AC #2: prior blocking proof gap closed by fresh attributable23-leaf private receipt checks and preserved safety controls.
- [x] AC #3: actual complete release repair/convergence/content/modes/digests/idempotence RED bar reviewed.
- [x] AC #4: actual later-source rollback and separately classified interruption; missing-prestate desired contract preserved.
- [x] AC #5: real release/CLI/children/private roots, no executed skips verified.
- [x] AC #6: strict parent sequencing/exact scratch/real close fault/count/cause/restoration/authority/lock bar reviewed; actual late fault execution pending GREEN.
- [x] AC #7: actual conflict/restoration/follow-on-native and supported-recording controls reviewed.
- [x] Independent RED approved through canonical transition; story returned open/red-approved for GREEN queue.
- [ ] GREEN implementation, unchanged RED passing, actual final-close fault rollback, and final product acceptance remain pending.

LEARNINGS: Reached validation and matched private positives matter more than nominal PASS counts; distinguish fresh independent, current author and carried-forward evidence; never claim an unexecuted fault boundary as rollback proof.
