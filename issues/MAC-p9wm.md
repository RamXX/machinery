---
id: MAC-p9wm
title: "Publish standalone native custody contract"
status: closed
priority: 0
type: task
labels: [docs, accepted]
parent: MAC-ui8a
created_at: 2026-09-06T12:04:37Z
created_by: ramirosalas
updated_at: 2026-09-06T12:31:39Z
content_hash: "sha256:e3e7436904c211526cb43067b88718713a055c3e4d9bdab6c1d624a06f80f8d5"
related: [MAC-l7m0]
assignee: dev-MAC-p9wm
follows: [MAC-l7m0, MAC-uzxr]
closed_at: 2026-09-06T12:31:38Z
close_reason: "Accepted: independent whole-prose review verified AC1-AC8, original plus all 19 negative mutants, exact two-path Git object/byte preservation, Pandoc structure/link rendering, and honest docs-only proof limits; frozen PM report /tmp/MAC-p9wm-pm-review.Hf8lA2/REPORT.md SHA256 19c0c4be6b4eb4b7035db3e9e576352ad1cfd3b24f414e287be7d8dfded38f8e"
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
## Implementation Evidence

PROOF:

### CI/Test Results
Commands run:
- `git diff --check a94e768adf461178e0562ad135c7c06d8163a3a4 2a63e4bb0d7b57c42dfd6f45fecfdc4f8a18cfbf -- docs/native-custody-contract.md docs/test-assurance-contract.md`
- exact two-path identity, mode, byte reconstruction, relative-link, heading/fence, candidate-object, conflict, and contamination checks recorded in the prior evidence block and immutable report
- `pvg verify docs/native-custody-contract.md docs/test-assurance-contract.md --format text`
Summary: bounded documentation checks PASS with 0 final failures; 0 product/native tests run and 0 applicable; independent complete-prose PM review pending.
Coverage: N/A (docs-only publication).

### Commit
Commit SHA: 2a63e4bb0d7b57c42dfd6f45fecfdc4f8a18cfbf
Base SHA: a94e768adf461178e0562ad135c7c06d8163a3a4

### AC Verification
- [x] AC #1: status/trust/standalone/unproved contract published.
- [x] AC #2: exact API/lifetime/ownership/budget contract published.
- [x] AC #3: exact runtime/service/profile/cleanup contract published.
- [x] AC #4: live-first/migration/native-matrix contract published.
- [x] AC #5: one-line companion and exact accepted-byte reconstruction.
- [ ] AC #6: developer artifacts delivered for independent PM review.
- [ ] AC #7: developer bounded checks pass; independent prose/render review pending.
- [x] AC #8: exact developer evidence and true-EOF contract recorded.

LEARNINGS:
- Parser-required evidence labels must begin at column zero in Notes; the final authoritative contract remains a true-EOF Comment.

## Implementation Evidence

PROOF:

### CI/Test Results
- Commands run:
  - `git diff --check a94e768adf461178e0562ad135c7c06d8163a3a4 2a63e4bb0d7b57c42dfd6f45fecfdc4f8a18cfbf -- docs/native-custody-contract.md docs/test-assurance-contract.md`
  - exact `git diff --name-status/--numstat/--shortstat`, `wc`, `stat`, `shasum -a 256`, `git hash-object`, and `git ls-tree` inventory
  - byte-for-byte `cmp` of the accepted base companion against the candidate with only line 5 removed
  - relative-link, heading, fence-pair, conflict-marker, candidate-object, and bounded public-contamination checks
  - `pvg verify docs/native-custody-contract.md docs/test-assurance-contract.md --format text`
- Summary: documentation diff/identity/reconstruction/link/structure/contamination checks PASS (10 bounded check classes, 0 failures after corrections); product unit/integration/E2E/native tests 0 run / 0 applicable. `pvg verify` reported PASS but scanned 0 documentation files, so it is a limitation rather than semantic evidence. Markdown linter/render executable unavailable; independent complete-prose/render review pending.
- Coverage: N/A for docs-only publication. No runtime or native proof claimed.
- Preserved correction: initial `pvg story verify-delivery MAC-p9wm` passed 3 and failed 6 because complete evidence was recorded in Comments rather than parser-visible Notes. This block supplies the exact required Notes headings; candidate commit is unchanged.

### Commit
- Branch: `story/MAC-p9wm`
- SHA: `2a63e4bb0d7b57c42dfd6f45fecfdc4f8a18cfbf`
- Base: `a94e768adf461178e0562ad135c7c06d8163a3a4`

### pvg verify
- `VERIFY: PASSED (0 files scanned, 0 issues)`; documentation exclusion recorded, not used as correctness proof.

### AC Verification
| AC # | Requirement | Location/evidence | Status |
|---|---|---|---|
| 1 | status/trust/standalone/unproved | new doc opening, §§1/8 | PASS |
| 2 | exact API/lifetime/ownership/budgets | new doc §§2-4 | PASS |
| 3 | exact runtime/contributor/checker/cleanup | new doc §§5-7 | PASS |
| 4 | live-first method/roles/migration/matrices | new doc §8 | PASS |
| 5 | minimum companion and byte preservation | companion line 5 plus exact reconstruction | PASS |
| 6 | coverage and negative review | full developer matrix delivered; independent PM pending | DELIVERED FOR REVIEW |
| 7 | bounded validation | exact checks above; independent PM pending | DELIVERED FOR REVIEW |
| 8 | exact evidence and true EOF | comments plus authoritative EOF contract | PASS |

LEARNINGS:
- Tracker verification parses Implementation Evidence from Notes, while true-EOF contract authority requires a final Comment; docs-only delivery needs both placements.
- Byte reconstruction and semantic matrices are the meaningful evidence when a generic source verifier intentionally scans zero documentation files.

## nd_contract
status: delivered

### evidence
- Transitioned via pvg story deliver on 2026-09-06.

### proof
- [ ] Developer evidence block must remain authoritative above this contract.


## History
- 2026-09-06T12:04:53Z dep_added: blocks MAC-qlw2
- 2026-09-06T12:04:53Z dep_added: blocks MAC-vx24
- 2026-09-06T12:04:53Z dep_added: blocks MAC-ou97
- 2026-09-06T12:12:01Z status: open -> in_progress
- 2026-09-06T12:12:01Z auto-follows: linked to predecessor MAC-l7m0
- 2026-09-06T12:12:01Z claimed by dev-MAC-p9wm
- 2026-09-06T12:21:36Z status: in_progress -> in_progress
- 2026-09-06T12:21:36Z auto-follows: linked to predecessor MAC-uzxr
- 2026-09-06T12:31:38Z status: in_progress -> closed
- 2026-09-06T12:31:39Z dep_removed: no_longer_blocks MAC-qlw2
- 2026-09-06T12:31:39Z dep_removed: no_longer_blocks MAC-vx24
- 2026-09-06T12:31:39Z dep_removed: no_longer_blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Related: [[MAC-l7m0]]
- Follows: [[MAC-l7m0]], [[MAC-uzxr]]

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


### 2026-09-06T12:10:20Z ramirosalas
AUTHORITATIVE PUBLIC PROJECTION CLARIFICATION — ENDPOINT PREPARATION AND CHECKER EXECUTION

This append-only clarification supplements APPROVED PUBLIC PROJECTION sections 5–8 and Acceptance Criteria 2–4. It preserves every earlier byte, clause, path, dependency, status, hold, and evidence record. It is exact projection of the already approved V4 architecture, not a new design decision, implemented flag, source/test grant, runtime action, or native proof.

### Explicit once-only Docker endpoint preparation and transport

The public contract must state that “no ambient daemon discovery” prohibits ambient authority during capture inheritance, mutation, run, replay, and cleanup; it does not prohibit the following single owned bounded preparation step:

- The contributor/integration lane exposes `--docker-endpoint <absolute local socket>` and passes that concrete endpoint to `CaptureDockerRuntime`. An explicitly supplied endpoint wins. Otherwise, preflight performs its existing Docker-context inspection exactly once during owned bounded provisioning, resolves one concrete local Unix socket endpoint, records/freezes that selection, and passes the same explicit value to both the lane and the checker command.
- The standalone checker command exposes the corresponding `--docker-endpoint <absolute local socket>`. When no explicit endpoint is supplied, that command may perform its existing context resolution exactly once during owned bounded preparation, freeze the resulting concrete descriptor, and must never consult ambient Docker context again during run or replay.
- Unsupported remote endpoints fail before replay. A context name, environment variable, caller string, old descriptor file, or later re-resolution is never runtime authority. Run and replay use the same captured live runtime/daemon binding and private client configuration.
- `CaptureDockerRuntime` is identity capture, not image provisioning or pulling. The separately owned provisioning step pulls only the exact authorized pinned image through registered custody, then capture freezes the runtime descriptor before native selection/replay. Checker execution never pulls during run or replay.
- The public document describes these flags and steps as required future behavior. It must not imply that the current binary already implements the flags or that documentation performed provisioning/capture.

### Actual checker command context, scope, runtime, run, and replay binding

The public contract must state that the normal checker command propagates its real command cancellation/deadline context, authenticated scope, and captured Docker runtime through checker orchestration, per-checker validation, and BOTH the primary run and committed-evidence replay. Required scoped execution:

- uses the real command context, never `context.Background` or a replacement owner context;
- retains the same authenticated owner/root and captured runtime across run and replay, with any phase child bounded by that SAME owner and the already cumulative deadlines;
- performs deterministic checker environment/sandbox sanitation first and applies the scoped attachment AFTER that sanitation, so sanitation cannot discard or rewrite custody and no ambient attachment is trusted;
- refuses a nil, stale, foreign, cancelled, closed, mismatched, or unscoped capability/runtime before target start;
- may retain service-free compatibility wrappers for separately bounded tests, but a required scoped command path cannot call or fall back to an unscoped compatibility wrapper for either run or replay;
- routes both real checker phases through the closed `RunCheckerDocker` profile and its accepted registry/image/platform/input/argv/work-root bindings rather than a direct generic `docker run --rm` client; and
- completes exact container/helper cleanup and validates absence before successful checker output, evidence publication, work-root release, or runtime release. Target result and cleanup result remain distinct, and cleanup failure prevents success.

The coverage matrix and independent whole-prose review required by AC6–AC7 must include these clarification rows. Add semantic negative mutants for: lane and checker receiving different endpoints; ambient context re-read between run and replay; remote endpoint admitted; image pulled during replay; `context.Background` substituted; replay left unscoped; attachment applied before sanitation and lost/mutated; service-free wrapper used by required execution; or success/output/release occurring before cleanup. Each mutant must be rejected for the intended contract reason; keyword presence remains insufficient.

No private tracker identifiers or workflow chronology are needed in shipped prose. The public wording may use generic consumer roles, but must preserve the exact `--docker-endpoint` spelling, absolute-local-socket constraint, once-only owned preparation exception, same-endpoint freeze/transport, no-pull replay rule, and real run-plus-replay context/scope/runtime lifecycle above.

## nd_contract
status: new

### evidence
- Exact approved endpoint-preparation/transport and real checker run/replay binding added at true EOF without replacing prior Body/history.
- Ownership remains exactly two documentation paths; dependencies, labels, claim/status, implementation/native holds, and no-fourth-review decision are unchanged.

### proof
- [x] Self-contained projection now includes the once-only owned endpoint-resolution exception and exact `--docker-endpoint` transport.
- [x] Self-contained projection now includes actual command context/authenticated scope/captured runtime across both checker run and replay, post-sanitation attachment, and cleanup-before-success.
- [ ] AC #1–#8 remain pending developer delivery and independent PM acceptance, including the expanded coverage/negative-mutant review under existing AC2–AC4 and AC6–AC7.


### 2026-09-06T12:21:36Z ramirosalas
## Implementation Evidence (DELIVERED FOR INDEPENDENT REVIEW)

PROOF:

### Candidate and exact scope
- Base: `a94e768adf461178e0562ad135c7c06d8163a3a4`
- Candidate/producing commit: `2a63e4bb0d7b57c42dfd6f45fecfdc4f8a18cfbf` on `story/MAC-p9wm`
- Exact changed paths: added `docs/native-custody-contract.md`; modified `docs/test-assurance-contract.md`; no others.
- Diff: 329 insertions, 0 deletions; 27,995 net added bytes; below 600-line/60,000-byte investigation thresholds.
- New supplement: 328 lines / 27,783 bytes / mode 0644 / SHA-256 `bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a` / Git blob `1c1581d1aec324d979593613b44976ed0007c45b`.
- Companion candidate: 576 lines / 109,579 bytes / mode 0644 / SHA-256 `171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d` / Git blob `acce64fee8db5a7565e8fa33422334de125e913f`.
- Companion base: mode 0644 / Git blob `50bd00288484b46ba69d40ff0a3cdd9ff7e1ed02`.

### Minimal companion hunk
One line was inserted at `docs/test-assurance-contract.md:5`:
`The [native custody architecture contract](native-custody-contract.md) refines only this contract's native-custody portion; it neither replaces the remainder nor establishes implementation or execution evidence.`

Removing only that line reconstructs the exact accepted source: 575 lines / 109,367 bytes / SHA-256 `22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8`; process-substitution `cmp` against the base Git object passed.

### Coverage matrix
| Projection/clarification | Public heading | Status |
|---|---|---|
| 1 status, guarantee, trust residuals, standalone and honest unproved state | opening, §§1 and 8 | PASS |
| 2 exact constructors/Scope/records/Limits/Diagnostic/defaults/caps/positive-zero semantics/opacity | §2 | PASS |
| 3 root/descendant relationships, bounded authentication, acquisition/handoff, Join/Serve/Close, Attach | §3 | PASS |
| 4 cumulative wall/owner/deadline and one shared cleanup-only grace | §4 | PASS |
| 5 runtime API, 0400 descriptor, closed JSON/canonicalization, live identity, private Docker client | §5 | PASS |
| EOF endpoint: exact `--docker-endpoint`, once-only owned local resolution, same freeze/transport, remote/no-pull rules | §5 “Once-only endpoint preparation and transport” | PASS |
| 6 contributor types, five programs, pinned profile/resources, observations, create/register/start, unresolved cleanup | §6 | PASS |
| 7 checker type, run/verify registry, exact sandbox/input/evidence/replay profile | §7 | PASS |
| EOF execution: real command context/scope/runtime for run+replay, sanitation-before-Attach, no fallback, cleanup-before-success | §7 “Real command context, scope, runtime, run, and replay” | PASS |
| 8 live-first proof, calibration and four roles, migration, two native matrices/unproved state | §8 | PASS |
| refinement-only discoverability and byte preservation | companion line 5 | PASS |

### Developer semantic negative-mutant review
The complete developer review rejects all 19 required mutants for contract-specific reasons: probe cancellation cannot revoke handed-off work; missing broker cannot promote root; handle Close is process-free; Attach mutation rejects before start; descriptor text is not authority; unresolved create/removal cannot succeed; observations are read-only; safe reference is not production GREEN; image architecture is not native host proof; private workflow is not a dependency; lane/checker endpoints cannot differ; ambient context cannot be reread; remote endpoints fail; replay cannot pull; `context.Background` cannot replace command context; replay cannot be unscoped; sanitation precedes Attach; required execution cannot use the service-free wrapper; and cleanup/absence precedes output/publication/release. Full per-mutant reasoning and heading pointers are frozen in the external report below. This is self-review only; independent PM verdict remains pending.

### Bounded validation
- `git diff --check <base> <candidate> -- docs/native-custody-contract.md docs/test-assurance-contract.md`: PASS.
- `git diff --name-status`, `--numstat`, `--shortstat`: exactly A/M owned paths, 328+1 insertions, 0 deletions.
- `wc`, `stat`, `shasum -a 256`, `git hash-object`, `git ls-tree`: exact identities above; both modes 0644.
- Exact accepted-byte reconstruction with `cmp`: PASS.
- Relative target `docs/native-custody-contract.md`: PASS.
- Contamination scan limited to the complete new doc and added companion line: 0 private IDs/tool names/contract markers/user paths/names/Dagger or actor/agent handles/story/epic refs.
- Conflict markers: 0. Heading inventory: 8 numbered sections plus both EOF clarification subsections. Fences: 14/even. Candidate-object comparisons: PASS. Worktree clean.
- Markdown linter/render executable unavailable; direct heading/fence/link checks used. Independent PM must inspect complete prose/rendering.
- `pvg verify docs/native-custody-contract.md docs/test-assurance-contract.md --format text`: `VERIFY: PASSED (0 files scanned, 0 issues)`; this excludes docs and is a recorded limitation, not correctness proof. The skill's `--format=text` spelling first failed as unknown and was corrected from installed help.
- Product unit/integration/E2E/native/runtime/container tests: 0 run / 0 applicable. Documentation coverage: N/A. No preflight, infrastructure, provisioning, process, container, or native matrix executed.
- Preserved diagnostics: an initial nonexistent epic ref check failed and was corrected by resolving the base-containing branch (`epic/MAC-ui8a`) and reading the base log; an initial fixed fence-count assertion incorrectly expected 10 and was corrected to an even-pair check after inventory showed 14. Neither exposed a document defect.

### External immutable report
- `/tmp/MAC-p9wm-delivery.sC5L4f/REPORT.md`
- 103 lines / 12,852 bytes / mode 0444
- SHA-256 `01ee0a554b8d405cc519911272821318b46e1030212547fac1d6af5d4bab10f9`
- Contains the complete coverage matrix, 19-row semantic negative-mutant matrix, raw bounded results, exact object identities, minimal hunk, preserved failures, and limitations.

### AC verification
| AC | Developer delivery status |
|---|---|
| 1 | PASS — approved required-behavior status/trust/standalone/unproved language published. |
| 2 | PASS — exact API, record, acquisition, ownership, attachment, and budget contract published. |
| 3 | PASS — exact runtime, contributor/checker, profile, ordering, and conservative cleanup contract published. |
| 4 | PASS — live-first method, distinct roles, migration, matrices, and honest proof limits published. |
| 5 | PASS — one companion line only; accepted bytes otherwise exactly reconstructed. |
| 6 | DELIVERED FOR INDEPENDENT REVIEW — full mapping and 19-mutant developer self-review supplied; independent verdict not claimed. |
| 7 | PASS for developer checks; independent whole-prose/render review remains pending. |
| 8 | PASS for exact developer delivery evidence and true-EOF contract; PM acceptance remains pending. |

LEARNINGS:
- A one-line insertion plus byte-level reconstruction preserves an accepted long-form contract more reliably than editing its native-custody section in place.
- Documentation verifiers may intentionally scan zero files; exact byte/object/link checks and semantic review must carry the evidence without inflating them into native proof.

LIMITATIONS:
- Independent whole-prose and negative-mutant PM review is pending; no PM approval is represented.
- No implementation, RED/GREEN, runtime, cleanup, native-host, release, or universal-correctness proof was produced by this docs-only delivery.

### 2026-09-06T12:22:04Z ramirosalas
## nd_contract
status: delivered

### evidence
- Base `a94e768adf461178e0562ad135c7c06d8163a3a4`; candidate `2a63e4bb0d7b57c42dfd6f45fecfdc4f8a18cfbf`; branch `story/MAC-p9wm`.
- Exactly two owned documentation paths changed: 329 insertions, 0 deletions, 27,995 net added bytes; both mode 0644.
- New document: SHA-256 `bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a`, Git blob `1c1581d1aec324d979593613b44976ed0007c45b`.
- Companion candidate: SHA-256 `171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d`, Git blob `acce64fee8db5a7565e8fa33422334de125e913f`; base blob `50bd00288484b46ba69d40ff0a3cdd9ff7e1ed02`.
- Removing only companion line 5 reconstructs the accepted 575-line / 109,367-byte / SHA-256 `22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8` source byte-for-byte.
- Bounded validation PASS: diff check, exact path/numstat/object/mode inventory, byte reconstruction, relative link target, fence pairing, heading inventory, candidate-object equality, 0 conflict markers, and 0 bounded public-contamination matches.
- Complete developer coverage matrix maps projection §§1-8 plus both true-EOF clarifications to public headings. Developer semantic review rejects all 19 required negative mutants for contract-specific reasons.
- External immutable evidence: `/tmp/MAC-p9wm-delivery.sC5L4f/REPORT.md`, mode 0444, 103 lines / 12,852 bytes, SHA-256 `01ee0a554b8d405cc519911272821318b46e1030212547fac1d6af5d4bab10f9`.
- Documentation verifier limitation recorded: corrected `pvg verify ... --format text` returned `VERIFY: PASSED (0 files scanned, 0 issues)` and is not used as semantic proof. Markdown linter/render executable unavailable; direct structure/link checks passed.
- Product/native tests: 0 run / 0 applicable; coverage N/A. No runtime, provisioning, process, container, preflight, native matrix, remote, merge, or release action occurred.

### proof
- [x] AC #1: standalone approved required-behavior status, trust residuals, and honest unproved state published without private dependency or execution claim.
- [x] AC #2: exact constructor/type/record/limit/acquisition/handoff/Join/Close/Attach/cumulative-budget contract published.
- [x] AC #3: exact Docker runtime codec/private client, contributor/checker APIs and profiles, ordering, unresolved-create, endpoint preparation, and real run/replay lifecycle published.
- [x] AC #4: live-first proof method, distinct pre-feature/reference-unsafe/reference-safe/production roles, migration, two native matrices, and proof limits published.
- [x] AC #5: one minimal companion line; every other accepted byte preserved exactly.
- [ ] AC #6: developer coverage and 19-mutant semantic self-review delivered; independent whole-prose PM verdict remains pending and is not claimed.
- [ ] AC #7: bounded developer checks completed; independent complete-prose/render review remains pending and is not claimed.
- [x] AC #8: exact developer hashes, counts, modes, minimal hunk, matrices, scans, limitations, and true-EOF delivery contract recorded; PM acceptance remains separate.

### 2026-09-06T12:24:03Z ramirosalas
## nd_contract
status: delivered

### evidence
- Frozen base/candidate: `a94e768adf461178e0562ad135c7c06d8163a3a4` -> `2a63e4bb0d7b57c42dfd6f45fecfdc4f8a18cfbf`; exactly two owned docs, 329 insertions / 0 deletions / 27,995 net added bytes, both 0644.
- Git blobs: new contract `1c1581d1aec324d979593613b44976ed0007c45b`; companion `acce64fee8db5a7565e8fa33422334de125e913f`; accepted companion base `50bd00288484b46ba69d40ff0a3cdd9ff7e1ed02`.
- SHA-256: new contract `bee289e553809b550b1f5cedf73415f600379ab54830a11733ba0dc7caaaba2a`; companion `171ebaa3e992b60bd90f1fc280818a125689390f8440b3a9a7b2c97e7f8b747d`; reconstructed accepted body `22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8`.
- Bounded document checks passed; developer coverage maps every projection and both endpoint/run-replay clarifications; all 19 required negative mutants rejected for specific contract reasons. Product/native tests 0 run / 0 applicable; no execution proof claimed.
- Primary immutable report: `/tmp/MAC-p9wm-delivery.sC5L4f/REPORT.md`, mode 0444, SHA-256 `01ee0a554b8d405cc519911272821318b46e1030212547fac1d6af5d4bab10f9`.
- Delivery-gate addendum: `/tmp/MAC-p9wm-delivery.sC5L4f/POST_DELIVERY.md`, mode 0444, SHA-256 `b97c47cd51e8c02856d09e9c990f160f9912ef5a6020bd0cd0ee9f48a88cb88c`.
- `pvg story verify-delivery MAC-p9wm` final result: 9 passed / 0 failed. Earlier 3/6 and 6/3 parser failures and their tracker-only corrections are preserved in the addendum; candidate objects never changed.

### proof
- [x] AC #1: status/trust/standalone/unproved contract published.
- [x] AC #2: exact API/lifetime/ownership/budget contract published.
- [x] AC #3: exact runtime/contributor/checker/profile/cleanup plus endpoint and real run/replay contract published.
- [x] AC #4: live-first method, distinct roles, migration, and two native matrices/unproved state published.
- [x] AC #5: one minimum companion line and exact accepted-byte reconstruction.
- [ ] AC #6: complete developer mapping and 19-mutant self-review delivered for independent whole-prose PM review; no PM verdict claimed.
- [ ] AC #7: bounded developer validation complete; independent prose/render review pending.
- [x] AC #8: exact evidence and authoritative true-EOF delivered contract recorded; acceptance remains separate.
