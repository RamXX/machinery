# Standalone executable test assurance  -  architecture contract

Status: approved architecture contract, version 1. Approval establishes the required behavior, not implementation completion or native execution evidence.
Scope: executable hard-TDD assurance for Machinery and its consumers, plus native custody for the contributor integration lane. The implementation must prove every applicable obligation below.
The [native custody architecture contract](native-custody-contract.md) refines only this contract's native-custody portion; it neither replaces the remainder nor establishes implementation or execution evidence.

## Implementation status in 0.7.0

Read this section before acting on anything below it. Everything below states the required behavior of the contract, in the present tense, as approved. It is the target, not an inventory of what the 0.7.0 binary does. In particular, **section 7's CLI grammar is the target contract: the `machinery tdd ...` command family and the `check --store` / `--assurance strict` flags do not exist in the 0.7.0 binary.**

Shipped in 0.7.0, reachable today:

- **The closed version-1 document grammar.** `internal/tdd` parses and validates `design/assurance/plan.json` and per-milestone manifests under the rules of sections 4 and 5, including cross-design resolution, control-namespace validation, and the review projections and subject digests of section 4 (`LoadPlan`, `LoadManifest`, `Validate`, `ReviewProjectionPlan`, `ReviewProjectionManifest`, `ReviewSubjectDigest`, `DesignPayloadDigest`, `ValidateControlNamespace`).
- **The authoritative obligation inventory of section 3.** `gates.AssuranceInventory(design)` derives it over a held immutable design snapshot: root BUILD milestones, committed oracle rows (including the relational formal oracles), machine-owned guard-clause IDs, modelith invariant IDs, and the plan's declared runtime obligations, keyed by the `{design, kind, owner, id}` tuple.
- **The content-addressed assurance store and replay-input retention of section 5.** `InitStore`, `Capture`, `MaterializeBundle`, `ExportStore` and `ImportStore` implement store initialization, validated capture (symlink escapes, hardlink aliases and special files rejected; the control namespace, `.git` and `attestations.yaml` excluded), the typed bundle encoding, and the validated export/import archive. `Status` is the read-only report; it performs no replay and launches no process.
- **Explicit registration of reviewed revisions.** `assuranceflow.Register` validates lineage, archives the exact control bytes, stages the successor and advances the store head by compare-and-advance under a bounded deadline. It commits authored data only and returns registered-not-executed; it never claims execution, replay or `Gv` acceptance.
- **The four closed native adapters of section 6:** `go-testing/v1`, `node-test-typescript/v1`, `python-unittest/v1` and `elixir-exunit/v1`, each with binary-embedded, hash-identified assertion and transport helpers, direct argv construction, and normalized event streams.
- **Adapter execution in the contributor lane.** `go run ./scripts/integration-lane --lane required` verifies every adapter runtime against the exact pins in `testdata/integration-lanes/assurance-runtime-pins.json` and executes frozen per-language runtime-probe and native-conformance fixtures through the production adapter chain under process custody, with exact per-case accounting. A missing or mismatched runtime fails the lane; nothing is skipped.

Not yet shipped in 0.7.0:

- **Every `machinery tdd ...` command.** The binary registers no `tdd` command. `store init`, `store export`, `store import`, `scaffold`, `capture`, `register`, `red`, `green`, `status` and `verify` exist only as library entry points, and `scaffold`, `red` and `green` have no implementation at all.
- **`machinery check --store <path>` and `--assurance strict`.** `check` accepts exactly `--impl`, `--commit`, `--gate`, `--complete` and `--warnings-as-errors`. `--complete` requires `--impl` and enforces the phase artifacts, closed milestones and zero warnings; it enforces no assurance replay.
- **Any gate that reads `design/assurance/`.** No gate in the `check` suite reads `plan.json` or a milestone manifest. `gates.AssuranceInventory` is called only from `internal/assuranceflow`. A design that commits an assurance declaration, valid or schema-invalid, sees no diagnostic and no change in its finding count. The `gtd` activation described in section 3 (cheap `gtd` from BUILD, controls, a configured external store or a declaration; hooks that block on missing adoption) is not implemented.
- **Independent replay over a consumer design.** `Execute` and `Verify` are not implemented. The adapters run only over the contributor lane's frozen fixtures, never over a consumer's declared suites, so no execution record, negative-sensitivity demonstration, or replay-this-invocation claim is available to a consumer.
- **Adoption of a design's tests by the shipped gates.** `Gt-tests` remains static discovery of stable-id references in test files and executes nothing; its own label says so. It is not a fallback for this contract and establishes none of the dimensions in section 1.

The consequence for a consumer of the 0.7.0 binary is exact: writing `design/assurance/plan.json` changes nothing that `machinery check` reports. Bind tests to oracle rows with the `Gt-tests` stable-id discipline until the CLI surface above lands, and read every present-tense statement below as the contract that surface must satisfy.

## 1. Decisions and guarantee boundary

Native final verification supports Go, TypeScript, Python, and Elixir in the first release through four CLOSED adapters, not an arbitrary shell-command protocol. A supported language does not mean every framework in that language is supported.

The target is mistake-prone, cooperating code on a trusted developer or CI host. A process with unrestricted same-user host access can modify the verifier, runtime, tests, or evidence. Native mode must display `provenance: unauthenticated-host` and `isolation: native-cooperative`; it must never claim protection from that adversary, hermetic execution, historical authorship order, or universal software correctness.

The strongest delivered test claim is: **this verifier invocation executed the same exact frozen suite against retained baseline/control/challenge source states and the current implementation; every required identity executed with the declared outcome, negative cases detected the reviewed unsafe variants, current GREEN passed, and the owned execution scope was cleaned up.**

Separate dimensions, never one ambiguous green badge:

| Dimension | Values |
|---|---|
| design consistency | checked / failed / not-performed |
| source/test binding | current / stale / missing |
| test execution | replayed-this-invocation / recorded-only / not-performed / failed |
| negative sensitivity | demonstrated / missing / failed |
| judgment | bound / stale / missing; never automatically true |
| formal execution | independently supplied result / not-performed; TDD does not imply TLC or Alloy ran |
| runtime residuals | tested / judged-not-applicable / unverified |
| custody | cleaned / cleanup-failed / unsupported |
| provenance | unauthenticated-host (only initial native value) |

There are no product dependencies on Paivot, its commands, tracker state, labels, commit markers, branch names, installed agents, or installed skills. Ordinary Git objects are optional immutable source carriers, not workflow authority.

### Bounded assumptions

Initial supported execution platforms are Darwin arm64 and Linux amd64. Existing cross-builds remain compilation evidence only. Other platforms fail `UNSUPPORTED_PLATFORM` for executable assurance rather than falling back to receipt inspection. The implementation must not silently waive existing Linux runtime test requirements because local native execution is authorized.

Initially support deterministic native tests and declared real local service fixtures. No framework watch mode, automatic retries, snapshots updated during tests, background test discovery, remote execution, or arbitrary provisioner commands. Dependencies and runtimes must be provisioned explicitly before replay; replay itself does not install, fetch, update, or use the active installed Machinery binary implicitly.

For clarity, first-release local fixtures are those initialized/reset by the frozen test scaffolding itself, using captured dependencies and in-process or scope-owned child processes; there is no generic service-provisioner/plugin interface in v1. Examples include a real private filesystem, in-process SQLite, or a real loopback server started by the test. Preexisting external databases, queues, credentials and Docker-managed consumer services are unsupported TDD fixture inputs until a closed service profile is added. The separate contributor lane retains its existing explicitly provisioned Docker/Java service contract; that does not make every consumer service automatically supported.

## 2. Components and dependency direction

1. `internal/tdd`: closed schemas, obligation reconciliation, exact snapshot/closure identities, protocol state machine, event accounting, replay, and evidence storage. It MUST NOT import `internal/gates`.
2. `internal/tdd/adapters`: four versioned native adapters plus binary-embedded, hash-identified assertion/transport helpers. They build argv directly and normalize native events; no shell parser or project-supplied reporter receives authority.
3. `internal/processscope`: native custody supervisor, direct-child guardians, scoped launch capabilities, explicit owned-resource cleanup. Reused by contributor infrastructure; no test semantics here.
4. Existing `internal/processcontrol`: compatibility entry point and explicit attachment to an active scope. Existing unscoped behavior remains available, but cannot satisfy scoped assurance custody.
5. Existing `internal/gates`: derive authoritative obligations and activate `gtd`; compare cheap binding/status through `internal/tdd`. Existing design snapshots remain the consistency boundary.
6. `internal/assuranceflow`: the privileged, closed orchestration layer imports gates, tdd, runtimeclosure and processscope; owns same-invocation preconditions, original snapshots, every finalizer, external-store publication and the sealed final result. CLI tdd/complete commands call this layer. Gates never import assuranceflow, and tdd never imports gates or assuranceflow. Hooks use only cheap binding/status, never an implicit heavy run.

The contributor lane consumes the execution/accounting/custody primitives and owns its independent closed test registry. It must not depend on a consumer BUILD or TDD milestone manifest to execute Machinery's own registered safety tests. Parser and process-custody libraries may be shared; product obligations and contributor inventory remain different authorities.

## 3. Authoritative obligations and activation

Add `gates.AssuranceInventory(design string) (tdd.Inventory, error)`, called only over an acquired immutable design snapshot. The inventory includes every root BUILD milestone and its expanded DoD oracle IDs, every committed oracle row in scope, machine-owned guard-clause IDs, invariant IDs, and declared runtime obligations. Existing generators remain authoritative for their own stable IDs; do not reinterpret a raw string as an ownerless ID.

An obligation key is the tuple `{design, kind, owner, id}`. Kinds are `oracle-row`, `guard-clause`, `invariant`, `runtime`. `design` and `owner` are canonical relative paths or declared entity IDs appropriate to the kind. Cross-machine, cross-consumer, and decomposed-parent obligations never pool merely because a short ID/name matches.

Every current oracle/guard/invariant obligation must belong to at least one current milestone; unresolved or unassigned obligations fail strict/complete assurance. A no-machine parent still has obligations if it owns relational policy/integrity/isolation or runtime requirements. Declared non-buildable decomposition parents reconcile child manifests and local obligations rather than pretending to have zero tests.

`check --complete` and `check --assurance strict` require this inventory and enforce executable replay even if no assurance files exist. An explicit `--gate` list cannot disable that requirement. Deleting, renaming, emptying or opting out of assurance files cannot convert strict success to cheap success.

Ordinary check/hooks activate cheap gtd when BUILD, assurance controls, configured external store or an assurance declaration exists, regardless of staged omission. Scaffold adds an authoritative BUILD line `Test assurance: machinery.tdd/v1` outside the control namespace. Known adoption from that line, current plan or store ledger makes missing controls/store/head blocking. If ALL authorities are absent, do not assert never-adopted: report `UNVERIFIED: no adoption/history authority available`. Ordinary legacy inspection can return a clearly limited design-only result, never TDD success or verified history. Hooks configured to require assurance block missing adoption/history. Strict/complete requires assurance regardless of every marker's deletion. This is not a detector immune to deleting every authority on a same-user host.

## 4. Closed document contract (version 1)

Use UTF-8 JSON with duplicate-key rejection, integer-only bounded numeric fields, no unknown keys, no trailing data, no NaN, no path expansion, and no optional meaning inferred from malformed input. All paths are normalized `/`-relative paths within an explicitly supplied repository root; reject absolute paths, `..`, symlinks, junctions, device files, sockets and ambiguous case/Unicode aliases. Dependency/runtime symlinks must be resolved into a separate verified closure before capture, never traversed during replay.

The schemas are normative closed records below. `Digest` is `sha256:` plus 64 lowercase hex digits; hashing is over exact bytes. `ID` is 1–128 ASCII characters `[A-Za-z0-9][A-Za-z0-9._:/-]*`. `Path` is a validated nonempty relative path other than ".". `RootPath` is either exactly "." or a Path; "." always names the explicitly supplied repository root, never a process's ambient working directory. RootPath is allowed for plan/TestRef/MilestoneKey design roots, implementation_roots, frozen_roots, Suite.root and Suite.dependency_roots. Repository remains the literal ".". A bundle/subject entry may use "." only with kind directory; source-file, manifest-file, assertion-file and other file-bearing paths cannot. Every nested root is resolved against the explicit repository root, not against another root field. `Ref` is a content-addressed bundle ID. Lists with set semantics reject duplicates and are sorted canonically for derived identities. User-authored file bytes are never whitespace-normalized for binding.

### `design/assurance/plan.json`

Required fields:

```
schema: "machinery.tdd.plan/v1"
design: RootPath
milestones: [{id: ID, manifest: Path}]
project_id: UUID
runtime_obligations: [{id: ID, category: Category, owner: string,
  source_refs: [{path:Path, anchor:string, digest:Digest}],
  disposition: "test" | "not-applicable" | "unverified", reason: string,
  review: Review, tests: [TestRef]}]
```

`Category` is `concurrency`, `delivery-replay`, `migration`, `restore`, `load`, `security-boundary`, or `observability`. Each category requires at least one explicit disposition even for an illustrative example. `test` requires mapped tests; `not-applicable` requires a nonempty rationale and current bound review; `unverified` is always a strict/complete blocker. These dispositions express reviewed scope, not an automated theorem that a category is irrelevant. Existing machine-readable NFR/failure/residual declarations reconcile bidirectionally; prose-only sources have exact file/anchor bindings in `source_refs` and a mandatory review of discovery completeness. The tool cannot prove that it understood every requirement hidden in prose. Unnamed prose cannot discharge a requirement.

`Review` = `{reviewer: string, rationale: string, subject_digest: Digest}`. The acyclic projection defined below binds design/control content and retained variant references, excluding all review records at specified schema positions. This is freshness-bound judgment, not an authenticated signature. Artifact deletion cannot replace a required review with silence. The full final control-file bytes, including every review, are separately frozen by execution evidence.

### `design/assurance/milestones/<id>.json`

Required fields:

```
schema: "machinery.tdd.milestone/v1"
id: ID
revision: positive integer
predecessor: null | Digest
repository: "."
implementation_roots: [RootPath]
frozen_roots: [RootPath]
subject_entries: [{path:RootPath,kind:"file"|"directory"}]
suites: [Suite]
obligations: [{key: ObligationKey, positive: [TestRef], negative: [TestRef]}]
baseline: null | Ref
variants: [Variant]
red_expectations: [Expectation]
checks: [Check]
red_controls: [{test:TestRef,assertion:ID,safe_variant:ID}]
limits: Limits
review: Review
```

`Suite` = `{id: ID, adapter: AdapterID, runtime: RuntimeRef, root: RootPath, files: [Path], tests: [Test], environment: [{name:string,value:string}], dependency_roots:[RootPath]}`.

Milestone IDs are canonical `M` plus an unsigned decimal BUILD milestone number, without leading zeroes except M0; filenames cannot inherit the broader ID grammar. Scaffold may create a draft with null baseline/variant source and incomplete arrays. Capture only needs its valid root/subject inventory. Load/status distinguish this draft explicitly; finalized validation requires every reference, review and required array populated in its selected milestone/dependency closure. RED validates that registered closure; complete validates every required milestone, with no future-draft exemption. Registration's explicit transition below is the only place an authored successor may differ from the current head. No draft receives an executable pass. Registration freezes the exact plan and selected manifest bytes but does not claim that future unregistered drafts were qualified; unrelated draft edits do not permit reusing an execution result whose captured full control inventory changed. Subsequent execution always rechecks that inventory.

`Test` = `{id: ID, native: NativeID, source: Path, role: "positive"|"negative"|"control"|"regression", assertions:[Assertion]}`. `TestRef` = `{design:RootPath,milestone:ID,suite:ID,test:ID}`. Every leaf has at least one bound assertion; aggregate parents are inventory nodes, not substitute leaf coverage. All references, including plan runtime obligations, variants and baseline expectations, use the full four-part TestRef. Design is canonical repository-relative design-root identity. Child references name the child explicitly. Milestones are unique within a design, suites within a milestone, and tests within a suite. M1/unit/T and M2/unit/T are DISTINCT tests. Resolve the entire declared design/decomposition graph first; dangling, ambiguous, cyclic-child and cross-project references fail. A parent may refer to a child's test only through a declared decomposition edge and that child's independently verified result within the same final invocation.

`Assertion` = `{id:ID, source:Path, line:positive integer, helper:"machinery-check/v1"}`. The adapter validates this is the exact frozen source call site of its supported assertion helper inside the named test, not a setup/import/cleanup statement or a comment. EVERY registered assertion must execute in baseline, safe-control, unsafe-challenge and GREEN runs. Aborting frameworks have no exception: an early failure preventing a later witness is INCOMPLETE_EVENTS. Split assertions into independent native tests, or arrange that every registered assertion is reached in every expected outcome. A parent failure never credits an unexecuted assertion. Different branches require separate registered tests/assertions; an early return cannot silently drop the second assertion.

`RuntimeRef` = `{profile:ID, version:string, platform:string, closure:Digest}`. `closure` covers executable/interpreter/compiler, framework, embedded transport/assertion helpers and relevant runtime libraries, not merely a version command. The verified local installation must match a provisioned closure; human-supplied hashes alone do not prove identity was checked.

`Variant` = `{id:ID, kind:"safe-control"|"unsafe-challenge", source:null|Ref, pair:""|ID, target_tests:[TestRef], expected:[Expectation], review:Review}`. Every nonempty pair has exactly one safe and one unsafe member with distinct subject digests and identical target sets. Each target passes safe and fails unsafe at the registered assertion. All nontarget expected outcomes are identical between pair members; additional differing behavior requires its own explicit targeted obligation. A no-op or test-edit challenge fails validation.

`Expectation` = `{test:TestRef, outcome:"pass"|"assertion-fail", assertions:[ID]}`. On `assertion-fail`, `assertions` is the exact nonempty expected failing assertion set. On `pass`, it is the exact asserted set that must evaluate true. Every leaf must have exactly one expectation in every baseline/variant run; no partial silently omitted suite. Parent failures are accepted only as the precisely accounted consequence of failed children, never as an independent expected failure.

`Check` = `{id:ID, kind:"design"|"architecture"|"typecheck"|"format"|"lint", profile:ID, inputs:[Path]}`. Profiles are closed, versioned tool adapters with pinned argv/config/closure. Unknown profiles block; a user-written shell command saying exit 0 is not a checked precondition. Initial coverage must include the actual required non-test checks of each migrated example; do not claim a universal formatter/linter integration from a prose list.

Initial profiles, and ONLY these, are `machinery-design/v1` (same-invocation nonimplementation gates, with the exact dependency-safe exclusion set defined in section 7), `machinery-architecture/v1` (same-invocation G4/Gt over captured scaffolding), `go-format/v1` (pinned gofmt; produce formatted bytes privately and compare exact input bytes), `go-vet/v1` (pinned `go vet` over exact selected packages, no ambient flags), `typescript-typecheck/v1` (pinned compiler with frozen config, noEmit and nonincremental fresh state), `python-compile/v1` (pinned CPython compile of every selected .py source without optimization or persisted bytecode), `elixir-format/v1` (pinned Mix format --check-formatted with frozen formatter config and exact files), and `elixir-compile/v1` (fresh pinned Mix compile --warnings-as-errors). These are precondition results, NOT assertion-test evidence. Python compilation is syntax checking, not a linter. Current required project checks outside this finite set block strict assurance as unsupported until an independently reviewed profile is added; they cannot be silently omitted or relabeled as compilation. Profile argv/options are fixed by the adapter, not supplied in the manifest.

`Limits` = `{wall_ms, cleanup_ms, stdout_bytes, stderr_bytes, event_bytes, event_count, jobs, bundle_bytes, entries, depth}`. Initial defaults: 600000, 10000, 4194304, 4194304, 16777216, 100000, 1, 536870912, 100000, 64 respectively. Positive finite values are required. Shipped absolute caps: 3600000 ms wall, 30000 ms cleanup, 64 MiB per output/event stream, 1000000 events, 4 concurrent jobs, 4 GiB per bundle, 1000000 file/directory entries, depth 128. Every JSON control/metadata input also has a fixed 16 MiB read/parse cap; directory walking stops at entry/depth limits, not only accumulated file bytes. Changes invalidate the revision. Exceeding a bound cancels production immediately and fails; truncation can never support success.

Native mode enforces these wall/output/job/storage limits. It must NOT advertise a portable whole-tree memory/CPU/PID quota. A request for unsupported kernel resource isolation fails explicitly; such isolation needs a separately reviewed backend.

### Aggregate execution budgets

`Limits.wall_ms` is ONE cumulative elapsed-wall deadline for the requested work of that milestone in this invocation: from first milestone preparation/capture validation through ALL selected suites, dependency/build work, retained baseline, every safe/unsafe variant, current GREEN, milestone preconditions, stream parsing and ordinary child cleanup. It is NOT renewed per adapter process, suite, source state, retry or phase. A GREEN request's required calibration is inside that same budget; a verify request's complete retained/current replay is likewise inside one budget. Scheduling/lock waits after that milestone starts count; elapsed time is never paused to hide another command's cost. The milestone deadline is the earlier of its own start+wall_ms and the owner's deadline. Shared initial loading and full final gates/publication count under the owner budget below.

Every assuranceflow.Run establishes a fixed maximum owner deadline of 14,400,000 ms (4 hours) from function entry, or the caller's earlier context deadline. It applies to the ENTIRE request, including all milestones/design dependencies, metadata discovery, store/snapshot locking, all process-producing checks, closing validation, publication and output. It exists even when the caller supplies context.Background. All-milestone execution therefore has both per-milestone limits and one finite whole-invocation ceiling; adding suites/variants/milestones cannot multiply the owner budget or renew it. No v1 CLI flag or manifest raises that ceiling. A larger portfolio must reduce scope for targeted work or receive a separately reviewed future budget-policy revision; targeted runs cannot claim omitted complete assurance.

Child commands/probes receive only the remaining milestone/owner budget, with any stricter native timeout applied as a minimum. `Command.DeadlineMS` means remaining allowed wall duration computed at dispatch, never an additional allowance after queueing. Queue/registration waits consume the same parent deadline. Ordinary phase-child cleanup has its declared cleanup_ms bound AND consumes remaining milestone/owner wall time.

Cancellation, any deadline, or the first terminal error atomically prohibits further verification work and enters FINAL cleanup with ONE shared grace deadline: now + max(cleanup_ms of selected milestones), capped at 30,000 ms. When no milestone was loaded, use 10,000 ms. All remaining child scopes/root/runtimes/views/store cleanup share that single monotonic deadline; do not grant a fresh grace period per resource, nested scope or repeated Close. Grace permits only owned kill/wait/close/file cleanup, not new targets/provisioning/replay, and cannot convert timeout into success. Cleanup that cannot complete before the shared deadline returns CUSTODY_ERROR/cleanup-failed with bounded owned-resource diagnostics and no seal. Normal successful finalization must fit within the owner deadline; grace is error-path cleanup only. As with the native trust boundary, kernel/host failure or an uninterruptible operating-system call cannot be promised a real-time userspace termination bound; report that residual honestly rather than certifying cleanup.

Status/register/store-transport operations similarly inherit caller cancellation and a fixed 600,000 ms owner deadline from entry, with one 10,000 ms final cleanup grace and no automatic retries. This bounds filesystem/locking work without inventing a new runner/profile subsystem. Existing byte/event/entry/depth caps remain independent ceilings. Tests must prove multiple suites/variants share the milestone budget, multiple milestones share the owner ceiling, zero/expired remaining budget prevents launch, cancellation during final Git/store waits returns no seal, and nested cleanup does not multiply the final grace. Deterministic virtual-clock tests may cover long-duration arithmetic; real bounded native timeout/cleanup tests still establish actual process custody.

### Native identities

- Go: `{package:string, test:string}`; `test` is the complete native test/subtest path.
- TypeScript: `{source:Path, path:[string], line:positive integer, column:positive integer}` after validated compiler source mapping; runtime correlation additionally uses entry-file plus native test/parent IDs.
- Python: `{module:string, class:string, method:string}`; exact `TestCase.id()` equality.
- Elixir: `{module:string, name:string, file:Path, line:positive integer}`; exact native module/function plus source location, not a prettified/truncated display label.

No arbitrary native-identity keys are accepted. The selected adapter discriminates the closed union.

### Bundles, revisions and evidence

Store retained inputs in `<store>/objects/<digest>/` with a closed `bundle.json`: `{schema:"machinery.tdd.bundle/v1", entries:[{path:RootPath,kind:"file"|"directory",mode:integer,size:integer,digest:Digest|null,role:"frozen"|"subject"|"design"|"dependency"}], tree_digest:Digest}`. Files have exact byte content in the content-addressed blobs namespace, verified on every read. Directories have size 0 and digest null. Retain every directory, including roots (path "." denotes the captured repository root directory), empty fixture directories and portable permission bits. Each parent must have a directory entry; duplicates, file/directory collisions and missing parents fail. Git import is only an input-carrier convenience and must materialize the same validated inventory, never require fetching a movable worktree later.

**Acyclic source/control-plane split:** TDD source bundles exclude ONLY the exact `design/assurance/` control namespace, whose allowed contents are authored `plan.json` and `milestones/`; unknown contents fail. They remain independently captured exact-byte CONTROL inputs, not ignored inputs. Archive their exact finalized bytes at `<store>/controls/<byte-digest>.json`. Objects, raw streams, runs, ledgers, locks and publication staging live ONLY in the external store below. Runtime source/fixtures/config cannot live in or import this control namespace; replay does not materialize it as source.

The exact `design/attestations.yaml` is a separately captured current-invocation judgment control, not test/runtime source; it may be refreshed for current implementation without editing frozen tests, but its complete current bytes must pass Gv. No other evidence path receives this exception: BUILD and acceptance artifacts remain captured inputs, so changing a frozen one requires a new qualification revision, preserving original RED and failed history. Applications that use these excluded control/evidence paths as runtime inputs are unsupported.

These are TDD source/control classifications ONLY. Accepted Gv policy is UNCHANGED: its full implementation-root inventory still includes `design/assurance/`, directory topology/modes, ignored/untracked entries and all other included content; only its existing top-level .git and exact attestation-record exceptions apply. When --impl is the repository root, finish authored TDD controls BEFORE refreshing Gv. Verification writes no files inside that root. Do not add a Gv receipt/cache/assurance/output exclusion to make a result pass.

Capture source first, then fill baseline/variant references in controls, then fill reviews, then register finalized control bytes in the external authored-revision ledger, then execute and append records. Draft/capture writes do not advance that ledger. Thus a source tree digest never depends on a manifest that refers back to it. `predecessor` names the archived exact-byte control digest of the prior revision. A topological chain is required; cycles and missing predecessors fail. Derived object/run publication is outside the source payload digest; authoritative control changes are independently detected and invalidate current execution binding.

Digest encoding is explicit. Files hash exact bytes. Tree/inventory SHA256 starts with UTF-8 domain `machinery.tdd.tree/v1`, then entries sorted by UTF-8 path. Each entry encodes one kind byte (0 directory, 1 file), U64-big-endian path length, path bytes, U32 permission bits, U64 logical size, 32 raw digest bytes (zero for directories), U64 role length, and ASCII role. The tree digest excludes its own field. Control and current judgment-control inventories use the same typed topology encoding with roles control and judgment-control respectively. Modes include portable rwx bits; special modes or declared ACL/nonportable-metadata dependencies are unsupported, not silently normalized. Filesystem timestamps and original native identity witnesses are not portable content digests.

Review projection C uses UTF-8 lexicographically sorted keys, no whitespace, shortest decimal integers, unescaped UTF-8 except quote/backslash/control characters, and schema-canonical arrays. Replace ONLY top-level manifest.review and every variants[i].review with null; on a plan replace ONLY runtime_obligations[i].review. review.subject_digest = SHA256(domain `machinery.tdd.review/v1`, U64 length and bytes of C(projected control), then raw authoritative design-payload digest). All reviews on a control bind that entire projection and all referenced sources. Full finalized control bytes INCLUDING reviews are separately frozen by execution evidence. This is acyclic; no review digest replaces exact frozen input identity.

Each retained source state has a full tree digest and all frozen bytes. A variant may change only `subject_entries`; no test, helper, fixture, manifest, dependency, configuration, checker, budget or runner alteration is an implementation challenge. Additions/deletions of subject files are represented explicitly. All other source inputs are frozen by default: omission from a hand-written `frozen_roots` list cannot turn a helper/config file mutable. Roots are topology boundaries, not exclusion globs.

The source inventory covers every directory and regular file beneath declared roots, including ignored/untracked inputs and empty directories. Materialize ancestor directories first, then exact files/modes, then apply directory modes deepest-first and verify complete topology. Directory additions/removal/mode changes, file-to-directory replacement and root-generation substitution stale evidence even when all file bytes match. Hold original rooted identity witnesses through final release; never persist those witnesses as later deletion/kill authority. A mutable subject directory cannot contain or be an ancestor of frozen entries; listing a subject file does not authorize modifying its parent. Generated private runtime output roots are separate from source and cannot change captured fixture topology.

TDD payload exclusions are only the exact separately captured control/judgment paths above, VCS administrative .git, and the adapter's fixed documented generated-output directories. No user exclusion glob. Imported/configured inputs in an excluded directory fail preparation rather than being silently omitted. Git-dependent source/gates require captured read-only Git objects/refs/config with hooks disabled, not ambient .git. External runtime/dependency closure must be verified/copied or held and revalidated, not accepted from lockfile text alone. Cross-root imports need complete declared roots. Case/symlink/hardlink substitution and concurrent mutation fail. Native mode cannot prove absence of arbitrary absolute host reads; completeness of input declaration remains a disclosed review assumption, not hermeticity.

Evidence files are append-only revisions, not an overwritten last-success JSON:

```
schema: "machinery.tdd.run/v1"
run_id: ID
design: RootPath
milestone: ID
manifest_digest: Digest
judgment_control_digest: Digest
control_inventory_digest: Digest
inventory_digest: Digest
frozen_digest: Digest
subject_digest: Digest
runtime_digest: Digest
phase: "red"|"green"|"verify"
result: "pass"|"fail"|"error"
executions: [{source:Ref, suite:ID, stream:Digest,
  discovered:[NativeID], started:[NativeID], completed:[NativeID],
  assertions:[{test:TestRef,id:ID,outcome:"pass"|"assertion-fail"}],
  outcome:string, exit_code:integer|null}]
custody: {status:"cleaned"|"cleanup-failed", transcript:Digest}
provenance: "unauthenticated-host"
diagnostics: [{code:ID, subject:string, message:string}]
```

All outcomes/errors use the closed enums defined below; `outcome:string` above is exactly a member of that enum, not arbitrary text. Each run record belongs to ONE explicit design/milestone/manifest; all-milestone verification aggregates such records by that full key, never by suite name alone. Preserve raw bounded native streams and byte hashes alongside normalized events. Receipt counts are derived outputs and are always checked against events. A receipt plus its self-written hashes is not replay proof.

### External store, transport, revisions and adoption

Every capture/RED/GREEN/status/verify/strict CLI invocation requires explicit `--store <path>`; no implicit installation/global-cache/Git-common-dir/tracker path is consulted. Hooks consume the explicit local `.machinery.json` field `assurance_store` and policy `assurance_required: true`; schema-invalid/missing store under that policy blocks. Config changes remain visible to existing full-root Gv and captured runner configuration. There is no environment-variable fallback that silently changes stores.

A store is a real rooted directory, opened with retained root identity and mode 0700. It must be OUTSIDE and nonoverlapping with every declared implementation, design, frozen, dependency and Gv root, including when --impl names the entire repository; canonical aliases and parent/child relationships are checked, not lexical prefix strings. If no such location can satisfy the supplied topology, refuse it. Store validation does not create or alter an existing caller directory. `store init` requires a new path and explicit project UUID and creates only its owned root. `store.json` is closed: `{schema:"machinery.tdd.store/v1",store_id:UUID,project_id:UUID}`. Plan project_id must match. Store IDs, not bare historical filesystem paths/inodes, identify archives; each invocation establishes NEW rooted native custody before reads/writes. Missing store/blob/control, a project mismatch or a symlink/substituted store root is blocking, not an empty fresh history.

The store contains only store.json, objects/, blobs/, controls/, runs/, ledger/ and transaction staging governed by its closed schema. `ledger/head.json` is the closed record `{schema:"machinery.tdd.head/v1",project_id:UUID,generation:nonnegative integer,previous:Digest|null,plans:[{design:RootPath,plan_digest:Digest}],milestones:[{design:RootPath,milestone:ID,revision:positive integer,manifest_digest:Digest,predecessor:Digest|null}]}`. Derived heads use the canonical JSON encoding C defined above, including sorted arrays, so an expected head plus identical selected controls has exactly one successor byte digest. Archive exact head bytes at ledger/heads/<byte-digest>.json before atomically advancing head.json. Plans sort by design; milestones sort by (design,milestone), with no duplicates. Only the INITIAL empty head has generation 0, null previous and empty arrays. Every committed successor has generation prior+1 and previous equal to the archived exact-byte prior head digest. Plan and manifest bytes referenced by a committed head must already exist in controls/. First milestone revision is 1 with null predecessor; every successor increments exactly once and names the prior archived manifest byte digest for the SAME design/milestone/project. Missing predecessors, same-revision byte replacement, restarting at 1, cross-milestone predecessors and rollback below a known head fail. A head records REGISTERED AUTHORED REVISIONS, not approval or execution success. No execution command advances it.

### Explicit authored-revision registration

The CLI owner is `machinery tdd register <design> --impl <path> --store <path> --expected-head <digest> [--milestone <id> ...] [--json]`, backed ONLY by `assuranceflow.Register` below. Repeated milestone flags form a unique sorted target set within that design; absent flags target all milestones declared by that design's plan. Child designs register under their own design command first when referenced. An empty target set is an error. No implicit register is performed by status, RED, GREEN, verify, hooks or complete. Registration is deliberately a distinct authored-data transaction, not test execution.

1. `store init` atomically creates store.json plus the archived generation-0 empty head and head.json, fsyncs and closes all created handles, and prints its project/store IDs and exact head digest. It accepts no source/manifest and cannot register a milestone. Failure cannot produce an initialized-success result; an existing partial destination is not silently adopted on retry. Explicit inspection/recovery or a new destination is required. The exact digest returned by init is the first register's expected-head; there is no wildcard/force/empty-string compare.
2. Scaffold and capture are DRAFT operations. Authors may edit an unregistered draft, capture baseline/variants, fill references and reviews, and correct validation failures without consuming a revision number. Capture still validates store/project/custody but does not require that a draft match an already registered manifest. Status explicitly distinguishes unregistered-draft, registered-not-executed, recorded-only and current replay; a well-formed inspection is not execution authorization.
3. Register reads the archived expected head (HISTORY_UNAVAILABLE if absent), acquires immutable source/control snapshots, resolves the authoritative design/decomposition graph, and validates the selected FINALIZED manifests, current plan, all selected references/reviews/retained bundles, root topology, project identity and lineage. It verifies source/reference/frozen calibration SHAPE, not native execution or current Gv judgment. For each target, require revision 1/null if absent from the EXPECTED head, otherwise that head's prior revision+1/prior exact manifest digest. Build the deterministic desired successor against this expected head even on an idempotent retry; do not reinterpret the already advanced current head as the requested predecessor. The current authored target is allowed to differ from the registered head ONLY in this registration transition. Referenced tests in other milestones must belong to other finalized targets in this transaction or an exactly current registered manifest in the store; a future draft cannot discharge a registered requirement.
4. Untargeted future milestones may remain explicitly draft/unregistered; their obligations remain pending and block complete, but do not block RED for a fully registered selected milestone. They still require valid declared identity/root structure; their documented draft null/incomplete fields do not become valid test evidence. Changing the exact bytes of a previously registered plan requires advancing ALL already-registered milestones of that design in the SAME registration transaction, with fresh reviews as required, because the plan is part of their frozen qualification input. Removing a previously registered milestone or plan is not an implicit retirement: first-release registration rejects it. This conservative rule avoids reusing one manifest revision under a changed plan; future retirement needs a separate reviewed protocol.
5. Release any store read reservation before acquiring its writer, retaining only immutable archived objects and validated store root identity. Acquire the external-store writer and compare the EXACT expected head digest, not just generation or target revision; the exact idempotent-success exception below is checked here before rejecting a mismatch. Archive and stage all selected finalized controls, the current plan and the entire derived next head. Unchanged untargeted entries remain in the next head; any untargeted REGISTERED control mismatch within the resolved design/dependency graph is blocking. Unregistered draft files are not head entries and may differ as explicitly allowed above. Entries belonging to unrelated designs remain unchanged and need not be loaded from an unrelated checkout. Finalize the original snapshot releases, reacquire a final input snapshot and compare exact source/control/root-generation identities, then complete that release before any head publication. Any stale input/release/store validation error aborts without advancing head. Registration does not require Gv to approve the newly authored controls: a separate explicit judgment refresh occurs after authoring/registration and before strict verification. It grants no implementation/test approval.
6. Commit by durable atomic compare-and-advance of head.json to the staged next head while holding the writer. All referenced immutable controls and prior/next heads must be durable first; successful commit appends exactly one generation. Writer/fsync/cleanup/output errors produce no successful Registration result. A process crash can leave either the old head or the complete new head, never a head naming unavailable staged bytes; readers validate that invariant and fail closed on corruption. Recovery never rolls back a committed head to erase a failed attempt.
7. After registration, every RED/GREEN/verify/strict invocation requires its current finalized manifest AND plan to match registered head entries exactly. Missing/unregistered/stale/rollback state blocks before any suite starts. Successful registration returns only `registered-not-executed` and the new head digest. Failed RED/GREEN remains failed at its already-consumed revision; changed frozen controls require the next revision, preserving original controls, raw failures and earlier RED. Merely registering N+1 does not make its execution state equal to N's.

Concurrent registrations with one expected head cannot both commit: the loser receives HEAD_CONFLICT without losing prior history. A retry is idempotent ONLY if the current head is exactly the deterministic desired successor of that exact expected head, with all archived controls identical; return `already-registered-not-executed`, not a new generation. If another transaction has since advanced the head, even in another milestone, the old compare fails; the caller must inspect/rebase its authored registration against the new head. No automatic merge or last-writer-wins.

Failed precommit validation consumes no revision number. Preserve any bounded raw failure observations once capture/execution has produced them; registration failure does not delete objects or older runs. If publication committed but a later close/output failed, the command returns error and the head may already be advanced: status exposes the complete committed head, and an exact idempotent retry can confirm registration. The author must not reuse that number for different bytes. Registration status and inspection may be recovered from validated persisted data because they are claims about authored bytes, never about whether tests executed. Local rollback limits remain as stated below.

Store export writes a new, bounded archive with complete referenced object/control/head history and checksums, no live sockets/PIDs/transaction locks. Import requires a NEW destination, project UUID and caller-supplied expected exported head digest, validates every entry and topology/limit before atomic publication, and preserves the archived chain and original failed records. It never marks imported receipts as replayed. Moving/copying repository and store preserves logical relative paths and content references; re-opening establishes fresh custody and recomputes current Gv/source bindings. No old native identity is reused as cleanup authority. A previously configured absolute hook store path must be explicitly repaired after relocation; no auto-search selects a possibly unrelated store.

On a new host the expected head is an explicit caller trust input; an unauthenticated local archive cannot prove it is globally newest. Known-local rollback protection and complete-chain verification are the claim, not global monotonicity without an independent authority. All execution still replays current input states. Missing local history is reported HISTORY_UNAVAILABLE; neither import nor deleting all adoption authorities constitutes evidence of never-adopted legacy. Existing BUILD/config/plan/store declarations impose their requirements independently; strict/complete requires a valid store/contract even when all markers are removed.

## 5. Protocol and actual assertion causality

The authored state machine is `empty store -> draft -> registered-not-executed`, followed independently by execution `red-observed -> green-observed -> replay-verified-this-invocation`; drift yields stale, not weaker success. Persisted observations are recorded-only, never sealed final execution authority. GREEN replays retained calibration before current GREEN unless calibration already completed in the same still-valid invocation; a stored RED receipt cannot authorize it by itself. A revision cannot move backward by overwriting evidence. New test/fixture/config changes create a new revision with `predecessor`, preserving prior failures and original frozen RED. There is NO formatting exception and no `tokens-equal` authorization.

1. Capture current authoritative design inventory and derive tests from these obligations, not the implementation's current behavior. Author/capture/review the selected draft, then explicitly register its finalized revision against the expected external head. Run declared non-test preconditions over the same immutable source snapshots only after exact registered binding is established.
2. Author full tests plus passing harness control, compilable baseline scaffolding and reviewed implementation-only safe/unsafe variants. Their purpose is calibration, not substitutes for integration tests against real GREEN software.
3. `red` executes the entire baseline and each pair. At least one new behavior assertion must fail baseline; a successful test exit, compilation failure or generic exception is not RED. Every required negative test must PASS on its named safe control and FAIL at its exact assertion on its paired unsafe challenge. The passing control is the same negative test on safer code, not an unrelated health check. `red_controls` additionally covers EVERY baseline assertion-fail expectation regardless of positive/negative/regression role: each full (TestRef, failing assertion ID) maps exactly once to a safe-control variant where the SAME native test and registered assertion execute and pass under the identical frozen closure. Later GREEN, another assertion, a health control or another test is insufficient calibration. A safe-control variant used only by red_controls has `pair:""`; paired variants have a nonempty pair ID. Every control variant must be referenced. All required negative challenges still require their complete paired safe/unsafe control. A normal safe-default stub may already pass negative tests. For an existing bug, the original unsafe baseline may be the challenge and a minimal safe-control variant can establish calibration; both remain retained, and this proves discrimination, not historical chronology.
4. Negative obligation minimum: each guard-falsifying clause; each refusal/denial/error/compensation/retry/terminal-safety oracle; each invariant; each applicable runtime failure/residual. Every milestone also has at least one challenge of its public boundary. Positive obligations cover successful outcomes. Classification is derived from typed oracle/invariant data; ambiguous rows require explicit reviewed classification, never automatic omission. An optional extra positive test does not offset a missing negative obligation.
5. Freeze EXACT tests, helpers, fixtures, configs, dependency locks/content, selectors, assertions, preconditions, budgets, native inventory and variants. The author cannot amend frozen assertions to make GREEN pass.
6. `green` runs all required tests and preconditions on current implementation with frozen non-subject closure. All leaves and registered assertions must pass. It records current state, not final independent replay.
7. `verify` ignores prior pass assertions as authority: it rereads retained bytes, reexecutes baseline and all safe/unsafe pairs, then reexecutes current GREEN and non-test preconditions. It verifies current inventory and unchanged checkout inputs immediately before publication. The whole operation is one root custody scope. Only successfully FINALIZED invocation can return `replayed-this-invocation`; Execute/Verify results remain provisional.

Each helper's `check(test, assertion_id, boolean)` evaluates a genuine native framework assertion, emits an entered/evaluated witness associated with the exact call site, and lets the native framework report failure. It never catches a product exception and converts it into assertion success/failure. Each adapter reconciles helper witness + native test lifecycle + native failure class/site + real process status; a matching string/exit code alone is insufficient. Boolean arguments are strictly boolean, not language truthiness. Diagnostic values are bounded and sanitized.

GO SPECIFIC: Go has no typed native assertion-failure event. Strict tests use the byte-pinned helper, whose false boolean emits a fixed source-attributed `testing.T.Errorf` and witness. Direct/custom `T.Error/Fatal/Fail/FailNow`, custom `TestMain`, or third-party assertion wrappers are unsupported in the strict suite until a closed adapter supports them. Validate typed call targets in frozen test/support code. Unmatched failure-site output, panic/race/timeout output, extra failure, or absent witness is ERROR. Ordinary logs cannot establish a failure witness. This restriction is deliberate; do not disguise Go JSON `fail` as assertion-specific evidence.

The helper protocol is accident-resistant, not a cryptographic channel against test code sharing its process. Paired passing/failing executions reject unconditional failure/no-op helpers and stale artifacts; semantic adequacy still needs independent review.

## 6. Native adapter profiles and supported surface

The release must contain all four adapters below plus real end-to-end baseline/control/challenge/GREEN tests for each. Initial EXACT compatibility entries are Go 1.27.1, Node 26.9.0 with TypeScript compiler 7.0.2, CPython 3.14.7, and Elixir/ExUnit/Mix 1.20.4 with Erlang/OTP 29.1.1 (ERTS 17.1). Availability of a runtime is not adapter verification. Platform-specific closure hashes are provisioned and checked by implementation/runtime stories. Adding another tested exact version is a compatibility-catalog change, never `>=` optimistic parsing.

### `go-testing/v1`

Use actual `go test -json -count=1` with exact packages, anchored full-name selection generated from declared identities, bounded native `-timeout`, fixed `-shuffle=off`, and declared build tags only. Disable ambient `GOFLAGS`, automatic toolchain downloads and network dependency fetching; resolve a complete local module/workspace/dependency closure first. Selected package/test sources must match the build-selected inventory; a build-tag-excluded file cannot supply evidence.

`go test -list` lists only roots, so it cannot attest dynamic subtests. Reconcile native `run`/`pause`/`cont`/`pass`/`fail` and package lifecycle with the complete frozen leaf inventory. Every declared child must run; parent-only pass is not coverage. Fixed-name deterministic table subtests are supported; benchmarks, unbounded fuzzing, nondeterministic/generated names and custom testing drivers are not strict test identities. Fuzz regression corpora must be exposed as finite named tests to qualify initially. Detect cached output even though the fixed invocation disables test-result caching.

Negative adapter oracles: all-pass baseline; compile/import failure; skipped/build-excluded test; parent passes with missing child; duplicate child name; cached result; panic after expected assertion; setup failure before assertion; forged aggregate output; extra `T.Errorf`; absent expected witness; source/config/toolchain drift.

### `node-test-typescript/v1`

Use the pinned TypeScript compiler in a separate bounded build step with frozen tsconfig, `noEmitOnError`, nonincremental clean output, source maps and embedded source content. This step must pass in RED and GREEN. Run only that invocation's compiled test outputs through Node's actual `node:test` runner and a binary-embedded reporter; no npm script indirection, project reporter, Jest/Vitest substitution, loaders, Babel, tsx, or stale dist directory. The initial TS profile supports TypeScript accepted by the pinned compiler targeting Node ESM, with the declared local dependency closure; browser/JSX test environments are unsupported profiles, not silent native equivalents.

Map source locations back only through verified compiler-produced maps whose source bytes are in the frozen bundle. Reconcile `test:enqueue`, `test:dequeue`, `test:complete`, hierarchy and final summaries. The live execution pair is dequeue/complete; buffered `test:start`/pass/fail reporting is cross-checked, not counted as an independent execution. Scope native IDs by entry file/process; native numeric IDs alone are not globally unique. Resolve every leaf to declared source/path/location. Require the helper-backed native `AssertionError` cause and source site for expected failures.

Reject skipped/todo/only/runOnly/expectFailure tests, watch/rerun-failures/changed/shard filters, cancelled children, hook failure, unresolved promises, import/compiler failure and `process.exit(0)` before a complete inventory. Fixed named nested tests are supported. Output JSON/TAP printed by application code has no reporter authority.

Node's native TypeScript stripping is NOT this profile: it omits type checking and ignores tsconfig features. Compiler errors are precondition errors, never RED. [Native event contract](https://nodejs.org/api/test.html), [TypeScript runtime limitations](https://nodejs.org/api/typescript.html), [compiler source maps](https://www.typescriptlang.org/tsconfig/sourceMap.html).

### `python-unittest/v1`

Use pinned CPython in isolated mode with an embedded bootstrap, a verified stdlib/dependency closure and explicit immutable sys.path roots. Disable user site/startup hooks, optimization (`-O`), bytecode reuse, automatic dependency fetching and project runner replacement. Invoke actual stdlib `unittest` discovery/loading; enumerate and reconcile full `TestCase.id()` identities before running a closed suite.

Use an embedded `TestResult` recording startTest/stopTest, addSuccess, addFailure, addError, addSkip, expectedFailure and unexpectedSuccess. Expected assertion failure requires native AssertionError from the registered helper in the test body; an assertion in setUp/tearDown does not qualify. Error, skip, expectedFailure and unexpectedSuccess all block. Require unmodified stdlib TestCase.run/result semantics; replacing failureException or lifecycle methods is unsupported. Standard TestCase and IsolatedAsyncioTestCase are supported; pytest, doctest, custom load_tests/result/runner classes and dynamic subTest parameter identity are unsupported initially. Represent parameter cases as independently named methods; `subTest` emits UNSUPPORTED_FEATURE rather than disappearing.

Negative oracles include import failure, setUp assertion, ordinary exception, optimized-away bare assert, load_tests omission, skip/xfail, zero tests, duplicate ID, fake summary, early sys.exit, unawaited/cancelled async work and changed helper/dependency. [Native result callbacks](https://docs.python.org/3/library/unittest.html).

### `elixir-exunit/v1`

Use actual Mix compilation and ExUnit execution under a binary-embedded bootstrap/formatter, never a project-defined mix alias as authority. Pin the BEAM runtime, Elixir/Mix/ExUnit libraries, mix.exs, mix.lock, config, test_helper and dependency bytes; use fresh private MIX_BUILD_PATH and offline dependencies. No implicit Hex/Rebar installation or network fetching. Compile and non-test checks pass before test execution; compiler errors are not RED.

Freeze and validate effective ExUnit options (seed, filters, max_cases, timeouts, formatter) after test_helper evaluation. Require complete native suite/module/test lifecycle and exact module/name/file/line identities. Expected failures are helper-backed `%ExUnit.AssertionError{}` with the bound test-body stack site; setup_all/setup/on_exit failures, exits, throws, arbitrary exceptions and VM timeout are errors. ExUnit async tests may remain async; max_cases and seed are explicit, and event interleaving is reconciled by identity rather than order. Assertions about messages evaluate actual bounded receive results through the Boolean helper; failure caused by the overall ExUnit test timeout remains an error, not assertion RED. The embedded formatter must not terminate itself on `suite_finished`: ExUnit stops every formatter once during its own teardown, and a formatter that races that teardown aborts a suite that already ran. A mix abort that follows a complete reconciled run is a post-execution runtime fault reported as UNEXPECTED_FAILURE with the witnessed outcomes preserved, never a BUILD_ERROR.

Reject excluded/skipped/invalid tests, max-failures truncation, stale/failed-only/partition selection, formatter removal, runtime reconfiguration, duplicate identities and incomplete suite termination. Plain ExUnit.Case and deterministic describe/test blocks are supported; property frameworks, doctest-generated cases, parameterized test identities and custom ExUnit runners require future closed profiles. Adapter service fixtures must use real dependencies; Ecto consumers must declare database isolation/reset rather than rely on a global test DB.

Negative oracles: compile failure, assertion in setup, skip/exclude, silent filter, post-configuration formatter replacement, ordinary raise/throw/exit, missing module/test finish, shuffled concurrent identity collision, fake CLI summary and stale compiled BEAM. [Formatter events](https://ex-unit.hexdocs.pm/ExUnit.Formatter.html), [native test identity](https://ex-unit.hexdocs.pm/ExUnit.Test.html), [Mix test options](https://mix.hexdocs.pm/Mix.Tasks.Test.html).

### Normalized event contract

Each adapter produces a bounded stream of closed records `{schema:"machinery.tdd.event/v1", sequence:integer, design:RootPath, milestone:ID, suite:ID, kind:EventKind, native:NativeID|null, assertion:ID|null, outcome:Outcome|null, source:Path|null, line:integer|null}`; every key is present, using null only where the kind permits. Design/milestone/suite are assigned by the owning invocation and must match the enclosing record; raw native output cannot reassign them. `EventKind` = suite-start, discovered, test-start, assertion, test-end, suite-end, diagnostic, error. `Outcome` = pass, assertion-fail, error, skipped, unsupported. Diagnostic payloads remain bounded raw side streams, not arbitrary extra JSON properties. Helper/native event order must form a legal per-test automaton. Only suite completion plus exact inventory and exit concordance can yield a successful execution record.

Machine diagnostics include MISSING_STORE, STORE_ROOT_MISMATCH, HISTORY_UNAVAILABLE, CONTROL_ROLLBACK, MISSING_CONTRACT, STALE_INPUT, INVALID_SCHEMA, UNSUPPORTED_ADAPTER, UNSUPPORTED_VERSION, UNSUPPORTED_PLATFORM, UNSUPPORTED_FEATURE, MISSING_TEST, DUPLICATE_TEST, INCOMPLETE_EVENTS, ASSERTION_MISMATCH, UNEXPECTED_FAILURE, BUILD_ERROR, TIMEOUT, OUTPUT_LIMIT, CUSTODY_ERROR and REPLAY_REQUIRED. Human text must explain the repair without disguising errors as expected RED.

## 7. CLI, gates and internal API wiring

Public commands (all use explicit paths, never the active installed binary as a dependency):

```
machinery tdd store init --store <path> --project <uuid>
machinery tdd store export --store <path> --out <new-archive>
machinery tdd store import --store <new-path> --archive <path> --project <uuid> --expected-head <digest>
machinery tdd scaffold <design> --impl <path> --store <path> --adapter <id> --milestone <id>
machinery tdd capture <design> --impl <path> --store <path> --milestone <id> --name <id>
machinery tdd register <design> --impl <path> --store <path> --expected-head <digest> [--milestone <id> ...] [--json]
machinery tdd red <design> --impl <path> --store <path> --milestone <id>
machinery tdd green <design> --impl <path> --store <path> --milestone <id>
machinery tdd status <design> --impl <path> --store <path> [--json]
machinery tdd verify <design> --impl <path> --store <path> [--milestone <id>] [--json]
machinery check <design> --impl <path> --store <path> --assurance strict
machinery check <design> --impl <path> --store <path> --complete
```

Scaffold writes new draft manifest/helper outputs only to nonexistent paths; no test semantics or approval fabricated. Capture writes a validated content-addressed bundle and prints its reference; capture is not RED evidence. Register separately commits finalized authored revisions and returns only registered-not-executed; it neither runs tests nor advances their execution state. RED/GREEN append execution records; only verify and the two strict check forms perform independent replay. There is no import-receipt, accept-exit-code, skip-negative or force-green option. A targeted verify reports milestone scope and cannot supply complete assurance for omitted milestones. Global final preflight remains the final integrated heavy operation.

Exit codes: 0 = requested contract satisfied (RED may contain only the exact expected native failures), 1 = evaluated mismatch/stale/missing required assurance, 2 = invocation/schema/unsupported/runtime/cleanup/protocol error. Interrupted runs use 2 with a stable diagnostic; preserve the causal signal in evidence rather than confusing exit 130 with expected RED. Ordinary status may return 0 for a well-formed but not-yet-GREEN draft, always with explicit state; strict never accepts that state.

Public internal signatures to copy into implementation story contracts:

```go
// internal/tdd; concrete records above define the named value types.
func LoadPlan(design string) (Plan, error)
func LoadManifest(path string) (Manifest, error)
func Validate(plan Plan, manifests []Manifest, inventory Inventory) error
func Capture(ctx context.Context, req CaptureRequest) (BundleRef, error)
func Status(ctx context.Context, req StatusRequest) (StatusReport, error)
func Execute(ctx context.Context, req ExecuteRequest) (Candidate, error)
func Verify(ctx context.Context, req VerifyRequest) (Candidate, error)

type Adapter interface {
    ID() string
    Prepare(context.Context, SuiteRequest) (PreparedSuite, error)
    Run(context.Context, PreparedSuite, EventSink) (Execution, error)
}
```

Normative request/value fields (field names and types, not merely conceptual inputs):

```go
type MilestoneKey struct { Design string; Milestone string }
type Inventory struct { Obligations []Obligation; Milestones []MilestoneKey; Digest string }
type InputView struct {
    SourceRoot string // immutable, topology-preserving source materialization
    DesignPath string // relative to SourceRoot; payload excludes reserved control plane
    ImplementationPaths []string // relative to SourceRoot
    ControlRoot string // separate immutable control materialization
    Revalidate func() error // original captured inputs; MUST be nonnil
    Release func() error // owner-controlled idempotent FINAL release; MUST be nonnil
}
type CaptureRequest struct {
    Inputs InputView; Manifest Manifest; Name string; Store string; Limits Limits
}
type StatusRequest struct {
    Inputs InputView; Plan Plan; Manifests []Manifest; Inventory Inventory; Store string
}
type ExecuteRequest struct {
    Inputs InputView; Plan Plan; Manifest Manifest; Inventory Inventory
    Phase string // closed enum red | green
    Store string; Runtimes []RuntimeHandle; Scope processscope.Scope
    Checks CheckExecutor
}
type VerifyRequest struct {
    Inputs InputView; Plan Plan; Manifests []Manifest; Inventory Inventory
    Milestone string // empty = all; otherwise one exact manifest ID
    Store string; Runtimes []RuntimeHandle; Scope processscope.Scope
    Checks CheckExecutor
}
type SuiteRequest struct {
    Inputs InputView; Suite Suite; Source BundleRef; Scratch string
    Runtime RuntimeHandle; Scope processscope.Scope; Limits Limits
}
type EventSink func(Event) error
```

`Manifest`, `Plan`, `Obligation`/`ObligationKey`, `Suite`, `Limits`, `RunRecord` and `Event` are the exact closed schema records above. `BundleRef` is the validated Ref plus its verified immutable materialization handle; it cannot be made valid by deserializing a path. `RuntimeHandle` holds the pinned closure's identity/root and Close/Validate operations. Close includes final process-free byte/topology/root-identity validation before releasing the retained handles, and propagates every validation/close error; any identity probe that launches a runtime must happen earlier under a live scope. `PreparedSuite` is opaque adapter-owned state produced ONLY by successful Prepare (including exact argv, executable/build outputs and native inventory); it is not a deserializable receipt. `Execution` contains the normalized event inventory, raw-stream references/digests, target exit status and independent custody result specified in the run schema. `StatusReport` contains the dimension values from section 1 plus diagnostics; `Candidate` contains provisional outcomes and pending precondition handles, never final success. Only `internal/assuranceflow.Verification` adds a sealed result after every finalization stage below. Constructors validate every path, reference, callback, runtime handle and scope; an InputView supplied with a missing validator/release callback or mutable/unverified source is rejected. Internal struct implementation details not exposed through these boundaries remain implementer choices.

Requests carry already-held input snapshots, authoritative inventory, manifests, verified runtime handles, closed CheckExecutor and required process scope. Receipt text cannot establish precondition or execution authority. Candidate reuse is private same-invocation work deduplication only. The final assuranceflow.Verification binds exact finalized source/control/judgment/runtime digests and the completed release boundary; it is non-deserializable, and cannot certify a newly acquired/changed tree or replace replay on another invocation.

`gates.CheckTDDAssurance(design, impl string, inventory tdd.Inventory, status tdd.StatusReport) *Gate` is cheap. StatusReport is produced by tdd.Status over the explicit validated store and held InputView; it contains StoreProjectID, StoreHeadDigest, SourceDigest, ControlDigest, JudgmentDigest, dimension values and diagnostics. Missing/mismatched binding is an error, never an empty success. Add `gates.RunOptions.TDDStatus *tdd.StatusReport` for this same-invocation result and `gates.RunOptions.TDDRequired bool` for the resolved requirement; CLI/hook code supplies them explicitly. A nil report under required assurance blocks. A well-formed Status may expose unregistered-draft/registered-not-executed for inspection; gtd emits their explicit unmet requirements rather than accepting registration as a pass. Required complete uses the full inventory; targeted RED validates only its explicit registered milestone/dependency closure and cannot discharge future milestones. Gates do not independently rediscover a store or deserialize a result file. While the flow owns the publication writer it derives StatusReport from that held store generation rather than recursively acquiring a second store lock. Hooks obtain/close a read-only store view and own its validation errors. This status is NEVER execution evidence: the gate always reports `replay not performed`. Strict orchestration finalizes the provisional tdd.Verify result before any success claim; do not make gates call back into itself through tdd.

Use accepted gates.AcquireSnapshot, Snapshot.RunSelected, CheckUnchanged AND final Snapshot.Release semantics. Do not bypass pending Gv finalization or print/publish final success before Release returns. Gate preconditions and tests use the same owned view. Extend exact topology tracking without changing Gv exclusions. Execution records publish only to the external store, not through a design-writer transaction. A command authoring scaffold/controls is a SEPARATE writer-acquire/expected-generation-check/publish/release transaction that invalidates prior assurance; it cannot claim combined success for the pre-edit tree.

Build-writer/template changes require this executable manifest flow, reviewed negative classification, exact freezing and final replay. `Gt` may remain a cheap discoverability check; it cannot claim executed coverage. `Gv` implementation judgments must bind actual test/source inventories. `Ga` keeps historical ancestor acceptance while complete assurance requires current replay; an ancestor approval is not current implementation approval.

### Dependency-safe precondition execution

CheckExecutor is a CLOSED production interface implemented only by the registry inside internal/assuranceflow; not a project plugin, JSON receipt or user callback. It bridges tdd requests to gates without reversing package dependencies:

```go
type CheckRequest struct {
    InvocationID string; StateID string; Inputs InputView
    Checks []Check; SourceDigest string; ControlDigest string
    Scope processscope.Scope; Runtimes []RuntimeHandle // live invocation-owned handles
}
type CheckExecutor interface {
    Run(context.Context, CheckRequest) (PendingChecks, error)
}
type CheckResult struct {
    InvocationID string; StateID string; Profile string
    SourceDigest string; ControlDigest string
    Status string // pass | fail | error; no pending pass is public
    Diagnostics []Diagnostic
}
```

PendingChecks exposes exactly `Provisional() ([]CheckResult, error)` and `Final() ([]CheckResult, error)`; Final errors until all owning views have completed release. Its concrete implementation is private to assuranceflow's fixed registry. PendingChecks is a non-deserializable owner-bound handle to the produced gate/tool results plus the actual snapshot generation and unexported invocation nonce. It exposes no constructor or pass-authorizing mutation. The flow owner resolves it ONLY after its owning snapshots complete Release, then checks every result matches invocation/state/profile/input digests and exactly the requested profile inventory. Imported results, another variant's pass, missing results and an unset executor fail. tdd consumes provisional results to decide whether execution may continue; any later release error invalidates the candidate.

For machinery-design/v1 the producer uses the held Snapshot's normal canonical selection, removing ONLY implementation G4/Gt, historical acceptance Ga, standing judgment Gv and recursive assurance Gtd; this is specifically deterministic design-precondition evidence, NOT a full complete-check verdict. machinery-architecture/v1 runs exactly G4/Gt against the SAME captured baseline/variant/current source state. It does not run Gv against deliberately unsafe variants or label their expected historical judgment staleness as a RED failure. Byte/topology identity must match the actual test materialization even if a gate requires its own private copy. The initial named compiler/format/lint profiles use the same verified runtime closure, input state and scope, with real process exits and bounded output; these exits establish tool-precondition results, not native assertion evidence.

The final complete/strict flow separately runs the full current design/implementation selection INCLUDING current Gv, Ga, Gtd status and required completion gates. It NEVER reuses the deliberately narrower RED precondition set as final acceptance. It passes the same live candidate to the orchestration layer, not recursively to check --complete or tdd.Verify. Existing formal/checker execution remains a separately named result unless the caller's full verification explicitly runs it.

### Explicit execution capability for final gates

A full final gate selection is NOT necessarily process-free. In accepted source, Ga's `runGitExact` creates context.Background and launches Git through processcontrol.Run; its commit/ancestor helpers do not inherit an outer scope today. The implementation MUST repair that call chain rather than assuming an ambient context/environment attachment survives it.

Add `gates.RunOptions.Execution *GateExecution` and `gates.RunOptions.ExecutionRequired bool`, with assuranceflow always setting ExecutionRequired true. This is distinct from TDDRequired: cheap ordinary hooks/status may require fresh assurance bindings without claiming scoped execution or final replay, and cannot construct Verification. A process-producing selection with ExecutionRequired true cannot call legacy unscoped helpers. Use the following gate-owned value boundary (gates imports processscope/runtimeclosure, not assuranceflow):

```go
type GateExecution struct {
    Context context.Context
    Scope processscope.Scope
    Git *runtimeclosure.Git // NEW validated pinned executable/dependency closure
}
```

Add the concrete NEW runtimeclosure.Git with private custody fields and these exact methods: `Executable() string`, `Digest() string`, `Validate(context.Context, processscope.Scope) error`, and `Close() error`. Its constructor is `OpenGit(context.Context, GitRequest) (*Git, error)`; GitRequest is `{RuntimeRoot string, ExpectedClosure string, Scope processscope.Scope}`. It verifies a provisioned Git 2.55.0 closure for the declared supported platform (executable plus runtime dependencies), using the same bounded exact-byte/topology/root-identity principles as existing Java closure handling; a missing declared dependency/closure or unsupported version fails closed. The provided expected digest is checked against actual opened bytes, not accepted as a receipt. Validate may perform registered identity probes while the supplied scope is OPEN; Close performs a FINAL pure byte/topology/root-identity revalidation, then releases the retained handles without spawning any process, propagating both validation and close errors. This is a new reviewed runtime-story responsibility, not a claim that runtimeclosure already implements Git. Initial support requires pinned platform-specific Git closures in the contributor/runtime inventory; runtime provisioning is an explicit implementation operation, not evidence supplied by this document.

assuranceflow creates GateExecution for EACH process-producing selection from the request's cancellation/deadline context, the still-open phase child Scope, and the live pinned Git runtime closure. Snapshot.RunSelected carries RunOptions.Execution to the Ga acceptance path. Add `checkAcceptanceWithExecution(..., execution *GateExecution)` and `runGitExactWithExecution(execution *GateExecution, dir string, args ...string)`; every acceptance resolution/head/commit/ancestry helper threads the SAME capability, including resolveReviewCommit/resolveReviewCommitExact, gitHeadAt/gitHeadAtExact, gitCommitOf, gitIsAncestor, checkCommitBinding and checkCommitAncestry. Ordinary inspection wrappers may preserve their old API and explicitly limited guarantee; ExecutionRequired/scoped selections cannot fall back to those unscoped wrappers. A nil/mismatched/cancelled/closed capability blocks before Cmd.Start.

The scoped Git runner derives timeout with context.WithTimeout(execution.Context, gitCommandTimeout), not context.Background; uses the validated handle's absolute executable; sets the existing gitcontrol.Environment sanitized environment and bounded stdout/stderr captures; then attaches the explicit scope with processcontrol.WithScope AND processcontrol.AttachScope AFTER environment sanitization. Both attachments must identify the same live scope, or fail. Preserve the existing exact cwd/commit/ancestry semantics and target-exit classification via processcontrol.ExitStatus. Pin/revalidate Git plus its required local dependency/runtime closure before and after final gates. Git is a Machinery gate runtime, not a fifth consumer test adapter and not test-execution evidence; capture its digest in the invocation runtime inventory. Do not consult the installed Machinery binary or permit project-configured shell Git wrappers.

This explicit attachment rule covers ALL process producers reachable in preconditions and final selection, not only native suites or formal helpers. Closed check profiles receive their Scope/Runtimes through CheckRequest; any future gate requiring a tool must declare its runtime and accept GateExecution/a correspondingly typed scoped handle before it can join this lane. Missing registration is UNSUPPORTED_EXECUTION, not an unscoped fallback. Read-only filesystem/hash operations need no process capability. Production scoped tests must audit the real process-producing call graph, rather than only asserting that RunOptions has a field.

### Finalization ownership, ordering and sealed success

The only public execution-success boundary is:

```go
// internal/assuranceflow imports gates+tdd+processscope, never the reverse.
type Request struct {
    // Run imposes section 4's aggregate deadline even for Background context.
    Mode string // red | green | verify | strict-check | complete-check
    Design string; Implementation string; Store string; Milestone string
}
func Run(ctx context.Context, req Request, output io.Writer) (Verification, error)
// Authored data only; not an execution/Verification constructor.
type RegisterRequest struct {
    Design string; Implementation string; Store string
    ExpectedHead string; Milestones []string // empty = all plan milestones
}
func Register(ctx context.Context, req RegisterRequest, output io.Writer) (Registration, error)
```

Registration exposes exactly ProjectID, StoreID, PreviousHead, HeadDigest, Generation, sorted MilestoneKeys and State (registered-not-executed or already-registered-not-executed). Its constructor is private; it carries NO replay/custody-success/test-pass accessor and cannot satisfy Verification. Register owns the entire snapshot/store transaction described in section 4, including expected-head comparison, release/reacquisition, fsync, cleanup and output. tdd owns pure lineage validation and deterministic next-head construction; assuranceflow owns commit authority. Store.init and import are separate store-construction operations, not registration substitutes. Register performs no subprocesses; any implementation that requires one must instead use the same scoped lifecycle as Run, not a background subprocess hidden in a finalizer.

Verification has an unexported constructor/nonce and immutable result accessors only. tdd.Candidate has no final success accessor/serialization. The flow owns all InputView.Release callbacks, runtime handles, execution materializations, pending gate handles, scope, store transaction and output. Each resource is registered exactly once before use; cleanup is idempotent, joined and attempted even after an earlier failure. A consumer cannot call release early and continue using the candidate.

Order is mandatory:

1. Acquire held original input snapshots and validated external store; require exact registered plan/manifest/head binding before suite execution. Capture source/control/judgment/topology identities. Open one root custody Scope and retain ALL invocation runtime handles, including the pinned Git handle. Create an execution child Scope for preconditions/native suites and explicitly attach every launch. Execution produces only Candidate and raw provisional observations; no success is printed.
2. Finish/drain native event parsing and streams. Close the execution CHILD Scope with bounded cleanup, checking its owned descendants/report. The ROOT Scope, runtime handles and any materializations still needed for checks remain OPEN and retained. Failure invalidates the candidate but does not skip later cleanup. Closing a child is not finalizing the root and grants no public success.
3. Check original input custody and call every owned original InputView.Release, including accepted gates.Snapshot.Release. This performs original attestation CheckUnchanged, closes retained captures/workspace, releases locks and FINALIZES pending Gv results. Only after those Release calls return may the owner resolve their PendingChecks and inspect final gate objects. The runtime/source materializations needed by subsequent runtime validation are separately owned, not dangling aliases into these released views. Any release/close/late-Gv failure is a hard error even if tests passed.
4. If still valid, release any store read reservation, retain validated store root identity and immutable captured head/object bytes, then acquire the external-store publication writer and compare its expected project/head digest/generation. Reacquire a fresh original design/implementation snapshot under the existing reader boundary. Compare exact source, frozen, control, judgment and FULL Gv inventories (including directory modes and root-generation identities) against the candidate. A change during release/reacquisition yields STALE_INPUT; no checked result is silently rebound. Hold this FINAL input view through all closing execution and cleanup below. Registration cannot occur inside this execution transaction.
5. Create a closing child Scope under the still-OPEN root. Run the final current gates appropriate to Mode on the final view, supplying GateExecution with this child, the inherited request context and live Git closure. Strict-check/complete-check run the FULL applicable closing selection, including Ga's REAL scoped Git queries, current Gv and Gtd status; red uses narrower declared preconditions and never claims final acceptance, while green labels its implementation scope. Do not describe full closing gates as process-free. Revalidate every runtime/executable closure now, routing any identity probes through this closing child before it closes. Drain ALL gate/tool/probe outputs and join their actual completion. Pending results remain private.
6. Close the closing CHILD Scope, then close the ROOT Scope. Root Close is the FINAL NO-LAUNCH BARRIER: atomically reject new registration/launches, clean and reap every registered remaining descendant/resource (including guardians of final-gate Git), and validate the terminal cleanup report. Both child and root errors invalidate the candidate. Then Close all runtime handles, requiring a final pure file/topology/root-identity revalidation after owned processes are gone, and Close remaining execution/build/dependency materializations using process-free finalizers. Only pure file/identity checks and direct native close/unlink operations may follow this barrier; Snapshot.Release, external-store publication/recovery, staging cleanup and output MUST NOT start commands. Any runtime validation/provisioning/cleanup requiring subprocesses must have completed under the closing scope BEFORE this step, or the dependency is unsupported. All resources still receive bounded cleanup on error, but no closed Scope is reopened or replaced to rescue success.
7. While the FINAL input view is still held, recheck its unchanged state AFTER process/runtime/materialization cleanup, then complete its Snapshot.Release and resolve pending Gv/results. Release must preserve every existing full-root/attestation capture/workspace/lock finalization error. Any cleanup-originated checkout modification is detected here. This is the final input-custody linearization point; only after release and all PendingChecks.Final calls succeed may publication proceed. No persistent observation can claim the mutable checkout stays unchanged after that released point; subsequent uses must rehash/replay.
8. Publish bounded raw observations/run records ONLY to the external store using its held transactional writer. This step NEVER advances or rewrites the authored-revision head; compare that exact head again before publishing references. Stored native aggregate outcomes are recorded-only and no persisted seal exists. Publication/fsync/transaction cleanup/writer Close and all store/report staging closes must succeed, using process-free filesystem operations. Failure returns error, preserves immutable failed/partial history for safe recovery, and issues NO Verification even if recorded-only bytes became durable.
9. Recheck cancellation. Render the complete, scope-labeled result only now; output write/flush failure returns error with NO returned seal. Only after successful output does Run construct/return Verification. The caller exits 0 only on nil error and completion within the fixed aggregate owner deadline defined in section 4. SIGINT/SIGTERM, late store errors or a partial JSON line cannot become a success capability.

Every error path converges on the same ownership order: stop active child launches and close them, close root, close runtimes/materializations, release any still-held input views, close store/staging, then return joined diagnostics without a seal. If failure occurs before the final reacquisition, do not acquire a new view just to manufacture a successful gate result. If a child Close cannot prove cleanup, root still attempts all owned cleanup, and the earlier error remains fatal. Cleanup uses its own bounded cancellation-independent cleanup budget, but that budget confers no authority to launch new verification commands. Native direct kill/wait/close operations required to retire the broker are cleanup, not new target launches.

No design writer is reacquired for execution recording: the external store has already been proven outside ALL original input/Gv roots, so publication cannot stale its own verified source. Authoring manifests, explicit register compare-and-advance, revision changes and Gv refreshes happen in separate transactions before verification and require new capture/review as appropriate. If a store path accidentally falls within any root, refuse it rather than weakening Gv's exclusions.

Status/hooks report persisted data as recorded-only; they NEVER reconstruct Verification, red approval or final execution authority from a store record. Post-publication errors may leave a durable observation, but cannot advance the execution assurance state. Pending transaction recovery may restore/read observations only; the next execution operation independently repeats its required calibration/replay.

Required negative finalization tests inject actual path/mode/root changes and real close/publication/output failures at EACH boundary above. Include the REAL complete/Ga path with pinned Git: demonstrate that its query is owned by the closing child while root/runtime remain live, cancellation cleans its registered guardian and ordinary descendants before return, and invoking its scoped runner after child/root Close refuses launch (no target-start event or marker). A controlled owned native-descendant fixture may exercise that cleanup seam, but a fake Git executable or mocked processcontrol result cannot replace successful actual Git commit/ancestor queries. Independently verify final-gate owned resources are gone before the complete CLI can emit success; preserve an unrelated process as a safety control. Exercise an actual late Git/runtime-probe failure after suites pass, plus closure validation attempting an after-barrier launch, and require no seal. Assert no returned seal/no complete success and final Gv error when applicable; preserve unrelated source/store contents and earlier failed history. Include success controls through the same complete CLI with --impl equal to the repository root, proving external publication leaves its full-root Gv subject unchanged.

## 8. Native custody contract and contributor integration

### Required guarantee and rejected shortcuts

On normal completion, assertion failure, timeout, output overflow, SIGINT/SIGTERM or provisioning error, the root supervisor must terminate/reap all registered owned commands and verify owned container cleanup before returning success. Real nested Java identity probes, TLC invocations and recursive full-path meta-tests are in scope. Merely killing an outer Go process group, exporting a ProvisionTLC pathname, following a PID file, scanning process names, or reporting closed pipes is insufficient.

Select a native cooperative custody broker with unreaped direct-child group guardians. This is a portable Linux/Darwin ownership design, not an assertion that portable native APIs contain arbitrary daemonizing code. All Machinery/internal tool launches in the assurance lane must route through the scope. Direct child processes normally inherit a guarded group. A dependency that creates its own session/group or detached service must use the scoped launch/service API or be rejected as an unsupported execution dependency; silently claiming total cleanup would be false. Active hostile escape and killing the supervisor itself remain out of scope, visibly reported.

### Ownership mechanism

- The root starts one private broker using the exact candidate executable (hidden internal subcommand), a private 0700 directory, authenticated local control channels and noninherited owner-liveness descriptor. Random scope/job capabilities prevent accidental cross-run attachment; they are not secrets from a hostile same-user process.
- Only the broker can create a command guardian. It directly starts each guardian as its child and group leader, and does NOT reap that guardian until the group has been terminated. The guardian runs the real command inside its group and reports the real result over a separate channel, then remains alive awaiting cleanup. If it crashes, its unreaped child identity still prevents its PID from being reused before the broker's final group signal. No external supplied PGID grants authority.
- The broker registers the direct-child identity before releasing the guardian to execute any target. Failed registration never starts the target. The group-kill-before-wait ordering is mandatory on every path; an eager goroutine calling Cmd.Wait would invalidate the PID-reuse argument.
- Nested Machinery processes get a scoped capability, not permission to create unowned process groups. Their own processcontrol launches are broker requests that create sibling owned guardians within the same root scope or a child logical scope. An intermediate parent's exit cannot abandon the registered descendants.
- Cancellation atomically changes OPEN to CLOSING, refuses new launches, cancels child scopes, terminates every owned group/container, then reaps guardians and verifies terminal resource records. Registration/cancellation races cannot launch after cleanup's inventory is closed. A broker failure/cleanup timeout produces CUSTODY_ERROR, never a pass.
- Signals affect only groups whose guardian is still an unreaped direct child of that broker. Retired IDs are never reused as cleanup authority. A stale capability, missing broker, forged registration, cross-scope request or already-reaped guardian fails closed without signalling a foreign numeric PID.
- Ordinary descendants that stay in guarded groups are cleaned with those groups. Explicit container runs additionally retain the exact daemon-created container ID and validated ownership record; stopping Docker's client is not container cleanup. Daemon disappearance prevents success and yields an actionable owned-resource report. Never touch unrelated containers or other user-owned resources.
- Owner-channel loss triggers bounded cleanup while the broker is alive. Simultaneous uncatchable broker death, kernel/host failure and actively escaping descendants cannot be guaranteed by native mode. Persist an incomplete private ledger for manual inspection, but never automatically kill by its historic PIDs after restart.

### Concrete interfaces and integration boundary

```go
// internal/processscope
type Scope interface {
    Run(context.Context, Command, Streams) (Result, error)
    Child(context.Context) (Scope, error)
    Attach(Command) (Command, error)
    Close(context.Context) (CleanupReport, error)
}
func Open(context.Context, Options) (Scope, error)
func Join(context.Context, Capability) (Scope, error)
func ServeInternal(args []string, io InternalIO) (handled bool, exitCode int)
```

`Command` is closed argv/cwd/environment/stdio/runtime-identity data, not a shell string. `Result` preserves target exit code/signal, start/completion, bounded output and cleanup errors separately. `Options` supplies the candidate helper executable closure, limits, scratch root and parent-liveness channel. `Capability` is an opaque live broker attachment, never a pathname+PID assertion. `Streams` are explicitly bounded; control records and child output cannot share framing. `CleanupReport` names opaque job/container handles and independently observed terminal state, without claiming the local report is a remote attestation.

Concrete custody boundary records: `Command{Executable string, Args []string, Dir string, Env []string, RuntimeDigest string, DeadlineMS int64}`; `Streams{Stdin io.Reader, Stdout io.Writer, Stderr io.Writer, StdoutLimit int64, StderrLimit int64}`; `Result{JobID string, Started bool, Completed bool, ExitCode int, Signal string, Cleanup CleanupReport}`; `Options{HelperExecutable string, HelperDigest string, ScratchRoot string, Limits Limits, OwnerLiveness *os.File}`; `CleanupReport{Status string, Jobs []ResourceState, Containers []ResourceState, Diagnostics []Diagnostic}`; `ResourceState{ID string, Registered bool, Terminated bool, Reaped bool}`. Status is exactly cleaned or cleanup-failed; container Reaped means removed from the daemon rather than waitpid. `InternalIO` supplies only verified inherited control/owner descriptors and bounded diagnostics; it is not project-configurable stdio. Capability's transport fields remain private; Join verifies the live broker handshake before use. The broker rejects absent runtime identities, custom inherited FDs, mutable executable paths, unsupported SysProcAttr and undeclared environment rather than silently reducing the claim.

Keep `processcontrol.Run(ctx, *exec.Cmd) error` as a compatibility API. Add explicit `processcontrol.WithScope(ctx, scope)` and `processcontrol.AttachScope(cmd, scope)`; the scoped path delegates to scope.Run and preserves `errors.Is` cancellation semantics and an inspectable target-exit error type. Do not assume callers can receive the original `*exec.ExitError` across a broker: audit/update exact dependent type assertions, or supply a documented `processcontrol.ExitStatus(err)` accessor. Unsupported stdin/file-descriptor/custom SysProcAttr use fails explicitly in scoped execution instead of falling back unscoped.

`ServeInternal` is wired into the candidate Machinery CLI and contributor-lane executable before ordinary CLI parsing, behind an authenticated inherited control channel; an untrusted environment variable alone cannot execute internal helper mode. Native test subprocesses JOIN the existing broker via an explicit verified attachment set by the lane. Existing sanitized environment constructors intentionally discard ambient values: formal/runtimeclosure calls must receive the scoped attachment explicitly AFTER sanitization, while preserving their current Java/JAR identity checks. No broad ambient environment forwarding.

Contributor integration requires the reviewed `internal/processscope`, `internal/processcontrol`, and explicit formal/runtimeclosure call-site wiring, plus separately owned supplemental tests and fragments. Final-Ga scoped execution additionally requires `internal/gates/accept.go`, suite RunOptions, runtimeclosure.Git and assuranceflow wiring. Preserve every original frozen RED/config/test inventory and failed record byte-for-byte. A provisioning-only wrapper is insufficient. Approve exact supplemental test names and ownership before authoring their RED.

Required real custody evidence on Linux amd64 AND Darwin arm64: observe live pinned nested JVMs before outer cancellation during BOTH provisioning and actual formal/meta-test execution; test timeout, signals, output overflow, normal safe/unsafe completion, failed assertion, early intermediate exit, failed registration and stale/forged scope requests. Prove intended owned descendants gone before return and unrelated live process/container plus caller-root sentinel preserved. A Linux cross-build, no-JVM-started negative, or outer elapsed-time bound is not this evidence.

## 9. Storage, security and operational behavior

Reuse existing bounded readers, immutable snapshots and transactional publication. External-store publication stages a complete object graph and atomically publishes references. Partial/interrupted records never count as completion. Objects are immutable and failed operations preserve prior observations/raw history. A durable record of native aggregate pass is recorded-only, not final workflow success: publication/close/output failure prevents the invocation seal even if some observation bytes became durable. Recovery verifies explicit store ownership and every reference before publishing/pruning; never delete unknown sentinels/journals to force progress.

Private execution roots are fresh, owned, and outside the checkout. No writable source input is shared between baseline/control/challenge/GREEN runs; each gets its own output/home/tmp/cache roots. Preserve the caller's root identity and unrelated files. Deterministic verdicts use exact identities/sets, not timing/output ordering; retain timestamps and raw ordering for diagnostics but exclude them from semantic verdict identity. Use fixed declared seeds; repeated reruns must not hide flaky failures. A failure remains failure, not an automatic retry until green.

Report per-suite counts, assertions, required obligations, source/runtime digests, limits, run scope, cleanup and trust label. No secrets in committed manifests. Native environment inputs must be explicitly declared and hashed; secret-dependent external services cannot support reproducible local complete assurance without a separately reviewed fixture/credential isolation contract. Do not log secret environment values or run tokens. Local service startup/reset/cleanup profiles must be closed and real; unsupported services yield an explicit blocker, not a mocked pass.

## 10. Migration and first-release completeness

Do not synthesize historical RED from old receipts or infer it from commit-message markers. Existing examples must receive new prospective assurance revisions, reviewed safe/unsafe calibration, frozen tests and actual current replay. Old acceptance evidence remains historical. Every shipped example already claiming complete implementation assurance MUST complete migration under an explicitly owned prospective revision before the new release can claim that lane passes. Existing design-only examples may stay visibly design-only; an implemented example cannot be reclassified/deselected merely because its adapter migration is hard. The complete-example lane must not downgrade to ordinary check, hide required tests or borrow a previous native PASS as strict assurance.

At least one real standalone consumer capstone per language must pass store-init/scaffold/capture/register/RED/GREEN/verify/complete and hooks with no tracker/toolchain-management utilities installed. Their intended unsafe variants must demonstrably fail the correct assertions. Machinery's own safety regression suite uses the required contributor lane and four adapter conformance fragments, plus same-CLI consumer tests; do not confuse self-hosting design files with the contributor registry.

The strict Go profile is NOT automatically compatible with existing Go CRM or Machinery contributor tests: direct T.Fatal patterns remain valid under their EXISTING native contributor/test contracts, but do not establish strict assertion-specific RED. Do not rewrite frozen tests under unrelated change authority, hide them through selectors, or claim compatibility from an old native PASS. Prospective adapter/example migration requires a new reviewed test revision and full required-test mapping. Retain original approved/failing RED and accepted history, preserve every existing required regression (running legacy native regressions independently until properly migrated), and provide the NEW strict tests' own same-assertion calibration. Existing BUILD/evidence/golden/helper maintenance does not authorize wholesale Go CRM test rewrites. Machinery's contributor registry remains independently authoritative; it need not masquerade as a consumer TDD manifest or accept imported TDD receipts.

Implement this contract in bounded, explicitly owned units: core schema/inventory/storage and authored-revision registration; native custody; four adapter implementations; replay/state machine; CLI/gate/hook integration including scoped final-Ga Git; example/runtime-residual migration; standalone final capstones. Every interface must have real producer/consumer wiring and frozen positive/negative tests. No helper or adapter may ship uncalled by normal complete verification.

The contributor runtime inventory must include provisioned Python, TypeScript compiler, Elixir/OTP, the separately scoped pinned Git closure for final Ga, and their exact source/runtime identities through separately owned fragments rather than editing a frozen pilot. Preserve existing lane semantics; if a frozen schema cannot express a new adapter requirement, retain it as v1 history and introduce an independently reviewed compatible v2 lane contract plus complete union migration. Never silently edit frozen v1 and call it the same RED.

Runtime stories must use the shared required lane with missing-prerequisite failures, no `skip-if-missing`, genuine processes/services and their own fragments. Existing per-story required Linux execution remains required; architecture only permits local native final verification, not substitution for unavailable platform proof. Full scripts/preflight.sh is still reserved for the final integrated gate; targeted native/profile/custody tests are the implementation feedback loop.

## 11. Implementation acceptance obligations

These obligations require executable acceptance evidence; the document alone is not sufficient:

1. Exact closed schema/API/CLI names above, version catalog, all numeric bounds (including shared milestone/owner deadlines and one final cleanup grace) and unsupported behavior; duplicate/unknown/path/alias/missing/null errors each have negative tests. Root-level design/implementation/frozen/suite/dependency roots accept only the explicit RootPath "." spelling; file paths and aliases cannot use it as a loophole.
2. Bundle topology includes empty directories, permission bits and rooted generation; control/reference ownership includes explicit design+milestone+suite+test. Authoritative inventory cannot shrink by file deletion, gate-list omission, opt-out, parent decomposition or short-ID collision. Whole-spec obligations and exact native selection reconcile both ways.
3. Finalization keeps root scope and runtime closures alive through ALL closing subprocesses, explicitly threads GateExecution into real Ga Git helpers, closes final child/root before runtime/materialization Close and final Snapshot.Release, and permits no target launch after that barrier. Positive actual Git queries and negative after-close/cancel/descendant/runtime-probe cases must prove that boundary through normal complete execution, not fake Git. Finalization must reject every cleanup/release/late-Gv/reacquisition/publication/store-close/output failure, and return no sealed result on error; external-store publication must preserve unchanged full-root Gv even with --impl repo. Four real adapter matrices include valid baseline+safe/unsafe pair+GREEN, every listed adapter-negative oracle, and source/helper/config/runtime drift. Parser fixtures supplement, never replace actual native execution.
4. EVERY baseline failing test/assertion, not just negative-role tests, requires an exact red_controls link to the SAME test/assertion passing on a retained safe-control variant. Expected RED requires registered native assertion causality AND that passing same-test safe control. Unconditional fail, never-called helper, thrown setup error, missing second assertion, no-op challenge and compilation failure cannot satisfy RED.
5. Freeze complete exact input inventories, including tests/helpers/fixtures/dependencies/config/runner/variants/budgets. String-literal whitespace, indentation, file addition/deletion, ignored import, symlink swap and runtime replacement invalidate evidence. Frozen amendments create a new revision and new replay, preserving old evidence.
6. Tamper with every receipt count/hash/pass field, remove retained source, replace baseline, narrow selector and change implementation after GREEN: final verification must rerun and fail appropriately. Freshness-only `check`/hook must visibly refuse to claim replay.
7. Actual custody tests cover normal/error/cancel/provisioning and nested JVM/meta-test boundaries on both required native platforms; foreign process/container/PID-reuse controls prove cleanup authority rather than broad process killing.
8. Same immutable view across source gates, tests and publication; concurrent checkout mutation and interrupted object publication fail closed without erasing prior evidence or unrelated files.
9. Initial check profiles have a real closed CheckExecutor producer/consumer boundary; imported/mismatched/pending gate results cannot authorize final success. Normal CLI plus default/staged hook plus `check --complete` use the same contract. Explicit gate lists cannot bypass required assurance. Final replay uses an in-memory invocation result only; imported JSON cannot construct it.
10. Shipped examples and four real consumer capstones validate runtime residual ownership and explicit design-only/unsupported migration. Required lane provisions all four runtimes before testing; no optional services or silent subset.
11. Publish separate design/source/test/replay/judgment/formal/custody/provenance dimensions with exact counts and residual claims. Verify documentation cannot describe a local self-written receipt as authenticated execution or a replay as historical chronology.
12. Prove store-init generation 0 -> first explicit register -> registered RED and N -> N+1 registration using the real CLI/store. Negative cases include unregistered RED, old/incorrect expected head, concurrent compare-and-advance, incomplete/dangling selected controls, plan change without advancing all affected registered milestones, deleted registered milestone, same-number replacement, missing predecessor, failed validation, postcommit close/output ambiguity, exact retry, retry after another head advancement, and rollback/import without the expected chain. Registration never calls tests or constructs Verification; execution never advances the authored head. Preserve prior failed RED and immutable controls across all cases.
13. Preserve original failed audit/history/frozen RED, existing user work, active installed Machinery and user-owned resources. No process exception, source-test waiver or installation replacement is implied by this architecture.

## 12. Alternatives, tradeoffs and review risks

- A separate explicit registration transaction was selected over implicit activation during RED: it adds one command but removes bootstrap ambiguity, distinguishes authored state from execution, and preserves failed-revision numbering. Exact plan changes conservatively advance all affected registered milestones; this is more work than silently rebinding old frozen qualifications.
- Final gate execution stays inside a live child/root scope; closing execution custody before Ga was rejected because Ga launches actual Git. The final no-launch barrier makes cleanup ordering auditable, at the cost of retaining runtime handles through closing checks.
- Generic commands, JUnit/TAP files and success exit codes were rejected as authoritative evidence: they hide selection, failure cause and missing execution. Closed adapters require more initial work and explicit compatibility maintenance.
- The first release requires all four languages. Supporting every framework immediately would be an unbounded promise; unsupported frameworks receive actionable migration diagnostics.
- Hash-only receipts were rejected for final assurance: a repository writer can forge them. Replay costs more, so normal hooks remain cheap and explicitly weaker. Future independently trusted signing/remote custody is a different security product decision.
- Requiring every negative test to fail a safe-default RED stub was rejected: it rewards unsafe stubs and misstates negative testing. Same-test safe/unsafe pairs measure sensitivity directly, at the cost of reviewed variants.
- Whole-tree kernel containment or OCI-only execution would better contain arbitrary detachment but changes native/platform/source/service contracts. User authorized native execution; this decision chooses explicit cooperative custody with clear unsupported escape behavior, not a false hostile-code guarantee.
- Pure descendant scanning/PID files were rejected due discovery/reuse races. The guardian mechanism depends on exact kill-before-reap ownership and cooperative attachment; its real Linux/Darwin challenge is an implementation acceptance prerequisite, not a proven property of this document.
- Exact byte freezing is intentionally stricter than “semantic formatting.” A correction costs a new revision and replay; this preserves honest evidence rather than treating unreliable token comparisons as semantics.

Technical feasibility requires actual implementation evidence, especially guardian custody and adapter assertion/event/source correlation. If real tests cannot meet these contracts, stop and revise the architecture through independent review rather than weaken a required outcome to match available code.
