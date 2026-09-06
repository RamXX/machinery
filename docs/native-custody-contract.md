# Native custody architecture contract

Status: approved architecture contract. This supplement refines only the native-custody portion of the standalone executable test-assurance contract. It defines required behavior; it does not claim that the behavior is implemented, that native execution has occurred, or that delivery evidence has been accepted.

This contract is standalone Machinery product, build, runtime, and test documentation. It grants no installation, provisioning, process, container, execution, or release authority and has no dependency on private coordination artifacts or workflow state.

## 1. Guarantee and trust boundary

Native cooperative custody targets mistake-prone, cooperating code on a trusted developer or CI host. It does not protect against unrestricted same-user host access, hostile escape, simultaneous broker/host/kernel failure, or active destruction of the supervisor. It does not claim hermeticity, historical authorship order, hard real-time termination, or universal correctness.

The claims `provenance: unauthenticated-host`, `isolation: native-cooperative`, and custody `cleaned | cleanup-failed | unsupported` are independent dimensions. None implies another. A cleanup timeout or custody ambiguity is a custody error and cannot be reported as success.

## 2. Public API and closed records

The additional inherited-handle constructors and the existing scope boundary are:

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

The boundary records retain these exact shapes:

```go
type Command struct { Executable string; Args []string; Dir string; Env []string; RuntimeDigest string; DeadlineMS int64 }
type Streams struct { Stdin io.Reader; Stdout io.Writer; Stderr io.Writer; StdoutLimit int64; StderrLimit int64 }
type Result struct { JobID string; Started bool; Completed bool; ExitCode int; Signal string; Cleanup CleanupReport }
type Options struct { HelperExecutable string; HelperDigest string; ScratchRoot string; Limits Limits; OwnerLiveness *os.File }
type CleanupReport struct { Status string; Jobs []ResourceState; Containers []ResourceState; Diagnostics []Diagnostic }
type ResourceState struct { ID string; Registered bool; Terminated bool; Reaped bool }
```

The ordinary Go representation of the existing semantic `Limits` record, including serialized names, is:

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

In that field order, defaults are exactly `600000, 10000, 4194304, 4194304, 16777216, 100000, 1, 536870912, 100000, 64`. Absolute shipped caps are:

| Limit | Absolute cap |
|---|---:|
| wall | 3,600,000 ms |
| cleanup | 30,000 ms |
| each output or event stream | 64 MiB |
| events | 1,000,000 |
| concurrent jobs | 4 |
| each bundle | 4 GiB |
| entries | 1,000,000 |
| depth | 128 |

Every value is positive and validated before duration or integer conversion. Explicit zero is invalid. Neither an omitted field nor an all-zero `Options.Limits` record silently defaults or clamps. `Diagnostic.Code` comes from the existing bounded catalog. `Diagnostic.Message` is bounded and repair-oriented, contains no secret, capability, or environment bytes, and conveys no authority.

`Capability`, `InternalIO`, `DockerRuntime`, broker frames, nonce and layout details, descriptor slots, and ownership state remain opaque and private except for the semantics specified here.

## 3. Root acquisition, inheritance, and work lifetime

### Root and descendant launch relationships

`Open(ctx, Options)` is the only root creator. It validates the exact helper closure, creates a private 0700 authority root plus launch-specific channels and challenge, directly starts the first broker as its actual child, and retains wait and cleanup ownership. `Open` completes only after the broker authenticates the root-owner bootstrap peer and acknowledges readiness. A failed bootstrap releases new resources and terminates and waits for that owned child within the already applicable shared cleanup budget. Marker text, a descriptor number, a claimed digest, a path, a PID, or an environment claim is not authentication.

A live broker creates guardian channels and verified attachments for descendants. A nested target obtains an authenticated inherited capability for that existing broker, and `Join` revalidates it. Missing or stale broker authority never promotes a guardian or nested target into a replacement root. Authentication is a bounded live exchange over private broker/root-installed channels independent of product standard I/O. It binds candidate and helper identity, root/scope/job/role identity, owner and operation authority, and the launch challenge.

### Acquisition contexts and handoff

Both inherited constructors require non-nil acquisition/probe contexts with finite deadlines. Validation precedence is:

1. nil context or no deadline: `INVALID_SCHEMA`;
2. already cancelled or expired: cancellation or `TIMEOUT`;
3. inspect bounded reserved intent;
4. authenticate present intent.

Authentication has one maximum of 5000 ms from constructor entry, bounded by the earlier probe deadline. Every read and retry shares that bound and the accepted 16 MiB control-record cap. Consumers check the error before the boolean. `InheritedInternalIO` returns zero/false/nil only for successful absence. Present-invalid intent returns zero/true/error and cannot fall through to ordinary parsing. `InheritedCapability` treats an absent required attachment as `CUSTODY_ERROR`. No error permits unscoped or root fallback.

Acquisition has one atomic terminal transition: `PENDING -> HANDED_OFF` or `PENDING -> FAILED`. Cancellation that wins before handoff releases acquired resources; success transfers exactly one usable handle. A successful handle retains only the independently authenticated original owner deadline, operation deadline, root and scope identities, and owner-liveness authority. Probe cancellation or expiry after handoff does not revoke work. Expiry of original authority or loss of liveness still enters cancellation and cleanup.

### Join and internal service ownership

`Join` consumes `Capability` on entry and uses a separate work context. Nil is `INVALID_SCHEMA`; prior cancellation or expiry fails before launch. A non-nil context without a deadline is permitted because authenticated parent and operation authority is already finite. The effective work deadline is the earliest authenticated owner deadline, operation deadline, and Join deadline if one is present. Join cancellation cancels the joined scope. Pre-handoff cancellation releases transport. Post-handoff cancellation closes or cancels the returned scope before any later launch.

`ServeInternal` consumes `InternalIO` on entry and owns it through execution and cleanup. It rejects zero, consumed, invalid, or wrong-role I/O with `handled=true` and `exitCode=1` before any product or target action. A valid internal caller that receives `handled=false` treats the result as protocol failure.

Opaque copies share one atomic ownership state: `UNUSED -> CONSUMED` or `UNUSED -> CLOSED`; exactly one concurrent consume or `Close` wins. A second consume fails. `Close` on zero, `CLOSED`, or `CONSUMED` is idempotent. Closing an unused handle is process-free local release only: close owned private descriptors, invalidate state, return local close errors, and never retry a possibly recycled descriptor. It never starts a process, network handshake, child wait, runtime probe, reauthentication, or new grace period.

### Scoped attachment

`Scope.Attach` preserves `Executable`, ordered `Args`, `Dir`, ordered declared `Env` excluding only broker-added reserved attachment representation, `RuntimeDigest`, and `DeadlineMS`, then adds reserved scoped state after sanitation. The exact binding covers every one of those fields. `Run` revalidates the command and attachment immediately before admission.

Removal, duplication, mutation, staleness, changed command data, post-Attach sanitation, or a scope/command/capability mismatch rejects before target start. A fresh attachment is allowed only while the scope remains OPEN and valid. `Run` cannot silently repair the attachment or fall back to unscoped execution.

## 4. Cumulative work and cleanup budgets

`Limits.wall_ms` is one cumulative elapsed deadline per requested milestone or invocation. It is never renewed per target, adapter, suite, state, retry, queue, scope, or phase. The owner ceiling is fixed at 14,400,000 ms from assurance-flow entry or the caller's earlier deadline. `Command.DeadlineMS` is the remaining duration computed at dispatch, not a new allowance after queueing.

Cancellation, any deadline, or the first terminal error forbids new verification work and enters FINAL cleanup with one shared monotonic grace: the current time plus the maximum selected `cleanup_ms`, capped at 30,000 ms; if no milestone is loaded, use 10,000 ms. All child, root, runtime, and resource cleanup shares that one grace. It cannot renew per resource or convert failure into success.

CLOSING rejects public `Run`, `Child`, `Attach`, and service starts. The broker may admit only fixed cleanup helpers for terminal-response settlement and exact-owned inspect/remove operations against the already bound daemon. Each uses the original remaining grace and its own registered guardian. Cleanup-only work cannot create, start, pull, provision, replay, run tests, or reopen general work. No process or runtime probe may start after final runtime release.

## 5. Docker runtime authority and closed descriptor

### Runtime API

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

### Descriptor topology and codec

The descriptor path class is `<fresh owned invocation root>/runtime/docker/<sha256-of-canonical-record>.json`. It is an owned, regular, non-symlink 0400 file, written once and retained through final cleanup. No local daemon identity descriptor is committed. The broker independently retains canonical bytes, digest, and live binding. File text, caller strings, labels, container IDs, and old JSON never construct authority.

All keys are mandatory. Duplicate or unknown keys, nulls, trailing data, and arbitrary settings are rejected:

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

The complete record is capped at 64 KiB. Scalar path, version, and ID fields are at most 4096 UTF-8 bytes. Decimal device and inode are `0` or nonzero-leading digits fitting `uint64`. Digests use exact lowercase grammar. Canonical encoding uses the shown field order, compact standard-Go JSON escaping, and one final LF. The digest is SHA-256 over those exact bytes and is not embedded recursively.

Endpoint, socket object, daemon, image, client closure, root, bytes, mode, and digest are validated before every mutation while the scope is live. Runtime `Close` performs process-free final byte, topology, and handle release and propagates errors.

### Capture, inheritance, and private client

Capture is identity capture, not image provisioning. It opens and snapshots the client closure and explicit local endpoint, queries actual client, daemon, and image identity through registered native helpers, constructs and registers the live handle, and returns only after success. Inherited retrieval accepts a descriptor digest only within the supplied authenticated root or child scope. It rejects absent, foreign-root, stale, or mismatched records. There is no ambient daemon discovery or import-as-authority.

The client launch is the exact executable followed by broker-owned `--config <fresh empty private config root>` and `--host unix://<captured endpoint>`, then the closed operation. The private HOME/config remains empty and is byte- and topology-checked. The environment contains only explicit validated PATH/runtime necessities plus `LANG=C`, `LC_ALL=C`, and `TZ=UTC`. Docker host, context, TLS, API, credential, and plugin overrides; caller argv prefixes; a shell; custom sockets; and user CLI configuration are not forwarded. The user's socket is never deleted, chmodded, replaced, or owned.

### Once-only endpoint preparation and transport

The prohibition on ambient daemon discovery applies during capture inheritance, mutation, run, replay, and cleanup. It permits only one owned, bounded preparation step:

- The contributor/integration lane exposes `--docker-endpoint <absolute local socket>` and passes that concrete endpoint to `CaptureDockerRuntime`. An explicitly supplied endpoint wins. Otherwise, preflight performs its existing Docker-context inspection exactly once during owned bounded provisioning, resolves and freezes one concrete local Unix socket endpoint, and passes the same explicit value to both the lane and checker command.
- The standalone checker exposes the same `--docker-endpoint <absolute local socket>`. Without an explicit endpoint, it may perform its existing context resolution exactly once during owned bounded preparation, freeze the resulting concrete descriptor, and never consult ambient Docker context again during run or replay.
- Unsupported remote endpoints fail before replay. A context name, environment variable, caller string, old descriptor file, or later re-resolution is never runtime authority. Run and replay use the same captured live runtime and daemon binding and the same private client configuration.
- A separately owned provisioning step may pull only the exact authorized pinned image through registered custody. Capture then freezes the runtime descriptor before native selection or replay. `CaptureDockerRuntime` itself does not provision or pull, and checker run and replay never pull.

These endpoint flags and lifecycle rules are required behavior. Their presence in this contract does not state that a current binary implements them or that publication performed endpoint resolution, provisioning, or capture.

## 6. Closed contributor service and conservative daemon lifecycle

### Contributor API and profile

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

`Run` is synchronous and cleans before return. There is no live lease, Start handle, arbitrary argv, or public `RegisterContainer`. The initial `Program` registry is closed:

| Program | Exact command |
|---|---|
| `python-version` | `python3 --version` |
| `print-42` | `python3 -c "print(6*7)"` |
| `sleep-300` | `python3 -c "import time; time.sleep(300)"` |
| `sleep-90` | `python3 -c "import time; time.sleep(90)"` |
| `sleep-900` | `python3 -c "import time; time.sleep(900)"` |

The current contributor capture binds the explicitly provisioned immutable image `python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc` on declared linux/amd64. Actual host, daemon, and image identities remain separately reported. The fixed profile uses that image/platform with network=none, a read-only root, no mounts, ports, devices, capability additions, host namespaces, restart, or auto-remove; memory 134217728 bytes; NanoCPUs 500000000; and PidsLimit 32. It accepts no overrides.

### Observations and exact admission order

Observations are read-only data, never custody or removal authority. Sequence starts at 1 and increments. Phases are exactly `registered`, `running`, `completed`, `removed`, and `ambiguous`; at most eight records are emitted under existing event and output caps. A nil channel requests no observations. A closed channel, overflow, or inability to deliver within the remaining deadline fails and cleans without panic, leak, or success. `registered`, `running`, and `removed` follow actual validation or inspection.

Read-only exact-ID inspection validates the same daemon and returns only the fixed safe fields. An unknown ID yields `Exists=false` only after authoritative same-daemon absence; daemon error is not absence. There is no broad name search or secret exposure.

For each daemon job, the broker creates a fresh 32-byte operation nonce and the name `machinery-ci-` plus lower64hex. Before helper dispatch it records a root/scope/job/descriptor/daemon/config/name-bound intent. Labels are `dev.machinery.integration-run=<broker root identity>`, `dev.machinery.integration-operation=<nonce>`, and `dev.machinery.integration-config=<config digest>`; labels are locators, never authority.

The broker registers an actual guardian before releasing a create-only helper, dispatches exactly one create request, and never retries an ambiguous create. After acknowledged creation it validates the full canonical ID, name, nonce, config, image, and daemon plus Created/not-started state; registers container ownership; rechecks OPEN state and budget; and only then starts the exact registered ID. It captures real wait, log, and inspect status under limits and keeps target error or exit distinct from cleanup.

### Unresolved creation and conservative cleanup

A dispatched create without a terminal server-backed response is `CREATE_UNRESOLVED`. Client death, timeout, pipe closure, an empty label query, one inspect-not-found, or removal of a matching appeared object cannot prove that no later creation will occur. Within the original grace the broker retains and waits for the original request and inspects the exact reserved name.

Only an acknowledged create followed by validated removal, or authoritative terminal precreation rejection, settles creation. Exhausted grace, lost or changed daemon identity, or ambiguous outstanding create/start returns `CUSTODY_ERROR`, `cleanup-failed`, and bounded owned-resource diagnostics. It never returns success, retries, starts a background sweeper, performs broad label/name deletion or prune, removes images or volumes, or recovers authority from a historic PID. Container `ResourceState.Reaped` means verified daemon removal.

## 7. Closed real-checker profile and execution binding

### Checker API and sandbox

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

`Phase` is exactly `run` or `verify` and selects an already registry-bound argv vector. No arbitrary extra engine args, environment, mounts, or resource flags are accepted. The caller retains `WorkRoot` through result and cleanup. Canonical private `WorkPath`/`WorkRoot` identity, read-only runtime-input topology and digest, runtime closure, registry image/platform, command vectors, and actual config are validated before launch and after execution.

The checker sandbox fixes pull=never, network=none, a read-only container root, cap-drop ALL, no-new-privileges, and workdir `/work`. Only `WorkPath` is mounted writable at `/work`; runtime inputs are read-only at `/checker`; `/tmp` is tmpfs `rw,noexec,nosuid,size=67108864`. The fixed environment is `HOME=/work/home`, `LANG=C`, `LC_ALL=C`, `TMPDIR=/tmp`, `TZ=UTC`, `PYTHONNOUSERSITE=1`, `PYTHONSAFEPATH=1`, empty `PYTHONPATH`, and `MACHINERY_CHECKER_RUNTIME_CLOSURE=<validated closure>`. UID and GID are validated. Initial limits are memory 134217728 bytes, NanoCPUs 500000000, and PidsLimit 32, with no overrides.

The registry binds `python3 /checker/adapter.py` for run and a `python3 -c` verification pass. The profile rejects `{design}` and preserves canonical token/config/manifest projection; exact `/work/evidence.json` and `/work/committed/evidence.json` replay; file-only evidence output; evidence and trace comparisons; closure revalidation; design snapshot `CheckUnchanged` and `Release`; and final cleanup before successful output or release. Runtime-specific exact image identities belong to captured contributor inventories and are not a timeless public promise.

### Real command context, scope, runtime, run, and replay

The normal checker command propagates its real cancellation/deadline context, authenticated scope, and captured Docker runtime through checker orchestration, per-checker validation, the primary run, and committed-evidence replay.

Required scoped execution:

- uses the real command context, never `context.Background` or a replacement owner context;
- retains the same authenticated owner/root and captured runtime across run and replay, with every phase child bounded by that same owner and the already cumulative deadlines;
- performs deterministic checker environment and sandbox sanitation first, then applies the scoped attachment, so sanitation cannot discard or rewrite custody and no ambient attachment is trusted;
- rejects nil, stale, foreign, cancelled, closed, mismatched, or unscoped capability or runtime before target start;
- may retain service-free compatibility wrappers for separately bounded tests, but never calls or falls back to an unscoped compatibility wrapper on the required scoped command path for run or replay;
- routes both real checker phases through the closed `RunCheckerDocker` profile and its registry, image, platform, input, argv, and work-root bindings, never through a direct generic `docker run --rm` client; and
- completes exact container and helper cleanup and validates absence before successful checker output, evidence publication, work-root release, or runtime release.

Target result and cleanup result remain distinct. Cleanup failure prevents success.

## 8. Required proof method and honest delivery state

### Live-first positive and adversarial evidence

Custody evidence is live-first. An independently owned harness observes the exact announced resource live on the same daemon; validates daemon, name, root or owner, operation, config, image identity, and `Running=true`; and only then releases the real trigger.

Evidence covers actual assertion failure while `Run` is active, host standard-output and standard-error overflow while the resource is active, cumulative timeout, real SIGINT and SIGTERM, owner-liveness loss, normal completion, registration failure, early intermediate exit, nested join, stale/forged/cross-scope/malformed/closed/reused handles, acquisition and admission races, and unresolved create or start. It proves exact owned absence and guardian/helper termination before success while separately owned foreign process and container controls survive every cleanup class. Container-output overflow and resource exhaustion use the real checker profile, not the no-output sleep fixture.

### Calibration and distinct implementation roles

Calibration freezes tests, helpers, fixtures, config, closure, budgets, and every expected leaf before replay. A deliberately unsafe operative reference and its safe-control counterpart differ only in authorized implementation bytes; the same assertion fails and passes while reaching the same real fixture. Separate unsafe/safe pairs are used where invariants differ.

A pre-feature or setup-failing source state is retained honestly but is not descendant-cleanup RED. The unsafe reference, safe reference control, pre-feature state, and production candidate are distinct roles. The production candidate is a later state after independent RED approval. A reference safe control never substitutes for production GREEN or native delivery. No environment behavior toggle, test mutation, generic nonzero result, document, receipt, or source-keyword check proves custody.

### Historical migration and native host matrices

Migration creates a new current reviewed revision while preserving every historical obligation, exact old source/test/control identity, failed record, and legacy regression as history. Every prior obligation maps to a current executed assertion that preserves or strengthens its outcome; renamed or restructured cases require explicit justification. New custody/runtime leaves are inventoried separately instead of inflating historical counts. Private issue history and workflow chronology are not part of the contract.

Required host matrices are native Darwin arm64 and native Linux amd64 for the same candidate and frozen inventories. Host, daemon, and image architectures are reported separately. An amd64 image on an arm64 daemon or host is not Linux amd64 host proof.

Implementation, real custody execution, cleanup results, production GREEN, and both native host matrices remain required and unproved until separately executed and accepted. Documentation, static checks, source-keyword checks, and reference controls are not native evidence. This document supplies no infrastructure, provisioning, process, container, installation, or execution authority.
