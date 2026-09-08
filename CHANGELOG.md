# Changelog

Notable user-facing changes to machinery. Release notes for generated-output and proof-scope
changes follow the discipline in [docs/release-notes.md](docs/release-notes.md); entries move
under their version heading when a release is cut.

## [Unreleased]

## [0.7.0] - 2026-09-08

### Fixed

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
