---
id: MAC-p9wm
title: "Publish standalone native custody contract"
status: open
priority: 0
type: task
labels: [docs]
parent: MAC-ui8a
created_at: 2026-09-06T12:04:37Z
created_by: ramirosalas
updated_at: 2026-09-06T12:06:45Z
content_hash: "sha256:ea2115feee79538c387c4480e43fa9e271a6bd241d13d540805cf7d92f2fc074"
related: [MAC-l7m0]
blocks: [MAC-qlw2, MAC-vx24, MAC-ou97]
---

## Description
## USER INTENT
Publish a precise, standalone Machinery contract so implementers and consumers can understand the approved native custody boundary without access to private coordination artifacts, while preserving the already accepted test-assurance contract and making no unearned implementation or native-proof claim.

## CONTEXT (EMBEDDED)
The accepted public architecture source is `docs/test-assurance-contract.md` at epic object `a94e768adf461178e0562ad135c7c06d8163a3a4`, 575 lines / 109367 bytes / SHA256 `22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8`, produced and accepted by MAC-l7m0. Its sections 4, 7, and 8 define cumulative budgets, public internal interfaces, and native custody. A terminal independent review approved the bounded native-custody refinements summarized below with no blocking architecture findings. That approval is architecture only: it does not prove implementation, RED, native execution, Linux/Darwin evidence, cleanup, or release.

This story is the legitimate public documentation owner for the approved supplement. The private proposal is input to this story, not a shippable normative artifact. The developer must project the complete contract below into product documentation and must not copy private tracker IDs, workflow/tool names, local paths, actor handles, source-branch chronology, protected-resource identifiers, or unearned evidence claims into shipped prose. Machinery must have no Paivot product/runtime/build/test dependency.

MAC-qlw2 remains the separate source/test producer. This story grants no processscope source, test, fixture, calibration-variant, native-execution, runtime, container, installation, or remote authority. Publication must land and be independently accepted before MAC-qlw2 source/RED work proceeds.

## OWNERSHIP
Only these two documentation paths are authorized:

- `docs/native-custody-contract.md` — NEW standalone normative architecture supplement.
- `docs/test-assurance-contract.md` — ONLY the minimum explicit companion link/refinement notice needed to make the supplement discoverable and disambiguate precedence; preserve the accepted 575-line body otherwise byte-for-byte.

No README, release note, integration guide, source file, test file, fixture, generated asset, workflow, or other documentation path is owned here. You are not alone in the codebase; preserve all other edits.

## APPROVED PUBLIC PROJECTION CONTRACT

### 1. Status, guarantee, and trust boundary
The new document must identify itself as an approved architecture contract that refines the native-custody portion of the accepted standalone test-assurance contract. It defines required behavior, not completed implementation or executed proof. Preserve the accepted trust boundary: native cooperative custody targets mistake-prone cooperating code on a trusted developer/CI host; it does not claim protection from unrestricted same-user host access, hostile escape, simultaneous broker/host/kernel failure, hermeticity, historical authorship order, hard real-time termination, or universal correctness. `provenance: unauthenticated-host`, `isolation: native-cooperative`, and custody `cleaned | cleanup-failed | unsupported` remain separate claims.

### 2. Exported constructors, existing boundary records, and representation
Publish these exact additional signatures together with the already accepted Scope boundary:

```go
func InheritedInternalIO(ctx context.Context) (InternalIO, bool, error)
func (io InternalIO) Close() error
func InheritedCapability(ctx context.Context) (Capability, error)
func (cap Capability) Close() error

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

Retain the accepted record shapes verbatim:

```go
type Command struct { Executable string; Args []string; Dir string; Env []string; RuntimeDigest string; DeadlineMS int64 }
type Streams struct { Stdin io.Reader; Stdout io.Writer; Stderr io.Writer; StdoutLimit int64; StderrLimit int64 }
type Result struct { JobID string; Started bool; Completed bool; ExitCode int; Signal string; Cleanup CleanupReport }
type Options struct { HelperExecutable string; HelperDigest string; ScratchRoot string; Limits Limits; OwnerLiveness *os.File }
type CleanupReport struct { Status string; Jobs []ResourceState; Containers []ResourceState; Diagnostics []Diagnostic }
type ResourceState struct { ID string; Registered bool; Terminated bool; Reaped bool }
```

Declare the ordinary Go representation choice for the existing semantic `Limits` record, with exact serialized names:

```go
type Limits struct {
    WallMS uint64 `json:"wall_ms"`
    CleanupMS uint64 `json:"cleanup_ms"`
    StdoutBytes uint64 `json:"stdout_bytes"`
    StderrBytes uint64 `json:"stderr_bytes"`
    EventBytes uint64 `json:"event_bytes"`
    EventCount uint64 `json:"event_count"`
    Jobs uint64 `json:"jobs"`
    BundleBytes uint64 `json:"bundle_bytes"`
    Entries uint64 `json:"entries"`
    Depth uint64 `json:"depth"`
}
type Diagnostic struct { Code string; Message string }
```

The exact field-order defaults are `600000, 10000, 4194304, 4194304, 16777216, 100000, 1, 536870912, 100000, 64`. Shipped absolute caps are 3600000 ms wall, 30000 ms cleanup, 64 MiB per output/event stream, 1000000 events, 4 concurrent jobs, 4 GiB per bundle, 1000000 entries, and depth 128. Values are positive and validated before duration/integer conversion. Explicit zero is invalid; no field and no all-zero `Options.Limits` record silently defaults or clamps. `Diagnostic.Code` comes from the existing bounded catalog; `Message` is bounded, repair-oriented, carries no secret/capability/environment bytes, and conveys no authority. Capability, InternalIO, DockerRuntime, broker frames, nonce/layout, descriptor slots, and ownership state remain opaque/private except for the public semantics below.

### 3. Acquisition, launch relationships, attachment, and lifetime
`Open(ctx, Options)` is the only root creator. It validates the exact helper closure, creates the private 0700 authority root and launch-specific channels/challenge, directly starts the first broker as its actual child, retains wait/cleanup ownership, and completes only after the broker authenticates the root-owner bootstrap peer and acknowledges readiness. Failed bootstrap releases new resources and terminates/waits that owned child within the already applicable shared cleanup budget. Marker text, descriptor number, claimed digest, path, PID, or environment claim is not authentication.

A live broker creates guardian channels and verified attachments for descendants. A nested target obtains an authenticated inherited capability for that existing broker and `Join` revalidates it. Missing/stale broker authority never promotes a guardian or nested target into a replacement root. Authentication is a bounded live exchange over broker/root-installed private channels independent of product stdio and binds candidate/helper identity, root/scope/job/role identity, owner/operation authority, and the launch challenge.

Both inherited constructors require non-nil acquisition/probe contexts with finite deadlines. Validation precedence is: (1) nil or no deadline -> `INVALID_SCHEMA`; (2) already cancelled/expired -> cancellation/`TIMEOUT`; (3) inspect bounded reserved intent; (4) authenticate present intent. Authentication has one maximum of 5000 ms from constructor entry, bounded by the earlier probe deadline; all reads/retries share it and the accepted 16 MiB control-record cap. Consumers check error before bool. `InheritedInternalIO` returns zero/false/nil only for successful absence; present-invalid intent returns zero/true/error and cannot fall through to ordinary parsing. `InheritedCapability` treats absent required attachment as `CUSTODY_ERROR`. No error permits unscoped/root fallback.

Acquisition owns one atomic terminal state, `PENDING -> HANDED_OFF` or `PENDING -> FAILED`. Cancellation winning before handoff releases acquired resources; success transfers exactly one usable handle. A successful handle retains only the independently authenticated original owner deadline, operation deadline, root/scope identity, and owner-liveness authority. Probe cancellation/expiry after handoff does not revoke work. Original authority expiry or liveness loss still enters cancellation/cleanup.

`Join` consumes Capability on entry and uses a separate work context. Nil is `INVALID_SCHEMA`; prior cancellation/expiry fails before launch. A non-nil context without a deadline is permitted because authenticated parent/operation authority is already finite. Effective work deadline is the earliest authenticated owner/operation deadline and Join deadline, if present. Join cancellation cancels the joined scope. Pre-handoff cancellation releases transport; post-handoff cancellation closes/cancels the returned scope before later launch. `ServeInternal` consumes InternalIO on entry, owns it through execution/cleanup, and rejects zero/consumed/invalid/wrong-role IO with handled=true and exitCode=1 before product/target action. A valid internal caller receiving handled=false treats it as protocol failure.

Opaque copies share one atomic ownership state: `UNUSED -> CONSUMED` or `UNUSED -> CLOSED`; exactly one concurrent consume/Close wins. A second consume fails. Close on zero/CLOSED/CONSUMED is idempotent. Unused-handle Close is local, process-free release only: close owned private descriptors, invalidate the state, return local close errors, never retry a possibly recycled descriptor, and never start a process, network handshake, child wait, runtime probe, reauthentication, or new grace period.

`Scope.Attach` preserves `Executable`, ordered `Args`, `Dir`, ordered declared `Env` excluding only broker-added reserved attachment representation, `RuntimeDigest`, and `DeadlineMS`, then adds reserved scoped state after sanitation. The exact binding covers those fields. `Run` revalidates command and attachment immediately before admission. Removal, duplication, mutation, staleness, changed command data, post-Attach sanitation, or scope/command/capability mismatch rejects before target start. A fresh Attach is allowed only while the scope remains OPEN and valid; Run cannot silently repair or fall back.

### 4. Cumulative work and cleanup budgets
Preserve `Limits.wall_ms` as one cumulative elapsed deadline per requested milestone/invocation work, never renewed per target, adapter, suite, state, retry, queue, scope, or phase. Preserve the fixed 14,400,000 ms owner ceiling from assurance-flow entry or the caller's earlier deadline. `Command.DeadlineMS` is remaining duration computed at dispatch, not an additional allowance after queueing.

Cancellation, any deadline, or first terminal error prohibits new verification work and enters FINAL cleanup with one shared monotonic grace: now plus the maximum selected `cleanup_ms`, capped at 30,000 ms; use 10,000 ms when no milestone loaded. All child/root/runtime/resource cleanup shares it. It cannot renew per resource or convert failure into success. CLOSING rejects public Run/Child/Attach/service starts. The broker may admit only fixed cleanup helpers for terminal-response settlement and exact-owned inspect/remove operations against the already bound daemon, each under the original remaining grace with its own registered guardian. Cleanup-only work cannot create/start/pull/provision/replay/tests or reopen general work. No process/runtime probe may start after final runtime release.

### 5. Docker runtime authority and codec
Publish the opaque runtime API:

```go
type DockerRuntimeRequest struct {
    ClientExecutable string
    ClientClosureDigest string
    DaemonEndpoint string
    ImageReference string
    ImagePlatform string
}
func CaptureDockerRuntime(context.Context, Scope, DockerRuntimeRequest) (*DockerRuntime, error)
func InheritedDockerRuntime(context.Context, Scope, string) (*DockerRuntime, error)
func (*DockerRuntime) Descriptor() []byte
func (*DockerRuntime) Validate(context.Context, Scope) error
func (*DockerRuntime) Close() error
```

The descriptor path class is `<fresh owned invocation root>/runtime/docker/<sha256-of-canonical-record>.json`, an owned regular non-symlink 0400 file written once and retained through final cleanup; no local daemon identity descriptor is committed. The broker independently retains canonical bytes/digest and live binding. File text, caller strings, labels, CIDs, or old JSON never construct authority. All keys below are mandatory; reject duplicate/unknown keys, nulls, trailing data, and arbitrary settings:

```json
{
  "schema": "machinery.custody.docker-runtime/v1",
  "invocation_id": "<64 lower hex>",
  "client": {
    "executable": "<canonical absolute regular snapshot path>",
    "closure_sha256": "<64 lower hex>",
    "version": "<actual nonempty bounded version string>"
  },
  "daemon": {
    "endpoint": "<canonical absolute local Unix socket path>",
    "socket_device": "<unsigned decimal string>",
    "socket_inode": "<unsigned decimal string>",
    "id": "<actual nonempty bounded daemon ID>",
    "server_version": "<actual nonempty bounded server version>",
    "os": "linux",
    "architecture": "<actual amd64 or arm64>"
  },
  "image": {
    "reference": "<canonical registry image@sha256:64hex>",
    "id": "sha256:<64 lower hex>",
    "os": "linux",
    "architecture": "<declared amd64 or arm64>"
  }
}
```

The record cap is 64 KiB; scalar path/version/ID fields are at most 4096 UTF-8 bytes; decimal device/inode is `0` or nonzero-leading digits fitting uint64; digests use exact lowercase grammar. Canonical encoding is the shown field order, compact standard-Go JSON escaping, one final LF; digest is SHA256 over those exact bytes and is not embedded recursively. Endpoint, socket object, daemon, image, client closure, root, bytes, mode, and digest are validated before every mutation while the scope is live. Runtime Close performs process-free final byte/topology/handle release and propagates errors.

Capture is identity capture, not image provisioning. It opens/snapshots the client closure and explicit local endpoint, queries actual client/daemon/image identity through registered native helpers, constructs/registers the live handle, and returns only after success. Inherited retrieval accepts a descriptor digest only within the supplied authenticated root/child scope and rejects absent, foreign-root, stale, or mismatched records. No ambient daemon discovery/import-as-authority.

The client launch is exact executable plus broker-owned `--config <fresh empty private config root>` and `--host unix://<captured endpoint>`, followed by the closed operation. The private HOME/config remains empty and byte/topology checked. Only explicit validated PATH/runtime necessities plus LANG=C, LC_ALL=C, and TZ=UTC are present. Do not forward Docker host/context/TLS/API/credential/plugin overrides, caller argv prefixes, shell, custom sockets, or user CLI configuration. Do not delete/chmod/replace/own the user's socket.

### 6. Closed contributor service and conservative daemon lifecycle
Publish these exact types and signatures:

```go
type ContributorContainerRequest struct { Program string }
type ContainerObservation struct {
    Sequence uint64
    Phase string
    JobID string
    ContainerID string
    Name string
    OperationNonce string
    ConfigDigest string
    DaemonID string
}
type DockerContainerState struct {
    Exists bool
    Running bool
    ID string
    Name string
    Owner string
    OperationNonce string
    ConfigDigest string
    DaemonID string
    ImageID string
}
type ContributorContainerResult struct {
    JobID string
    Started bool
    Completed bool
    ExitCode int
    Stdout []byte
    Stderr []byte
    Cleanup CleanupReport
}
func OpenContributorDocker(context.Context, Scope, *DockerRuntime) (*ContributorDocker, error)
func (*ContributorDocker) Run(context.Context, ContributorContainerRequest, chan<- ContainerObservation) (ContributorContainerResult, error)
func (*ContributorDocker) Close(context.Context) (CleanupReport, error)
func InspectDockerContainer(context.Context, Scope, *DockerRuntime, string) (DockerContainerState, error)
```

Run is synchronous and cleans before return; expose no live lease, Start handle, arbitrary argv, or public RegisterContainer. The closed initial `Program` mappings are:

- `python-version` -> `python3 --version`
- `print-42` -> `python3 -c "print(6*7)"`
- `sleep-300` -> `python3 -c "import time; time.sleep(300)"`
- `sleep-90` -> `python3 -c "import time; time.sleep(90)"`
- `sleep-900` -> `python3 -c "import time; time.sleep(900)"`

The current contributor capture binds the explicitly provisioned immutable image `python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc` on declared linux/amd64; actual host/daemon/image identities remain separately reported. The profile fixes image/platform and initially enforces network=none, read-only root, no mounts/ports/devices/capability additions/host namespaces/restart/auto-remove, memory 134217728 bytes, NanoCPUs 500000000, and PidsLimit 32; no overrides.

Observations are read-only data, never custody/removal authority. Sequence starts at 1 and increments; phases are exactly `registered`, `running`, `completed`, `removed`, `ambiguous`; maximum eight records under existing event/output caps. A nil channel requests none. Closed channel, overflow, or inability to deliver within the remaining deadline fails and cleans without panic/leak/success. `registered`, `running`, and `removed` follow actual validation/inspection. Read-only exact-ID inspection validates the same daemon and returns only the fixed safe fields. Unknown ID is Exists=false only after authoritative same-daemon absence; daemon error is not absence. No broad name search or secret exposure.

For every daemon job the broker creates a fresh 32-byte operation nonce and name `machinery-ci-` plus lower64hex, and records a root/scope/job/descriptor/daemon/config/name-bound intent before helper dispatch. Labels are `dev.machinery.integration-run=<broker root identity>`, `dev.machinery.integration-operation=<nonce>`, and `dev.machinery.integration-config=<config digest>`; they are locators, never authority. Register an actual guardian before releasing a create-only helper; dispatch exactly one create request; never retry ambiguous create; on acknowledged creation validate full canonical ID/name/nonce/config/image/daemon and Created/not-started state, register container ownership, recheck OPEN/budget, then start exact registered ID. Capture real wait/log/inspect status under limits and keep target error/exit distinct from cleanup.

A dispatched create without terminal server-backed response is `CREATE_UNRESOLVED`. Client death, timeout, pipe closure, empty label query, one inspect-not-found, or removal of a matching appeared object cannot prove no later creation. Within the original grace retain/wait the original request and inspect the exact reserved name. Only acknowledged create plus validated removal, or authoritative terminal precreation rejection, settles creation. Exhausted grace, lost/changed daemon identity, or ambiguous outstanding create/start returns `CUSTODY_ERROR`, cleanup-failed, and bounded owned-resource diagnostics; never success, retry, background sweeper, broad label/name deletion, prune, image/volume removal, or historic-PID recovery. Container ResourceState.Reaped means verified daemon removal.

### 7. Closed real-checker profile
Publish the finite checker boundary:

```go
type CheckerDockerRequest struct {
    WorkPath string
    WorkRoot *os.Root
    InputsDigest string
    RuntimeClosure string
    RunArgs []string
    VerifyArgs []string
    Phase string
    UID uint32
    GID uint32
    TimeoutMS uint64
}
func RunCheckerDocker(context.Context, Scope, *DockerRuntime, CheckerDockerRequest, Streams, chan<- ContainerObservation) (Result, error)
```

`Phase` is exactly `run` or `verify` and selects an already registry-bound argv vector. No arbitrary extra engine args, environment, mounts, or resource flags. The caller retains WorkRoot through result and cleanup. Validate canonical private WorkPath/WorkRoot identity, readonly runtime-input topology/digest, runtime closure, registry image/platform, command vectors, and actual config before launch and after execution.

Preserve the checker sandbox: pull=never, network=none, read-only container root, cap-drop ALL, no-new-privileges, workdir `/work`; writable bind of only WorkPath to `/work`; readonly runtime inputs to `/checker`; `/tmp` tmpfs `rw,noexec,nosuid,size=67108864`; fixed HOME=/work/home, LANG=C, LC_ALL=C, TMPDIR=/tmp, TZ=UTC, PYTHONNOUSERSITE=1, PYTHONSAFEPATH=1, empty PYTHONPATH, and `MACHINERY_CHECKER_RUNTIME_CLOSURE=<validated closure>`; validated UID/GID; initial memory 134217728 bytes, NanoCPUs 500000000, PidsLimit 32, no overrides. Preserve registry-bound `python3 /checker/adapter.py` run and `python3 -c` verification pass, rejection of `{design}`, canonical token/config/manifest projection, exact `/work/evidence.json` and `/work/committed/evidence.json` replay, file-only evidence output, evidence/trace comparisons, closure revalidation, design snapshot CheckUnchanged/Release, and final cleanup before successful output/release. Runtime-specific exact image identities belong to captured contributor inventories, not a timeless public promise.

### 8. Proof method, migration, and honest delivery state
Required custody evidence is live-first: an independently owned harness observes the exact announced resource live on the same daemon, validates daemon/name/root-or-owner/operation/config/image identity and Running=true, then releases the real trigger. Cover actual assertion failure while Run is active, host stdout/stderr overflow while the resource is active, cumulative timeout, real SIGINT and SIGTERM, owner-liveness loss, normal completion, registration failure, early intermediate exit, nested join, stale/forged/cross-scope/malformed/closed/reused handles, acquisition and admission races, and unresolved create/start. Prove exact owned absence and guardian/helper termination before success while separately owned foreign process/container controls survive every cleanup class. Container-output overflow/resource exhaustion belongs to the real checker profile, not the no-output sleep fixture.

Calibration freezes tests/helpers/fixtures/config/closure/budgets and every expected leaf before replay. A deliberately unsafe operative reference and its safe-control counterpart differ only in authorized implementation bytes; the same assertion must fail/pass while reaching the same real fixture. Use separate unsafe/safe pairs where invariants differ. A pre-feature or setup-failing source state is retained honestly but is not descendant-cleanup RED. The production candidate is a separate later state after independent RED approval; a reference safe control never substitutes for production GREEN or native delivery. No environment behavior toggle, test mutation, generic nonzero, document, receipt, or source keyword check proves custody.

Migration creates a new current reviewed revision while preserving every historical obligation, exact old source/test/control identity, failed record, and legacy regression as history. Every prior obligation maps to a current executed assertion that preserves or strengthens its outcome; renamed/restructured cases require explicit justification. New custody/runtime leaves are inventoried separately rather than inflating historical counts. Do not embed private issue history or workflow chronology in the public contract.

Required host matrices remain native Darwin arm64 and native Linux amd64 for the same candidate and frozen inventories. Host, daemon, and image architectures are reported separately; an amd64 image on an arm64 daemon/host is not Linux amd64 host proof. Until real implementation and both native matrices are accepted, documentation must say those outcomes remain required/unproved. The public document itself supplies no infrastructure, provisioning, process, container, installation, or execution authority.

## BOUNDARY MAP
PRODUCES:
- docs/native-custody-contract.md -> approved standalone native-custody supplement containing the exact exported constructors/types, acquisition/work lifetimes, opaque-handle ownership, cumulative budgets, Docker runtime codec, closed contributor/checker profiles, conservative cleanup, and live-first proof/trust limits above.
- docs/test-assurance-contract.md -> minimum companion link/refinement notice naming docs/native-custody-contract.md and stating that it refines the native-custody portion without replacing the remainder of the accepted contract.

CONSUMES:
- MAC-l7m0: docs/test-assurance-contract.md
  schema: accepted 575-line standalone executable test-assurance contract, SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; sections 4, 7, and 8 remain normative except for the explicitly linked approved refinements published by this story.

## ACCEPTANCE CRITERIA
1. `docs/native-custody-contract.md` is a self-contained approved architecture supplement whose status, guarantee, residual/trust limits, standalone dependency boundary, and unproved implementation/native-delivery state match projection sections 1 and 8; it neither references private coordination nor claims execution.
2. The public document reproduces all exact exported signatures, declared record shapes, `Limits`/`Diagnostic` representation, validation precedence, 5000 ms/16 MiB acquisition bounds, atomic handoff/copy/Close rules, Join work semantics, root-bootstrap/descendant relationships, Attach binding, and cumulative owner/cleanup budgets in projection sections 2–4 without weakening or inventing authority.
3. The public document reproduces the closed Docker runtime JSON/API/canonicalization/private-client contract and the contributor/checker types, signatures, profiles, resource limits, observation semantics, exact create/register/start/remove ordering, `CREATE_UNRESOLVED` treatment, and cleanup-only admission in projection sections 5–7.
4. The public document specifies the live-first positive/adversarial proof method, distinct pre-feature/reference-safe/reference-unsafe/production roles, historical-to-current obligation preservation, two native host matrices, and honest remaining-proof language in projection section 8; prose/static checks are never described as native evidence.
5. `docs/test-assurance-contract.md` changes only by one minimum companion/refinement link or notice; a byte-level comparison against the accepted 575-line source demonstrates every other original byte is preserved, and the notice does not imply the supplement replaces unrelated contract sections or proves delivery.
6. A coverage matrix maps every numbered projection section and every exact API/schema/profile/budget/proof clause above to public document headings. An independent reviewer reads the complete prose and records semantic verdicts for the original plus explicit negative mutants: probe context incorrectly controls post-handoff work; missing broker promotes root; Close launches cleanup; Attach mutation is tolerated; descriptor text grants authority; unresolved create/removal becomes success; observation grants mutation; reference safe control substitutes for GREEN; host/image architecture substitutes for native host proof; or Paivot/private history becomes a Machinery dependency.
7. Targeted documentation validation is bounded to the two owned paths: Markdown/render/link checks if supported, `git diff --check`, exact path/diff/stat/byte comparison, coverage/contamination review, and independent whole-prose semantic review. Keyword/count checks may assist navigation but cannot establish correctness; no fake Go RED, product coverage, runtime/native execution claim, heavy preflight, or hard-tdd label is permitted.
8. Delivery records exact base/subject hashes, changed line/byte counts, the minimal companion hunk, coverage matrix, contamination scan scope/results, reviewer findings, and an authoritative `nd_contract` at true EOF. Any architecture ambiguity stops for escalation rather than weakening the approved outcomes or starting a fourth architecture review.

## TESTING REQUIREMENTS
- This is docs-only architecture publication. Unit/integration/E2E product tests are not applicable and must not be invented. Do not create or edit test files.
- Independent whole-prose semantic review is mandatory. The reviewer must compare the complete new document against this embedded contract and the unchanged accepted source, inspect rendered links/headings, and adjudicate the named negative mutants with written reasons. A grep-only or count-only report is insufficient.
- Bounded checks may include Markdown/link tooling already present in the repository, `git diff --check -- docs/native-custody-contract.md docs/test-assurance-contract.md`, exact accepted-source reconstruction plus companion-hunk comparison, changed-line/byte counts, and a public-contamination scan limited to the new document and newly inserted companion text. Do not run `scripts/preflight.sh`.
- The accepted 575-line source is the comparison baseline. Preserve its body except for the single discoverability/refinement insertion. Do not silently reflow, rewrap, reformat, reorder, normalize whitespace, or update unrelated wording.
- Positive/original and negative-mutant artifacts are review evidence only, not production tests or native custody proof. No required native host/runtime/container is provisioned or exercised here.

## OUT OF SCOPE
- Any processscope, processcontrol, assurance, gate, runtime, checker, CLI, workflow, test, fixture, calibration, or implementation edit: owned by existing bounded implementation stories after this publication is accepted.
- Exact RED author/calibration-author selection, frozen leaf/helper/fixture/config inventories, reference implementation artifacts, actual RED/GREEN, and native Linux/Darwin execution: separate before-edit and delivery gates.
- Prospective amendments or ownership for downstream contributor/checker/lane stories: only their named owners may receive later, exact scope updates after this public contract lands.
- Any disposition, successor, cancellation, dependency transfer, or status change for the protected executable-coverage audit story: user decision only.
- README/release/integration guidance beyond the two owned paths: existing documentation/integration owners consume this accepted supplement later.
- Runtime installation, provisioning, container/process operations, active binary replacement, remote/GitHub mutation, sync/fetch/push, worktree/ref/branch changes, final preflight, merge, or release.

## DIFF BUDGET
- Exactly 2 documentation paths; under 600 changed lines and under 60000 changed bytes total. The budget is an investigation trigger only: required approved clauses must not be trimmed to fit it.

## MANDATORY SKILLS
- `developer` for the single docs-only delivery and append-only evidence.
- `pm_acceptor` for independent complete-prose semantic/negative-control review.
- No architecture redesign skill or fourth Anchor loop; ambiguities are escalated without changing the approved contract.

## DELIVERY REQUIREMENTS
- Work only in the dispatcher-owned story branch/worktree after this story becomes ready; preserve all concurrent edits.
- Do not apply `hard-tdd`. Do not claim a document, keyword check, or receipt is implementation/native proof.
- Developer delivers with exact evidence and proof; independent PM accepts or rejects. Neither role closes/accepts outside its authority.
- All tracker notes are append-only; the last true-EOF `nd_contract` is authoritative.

## nd_contract
status: new

### evidence
- Created from accepted public contract SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8 and terminal approved native-custody architecture; no implementation or native proof claimed.
- Accepted prerequisite MAC-l7m0 is satisfied and consumed; publication must be accepted before MAC-qlw2 source/RED work proceeds.

### proof
- [ ] AC #1: standalone status/trust boundary and honest unproved state published.
- [ ] AC #2: exact constructor/type/lifetime/ownership/budget contract published.
- [ ] AC #3: exact runtime/service/profile/conservative-cleanup contract published.
- [ ] AC #4: exact live-first proof/migration/native-matrix contract published.
- [ ] AC #5: accepted public contract preserved outside the minimum companion hunk.
- [ ] AC #6: independent whole-prose coverage and negative-mutant review complete.
- [ ] AC #7: bounded two-path checks complete without product/native proof inflation.
- [ ] AC #8: exact delivery evidence and true-EOF contract recorded.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-06T12:04:53Z dep_added: blocks MAC-qlw2
- 2026-09-06T12:04:53Z dep_added: blocks MAC-vx24
- 2026-09-06T12:04:53Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-qlw2]], [[MAC-vx24]], [[MAC-ou97]]
- Related: [[MAC-l7m0]]

## Comments

### 2026-09-06T12:05:48Z ramirosalas
PUBLICATION PREREQUISITE CREATION CHECKPOINT

MAC-p9wm is the sole public-document owner for `docs/native-custody-contract.md` plus the minimum companion notice in `docs/test-assurance-contract.md`. Accepted MAC-l7m0 is the satisfied normative source and is recorded as a related producer. Direct blockers added: MAC-qlw2, MAC-vx24, and final capstone MAC-ou97 each depend on MAC-p9wm. No implementation, test, source, native, runtime, ref, installation, preflight, remote, or protected-audit authority is conferred.

The terminal architecture remains approved with no fourth review loop. Exact inventories, author identities, actual RED/GREEN, and native Darwin arm64/Linux amd64 proof remain separate later gates. Public prose must strip private IDs/workflow/local chronology and must not claim those outcomes completed.

## nd_contract
status: new

### evidence
- Created as task/P0 under MAC-ui8a with label `docs`; accepted producer MAC-l7m0 related.
- Dependency edges verified at creation time: MAC-qlw2, MAC-vx24, and MAC-ou97 are blocked by MAC-p9wm.
- Scope is exactly two documentation paths and no `hard-tdd` label.

### proof
- [ ] AC #1-#8: docs-only delivery and independent semantic review pending.


### 2026-09-06T12:06:45Z ramirosalas
FINAL SR-PM VALIDATION CHECKPOINT

Terminology and source calibration: the accepted source document was read completely from exact epic object `a94e768adf461178e0562ad135c7c06d8163a3a4`, verified at 575 lines / 109367 bytes / SHA256 `22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8`. The approved V4 proposal and terminal Anchor were read completely at their recorded SHA256 identities. `docs/native-custody-contract.md` is absent at that object and is explicitly PRODUCED here. Every prospective API/schema string comes from the approved architecture and is labeled required future behavior, never existing Go implementation.

Structural results:
- `pvg nd dep cycles`: no dependency cycles.
- `pvg lint --backlog --epic MAC-ui8a --json`: zero errors; ten inherited `vertical-slice` review heuristics on other existing stories, none on MAC-p9wm. No repair is authorized or required here.
- `pvg rtm check --epic MAC-ui8a --json`: passed; 0 tagged requirements / 0 uncovered / 42 stories checked at command time.
- Read-back confirms MAC-p9wm is open/new, P0 task, parent MAC-ui8a, label `docs` only, related accepted producer MAC-l7m0, and directly blocks MAC-qlw2/MAC-vx24/MAC-ou97. MAC-qlw2 remains claimed/in_progress; MAC-vx24 and MAC-ou97 remain open/new; no accepted status changed.

Adversarial self-review verdict: MAC-p9wm = clean. Reality: both owned paths are grounded in the exact accepted Git object or this story's PRODUCES. Depth: the complete approved public projection is embedded, including precise APIs, schema, budgets, profiles, cleanup and proof limits. Scope: two docs paths only; adjacent implementation/RED/native/integration work has genuine separate owners. Coverage: positive source preservation and ten semantic negative mutants are enumerated. Hard rules: no hard-tdd, no fake product/native tests, no Paivot product dependency, no fourth architecture review, and no protected-audit disposition.

## nd_contract
status: new

### evidence
- Exact immutable inputs, canonical ownership/dependencies, cycle check, scoped lint, scoped RTM, terminology audit, and self-review recorded above.
- This is publication planning only; no document/source/test/native/runtime/ref/remote/install/preflight mutation or completion claim.

### proof
- [x] Story is self-contained, ≤8 AC, docs-only, exactly scoped, dependency-correct, and structurally clean.
- [ ] AC #1-#8 remain pending developer delivery and independent PM acceptance.

