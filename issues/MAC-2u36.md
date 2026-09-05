---
id: MAC-2u36
title: "Converge installer reruns on recorded targets"
status: open
priority: 0
type: bug
labels: [hard-tdd, delivered]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T23:04:15Z
content_hash: "sha256:cb9779c6532fe8aa70dfa76d67cd07f06013eb718255efb42d36a29fb6042b06"
blocks: [MAC-gcrr, MAC-ou97]
follows: [MAC-olrx]
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
Read-only consumers unless a concrete need is separately reviewed: internal/install/targets.go, internal/install/transaction.go, internal/install/lock.go, cmd/machinery/install.go and cmd/machinery/update.go. No blanket CLI/target/lock/journal rewrite is included. You are not alone in the codebase; preserve others' edits and coordinate shared paths with dispatcher.

## Boundary Map
PRODUCES:
- internal/install/update.go -> complete recorded-plan convergence, including safely missing owned native artifacts, under the existing update transaction.
- internal/install/receipt.go -> validated complete receipt publication coordinated with successful full direct refresh.
- internal/install/install.go -> bounded child-refresh participation while ordinary install continues to persist validated receipts.
- internal/install/bootstrap_receipt_test.go -> real CLI RED/GREEN matrix for recorded-plan, receipt-publication and standalone overlap defects with preserved controls.
- internal/install/receipt_test.go -> focused receipt-publication regression proof plus accurately reached AC2 parser/topology/inventory/digest rejection and matched private-fixture controls.
- internal/install/install_test.go -> focused standalone-install regression proof if needed.
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

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Follows: [[MAC-olrx]]

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

