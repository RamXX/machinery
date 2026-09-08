# Changelog

Notable user-facing changes to machinery. Release notes for generated-output and proof-scope
changes follow the discipline in [docs/release-notes.md](docs/release-notes.md); entries move
under their version heading when a release is cut.

## [Unreleased]

### Fixed

- **`machinery update` accepts the plugin inventory the current Claude Code writes.** The Claude
  Code plugin refresh read each `claude plugin list --json` entry as a closed record and rejected
  the fields Claude Code now adds (`enabled`, `installPath`, `installedAt`, `lastUpdated`,
  `mcpServers`, `version`), so on 0.7.0 every Claude Code user saw `Claude Code plugin inventory
  was not understood: Claude plugin entry 0 has unknown fields [...]`, the update returned an
  error, and the plugin never refreshed (MAC-9zpf). machinery consumes `id` and `scope` only;
  the entry is now an open record for the fields it does not read, and it still fails closed
  when a consumed field is absent or malformed. The exact inventory Claude Code wrote on
  2026-09-08 is pinned as a fixture. The Codex inventory reader is unchanged: it is still a
  closed record, because the fields it consumes are the whole record.
- **The first `machinery update` over a 0.6.11 install no longer leaves the receipt describing
  the previous release.** After `machinery update --version v0.7.0` on a 0.6.11 install, doctor
  reported the installed skill and the build-writer role as invalid (`artifact digest is
  sha256:e2262eda..., want receipt-bound sha256:cc1d8dd4...`) although every file on disk was
  byte-identical to the release, and a second update converged (MAC-d3ov). Cause: 0.7.0 moved
  receipt finalization from the placement child to the update parent (MAC-2u36), but on a
  cross-version update the parent is the old binary and the child is the new one. A 0.6.11
  parent never finalizes, and the 0.7.0 child left the receipt to a parent that would not write
  it, so the committed receipt still carried the 0.6.11 digests. The parent now announces to its
  placement children that it publishes the receipt; a child that hears no announcement (a
  parent older than 0.7.1) records its own placement, retaining the recorded digest of any
  artifact a sibling child still has to place. The announcement rides the environment rather
  than the lock capability payload because every child before 0.7.1 compares that payload byte
  for byte, and a downgrade through `machinery update` runs exactly such a child. A test runs a
  real placement child under a prepared parent transaction both ways: without the announcement
  the receipt is rewritten and doctor's receipt-bound check passes for the placed tree; with it
  the receipt is untouched. Reproduced and verified against the published v0.6.11 and v0.7.0
  assets in a fresh `HOME`.

### Added

- **`machinery doctor --repair` compacts the governance hook state store.** The per-user store the
  hooks keep their gate obligations in now has a retention policy applied on every arming write: at
  most 8 route snapshots per project root, and a store-wide ceiling of 512 entries, an eighth of the
  4096-entry limit past which every bounded inventory fails and every governed shell and write tool
  on the machine is blocked. Compaction reclaims only whole generations whose recorded project root
  no longer exists on disk, so it can shrink the store but never discharge a live obligation.
  `machinery doctor` reports the store's path, entry count and both bounds; `machinery doctor
  --repair` compacts it. Both work when the store is already at or above the limit, which is the
  only moment either matters. See the plugin guide's durable-state section for the policy and its
  residuals.

### Fixed

- **The hook repairs its own state instead of bricking every repository.** A store filled past the
  fail-closed limit used to deny every governed tool call until a human pruned it by hand. The first
  inventory that hits the limit now compacts once under a repair ceiling and retries. This covers
  ledgers written by 0.7.1 and later: a ledger written by 0.7.0 or older carries no project root, so
  nothing can tell its obligation from a live one and compaction retains it. A store already over
  the limit at upgrade time therefore needs a one-time manual removal of its `<digest>.state` files,
  never of the store directory, whose initialization marker distinguishes a first run from a lost
  store.

### Changed

- **`machinery doctor` exits nonzero on an unusable hook state store.** The new store report fails
  the command when the store is at or above its 4096-entry limit, or when the store path cannot be
  resolved or inspected (a container with no resolvable HOME, for instance). A store above the
  retention ceiling but below the limit reports and passes, so the common state does not newly fail.
  Any CI job or Makefile target that treats `machinery doctor` as a gate has a new failure source.

### Compatibility

- **The hook state ledger records its project root, which an older binary rejects.** This is what
  lets retention prove an obligation is dead rather than guessing. A binary older than 0.7.1 reads a
  ledger written by 0.7.1 as noncanonical and fails closed on it, so a downgrade, or a second older
  binary sharing the same user home, blocks governance for the affected projects until their
  `<digest>.state` files are removed. Remove the files, never the store directory. Reading in the
  other direction is unaffected: 0.7.1 reads a pre-upgrade ledger normally and only declines to
  reclaim it.

## [0.7.0] - 2026-09-08

### Fixed

- **The partial-witness journal recovery proof no longer cuts a record it did not write.** The
  proof appends the first N bytes of a witness record to a formal transaction journal, for every
  N, and requires recovery to derive the rest. It took the number of cuts from a record seeded in
  one directory but appended the bytes of a record seeded fresh in another. A native witness is
  `unix:<device>:<inode>:<creation sec>:<creation nsec>` in hex, so the encoded record length moves
  with the hex width of the inode and of the creation-time nanoseconds: over 200 seeds on one host
  the record came out 83, 84, or 85 bytes. When the record under test was the shorter one, the last
  cuts sliced past its end into the spare capacity of the encoding buffer, appending a complete
  record plus stray bytes, and the reader rejected those bytes as malformed trailing data, which is
  what they were. It surfaced as `new/byte-082` on a hosted macOS runner and reproduced here at
  `new/byte-084` in 1 of 3 runs. Each cut is now bounded by the record it appends, and two pinned
  cases hold the boundary steady on any host: a sweep with the narrowest and the widest
  creation-time fields a witness can carry, and a check that a byte which cannot begin the next
  expected record stays rejected. The journal reader is unchanged: every true prefix of a valid
  record was already recovered, and the ABA and tamper rejections are untouched.
- **A delivered close report is no longer reported as a lost control channel.** The broker answers
  the root close and then exits: it writes the report, leaves the root channel loop, retires,
  writes its ledger and closes the control connection, all within a few milliseconds. On the
  caller's side one demux goroutine reads that reply into a buffered channel and then, a moment
  later, reads the end of the channel and closes the end-of-channel signal. A caller that had not
  yet reached its own select when both became ready got whichever of the two the runtime picked,
  so a `cleaned` report already sitting in the buffer was announced as `STALE_CAPABILITY: root:
  broker channel closed` with a synthesised `cleanup-failed` report carrying no job diagnostics.
  It only ever hit the root close, since no other close ends the broker, and only on a host slow
  enough to deschedule the caller between sending the close and waiting for its answer: three
  adapter suites failed that way in the hosted non-race macOS job on a loaded three-core runner.
  The caller now drains an already-delivered reply before it reports the end of the channel, which
  is a settled read rather than a second race: the single demux goroutine cannot signal the end of
  the channel before it has delivered every reply it read. No budget or cap changed.
- **A killed process group is no longer read as a survivor while its members await reaping.** A
  job's retirement signalled the group, reaped the guardian, and then probed the group once. The
  members of a job are children of that guardian, so at the instant it is reaped they are orphans
  the platform's init has yet to reap, and on Linux a killed-but-unreaped member still answers the
  group probe (macOS skips those members, which is why the condition never showed there). The
  single probe read them as a survivor and marked a job that terminated exactly as asked
  unterminated: a cancelled required-lane run published `cleanup-failed` with `Terminated:false
  Reaped:true` and no diagnostic naming any survivor, in 3 of 40 runs on a two-core Linux host.
  The retirement now polls the group until it drains, inside the reap window that already bounds
  that stage, so nothing waits longer than before and a group that truly refuses to die is still
  reported unterminated. No budget or cap changed.
- **The custody provisioning test no longer races the pinned Java identity probe it observes.**
  The observation read three process-table snapshots: one to decide a nested pinned JVM was live,
  a second to name its pid, a third to walk its ancestry. Provisioning runs a pinned
  `java -XshowSettings:properties -version` identity probe under the private cache, which
  satisfies the first snapshot and exits within a fraction of a second, so the second found
  nothing and the walk reached nothing: `provisioning JVM process 0 does not belong to the lane
  process tree`, in 6 of 40 runs on a two-core Linux host and in the hosted required lane. All
  three questions are now answered from one snapshot, and the ancestry is walked in the same
  snapshot the process was observed live in. Test-only: the observation is unchanged in what it
  demands, and no budget or cap changed.
- **A failed integration lane says what its native runner reported, and CI keeps the evidence.**
  The lane accounted a failed suite from the runner's event stream and printed only that
  accounting (`native test ... fail failed/skip`, `required execution incomplete: selected 1
  started 1 passed 0`), while the event stream itself and the report went to a private temporary
  directory no workflow collects. A failure only a hosted runner reproduces was undiagnosable
  from the log it left behind. A failed suite execution now replays the bounded tail of that
  runner's own stdout and stderr next to the accounting: the last 200 lines, at most 64 KiB of
  them, each block delimited and naming the suite. `--report-dir`, defaulted from
  `MACHINERY_INTEGRATION_REPORT_DIR` because the workflow and preflight invocations of the lane
  are pinned to an exact argument string, names the directory the report and its retained event
  streams are written to; an unset variable keeps the private temporary directory. `ci.yml` and
  `formal.yml` name that directory and upload it as an artifact when the job fails, with a
  seven-day retention. No budget or cap changed.
- **A child scope's control channel is no longer collected while its descriptor is in flight.**
  The broker created the socket pair for a new child scope, sent one end to the caller, and closed
  its own copy the moment the send returned. For as long as the descriptor was in flight its only
  reference was the copy inside that message, which is what the platform's in-flight descriptor
  collector treats as unreachable: it flushes the receive side of the socket the descriptor names.
  The caller received a live, connected channel it could still send on and every read of which
  returned end of file, so its next request failed `STALE_CAPABILITY: scope-...: broker channel
  closed` while the broker was alive, healthy, and still serving that very channel. On a loaded
  host this took out roughly one run in six of a 200-retirement custody run, and in a
  `verify-formal` pass it ends the run. The sender now keeps its own reference to a handed-over
  descriptor for as long as it is in flight and releases it once the owner proves it holds it,
  with the first record it sends; the capability channel a joined child is launched with follows
  the same rule, and a control record's descriptor rights are held alive across the send that
  transfers them. No budget or cap changed.
- **A custody suite's owner-loss handoff says why it never arrived.** The owner helper polled for
  its grandchild's pid for a fixed 10 s and then wrote whatever it had read, so a chain that had
  not come up inside that window wrote an empty file, which the reader ignores; the reader then
  waited out its own separate 20 s and reported a file that never appeared. Neither number was
  derived from the 60 s wall the helper declares, and neither the helper's own launch failure nor
  its log reached the message. The helper now polls under the wall it declares, keeping back a
  fixed slack to report with, and always writes a record naming what happened; the reader derives
  its wait from that same wall and prints the helper's log when the record is missing or is not a
  pid. Test-only: no product budget or shipped cap changed.
- **A custody challenge test no longer races the broker it built from a hook it replaces.** The
  challenge cases run the broker in process and replace a package-level hook to build a second
  broker from the replacement. The broker reads those hooks once, when it is built, and everything
  the test observes of the first broker afterwards travels through a kernel socket, which gives
  the race detector no ordering: `go test -race -count=5 ./internal/processscope` reported
  `race detected during execution of test` on an idle host within a second, while a single-count
  sweep never did. The harness now joins the broker goroutine it started as part of its cleanup,
  and each case retires and joins the broker built from the previous hook before replacing it.
  Test-only: no product behaviour changed.
- **A scope close joins a retirement already claimed instead of collecting nothing.** The join
  0.7.0 gave `retireJob` was unreachable for the case it was written for: both collectors filtered
  the job table on jobs no retirement had claimed, so a job claimed before the snapshot was never
  passed to the join at all. A `Run` cancelled by its caller is retired on its own goroutine, and
  the close that follows it arrived while that retirement sat between marking the job terminated
  and reaping its guardian, so the close published that half-written state:
  `cleanup-failed` with `Terminated:true Reaped:false` and no diagnostic, on a job that went on to
  reap cleanly. `go test -race -count=5 ./internal/processscope` reported it on an idle host. Both
  collectors now pass every job in scope through the join, which returns at once for a retirement
  that has already settled, and the claim itself is taken and read under the broker's own lock. No
  budget or cap changed.
- **A scope close reports the settled state of a job already being retired.** A job cancelled by
  its caller is retired on its own goroutine, and the guardian may report the target's death as an
  ordinary exit rather than a terminal drain, so `Run` returns while that retirement is still
  between marking the job terminated and reaping its guardian. The close that followed collected
  only jobs no retirement had claimed, skipped that one, and published whatever half-written state
  it found: `cleanup-failed` with `Terminated:true Reaped:false` and no diagnostic, on a job that
  went on to terminate and reap cleanly. A close now joins a retirement already in flight instead
  of skipping it, bounded by that retirement's own budget, so a job that truly refuses to die is
  still reported unreaped with its `TIMEOUT` diagnostic. No shipped cap changed: the per-job
  cleanup budget, the 30,000 ms cleanup cap and the 2 s hard reap window are unchanged, and
  nothing runs longer than before, only the report waits for a settled answer.
- **A guarded job that finishes before its caller can wait for it is no longer reported as a
  timeout.** The control channel registered a job's result channel when the broker's started frame
  arrived and retired that registration on delivery of the result, while `Run` claimed the channel
  only after the started reply reached it. One reader handles both frames in order, so a job short
  enough to finish first, an identity probe or any small guarded command, had its result delivered
  and its registration retired before its own caller claimed it: `Run` then registered a second,
  empty channel, waited out its whole deadline on it, and reported `TIMEOUT: job terminated by
  scope cancellation` on a job that had already exited 0. The registration now belongs to the
  caller for the whole call, so a result that arrives first is waiting in the channel the caller
  claims, and a duplicate result for an already-reported job is dropped instead of stalling the
  reader every other frame on that connection depends on. No budget changed.
- **A custody root grants the wall it declares, not the window its owner was opened with.**
  `processscope.Open` bounds a scope's cumulative wall by the earliest of the declared `wall_ms`
  and the open context's own deadline, so an open context sized as a bootstrap window quietly
  becomes the real root wall. Every custody root the suites open now derives its open context from
  the wall it declares, the correction 0.7.0 already applied to the contributor lane and to
  `verify-formal`'s candidate scope. A 7-spec TLC portfolio measured at 85 s under the race
  detector on 2 vCPU used to run out of a 60 s bootstrap window and fail every remaining spec with
  `BUDGET_EXHAUSTED: root: inherited wall deadline has passed` against a declared 2,400,000 ms
  wall. No cap changed: the declared budgets are what they always were.
- **A finished Elixir suite whose ExUnit teardown crashes is no longer reported as a build
  error.** The embedded `elixir-exunit/v1` formatter used to stop itself while handling ExUnit's
  final `suite_finished` event, racing the `ExUnit.EventManager.stop/1` call the runner makes
  immediately afterwards: when the formatter supervisor had not yet processed the exit signal, the
  teardown exited `{:noproc, {GenServer, :stop, [pid, :normal, :infinity]}}` and killed the runner
  after the suite had already executed, intermittently on a loaded host. The formatter now closes
  its event stream in place and stays alive, so ExUnit stops it exactly once. Independently, a mix
  abort that happens after a complete, reconciled native run is classified as
  `UNEXPECTED_FAILURE` naming the reconciled outcome, with the witnessed assertion outcomes
  preserved on the execution record, instead of a `BUILD_ERROR` claiming a suite that compiled and
  ran failed to build.
- **A matrix's prose is narrative again, not a malformed `CLAUSES` declaration.** A declaration is
  a `CLAUSES{...}` group on a row of the named-unit contract table, which is the only shape 0.6.11
  read. An invariant-coverage bullet quoting a guard's vocabulary, an enumeration wrapped across
  lines, a contract cell that mentions "the CLAUSES declaration here" or describes a RETIRED site,
  and a row for a non-guard unit are all accepted as they were, instead of being reported as
  malformed declarations whose "guard name" is a sentence fragment. Ambiguity is still rejected: a
  row carrying two `CLAUSES{...}` groups, or a `RETIRED{...}` group the declaration does not carry,
  names no single vocabulary and fails.
- **A declared guard no oracle governs owes nothing again.** 0.6.11 resolved a declaration's
  falsifying-clause obligations across every committed oracle, so a machine could declare the
  clauses of a guard that sits on its creation edge rather than on a guarded transition, and owe
  no tests. That case is silent once more. The ownership rule this release added stands: a
  declaration whose rows only a SIBLING machine's oracle could supply is still an error, and the
  diagnostic now names the sibling.
- **Consumer `READS` members keep their ubiquitous-language grammar.** A member is any trimmed,
  non-empty text, as it was in 0.6.11: `READS{Order.id, occurrence time, PAIR KEY}` declares three
  fields. Empty members and duplicates are still rejected, and an empty member is now reported as
  an empty member rather than as an "empty or malformed READS field" naming text that is neither.
- **`READS` is an English verb again outside a declaration.** A consumer declaration is a
  `READS{...}` group, as it was in 0.6.11. A residual or contract cell narrating that a machine
  "only READS those rows" off an event another layer produces is prose: it is no longer collected
  as a declaration and then reported as an incomplete one. A row that does carry a group is held
  to exactly one complete declaration, unchanged: a `READS{...}` wrapped across two lines is an
  incomplete declaration and still says so, because the group is what a row declares. Join the
  lines.
- **Installer reruns converge on the recorded installation.** The one-line installer over an
  existing install now refreshes the complete recorded home/native-target/plugin plan (each
  group's copy/symlink mode preserved) instead of the default homes only, identical to
  `machinery update` without selectors. Safely edited or missing owned artifacts are repaired to
  exact current-release content; corrupt, unsupported, or unsafe receipts, unsafe artifact or
  parent substitutions, and uncertain plugin ownership fail closed with diagnostics (including
  under `--skip-plugins`), never silently falling back to the defaults. A late failure rolls the
  whole direct transaction back to the exact pre-run state (a previously absent artifact is
  restored to absence), and receipt publication is validated and parent-owned; concurrent foreign
  changes are refused, not overwritten.
- **Host plugin refresh failures are reported as failures.** A missing host CLI, a failed or
  misunderstood plugin inventory, or a failed refresh is a returned error naming the exact retry;
  the direct update stays committed and the obligation is recorded for the next update, instead
  of a warning over a claimed-successful update. Host-owned plugin caches remain outside
  machinery's rollback.
- **Oracle coverage (`Gt`) credits discovery only.** Static test-reference analysis is labeled
  "static discovery; tests not executed", and bypass shapes (commented-out or unused tests,
  uncalled parsers and helpers, evidence living outside the test) are rejected instead of
  passing.
- **OpenCode governance subprocesses are bounded and validated.** A deadline plus a capped output
  capture terminates runaway children, and unrecognized, truncated, or combined-protocol hook
  responses block instead of best-effort parsing.
- **Unmodeled saga compensation routes are rejected** in refine and compose instead of proving
  green against routes no model defines.
- **Go-crm boundary mapping** covers `internal/testoracle`, and FSM conformance binds to committed
  oracle expectations.
- **Guarded jobs keep their declared wall budget.** The required integration lane and
  `verify-formal` opened their custody scopes through a 60-second bootstrap context that silently
  clamped the declared 3,600,000 ms wall (and with it the formal-provision, image-pull, and suite
  budgets), so a cold JVM/TLC provision on a loaded host died as a scope timeout. The open context
  now derives from the declared wall itself; caps are unchanged and an overrunning job still dies.
- **Large designs verify again.** `machinery verify-formal` failed on designs big enough to
  outgrow three single-shot assumptions in native custody, reporting cleanly terminated and
  reaped engine runs as `cleanup-failed` and then failing every remaining check with
  `STALE_CAPABILITY`. The cleanup grace is now accounted per retirement instead of as one
  absolute instant shared by every job a run ever launches; the broker decides whether a close
  ends it while holding its own lock, instead of a concurrent read that could kill it outright;
  and a control record larger than the socket send buffer, such as a cleanup report naming every
  retired job, is written whole instead of abandoned as a short write. Every shipped budget keeps
  its current value, cleanup time stays bounded, and a job that really does overrun its budget is
  still reported.
- **Custody results are awaited for the job's real deadline.** The broker's terminal-result wait
  is bounded by the effective deadline plus the cleanup grace instead of a fixed window, so a
  legitimately long job on a loaded host is no longer abandoned as a custody error, and a result
  frame arriving while the job channel was still being registered is no longer lost.
- **Assurance store writers release staging in ownership order.** A committed writer no longer
  fails with a custody error (or deletes a fresh live token) when a concurrent writer reacquires
  the staging lock the instant it drops.
- **Same-inode content ABA is rejected on Linux.** Formal transaction journals, directory
  inventories, and the Java runtime closure add a kernel mutation-event witness to their identity
  checks, so a rewrite that leaves coarse inode timestamps equal is still detected.

### Added

- **`machinery recover`**: pass a design directory for a read-only report of an interrupted
  publication: expected outputs with content/mode status, journal and residue locations,
  live-writer status, and the safe recovery decision. `--apply` finalizes only a fully
  revalidated interrupted publication; refusals preserve all evidence.
- **Required integration lane**: `go run ./scripts/integration-lane --lane required` executes
  every infrastructure-dependent suite (Docker-backed checker lifecycle, publication recovery,
  native custody, adapter governance, and the documented checker registry/bind-path example) with
  exact test inventories, pinned runtimes, and real provisioning/teardown accounting. Missing
  infrastructure fails the lane with a diagnostic; nothing is skipped.
- **Native custody for machinery's own subprocesses.** Formal verification and tooling
  subprocesses run inside an owned process-custody scope with registration before launch, bounded
  budgets, and terminal-kill-before-reap cleanup. Limits: cleanup failures are reported,
  never concealed, and custody is process-ownership containment, not a defense against a
  hostile host.
- **Checker container budgets and lifetime ownership.** Every checker container gets closed
  finite budgets (128 MiB memory, half a CPU, a 32-process pid limit), a registered identity,
  and force-removal on every exit path; an output breach aborts and cleans up instead of
  buffering.
- **Standalone contract documents** for native custody and executable test assurance
  (`docs/native-custody-contract.md`, `docs/test-assurance-contract.md`).
- **Executable test assurance: the version-1 substrate and four native test adapters, with no CLI
  surface yet.** This release lands the mechanism, not a consumer command. There is no
  `machinery tdd` command, no `machinery check --store` or `--assurance strict`, and no gate that
  reads `design/assurance/`: a design that commits `plan.json` today sees no diagnostic and no
  change in its finding count from `machinery check`. What ships is the closed version-1 document
  grammar and its validation, the authoritative obligation inventory over a held design snapshot
  (BUILD milestones, committed oracle rows, guard-clause ids, invariant ids, declared runtime
  obligations), and closed adapters for Go (`go test`, Go 1.27.1), TypeScript (`tsc` plus
  `node --test`, Node 26.8.1 / TypeScript 7.0.2), Python (`python -m unittest`, CPython 3.14.7),
  and Elixir (`mix test`, Elixir 1.20.4 / OTP 29). The one place those adapters execute is the
  required integration lane, which verifies every adapter runtime against the exact pins and runs
  frozen per-language probe and native-conformance fixtures under process custody with exact
  per-case accounting; a missing or mismatched runtime fails the lane. Inside that lane each
  registered assertion must be a typed helper call at its frozen line, its outcome is
  witness-bound to that call site, and skips, expected failures, unwitnessed failures, and late
  source mutation are rejected. `docs/test-assurance-contract.md` carries an
  "Implementation status in 0.7.0" section naming shipped versus target surfaces; its section 7
  CLI grammar is the target contract. Until it lands, bind tests to oracle rows with the
  `Gt-tests` stable-id discipline.
- **Replay inputs are retained outside governed sources** (library surface; no command exposes it
  yet). Capture publishes the exact subject, dependency, frozen, and design payload inputs as
  content-addressed blobs in a private assurance store (symlink escapes, hardlink aliases, and
  special files are rejected; the control namespace, `.git`, and `attestations.yaml` are
  excluded), with a typed bundle encoding, a validated export/import archive, and a read-only
  status report that never performs a replay.
- **Reviewed assurance revisions are registered explicitly** (library surface; no command exposes
  it yet). Registration validates lineage (revision 1, or the predecessor plus one with the same
  key), archives the exact control bytes, stages the successor, and advances the store head by
  compare-and-advance under a bounded deadline; drafts never discharge registered requirements,
  and a registered deletion or an untargeted registered mismatch blocks. It commits authored data
  only and returns registered-not-executed; it never claims execution or replay.
- **Release publication is gated on exact-commit verification.** `docs/release-policy.json` is
  the single policy source: a release requires successful `ci`, `formal`, and `security` runs on
  the exact release commit (every required job, including the required integration lane), a tag
  matching the numeric version pattern, and main ancestry; scheduled runs never count as
  evidence. The release workflow runs `scripts/release-policy` against that commit before
  building and again before publishing, the policy is tested by `go test ./scripts/release-policy`,
  and the CI, formal, and security workflows run with read-only permissions. The branch-protection
  payload is documented but has not been applied remotely.
- **windows/amd64 release artifacts.** Each release now publishes a cross-compiled
  `machinery-windows-amd64` binary and `machinery_<version>_windows_amd64.tar.gz` next to the
  Linux and macOS assets, and the publish inventory is accounted exactly across build, manifest,
  and publish. The one-line installer and `machinery update` still refuse Windows, and native
  Windows runtime guarantees (process custody, formal verification, assurance lanes) are not
  claimed.

### Changed

- **Guard clause declarations bind to their owning machine.** `Alpha.matrix.md` binds only
  `Alpha.machine.json` and `Alpha.oracle.md`, so two machines may use one guard name without
  either arming or satisfying the other's falsifying-clause obligation. Upgrade impact: a
  declaration whose oracle rows only another machine could supply is now an error, and one row
  declares one vocabulary (a second `CLAUSES{...}` group on the row, or a `RETIRED{...}` group the
  declaration does not carry, is rejected). Move the declaration to the matrix of the machine
  whose oracle governs the guard, and keep one group per row.
- **Attestation evidence v2 kinds (`plan`/`current`/`historical`).** Example attestations
  migrated to explicit kinds: a `current` claim requires an implementation subject, legacy
  design-only covers are rejected with migration guidance, and implementation-subject changes
  invalidate stale reviews so a review binds the exact scope it attests.
- **Example attestations carry explicit v2 kinds.** All eight bundled examples declare a kind on
  every row: the seven design-only examples attest their behavioral claims as plans, and go-crm
  carries a current implementation review over `examples/go-crm/impl`. `machinery check` on a
  design-only example therefore reports its missing-current warnings and exits 0.
- **Preflight asserts the per-example attestation state.** `scripts/preflight.sh` phase 11 keeps
  go-crm strict under `--warnings-as-errors --impl --complete` and requires each design-only
  example to pass plain `check` with exactly the plan-only warning set derived from its committed
  `attestations.yaml`; any other warning, error, or drift fails. The race-sweep per-package alarm
  moves to 30 minutes in preflight and its CI mirror.
- **Nightly runs use full history.** The nightly golden job checks out the full history so the Ga
  acceptance gate can prove reviewed commits are ancestors of HEAD, and the nightly workflow runs
  the required integration lane.
- **Frozen-test identity requires exact bytes.** Formatting-only rewrites are no longer treated
  as the same frozen test; evidence replay requires exact identity with no whitespace exemption.
- **Event-contract consumer `reads` bind per edge.** A consumer that legitimately reads more than
  its siblings declares its exact read set on its own edge instead of sharing the intersection.
- **Regeneration advice no longer accepts new debt**: baseline/ratchet guidance lists only the
  steps the ratchet actually needs.
- **Milestone packets satisfy the standalone portfolio contract**, and incomplete executable
  assurance declarations are rejected by the declaration validator (a library surface; no
  command reads a declaration in this release).

### Compatibility and migration

Every tightening below is deliberate and was accepted with its own evidence before release; none of
it is incidental. A design that was green on v0.6.11 can still become blocking on 0.7.0 without any
design edit, so each entry states what still works unchanged, what is rejected now, the exact
finding a consumer sees, and the exact edit or command that migrates. The reference counts come
from one real private design that exited 0 on v0.6.11 and reports 78 blocking findings on 0.7.0.

**Regeneration stamp.** Upgrading restamps generated artifacts and nothing else. The families that
carry `machinery-version:` are `machines/*.tla`, `machines/*.cfg`, `machines/*.oracle.md`, and
`formal/*.als`; on the reference design that is 254 files whose only changed line moves from
`machinery-version: v0.6.11` to `machinery-version: v0.7.0`. Regenerate with the commands the gate
suite prints for the design (`machinery oracle <design>`, `machinery tla <design>`,
`machinery alloy <design>`, `machinery refine`/`machinery compose`/`machinery pack generate` where
the design carries those families) and commit the stamp-only diff on its own.

**Per-edge consumer `READS` declarations (`Gx-trace`, armed designs only).** Unchanged: a design
that does not arm `machinery:reads-complete` in `ARCHITECTURE.md` is untouched, and a single-
consumer event keeps its existing single `READS{...}` declaration as long as ownership resolves to
exactly one participant. Rejected now: an event that fans out to two or more consumers, where one
`READS{...}` used to satisfy every participant. Each `(event, consumer)` edge owes its own
declaration. Finding (45 on the reference design):

```
event-contract row 7 (event 'markPaid'): no consumer READS declaration for event 'markPaid', consumer 'audit'; state READS{field, ...} beside the event name and name this participant in the matrix consumer column, or waive this consumer cell with '(no reads: <reason>)'
```

Migrate: in the payload matrix, give the row a `consumer` column naming the exact event-contract
participant and write that participant's own `READS{...}`; repeat per consumer. A participant that
reads nothing gets `(no reads: <reason>)` in its consumer cell instead. A waiver never transfers to
a sibling, and one consumer may read a strict superset without imposing it on the others.

**Legacy `READS` with ambiguous consumer ownership.** Unchanged: a legacy declaration with no
consumer column still resolves when the event has exactly one distinct, named owner. Rejected now:
the same declaration when the event has more than one participant, or an unnamed one. Finding (26
on the reference design):

```
Payments.matrix.md:41: legacy READS for event 'markPaid' has ambiguous consumer ownership; add an explicit consumer column matching each event-contract participant
```

Migrate: add the `consumer` column to that matrix table and split the row per participant, as
above. A consumer name that is not an exact event-contract participant is its own finding
(`names consumer '...', not an exact event-contract consumer`), so copy the participant spelling
from the `ARCHITECTURE.md` event-contract table.

**Conflicting `READS` field sets across rows and matrices.** Unchanged: repeating the same edge
with the identical field set in several rows or several matrices. Rejected now: two rows for one
`(event, consumer)` edge whose exact field sets differ. Finding (4 on the reference design):

```
Audit.matrix.md:12: conflicting READS for event 'markPaid', consumer 'audit'; exact field set differs from Payments.matrix.md:41 (all rows and machines for this edge must agree)
```

Migrate: pick the true read set for that edge and make every row and machine state it identically,
or split the edge by naming distinct consumers.

**One complete `READS` group per row.** Unchanged: `READS` as an English verb in residual prose or
in a contract cell ("the projection only READS those rows") is narrative again and is no longer
collected as a declaration; a member is any trimmed, non-empty text in the design's ubiquitous
language, so `READS{Order.id, occurrence time, PAIR KEY}` declares three fields. Empty and
duplicate members are rejected as they always were, and an empty member is now reported as an empty
member rather than as a field name that is neither. Rejected now: a row that carries a `READS{...}`
group wrapped across two markdown lines, or two groups on one row. Findings (1 on the reference
design):

```
Payments.matrix.md:62: event 'markPaid', consumer 'ledger': expected exactly one complete READS{field, ...} declaration per row
Payments.matrix.md:62: event 'markPaid', consumer 'ledger': empty READS member ''; name each field explicitly, and drop the stray separator
```

Migrate: join the wrapped group onto one line so the row carries exactly one closed
`READS{field, ...}`, and drop the stray separator that produced the empty member.

**One `CLAUSES` vocabulary per row (`Gd-idcite`).** Unchanged: a matrix's prose is narrative, not a
declaration. An invariant-coverage bullet that quotes a guard's vocabulary, a contract cell that
mentions "the CLAUSES declaration here", a description of a `RETIRED` site, an enumeration wrapped
across lines, and a row for a non-guard unit are all accepted exactly as they were on v0.6.11.
Rejected now: a named-unit contract row carrying two `CLAUSES{...}` groups, or a `RETIRED{...}`
group that the declaration itself does not carry, because such a row names no single vocabulary.
Finding (1 on the reference design):

```
Payments.matrix.md:18: machine "Payments" guard "guardPaid": malformed CLAUSES declaration; require one named row with a single CLAUSES{...} and optional RETIRED{...}
```

Migrate: keep one `CLAUSES{...}` group, with at most one `RETIRED{...}` inside the same
declaration, on the unit's own row of the named-unit contract table.

**Guard clause ownership resolves against sibling oracles.** Unchanged: a declared guard that no
committed oracle governs at all owes nothing and reports nothing, which is v0.6.11 behavior
restored. This covers the common creation-edge guard: a guard that gates an insert rather than a
guarded transition is silent, and 0.7.0 does not require it to gate a transition. Rejected now: a
declaration in `Alpha.matrix.md` whose falsifying-clause rows only `Beta.oracle.md` could supply.
`Alpha.matrix.md` binds only `Alpha.machine.json` and `Alpha.oracle.md`, so two machines may reuse
one guard name without either arming or satisfying the other's obligation. Finding:

```
Alpha.matrix.md:23: machine "Alpha" guard "guardApproved": owning oracle has no transition governed by this guard, but Beta's does; another machine cannot supply it
```

Migrate: move the declaration to the matrix of the machine whose oracle governs the guard.

**Oracle coverage (`Gt-tests`) credits active discovery only.** Unchanged: a suite that names each
stable id as a whole token in an active test body, or that cites the oracle file by name inside a
single test with a connected read-parse-assert flow, still passes; `Gt` still executes no tests and
labels itself `static discovery; tests not executed; unsupported parser structures remain
uncovered`. Rejected now: evidence that is not executable at all, which used to pass: commented-out
or disabled tests, unused declarations, uncalled parsers and helpers, and mentions outside a test
body. Finding:

```
Payments.oracle.md: 12 of 40 stable ids appear in no test file (PAY-001, PAY-002, ...); key the tests on the stable ids, or parse the committed table at runtime by naming Payments.oracle.md in a test
```

Migrate: move the id references into active test bodies, or keep one real test that reads and
parses the committed oracle table.

**Attestation evidence v2 kinds (`Gv-attest`).** Unchanged: `attestation_version: 1` files still
parse, and every claim the vocabulary classifies as design-only keeps passing as an implicit plan
judgment, with a note asking for the explicit kind. `machinery check <design>` on a design-only
design reports its missing-current warnings and still exits 0. Rejected now: a v1 row for one of the
three claims the vocabulary classifies as `current` (`gt.conformance-test-shape`,
`g4.standin-coverage`, `g4.pack-event-discipline`), which used to pass as a design-only cover; a v2
`kind: current` row with no complete implementation subject; and a `kind: current` row evaluated
without `--impl`. Findings:

```
GV_MISSING_IMPLEMENTATION_SUBJECT: gt.conformance-test-shape has legacy design-only covers; review the complete implementation/test scope and run machinery attest --design <design> --claim gt.conformance-test-shape --kind current --impl <root> --attestor <reviewer> --date YYYY-MM-DD, or explicitly recast as v2 kind=plan
GV_MISSING_IMPLEMENTATION_SUBJECT: gt.conformance-test-shape requires a complete implementation subject
GV_IMPL_REQUIRED: gt.conformance-test-shape requires --impl <reviewed-root>; the receipt locator grants no read authority
```

Migrate, whichever is true of the design. Design-only, no implementation to review yet: set
`attestation_version: 2` and add `kind: plan` to the row; `machinery check` then warns
`gt.conformance-test-shape: plan only; current implementation review missing` and exits 0. A real
reviewed implementation: run

```
machinery attest --design <design> --claim <claim> --kind current --impl <root> --attestor <reviewer> --date YYYY-MM-DD
```

and pass `--impl <root>` to every later `machinery check` that evaluates the claim. A subject whose
bytes move after the review invalidates it (`GV_STALE_CONTENT`, `GV_SCOPE_INVENTORY`); review the
changed scope and attest again. `ga.review-quality` is the one historical-class claim: it takes
`kind: historical` and establishes history, not current implementation approval. Every other claim
in the vocabulary is plan-class and takes `kind: plan`.

**`--impl` arming.** Unchanged: `G4-import` and `Gt-tests` still run only when `--impl` is
supplied; `--complete` still requires `--impl`; an explicit `--gate g4` or `--gate gt` without
`--impl` is still an invocation error. Changed: a decomposed parent with no `machines/` and a
supplied `--impl` now also runs `Gt` when the parent owns relational obligations (a policy or
isolation annotation, or a committed relational oracle under `formal/`). v0.6.11 dropped `Gt` for
that parent and reported nothing, so its policy and isolation test obligations were invisible.
The selection note says so: `gt checks parent-owned relational obligations`. Migrate: give the
parent's own tests the parent oracle's stable ids, or run without `--impl` to keep the design-only
selection.

**`G4-import` ignore globs.** Unchanged, and stated because it is easy to reach for the wrong file:
`G4` prunes the implementation tree with the `ignore` glob list of the Architecture Contract in
`ARCHITECTURE.md`, not with `.machineryignore`. `.machineryignore` at the design root still governs
design-tree reads only, with the same gitignore-shaped subset (no negations, no re-inclusions).
Neither file changed in 0.7.0.

**Regeneration advice no longer proposes `machinery baseline`.** Unchanged: `machinery baseline
<design> --impl <dir>` remains a supported explicit command. Changed: a version-skew or
regeneration message on a design carrying a ratchet no longer lists it among the regeneration
steps, because a baseline snapshot accepts tolerated import offenders and can widen accepted
architecture debt. Its own output and help now say that a baseline accepts debt and needs review
even when it proposes no new dependency rule. Migrate: nothing, unless a script pasted the printed
regeneration list verbatim, in which case drop the baseline line from it and run baselines
deliberately.

**Installer reruns and update receipts.** Unchanged: `machinery update` without selectors, and
explicit `--home`/`--target` selectors, behave as before; selectors still cannot be combined with
`--bootstrap-defaults`. Changed: rerunning the one-line installer over an existing install now
refreshes the complete recorded home, native-target, and host-plugin plan, preserving each recorded
group's copy/symlink mode, instead of refreshing the default homes only. Rejected now: a corrupt,
unsupported, or unsafe receipt, an unsafe artifact or parent substitution, and uncertain plugin
ownership all fail closed with a diagnostic naming the problem, including under `--skip-plugins`,
where older releases silently refreshed the default homes instead. Safely edited or missing owned
regular content with unchanged safe parents is repaired to exact current-release content. Migrate:
a failing rerun names the problem; repair or remove the reported artifact and rerun, or run
`machinery install` with explicit `--home`/`--target` selectors to record a fresh plan.

**Host plugin refresh failures are returned failures.** Changed: a missing host CLI, a failed or
misunderstood plugin inventory, or a failed refresh is now a returned error naming the exact retry,
where 0.6.11 reported a warning over a claimed-successful update. The direct update stays
committed, and the retry obligation is recorded for the next update. Host-owned plugin caches are
never rolled back by machinery. Migrate: run the exact command the error names (for Claude Code,
`claude plugin update machinery@machinery`), or pass `--skip-plugins` to opt out of host plugin
refresh entirely.

**windows/amd64 release artifacts without installer support.** Added, with a stated limit: releases
now publish `machinery-windows-amd64` and `machinery_<version>_windows_amd64.tar.gz`, but the
one-line installer and `machinery update` still refuse Windows, so the asset is downloaded and
placed by hand. Native Windows runtime guarantees are not claimed: process custody, formal
verification, and the assurance lanes are unix-only and refuse on Windows. Migrate: use Linux or
macOS for the full toolchain.

**Contributors: the `SKIP_PREFLIGHT` bypass is gone.** Rejected now: `SKIP_PREFLIGHT=1 git push`.
`scripts/preflight.sh` no longer honors the variable, and the pre-push hook, `CONTRIBUTING.md`, and
the `make hooks` message say so. Migrate: run the full preflight, or remove the hook locally
(`git config --unset core.hooksPath`) and accept that the exact-commit release gate is the only
remaining check.

**Known limitation, unchanged since 0.6.x: directory change witnesses on Linux before 6.13.** The
fail-closed directory inventories (`internal/dirscan`, the gate confinement walker, the oracle
inventory, the governance hook's design inventory) detect concurrent mutation through the
directory's inode change stamp. On Linux kernels without multigrain timestamps (6.12 and older,
including Ubuntu 24.04 LTS at 6.8) that stamp is one timer tick coarse and `STATX_CHANGE_COOKIE`
is not exposed, so a create-then-delete completed inside one tick is not detected; hosted CI
passes because its runner kernel is 6.17. Not a regression. A kernel-event
witness (inotify, kqueue) is under review for 0.7.1. Until then, inventory results on such kernels
are stamp-witnessed only.
