# Changelog

Notable user-facing changes to machinery. Release notes for generated-output and proof-scope
changes follow the discipline in [docs/release-notes.md](docs/release-notes.md); entries move
under their version heading when a release is cut.

## [Unreleased]

### Fixed

- **Installer reruns converge on the recorded installation.** The one-line installer over an
  existing install now refreshes the complete recorded home/native-target/plugin plan (each
  group's copy/symlink mode preserved) instead of the default homes only — identical to
  `machinery update` without selectors. Safely edited or missing owned artifacts are repaired to
  exact current-release content; corrupt, unsupported, or unsafe receipts, unsafe artifact or
  parent substitutions, and uncertain plugin ownership fail closed with diagnostics (including
  under `--skip-plugins`), never silently falling back to the defaults. A late failure rolls the
  whole direct transaction back to the exact pre-run state — a previously absent artifact is
  restored to absence — and receipt publication is validated and parent-owned; concurrent foreign
  changes are refused, not overwritten.
- **Host plugin refresh failures are reported as failures.** A missing host CLI, a failed or
  misunderstood plugin inventory, or a failed refresh is a returned error naming the exact retry;
  the direct update stays committed and the obligation is recorded for the next update, instead
  of a warning over a claimed-successful update. Host-owned plugin caches remain outside
  machinery's rollback.
- **Oracle coverage (`Gt`) credits discovery only.** Static test-reference analysis is labeled
  "static discovery; tests not executed", and bypass shapes — commented-out or unused tests,
  uncalled parsers and helpers, evidence living outside the test — are rejected instead of
  passing.
- **OpenCode governance subprocesses are bounded and validated.** A deadline plus a capped output
  capture terminates runaway children, and unrecognized, truncated, or combined-protocol hook
  responses block instead of best-effort parsing.
- **Unmodeled saga compensation routes are rejected** in refine and compose instead of proving
  green against routes no model defines.
- **Go-crm boundary mapping** covers `internal/testoracle`, and FSM conformance binds to committed
  oracle expectations.

### Added

- **`machinery recover`**: pass a design directory for a read-only report of an interrupted
  publication — expected outputs with content/mode status, journal and residue locations,
  live-writer status, and the safe recovery decision. `--apply` finalizes only a fully
  revalidated interrupted publication; refusals preserve all evidence.
- **Required integration lane**: `go run ./scripts/integration-lane --lane required` executes
  every infrastructure-dependent suite (Docker-backed checker lifecycle, publication recovery,
  native custody, adapter governance, and the documented checker registry/bind-path example) with
  exact test inventories, pinned runtimes, and real provisioning/teardown accounting. Missing
  infrastructure fails the lane with a diagnostic; nothing is skipped.
- **Native custody for machinery's own subprocesses.** Formal verification and tooling
  subprocesses run inside an owned process-custody scope with registration before launch, bounded
  budgets, and terminal-kill-before-reap cleanup. Honest limits: cleanup failures are reported,
  never concealed, and custody is process-ownership containment — not a defense against a
  hostile host.
- **Checker container budgets and lifetime ownership.** Every checker container gets closed
  finite budgets — 128 MiB memory, half a CPU, a 32-process pid limit — a registered identity,
  and force-removal on every exit path; an output breach aborts and cleans up instead of
  buffering.
- **Standalone contract documents** for native custody and executable test assurance
  (`docs/native-custody-contract.md`, `docs/test-assurance-contract.md`).

### Changed

- **Attestation evidence v2 kinds (`plan`/`current`/`historical`).** Example attestations
  migrated to truthful kinds: a `current` claim requires an implementation subject, legacy
  design-only covers are rejected with migration guidance, and implementation-subject changes
  invalidate stale reviews so a review binds the exact scope it attests.
- **Frozen-test identity requires exact bytes.** Formatting-only rewrites are no longer treated
  as the same frozen test; evidence replay requires exact identity with no whitespace exemption.
- **Event-contract consumer `reads` bind per edge.** A consumer that legitimately reads more than
  its siblings declares its exact read set on its own edge instead of sharing the intersection.
- **Regeneration advice no longer accepts new debt**: baseline/ratchet guidance lists only the
  steps the ratchet actually needs.
- **Milestone packets satisfy the standalone portfolio contract**, and incomplete executable
  assurance declarations are rejected.
