---
id: MAC-uzxr
title: "Bind Go CRM FSM conformance to committed oracle expectations"
status: in_progress
priority: 0
type: bug
labels: [hard-tdd, red-approved]
parent: MAC-ui8a
created_at: 2026-09-06T03:01:38Z
created_by: ramirosalas
updated_at: 2026-09-06T07:47:16Z
content_hash: "sha256:5e87b61c5c8c7d100bf5943c30352a6d8d480803be73c4a55380c32372b2bfde"
blocks: [MAC-ou97, MAC-hgz1]
follows: [MAC-2u36, MAC-a89e, MAC-p8ce, MAC-olrx]
assignee: dev-MAC-uzxr
---

## Description
## USER INTENT
A current example conformance claim must describe real tests bound to committed FSM expectations, not handwritten expectations that merely reuse stable row names.

## Context (Embedded)
Confirmed absent wholesale FSM parser in Go CRM's five transition suites. PM's actual replay passed 218 leaves (domain130/session60/cli28), zero failures/skips, with all197 committed IDs present in executed names. This proves bounded existing transition execution, not linkage to parsed current oracle expectations. Raw /tmp/MAC-lnu6-pm-go-crm-transitions-713184d.jsonl SHA256 8a88436b9d6212a45eb422c4f4d609902679a15343ba796b1c72174aff03a662; wall1.503486s.
internal/gates/attest.go:151 requires wholesale committed oracle parsing plus per-row next state AND expected actions. Five handwritten test tables call real Fire/State but do not parse the committed FSM oracle tables. Existing firedInOrder loops accept expected actions as a subsequence and any actual list for an empty expectation. Source establishes that adequacy gap; no extra-effect mutant was executed by the discovery review. Registry wording does not expressly require exact-list equality. Entry/exit effects are legitimate and MUST be reconciled from committed semantics rather than blindly replacing containment with transition-column equality.

## Confirmed Task entry-action defect and bounded GREEN repair
Runtime-confirmed on unchanged70652b948 production while planning AC2 reconciliation: TASK-754183 (rolledBack, always, fallback) reaches Cancelled but returns only recordRoutingError; committed Task.oracle.md and Task.machine.json require Cancelled entry recordTaskClosed after the transition action. Matched TASK-67b0ff normal persist-to-Cancelled returns commitStatus,recordTaskClosed and passes. Actual isolated probe command from /tmp/MAC-uzxr-task-entry-probe.FGj02Q/impl: go test -count=1 -timeout=120s ./internal/domain -run '^TestTaskOracleEntryProbe$' -json. Author's corrected pipefail execution exited1:1PASS/1FAIL/0SKIP,package0.286s. Initial tee pipeline status0 was only the pipeline wrapper status, not test success.
Verified raw probe-pipefail.jsonl SHA25690e3d11c3df38725d77637bc187d7e27dcf0a2b65f0934e6b71c785b177e783c; isolated task_entry_probe_test.go SHA256f6809da70ce0130f79020f8d032194c878e3b17acdeebfaa12f22ca54bc97e84; copied and committed task.go SHA25639af26a093f855acf269da153915fbecf6e78c0f3dbba85b087bf4b83586e35b. Task oracle SHA25670d3f0669117c85aef8c3a6e061df147f43c6ced036805c24eb200cfe36e0d56; machine SHA256ba243728cf359b02cc3d7cb0691eea2bd2b0a90a49c11a46846b820ac250d9b5.
Additional isolated copied-test reconnaissance tightened length/order and hand-reconciled Task/CommandExecution entry expectations across five transition suites plus TestTaskTerminalRejectsEverything and TestCommandExecutionTerminalExits:225 leaves224PASS/1FAIL TASK-754183/0SKIP,author-reported outer0.917s,actual exit1. Raw exact-effects.jsonl SHA256c6ce626ee381f97d020a2bea090d1a626e9b9adfa421c75ffd877cdfb8ee42a4. Its four copied-test diffs and raw outcomes were read; these handwritten diagnostic probes are NOT committed parser-backed tests, approved RED or completeness proof.
Same-P0 scope disposition: this tiny application mismatch directly blocks the story's required complete-effect conformance; repair it here rather than creating a duplicate owner/prerequisite. Only Task.fireRolledBack's fallback may change in production: retain recordRoutingError and Cancelled routing, then execute existing recordTaskClosed and report both actions in that exact order. Existing recordTaskClosed clears Rejection; returning its name without performing that real context mutation is not a fix. Do not change guard priority, valid prior routing, event filtering, final-state behavior, action definitions, public interfaces or other production functions.
Bounded exact-source inspection: Deal/User/Session declare no entry/exit actions; Task declares entry only for Done/Cancelled and normal persist terminal routes already call it; CommandExecution declares eight entry actions and its target transitions use existing enter handling. This establishes no additional missing-entry variant in that bounded review, not absence of all possible defects. Graph generation02:42:16Z reported metadata_match/no recorded gap best effort; receiver-name collisions required exact70652b948 source fallback.
No production edits are authorized during RED. Independent exact old-test/helper/reconciliation review remains required BEFORE those edits; only after genuine RED delivery and approval may the separate GREEN implementer make this narrow task.go repair. Any further concrete mismatch returns to Sr PM before expanding production scope. MAC-hgz1 and MAC-ou97 already depend on this story, so no dependency change; consumer uses the accepted parser tests AND this repaired Task source. MAC-lnu6 evidence hold remains unchanged.

## Ownership
Eight forecast paths (the prior seven test-support paths plus one narrowly limited production file):
- examples/go-crm/impl/internal/domain/deal_test.go
- examples/go-crm/impl/internal/domain/task_test.go
- examples/go-crm/impl/internal/domain/user_test.go
- examples/go-crm/impl/internal/session/machine_test.go
- examples/go-crm/impl/internal/cli/command_test.go
- examples/go-crm/impl/internal/testoracle/fsm.go (new test-support parser/reconciliation only)
- examples/go-crm/impl/internal/testoracle/fsm_test.go (new parser and real subprocess sensitivity proof)
- examples/go-crm/impl/internal/domain/task.go (GREEN only, fireRolledBack fallback TASK-754183 entry-action call and ordered Effect.Actions; no other production edits)
Only exact independently PM-authorized existing test/helper amendments may change the five old files. Preserve their real inputs/guard branch cases, terminal supplements and safety intentions. New test-support paths are reviewed outputs, not assumed existing APIs; concrete helper shape/use must be reviewed before authoring if it affects frozen assertions.
Read-only: five committed design/machines/{Deal,Task,User,Session,CommandExecution}.oracle.md and .machine.json files; production Fire implementations except the exact task.go fallback extension above; root Machinery parser/gate/attestation code; all BUILD/attestations/acceptance/go.mod/go.sum. No generated-oracle edits or fixture weakening to make tests pass. Unexpected production mismatch returns to Sr PM; no blanket implementation-fix ownership. You are not alone; preserve other work.

## Boundary Map
PRODUCES:
- Five exact transition test files -> actual Fire/State conformance driven by parsed committed row expectations with complete row/guard input reconciliation.
- internal/testoracle/fsm.go under the Go CRM impl -> bounded test-support parser and expectation reconciliation, not a new Machinery production API.
- internal/testoracle/fsm_test.go under the Go CRM impl -> matched real native controls and adversarial parser/assertion sensitivity.
- examples/go-crm/impl/internal/domain/task.go -> existing func (t *Task) Fire(evt TaskEvent) Effect delegates to func (t *Task) fireRolledBack(evt TaskEvent) Effect; corrected TASK-754183 preserves Cancelled routing and executes existing func (t *Task) recordTaskClosed() after recordRoutingError, with actual context change and ordered Effect.Actions.
CONSUMES:
- Existing Go CRM transition interfaces.
  source: TestDealTransitions/TestTaskTransitions/TestUserTransitions/TestSessionTransitions/TestCommandExecutionTransitions instantiate real domain/session/CLI machines, call Fire on concrete events and inspect State and Effect.Actions.
- Existing committed oracle grammar and semantics.
  source: each .oracle.md contains State entry / exit actions plus Transitions columns test id, stable id, source, trigger, guard, target, actions; corresponding .machine.json defines ordered guarded alternatives, internal/external transitions and entry/exit actions. Parse these actual committed sources, not matrix citations only.
- Existing claim vocabulary.
  source: internal/gates/attest.go gt.conformance-test-shape requires wholesale parser, next state and expected actions; no core claim-model edit here.

### Story Acceptance Criteria
1. All five FSM conformance suites parse the committed oracle tables during real execution and derive expected next state/actions from them, with an explicit closed mapping from each current row/guard branch to actual setup/event inputs. Check source/trigger/guard/target identity, not only stable-ID string membership. Missing, duplicate, malformed, newly added unmapped or unused rows fail with named diagnostics; preserve complete existing guard-clause coverage and additional legitimate supplement cases.
2. Reconcile full observable action expectations using the actual committed transition AND entry/exit semantics, including internal transitions and fallback/guard priority. Independently review the concrete reconciliation before freezing revised assertions. Correct legitimate entry/exit actions pass; missing, reordered, duplicated or unrelated extra effects fail, including an empty-transition-action row. Do not claim this exactness was an explicit old registry quotation; it is required adequacy proof for this repaired current claim.
3. Real native subprocess sensitivity on isolated copied source/oracle fixtures demonstrates that changing committed expected next state or actions without changing implementation causes the intended conformance failure, not parse/setup failure. Separately inject a reviewed extra-effect implementation variant and require the same frozen assertions to reject it, with unchanged implementation/oracle controls passing. No returned fake process result, warning-only negative, modified shared oracle or production fault API.
4. Report bidirectional committed-row versus actual executed-row inventory for all five machines; current baseline197 IDs/218 leaves is historical, not a hardcoded ceiling. All selected rows/guard witnesses terminate with expected real outcomes and zero skips; inspect real next-state/action diagnostics and preserve terminal supplements. A passing parser unit test or named-ID count alone is insufficient.
5. The actual focused native Go CRM suite and parser/sensitivity proof pass on the delivered candidate, old unrelated tests remain unchanged, and the consumer migration can use the exact reviewed source/test scope and logs to assess a current claim. No automatic attestation, implementation acceptance renewal, full-suite or authenticated-execution assertion is made by this story.

## Testing Requirements
Unit tests plus Integration tests: MANDATORY (no mocks of Fire, compiler, filesystem, process output or assertion result). Service-free native Go and real temporary copies/subprocesses; no Docker or Paivot dependency. Review exact old-test amendments before RED writes; test-support scaffolding may support a qualifying RED only when the unchanged old suites demonstrate the specific missing parser/sensitivity behavior. Compile/API/fixture/timeout failures are not RED.
AC2 mandatory regression details for the confirmed fallback: revised task_test.go coverage must derive expected actions from the committed transition plus entry semantics, assert Cancelled and exactly [recordRoutingError,recordTaskClosed], and assert actual Rejection clearing. Matched TASK-67b0ff control asserts [commitStatus,recordTaskClosed] plus real entry behavior; retain TASK-c56bd7 Done, TASK-3f585f/ TASK-98c3ba valid prior branches, non-always no-transition and terminal-event controls. No new standalone test file or production fault seam is needed. The exact assertion/setup changes in the already owned task_test.go must appear in the independent PM amendment proposal before authoring.
Keep evidence classes separate: the discovered fallback assertion must genuinely fail unchanged70652b948 for missing entry behavior; it is NOT proof that the absent parser is detected. Original AC1/AC3 still require independent intended parser/oracle-mutation/extra-effect RED with matched valid controls; choose unaffected scoped controls or classify the known fallback failure explicitly, never relabel an already-broken full control as passing. GREEN must pass the complete frozen suite on the candidate and reject a real isolated regression that removes the entry call or only fabricates its returned name; reuse bounded sensitivity infrastructure, with precise reviewed mutation and no shared-source mutation. This extends the existing missing-effect adequacy proof, not permission to alter frozen assertions after production implementation.
Focused existing control from examples/go-crm/impl: go test -count=1 -timeout=120s ./internal/domain ./internal/session ./internal/cli -run '^(TestDealTransitions|TestTaskTransitions|TestUserTransitions|TestSessionTransitions|TestCommandExecutionTransitions)$' -json
Run new internal/testoracle tests explicitly plus relevant existing terminal supplements. Bound outer meta-tests and avoid recursive selection of themselves. Propose exact new names/commands and mutation hunks for independent RED review; freeze unchanged-control and unsafe variants separately with SHA/input hashes. No hidden skip, generated expected-output replacement, or acceptance based on one error. Record host/toolchain, exact source/reference/mutant SHAs, commands, leaf/row inventories, logs and durations.

## OUT OF SCOPE
- Core plan/current/historical schema and freshness custody belong to MAC-p7jd; future authenticated runner/replay enforcement belongs to MAC-l7m0/MAC-vx24.
- BUILD/attestation consumer migration is a separately sequenced repair; no simultaneous ownership of MAC-lnu6 files.
- New application behavior beyond the exact committed TASK-754183 entry-action correction, root parser/generator edits, external services and global preflight.

## DIFF BUDGET
- Eight ownership paths and all five AC remain unchanged. Measured external proposal against 4ef6baffc4de72f2a3e935348f180283fe53c20b: seven paths, 2120 additions plus 330 deletions = 2450 changed LOC (net1790), including a NONQUALIFYING 48-line scaffold. Initial RED485 additions plus storage repair14 additions/4 deletions =503 cumulative historical changed LOC; proposal plus history totals2953 cumulative changed LOC BEFORE GREEN. Aggregate proposed tree against epic70652b948 is2615 additions/330 deletions =2945 changed LOC (net2285), avoiding double-counted storage history.
- Measured decomposition: five suite amendments1228 changed LOC (436 for218 tuple substitutions and792 for typed witnesses/wrappers/exact-effect/context/inventory checks and supplements); additive frozen parser/inventory/mutation tests1174; compile scaffold48. Full source breakdown and evidence limits are in the fully read /tmp/MAC-uzxr-amendment.QYZfzX/REVIEW.md SHA2567105508cf12fa8ea73bfd9db81d97dc5a068ad08db1cae39166a13a7dca8fdc6; full unapplied patch SHA2562f722fa783fa4a27cbbd7ec92ad706e7bf86079801b0d80d3fe0a8c03f07e249.
- Old750-1150 and provisional1500-2200 forecasts are obsolete cost history. Conditional planning forecast approximately3300-3500 cumulative changed LOC if the prior300-450-line bounded parser estimate holds: GREEN must additionally delete/replace the48-line scaffold and implement the narrowly owned Task fallback call. Approximately3200-3400 aggregate final tree changed LOC under that assumption; neither range is a measured final result or a cap. Report actual parser additions/deletions and Task delta separately; any scope/API/path addition returns for review. Do not trim mandatory proof, invent parser implementation credit, or treat this forecast as exact test-edit authorization.
- Existing focused baseline1.503486s. Preserve the current5m outer/120s child Go bounds and approved context deadlines; actual cost remains to be measured. The proposal forecasts20-45s cached for43 sequential children across three nonrecursive subprocess families; this is not executed final candidate timing or authorization to expand deadlines. Compile-only/builder diagnostics remain nonqualifying. Independent exact technical/test-amendment review, final RED freeze/approval and GREEN proof remain separate pending stages.

## Discovered During
MAC-lnu6 independent substantive amendment review at 713184db16a12b8c3763b4aa8f5bf025721abf22. Full report /tmp/MAC-lnu6-PM-AMENDMENT-713184d.md SHA256 6fc5223d3e5d48170f89ff9e29b634b256bfe2454b710378f08cb60d875aadf1 was read completely (227 newline-terminated lines; final content included). The original review's no-runtime-mutation limit remains historical; the later isolated Task and exact-effects probes above are now explicitly reported but do not establish approved parser RED. Exact epic 70652b948 source inspected; graph generation 2026-09-06T02:42:16Z is best effort, not completeness proof.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Hard TDD with independent RED and exact existing-test amendment review before edits. Preserve all unrelated tests, goldens, generated evidence and existing claims. Shared tracker is development coordination only; product remains standalone. No remote, full preflight, installed assets, user services or healthy-worktree cleanup. No source/docs/tests changed during triage. Use supported delivery; independent PM accepts.

## nd_contract
status: new

### evidence
- P0 source-established conformance parser defect plus separately identified action-adequacy gap; parent supplied independently verified native218-pass control. No new mutation run or repair performed.
- Existing claims/tests/ownership remain held until independent exact test amendment review; no core schema duplication.

### proof
- [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
- [ ] AC #2: correct entry/exit-aware complete effects and adversarial sensitivity.
- [ ] AC #3: real expected-state/action and extra-effect unsafe variants fail for intended causes.
- [ ] AC #4: complete executed-row inventory and preserved controls.
- [ ] AC #5: native candidate proof without automatic claim renewal.



## Acceptance Criteria


## Design


## Notes
TASK ENTRY-ACTION SAME-P0 CANONICAL EXTENSION
PRE-RED PM SCOPE DISPOSITION — MAC-uzxr
Bounded initial authoring authorization only; no delivered-state review, rejection, approve-red, acceptance, close or GREEN authorization.
External complete review /tmp/MAC-uzxr-PM-scope-review.md SHA256 a90e7be8958cbb692f4d7595889d67023733b4eb3c58878012bb2a9e41930202. Full134-line proposal SHA256cee9d4a6ebd7cd9e920bc32260d02bf07906923f13b415cf9ef4b0727ea4363b and entire1323-line appendix SHA256d9976dcd961d178e3fc7e1e68059125e592a152cb481e602200c58c1eb8bfb36 independently read.

Authorized initial stage ONLY new examples/go-crm/impl/internal/testoracle/fsm_test.go, stdlib real copy/subprocess support and exactly USER-e20d04 paired target/action variants plus pinned Session fireAnonymous/SEvResume extra-effect variant. Use full unaffected TestUserTransitions child for User and TestSessionTransitions child for Session, identical passing-control/mutant selection, genuine unsafe-native-variant-accepted failure on unchanged old suites; no missing API/parser-log/setup/compile/timeout RED. Separately record all5 original suites plus2 terminal baseline. Sequential125s context/120s Go children,270s context/300s Go outer; SHA manifests, actual native terminal inventory and exact intended diagnostic. Commit tdd-red then stop for independent replay; no complete-story RED approval. No fsm.go, old test or production edits at this stage.

All218 appendix tuple-to-RowID conversions independently reconciled as mechanical later scope, not permission to partially edit old suites now. Task specific seed/Rejection checks and3-prior non-always supplement are acceptable scope; prepare exact later diff. HOLD shared five-suite/helper/scaffold/parser-test writes until full external patch shows every actual event/guard and falsifying-clause expression, closed relevant guard keys/priority, JSON positional metadata grammar, concrete effect assertions, registered-versus-real successful execution inventory, exact remaining CLI/structural/Task mutation hunks and per-file cost. Both original terminal supplement bodies stay byte-identical. No parser or inventory assertion authoring authority inferred from conceptual paragraphs.
First fsm_test.go contents freeze at initial tdd-red. Any later additive same-file patch requires separate exact PM authorization before writing, TEST-EDIT AUTHORIZED record and [test-edit-authorized] commit marker, preserving every first-stage byte/assertion. None is granted for an unwritten future patch here. Independently review/freeze final combined RED before GREEN. Scaffold failures remain nonqualifying and original semantic RED stays replayable. Task missing-call/name-only mutation requires actual corrected candidate anchor and passing corrected control after GREEN; no synthetic fixed baseline proof before GREEN.

Cost recommendation only, canonical unchanged by PM: 807 existing test lines;218 tuple replacement lines alone436 additions+deletions. Strict dual parser, closed witness builders and native mutation proofs justify SrPM forecast review. Approx decomposition1500-2150 changed LOC; recommend provisional1500-2200 across same8 paths or a concrete measured1250-1750 decomposition covering all AC. Clarify changed LOC vs net additions, preserve all proof, no extra paths/architecture. Initial5m runtime budget unchanged pending measurement.
INITIAL_RED_REPLAY: VALIDATED — MAC-uzxr initial checkpoint only.
Independent report /tmp/MAC-uzxr-PM-initial-red-replay.md SHA25609291b1d33e9a7c77981c76f2cba2b01415eed3695de5d44f5c650fd31d119b6.
Exact998a5a3b3107292365c7c3e6ddb85c072c534145, only485-line fsm_test.go SHAd17a16229e9139d4f9c72e9d2f6fc04801d1b7f7246ac58ffbb092f04656d396. Complete source and author report read. Detached PM checkout /tmp/MAC-uzxr-pm-initial-red.csxQrh/review clean; static pvg verify1file0issues, gofmt/diff-check clean.
Independent actual native replay exit1, toolwall3.121269333s, exactly3 intended unsafe-native-variant-accepted enclosing failures and200 childPASS/0FAIL/0SKIP. Log /tmp/MAC-uzxr-pm-initial-red.csxQrh/replay.jsonl SHA256f8e10d43309778babbe609643c2adde672ad09c87305470745b706b5200dc67b. Separate original suite+supplements baseline actualexit0,225PASS/0FAIL/0SKIP,wall0.86440525s; baseline.jsonl SHA256ee99c3aa35061bfc763ad6566a616920e98327b79f06e1637898fde8874f4ea1.
Commands from exact detached examples/go-crm/impl used GOWORK=off GOPROXY=off GOTOOLCHAIN=local, pipefail, Go300s meta/120s baseline. Six children complete full User20/User20/Session60 inventories for each control+mutant, all exit0. Independent verifier matched author and PM receipts/raw hashes, exact package/run/terminal identities, all46 input hashes per control against committed git objects, exact2/2/1 mutation-only files, all child fixture cwds removed, and both225 baseline inventories equal218 appendix labels+7 original supplement identities. Retained PM proof /var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/crm-fsm-native-proof-807612870.
This is genuine unsafe-variant acceptance RED, not parser execution, successful negative rejection, Task repair or full AC evidence. No full approve-red/deliver/accept/reject transition.

TEST-EDIT AUTHORIZED: examples/go-crm/impl/internal/testoracle/fsm_test.go -- EXACT storage-only repair against998a5a3; author may change ONLY these3 regions in a separate [test-edit-authorized] commit, preserving historical998a5a3 and raw RED evidence:
1. Add stdlib import "flag" in gofmt order.
2. Immediately before var nativeUserLeaves, add:
var nativeProofParent = flag.String("fsm-native-proof-dir", "", "Retain native proof under this existing absolute directory (default: clean test temporary output)")
3. In TestFSMNativeOracleSensitivity replace only the existing proof, err := os.MkdirTemp("", "crm-fsm-native-proof-"), its if err block and retained-directory t.Logf statement with:
proof := t.TempDir()
retained := *nativeProofParent != ""
if retained {
    if !filepath.IsAbs(*nativeProofParent) {
        t.Fatal("fsm-native-proof-dir must be an existing absolute directory")
    }
    proof, err = os.MkdirTemp(*nativeProofParent, "crm-fsm-native-proof-")
    if err != nil {
        t.Fatal(err)
    }
}
t.Logf("native proof directory: %s; retained=%t; toolchain=%s %s/%s", proof, retained, runtime.Version(), runtime.GOOS, runtime.GOARCH)

Flag owned solely by this test file controls evidence storage, NEVER execution gating/skips/selection/expectations/oracle versions. Both valid modes unconditionally run all3 cases unchanged. Default uses testing.T.TempDir automatic cleanup/error reporting; do not swallow cleanup errors in a custom RemoveAll. Explicit export requires existing absolute parent and retains only the owned unique proof directory. Invalid parent is visible output setup failure, never semantic RED. All original fixtures already use t.TempDir and remain unchanged. No historical proof deletion or pruning authority. PM measured197672bytes/21 retained files per default invocation; unconditional accumulation justifies this bounded lifecycle correction but does not invalidate initial RED.
All mutation bytes, nativeUserLeaves/nativeSessionLeaves, cases/diagnostics/assertions, nativeRun/nativeCheckRun/nativeManifest/nativeCopy, command/timeouts and result logic byte-identical. No unrelated later parser/helper/additive amendment in this repair.

Required finite actual verification after repair: run same meta command once default, once with -args -fsm-native-proof-dir=<existing-absolute-review-directory>; both actualexit1/3 intended outerFAIL/200 childPASS0SKIP. Default emitted proof path must disappear after process ends; explicit mode21 evidence files remain and six receipts/logs/46input manifests/exact2/2/1 mutation-only files are checked. All fixtures clean in both. Same env/pipefail; logs and repairedfile hashes, actual runtime/exits, only-authorized-diff, gofmt/diff/scoped pvg verify. Cleanup errors are real failures. Commit separate [test-edit-authorized] repair, then independent verification checkpoint; full RED still held.
STORAGE_REPAIR: VALIDATED — MAC-uzxr bounded checkpoint only.
Independent report /tmp/MAC-uzxr-PM-storage-verification.md SHA256b52cda2189988e9516cc10c59cda850e39f82502405eaec75ba0fdcc15bbd184.
Candidate4ef6baffc4de72f2a3e935348f180283fe53c20b directly descends from original998a5a3b3107292365c7c3e6ddb85c072c534145. Subject carries [test-edit-authorized]. Solefsm_test.go14add4delete/18changedLOC/net+10; SHA256326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba. Complete byte-equality reconstruction against original proves ONLY3 authorized transformations; all semantic assertions, mutations, native names and helpers/commands/deadlines unchanged. No later additive parser/test amendment included.
Detached /tmp/MAC-uzxr-pm-storage.Mxluug/review exactcandidateclean. pvg story verify-tdd --base70652b948bf090008b1965c85daf36ea374daea4 --json checked2commits/no violations, normal audit/no waiver; scoped pvg verify1file0issues; gofmt/diffcheckclean. Graph readygeneration02:42:16Z with branchfilemissing; full exactsource fallback, best effort only.

Independent real default and export meta commands from detached examples/go-crm/impl, GOWORKoff/GOPROXYoff/GOTOOLCHAINlocal, pipefail, Go300s. Default actualexit1/wall3.492042875s; export actualexit1/wall3.062419291s. EACH3 intended unsafe-native-variant-accepted outerleafFAIL,200 checkedchildPASS0SKIP; no setup/timeout credit.
Default /tmp/MAC-uzxr-pm-storage.Mxluug/default.jsonl SHA2569ac83cb0f357a4aa7d529872b3ba85cd60e38a5ab19178180bf036dcca6a6b8e. Emitted proof /var/folders/gh/7c54cw2s52v6q6czy3xqhb_w0000gn/T/TestFSMNativeOracleSensitivity4153478124/001 absent afterexit; all6fixturecwdsabsent. Default outerlog retains actualexit0 child summaries and checked20/20/60 controls+acceptedmutants. Raw defaultchildlogs intentionallycleaned; no retained raw per-name defaultaudit claimed.
Export /tmp/MAC-uzxr-pm-storage.Mxluug/export.jsonl SHA256bc7148f9887ef4429231d484cd565e622c299cef4496a44b00b17ff260cb6076; proof crm-fsm-native-proof-4283665849 under same root retained21files. Independent git-object/read-only audit checked6receipts/rawhashes/exactcommands/package/name RUN/PASSonce/packagecompletion; all46 control input hashes/case; exact2/2/1 mutationonly files from unique anchors; allfixturecwdsclean.
Read complete author checkpoint/audit.cjs/audit.jsonl and verified all suppliedhashes. Did NOT execute authorscript accessing developerworktree. Independently checked author default/export lifecycle/outcomes and retained6receipt/native/manifests against candidate gitobjects; matched report. OriginalRED/evidencepreserved.

No further storage repair required. Author may prepare complete EXTERNAL later patch and measured per-file decomposition for separate exact PM/SrPM review. Applying all5suite/helper/parser/additive amendments stays HELD. No fullapprove-red/deliver/accept/reject transition; no new test-edit permission, AC/path/scope change, candidate parser/Taskfix proof or consumerclaimrenewal.
SR PM MEASURED FORECAST / PRESERVATION REPAIR. Canonical eight paths and five AC unchanged; actual external proposal2450changed plus503historical=2953 cumulative beforeGREEN, aggregate2945changed. Conditional3300-3500 cumulative/3200-3400 aggregate forecast assumes prior300-450-line parser estimate and separately charges48-line scaffold replacement plus narrowTask correction. Old750-1150/provisional1500-2200 obsolete; no proof trimmed or test permission granted. First --description command succeeded but verification failed because nd treats same-level nested headings as section boundaries and retained an old canonical suffix. Original failed readback preserved in /tmp/machinery-private-triage.JF2BDG/READONLY-HANDOFF.md. Installed --body-file has the same Description semantics. Root-authorized guarded pvg nd edit bound full live raw header/body to journal export and checked an external apply_patch trial, removed ONLY duplicate canonical block under nd exclusive lock, then verified exact full body90c15690a83bd2200723fdaa8bdd27e3871e189959b4936083afe06336d35590 and identical tail/metadata except normal hash/time. Exact expected/proposed manifests and editor remain external. First hgz1 preparation-only cross-realm assertion failed before any tracker write, then root explicitly authorized scalar-array comparison correction; historical failure is not proof. Scoped structural verification after exact canonical repair: pvg lint --backlog --epic MAC-ui8a scanned37 issues,0 errors/0 review findings; pvg rtm check37 stories/19closed/0 extracted requirements; pvg nd dep cycles found none. These are structure checks, not AC proof. Claims/status/labels/dependencies retained; no source/test/example/toolchain/remote/preflight mutation or Paivot product dependency.
EXPANDED RED FREEZE CHECKPOINT: exact authorized seven-path patch2f722fa783fa4a27cbbd7ec92ad706e7bf86079801b0d80d3fe0a8c03f07e249 committed acf1bba2d37784fa175b5ee2cdd0f9b5ac0d5c3f over4ef6baf with literal [test-edit-authorized] and tdd-red. Full report /tmp/MAC-uzxr-final-red.nOXYuf/REVIEW.md SHA2565cb98377240aa8be22f0b9d88b74a7a6bd959fbfd8f1efb2907d08b02f68ad5c. Exact seven result hashes/frozen495prefix/99committedexamplefiles/218originaltuples/twoterminalbodies verified; Task unchanged. Measured2120add330del2450changed/net1790; historical503changed separate. Actual all10 contractroots and focusednative plus3meta families ran15.526s: contracts0PASS28FAIL0SKIP, focused10supplementPASS5transitionrootFAIL0SKIP, original/candidate/remainingmeta3/3/17outerFAIL. All current parser/control failures NONQUALIFYING ErrScaffold; all intended later mutants/most nestedcontractbranches unreached. Actual23children10supplementPASS25transitionrootFAIL0SKIP,0transitionrowleaves/0mutants; all23fixturecwdsremoved. Export52files=46receipts/rawlogs+6manifests47committedinputhasheseach;17latercontrols fail before manifest export, no nonexistent manifest credited. Raw and complete native identity receipts in report. Normal verify-tdd3commits0violations/no waiver; scoped static7files0issues; gofmt/diffcheckclean. Original genuine998a5a3 threeunsafeacceptedouterFAIL200childPASS, validatedstorage4ef6baf bothmodes, historical225baseline and separate actualTaskentrydefect preserved as distinct evidence. No parserimplementation, productionedit, source/testcorrection, delivery, approve-red, GREEN, consumerclaimrenewal, remote/toolchain/service changes or background process. STOP clean for independent combined-evidence RED adjudication.
## Implementation Evidence (DELIVERED)

PROOF:

### CI/Test Results

Commands run:
- `cd examples/go-crm/impl && GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=120s ./internal/testoracle -run '^TestFSMParseExactFieldMembership$' -json` (pre-fix RED: 24 leaves, 14 PASS / 10 intended FAIL / 0 SKIP; every failure was an accepted composite structural key).
- `cd examples/go-crm/impl && GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=300s ./internal/testoracle -json` (PASS; full parser, native-sensitivity, inventory, cleanup and 20-mutant meta families; 23.105s).
- `cd examples/go-crm/impl && GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=120s ./internal/domain ./internal/session ./internal/cli -run '^(TestDealTransitions|TestTaskTransitions|TestUserTransitions|TestSessionTransitions|TestCommandExecutionTransitions)$' -json` (PASS; 228 native leaves: 218 oracle witnesses plus 10 terminal supplements; 0 FAIL / 0 SKIP).
- `cd examples/go-crm/impl && GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=120s ./internal/testoracle -run '^TestFSM(ParseCommittedMachines|Effects|Bind|ParseRejects|JSON|Invoke|OneFire|Load|Parser|Nested)$'` (PASS; 68 preserved parser contracts).
- `pvg verify examples/go-crm/impl/internal/testoracle/fsm.go` (PASS: 1 file, 0 issues).
- `pvg story verify-tdd --range 70652b948bf090008b1965c85daf36ea374daea4..HEAD` (PASS: 6 commits, no unauthorized test edits).
- `git diff --check 70652b948bf090008b1965c85daf36ea374daea4..HEAD` (PASS).

Summary: repaired regression is green: 24/24 PASS, 0 FAIL, 0 SKIP. The complete testoracle candidate run PASSed in 23.105s; direct native run PASSed all 228 leaves; preserved parser run PASSed 68 contracts. Coverage: prescribed regression and behavior coverage executed; statement/branch coverage not measured.

### Commit

- Branch: `story/MAC-uzxr`
- SHA: `bb5205c32e7b17ebefc5b050b7bf7417b0720e72`
- Scope: `examples/go-crm/impl/internal/testoracle/fsm.go` only; exact string-token membership replaces substring matching. No Task change.

### Custody

- Six original frozen test hashes, supplemental field-membership test hash `201a470a38915709aab457daa4d94011cbaea873428b05a21c2d6e8b6754583b`, original 495-line fsm_test prefix, and Task hash were rechecked unchanged.
- `git diff --check` clean; hard-TDD guard passed. No frozen test was edited.

### pvg verify

- `VERIFY: PASSED (1 files scanned, 0 issues)`

### AC Verification

| AC | Requirement | Evidence | Status |
|---|---|---|---|
| 1 | Exact closed structural-field membership | Public Parse rejects all ten composite unknown keys at root/state/transition/invoke while declared data controls remain unchanged | PASS |
| 2 | Preserve reconciliation and existing Task behavior | Full parser/native run passes; Task source hash unchanged | PASS |
| 3 | Preserve sensitivity proof | Full testoracle run passes native expected-state/action and extra-effect sensitivity families | PASS |
| 4 | Preserve executed inventory | Direct five-suite run passes 228 leaves; full inventory meta family passes | PASS |
| 5 | Fresh candidate proof and deliverable scope | Focused parser/native/meta runs pass, frozen custody and static/TDD checks pass | PASS |

LEARNINGS:
- Space-delimited allowlists need token equality; boundary substring matching admits composite keys.
- Structural-name validation must remain scoped to structural objects so metadata/context data keys stay inert.
- The supplemental frozen public-Parse regression makes this parser boundary directly auditable.

# MAC-uzxr independent supplemental RED review

Disposition: SUPPLEMENTAL RED APPROVED for exact candidate d3d257bd199801d88bdcce8ac8f89fdb8f6bd64c, not whole-story acceptance or GREEN implementation approval. New freeze: examples/go-crm/impl/internal/testoracle/fsm_field_membership_test.go,170lines,SHA256201a470a38915709aab457daa4d94011cbaea873428b05a21c2d6e8b6754583b. The original six frozen files and495-line prefix remain locked. No new test edit, grammar exception, production text or additional path is authorized by this disposition. Root must route separate GREEN within the current canonical scope.

## Implementation Evidence

### Commit

SHA: d3d257bd199801d88bdcce8ac8f89fdb8f6bd64c. Subject: test(MAC-uzxr): tdd-red -- require exact structural field membership. Only delta from rejected dc3a8a365c21d5370c0825de992e2461ea0785a9 is the one added170-line test. Full100-file exact Git archive audited; all99 previous example files, parser,Task,modules,oracles and old tests are byte-identical. No implementation exists for the new fix yet.

### CI/Test Results

Commands run:
- Own external exact Git archive, cwd /tmp/MAC-uzxr-PM-supplemental.DsuF62/examples/go-crm/impl: GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=120s ./internal/testoracle -run '^TestFSMParseExactFieldMembership$' -json
- Same cwd/environment: go test -count=1 -timeout=120s ./internal/testoracle -run '^TestFSM(ParseCommittedMachines|Effects|Bind|ParseRejects|JSON|Invoke|OneFire|Load|Parser|Nested)' -json
- Repository root: pvg story verify-tdd --range 70652b948bf090008b1965c85daf36ea374daea4..d3d257bd199801d88bdcce8ac8f89fdb8f6bd64c
- Repository root: pvg verify /tmp/MAC-uzxr-PM-supplemental.DsuF62/examples/go-crm/impl/internal/testoracle/fsm_field_membership_test.go --include-tests --format text
- Repository root: pvg story verify-delivery MAC-uzxr; git diff --check dc3a8a365c21d5370c0825de992e2461ea0785a9 d3d257bd199801d88bdcce8ac8f89fdb8f6bd64c
- Repository root: node /tmp/MAC-uzxr-PM-supplemental.DsuF62/check.cjs (exact Git source and actual PM/author leaf/diagnostic audit).

Summary: new regression24terminal leaves14PASS/10intendedFAIL/0SKIP, actual Go exit1,634ms. All68 original parser leavesPASS/0FAIL/0SKIP,exit0,354ms. Both finite Go commands completed; wrapper timeout130s, Go120s, Go1.27.1 darwin/arm64, ordinary existing build cache, -count=1 excludes test-result reuse. No compile/import/setup/timeout/stub/fake-process failure counted as RED. Static1file0issues,TDD5commits0violations,diffcheckclean. Delivery validator9OK/0FAIL verifies current record shape only; previous R2 failure remains historical and final GREEN must satisfy it again.

Coverage:100% of prescribed ten malformed composite-key cases, four ordinary-unknown controls, raw/remarshaled controls and seven individual declared metadata/context positions plus combined control. All68 preserved original parser contracts executed. Not statement/branch coverage or a fresh native218-witness/43-child claim.

### Exact outcomes and independent review

All new170lines were read. Tests call public Parse on actual committed Session MD+JSON. Deterministic targets are root,states.Anonymous,states.Anonymous.on.login,states.Authenticating.invoke; all exist with required object types in exact committed JSON. Each negative freshly decodes the committed file, checks the key absent, inserts one arbitrary object, marshals/redecodes and checks exact insertion. Each independent subtest requires nil Suite and exact `oracle-parse: unknown field KEY`; nil error fails immediately. No copied parser, helper factoring, source-pattern assertion, source mutation, setup bypass or weakened diagnostic.

All ten actual failures report60rows,error=nil and the exact scope/key:
- root: id initial; _comment _role; context states.
- state: on after; type entry; invoke _refusal.
- transition: target guard; guard actions.
- invoke: src input; onDone onError.

These exactly reproduce R1 from independent rejected-candidate report /tmp/MAC-uzxr-PM-green.7BaxI6/REVIEW.md SHA256dec4dd53d5f57dac813c9290fced9c4b366484fd8fc027da6ed8d695a3d2b74a. Four notAllowed negatives PASS at the same scopes with the demanded exact diagnostic. Raw committed and remarshaled positives PASS. Eight valid_data controls PASS:comment,role,delays,counters,refusal,invoke_input,context,combined. Each valid control compares complete Name,Rows (all identities and ordered actions),States (all names/kinds/entry/exit) with the actual committed control; the original five-machine direct-cell contracts also pass. Structural-looking/composite keys occur as legitimate metadata/context data, with nested object/array/null/bool/number/string content and deliberately unresolved expression strings. This prevents a global-name/underscore ban or evaluation-based workaround. No allowed-field/API/dialect expansion.

Actual run/terminal identities and complete per-leaf output were independently audited in both PM and author raw JSON, including slash-containing names and space-to-underscore normalization; no invented event streams. New test is a sound bounded regression supplement to the previously reviewed full acceptance bar. Existing broader frozen contracts still govern general parsing/effects/inventory semantics.

### Custody, evidence and limits

- PM red.jsonl SHA256d702ffb99ec0c29c901e5dcd3ea43aca79adbcc9d18ed7c30de7e30437129333; contracts.jsonl796f9c8b48fd18b4103433a8ad5b1216076952a155a887844c502aa9e56b855c. red-receipt.json/contracts-receipt.json carry exactargv/cwd/source/exit/deadline/duration and every actual leaf/output.
- Independent audit.json SHA2568652260363a8d4f73ea1177cec141dffe73f3859ae67dadf33bcffe081327719; check.cjs7e379198df3b54ca4d0315d6b742c560f10642c13cfd367cab0504274d31b424. Full100source hashes, six original freezes, deterministic targets and both PM/author actual inventories retained.
- Author REPORT.md93ccb6b88d67bfff40301efee71ddd538cbd25b967abcaf7b7e12c59ec4bd20e and FINAL.md06fe47865cab37d81da5f2265d87ba486e0469bb1ea92f1c63d8dd8f05b4ce5d fully read/hashmatched. Actual author red42a55661d42785a3adf63f98da602eb0d684c576878d205bf944648acb0d6760 and contracts8bf0c058647e700c7b4f2736834e1d1ee9f3465185e14ca6095c9cf0d2fa58a6 independently matched exact outcomes, not accepted solely from summary.
- Six original frozen hashes exactly match previous PM audit at dc3a8a; original prefix326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba. Parser8d2675a64b04050cb11583d240db0f6932d1d1a027900861bc4c6f8c4695189e and Task3d8145088360489fde9a187be00bc89d350ceaca9619120e1721f1328e5cb252 unchanged.
- Read live authoritative ninth-path amendment and unchanged five AC/closed contract; scope FINAL.md7451fd74ce099126a8bccca2f02c71a8b67d5bfebf41e96242f98c22fc9db3af governs added file. Graph main generation2026-09-06T02:42:16Z is best effort; new testoracle files missing, committed Session files metadata_match. Exact Git fallback used; no dev-worktree internals accessed. Mandatory PM/developer/nd/pvg/codebase-memory/vault discipline applied.
-170newLOC,3745cumulative/3713aggregate across9paths, within conditional120–220 new-regression forecast. No proof trimmed, added runtime bound, framework/API/production seam or Paivot product dependency.
- No extra full native/meta rerun at this test-only increment: parent explicitly reserves all original focused/43-child/20mutation/defaultcleanup/source-inventory obligations for final repaired GREEN; no concrete coupling gap found. Final GREEN must pass this new24-leaf regression plus all original68contracts,218native witnesses+10supplements,3/3/17meta and original dynamic inventory/cleanup obligations on one fresh candidate. All seven test files then remain frozen.
- Preserve original genuine semantic RED and separate Task defect; scaffold failures remain NONQUALIFYING. Supplemental ten failures are genuine public-parser unsafe acceptance, not reclassification of old scaffold outcomes. Current old red-approved label alone did not authorize this test; explicit new PM approval is this checkpoint. No whole-story accept/close/merge, new test edit, consumer attestation or authenticated-execution claim.

### AC Verification

| AC | Supplemental decision |
|---|---|
|1|Sound exact-key regression and genuine RED independently verified; exact-membership implementation remains pending separateGREEN.|
|2|Original implementation/tests unchanged and parser-effect contractsPASS; full repaired-candidate context/native replay pending.|
|3|Original20unsafe-variant proof preserved, not rerun or freshly claimed; finalGREEN owes it.|
|4|Full positive Session parser outputs preserved; no fresh218native execution inventory claim.|
|5|Current supplemental delivery shape9/9,source custody and evidence reviewed; whole-story acceptance and final consumer handoff pending.|

LEARNINGS: Pair exact structural-key negatives with full-output valid-data controls; preserve source and evidence classes across supplemental TDD. Supported phase transition results and authoritative final EOF readback are recorded separately in FINAL.md, including any partial command failure without waiver or manual labels.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-06.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


# MAC-uzxr supplemental regression RED candidate

## Implementation Evidence

Candidate supplemental RED awaiting independent PM review. This is not GREEN delivery or approval. Rejected source dc3a8a365c21d5370c0825de992e2461ea0785a9 is unchanged; the historical red-approved label does not approve this new test.

PROOF:

### Commit

SHA: d3d257bd199801d88bdcce8ac8f89fdb8f6bd64c
Branch: story/MAC-uzxr. Subject: test(MAC-uzxr): tdd-red -- require exact structural field membership.
Sole change: examples/go-crm/impl/internal/testoracle/fsm_field_membership_test.go, 170 additions. SHA256201a470a38915709aab457daa4d94011cbaea873428b05a21c2d6e8b6754583b. Committed before final behavioral replay; no implementation or existing-test edits.

### CI/Test Results

Commands run:
- From /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-uzxr/examples/go-crm/impl: GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=120s ./internal/testoracle -run '^TestFSMParseExactFieldMembership$' -json
- Same cwd/environment: go test -count=1 -timeout=120s ./internal/testoracle -run '^TestFSM(ParseCommittedMachines|Effects|Bind|ParseRejects|JSON|Invoke|OneFire|Load|Parser|Nested)' -json
- From /Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-uzxr: pvg verify examples/go-crm/impl/internal/testoracle/fsm_field_membership_test.go --include-tests --format text
- Same worktree: pvg story verify-tdd --base 70652b948bf090008b1965c85daf36ea374daea4 --json
- Same worktree: git diff --check; git status --short; node /tmp/MAC-uzxr-supplemental-red.8wyzmS/audit.cjs

Summary: supplemental regression 24 terminal leaves, 14 PASS / 10 intended FAIL / 0 SKIP, actual Go exit1, 747.200166ms. Original parser contracts 68 PASS / 0 FAIL / 0 SKIP, actual exit0, 352.892833ms. Static verify PASS1file/0issues; normal TDD audit5commits/0violations; git diff check clean; worktree clean. All processes completed synchronously. No warning, compile/import/setup/timeout failure or skipped case is credited as RED.

Go1.27.1 darwin/arm64; ordinary existing local build cache, uncached test execution via -count=1. External spawnSync runner bounds each real Go command to130000ms, Go timeout120s unchanged. It records actual stdout/stderr, argv/cwd/env, producing commit, start time, elapsed wall time, exit, signal, output hash and every native leaf/output in red-receipt.json and contracts-receipt.json. The runner's exit0 means its evidence assertions passed; the supplemental Go subprocess actually exited1.

Coverage: 100% of the ten specified composite-key cases, four ordinary-unknown controls and seven declared metadata/context positions plus combined data, raw committed and remarshaled controls. All68 original contract leaves executed. No statement/branch coverage or whole-story/full-repository/native43-child coverage is claimed by this run.

### Exact regression outcomes

Every leaf has prefix TestFSMParseExactFieldMembership/. Native Go names replace spaces with underscores; the diagnostics below name the exact unnormalized JSON key. The full per-leaf output and terminal inventory are in the actual receipt, not reconstructed events.

| Leaf suffix | Actual terminal |
|---|---|
| committed_control | PASS |
| remarshaled_control | PASS |
| unknown/root/id_initial | FAIL: key="id initial",60rows,nil error |
| unknown/root/_comment__role | FAIL: key="_comment _role",60rows,nil error |
| unknown/root/context_states | FAIL: key="context states",60rows,nil error |
| unknown/state/on_after | FAIL: key="on after",60rows,nil error |
| unknown/state/type_entry | FAIL: key="type entry",60rows,nil error |
| unknown/state/invoke__refusal | FAIL: key="invoke _refusal",60rows,nil error |
| unknown/transition/target_guard | FAIL: key="target guard",60rows,nil error |
| unknown/transition/guard_actions | FAIL: key="guard actions",60rows,nil error |
| unknown/invoke/src_input | FAIL: key="src input",60rows,nil error |
| unknown/invoke/onDone_onError | FAIL: key="onDone onError",60rows,nil error |
| unknown/root/notAllowed | PASS |
| unknown/state/notAllowed | PASS |
| unknown/transition/notAllowed | PASS |
| unknown/invoke/notAllowed | PASS |
| valid_data/comment | PASS |
| valid_data/role | PASS |
| valid_data/delays | PASS |
| valid_data/counters | PASS |
| valid_data/refusal | PASS |
| valid_data/invoke_input | PASS |
| valid_data/context | PASS |
| valid_data/combined | PASS |

Each negative starts from a new decode of the actual committed Session.machine.json, deterministically selects the reviewed object, verifies the key absent, inserts one arbitrary object value, then marshals/redecodes and verifies the actual insertion before calling public Parse with the unchanged committed Markdown. All negatives demand nil Suite and exact oracle-parse: unknown field KEY. nil error fails directly with unsafe unknown field accepted; no baseline exception can pass it. All ordinary controls reached that exact expected error. Subtests are independent, so all ten real unsafe acceptances were observed.

Raw Session control requires Name session and60rows. Every valid addition compares Name, complete Rows (all embedded identity fields and ordered action slices) and complete States (name/kind/entry/exit) with that control. Individual and combined controls cover root_comment/_role strings, _delays/_counters metadata maps, state_refusal, invoke.input expression strings and context arbitrary nested object/array/null/bool/number/string data. All composite and notAllowed strings also occur as legitimate metadata/context data keys. Nonexistent expression references remain inert; no evaluation, global key ban, underscore ban, API change or schema expansion is licensed. Original direct-Markdown current-machine contracts also pass for all five examples.

### Source custody and cost

External source-custody.json verifies all100 current example files against candidate Git objects; the99 preexisting files also match rejected dc3a8a byte-for-byte. The full repository diff since rejected source is exactly one added test file. Parser SHA2568d2675a64b04050cb11583d240db0f6932d1d1a027900861bc4c6f8c4695189e and Task SHA2563d8145088360489fde9a187be00bc89d350ceaca9619120e1721f1328e5cb252 remain unchanged. Six frozen files (relative examples/go-crm/impl):

- internal/cli/command_test.go:38945b9307d97dadb872ac5a261ff86d712c218282774404b432842e34577d60
- internal/domain/deal_test.go:2bfd0893253b9f3c4b6cf1badbf3402a952551f5b4f5124a3d0a9afb7bf432cd
- internal/domain/task_test.go:35b8a05a710792ff21dd827432d658deba1ee190b8739196b084cdc61f7b2b88
- internal/domain/user_test.go:d4fab40b760e5f1a94a5cab88bea6625495c04dd1690dc197818bfd1d056b53d
- internal/session/machine_test.go:97e612a68e22b32010ec7c696cb1ee248115f5a41f3b80400a40160a8782a11f
- internal/testoracle/fsm_test.go:4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25

Original495-line prefix SHA256326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba unchanged. Main497419ab4512fcff765cd5feb27aed4c67b5608d and epic70652b948bf090008b1965c85daf36ea374daea4 unchanged. Measured aggregate3382add331delete=3713changedLOC/9paths; cumulative3575prior+170=3745changedLOC. New170LOC fits120–220 forecast; no proof trimming, API expansion or overrun.

### Artifacts

All in /tmp/MAC-uzxr-supplemental-red.8wyzmS:
- red.jsonl SHA25642a55661d42785a3adf63f98da602eb0d684c576878d205bf944648acb0d6760
- contracts.jsonl SHA2568bf0c058647e700c7b4f2736834e1d1ee9f3465185e14ca6095c9cf0d2fa58a6
- red-receipt.json SHA2569ae996790e0c5f28070d3c4bbed832df480f8dcc57aadd70adb91237383ad679
- contracts-receipt.json SHA256e0c8ff912345ba658b7d735bc5458587e6631e24433c77fd548e2f93d3d04103
- source-custody.json SHA256cc8b113c80810bc05da0a6d6b330b0814a121cdd5bac1eefa10b2d2f3f769fa9
- run.cjs SHA2564df2dd8bdff6c8294c1420774e29c6659f88a9b7c4a54e0d174c9a7b6db5026d
- audit.cjs SHA256b5155dd030e1a8a34ad5f6c77429212162f9a5966db86f553288351f32d9a444

### AC Verification

| AC | Supplemental proof | Disposition |
|---|---|---|
| 1 | Public Parse rejects ordinary keys but actually accepts all10 unknown composites. Exact-field regression now committed. | Genuine RED; implementation repair pending independent approval. |
| 2 | Existing complete effects/current parser contracts pass; source and frozen tests unchanged. | No new full native/context proof; final GREEN must replay original obligations. |
| 3 | Existing historical real20-variant evidence preserved. New regression directly calls public Parse. | No new claim of original43-child replay; final GREEN owed. |
| 4 | Full Session rows/states/actions preserved by ten valid controls; all68 original contract leaves pass. | No fresh218 native-witness inventory claim. |
| 5 | Committed bounded RED candidate, source custody and recognized delivery sections supplied. | Independent supplemental RED review, separate GREEN/full proof and final acceptance remain pending. |

### Limits and phase boundary

No native full focused/meta replay, global preflight, services, Docker, remote/fetch/pull/push/sync/GH, install/update/upgrade/devlink, installed asset, module, oracle, Task, native harness, product documentation or unrelated source changes. Existing healthy worktree retained. Machinery remains standalone; private pvg/nd coordination does not enter product/test dependencies. No child agents. Mandatory developer/nd/pvg/codebase-memory/pm_acceptor lens and vault-knowledge instructions read; graph generation2026-09-06T02:42:16Z is best effort with missing testoracle files, so exact source was used. Canonical scope amendment/closed contract and independent rejection govern this bounded phase. No skill instruction overrides root's narrower authorization.

Original genuine RED and nonqualifying scaffold failures remain historical and separate. The newly observed ten failures reproduce already-recorded R1; no duplicate bug is created. R2's historical3/9 delivery-shape failure remains intact. Later supported transition/readback/9-of-9 result will be recorded separately, with no claim that evidence shape proves product correctness.

LEARNINGS:
- Ordinary unknown-field controls can pass while composite names bypass membership; matched public parser cases expose that distinction.
- Valid metadata/context controls must preserve complete parsed output and use structural-looking data keys to rule out global bans.
- Keep intended Go exit1 separate from an evidence runner's exit0, and inspect all native terminals.
- Delivery headings and authoritative EOF placement require separate supported readback even when substantive runtime evidence exists.

## nd_contract
status: delivered

### evidence
- Supplemental RED candidate d3d257bd199801d88bdcce8ac8f89fdb8f6bd64c: actual14PASS10intendedFAIL0SKIP; original68contractsPASS; one added test, no source correction. Awaiting independent RED PM checkpoint; historical red-approved label does not approve this file.

### proof
- [x] Supplemental AC #1 regression: all10 specified composite-field failures and14 passing controls executed against unchanged rejected source.
- [x] Supplemental custody: six frozen hashes/original prefix/99existingexamplefiles unchanged; committed tdd-red before replay.
- [ ] AC #1 full contract repair requires separate GREEN after independent RED approval.
- [ ] AC #2/#3/#4 complete final native/context/mutation/inventory evidence must be replayed on repaired candidate.
- [ ] AC #5 whole-story acceptance and consumer handoff remain pending; this delivery is supplemental RED only.


## nd_contract
status: rejected

### evidence
- PM rejection applied via pvg story reject on 2026-09-05.

### proof
- [ ] Story requires another developer delivery before it can be accepted.


# MAC-uzxr independent GREEN review — REJECTED

Candidate: dc3a8a365c21d5370c0825de992e2461ea0785a9. Base: 70652b948bf090008b1965c85daf36ea374daea4. Frozen RED: acf1bba2d37784fa175b5ee2cdd0f9b5ac0d5c3f.

## Decision and exhaustive established blockers

REJECTED: one reproduced implementation defect in the agreed closed parser contract, plus a separate delivery-record shape failure. The six frozen files are unchanged and every required frozen test passes. No frozen test is alleged wrong, and no test edit or grammar exception is authorized by this decision.

### R1 — unknown composite JSON fields bypass the closed structural allowlist

EXPECTED: AC1 requires malformed inputs to fail; the independently reviewed closed CONTRACT.md expressly says unknown fields fail even when underscore-prefixed, only listed metadata positions are data, and unsupported structural shapes produce oracle-parse. Root, state, transition and invoke allowed-field sets are exact field names.

DELIVERED: examples/go-crm/impl/internal/testoracle/fsm.go:305–316 uses strings.Contains(" "+allowed+" ", " "+key+" ") for membership. A key containing two adjacent allowed names matches that substring and is silently ignored. Call sites at lines372,415,462,500 expose the error in all four structural scopes.

Actual public Parse counterexample (unmodified candidate package, separate external diagnostic module crm/pmprobe with local replace to the exact Git archive): `GOWORK=off GOPROXY=off GOTOOLCHAIN=local go run -mod=mod .`, cwd /tmp/MAC-uzxr-PM-green.7BaxI6/probe, wrapper deadline130s, actual356ms, exit0, no stderr. Exit0 means the diagnostic program ran, NOT that closed parsing met its contract. The matched committed Session pair yields60rows,nil; ten added unknown composite fields also incorrectly yield60rows,nil:

| Structural scope | Incorrectly accepted unknown keys |
|---|---|
| Root | `id initial`, `_comment _role`, `context states` |
| State | `on after`, `type entry`, `invoke _refusal` |
| Transition | `target guard`, `guard actions` |
| Invoke | `src input`, `onDone onError` |

Every inserted value is an arbitrary object `{unexpectedSemanticData:true}`, not allowed metadata. A matched ordinary `notAllowed` key at each of the same four scopes correctly yields0rows, `oracle-parse: unknown field notAllowed`. Thus the rejection path is reached for ordinary unknowns while composite names bypass it. The real parser is called directly; no stub, rewritten parser, fake process result, test edit, or synthetic success log. The separate diagnostic selects existing Session structural objects; its map iteration does not alter the invariant outcome. It never writes the candidate source/oracles.

GAP: exact allowlist membership is not enforced. The passing frozen negative cases cover ordinary unknown keys but do not cover names spanning adjacent allowlist tokens. This is an implementation counterexample to the existing closed contract, not an invented broader grammar, execution-authentication requirement, or demand to relax frozen tests.

FIX: root/Sr PM should arrange bounded same-story regression-first ownership for a separate new regression file (not any of the six frozen files). The regression must call the public parser with matched valid controls and assert oracle-parse for these composite keys at all four scopes, including `_comment _role`; retain acceptance of the actual declared metadata/context dialect. Demonstrate genuine failing assertions on dc3a8a before the separate implementation repair. Replace substring membership with exact field-name membership in owned fsm.go; do not change allowed fields, parser API, existing frozen tests, generated oracles, Task behavior, or unrelated source. Return through the independent regression/RED checkpoint and subsequent GREEN proof; this review does not approve that future RED or grant a new path implicitly. The existing six test hashes remain locked.

Raw probe.stdout SHA2566158b871c33f6ced2ce1e9dc2a442247dae0b81a66c9e4c73032a73cfa80b2e2. Probe main.go SHA256913a35384167fe1432ac45e228c1c99c1d7cefad77096786a409aaeeb77dbd51; probe/go.mod SHA256bca848201dbb16aafff54eee53bb39304f093678cf1149359a9c5222f37b4fac. Exact command/environment/deadline/hash receipt: probe-receipt.json. Candidate fsm.go SHA2568d2675a64b04050cb11583d240db0f6932d1d1a027900861bc4c6f8c4695189e.

### R2 — supported delivery proof validation fails six shape checks

EXPECTED: supported canonical implementation evidence, CI/Test Results, Commands run:, Summary:, producing commit SHA, and AC verification must be recognized by pvg story verify-delivery, with an authoritative delivered contract at EOF.

DELIVERED: substantive command/results/commit/AC evidence really exists in the fully read report and live Body. Nevertheless the actual supported verifier exits1:3OK (delivered label, last contract, EOF),6FAIL (missing recognized `## Implementation Evidence`, CI/Test Results section, `Commands run:` list, `Summary:` line, commit SHA, AC checklist/table). This is a record shape/placement defect, not absence of executed tests or their source SHA.

GAP: the normal delivery validator is not green; no waiver or manual acceptance is appropriate.

FIX: during re-delivery add correctly shaped canonical evidence through supported append-only nd operations, preserving all prior substantive evidence, historical failures and terminal contracts. Never --description or --body-file. Verify the actual last contract is delivered at EOF, and require verify-delivery9OK/0FAIL before PM acceptance. Do not alter source just to correct tracker formatting. Raw delivery-shape.stdout SHA256f71c17f110481fd8250dd75e25e278ed5e3a03a0aeb2560a270250a7f61904be; full command receipt preserved.

No other implementation, scope, test-integrity, action semantics, process-lifecycle, runtime-inventory, or cost blocker was established by this complete bounded review. This is not a claim of absence of every possible defect.

## Review method and source closure

Applied pm_acceptor GREEN lens, developer/nd/pvg/vault-knowledge/codebase-memory discipline. Fully read current canonical ownership, five AC, authoritative Notes/contracts, current author report and closed CONTRACT.md SHA256b3f444c5398b558916dec80a4dacf8c651420b6aa36d876326d606312baf8cdd. Prior exact amendment review296a026c28fcaf4b3b25bfdcaccf7edc423d5801b7c93d6dac89a253350b14d2 remains applicable to byte-identical frozen tests, not a substitute for current source/runtime checks.

Graph project Users-ramirosalas-workspace-machinery is ready and indexes main generation2026-09-06T02:42:16Z. Six existing paths report metadata_match/no_recorded_issue best effort; new testoracle files are missing from that main index. Exact committed Git objects and a new external `git archive dc3a8a... examples/go-crm` supply candidate source. No developer-worktree internals were accessed. The author external audit.cjs was read as evidence, NOT executed (it references its dev checkout); independent audit uses only exact Git objects, own archive and external raw proofs.

All99 committed example files match own archive byte-for-byte after all diagnostics. Exactly two implementation paths differ from frozen RED; six frozen test hashes and all other source/config/module/oracle/BUILD/evidence bytes remain intact. Global base..candidate diff has exactly the original eight story paths. Complete99-path hash inventory is in audit.json. Six frozen hashes:

- internal/cli/command_test.go:38945b9307d97dadb872ac5a261ff86d712c218282774404b432842e34577d60
- internal/domain/deal_test.go:2bfd0893253b9f3c4b6cf1badbf3402a952551f5b4f5124a3d0a9afb7bf432cd
- internal/domain/task_test.go:35b8a05a710792ff21dd827432d658deba1ee190b8739196b084cdc61f7b2b88
- internal/domain/user_test.go:d4fab40b760e5f1a94a5cab88bea6625495c04dd1690dc197818bfd1d056b53d
- internal/session/machine_test.go:97e612a68e22b32010ec7c696cb1ee248115f5a41f3b80400a40160a8782a11f
- internal/testoracle/fsm_test.go:4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25

All paths above relative examples/go-crm/impl. Original495-line prefix remains326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba.

The full643-line implementation was read. Token-wise JSON decoding detects nested duplicates/trailing values; typed root/state/transition/invoke validation, exact state/row counterpart reconciliation and alternative order precede fallback reachability. Markdown validates required headings/tables/columns/separators/footer, identities, stable stimulus hashes, references and final-state outgoing edges. Stable IDs deliberately exclude target/actions. Bind uses actual pre-Fire closed guard facts and first enabled alternative; Finish detects unused parsed rows. Check uses full ordered equality; external/self effects are exit+transition+entry, internal transition-only, no implicit always or initial entry. The allowlist defect above is the established exception to structural closure.

All five typed wrappers remain the previously reviewed exact frozen bytes: original218 setup/event tuples/order preserved, fact derivation independent of expected row outcome, CLI fixture argv/callback purity and Session pre-Fire time bounds intact, registrations before Fire, observations only after all assertions and exact actual native-name check. Source and runtime confirm real Fire calls, no fake execution seam. Candidate verifier checks real package/test lifecycle and bidirectional row/registration/observation/native sets, slash-bearing names, missing/duplicate/extra identities;218 mandatory witnesses are a subset, not a row ceiling.

Task SHA2563d8145088360489fde9a187be00bc89d350ceaca9619120e1721f1328e5cb252. Exact fallback diff2add/1delete executes existing recordTaskClosed after recordRoutingError and Cancelled assignment, returns both action names in order. Actual Rejection clearing is checked for fallback754183, normalCancelled67b0ff and Donec56bd7. Valid prior branches, non-always context and terminal supplements are preserved. Both exact Task regression anchors match once; missing-call/name-only mutations fail at intended exact actions/context diagnostics. No other production edit.

Documentation inventory/search found no changed public command/config contract requiring an in-scope doc update; BUILD/consumer evidence are explicitly out of scope and unchanged. Native proof flag/storage behavior stays frozen. No Paivot product dependency, framework, API seam, toolchain/source-selector mechanism, or service dependency was added.

## Independent required execution and raw hashes

Own exact archive cwd /tmp/MAC-uzxr-PM-green.7BaxI6/examples/go-crm/impl; Go1.27.1 darwin/arm64; GOWORK=off GOPROXY=off GOTOOLCHAIN=local. Every Go test command uses -count=1. Full argv/environment/cwd, bounded durations, actual terminals and raw SHA are in named `*-receipt.json`. Go120s contracts/native and300s meta; diagnostic runner130s/310s; frozen child context125s/WaitDelay5s and outer270s unchanged. All synchronous, no timeout, panic, compile-only, skip or unexpected stderr credited.

| Group | Actual terminal leaves | Wall ms | Raw JSONL SHA256 |
|---|---:|---:|---|
| contracts |68PASS/0FAIL/0SKIP|601|f7990a6b9ed4390b79654bc01d1fe5bfb734aa609dbf8a3b34a6388ff5994465|
| focused |228PASS/0FAIL/0SKIP|1107|469ad67226e53e9a0b0dc362c6a08fcaf88e7df6a6ea49bab6cf667eb2dc33ac|
| TestFSMNativeOracleSensitivity |3PASS/0FAIL/0SKIP|3421|ca86d48c08e658c13960f75486eeaf2b24f27588ef864bbe2df9a4c9f8e13ac1|
| TestFSMCandidateExecutedInventory |3PASS/0FAIL/0SKIP|2114|d6a999812a2942501bf4b1fcd78a11614b6a04e423f9fc674b2cbaf26f188d9b|
| TestFSMRemainingNativeSensitivity |17PASS/0FAIL/0SKIP|18556|8d84f0fe567bcd29903da1016c5c1b5b537f113fe00704d2d5e3d205cc2c5904|
| default NativeOracleSensitivity |3PASS/0FAIL/0SKIP|3425|9acb41262ea1ef212120ba9687ee19863ccb1b0d199868ea8b82d432b6ac2108|
| default CandidateExecutedInventory |3PASS/0FAIL/0SKIP|2172|3f1c3557e4fe8a7af46a2ca3134f768da6be90c74c87f82539ee38eaa8e3f376|
| default RemainingNativeSensitivity |17PASS/0FAIL/0SKIP|18496|e8859a3bf441cc854a497edb85dcdf5bd708914a2d63cf64cda1599199ff8565|

Five-group total25.799s; three default families24.093s. Existing Go build cache used, not a cold-build claim. Explicit full command selectors are in run.cjs and receipts; all three exported meta groups additionally use `-args -fsm-native-proof-dir=/private/tmp/MAC-uzxr-PM-green.7BaxI6`. Default families omit that flag.

Independent audit reads BOTH own and author raw artifacts and reconstructs source expectations from Git, not author summaries. For each:197rows,47states,118orderedgroups,218registered/218successful observations and10separate supplements. Deal58rows/75witnesses,Task31/35,User20/20,Session60/60,CommandExecution28/28. Exact actual full native leaf set agrees with preserved frozen witness arrays, including slash labels, and all observations occur between native run and terminalPASS after registration. Expected next/actions independently reconciled from committed MD cells and JSON entry/exit data; rows covered both ways. Current row/witness coverage100%; no statement/branch/full-repository coverage claim.

For each exported family set:43real children,28exit0/15intendedexit1; exact all native identities and individual outcomes checked:1398PASS plus15intendedFAIL (6semantic target leaves,9structural pre-Fire roots),0SKIP. Twenty exact unsafe mutations reconstructed from once-only original-byte anchors,47paths in each of43complete manifests,149export artifacts. Later17exact mutation vectors match the independently reviewed AST inventory. All matched controls pass complete selected suites. Five inventory mutants have actual native assertionsPASS but real malformed output: duplicate registration120/60reg/obs; missing0/60; unknown60/60 with UnknownNative identities; missingexecution60/0; duplicateexecution60/120. Frozen verifier rejects each at its required diagnostic; structural outcomes are not semantic proof.

43export receipt cwd references resolve to23removed fixtures. Each default replay set adds43actual children; all3default proof paths and23distinct fixture paths are absent after completion (46references/26distinctpaths). Explicit exports retained intentionally. All99archive source hashes remain exact after execution.

Audit artifact audit.json SHA256f5e3902319197e3394b6ac19f3b6ad2a6be71286c32bdc2e6994db5f8a203926; audit.cjs08d62eb4ce5522397fbf2a0457c5b2bb7e477d1e79e451ab0426ffdce84420d1; audit.stdout2389ac7b438d1950b24d2c3b992cec557e3cc7c9eb9c27af8cf23bd95dd399f1. Raw receipts, full per-child native names/outcomes,47-input hashes, mutations and default cleanup paths are retained. These are ordinary reviewable execution artifacts, NOT authenticated provenance or an attestation.

## Quality, cost, history and AC disposition

- Normal pvg story verify-tdd explicit base..candidate range:4commits,0violations; raw e3f056818bfe3379bd7bd1984daf3af39d28e5b43c9093107627942878d7aa36. Eight explicitly scoped source/test files static0issues; raw c927a11a18cf52ecaa85f4a1b108adc1e2c3200cc88af9cff2d2abe9f874f236. Go signatures typed, no scaffold/TODO stubs credited, git diff --check clean. Static success is not parser correctness.
- Actual GREEN parser607add/12delete=619changed,Task2add/1delete=3, total622. Prior2953+622=3575cumulative; base aggregate3212add/331delete=3543changed across8paths. Conditional3300–3500/3200–3400 forecasts were not caps; explicit closed validation explains modest excess and no scope/path/API expansion. Cost is not a rejection reason and no proof is trimmed. Author25.381s vs PM25.799s required groups is consistent.
- Preserved original genuine semantic RED3unsafe-accepted outerFAIL/200childPASS, storage verification and separate unchanged Task defect are not rewritten. Expanded scaffold ErrScaffold failures remain NONQUALIFYING historical evidence. Current GREEN passes do not retroactively convert them to RED evidence.
- Reviewer utility first audited diagnostic substrings against escaped raw JSON and failed its own assertion; corrected the external audit to inspect decoded Output, then all audit checks pass. This utility failure is not a product/native failure or proof credit. A nonexistent top-level example README lookup was corrected with exact archived Markdown inventory; no source action resulted.
- AC1 NOT MET because of R1. AC2 verified for current semantics, Task context and frozen adversarial controls. AC3 verified for all specified real native variants. AC4 verified current bidirectional inventory. AC5 full story handoff NOT ACCEPTED pending R1 repair and recognized R2 delivery evidence, despite required native proof passing. All newly repaired behavior must return with fresh candidate proof.
- LEARNINGS: exact allowed-name membership cannot use a substring shortcut; passing ordinary unknown-key tests does not establish a closed schema. Preserve negative evidence by class and validate delivery record shape separately from substantive runtime evidence.

No acceptance/close/merge/claim renewal, remote/preflight/install/update/assets/service/Dagger operation, developer-worktree access, or source/test edit was performed by PM. Root owns routing and any new regression ownership. Main497419ab4512fcff765cd5feb27aed4c67b5608d and epic70652b948bf090008b1965c85daf36ea374daea4 unchanged. Supported rejection and authoritative EOF readback are recorded separately in FINAL.md after this report is hashed.


## nd_contract
status: delivered

### evidence
- Terminal GREEN delivery MAC-uzxr: supported pvg story deliver MAC-uzxr exited0 without waiver. Candidate dc3a8a365c21d5370c0825de992e2461ea0785a9 on story/MAC-uzxr. Worktree clean. Independent PM accepts or rejects; this developer did not accept/close/merge.
- Full PROOF, LEARNINGS, exact eight-path source scope, command argv and outcomes are in the preceding appended report and /tmp/MAC-uzxr-green.Td49Am/REVIEW.md SHA256918947599ceb237b56b1a87492f4127eee45f8c625e070ad558ef623d7762727. Primary native-audit.json SHA2561d6902dc46bf95a7a128a9c33da1cc802c79c77b2530afcbd9ee546882dc4db8; source-closure.json SHA256fba784f86918e105208e76d95a42f2b39acac8be114dbef95925a3628f877455. All exact producing Go commands/argv/cwd/sourceSHA/durations/rawSHA/full native identities are retained in named receipt files; all used GOWORK=off GOPROXY=off GOTOOLCHAIN=local and -count=1, Go120s native/unit and300s meta with unchanged125s/270s contexts and5s child WaitDelay.
- Actual required run:68 parser nested leavesPASS,228 focused native leavesPASS (218 transition witnesses+10 supplements),23 meta outer leavesPASS (3 original mutations+3 candidate packages+17 remaining mutations),0 unexpectedFAIL/0SKIP. Exact current inventory197rows/47states/118orderedgroups,218registrations and218successful observations: Deal58/75,Task31/35,User20/20,Session60/60,CommandExecution28/28 rows/witnesses. Coverage100% current rows and original witnesses; no statement coverage or full-repository test claim.
- All20 actual unsafe variants rejected with matched complete passing controls and exact intended diagnostics.43 exported sequential children:28exit0/15 intendedexit1;1398 native leaf/rootPASS+15 intendedFAIL (6semantic leaves,9structural pre-Fire roots),0SKIP. Five inventory mutants keep native assertionsPASS but the actual-output verifier rejects their malformed registration/execution inventory. Both Task future anchors matched once and real missing/name-only entry mutations failed at exact actions/context assertions.
- Custody:99 committed examplefiles verified;47 current inputpaths in43 complete manifests; all20 mutation hunks reconstructed from original bytes with once-only anchors and exact resulting hashes.149 exported raw/receipt/manifest/mutation artifacts retained;43 receipt cwd references resolve to23 removed fixture directories. All three default meta families independentlyPASS with43 further actual children;3 default proof directories and23 distinct child directories removed. Test-cache reuse excluded by -count=1. Export five-group25.381077583s; default lifecycle families24.461417250s on existing build cache; no cold-build measurement claimed.
- Six frozen SHA256 unchanged (paths relative examples/go-crm/impl): internal/cli/command_test.go38945b9307d97dadb872ac5a261ff86d712c218282774404b432842e34577d60; internal/domain/deal_test.go2bfd0893253b9f3c4b6cf1badbf3402a952551f5b4f5124a3d0a9afb7bf432cd; internal/domain/task_test.go35b8a05a710792ff21dd827432d658deba1ee190b8739196b084cdc61f7b2b88; internal/domain/user_test.god4fab40b760e5f1a94a5cab88bea6625495c04dd1690dc197818bfd1d056b53d; internal/session/machine_test.go97e612a68e22b32010ec7c696cb1ee248115f5a41f3b80400a40160a8782a11f; internal/testoracle/fsm_test.go4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25. Original495lineprefix326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba. Only owned parser implementation and Task.fireRolledBack fallback changed; no test/API/path broadening.
- Normal pvg story verify-tdd --base70652b948bf090008b1965c85daf36ea374daea4 checked4commits0violations/no waiver; eight explicitly scoped pvg verify paths --include-tests --format text PASS8files0issues; gofmt and git diff --check clean. Initial skill-documented --format=text was rejected by installed pvg, corrected using read-only help to supported --format text. No mutating subcommand help, unresolved product failure, remote/install/preflight/service/background operation.
- Measured cost: fsm.go607add/12delete final643lines; Task2add/1delete; GREEN622changedLOC. Historical2953+GREEN622=3575cumulative; aggregate3212add/331delete=3543changedLOC across8paths. Explicit closed grammar/type/metadata/duplicate-key checks and error precedence exceed the conditional parser estimate; no additional file/framework or proof trimming.
- Historical genuine RED3unsafe-accepted outerFAIL/200childPASS and separate Task defect are preserved. Expanded scaffold zero-row/zero-mutant failures remain NONQUALIFYING historical evidence, not new RED credit. Independent PM review and consumer migration remain pending; no automatic attestation/acceptance/authenticated-execution renewal.

### proof
- [x] AC #1: complete current parsing and original actual input/guard binding; missing/duplicate/malformed/unmapped/unused identity sensitivity with real controls.
- [x] AC #2: exact one-Fire ordered effects plus real Task entry/context repair and missing/name-only regression sensitivity.
- [x] AC #3: all20 actual unsafe variants rejected with complete passing controls and pinned intended diagnostics.
- [x] AC #4:197 current rows mapped bidirectionally to218 successful actual witnesses;10 separate supplements;0SKIP.
- [x] AC #5: all required frozen focused/parser/meta proof passed on committed candidate; unchanged source closure and reviewable consumer handoff; PM acceptance pending.


## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-05.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


# MAC-uzxr GREEN delivery

Candidate: `dc3a8a365c21d5370c0825de992e2461ea0785a9`, branch `story/MAC-uzxr`, worktree `/Users/ramirosalas/workspace/machinery/.claude/worktrees/dev-MAC-uzxr`. Implementation was committed before the complete verification. Independent PM acceptance remains pending.

## Implementation and boundary

Only two implementation files changed from frozen RED `acf1bba2d37784fa175b5ee2cdd0f9b5ac0d5c3f`: the private Go CRM test oracle helper and the exact Task rollback fallback. The helper uses only the standard library, reads both current files, rejects duplicate JSON keys including nested data/metadata, validates the closed Markdown/JSON dialect, and reconciles states and ordered transition alternatives. Stable IDs hash stimulus identity and deliberately exclude outcomes. Bound witnesses supply actual source/event/guard facts; missing/extra facts, wrong priority, duplicate names and unused rows fail. Effects represent one Fire: source exit, transition, target entry for external/self transitions; transition only for internal transitions. Registration and successful execution remain distinct test records.

The Task fallback still records the routing error and routes to Cancelled, then invokes the existing recordTaskClosed method and reports [recordRoutingError, recordTaskClosed]. It clears the actual Rejection value. No other Task guard, action, event, routing or final-state definition changed. Both frozen future Task mutation anchors matched exactly once during real mutant creation.

The source audit compared all99 committed example files to the delivered git object. The only two changed files versus frozen RED are the owned implementations. Ten committed design inputs, modules/config, BUILD/acceptance/attestations, generated files, unrelated production and all six frozen assertion files remain unchanged. There is no new Machinery API, runtime parser dependency, Paivot product dependency, new source/test path, installer or installed-asset mutation.

Skills: developer, nd, pvg, vlt, vault-knowledge, codebase-memory, and the mandatory pm_acceptor review lens were read. PM role transitions were not performed. Vault context was read through pvg notes. Graph-first index was ready at generation2026-09-06T02:42:16Z; testoracle files were missing from the main index and Task receiver lookup collided with Deal, so exact branch source supplied evidence. No graph completeness claim is made.

## PROOF: actual commands and outcomes

All Go commands ran with GOWORK=off, GOPROXY=off, GOTOOLCHAIN=local from the worktree's `examples/go-crm/impl`; host `go1.27.1 darwin/arm64`. Exact argv, cwd, environment, source SHA, raw SHA, exit, duration and full terminal identities are in each `*-receipt.json`. `run.cjs` uses synchronous spawnSync with external130s unit/focused and310s meta bounds. Frozen subprocesses retain Go120s, context125s, WaitDelay5s, sequential270s outer contexts and300s Go meta deadlines. No timeout was expanded.

1. `go test -count=1 -timeout=120s ./internal/testoracle -run '^TestFSM(Parse|Effects|Bind|JSON|Invoke|OneFire|Load|Parser|Nested)' -json`: exit0,68 terminal contract leaves PASS,0FAIL,0SKIP;0.500100333s. This executes all ten contract roots and their nested cases, not just current-file positive parsing.
2. `go test -count=1 -timeout=120s ./internal/domain ./internal/session ./internal/cli -run '^(TestDealTransitions|TestTaskTransitions|TestUserTransitions|TestSessionTransitions|TestCommandExecutionTransitions|TestTaskTerminalRejectsEverything|TestTaskRollbackNonAlwaysPreservesContext|TestCommandExecutionTerminalExits)$' -json`: exit0,228 leaves PASS,0FAIL,0SKIP;0.725994208s. Exactly218 transition witnesses and10 terminal/non-always supplements.
3. `go test -count=1 -timeout=300s ./internal/testoracle -run '^TestFSMNativeOracleSensitivity$' -json -args -fsm-native-proof-dir=/private/tmp/MAC-uzxr-green.Td49Am`: exit0,3 outerPASS,6 actual children;3.463955917s.
4. `go test -count=1 -timeout=300s ./internal/testoracle -run '^TestFSMCandidateExecutedInventory$' -json -args -fsm-native-proof-dir=/private/tmp/MAC-uzxr-green.Td49Am`: exit0,3 outerPASS,3 actual full candidate children;2.186817375s.
5. `go test -count=1 -timeout=300s ./internal/testoracle -run '^TestFSMRemainingNativeSensitivity$' -json -args -fsm-native-proof-dir=/private/tmp/MAC-uzxr-green.Td49Am`: exit0,17 outerPASS,34 actual children;18.504209750s.

Five-group total25.381077583s. These observations used the existing build cache with uncached test execution (`-count=1`); no cold-build measurement is claimed. Earlier targeted iteration passed immediately, before commitment, with parser package0.387s and native domain/session/CLI0.298/0.774/0.541s. No implementation test-fix iteration, compiler failure, skip or timeout occurred.

Default lifecycle validation reran each meta command above without `-args -fsm-native-proof-dir`: Native3PASS in3.697098208s, Candidate3PASS in2.186476125s, Remaining17PASS in18.577842917s; total24.461417250s. Their actual output identifies three default proof directories and43 child cwd references (23 distinct child directories); all were absent after completion. These default runs execute a further43 children; exported43 children remain the primary inspectable raw native evidence.

The story-scoped focused suite and every frozen parser/meta test ran. Root/full-repository tests, unrelated Go CRM tests outside the specified selection, global preflight, services and consumer attestation migration were not run or claimed. Coverage reported here is100% of current197 parsed rows and100% of218 preserved transition witnesses, plus all10 supplements; no statement/branch coverage percentage is claimed.

## Actual row, witness and native inventory

Independent external `node /tmp/MAC-uzxr-green.Td49Am/audit.cjs` exited0. It rereads actual JSONL output and git/source bytes; it does not fabricate process events. It verifies source closure, raw receipts, once-run/once-terminal identities, exact registered and successful observed identities, expected next/actions directly from current Markdown and state semantics, complete row coverage both ways, original source mutation anchors and resulting hashes, and cleanup paths.

| Machine | Current rows | Registered witnesses | Successful observed witnesses |
|---|---:|---:|---:|
| Deal |58|75|75|
| Task |31|35|35|
| User |20|20|20|
| Session |60|60|60|
| CommandExecution |28|28|28|
| Total |197|218|218|

There are47 states and118 ordered source/event groups. All observed witnesses have actual native terminal PASS and matching identity/next/actions. Supplements are separately10PASS; registration alone is not execution credit. All original218 inputs, tuple order, guard clauses and native identities are retained by the six frozen byte hashes. New coherent unmapped rows fail explicitly rather than silently disappearing; implementation has no197/218 ceiling.

The exported meta families contain43 real children:28 exit0 and15 intentionally failing unsafe children. Their native leaf/root terminal inventory is1398PASS,15 intendedFAIL,0SKIP. The15 failures divide into six semantic single-leaf failures and nine structural pre-Fire root failures. Five inventory-corruption mutants retain native assertion PASS and are rejected by the same actual-output inventory verifier with their pinned diagnostic; their native success is not credited as accepted inventory. Every original3 plus later17 mutation case has a complete passing unchanged control, matching package/suite selection and exact intended diagnostic checked by the frozen test. No failed control, compile/setup failure, missing anchor, timeout or unexecuted future Task anchor is counted as negative sensitivity.

All20 mutations were reconstructed from actual `mutations.json` old/new bytes, each anchor matched once, and43 control/candidate/mutant manifests each contain the same47 source-input paths. All unrelated input hashes match.149 explicit-export artifacts remain:43 raw child logs,43 execution receipts,43 manifests,20 exact mutation files. All43 receipt cwd references (23 distinct directories) were removed. Raw default outer logs remain intentionally as audit artifacts while automatic inner proof storage was removed.

## Raw evidence and frozen custody

Primary directory: `/tmp/MAC-uzxr-green.Td49Am` (macOS real path `/private/tmp/MAC-uzxr-green.Td49Am`). `native-audit.json` records all raw child terminal identities, diagnostics, registrations, observations and cleanup paths. `source-closure.json` records all99 committed source hashes. `hash-inventory.json` records proof file SHA256 values.

| Artifact | SHA256 |
|---|---|
| contracts.jsonl |4086c1877d155420bd14c48a8fdb2afc437f75c489b536575bcf5c569b7d5e53|
| focused.jsonl |fa1138488cdc028ba0187e09d6d6b034ad0d1c780ea4b272c76b5070f216bb3a|
| TestFSMNativeOracleSensitivity.jsonl |d01b8090f29237025f13098b0ffdf7bb109e829548f5730f269d9e6897e65235|
| TestFSMCandidateExecutedInventory.jsonl |eaf540e362a2830e55acff8a42785fb7a97f13d60545f9c00aaa3c5729d08042|
| TestFSMRemainingNativeSensitivity.jsonl |2570e36dd2e94f29ffcec7b17152a984bb28630d0f83ac4594f5797d162ab010|
| default.jsonl |08528ab3934f400f7f8b719fbcd969bf61d895acfaa0f0c2062c67e6ce51b7e8|
| default-native.jsonl |f71744e22b64696dca237b9dc9eaf6041c87662dd07c8683321ca5b873bc6953|
| default-remaining.jsonl |beea1dddadb0a5ddd548273b274e3a7b74022720012baf4452bf61529bd9f407|
| native-audit.json |1d6902dc46bf95a7a128a9c33da1cc802c79c77b2530afcbd9ee546882dc4db8|
| source-closure.json |fba784f86918e105208e76d95a42f2b39acac8be114dbef95925a3628f877455|

Six unchanged frozen hashes, paths relative to `examples/go-crm/impl`:

- internal/cli/command_test.go:38945b9307d97dadb872ac5a261ff86d712c218282774404b432842e34577d60
- internal/domain/deal_test.go:2bfd0893253b9f3c4b6cf1badbf3402a952551f5b4f5124a3d0a9afb7bf432cd
- internal/domain/task_test.go:35b8a05a710792ff21dd827432d658deba1ee190b8739196b084cdc61f7b2b88
- internal/domain/user_test.go:d4fab40b760e5f1a94a5cab88bea6625495c04dd1690dc197818bfd1d056b53d
- internal/session/machine_test.go:97e612a68e22b32010ec7c696cb1ee248115f5a41f3b80400a40160a8782a11f
- internal/testoracle/fsm_test.go:4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25

The original495-line fsm_test prefix remains326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba. Implementation hashes: fsm.go8d2675a64b04050cb11583d240db0f6932d1d1a027900861bc4c6f8c4695189e; task.go3d8145088360489fde9a187be00bc89d350ceaca9619120e1721f1328e5cb252.

Original independent RED remains distinct historical evidence: genuine3 unsafe-accepted outerFAIL/200 childPASS and separate missing Task entry effect. Expanded scaffold outcomes previously reached zero transition rows/mutants and remain NONQUALIFYING; they are not retroactively relabeled as additional RED. The canonical closed-contract hash b3f444c5398b558916dec80a4dacf8c651420b6aa36d876326d606312baf8cdd and independent review/final hashes072af4cda455158288e635968040fa0930e1f40a0fd0fead10314497341d670c/726513393cbb8d1ee2c8539e561cabf0a9ecd6659a3959786bd4e3d60aadc4f5 were checked.

## Quality, cost and limits

`pvg story verify-tdd --base 70652b948bf090008b1965c85daf36ea374daea4 --json`: four commits checked, zero violations, no waiver. Scoped `pvg verify` with all eight explicit story paths, `--include-tests --format text`: PASS8files/0issues. Initial implementation-only scan also PASS2files/0issues. `gofmt` applied only to two owned implementation files; `git diff --check` passes and worktree is clean.

One process-documentation discrepancy occurred: the developer skill's `--format=text` syntax was rejected by installed pvg with exit1. Read-only `pvg verify --help` specifies `--format text`; that supported syntax passed. No mutating story subcommand help was called. No product failure, unresolved warning, external service, remote operation, install/update/devlink, root/epic branch operation or background process remains.

Measured GREEN delta: fsm.go607 additions/12 deletions (619 changed LOC, final643 lines); Task2 additions/1 deletion (3 changed LOC). Total609 additions/13 deletions=622 changed LOC. Cumulative story cost is2953 historical/pre-GREEN+622=3575 changed LOC. Final aggregate against epic70652b948 is3212 additions/331 deletions=3543 changed LOC across exactly eight paths. The conditional3300–3500 cumulative/3200–3400 aggregate forecast was exceeded modestly: explicit closed schema/type/metadata/duplicate-key validation, Markdown table grammar, identity-before-reachability diagnostics and typed Go error handling expanded the parser beyond the300–450-line estimate. No extra file/API/framework or proof trimming was used to meet the estimate.

## AC verification

| AC | Evidence | Status |
|---|---|---|
|1|Current Parse/Load contracts, all218 actual witness bindings and nine real structural mutants;197 rows mapped bidirectionally|PASS|
|2|Exact one-Fire effects, every nested effect variant, real Task entry/context clearing and missing/name-only mutations|PASS|
|3|Original three real semantic variants plus later semantic cases reject with valid full controls and exact diagnostics|PASS|
|4|Five-machine197-row/218-successful-witness inventory, independent raw audit, all10 supplements, zero skips|PASS|
|5|Complete required focused and parser/meta proof at committed candidate, unchanged frozen/source closure, reviewable consumer handoff|PASS; independent PM acceptance pending|

## LEARNINGS

- Reconcile counterpart identity/order before rejecting coherent unreachable fallback alternatives; otherwise malformed-control precedence can conceal an identity mismatch.
- Nested duplicate JSON keys require token-level parsing before maps collapse them; inert data still needs closed structural validation.
- Action-name equality does not establish an entry action ran. The Task context mutation and real name-only forgery close that gap.
- Successful native leaves, registrations and parsed row identities are separate inventories.43 complete children, exact mutation manifests and lifecycle audit make the claim independently inspectable.

## nd_contract
status: delivered

### evidence
- GREEN dc3a8a365c21d5370c0825de992e2461ea0785a9, only owned parser and Task fallback implementation changed; six frozen hashes above unchanged.
- PROOF above supplies exact source/commands/logs/native inventory/mutation controls/custody/cleanup/timing/cost.68 parser+228 focused+23 meta outer leaves PASS; exported43 actual children validate20 unsafe variants and full candidate inventories.0 unintended failures/0SKIP.
- Normal TDD4commits0violations; scoped8files0issues; clean committed worktree. Independent PM acceptance and later consumer migration remain separate.

### proof
- [x] AC #1:197 current rows bound to218 original actual input/guard witnesses; structural closure proved.
- [x] AC #2:exact ordered effects and real Task entry/context execution; unsafe variants rejected.
- [x] AC #3:all20 real unsafe variants with complete passing controls and exact intended diagnostics.
- [x] AC #4:197-row/218-successful-native-witness bidirectional inventory plus10 supplements, zero skips.
- [x] AC #5:required focused/parser/meta candidate proof complete with exact frozen source scope and reviewable logs; no automatic acceptance/attestation renewal.


## nd_contract
status: red-approved

### evidence
- RED tests approved via pvg story approve-red on 2026-09-05.

### proof
- [ ] GREEN developer must implement against the approved RED tests without modifying them.


## INDEPENDENT COMBINED-EVIDENCE RED DECISION — MAC-uzxr

Full PM report /tmp/MAC-uzxr-PM-combined.au6aPZ/REVIEW.md SHA256072af4cda455158288e635968040fa0930e1f40a0fd0fead10314497341d670c. This records the evidence-based decision; the supported approve-red transition and terminal readback follow separately, without waiver.

# MAC-uzxr independent combined-evidence RED adjudication

PM decision: **APPROVE RED** for frozen commit `acf1bba2d37784fa175b5ee2cdd0f9b5ac0d5c3f`, subject to the supported approve-red command succeeding without waiver. This approves the test bar for a fresh GREEN developer; it does not accept implementation, claim completed conformance, mark any full AC proved, or authorize broader ownership. The current scaffold failures earn no behavioral RED credit.

## Why the combined evidence is sufficient

The canonical Testing Requirements expressly permit test-support scaffolding to support a qualifying RED when unchanged old suites demonstrate the specific missing parser/sensitivity behavior. That condition is independently met here, without depending on ErrScaffold:

1. The original immutable998a5a3 semantic tests, preserved byte-for-byte through storage4ef6baf and as the entire495-line prefix in acf1bba, actually run complete unchanged User/Session controls and exact unsafe variants. A fresh PM replay of4ef6baf again produced three intended outer `unsafe native variant was accepted` failures, with all200 child leaves passing. Coherent expected-state/action mutations retained valid stable stimulus identities; the Session mutation introduced an observable extra effect on an empty-effect row. There was no parser/setup/control failure in that replay. This is genuine evidence of the missing sensitivity the story repairs.
2. The Task missing entry action is independently established on unchanged production by the preserved real probe: normal TASK-67b0ff PASS, fallback TASK-754183 FAIL for missing recordTaskClosed. The wider strict-effects diagnostic has224PASS/1FAIL and the same sole fallback. These prove a separate application discrepancy and are not relabeled as parser RED. The frozen new Task assertions additionally require actual Rejection clearing, retaining normal Cancelled/Done, valid-prior, non-always and terminal controls. A forged action name will not satisfy the context assertion.
3. Exact independent amendment review296a026c28fcaf4b3b25bfdcaccf7edc423d5801b7c93d6dac89a253350b14d2 established the complete prospective test bar: all218 original actual input/guard witnesses, closed MD/JSON parsing and identity/order reconciliation, exact one-Fire effects, real successful execution inventory, and all17 additional exact structural/semantic/inventory variants. The candidate is exactly that authorized byte sequence, now committed with both audit markers. The tests and minimum interface compile, and finite replay confirms which code is reached at the boundary. No observed failure establishes a concrete defect in the assertions themselves.

RED does not require implementing the GREEN parser merely to run every future passing control. Requiring that would defeat the explicit canonical staging allowance. Conversely, the new scaffold-bound outcomes do not prove parser sensitivity, effects, guard binding or native inventory; approval relies on the preserved genuine behavior plus the reviewed frozen contracts, not on counting those failures. The GREEN review must establish all presently unreachable success and negative paths on the exact candidate. This is a RED approval judgment, not a preapproval of those future results.

## Independent scope/static audit

Main-agent skills read in this review conversation: pm_acceptor, developer, nd, codebase-memory, vault-knowledge; pvg skill additionally read for the supported RED transition. Canonical current description/five AC, latest Notes and author checkpoint were read; no final-delivery or acceptance procedure was substituted for the requested RED phase. Author report `/tmp/MAC-uzxr-final-red.nOXYuf/REVIEW.md` SHA256 `5cb98377240aa8be22f0b9d88b74a7a6bd959fbfd8f1efb2907d08b02f68ad5c` was read completely.

Graph-first index remained ready, generation2026-09-06T02:42:16Z; seven test/support paths plus Task were coverage-checked. Existing files report metadata_match/no recorded gap; the new testoracle paths are missing from the main index. Exact committed git objects supply authoritative branch content; graph completeness is not assumed.

Owned detached checkout `/tmp/MAC-uzxr-PM-combined.au6aPZ/current` is pinned to acf1bba. A separate owned archive contains exact historical4ef6baf example source. No developer-worktree internals were inspected. Every one of99 current example files was checked against the commit, all seven allowed resulting hashes matched the previous authorization, and every other file matched4ef6baf. Exactly seven paths differ. The495-line prefix remains SHA256 `326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba`; expanded fsm_test.go `4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25`. Production Task remains `39af26a093f855acf269da153915fbecf6e78c0f3dbba85b087bf4b83586e35b`. The bounded source/tuple/model/AST reconstruction from the prior exact review remains applicable to identical bytes:218 witnesses/197rows/47states/118orderedgroups, preserved ordering and both original terminal bodies,17 exact mutations, closed current dialect and real pre-Fire guard facts.

`pvg story verify-tdd --base 70652b948bf090008b1965c85daf36ea374daea4 --json` from the detached candidate checked three commits with zero violations and no waiver. Scoped `pvg verify` all seven exact paths with --include-tests passed7files/0issues. Diff whitespace check and final detached status were clean. Those checks do not prove a parser exists. The exact48-line fsm.go is still ErrScaffold. Three audit commits are998a5a3 (tdd-red),4ef6baf ([test-edit-authorized] storage), acf1bba ([test-edit-authorized] and tdd-red full freeze). Canonical eight paths/five AC and measured2450 later changed LOC are unchanged.

## Current independent actual replay

Host go1.27.1 darwin/arm64. Every command used GOWORK=off, GOPROXY=off, GOTOOLCHAIN=local and -count=1. Exact argv/environment/cwd/duration/outcomes are in each `*-receipt.json`; commands run synchronously with explicit130s/310s external bounds, Go120s/300s and existing child120s/context125s/WaitDelay5s and meta270s limits.

| Group | Actual exit | Terminal leaves | Wall |
|---|---:|---|---:|
| Ten parser/contract roots |1|0PASS/28FAIL/0SKIP|0.629s|
| Five transition suites and three supplement roots |1|10 supplementPASS/5 transition-rootFAIL/0SKIP|1.083s|
| TestFSMNativeOracleSensitivity |1|3 outerFAIL/0PASS/0SKIP|2.154s|
| TestFSMCandidateExecutedInventory |1|3 outerFAIL/0PASS/0SKIP|2.154s|
| TestFSMRemainingNativeSensitivity |1|17 outerFAIL/0PASS/0SKIP|10.376s|

Five-group total16.396s. All current parser/control failures are NONQUALIFYING ErrScaffold; required positive Parse failure makes many nested malformed-input and effect checks unreachable. No skip, compile error, panic, process timeout or nonempty stderr occurred. The ten passing leaves are exclusively the seven original terminal supplements plus three new Task non-always cases.

Current23 actual children contain10 supplementPASS,25 transition-rootFAIL, zero transition-row leaves, zero registration/observation records, zero mutants, zero skips. The full candidate checker sees process exit1; neither its successful reconciliation nor any negative inventory corruption case is proved. All17 later variants stop at failed controls, so future Task anchors are not reached. An independent audit also read the author's corresponding raw children and corroborated exactly those same counts and limits.

Each current export set has52 files:23 raw logs,23 execution receipts and6 manifests of47 committed input hashes. All receipts have correct package/command/actual once-run and terminal inventories, raw SHA256, exit and removed fixture cwd. No mutant artifacts exist. The17 failed remaining controls export no manifests because manifest creation follows successful controls; none is invented. New candidate default storage was independently replayed once (2.153s,3 outerFAIL at scaffold); the emitted proof directory and all3 child cwds were absent afterward. Historical cleanup/export proof remains preserved, and passing GREEN lifecycle verification is still required.

Current raw SHA256 values under `/tmp/MAC-uzxr-PM-combined.au6aPZ/`:

- contracts.jsonl: `a6e45c397b29d1bc220a37544156bfb73dca352cab65e0e3548a6b0bb9ef56bb`
- focused.jsonl: `c298b6b6c30c7db68df012fed4005d5d1eb10f2a199f61f957919caa8e00f491`
- TestFSMNativeOracleSensitivity.jsonl: `857006cf7e89cb76c79557a714a02836a9348401b3f04951841845006f57e1a6`
- TestFSMCandidateExecutedInventory.jsonl: `3bc0d8fa3d1392a5289db7efc16505bf8fd69fe5c2b47242a88b2376e90f52b5`
- TestFSMRemainingNativeSensitivity.jsonl: `b61fcd8cc67d255cb5248188ebfa1e466e8cf56dac8c0990f24f6c9bcb40bb00`
- current-default.jsonl: `f5b0d1773a9024782d970850463e11af64160fd6cad4543eb2d2faa5226e8efd`

## Genuine historical replay and independent raw audit

From the exact4ef6baf archive, ran `go test -count=1 -timeout=300s ./internal/testoracle -run '^TestFSMNativeOracleSensitivity$' -json -args -fsm-native-proof-dir=/tmp/MAC-uzxr-PM-combined.au6aPZ` under the same offline/local environment. Actual exit1,3 intended outerFAIL,3.398s. All six children exited0 with200 native leaves PASS/0FAIL/0SKIP; each User pair exercised all20 leaves and the Session pair all60. Raw historical-genuine.jsonl SHA256 `a8f978b85b1031dac050b1f5cd49a0c7b45e816c2b67c25deb72d63622f2e968`.

Independently reconstructed46 control input hashes from historical git objects, exact once-only2/2/1 mutation files from recorded old/new bytes and all resulting hashes. Verified all6 receipts, commands/package identities, exact native names and once-run/once-PASS termination; all fixture cwds removed. Explicit export retains21 proof files. The outer diagnostics are actual unsafe acceptance, not scaffold failure. These historical tests are unchanged and no fixed Task baseline was fabricated.

Previously independent225PASS baseline raw `/tmp/MAC-uzxr-pm-initial-red.csxQrh/baseline.jsonl`, SHA256 `ee99c3aa35061bfc763ad6566a616920e98327b79f06e1637898fde8874f4ea1`, was re-audited for exact218 transition plus7 supplement identities, including slash labels. It was not unnecessarily rerun. Original semantic report09291b1d33e9a7c77981c76f2cba2b01415eed3695de5d44f5c650fd31d119b6 and storage reportb52cda2189988e9516cc10c59cda850e39f82502405eaec75ba0fdcc15bbd184 remain intact.

Read the exact Task probe source/raw, verified its production source equals the current unchanged Task hash and its exact1PASS normal/1FAIL fallback outcomes. Probe raw SHA256 `90e3d11c3df38725d77637bc187d7e27dcf0a2b65f0934e6b71c785b177e783c`; strict-effects raw `c6ce626ee381f97d020a2bea090d1a626e9b9adfa421c75ffd877cdfb8ee42a4` independently has224PASS/1FAIL/0SKIP with exactly the historical225 identity set and sole TASK-754183 failure. These are application-effect diagnostics, not proof of parsed expectations or fake-name rejection on a corrected candidate.

Independent audit script `audit.cjs` SHA256 `7097b7571fead561adacde188933bea3ff2e04edafc3fc60d1d0cda66dca6dfa`; raw `audit.jsonl` SHA256 `abaa51f3f769410907eb17e283f17ce7effd916b088d8523a3919d72c9d175a8`. Synchronous runner `run.cjs` SHA256 `542381f49ab05efcb3f73c5fd06cdf559401105a71b8e38ca43b776c324d8507`. Audit exited0 in4.220670875s. An initial pvg approve-red --help lookup was unsupported and exited1 without mutation; documented skill syntax is used for the actual transition, not an inferred flag. No product error was concealed.

## Frozen GREEN obligations and phase boundary

Root must dispatch a **fresh GREEN developer**, separate from the RED author's context. Six Go test files (five original suites plus fsm_test.go) at acf1bba are frozen; no edit, deletion, weakening, skip or helper bypass is authorized. The seven-file freeze includes the exact minimum fsm.go API, whose non-test implementation body is the explicitly owned GREEN work. Implement its reviewed closed contract without a broader parser API/product dependency. Preserve current source/oracle/module/config closure: do not change the ten committed design inputs, module files, generated evidence, BUILD/attestations or unrelated production. Private implementation helpers within the owned parser are ordinary GREEN work; changing frozen assertions or widening the interface is not.

Only the already-scoped Task.fireRolledBack fallback may change in application production: recordRoutingError, Cancelled routing, actual existing recordTaskClosed call and ordered [recordRoutingError,recordTaskClosed]. No other guard/routing/event/final-state/action behavior is authorized. Future fake-name and missing-call anchors must then match exactly once; failed controls or missing anchors never count as successful regressions. A new concrete mismatch returns through the existing scope/test-dispute process.

GREEN must pass the complete focused native suite with actual parsed row/guard execution and all terminal/non-always supplements; every parser contract and nested rejection table; full bidirectional parsed-row/successful-observation/native-leaf inventory; all20 actual mutation cases (original3 plus later17) with valid complete unchanged controls and exact intended diagnostics. The full meta families contain43 sequential children when controls succeed. Prove source/input/manifests/raw output custody, unchanged unrelated bytes, all actual native identities/outcomes, zero skips, default cleanup and intentional export. Report measured full runtime and actual implementation/diff cost;16.396s scaffold replay is not full candidate timing. Run normal TDD audit without waiver, then independent GREEN PM review. No full AC, current consumer claim, acceptance or authenticated-execution assertion is granted now.

## nd_contract
status: in_progress

### evidence
- PM combined-evidence decision APPROVE RED for acf1bba; supported transition/readback still to be recorded after this report.
- Genuine historical sensitivity replay and separate Task defect support the explicit canonical staging allowance; current scaffold outcomes remain NONQUALIFYING.
- No source/test edits, implementation, remote/install/assets/service/preflight operations, child agents or acceptance/close performed by PM. Detached review and historical archives retained; no worktree cleanup.

### proof
- [ ] AC #1: current parser implementation and complete actual input/guard binding pending GREEN.
- [ ] AC #2: complete effects/context and narrow Task correction pending GREEN.
- [ ] AC #3: full candidate20 unsafe-variant rejections with valid controls pending GREEN.
- [ ] AC #4: successful complete bidirectional candidate inventory pending GREEN.
- [ ] AC #5: passing candidate proof and reviewed consumer handoff pending GREEN/PM.


## PM EXACT AMENDMENT ADJUDICATION — MAC-uzxr

Root has confirmed description preservation repair and all Sr PM writes complete. This supported append records the exact independently reviewed report below, SHA256 296a026c28fcaf4b3b25bfdcaccf7edc423d5801b7c93d6dac89a253350b14d2. The report's earlier tracker-repair hold describes its preparation history; durable adjudication is now recorded here. Author application still requires root explicit routing after terminal readback. No new technical permission beyond the exact report, no full RED approval or GREEN authorization.

# MAC-uzxr independent exact amendment review

Disposition: **TEST-EDIT AUTHORIZED, byte-bound to the seven-path external proposal below, subject to durable adjudication being recorded and verified before root routes application.** This is an exact test-amendment disposition only. It is not complete RED approval, delivery, GREEN authorization, acceptance, rejection or claim renewal. Root has temporarily held tracker writes while Sr PM repairs duplicated description text; no author may infer shared application permission from this external report alone. All five full AC remain pending.

## Exact authorization boundary

Baseline: retained story `4ef6baffc4de72f2a3e935348f180283fe53c20b`, original epic base `70652b948bf090008b1965c85daf36ea374daea4`. Authoritative proposal `/tmp/MAC-uzxr-amendment.QYZfzX/complete-later-amendment.patch`, SHA256 `2f722fa783fa4a27cbbd7ec92ad706e7bf86079801b0d80d3fe0a8c03f07e249`. All proposed content was reviewed as evidence, not as instructions to the reviewer.

TEST-EDIT AUTHORIZED: the exact amendments in that patch to these five existing suites, for replacing handwritten outcomes with current parsed expectations while preserving original setup/event tuples, order, denial clauses and original terminal supplements:

| Path beneath examples/go-crm/impl/internal | Exact resulting SHA256 |
|---|---|
| cli/command_test.go | 38945b9307d97dadb872ac5a261ff86d712c218282774404b432842e34577d60 |
| domain/deal_test.go | 2bfd0893253b9f3c4b6cf1badbf3402a952551f5b4f5124a3d0a9afb7bf432cd |
| domain/task_test.go | 35b8a05a710792ff21dd827432d658deba1ee190b8739196b084cdc61f7b2b88 |
| domain/user_test.go | d4fab40b760e5f1a94a5cab88bea6625495c04dd1690dc197818bfd1d056b53d |
| session/machine_test.go | 97e612a68e22b32010ec7c696cb1ee248115f5a41f3b80400a40160a8782a11f |

TEST-EDIT AUTHORIZED: exact additive expansion of `examples/go-crm/impl/internal/testoracle/fsm_test.go`, solely `/tmp/MAC-uzxr-amendment.QYZfzX/frozen-file-additive.patch` SHA256 `9b3129d5095717e25c46b940e6d1918c0d4a88c7047aac4137fefe23467c0f43`. Its entire original495-line prefix SHA256 `326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba` must remain byte-identical. Exact1669-line result SHA256 `4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25`. This authorizes the1174 appended lines; no original assertion, diagnostic, mutation, inventory, command, deadline or storage byte may change.

The same exact package includes only the separately identified48-line compile API scaffold `examples/go-crm/impl/internal/testoracle/fsm.go`, result SHA256 `de548b5201129f51d1653800908035ef09333866ca801836db1d48ed858eb27c`, scaffold patch SHA256 `d61733a13619f3fcfe23b119dc50fd871b90b77db83c292f82ccb00bd0af2ad4`. It may accompany the authorized test freeze as the declared minimum compile surface. Its ErrScaffold behavior is explicitly NONQUALIFYING; this is no permission to implement a whole parser, alter API shape, change production or claim completed RED. No eighth-path production hunk is authorized now. Task production remains SHA256 `39af26a093f855acf269da153915fbecf6e78c0f3dbba85b087bf4b83586e35b`.

Use a separate commit whose subject carries literal `[test-edit-authorized]` and `tdd-red` for the complete expanded test freeze. Preserve original998a5a3, storage4ef6baf and their proof artifacts. Recompute exact per-file SHA256 and the original495-line prefix after application and commit. Any changed byte beyond this patch, even a further additive test assertion in the frozen file, returns for exact review. No blanket parser authorization or RED-DISPUTE waiver is granted.

## Independent evidence and reconstruction

Read the complete canonical description/five AC and authoritative chronological Notes before the concurrent forecast edit. Read pm_acceptor, developer, nd, codebase-memory and vault-knowledge skills. This is the explicitly bounded amendment phase, so their final-delivery transitions and implementation actions do not apply. Read all four governing historical reports/appendix, complete REVIEW.md/CONTRACT.md, all proposed behavioral helper/assertion/mutation source, full exact mutation definitions and source-hash inventory. CONTRACT.md SHA256 `b3f444c5398b558916dec80a4dacf8c651420b6aa36d876326d606312baf8cdd`; exact mutations SHA256 `5808dd7a440a7a66c7cf161d9127757db5f456870712e53c1b75a109f1a3e5b0`.

Graph-first discovery used Users-ramirosalas-workspace-machinery, ready generation2026-09-06T02:42:16Z. Coverage checked all18 scope paths and four supporting production/authz paths. Existing files reported metadata_match/no recorded issue; new testoracle files are missing from the main graph. Receiver collisions and branch freshness mean this metadata is best effort only; exact committed git objects supplied the bounded authoritative source. No reindex, developer-worktree read or graph-based completeness claim.

Independent external audit reads git objects directly, not author snapshots as authority. It verified all98 baseline-go-crm files against4ef6baf, all listed input/result hashes, and all unrelated proposed files unchanged. It rebuilt all197 Markdown identities, stimulus SHA tags, corresponding JSON transitions, all47 state kinds/entry/exit declarations and118 source/trigger ordered groups. Witness/row counts independently matched Deal75/58, Task35/31, User20/20, Session60/60, CommandExecution28/28. All218 original label/setup/event expressions and order matched exactly; each original five-field tuple remains in the appendix. All218 native mandatory names, including slash-bearing labels, matched the actual original labels with Go space normalization. Both original terminal supplement bodies are byte-identical.

The audit separately joined the author's actual218 pre-Fire diagnostic payloads against the reconstructed git model, requiring exact guard-key sets and first-enabled selection to match every explicit rowID. Five diagnostic top-level PASS and zero FAIL/SKIP are corroborated from the actual log. This does not reclassify the builder-only run as Fire execution or parser proof. The diagnostic log SHA256 is `00cf14a6ca5316d1579b9bb291669bd42d0922e50ef52f3885c9f23c4183bb34`; compile-only log `ee6a4e3867f6c435c79e5c112c0e551bee67a6e698d7509951b563dcbc27cfba` remains zero behavioral tests/NONQUALIFYING.

Independent Go AST extraction of the final laterNativeCases produced exactly17 records, deeply equal to the supplied exact JSON. Every currently applicable replacement has exactly one anchor. The two future Task anchors have zero current matches as intentionally required; no repaired production reference was manufactured. All coherent new-row/trigger/guard/source mutants independently preserve valid stimulus hashes and matching MD/JSON next/actions. An in-memory application of every complete-patch hunk to exact git bytes produced exactly all seven proposed outputs. `git apply --check` against an owned git archive also exited0.

Raw independent audit: `/tmp/MAC-uzxr-PM-amendment.DFEjTI/audit.jsonl`, SHA256 `f73311bbd9611e33b4d56c17e46fa965b932d0ec814ced7ccab754cc47f3571c`; script `audit.cjs`, SHA256 `e4d01cc7a4e7a9dcd029388bfd8af239a05923131e01d76d632a9b4fdb2a785e`. AST source `mutations.go`, SHA256 `c7af6895e208c8782e4675ea176fa755dd247e87fed61fa7e374f1a8e6bc3ea1`; extracted `mutations.json`, SHA256 `9b54519cf1a6f98572ada2a80ecdbf7157f10c0d34ce870461dbae8a1a4dcb4b`. Final pipefail audit exited0, tool wall1.264907959s. No product test was rerun for this source authorization; this is not new runtime conformance evidence.

One read-only guessed authorizer.go path failed because the file is authz.go; graph discovery resolved it. One independent `go run` extractor invocation incorrectly treated the target `_test.go` as source; compiling the external extractor then invoking it corrected that diagnostic command. Neither failure is a product defect or RED evidence. `git diff --no-index --numstat` returned1 as expected for differences, not a failing test.

## Six prior amendment requirements adjudicated

1. **Actual inputs/guards:** all five typed builders derive source/trigger/guard truth from pre-Fire objects/events; only Witness.RowID uses the explicit mapping. Domain formulas match the independently inspected authorization semantics. Deal amount/date/write/authority denials remain separate; Task caller authority, source scope and target scope remain separate, including fixtures with several false facts. Named clauses verify the stated denial. Unknown source/events and missing fact keys fail visibly. Session expiry uses one pre-Fire time and its preserved one-hour-separated fixtures; the boundary exclusion prevents time ambiguity. CLI argv is a bounded fixture vocabulary; callback outcomes are tied to the actual withAuthorize argument, called with actual command fields twice and checked for result/purity with a copied Args slice. It does not use a production guard as its own expected result.

2. **Closed parser contract:** the declared Parse/Load/Bind/Finish/Check surface has no product Machinery import or generic engine seam. Concrete tests exercise committed current MD and JSON, strict headings/tables/footer, malformed/duplicate rows and states, duplicate JSON keys/trailing values, unknown fields and specified metadata types/positions, source/target/action/guard disagreements, source/trigger key facts, ordered guard priority, unused/duplicate witnesses and current Load rereads. Root context and allowed metadata remain data; no arbitrary underscore exemption is licensed. The current four-letter/six-hex dialect is expressly bounded and future collision extensions must fail visibly. The contract's full closed grammar remains a GREEN implementation/review obligation; the present scaffold implements none of it and no parser closure is claimed proved at runtime.

3. **Effects/context:** explicit tests require ordered source-exit + transition + target-entry for external transitions; internal is transition-only with source preserved. Explicit self, empty transition with entry, no construction entry and no recursive always/invoke step are pinned. Repeated expected actions retain multiplicity; missing/reordered/duplicated/unrelated actual actions fail. Task seeds/checks Rejection for fallback Cancelled, normal Cancelled and Done. The seed follows binding but does not change any guard fact. Name-only Task forgery will reach the real context check after exact action success; removing both call and name must instead fail the exact action diagnostic. Original nonterminal prior routes/terminal supplements are preserved, and the new three-case non-always supplement checks unchanged state/prior/rejection and empty effects.

4. **Actual execution inventory:** wrappers register all witnesses before any row Fire, Finish their registration inventory, invoke real Fire once per original case, assert parsed state/actions and Task context where relevant, verify actual native name, and only then emit observations. The candidate verifier accepts actual child bytes only, enforces process/package completion, zero skips/failures, exact once-run/once-terminal identities, registration before execution, observation during that exact native run before PASS, full parsed row identity and actual ordered state/actions. It compares rows/observations and native names/registrations both directions. Original218 are mandatory preserved names in the dynamic candidate inventory, not a fixed row-count test; original terminal and new non-always supplements are explicit separate evidence. Existing sensitivity controls deliberately retain their exact frozen full-suite inventories. Ordinary logs remain unauthenticated execution evidence.

5. **Exact remaining controls/mutations:** nine structural cases must fail before Fire leaves with the named parse/binding/unused diagnostics; three semantic cases must reach the precise action/context leaf; five inventory corruptions preserve native assertion PASS and require the same actual-output verifier to reject missing/duplicate/unknown records. All controls use the same unchanged affected full suite as their paired variant; none substitutes an already-failing full Task suite for a passing unaffected control. Exact future Task anchor is restricted to the already-scoped fallback and must match once after GREEN. No source fault API, fake process return, synthetic native event stream, parser-output shortcut or shared mutation is proposed. Diagnostic reachability is structurally justified, not yet executed/credited for the candidate.

6. **Freeze/cost/lifecycle:** exact prefix preservation and patch reconstruction are verified. Independently measured five-suite changed LOC1228, appended parser/inventory/native assertions1174, scaffold48:2120 additions/330 deletions,2450 later changed LOC/net1790. With503 historical changed LOC this is2953 cumulative; original-base aggregate2945 is a different measure. Sr PM owns canonical forecast text, without safety-proof trimming. Existing child Go120s/context125s/WaitDelay5s and outer context270s/Go300s bounds remain; children are sequential and do not select meta-tests. New proof directories use default TempDir cleanup or the existing explicit export option. No new skip/source selector, unconditional retention or cleanup of historical proof is authorized.

No blocking defect was established in this exact amendment package. This conclusion is limited to the reviewed test design/edits, not unwritten parser correctness or future candidate outcomes.

## Mandatory next checkpoint, not bypassed

After root confirms durable adjudication and routes the exact patch, the author must stop again after the separate marked commit, fresh source/hash inventory and finite genuine RED evidence. Preserve and present the independently validated original semantic history: original998a5a3 three intended outer failures/200 childPASS, historical225 baseline, and validated storage4ef6baf both modes. The corresponding complete reports remain `/tmp/MAC-uzxr-PM-initial-red-replay.md` SHA256 `09291b1d33e9a7c77981c76f2cba2b01415eed3695de5d44f5c650fd31d119b6` and `/tmp/MAC-uzxr-PM-storage-verification.md` SHA256 `b52cda2189988e9516cc10c59cda850e39f82502405eaec75ba0fdcc15bbd184`. Governing scope/appendix hashes remain a90e7be8958cbb692f4d7595889d67023733b4eb3c58878012bb2a9e41930202 and d9976dcd961d178e3fc7e1e68059125e592a152cb481e602200c58c1eb8bfb36.

Final RED review must explicitly account for both the genuine missing-sensitivity evidence on unchanged old suites and the separately confirmed Task entry defect; scaffolding, inability to reach a control, missing future anchors, compile-only, empty runs, setup errors or deadline failures cannot substitute. Run the added contract groups with finite declared commands and classify their actual outcomes honestly; a currently failed scaffold control cannot earn semantic or inventory credit. The final independent PM decides whether combined evidence supports approve-red; it is not preapproved here. GREEN remains a separate future dispatch restricted to the bounded parser and exact Task fallback, with all frozen tests unchanged. Candidate success and all17+3 rejection cases, complete executed inventory, cleanup/export behavior and actual finite timing remain pending.

## nd_contract
status: in_progress

### evidence
- Exact external test amendment technically authorized as byte-bound above; durable tracker adjudication intentionally held by root pending separate description preservation repair.
- Independent source/tuple/model/AST/hash/patch audit PASS; no shared source, test, example, worktree, binary, installed asset, service, remote or claim mutations.
- No RED approval, delivery, acceptance, rejection transition, source implementation, product Paivot dependency or consumer renewal.

### proof
- [ ] AC #1: implemented current parser and complete binding execution remain pending.
- [ ] AC #2: candidate complete effects/context behavior and narrow Task repair remain pending.
- [ ] AC #3: preserved initial genuine RED; final candidate unsafe-variant rejection remains pending.
- [ ] AC #4: complete successful candidate row/witness/native inventory remains pending.
- [ ] AC #5: full focused candidate proof and consumer handoff remain pending.


## External complete amendment proposal checkpoint

The complete external proposal against clean retained story commit 4ef6baffc4de72f2a3e935348f180283fe53c20b is ready for independent review. No shared source edits, commits, production Task fix, delivery or approve-red transition performed. All eight paths/five AC unchanged; root routes the measured cost to SrPM.

## nd_contract
status: in_progress

### evidence
- Authoritative review index /tmp/MAC-uzxr-amendment.QYZfzX/REVIEW.md; closed contract CONTRACT.md; source-hashes.json contains baseline/proposed seven paths, unchanged Task and ten design inputs.
- Full unapplied patch /tmp/MAC-uzxr-amendment.QYZfzX/complete-later-amendment.patch SHA256 2f722fa783fa4a27cbbd7ec92ad706e7bf86079801b0d80d3fe0a8c03f07e249. Separate frozen-file-additive.patch SHA256 9b3129d5095717e25c46b940e6d1918c0d4a88c7047aac4137fefe23467c0f43; exact initial semantic/storage495-line prefix SHA256326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba preserved.
- Measured git diff --no-index --numstat: CLI192add61del, Deal221add107del, Tasktests175add47del, User117add33del, Session193add82del, scaffold48add, frozen-test append1174add. Later proposal2120add330del=2450changed/net1790. Historical initial RED/storage503changed remains separate. Estimates are not canonical scope edits.
- Scratch read-only git apply --check complete-later-amendment.patch against copied baseline exited0. Actual pre-Fire builder diagnostic5top-levelPASS0SKIP emits218facts; author audit reconciles all197rows/47states/118orderedgroups/stablehashes, exact tuple preservation and unchanged original terminal bodies. Diagnostic/audit explicitly NONQUALIFYING: no parser or Fire conformance execution in diagnostic copy.
- Scratch compile-only go test -count=1 -run ^$ -timeout=120s on four packages -json exited0 with ZERO behavioral tests; NONQUALIFYING. Minimum48-line API scaffold returns ErrScaffold; no whole parser implementation or fabricated passing Task baseline. Initial real semantic RED998a5a3 and independent storage validation4ef6baf remain separate actual evidence.
- Exact remaining17 AST-extracted mutation hunks at exact-remaining-mutations.json SHA2565808dd7a440a7a66c7cf161d9127757db5f456870712e53c1b75a109f1a3e5b0:9structural/3semantic/5actual-output inventory variants, each matched full affected-suite controls. Actual candidate success inventory is proposed, not yet passing evidence. Future Task anchor remains absent before authorized GREEN.
- No background processes, external services, remote/install/preflight changes, Paivot product dependency or claim renewal. Separate exact TEST-EDIT AUTHORIZED review remains required before shared application.

### proof
- [ ] AC #1: Complete parsed current-row conformance remains pending approved amendment and real GREEN implementation/replay.
- [ ] AC #2: Closed structural/witness mapping rejection assertions proposed; final candidate execution pending.
- [ ] AC #3: Complete ordered effects and actual Task entry-context assertions proposed; real repair and sensitivity pending.
- [ ] AC #4: Initial genuine semantic RED independently validated; complete remaining native/inventory sensitivity pending final freeze and replay.
- [ ] AC #5: All218 witnesses preserved in external audit; complete successful candidate row/witness/native bidirectional inventory and final review pending.

## nd_contract
status: in_progress

### evidence
- Bounded exact storage correction independently verified in both actual modes, original semantic RED preserved. Report/hash and raw proof paths above.
- Default cleanup/exportretention meets exact authorization; all behavioralassertionsunchanged. All later source/test/helper edits still held pending concrete amendmentreview.
- Claim/worktree retained in_progress,hard-tdd,dev-MAC-uzxr. Only own detached replay/externalreport/tracker writes; no productsource/remote/preflight/install/Docker/service/privateauditwaiver/subagents/backgroundprocesses.

### proof
- [ ] AC #1: complete parser and closed actual witnessbinding pending.
- [ ] AC #2: complete effects/context and Taskrepair pending.
- [ ] AC #3: initial genuineRED/storagecheckpoint verified; candidate unsafevariantrejection pending.
- [ ] AC #4: dynamic candidate inventorypending; default raw childfiles intentionally cleaned.
- [ ] AC #5: complete passing candidateproof/consumerhandoffpending.

## nd_contract
status: in_progress

### evidence
- Storage-only TEST-EDIT AUTHORIZED repair committed4ef6baffc4de72f2a3e935348f180283fe53c20b with [test-edit-authorized]; original semantic tdd-red998a5a3 preserved. Exact independent review /tmp/MAC-uzxr-PM-initial-red-replay.md09291b1d33e9a7c77981c76f2cba2b01415eed3695de5d44f5c650fd31d119b6 fully read. Only flag import/declaration/proof allocation-log changed; fsm_test.go14add4delete,18changedLOC/net+10,SHA256326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba.
- Actual default and explicit export go test -count=1 -timeout=300s ./internal/testoracle -run ^TestFSMNativeOracleSensitivity$ -json commands from impl,offline local environment and pipefail:both exit1 with3 intended outerFAIL,200 childPASS0FAIL0SKIP. Default wall3.205103916s/package3.115s;export wall2.972808833s/package2.930s. Default emitted proof path absent after run; export retained21files; all owned fixture cwd paths removed.
- Full external proof /tmp/MAC-uzxr-storage-review.dGuJPh/checkpoint.md. Default log SHA2562954c70326cfacc4a769877492f9c6e371411ab5247c3e2ddb91e4497906a31b;exporta530a99f92b9f4509207d93c028051886993d826d1393772c9877cd622ada954. Export proof crm-fsm-native-proof-2940413560 under review dir retains6receipts/raw logs,46control hashes each,exact2/2/1mutation files. Default child raw evidence intentionally cleaned; outer actual summaries retained.
- Read-only exact reconstruction audit PASS:only3authorized edits over998a5a3; all semantic assertions/mutations/inventories/helpers/commands/deadlines byte-identical. Verified export command/package/run-terminal inventories/raw hashes/source hashes/reconstructed mutations and cleanup. audit.cjs4068acff248b538ef2fc100bd9404dc1535be6b80f03979904f46363ac55241d; audit.jsonlf2bc73a52785197faa5db00c79a7839dd37403de9de415b93e6db76d96b5dc2d.
- gofmt/diff-check/scoped pvg verify PASS1file0issues; worktree clean. Original logs preserved, no other shared edits or source/product/remote/service/toolchain/consumer changes. STOP for independent storage-repair verification. Complete external later amendment outstanding; no full story delivery/approve-red/acceptance.

### proof
- [ ] AC #1: parsed current rows and closed actual guard/event bindings pending.
- [ ] AC #2: complete effects/context assertions and Task GREEN repair pending.
- [ ] AC #3: initial semantic RED validated; storage repair committed awaiting verification; full sensitivity pending.
- [ ] AC #4: full candidate successful execution inventory pending.
- [ ] AC #5: focused corrected candidate proof and consumer handoff pending.

## nd_contract
status: in_progress

### evidence
- Initial998a5a3 semantic RED independently replayed and validated with exact manifests/native identities; full report/hash above. Graph readygeneration02:42:16Z best effort, new branch test absent from main graph so full committed source fallback.
- Exact storage-only repair authorized above; not implemented by PM. HistoricalRED preserved; all complete later five-suite/parser/helper/amendment and cost decisions HELD. Canonical8paths/5AC unchanged.
- Only own detached replay artifacts/external review/tracker writes. Main clean; developer claim/worktree retained, no product/source/service/remote/toolchain changes or background processes.

### proof
- [ ] AC #1: parser and closed actual witness binding pending.
- [ ] AC #2: complete effects and actual Task context repair pending.
- [ ] AC #3: initial missing-sensitivity RED validated; candidate successful unsafe rejection/missing-call proof pending.
- [ ] AC #4: historical218+7 baseline replayed; candidate dynamic successful-row inventory pending.
- [ ] AC #5: candidate passing focused suite/sensitivity pending; no claim renewal.

## nd_contract
status: in_progress

### evidence
- Initial independent-authorized semantic RED committed998a5a3b3107292365c7c3e6ddb85c072c534145,parent70652b948. Only new internal/testoracle/fsm_test.go485additions0deletions;changedLOC485/net+485,SHA256d17a16229e9139d4f9c72e9d2f6fc04801d1b7f7246ac58ffbb092f04656d396. Old suites/production/design/modules unchanged; worktree clean. Authorization /tmp/MAC-uzxr-PM-scope-review.md a90e7be8958cbb692f4d7595889d67023733b4eb3c58878012bb2a9e41930202 fully read.
- Real go test -count=1 -timeout=300s ./internal/testoracle -run ^TestFSMNativeOracleSensitivity$ -json from impl: actual exit1;3 enclosing leaves FAIL exactly unsafe native variant was accepted. Full User controls/mutants20leaves each for2cases,Session60each=>200 native PASS0FAIL0SKIP across6children,all child exit0. No compile/API/setup/timeout outcome credited. Toolchain go1.27.1darwin/arm64,final command wall4.756s including verify.
- Raw final /tmp/MAC-uzxr-initial-red-frozen.jsonl SHA256dfbfc3a81e32d75463646dc82db5a825e74f37b38ae3d3942491372517d13c94. Complete raw children/input manifests/exact mutations/command-exit-duration JSON retained /tmp/MAC-uzxr-initial-red-native-proof. Detailed review handoff /tmp/MAC-uzxr-initial-red-checkpoint.md. Native original all5+2terminal supplements baseline225PASS0FAIL0SKIP,exit0,wall0.930s; /tmp/MAC-uzxr-original-baseline.jsonl SHA256b7bfa3f43ae9c7db0a3cbbca262381a226d2735954e3832da38662d5ec4a21c3.
- Initial Session anchor matched twice and failed setup; that attempt excluded from semantic proof. Exact fireAnonymous header anchor fixed before freeze, authorized one-line mutation unchanged. pvg verify skill flag spelling corrected to --format text --include-tests; final VERIFY1file0issues. gofmt/diff-check clean.
- STOP for independent replay of initial commit; no story delivery/approve-red/GREEN or later shared amendments. Frozen initial file requires exact separate PM authorization for any additive later patch. Complete larger external amendment still outstanding; no claim it is ready. No remote/install/preflight/service/consumer writes; claim retained.

### proof
- [ ] AC #1: full parsed current rows and actual closed guard/event bindings pending.
- [ ] AC #2: complete effects/context assertions and Task GREEN correction pending.
- [ ] AC #3: initial3 semantic RED cases committed, independent replay and eventual correct rejection plus remaining variants pending.
- [ ] AC #4: full candidate successful-execution inventory pending; native control names are not parser proof.
- [ ] AC #5: focused corrected candidate proof and consumer handoff pending.

## nd_contract
status: in_progress

### evidence
- Supported full canonical read; direct committed70652b948 read-only comparison verifies all218 original tuples/order,197 MD/JSON row identities,197 stimulus hashes,47 state declarations,118 ordered source/trigger groups. Source mapping is not actual guard/execution proof.
- internal/oracle/oracle.go StableID excludes target/actions: USER|Active|on:disable|guardAdminAuthority hashes e20d04, so proposed paired target/action variants legitimately retain generated ID. Generator collision-prefix extension means six-only parser remains explicitly bounded current dialect.
- Independent diagnostic log hash/count extraction: probe1PASS1FAIL0SKIP; wider225leaves224PASS1FAIL0SKIP, soleTASK-754183; no new runtime test/mutation run. Initial analytical zero-row selection and Ruby syntax attempt corrected; final positive47-state/118-priority verification only credited.
- Graph refreshed readygeneration2026-09-06T02:42:16Z, existing cited paths metadata_match/no_recorded_issue, new helpers missing; best effort only and exact-source fallback used. Main clean497419ab4512fcff765cd5feb27aed4c67b5608d; story ref70652b948bf090008b1965c85daf36ea374daea4. No developer worktree internals, source/test/docs/evidence/remote/service/toolchain writes. Only external review and this coordination note.
- Claim/worktree retained: in_progress,hard-tdd,dev-MAC-uzxr. All broader authoring/freeze/RED/GREEN/acceptance decisions pending; no MAC-hgz1/MAC-lnu6 claim renewal.

### proof
- [ ] AC #1: actual committed-row parser and exact closed event/guard mapping pending concrete amendment review.
- [ ] AC #2: concrete complete-effect tests and actual Task context repair pending final RED/GREEN.
- [ ] AC #3: independently replayed real semantic/extra-effect/missing-call controls pending.
- [ ] AC #4: complete dynamic successful execution inventory and preserved supplements pending.
- [ ] AC #5: complete candidate focused native and sensitivity proof pending; no consumer acceptance renewed.

## nd_contract
status: in_progress

### evidence
- READ-ONLY RED proposal checkpoint at story/MAC-uzxr 70652b948bf090008b1965c85daf36ea374daea4. Canonical story and 03:35 UTC eight-path Task extension read completely. Developer/nd/codebase-memory skills applied; parent Verify graph plus direct exact source review, coverage generation 2026-09-06T02:42:16Z best effort.
- External proposal /tmp/MAC-uzxr-RED-proposal.md and exact /tmp/MAC-uzxr-witness-inventory.md prepared for independent PM; 197 committed rows/218 existing witness tuples, Deal58/75 Task31/35 User20/20 Session60/60 Command28/28. No source/helper/test edits, RED commit/approval, or delivery. All original suites and shared production remain clean.
- Runtime DISCOVERED_BUG TASK-754183 reported and canonically folded into this P0 by SrPM: isolated real control TASK-67b0ff PASS; fallback Cancelled missing recordTaskClosed FAIL. Probe-pipefail.jsonl SHA25690e3d11c3df38725d77637bc187d7e27dcf0a2b65f0934e6b71c785b177e783c, actual exit1,1PASS1FAIL0SKIP,package0.286s.
- Additional copy-only hand-reconciled exact-effect reconnaissance all5 suites +2 terminal supplements:224PASS1FAIL0SKIP/225leaves, sole TASK-754183. Log /tmp/MAC-uzxr-task-entry-probe.FGj02Q/exact-effects.jsonl SHA256c6ce626ee381f97d020a2bea090d1a626e9b9adfa421c75ffd877cdfb8ee42a4,actual exit1,outer0.917s. Neither probe is parser-backed approved RED.
- Proposal explicitly requires real Task Rejection clearing and name-only missing-call variant; unchanged old-suite semantic mutation failure must establish parser RED before scaffold errors. Eight-path canonical750-1150LOC forecast may grow to1250-1750 for closed guard bindings/strict paired-source parser/real subprocess controls; material growth submitted for independent review, no safety coverage trimmed. Claim/worktree retained for checkpoint; no remote/toolchain/service/consumer writes.

### proof
- [ ] AC #1: parsed current rows and closed actual witness binding pending independent exact amendment review.
- [ ] AC #2: complete entry/exit effects and actual Task context effect pending RED/GREEN.
- [ ] AC #3: genuine paired subprocess oracle/state/actions/extra-effect/name-only sensitivity pending.
- [ ] AC #4: dynamic bidirectional actual execution inventory pending; external source inventory is not execution proof.
- [ ] AC #5: candidate focused suite proof pending; no claim renewed.

## nd_contract
status: in_progress

### evidence
- Sr PM bug_triage: confirmed TASK-754183 missing Cancelled entry action is folded into existing P0 MAC-uzxr by explicit root authorization; no new issue or dependency. Eight owned paths/~750-1150 LOC, only added production path task.go and only fireRolledBack fallback repair after independent RED approval.
- Exact70652b948 source read; Task.Fire(evt TaskEvent) Effect, fireRolledBack(evt TaskEvent) Effect and recordTaskClosed() verified. Existing action clears Rejection; canonical AC2 proof must assert actual context mutation, Cancelled and ordered actions, not merely a returned name.
- Isolated probe read/hash matched: task_entry_probe_test.go f6809da70ce0130f79020f8d032194c878e3b17acdeebfaa12f22ca54bc97e84; task.go copied/committed39af26a093f855acf269da153915fbecf6e78c0f3dbba85b087bf4b83586e35b; probe-pipefail.jsonl90e3d11c3df38725d77637bc187d7e27dcf0a2b65f0934e6b71c785b177e783c. Actual corrected exit1:1PASS normal TASK-67b0ff,1FAIL fallback TASK-754183,0SKIP,package0.286s; initial tee status0 is not success.
- Additional copied-test exact-effects.jsonl hashc6ce626ee381f97d020a2bea090d1a626e9b9adfa421c75ffd877cdfb8ee42a4 and four copy-only diffs read:225 leaves224PASS/1FAIL samefallback/0SKIP,author-reported0.917s wall,actual exit1. These hand-reconciled diagnostics do not prove the missing parser is detected and are NOT approved/frozen RED.
- Original five AC compared byte-exact after edit; unique canonical headings and intact quoted commands read back. Claims/status/labels preserved: in_progress,hard-tdd,dev-MAC-uzxr. No source/test/docs/evidence/golden edits, no production fix, no review approval or delivery by Sr PM.
- All five existing-test amendments and concrete parser/reconciliation/sensitivity design still require exact independent PM authorization before edits. No production changes until RED approved; no blanket repair permission. MAC-hgz1 consumes accepted parser/test AND corrected Task source, MAC-ou97 remains blocked, MAC-lnu6 evidence hold unchanged.
- Scoped lint37 issues PASS0 errors/0 findings; no dependency cycles. RTM22 stories,4 closed,0 extracted requirements: structural result only, NOT five-AC proof. Root main497419a remains clean.
- Self-review fixed scope-honesty defect by including directly related tiny production correction; no new API, generated-oracle changes, broad rewrite or unsupported additional bug claimed.

### proof
- [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
- [ ] AC #2: entry/exit-aware complete effects plus TASK-754183 actual context-effect repair and safe controls.
- [ ] AC #3: real next-state/action/extra-effect and missing-call sensitivity, independently frozen before GREEN.
- [ ] AC #4: complete executed-row inventory with terminal supplements and honest failure attribution.
- [ ] AC #5: full focused native candidate proof and reviewed consumer scope handoff, without automatic claim renewal.


Prior canonical retained verbatim as historical quoted context:
> ## USER INTENT
> A current example conformance claim must describe real tests bound to committed FSM expectations, not handwritten expectations that merely reuse stable row names.
> 
> ## Context (Embedded)
> Confirmed absent wholesale FSM parser in Go CRM's five transition suites. PM's actual replay passed 218 leaves (domain130/session60/cli28), zero failures/skips, with all197 committed IDs present in executed names. This proves bounded existing transition execution, not linkage to parsed current oracle expectations. Raw /tmp/MAC-lnu6-pm-go-crm-transitions-713184d.jsonl SHA256 8a88436b9d6212a45eb422c4f4d609902679a15343ba796b1c72174aff03a662; wall1.503486s.
> internal/gates/attest.go:151 requires wholesale committed oracle parsing plus per-row next state AND expected actions. Five handwritten test tables call real Fire/State but do not parse the committed FSM oracle tables. Existing firedInOrder loops accept expected actions as a subsequence and any actual list for an empty expectation. Source establishes that adequacy gap; no extra-effect mutant was executed by the discovery review. Registry wording does not expressly require exact-list equality. Entry/exit effects are legitimate and MUST be reconciled from committed semantics rather than blindly replacing containment with transition-column equality.
> 
> ## Ownership
> Seven forecast paths:
> - examples/go-crm/impl/internal/domain/deal_test.go
> - examples/go-crm/impl/internal/domain/task_test.go
> - examples/go-crm/impl/internal/domain/user_test.go
> - examples/go-crm/impl/internal/session/machine_test.go
> - examples/go-crm/impl/internal/cli/command_test.go
> - examples/go-crm/impl/internal/testoracle/fsm.go (new test-support parser/reconciliation only)
> - examples/go-crm/impl/internal/testoracle/fsm_test.go (new parser and real subprocess sensitivity proof)
> Only exact independently PM-authorized existing test/helper amendments may change the five old files. Preserve their real inputs/guard branch cases, terminal supplements and safety intentions. New test-support paths are reviewed outputs, not assumed existing APIs; concrete helper shape/use must be reviewed before authoring if it affects frozen assertions.
> Read-only: five committed design/machines/{Deal,Task,User,Session,CommandExecution}.oracle.md and .machine.json files; production Fire implementations; root Machinery parser/gate/attestation code; all BUILD/attestations/acceptance/go.mod/go.sum. No generated-oracle edits or fixture weakening to make tests pass. Unexpected production mismatch returns to Sr PM; no blanket implementation-fix ownership. You are not alone; preserve other work.
> 
> ## Boundary Map
> PRODUCES:
> - Five exact transition test files -> actual Fire/State conformance driven by parsed committed row expectations with complete row/guard input reconciliation.
> - internal/testoracle/fsm.go under the Go CRM impl -> bounded test-support parser and expectation reconciliation, not a new Machinery production API.
> - internal/testoracle/fsm_test.go under the Go CRM impl -> matched real native controls and adversarial parser/assertion sensitivity.
> CONSUMES:
> - Existing Go CRM transition interfaces.
>   source: TestDealTransitions/TestTaskTransitions/TestUserTransitions/TestSessionTransitions/TestCommandExecutionTransitions instantiate real domain/session/CLI machines, call Fire on concrete events and inspect State and Effect.Actions.
> - Existing committed oracle grammar and semantics.
>   source: each .oracle.md contains State entry / exit actions plus Transitions columns test id, stable id, source, trigger, guard, target, actions; corresponding .machine.json defines ordered guarded alternatives, internal/external transitions and entry/exit actions. Parse these actual committed sources, not matrix citations only.
> - Existing claim vocabulary.
>   source: internal/gates/attest.go gt.conformance-test-shape requires wholesale parser, next state and expected actions; no core claim-model edit here.
> 
> ### Story Acceptance Criteria
> 1. All five FSM conformance suites parse the committed oracle tables during real execution and derive expected next state/actions from them, with an explicit closed mapping from each current row/guard branch to actual setup/event inputs. Check source/trigger/guard/target identity, not only stable-ID string membership. Missing, duplicate, malformed, newly added unmapped or unused rows fail with named diagnostics; preserve complete existing guard-clause coverage and additional legitimate supplement cases.
> 2. Reconcile full observable action expectations using the actual committed transition AND entry/exit semantics, including internal transitions and fallback/guard priority. Independently review the concrete reconciliation before freezing revised assertions. Correct legitimate entry/exit actions pass; missing, reordered, duplicated or unrelated extra effects fail, including an empty-transition-action row. Do not claim this exactness was an explicit old registry quotation; it is required adequacy proof for this repaired current claim.
> 3. Real native subprocess sensitivity on isolated copied source/oracle fixtures demonstrates that changing committed expected next state or actions without changing implementation causes the intended conformance failure, not parse/setup failure. Separately inject a reviewed extra-effect implementation variant and require the same frozen assertions to reject it, with unchanged implementation/oracle controls passing. No returned fake process result, warning-only negative, modified shared oracle or production fault API.
> 4. Report bidirectional committed-row versus actual executed-row inventory for all five machines; current baseline197 IDs/218 leaves is historical, not a hardcoded ceiling. All selected rows/guard witnesses terminate with expected real outcomes and zero skips; inspect real next-state/action diagnostics and preserve terminal supplements. A passing parser unit test or named-ID count alone is insufficient.
> 5. The actual focused native Go CRM suite and parser/sensitivity proof pass on the delivered candidate, old unrelated tests remain unchanged, and the consumer migration can use the exact reviewed source/test scope and logs to assess a current claim. No automatic attestation, implementation acceptance renewal, full-suite or authenticated-execution assertion is made by this story.
> 
> ## Testing Requirements
> Unit tests plus Integration tests: MANDATORY (no mocks of Fire, compiler, filesystem, process output or assertion result). Service-free native Go and real temporary copies/subprocesses; no Docker or Paivot dependency. Review exact old-test amendments before RED writes; test-support scaffolding may support a qualifying RED only when the unchanged old suites demonstrate the specific missing parser/sensitivity behavior. Compile/API/fixture/timeout failures are not RED.
> Focused existing control from examples/go-crm/impl: go test -count=1 -timeout=120s ./internal/domain ./internal/session ./internal/cli -run '^(TestDealTransitions|TestTaskTransitions|TestUserTransitions|TestSessionTransitions|TestCommandExecutionTransitions)$' -json
> Run new internal/testoracle tests explicitly plus relevant existing terminal supplements. Bound outer meta-tests and avoid recursive selection of themselves. Propose exact new names/commands and mutation hunks for independent RED review; freeze unchanged-control and unsafe variants separately with SHA/input hashes. No hidden skip, generated expected-output replacement, or acceptance based on one error. Record host/toolchain, exact source/reference/mutant SHAs, commands, leaf/row inventories, logs and durations.
> 
> ## OUT OF SCOPE
> - Core plan/current/historical schema and freshness custody belong to MAC-p7jd; future authenticated runner/replay enforcement belongs to MAC-l7m0/MAC-vx24.
> - BUILD/attestation consumer migration is a separately sequenced repair; no simultaneous ownership of MAC-lnu6 files.
> - New application behavior, root parser/generator edits, external services and global preflight.
> 
> ## DIFF BUDGET
> - Seven paths, forecast700-1100 changed LOC, largely explicit row/guard reconciliation and paired subprocess tests. Report actual per-file cost; independent review required for material growth, never trim safety witnesses.
> - Existing focused baseline1.503486s; new real copied-tree builds/mutations cost more. Initial outer proof budget5m with bounded individual120s child suites; report measured runtime rather than assuming this fit.
> 
> ## Discovered During
> MAC-lnu6 independent substantive amendment review at 713184db16a12b8c3763b4aa8f5bf025721abf22. Full report /tmp/MAC-lnu6-PM-AMENDMENT-713184d.md SHA256 6fc5223d3e5d48170f89ff9e29b634b256bfe2454b710378f08cb60d875aadf1 was read completely (227 newline-terminated lines; final content included). No runtime mutation proof beyond the explicitly reported native replay is claimed. Exact epic 70652b948 source inspected; graph generation 2026-09-06T02:42:16Z is best effort, not completeness proof.
> 
> ## MANDATORY SKILLS
> - developer; codebase-memory; pm_acceptor.
> 
> ## Delivery Requirements
> Hard TDD with independent RED and exact existing-test amendment review before edits. Preserve all unrelated tests, goldens, generated evidence and existing claims. Shared tracker is development coordination only; product remains standalone. No remote, full preflight, installed assets, user services or healthy-worktree cleanup. No source/docs/tests changed during triage. Use supported delivery; independent PM accepts.
> 
> ## nd_contract
> status: new
> 
> ### evidence
> - P0 source-established conformance parser defect plus separately identified action-adequacy gap; parent supplied independently verified native218-pass control. No new mutation run or repair performed.
> - Existing claims/tests/ownership remain held until independent exact test amendment review; no core schema duplication.
> 
> ### proof
> - [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
> - [ ] AC #2: correct entry/exit-aware complete effects and adversarial sensitivity.
> - [ ] AC #3: real expected-state/action and extra-effect unsafe variants fail for intended causes.
> - [ ] AC #4: complete executed-row inventory and preserved controls.
> - [ ] AC #5: native candidate proof without automatic claim renewal.
> 

## nd_contract
status: new

### evidence
- Created P0 under MAC-ui8a; open/unclaimed/hard-tdd. MAC-ou97 explicitly depends on this repair. No existing state/claim/test/source changes.
- Actual canonical readback confirms five AC, explicit nonoverlapping ownership and intact quoted command. Scoped lint PASS36issues/0errors/0review; dependency cycles none; RTM0extracted requirements/21stories/4closed is structural only, not AC proof.
- Independent test/handoff review and exact existing-test amendment authorization remain pending; no RED approval or implementation permission granted by triage.
- P0 source-established conformance parser defect plus separately identified action-adequacy gap; parent supplied independently verified native218-pass control. No new mutation run or repair performed.
- Existing claims/tests/ownership remain held until independent exact test amendment review; no core schema duplication.

### proof
- [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
- [ ] AC #2: correct entry/exit-aware complete effects and adversarial sensitivity.
- [ ] AC #3: real expected-state/action and extra-effect unsafe variants fail for intended causes.
- [ ] AC #4: complete executed-row inventory and preserved controls.
- [ ] AC #5: native candidate proof without automatic claim renewal.


## History
- 2026-09-06T03:03:05Z dep_added: blocks MAC-ou97
- 2026-09-06T03:17:08Z dep_added: blocks MAC-hgz1
- 2026-09-06T03:25:44Z status: open -> in_progress
- 2026-09-06T03:25:44Z auto-follows: linked to predecessor MAC-2u36
- 2026-09-06T03:25:44Z claimed by dev-MAC-uzxr
- 2026-09-06T06:07:57Z status: in_progress -> open
- 2026-09-06T06:14:21Z status: open -> in_progress
- 2026-09-06T06:14:21Z auto-follows: linked to predecessor MAC-a89e
- 2026-09-06T06:14:21Z claimed by dev-MAC-uzxr
- 2026-09-06T06:28:13Z status: in_progress -> in_progress
- 2026-09-06T06:28:13Z auto-follows: linked to predecessor MAC-p8ce
- 2026-09-06T06:51:32Z status: in_progress -> open
- 2026-09-06T06:51:32Z released by ramirosalas
- 2026-09-06T07:01:37Z status: open -> in_progress
- 2026-09-06T07:01:37Z auto-follows: linked to predecessor MAC-olrx
- 2026-09-06T07:01:37Z claimed by dev-MAC-uzxr
- 2026-09-06T07:10:06Z status: in_progress -> in_progress
- 2026-09-06T07:19:00Z status: in_progress -> open
- 2026-09-06T07:20:28Z released by ramirosalas
- 2026-09-06T07:43:22Z status: open -> in_progress
- 2026-09-06T07:43:22Z claimed by dev-MAC-uzxr

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-ou97]], [[MAC-hgz1]]
- Follows: [[MAC-2u36]], [[MAC-a89e]], [[MAC-p8ce]], [[MAC-olrx]]

## Comments

### 2026-09-06T03:05:27Z ramirosalas
## nd_contract
status: new

### evidence
- Created P0 under MAC-ui8a; open/unclaimed/hard-tdd. MAC-ou97 explicitly depends on this repair. No existing state/claim/test/source changes.
- Actual canonical readback confirms five AC, explicit nonoverlapping ownership and intact quoted command. Scoped lint PASS36issues/0errors/0review; dependency cycles none; RTM0extracted requirements/21stories/4closed is structural only, not AC proof.
- Independent test/handoff review and exact existing-test amendment authorization remain pending; no RED approval or implementation permission granted by triage.
- P0 source-established conformance parser defect plus separately identified action-adequacy gap; parent supplied independently verified native218-pass control. No new mutation run or repair performed.
- Existing claims/tests/ownership remain held until independent exact test amendment review; no core schema duplication.

### proof
- [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
- [ ] AC #2: correct entry/exit-aware complete effects and adversarial sensitivity.
- [ ] AC #3: real expected-state/action and extra-effect unsafe variants fail for intended causes.
- [ ] AC #4: complete executed-row inventory and preserved controls.
- [ ] AC #5: native candidate proof without automatic claim renewal.


### 2026-09-06T03:35:09Z ramirosalas
## nd_contract
status: in_progress

### evidence
- Sr PM bug_triage: confirmed TASK-754183 missing Cancelled entry action is folded into existing P0 MAC-uzxr by explicit root authorization; no new issue or dependency. Eight owned paths/~750-1150 LOC, only added production path task.go and only fireRolledBack fallback repair after independent RED approval.
- Exact70652b948 source read; Task.Fire(evt TaskEvent) Effect, fireRolledBack(evt TaskEvent) Effect and recordTaskClosed() verified. Existing action clears Rejection; canonical AC2 proof must assert actual context mutation, Cancelled and ordered actions, not merely a returned name.
- Isolated probe read/hash matched: task_entry_probe_test.go f6809da70ce0130f79020f8d032194c878e3b17acdeebfaa12f22ca54bc97e84; task.go copied/committed39af26a093f855acf269da153915fbecf6e78c0f3dbba85b087bf4b83586e35b; probe-pipefail.jsonl90e3d11c3df38725d77637bc187d7e27dcf0a2b65f0934e6b71c785b177e783c. Actual corrected exit1:1PASS normal TASK-67b0ff,1FAIL fallback TASK-754183,0SKIP,package0.286s; initial tee status0 is not success.
- Additional copied-test exact-effects.jsonl hashc6ce626ee381f97d020a2bea090d1a626e9b9adfa421c75ffd877cdfb8ee42a4 and four copy-only diffs read:225 leaves224PASS/1FAIL samefallback/0SKIP,author-reported0.917s wall,actual exit1. These hand-reconciled diagnostics do not prove the missing parser is detected and are NOT approved/frozen RED.
- Original five AC compared byte-exact after edit; unique canonical headings and intact quoted commands read back. Claims/status/labels preserved: in_progress,hard-tdd,dev-MAC-uzxr. No source/test/docs/evidence/golden edits, no production fix, no review approval or delivery by Sr PM.
- All five existing-test amendments and concrete parser/reconciliation/sensitivity design still require exact independent PM authorization before edits. No production changes until RED approved; no blanket repair permission. MAC-hgz1 consumes accepted parser/test AND corrected Task source, MAC-ou97 remains blocked, MAC-lnu6 evidence hold unchanged.
- Scoped lint37 issues PASS0 errors/0 findings; no dependency cycles. RTM22 stories,4 closed,0 extracted requirements: structural result only, NOT five-AC proof. Root main497419a remains clean.
- Self-review fixed scope-honesty defect by including directly related tiny production correction; no new API, generated-oracle changes, broad rewrite or unsupported additional bug claimed.

### proof
- [ ] AC #1: actual committed-row parser and closed input/guard reconciliation.
- [ ] AC #2: entry/exit-aware complete effects plus TASK-754183 actual context-effect repair and safe controls.
- [ ] AC #3: real next-state/action/extra-effect and missing-call sensitivity, independently frozen before GREEN.
- [ ] AC #4: complete executed-row inventory with terminal supplements and honest failure attribution.
- [ ] AC #5: full focused native candidate proof and reviewed consumer scope handoff, without automatic claim renewal.


### 2026-09-06T03:49:41Z ramirosalas
## nd_contract
status: in_progress

### evidence
- TERMINAL PRE-RED PM SCOPE DISPOSITION: supported append-notes readback places the new review before older historical contracts; this supported chronological comment makes the current contract unambiguous without rewriting history.
- Full independent review /tmp/MAC-uzxr-PM-scope-review.md SHA256a90e7be8958cbb692f4d7595889d67023733b4eb3c58878012bb2a9e41930202 and preceding PRE-RED PM SCOPE DISPOSITION note are authoritative scope detail. Verified unchanged status in_progress,hard-tdd,dev-MAC-uzxr.
- Authorize ONLY initial new fsm_test.go stdlib real-process stage: paired USER-e20d04 target/action and pinned Session Anonymous-resume extra-effect variants, complete unaffected per-machine User/Session controls, actual unsafe-variant-accepted RED on unchanged old suites, separate all5+2supplement baseline, bounded contexts, SHA/manifests/native outcomes. Commit tdd-red then independent replay checkpoint. No fsm.go, old-test or production writes now; no story delivery/approve-red/acceptance.
- HOLD complete later five-suite/helper amendments until exact guard/event expressions, parser/reconciliation assertions, execution-inventory verifier and all remaining mutation hunks/per-file costs reviewed. First fsm_test assertions freeze; future additive same-file patch needs separate exact PM authorization and machine-readable audit markers, none implied now. Final combined RED must be independently frozen/replayed before GREEN task.go fallback repair.
- Independent exact70652b948 source comparison:218 original ordered tuples,197 MD/JSON identities and stable stimulus hashes,47 states,118 priority groups. USER generated stable ID excludes target/actions and remains valid in proposed coherent semantic variants. Diagnostic logs independently hash/count matched1PASS1FAIL and224PASS1FAIL/0SKIP; no new runtime tests or mutations by PM. Graph generation02:42:16Z best effort, source fallback used.
- SrPM recommendation only: revise same8-path forecast provisionally1500-2200 additions+deletions or justify1250-1750 with measured decomposition;218 tuple replacements alone436 changed lines. Canonical scope/AC unchanged; no proof trimming or extra architecture.
- Only external review/coordination writes. Claim/worktree preserved; no developer-worktree internals, source/test/docs/evidence/service/remote/toolchain/consumer writes.

### proof
- [ ] AC #1: exact parser and closed actual event/guard binding pending.
- [ ] AC #2: concrete effects/context assertions and Task repair pending RED/GREEN.
- [ ] AC #3: independent native semantic/extra-effect/missing-call proof pending.
- [ ] AC #4: full actual successful execution inventory and supplements pending.
- [ ] AC #5: full focused candidate proof pending; no claim renewal.

### 2026-09-06T04:16:55Z ramirosalas
## nd_contract
status: in_progress

### evidence
- TERMINAL INITIAL RED CHECKPOINT: INITIAL_RED_REPLAY VALIDATED at998a5a3b3107292365c7c3e6ddb85c072c534145, not full RED approval/delivery/acceptance/rejection. The preceding INITIAL_RED_REPLAY note contains exact durable storage-only TEST-EDIT AUTHORIZED statements; read them before the separate repair. Report /tmp/MAC-uzxr-PM-initial-red-replay.md SHA25609291b1d33e9a7c77981c76f2cba2b01415eed3695de5d44f5c650fd31d119b6.
- Detached real replay actualexit1/3 intended outerFAIL/200 childPASS0SKIP; rawf8e10d43309778babbe609643c2adde672ad09c87305470745b706b5200dc67b. Separate225PASS baseline exact218+7identities, raw ee99c3aa35061bfc763ad6566a616920e98327b79f06e1637898fde8874f4ea1. Author and PM46-input manifests/exact2/2/1 mutations/six receipts and native terminal identities verified. Fixture cwds cleaned;197672bytes/21proof files retained unconditionally on ordinary run.
- TEST-EDIT AUTHORIZED is ONLY stdlib flag import, nativeProofParent flag and exact3rd storage block supplied in preceding note: default t.TempDir built-in cleanup/error reporting; explicit existing absolute export parent via -fsm-native-proof-dir, no execution gates or expected-outcome changes. Preserve all semantic assertions/mutations/native inventories byte-for-byte; separate [test-edit-authorized] commit, original998a5a3 and RED logs preserved. Both modes need actual3FAIL/200PASS replay plus default cleanup/export retention checks before next independent verification.
- All later five-suite/helper/parser/additive patches and fullRED/GREEN decisions remain HELD. No AC/path/architecture/canonical forecast changes. Claim and healthy developer worktree retained, statusin_progress/hard-tdd/dev-MAC-uzxr. No product/source edits by PM, remote/toolchain/service changes, subagents or running jobs.

### proof
- [ ] AC #1: parser and closed actual witness binding pending.
- [ ] AC #2: complete effects/context and Task repair pending.
- [ ] AC #3: valid initial missing-sensitivity RED; candidate rejection proof pending.
- [ ] AC #4: historical225 baseline verified; dynamic candidate inventory pending.
- [ ] AC #5: complete passing candidate proof and consumer handoff pending.

### 2026-09-06T04:25:57Z ramirosalas
## nd_contract
status: in_progress

### evidence
- TERMINAL STORAGE CHECKPOINT: STORAGE_REPAIR VALIDATED at4ef6baffc4de72f2a3e935348f180283fe53c20b; original998a5a3 preserved. Complete independent report /tmp/MAC-uzxr-PM-storage-verification.md SHA256b52cda2189988e9516cc10c59cda850e39f82502405eaec75ba0fdcc15bbd184 and preceding STORAGE_REPAIR note are authoritative detail.
- Exactly3 authorized transformations/18changedLOC, fullcandidatebytecomparison; semantic assertions/nativeinventoryunchanged. [test-edit-authorized] marker and normal verify-tdd original70652b948 base PASS2commits/no violations/no waiver.
- Independent default/export both actualexit1,3 intendedREDleafFAIL/200checkedchildPASS0SKIP. Defaultproof+6fixturecwdsabsent, rawdefaultchildfilesintentionallycleaned. Export21filesretained;6actualreceipts/rawhashes/exactnativeidentities/46inputmanifests/exact2/2/1mutations verified. PMdefault9ac83cb0f357a4aa7d529872b3ba85cd60e38a5ab19178180bf036dcca6a6b8e, PMexportbc7148f9887ef4429231d484cd565e622c299cef4496a44b00b17ff260cb6076. Author evidence separately corroborated via gitobjects, no developerworktree read.
- No further storage repair needed. Author may prepare full external laterpatch and measureddecomposition. ALL shared5suite/helper/parser/additive edits HELD for separateexactreview; fullREDapproval/GREEN/delivery/acceptance/rejection not performed. AllACpending, claim/worktreeretained, no product/remote/service/toolchainwrites.

### proof
- [ ] AC #1: parser/closed binding pending.
- [ ] AC #2: complete effects/context/Taskrepair pending.
- [ ] AC #3: initialRED/storagevalidated, candidate rejectionproofpending.
- [ ] AC #4: dynamic candidate executedinventorypending.
- [ ] AC #5: fullpassingcandidate/consumerhandoffpending.

### 2026-09-06T05:05:14Z ramirosalas
## nd_contract
status: in_progress

### evidence
- TERMINAL EXTERNAL AMENDMENT CHECKPOINT: complete unapplied package /tmp/MAC-uzxr-amendment.QYZfzX/REVIEW.md SHA2567105508cf12fa8ea73bfd9db81d97dc5a068ad08db1cae39166a13a7dca8fdc6. Preceding External complete amendment proposal checkpoint contains detailed current evidence. Supported append-notes insertion precedes older chronological comments; this final comment disambiguates latest contract without rewriting history.
- Complete seven-path patch SHA2562f722fa783fa4a27cbbd7ec92ad706e7bf86079801b0d80d3fe0a8c03f07e249, exact additive frozen-file patch9b3129d5095717e25c46b940e6d1918c0d4a88c7047aac4137fefe23467c0f43. All initial495semantic/storage lines unchanged. Retained story clean4ef6baffc4de72f2a3e935348f180283fe53c20b; Task production unchanged. No shared writes/commits, delivery or approve-red.
- Measured later stage2120add330del=2450changed/net1790 across seven proposedpaths; priorhistory503changed separate. Root/SrPM owns forecast canonicalization; five AC/eight paths unchanged.
- Scratch apply-check exit0; author audit218preservedtuple/actualpre-Firefacts to197rows/47states/118groups and frozen/terminal/source integrity PASS. Builder diagnostic5top-levelPASS0SKIP and compile-only4packages exit0/ZERO behavioral tests explicitly NONQUALIFYING. No whole parser or fabricated fixed Task baseline. Original validated genuine RED998a5a3/storage4ef6baf remain actual initial proof.
- Complete concrete builders/dialect/assertions/actual candidateinventory/17exactmutations proposed. Separate exact TEST-EDIT AUTHORIZED independent review required before shared application; all final candidate execution/GREEN pending. No subagents/background/remote/toolchain/service/product-Paivot changes; claim retained.

### proof
- [ ] AC #1: approved parser and closed binding implementation/replay pending.
- [ ] AC #2: complete effects/context and narrowly authorized Task repair pending.
- [ ] AC #3: initial genuine RED/storage validated; final candidate unsafe-variant rejection pending.
- [ ] AC #4: successful actual candidate row/witness/native inventory pending.
- [ ] AC #5: complete passing candidate proof and consumer handoff pending.

### 2026-09-06T05:26:01Z ramirosalas
## nd_contract
status: in_progress

### evidence
- SR PM MEASURED FORECAST / PRESERVATION REPAIR. Canonical eight paths and five AC unchanged; actual external proposal2450changed plus503historical=2953 cumulative beforeGREEN, aggregate2945changed. Conditional3300-3500 cumulative/3200-3400 aggregate forecast assumes prior300-450-line parser estimate and separately charges48-line scaffold replacement plus narrowTask correction. Old750-1150/provisional1500-2200 obsolete; no proof trimmed or test permission granted. First --description command succeeded but verification failed because nd treats same-level nested headings as section boundaries and retained an old canonical suffix. Original failed readback preserved in /tmp/machinery-private-triage.JF2BDG/READONLY-HANDOFF.md. Installed --body-file has the same Description semantics. Root-authorized guarded pvg nd edit bound full live raw header/body to journal export and checked an external apply_patch trial, removed ONLY duplicate canonical block under nd exclusive lock, then verified exact full body90c15690a83bd2200723fdaa8bdd27e3871e189959b4936083afe06336d35590 and identical tail/metadata except normal hash/time. Exact expected/proposed manifests and editor remain external. First hgz1 preparation-only cross-realm assertion failed before any tracker write, then root explicitly authorized scalar-array comparison correction; historical failure is not proof. Scoped structural verification after exact canonical repair: pvg lint --backlog --epic MAC-ui8a scanned37 issues,0 errors/0 review findings; pvg rtm check37 stories/19closed/0 extracted requirements; pvg nd dep cycles found none. These are structure checks, not AC proof. Claims/status/labels/dependencies retained; no source/test/example/toolchain/remote/preflight mutation or Paivot product dependency.

### proof
- [ ] AC #1: Original parsed-row/closed-witness AC remains pending independent final amendment/freeze and implementation.
- [ ] AC #2: Original complete-effect/context AC remains pending actual Task repair and candidate proof.
- [ ] AC #3: Original native unsafe-variant sensitivity AC remains pending complete frozen candidate replay.
- [ ] AC #4: Original dynamic executed-row inventory AC remains pending final proof.
- [ ] AC #5: Original candidate conformance/consumer handoff AC remains pending; no current claim renewal.

### 2026-09-06T05:31:32Z ramirosalas
## nd_contract
status: in_progress

### evidence
- TERMINAL PM EXACT AMENDMENT ADJUDICATION: TEST-EDIT AUTHORIZED only for complete-later-amendment.patch SHA2562f722fa783fa4a27cbbd7ec92ad706e7bf86079801b0d80d3fe0a8c03f07e249 against retained4ef6baffc4de72f2a3e935348f180283fe53c20b. Full exact disposition/report is preserved in preceding PM EXACT AMENDMENT ADJUDICATION Notes and /tmp/MAC-uzxr-PM-amendment.DFEjTI/REVIEW.md SHA256296a026c28fcaf4b3b25bfdcaccf7edc423d5801b7c93d6dac89a253350b14d2. Root explicit routing remains required before author applies it.
- Exact five-suite result hashes and1174-line additive frozen-file expansion are byte-bound by that report; frozen-file-additive.patch SHA2569b3129d5095717e25c46b940e6d1918c0d4a88c7047aac4137fefe23467c0f43. Preserve entire495-line prefix SHA256326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba; expanded1669-line result4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25. Only separate48-line compile scaffold fsm.go resultde548b5201129f51d1653800908035ef09333866ca801836db1d48ed858eb27c may accompany the freeze; ErrScaffold, compile-only, empty/setup/timeout failures remain NONQUALIFYING. No whole-parser implementation or production Task edit now.
- Independent exact git-object/tuple/MD/JSON/guard-fact/AST/patch audit PASS:218 original witnesses,197rows,47states,118orderedgroups,17exactmutations; all original setup/event/order and two terminal supplement bodies preserved. Raw /tmp/MAC-uzxr-PM-amendment.DFEjTI/audit.jsonl SHA256f73311bbd9611e33b4d56c17e46fa965b932d0ec814ced7ccab754cc47f3571c. Candidate runtime conformance and unsafe-variant rejection are not yet established.
- Original998a5a3 genuine semantic RED (3 intended outerFAIL/200childPASS) and storage4ef6baf verification plus historical225baseline remain preserved. A separate [test-edit-authorized] and tdd-red marked amendment commit, fresh full per-file/prefix hash inventory and finite genuine RED evidence must return to an independent PM checkpoint before any approve-red or GREEN dispatch. This disposition does not bypass the author's RED pause, authorize further test bytes, renew consumer claims, or credit missing future Task anchors/scaffold control failures.
- Durable record is append-only. Canonical eight paths/five AC, statusin_progress/hard-tdd/dev-MAC-uzxr claim, dependencies and all prior Notes/history preserved; no description rewrite, source/test/example/service/remote/preflight/toolchain change, product Paivot dependency, delivery/acceptance/rejection/RED-approval transition.

### proof
- [ ] AC #1: implemented current parser and complete binding execution pending.
- [ ] AC #2: complete candidate effects/context and narrow Task repair pending.
- [ ] AC #3: preserved initial genuine RED; final candidate unsafe-variant rejection pending.
- [ ] AC #4: full successful candidate row/witness/native inventory pending.
- [ ] AC #5: focused passing candidate proof and consumer handoff pending.

### 2026-09-06T05:39:42Z ramirosalas
## nd_contract
status: in_progress

### evidence
- TERMINAL EXPANDED RED FREEZE: exactauthorizedpatch2f722fa783fa4a27cbbd7ec92ad706e7bf86079801b0d80d3fe0a8c03f07e249 committedacf1bba2d37784fa175b5ee2cdd0f9b5ac0d5c3f with [test-edit-authorized] and tdd-red. Full detailed checkpoint in preceding Notes and /tmp/MAC-uzxr-final-red.nOXYuf/REVIEW.md SHA2565cb98377240aa8be22f0b9d88b74a7a6bd959fbfd8f1efb2907d08b02f68ad5c.
- Cleanretainedstory, exact7hashes/495prefix326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba/218tuples/2terminalbodies/197rows47states118groups preserved; full99file sourceinventorydd1a4498a35c6d7f521d68e4e0726202fe08ede5985272f142887aa550cddcd4. Task unchanged.2120add330del=2450changed/net1790; no scope/costrewrite.
- Actualfiveboundedcommands15.526s: contract0PASS28FAIL0SKIP; focused10supplementPASS5transitionrootFAIL; meta3/3/17outerFAIL. ALL current parser/control failures NONQUALIFYING scaffold, not semanticRED or negative success.23actualchildren10supplementPASS25rootFAIL0SKIP,0transitionrows/0mutants; allcwdsremoved.52exportfiles/6manifests47hasheseach; childinventoryfb4eb1daece9f31ac602812455d63f76fcd38005f8611f8d3d6f8f29b94aed40. Latermutants/futureTaskanchors not reached, not claimed.
- Normal verify-tdd3commits0violations/no waiver; scopedstatic7files0issues; formattingclean. Original998a5a3 genuineunsafeacceptedRED200childPASS, storage4ef6baf validation, historical225baseline and separately confirmedTaskdefect remain distinct preserved evidence; no fabricatedfixedreference or parserproof.
- STOP for independent combined-evidence finalREDdecision. No approve-red/delivery/GREEN/acceptance/rejection/claimrenewal, sourceimplementation, furtherfrozenbytes, remote/toolchain/service/productPaivot changes or runningbackground work. Assignee/status/label retaineddev-MAC-uzxr/in_progress/hard-tdd.

### proof
- [ ] AC #1: frozen parser/closedbinding contracts; real implementation and complete execution pending.
- [ ] AC #2: exact effects/context freeze and separateTaskdefect; real candidate repair/sensitivity pending.
- [ ] AC #3: preserved initial genuine RED; all final candidate unsafe-variant rejections pending.
- [ ] AC #4: current transitionrows unreached at scaffold; actual successful bidirectional candidate inventory pending.
- [ ] AC #5: complete passing candidate and consumerhandoff pending independentRED/GREEN checkpoints.

### 2026-09-06T06:09:09Z ramirosalas
## nd_contract
status: new

### evidence
- TERMINAL INDEPENDENT RED APPROVAL — MAC-uzxr at frozenacf1bba2d37784fa175b5ee2cdd0f9b5ac0d5c3f. Supported pvg story approve-red MAC-uzxr exited0 from own detached candidate, no waiver or forced transition. Readback is open with hard-tdd,red-approved; recorded assignee remains dev-MAC-uzxr as the supported command left it. No PM re-claim/claim renewal, delivery, GREEN implementation, acceptance or close.
- Full combined-evidence decision/report /tmp/MAC-uzxr-PM-combined.au6aPZ/REVIEW.md SHA256072af4cda455158288e635968040fa0930e1f40a0fd0fead10314497341d670c is appended in Notes and authoritative for rationale/limits. Approval applies the canonical explicit allowance for scaffold-supported RED when unchanged old suites genuinely demonstrate the missing sensitivity. Fresh historical4ef6baf replay gives3 intended unsafe-accepted outerFAIL/200childPASS/0SKIP,6exactcontrols-mutants46inputs2/2/1changedfiles; rawa8f978b85b1031dac050b1f5cd49a0c7b45e816c2b67c25deb72d63622f2e968. Separate unchangedTask1normalPASS1fallbackFAIL and strict224PASS1FAIL plus historical225baseline exactidentities re-audited. Those evidence classes are distinct.
- Current independent5groups16.396s: parser28FAIL; focused10supplementPASS5transitionrootFAIL; meta3/3/17outerFAIL. All current parser/control outcomes are NONQUALIFYING ErrScaffold.23actualchildren10supplementPASS25rootFAIL0SKIP, ZERO transitionrows/registrations/observations/mutants. FutureTaskanchors unreached.52exportfiles6manifests47inputhasheseach23cwdsremoved. Added default candidate replay2.153s verifies proof and3cwds cleaned. No current parser closure, negative sensitivity or successful inventory is credited.
- Independent audit /tmp/MAC-uzxr-PM-combined.au6aPZ/audit.jsonl SHA256abaa51f3f769410907eb17e283f17ce7effd916b088d8523a3919d72c9d175a8 verifies author and PM raw receipts/nativeidentities,99committedfiles,exact7authorizedhashes,495lineprefix326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba and unchangedTask. Prior full218witness/197row/47state/118orderedgroup/17ASTmutant review applies to identical bytes. Normal TDD audit3commits0violations, scoped static7files0issues, clean detachedsource; no waiver. Supported approval output SHA256d2b843614b44df9dec5bef0c06ad35820b35e9a0c7b5d37d4dc5bc314b1c5095.
- Root must dispatch FRESH GREEN separate from REDauthor context. Six Go test files atacf1bba remain immutable; implement only owned fsm.go behind reviewed minimumAPI and exactTask.fireRolledBack fallback realrecordTaskClosed plus orderedactions. Preserve tenoracles/modules/config/sourceclosure/otherproduction and sixfrozenassertionfiles. Any assertion/API/scope issue returns for exact review; no blankettestrepair or productPaivot coupling.
- GREEN must actually pass all frozen parser contracts/nested cases, fiveparsedtransition suites and10supplements, fullbidirectional successful row/witness/nativeinventory, and all20realvariants(original3+later17) with valid matchedfullcontrols/exactdiagnostics. Full43sequentialchild families, actualhash/input/nativeoutcomeinventory,zeroSKIP,defaultcleanup/explicitexport and measuredcomplete runtime/cost remain mandatory. No currentconsumerclaim, acceptance or authenticatedexecution renewal. All originalfivefullACpending. Canonicaldescription/eightpaths/dependencies/history preserved; no sharedsource/test/example/install/remote/preflight/service changes or children.

### proof
- [ ] AC #1: actual current parser and complete input/guard binding pending GREEN.
- [ ] AC #2: real candidate full effects/context and narrow Task correction pending GREEN.
- [ ] AC #3: all20 candidate unsafe-variant rejections with valid controls pending GREEN.
- [ ] AC #4: successful complete bidirectional candidate inventory pending GREEN.
- [ ] AC #5: full focused passing candidate and reviewed consumer handoff pending GREEN/PM.

### 2026-09-06T06:29:31Z ramirosalas
## nd_contract
status: delivered

### evidence
- Terminal GREEN delivery MAC-uzxr: supported pvg story deliver MAC-uzxr exited0 without waiver. Candidate dc3a8a365c21d5370c0825de992e2461ea0785a9 on story/MAC-uzxr. Worktree clean. Independent PM accepts or rejects; this developer did not accept/close/merge.
- Full PROOF, LEARNINGS, exact eight-path source scope, command argv and outcomes are in the preceding appended report and /tmp/MAC-uzxr-green.Td49Am/REVIEW.md SHA256918947599ceb237b56b1a87492f4127eee45f8c625e070ad558ef623d7762727. Primary native-audit.json SHA2561d6902dc46bf95a7a128a9c33da1cc802c79c77b2530afcbd9ee546882dc4db8; source-closure.json SHA256fba784f86918e105208e76d95a42f2b39acac8be114dbef95925a3628f877455. All exact producing Go commands/argv/cwd/sourceSHA/durations/rawSHA/full native identities are retained in named receipt files; all used GOWORK=off GOPROXY=off GOTOOLCHAIN=local and -count=1, Go120s native/unit and300s meta with unchanged125s/270s contexts and5s child WaitDelay.
- Actual required run:68 parser nested leavesPASS,228 focused native leavesPASS (218 transition witnesses+10 supplements),23 meta outer leavesPASS (3 original mutations+3 candidate packages+17 remaining mutations),0 unexpectedFAIL/0SKIP. Exact current inventory197rows/47states/118orderedgroups,218registrations and218successful observations: Deal58/75,Task31/35,User20/20,Session60/60,CommandExecution28/28 rows/witnesses. Coverage100% current rows and original witnesses; no statement coverage or full-repository test claim.
- All20 actual unsafe variants rejected with matched complete passing controls and exact intended diagnostics.43 exported sequential children:28exit0/15 intendedexit1;1398 native leaf/rootPASS+15 intendedFAIL (6semantic leaves,9structural pre-Fire roots),0SKIP. Five inventory mutants keep native assertionsPASS but the actual-output verifier rejects their malformed registration/execution inventory. Both Task future anchors matched once and real missing/name-only entry mutations failed at exact actions/context assertions.
- Custody:99 committed examplefiles verified;47 current inputpaths in43 complete manifests; all20 mutation hunks reconstructed from original bytes with once-only anchors and exact resulting hashes.149 exported raw/receipt/manifest/mutation artifacts retained;43 receipt cwd references resolve to23 removed fixture directories. All three default meta families independentlyPASS with43 further actual children;3 default proof directories and23 distinct child directories removed. Test-cache reuse excluded by -count=1. Export five-group25.381077583s; default lifecycle families24.461417250s on existing build cache; no cold-build measurement claimed.
- Six frozen SHA256 unchanged (paths relative examples/go-crm/impl): internal/cli/command_test.go38945b9307d97dadb872ac5a261ff86d712c218282774404b432842e34577d60; internal/domain/deal_test.go2bfd0893253b9f3c4b6cf1badbf3402a952551f5b4f5124a3d0a9afb7bf432cd; internal/domain/task_test.go35b8a05a710792ff21dd827432d658deba1ee190b8739196b084cdc61f7b2b88; internal/domain/user_test.god4fab40b760e5f1a94a5cab88bea6625495c04dd1690dc197818bfd1d056b53d; internal/session/machine_test.go97e612a68e22b32010ec7c696cb1ee248115f5a41f3b80400a40160a8782a11f; internal/testoracle/fsm_test.go4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25. Original495lineprefix326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba. Only owned parser implementation and Task.fireRolledBack fallback changed; no test/API/path broadening.
- Normal pvg story verify-tdd --base70652b948bf090008b1965c85daf36ea374daea4 checked4commits0violations/no waiver; eight explicitly scoped pvg verify paths --include-tests --format text PASS8files0issues; gofmt and git diff --check clean. Initial skill-documented --format=text was rejected by installed pvg, corrected using read-only help to supported --format text. No mutating subcommand help, unresolved product failure, remote/install/preflight/service/background operation.
- Measured cost: fsm.go607add/12delete final643lines; Task2add/1delete; GREEN622changedLOC. Historical2953+GREEN622=3575cumulative; aggregate3212add/331delete=3543changedLOC across8paths. Explicit closed grammar/type/metadata/duplicate-key checks and error precedence exceed the conditional parser estimate; no additional file/framework or proof trimming.
- Historical genuine RED3unsafe-accepted outerFAIL/200childPASS and separate Task defect are preserved. Expanded scaffold zero-row/zero-mutant failures remain NONQUALIFYING historical evidence, not new RED credit. Independent PM review and consumer migration remain pending; no automatic attestation/acceptance/authenticated-execution renewal.

### proof
- [x] AC #1: complete current parsing and original actual input/guard binding; missing/duplicate/malformed/unmapped/unused identity sensitivity with real controls.
- [x] AC #2: exact one-Fire ordered effects plus real Task entry/context repair and missing/name-only regression sensitivity.
- [x] AC #3: all20 actual unsafe variants rejected with complete passing controls and pinned intended diagnostics.
- [x] AC #4:197 current rows mapped bidirectionally to218 successful actual witnesses;10 separate supplements;0SKIP.
- [x] AC #5: all required frozen focused/parser/meta proof passed on committed candidate; unchanged source closure and reviewable consumer handoff; PM acceptance pending.


### 2026-09-06T06:51:32Z ramirosalas
REJECTED: R1 AC1/closed contract: public Parse at exact dc3a8a accepts ten unknown composite JSON keys across root/state/transition/invoke because fsm.go objectFields uses substring membership. Matched committed Session60-row control and four ordinary unknown-key rejections prove the gap. FIX: regression-first separate new test ownership via root/SrPM, exact-key implementation repair only in owned fsm.go, retain all six frozen tests; no test edit/grammar waiver or new RED approval. R2: actual verify-delivery3OK/6FAIL requires recognized canonical Implementation Evidence, CI/Test Results, Commands run, Summary, producing SHA and AC verification plus authoritative EOF contract; correct append-only on re-delivery and require9/9. Full independent report /tmp/MAC-uzxr-PM-green.7BaxI6/REVIEW.md SHA256dec4dd53d5f57dac813c9290fced9c4b366484fd8fc027da6ed8d695a3d2b74a. Frozen68/228/3/3/17 and default3/3/17 allPASS, no tampering; current raw proof preserved. No acceptance or source edits.

### 2026-09-06T06:51:33Z ramirosalas
## nd_contract
status: rejected

### evidence
- TERMINAL INDEPENDENT GREEN REJECTION MAC-uzxr at dc3a8a365c21d5370c0825de992e2461ea0785a9. Supported pvg story reject exited0; actual readback open/rejected, not accepted/closed/delivered. Full detailed EXPECTED/DELIVERED/GAP/FIX and all evidence in appended PM report /tmp/MAC-uzxr-PM-green.7BaxI6/REVIEW.md SHA256dec4dd53d5f57dac813c9290fced9c4b366484fd8fc027da6ed8d695a3d2b74a.
- R1 confirmed public Parse implementation defect in fsm.go objectFields: substring allowlist admits ten unknown composite field names across root/state/transition/invoke (including id initial,_comment _role,on after,target guard,src input,onDone onError). Exact committed Session control60rows,nil and four ordinary unknown-field oracle-parse controls; ten unsafe composite inputs wrongly60rows,nil. Separate external module imports unmodified exact Git archive; raw probe.stdout SHA2566158b871c33f6ced2ce1e9dc2a442247dae0b81a66c9e4c73032a73cfa80b2e2. Diagnostic exit0 is not contract success. Require genuine regression-first same-story rework via root/SrPM new-path ownership and independent checkpoint, then exact-key membership fix only; no change to declared grammar/API/Task or any of six frozen tests. No test-edit authorization or new RED approval here.
- R2 observed supported verify-delivery exit1:3OK/6FAIL for recognized Implementation Evidence, CI/Test Results, Commands run, Summary, commit SHA and AC verification shape. Substantive report/results exist; this is shape/placement failure, not missing executions. Raw SHA256f71c17f110481fd8250dd75e25e278ed5e3a03a0aeb2560a270250a7f61904be. Re-delivery must use canonical supported append-only evidence, actual final delivered contract at EOF and9OK/0FAIL; never --description/--body-file or waiver.
- Independent exact-archive frozen proof allPASS:68 parser,218 transitions+10supplements,3original/3candidate/17remaining meta; default3/3/17PASS0SKIP. Actual five-group25.799s/default24.093s, finite unchanged deadlines, Go1.27.1 darwin/arm64 GOWORK/GOPROXYoff GOTOOLCHAINlocal -count=1. Both PM and author raw audit:197rows47states118groups,218registrations/218successfulobservations; all20 exact variants43realchildren28exit0/15intendedexit1,1398nativePASS15intendedFAIL0SKIP,43x47input manifests149exports. Exact full child/native outcomes and actual inventory corruption checked. All23distinct exported fixture dirs removed; default adds43children with3proofdirs+23fixture dirs removed. Raw audit.json SHA256f5e3902319197e3394b6ac19f3b6ad2a6be71286c32bdc2e6994db5f8a203926. Ordinary artifacts, no authenticated claim.
- All99 committed example files match own archive; six frozen hashes and original495prefix326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba unchanged. GREEN only parser607add12delete and exactTask2add1delete=622changed;3575cumulative3543aggregate across8paths, justified conditional forecast excess not a cap failure. TDD4commits0violations, scopedstatic8files0issues, diffcheck clean. Real Task fallback context/call/actions and both regression mutants verified. No other established source blocker in full bounded review.
- Historical initial genuine RED3unsafe-accepted outerFAIL/200childPASS, storage proof and separate Task defect remain intact. Expanded scaffold ErrScaffold outcomes remain NONQUALIFYING; no retrospective RED credit. No PM source/test/example edit, claim renewal, install/assets/remote/preflight/service/Dagger action, developer-worktree access, merge/accept/close or child agents. Root owns new regression routing; main497419ab4512fcff765cd5feb27aed4c67b5608d and epic70652b948bf090008b1965c85daf36ea374daea4 unchanged.

### proof
- [ ] AC #1: NOT MET; exact structural-field closure fails for composite unknown keys. Regression-first repair and independent review required.
- [x] AC #2: Current exact one-Fire effects, Task real entry/context correction and missing/name-only variants independently verified; no whole-story acceptance.
- [x] AC #3: Current specified20 real unsafe variants and complete matched controls independently verified, correctly classified semantic/structural/inventory.
- [x] AC #4: Current197rows/218successfulwitnesses+10supplements bidirectionally reconciled to exact native identities with0SKIP.
- [ ] AC #5: Story not accepted; R1 implementation repair plus R2 canonical delivery shape and fresh reviewed candidate handoff remain pending.


### 2026-09-06T06:57:21Z ramirosalas
## AUTHORITATIVE SAME-STORY REJECTION SCOPE AMENDMENT — MAC-uzxr

This supported append-only live amendment supersedes ONLY prior eight-path ownership and conditional cost forecasts for the confirmed dc3a8a365c21d5370c0825de992e2461ea0785a9 rejection repair. All five original AC, the approved closed dialect/API, six frozen test files and original495-line prefix, previous exact amendment authority, original RED/scaffold evidence classes, metadata and dependency chain remain unchanged. The new ninth path below is current canonical ownership even though historical Description/Notes retain eight-path/no-new-file forecasts. No duplicate bug/epic, new production seam/API, Task change, existing-test amendment, phase/claim transition, GREEN approval or automatic consumer claim is introduced.

## Confirmed R1 and exact boundary

Independent PM REJECTED exactdc3a8a against70652b948bf090008b1965c85daf36ea374daea4. Full /tmp/MAC-uzxr-PM-green.7BaxI6/REVIEW.md SHA256dec4dd53d5f57dac813c9290fced9c4b366484fd8fc027da6ed8d695a3d2b74a and closed CONTRACT.md SHA256b3f444c5398b558916dec80a4dacf8c651420b6aa36d876326d606312baf8cdd were read in full. objectFields in already-owned examples/go-crm/impl/internal/testoracle/fsm.go uses strings.Contains over a space-delimited field list: a composite key spanning adjacent allowed names is incorrectly admitted. Public Parse actually returned60rows,nil for ten unknown keys across root/state/transition/invoke; matched Session control passed and ordinary notAllowed at each scope returned oracle-parse. Diagnostic process exit0 means the probe ran, not contract success. Existing passing tests do not close this missing negative case.

PRODUCES:
- examples/go-crm/impl/internal/testoracle/fsm_field_membership_test.go -> NEW TestFSMParseExactFieldMembership(t *testing.T), real public parser regression with isolated in-memory copies of actual committed Session inputs and deterministic structural targets. This is the only additional path, absent atdc3a8a; do not edit any of the six frozen files or factor their helpers.
- Already-owned examples/go-crm/impl/internal/testoracle/fsm.go -> later GREEN exact structural field-name membership correction only, retaining the existing public API, allowed fields, error category and all dialect/reconciliation semantics. No implementation during RED or by SrPM. Selection of concrete implementation text belongs to the separately routed GREEN author after independent RED approval.

CONSUMES:
- Existing Go CRM test-support parser.
  spec: func Parse(oracle, machine []byte) (*Suite, error); Suite exposes Name string, Rows []Row and States []State. This package is private example test support, not a new Machinery production API. Call Parse, not objectFields directly and not a source-pattern assertion or copied parser. Existing fsm_test.go TestFSMParseCommittedMachines reads actual inputs from ../../../design/machines and calls Parse/Load; use that source-verified real-input convention without editing its helpers.
- Existing committed Session input pair.
  source: examples/go-crm/design/machines/Session.oracle.md and Session.machine.json. Deterministic verified JSON targets are root; states.Anonymous; states.Anonymous.on.login transition; states.Authenticating.invoke. Decode a fresh copy for each single-key mutation; assert expected target existence and JSON object shape, avoiding map-iteration-dependent selection and accidental replacement of a valid key. Marshal into private memory only; do not modify committed oracle/evidence files.

## Fresh RED matrix and independent hold

Use exactly the public Parse contract on the matched committed Session pair. At rejecteddc3a8a, the raw unchanged control must succeed with Name session and all60 current rows. Preserve row/state identities and ordered actions in valid controls, not merely a nil error or string membership. The60 count is the exact selected committed fixture expectation, not a global197/218 ceiling or future schema restriction.

Ten separate unknown-key cases, each with an arbitrary object value such as {unexpectedSemanticData:true}, must fail with a nil result and specific oracle-parse unknown-field diagnostic naming that exact key:
- root: id initial; _comment _role; context states.
- state: on after; type entry; invoke _refusal.
- transition: target guard; guard actions.
- invoke: src input; onDone onError.

At all four same structural targets, separate ordinary notAllowed controls must still reject with oracle-parse: unknown field notAllowed. Each case must execute independently; an earlier failing composite assertion cannot prevent later scopes from running. Baseline RED must show actual nil-error/accepted malformed input as the intended failed assertion, with matched unchanged and ordinary-unknown controls passing. Compile/import/setup/timeout failures, arbitrary errors, skipped cases or a probe program's own exit0 are not RED proof. No fixture branch may accept the unsafe baseline to manufacture a passing test.

Valid declared metadata/context controls are equally mandatory: root _comment/_role strings, _delays/_counters string-to-string maps, state _refusal string-to-string map and invoke.input expression-string map remain accepted in their declared positions. Root context remains a data-only JSON object with arbitrary data values, including nested objects/arrays/null and data keys that resemble structural/composite keys; metadata map keys likewise remain data where the existing closed contract permits them. Prove these additions do not change parsed rows/states/actions and are never evaluated. This distinguishes exact structural membership from a global key-name ban or underscore ban. Unknown structural composite _comment _role remains invalid despite resembling two metadata names. Retain existing duplicate-key/type/unsupported-shape/identity semantics and all frozen dialect tests without amendment. No allowed field, metadata position, schema/API, runtime behavior, new grammar exception or name-normalization rule may change.

Root must route a fresh RED author scoped solely to the new file at exactdc3a8a. Suggested native command from examples/go-crm/impl: GOWORK=off GOPROXY=off GOTOOLCHAIN=local go test -count=1 -timeout=120s ./internal/testoracle -run '^TestFSMParseExactFieldMembership$' -json. Record source/test hashes, exact cwd/argv/environment, every discovered leaf terminal, actual failures/controls, durations and raw logs. Use existing local Go toolchain and ordinary package execution; no dependency/module changes, service, Docker, native harness modification, production seam, skip-if-missing, test-result fabrication or Paivot runtime dependency. Independent PM must replay/review exact new RED test bytes and genuine baseline behavior BEFORE separate GREEN makes the already-owned fsm.go correction. This scope amendment is not approve-red, old-test edit authority or GREEN application permission. Freeze the new regression hash after that independent checkpoint; all original six frozen files remain byte-identical throughout.

## Preserved freeze and full final proof

Frozen hashes, paths relative examples/go-crm/impl:
- internal/cli/command_test.go:38945b9307d97dadb872ac5a261ff86d712c218282774404b432842e34577d60
- internal/domain/deal_test.go:2bfd0893253b9f3c4b6cf1badbf3402a952551f5b4f5124a3d0a9afb7bf432cd
- internal/domain/task_test.go:35b8a05a710792ff21dd827432d658deba1ee190b8739196b084cdc61f7b2b88
- internal/domain/user_test.go:d4fab40b760e5f1a94a5cab88bea6625495c04dd1690dc197818bfd1d056b53d
- internal/session/machine_test.go:97e612a68e22b32010ec7c696cb1ee248115f5a41f3b80400a40160a8782a11f
- internal/testoracle/fsm_test.go:4578841816e77712ed249a349e1f63c13c7eac233f15eccacb2d0ee916527b25
Original495-line prefix SHA256326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba remains exact. Preserve all218 original tuple/setup/event witnesses and original terminal supplements, approved Task real fallback/entry/context behavior and exact unsafe variant anchors. No Task/runtime/native-harness/oracle/module/BUILD/attestation/acceptance change is authorized. hgz1 consumes the accepted same-story corrected parser/source/tests/full evidence as already sequenced; no downstream edit or dependency change is necessary, and lnu6 holds stay unchanged.

After GREEN, re-run the new regression and all original required parser/focused/native-sensitivity/executed-inventory families with matched controls, exact current native identity/row/registration/observation audit, all20 original unsafe variants, default cleanup/explicit export, unchanged deadlines and same-candidate source/frozen-hash/TDD/static proof. Previously reported68 parser/228 focused/3+3+17 meta leaves,43 children and197rows/218witnesses are historical measurements to reconcile, not fresh repair proof or a future hard ceiling. Preserve genuine original semantic RED and nonqualifying scaffold failures separately; no retrospective relabeling or authenticated execution claim.

## Honest revised cost

Exactdc3a8a base aggregate is3212 additions+331 deletions=3543 changedLOC across eight paths. Prior historical2953 plus GREEN622=3575 cumulative changedLOC. Independent PM explicitly found the modest closed-parser forecast excess justified, not a cost rejection: full closed grammar/type/metadata/duplicate/error precedence explains implementation619changed plus exactTask3changed. These measurements replace the obsolete conditional3300–3500 cumulative/3200–3400 aggregate forecast; neither was a cap.

New bounded estimate:120–220 added regression LOC for14 independent negative cases plus real matched metadata/context/committed controls, deterministic object targeting and actual Parse assertions;5–20 changedLOC for exact membership in already-owned fsm.go. Nine total possible paths, conditional approximately3670–3790 aggregate changedLOC and3700–3820 cumulative changedLOC. Report exact measured additions/deletions/cumulative rework and runtime separately. This is a forecast, not a cap or permission to trim full proof; explain material overrun and escalate any new path/API/seam before editing. The direct parser matrix should be small beside unchanged native43-child cost, but no unexecuted timing is claimed and no timeout expansion is authorized.

## R2 delivery record repair — later re-delivery only

The independent supported verify-delivery result was3OK/6FAIL despite substantive executions, producing SHA and AC evidence actually existing. Retain that failure and all real logs; it is evidence-format/placement failure, not absent test execution. Later repaired-candidate delivery must append recognized standard ## Implementation Evidence and CI/Test Results sections, a Commands run: list with actual commands, Summary: line with actual results, exact producing commit SHA and AC checklist/table. Include the fresh regression and unchanged full native proof with explicit limits. Use supported append-only nd operations and normal delivery transition; never --description/--body-file, destructive rewrite or waiver. Put the authoritative nd_contract status: delivered at actual EOF after all evidence, and require actual pvg story verify-delivery MAC-uzxr9OK/0FAIL before PM acceptance. Do not mark delivered or run acceptance/claim choreography in this scope task, and do not recycle rejected-candidate results as fresh repaired execution.

## nd_contract
status: rejected

### evidence
- SrPM source-confirmed same-story R1/R2 scope amendment; full independent PM report/closed contract/current canonical and latest rejection evidence read. Exact source interfaces and Session target objects verified from committeddc3a8a; graph main index ready but new testoracle symbol absent, so committed-source fallback used. Existing five AC/history preserved; only append-only ninth-path/cost supersession and independent RED/re-delivery holds are added.
- No source/test/example implementation, frozen-test permission, schema/API/native-harness/Task/dependency/claim/status/label change, acceptance, remote/preflight/install/assets/service operation. Machinery remains standalone; Paivot/nd are private coordination only. Scoped structural checks and exact append readback will be reported separately, never as AC execution or epic completion proof.

### proof
- [ ] AC #1: R1 remains pending genuine new RED, independent review and exact structural membership repair on a fresh candidate.
- [x] AC #2: Prior independent one-Fire actions/Task entry-context and missing/name-only variant evidence retained; no fresh execution or whole-story acceptance claimed.
- [x] AC #3: Prior specified20 real unsafe variants and matched controls retained; repaired candidate must replay them.
- [x] AC #4: Prior exact197rows/218successfulwitnesses+10supplements/native bidirectional inventory retained; fresh candidate proof remains required.
- [ ] AC #5: R1 repair and R2 recognized delivery9/9 plus authoritative delivered EOF contract and independent reviewed handoff remain pending.

### 2026-09-06T07:11:02Z ramirosalas
## nd_contract
status: delivered

### evidence
- Supplemental RED candidate d3d257bd199801d88bdcce8ac8f89fdb8f6bd64c: actual14PASS10intendedFAIL0SKIP; original68contractsPASS; one added test, no source correction. Awaiting independent RED PM checkpoint; historical red-approved label does not approve this file.

### proof
- [x] Supplemental AC #1 regression: all10 specified composite-field failures and14 passing controls executed against unchanged rejected source.
- [x] Supplemental custody: six frozen hashes/original prefix/99existingexamplefiles unchanged; committed tdd-red before replay.
- [ ] AC #1 full contract repair requires separate GREEN after independent RED approval.
- [ ] AC #2/#3/#4 complete final native/context/mutation/inventory evidence must be replayed on repaired candidate.
- [ ] AC #5 whole-story acceptance and consumer handoff remain pending; this delivery is supplemental RED only.


### 2026-09-06T07:20:28Z ramirosalas
## nd_contract
status: new

### evidence
- TERMINAL INDEPENDENT SUPPLEMENTAL RED APPROVAL MAC-uzxr: exactd3d257bd199801d88bdcce8ac8f89fdb8f6bd64c, new170-line examples/go-crm/impl/internal/testoracle/fsm_field_membership_test.go frozenSHA256201a470a38915709aab457daa4d94011cbaea873428b05a21c2d6e8b6754583b. Full review /tmp/MAC-uzxr-PM-supplemental.DsuF62/REVIEW.md SHA2567e7328ccdce810e62a65e108402b010ab0012b6064da2882febb520f50c0c545 appended intact. This is the explicit new supplemental approval, not reliance on the old label; no whole-story acceptance/close or new test-edit permission.
- Independently read all170lines, current nine-path amendment and unchanged closed contract/fiveAC. Public Parse on real committed Session inputs; exact deterministic root/Anonymous/Anonymous.on.login/Authenticating.invoke objects. Ten independent unknown-composite cases demand nil Suite and exact oracle-parse naming actualkey; four ordinary unknown controls and raw/remarshaled plus8metadata/context controls preserve full Name/Rows/States/orderedactions. Structural-looking data keys and unresolved inert expressions exclude global-name bans/evaluation. No copied parser, fixture/source mutation, weakening or schema/API change.
- Own exact Git archive actual Go tests:24leaves14PASS10genuine unsafe-acceptedFAIL0SKIP,exit1,634ms; all68originalcontractsPASS0FAIL0SKIP,exit0,354ms. Ten actual failures60rows,nil reproduce R1; not compile/setup/timeout/stub proof. Go1.27.1 darwin/arm64,GOWORKoff,GOPROXYoff,GOTOOLCHAINlocal,-count=1,Go120s/wrapper130s,all synchronous. Rawred d702ffb99ec0c29c901e5dcd3ea43aca79adbcc9d18ed7c30de7e30437129333; contracts796f9c8b48fd18b4103433a8ad5b1216076952a155a887844c502aa9e56b855c. Full exact actual PM+author leaf/outcome/diagnostic and100source custody audit8652260363a8d4f73ea1177cec141dffe73f3859ae67dadf33bcffe081327719. All99priorfiles/sixfrozenhashes/original495prefix326a632d89f522c7b40efc1e4d6a95bf81b0508db0cfc5c12459f9feec2132ba unchanged. Parser/Task remain rejected-source bytes; no implementation correction yet.
- Normal TDD5commits0violations; scopedstatic1file0issues;diffcheckclean. Current supplemental delivery verify-delivery9OK0FAIL independently observed; old R2sixshapeFAIL history preserved and finalGREEN must pass9/9 again.170newLOC,3745cumulative3713aggregate/9paths within conditionalforecast; no proof trimming.
- Actual supported pvg story approve-red exited1 because historical red-approved label already existed AFTER it moved stateopen and removed delivered/rejected. Rawstderr/receipt and intermediate readback retained; no successful-command fiction, waiver or manual labels. Root-authorized supported pvg story release then exited0 and cleared retained assignee. This exact approved EOF comment records independent supplemental approval; verified ready stateopen/unassigned/hard-tdd,red-approved, no delivered/rejected/accepted. No recovery or claim renewal.
- Root must dispatch separateGREEN within canonical existing fsm.go exact-membership scope; this PM chooses no production text. Seven testfiles now frozen. Final fresh candidate must pass new24regression plus alloriginal68parser/218native+10supplement/3+3+17meta families, all20unsafevariants43children, dynamic row/registration/observation/native audit, defaultcleanup/explicitexport and source/freeze/quality/delivery checks. No repeated fullnative proof claimed for this test-only increment. Original genuineRED, separateTaskdefect and NONQUALIFYING scaffold evidence remain separate. No authenticatedexecution/attestation/consumeracceptance claim. No PMsource/test changes, remote/preflight/install/assets/service/Dagger actions, developer-worktree reads, children or merge.

### proof
- [x] Supplemental RED: exact new regression and all10 genuine malformed-key assertion failures with14matchedpassingcontrols independently validated and frozen.
- [x] Supplemental custody: one addedfile only,all99priorfiles/sixfreezes/495prefix unchanged; original68parsercontractsPASS.
- [ ] AC #1: actual exact-key implementation repair remains pending separately dispatchedGREEN.
- [ ] AC #2: complete repaired-candidate effects/context and native proof pending; prior evidence preserved.
- [ ] AC #3: all20originalrealunsafevariant/43child candidate proof remains finalGREEN obligation.
- [ ] AC #4: fresh complete dynamic native/row/witness inventory remains finalGREEN obligation.
- [ ] AC #5: fresh final candidate proof/recognizeddelivery/independentPMacceptance and consumerhandoff pending.

