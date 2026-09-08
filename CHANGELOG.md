# Changelog

Notable user-facing changes to machinery. Release notes for generated-output and proof-scope
changes follow the discipline in [docs/release-notes.md](docs/release-notes.md); entries move
under their version heading when a release is cut.

## [Unreleased]

## [0.7.0] - 2026-09-07

### Fixed

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
- **Executable assurance declarations and four native test adapters.** A design may declare
  `design/assurance/plan.json` and per-milestone manifests under a closed version-1 grammar that
  binds every oracle row, guard clause, invariant, and runtime obligation to registered native
  tests. Adapters for Go (`go test`, Go 1.27.1), TypeScript (`tsc` plus `node --test`, Node
  26.8.1 / TypeScript 7.0.2), Python (`python -m unittest`, CPython 3.14.7), and Elixir
  (`mix test`, Elixir 1.20.4 / OTP 29) run the real native runner under process custody in a
  closed environment against a pinned, byte-verified runtime closure; each registered assertion
  must be a typed helper call at its frozen line, and its outcome is witness-bound to that call
  site. Skips, expected failures, unwitnessed failures, and late source mutation are rejected.
  The required integration lane carries the assurance catalog: it verifies every adapter runtime
  against the exact pins and executes frozen per-language probe fixtures with exact per-case
  accounting; a missing or mismatched runtime fails the lane.
- **Replay inputs are retained outside governed sources.** Capture publishes the exact subject,
  dependency, frozen, and design payload inputs as content-addressed blobs in a private assurance
  store (symlink escapes, hardlink aliases, and special files are rejected; the control namespace,
  `.git`, and `attestations.yaml` are excluded), with a typed bundle encoding, a validated
  export/import archive, and a read-only status report that never performs a replay.
- **Reviewed assurance revisions are registered explicitly.** Registration validates lineage
  (revision 1, or the predecessor plus one with the same key), archives the exact control bytes,
  stages the successor, and advances the store head by compare-and-advance under a bounded
  deadline; drafts never discharge registered requirements, and a registered deletion or an
  untargeted registered mismatch blocks.
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
  assurance declarations are rejected.
