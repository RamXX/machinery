# MIGRATION: machinery tooling, Python to Go (100%)

Status: not started. Every box below is unchecked on purpose; tick them as you go.

Goal: replace all Python tooling with a single Go binary (`machinery`), matching modelith's
distribution model, until no Python remains and we can delete it with proof, not hope. TLA+/TLC
stays on the JVM (out of scope); Go orchestrates it by shelling out to `java`.

## Guiding principles

- **Behavior is frozen and is the spec.** During migration the Python tools change only for
  bug-for-bug parity. No feature work, no "while we're here" improvements. Every improvement waits
  until after Python is deleted, so the diff we are validating is pure.
- **Golden + differential, not faith.** The tools are deterministic (content-hashed oracle ids,
  fixed TLA+ text, exact gate strings and `checked:` counts, fixed exit codes). We capture Python
  output as a golden corpus and prove Go reproduces it byte-for-byte and exit-code-for-exit-code,
  everywhere, before Python is removed.
- **Port in dependency order, lowest risk first.** The shared IR underpins everything, so it goes
  first and is proven hardest. Pure generators next. The gate suite last.
- **Two implementations run side by side in CI** (shadow mode) for a burn-in window; a single
  divergence fails the build. Python is deleted only after the shadow is silent.
- **Every phase has a runnable exit gate.** A phase is done when its gate command passes, not when
  the code looks right.

## Non-goals / scope boundaries

- [ ] TLA+/TLC stays Java. `machinery verify-formal` shells out to `java -cp tla2tools.jar`; we do
      not reimplement a model checker.
- [ ] No change to the methodology, the design artifacts, the CLI-visible behavior, or the gate
      semantics. Same findings, same strings, same counts, same exit codes.
- [ ] `bash` in `tlc.sh` / `verify_formal.sh` is not Python; it may stay. This plan folds both into
      the Go binary anyway so the end state is one binary, but that is a convenience, not a blocker.

## Target architecture

- [ ] One Go module at the repo root (`go.mod`), cobra-based like modelith, producing a single
      static binary `machinery`.
- [ ] Layout:
      - `cmd/machinery/main.go` (cobra root + version)
      - `internal/ir/` (machine model, order-preserving JSON/YAML, `WalkStates`, `TransitionsOf`,
        `ActionNames`, markdown-table parsing) -- the port of `machine_lint.py`'s shared IR
      - `internal/lint/` (structural lint + matrix reconciliation) -- rest of `machine_lint.py`
      - `internal/oracle/` (`oracle_gen.py`)
      - `internal/tla/` (`tla_gen.py`)
      - `internal/refine/` (`refine_gen.py`: linear-lifecycle, terminal-lifecycle, saga)
      - `internal/compose/` (`compose_gen.py`)
      - `internal/gates/` (`machinery_check.py`: G2-c4, G3-machine, Gx-trace, G4-import)
      - `internal/formal/` (verify-formal orchestration + tlc invocation)
      - `internal/diag/` (doctor/preflight)
- [ ] Subcommands map 1:1 to today's entry points:
      `machinery lint <dir>`, `machinery oracle <dir>`, `machinery tla <machine> [out]`,
      `machinery refine <machine> <sem> [out]`, `machinery compose <comp> <coord> [out]`,
      `machinery check <design> [--impl d] [--gate ...]`, `machinery verify-formal <design>`,
      `machinery doctor`, `machinery preflight`.

---

## Phase 0 -- Freeze and capture the golden corpus

Goal: pin the exact observable behavior we must reproduce.

- [ ] Tag the pre-migration commit (`git tag pre-go-migration`) for rollback.
- [ ] Freeze the Python tools: open a `migration/freeze` note; only parity fixes land on the tools
      until deletion.
- [ ] Enumerate the CLI contract of every tool: args, flags, stdout format, stderr format, and the
      exact exit-code rules (0 clean; non-zero on ERROR/DRIFT; the specific `sys.exit` messages).
      Write it into `internal/README-contract.md` as the spec.
- [ ] Define the **corpus** (see "Corpus" below) and commit it under `testdata/`.
- [ ] Build `scripts/capture-golden.sh`: run every Python tool over the whole corpus and write
      stdout+stderr+exit-code+emitted-files to `testdata/golden/<case>/`. Commit the golden tree.
- [ ] Exit gate: `scripts/capture-golden.sh` is deterministic -- run it twice, `git diff` is empty.

## Phase 1 -- Go scaffold and the differential harness (build the ruler before the thing it measures)

Goal: a harness that, for any tool and input, asserts Go output == Python output, byte-for-byte,
and exit-code equal. It fails everywhere at first (Go is stubs); that is correct.

- [ ] `go mod init`; add `gopkg.in/yaml.v3`, cobra, and an order-preserving JSON decoder
      (`encoding/json` via `json.Decoder` token stream, or a small ordered-map type), and
      `github.com/bmatcuk/doublestar/v4` for `**` globs.
- [ ] Stub every subcommand to exit 2 with "not implemented".
- [ ] `scripts/diff-tool.sh <tool> <args...>`: runs `python3 <tool>.py <args>` and
      `machinery <sub> <args>`, captures stdout/stderr/exit/emitted-files, and diffs them; nonzero
      on any difference. Normalize only absolute paths and nothing else.
- [ ] `scripts/diff-all.sh`: drives `diff-tool.sh` across the entire corpus for every subcommand.
- [ ] CI job `differential` that runs `diff-all.sh` (allowed to fail during migration; it is the
      live scoreboard).
- [ ] Exit gate: `diff-all.sh` runs end to end and reports per-case PASS/FAIL (all FAIL now).

## Phase 2 -- Shared IR (`machine_lint.py` primitives) + the ordering guarantees

Goal: parse machines and markdown identically to Python, including source order. This is the
highest-risk fidelity work; do it first and prove it hardest.

- [ ] Port an order-preserving machine loader: JSON object key order MUST survive (states, and the
      `on`/`after` maps inside a state, are emitted in source order downstream). Verify a decoder
      that preserves order; never unmarshal into a bare `map[string]any`.
- [ ] Port `WalkStates`, `TransitionsOf` (on / after / always / state-onDone / invoke onDone /
      onError), `ActionNames` (string, `{type}`, list-of), `InvokesOf`, target resolution.
- [ ] Port the transition-value polymorphism (`_norm`): a transition is a string, a dict, or a list
      of either; array targets and non-string guards are recorded as problems, not narrowed.
- [ ] Port markdown parsing: `ParseMdTables` (pipe split, strip, separator-row detection
      `set(sep) <= {'-',':',' '}`), `FindCol`, `_clean_cell`, the documentation-row skips
      (`(final)`, `(any event)`, `(internal)`).
- [ ] Add an `machinery ir-dump <machine>` hidden subcommand that serializes the parsed IR
      (states in order, every transition tuple) to canonical JSON. Add the same dump to Python.
- [ ] Exit gate: for every machine in the corpus, `ir-dump` from Go and Python are byte-identical
      (`diff-tool.sh ir-dump ...` PASS for all). This proves ordering and traversal parity before
      anything is generated from it.

## Phase 3 -- oracle_gen (byte-exact, content-hashed ids)

Goal: identical oracles, because their stable ids are referenced by every matrix and BUILD.md.

- [ ] Port `stable_id`: `sha256("{tag}|{source}|{trig}|{guard or ''}")` hex[:6], `tag =
      upper(id)[:4]`, guard-None to empty string, and the `seen`-counter `.N` disambiguation in
      source order. This must be bit-exact.
- [ ] Port `render`: the entry/exit table, the transition table, trailing blank lines, and the
      final "Total transitions" line, including exact whitespace and the terminal newline.
- [ ] Exit gate: `machinery oracle` regenerates `go-crm`, `fulfillment`, `portfolio-engine`, and the
      synthetic fixtures with zero byte difference vs the committed oracles and vs Python
      (`diff-tool.sh oracle ...` PASS for all). Any diff here fails the migration.

## Phase 4 -- machine_lint (structural lint + matrix reconciliation)

Goal: identical findings and identical CLI text/exit codes.

- [ ] Port `LintMachine`: unsupported-key detection (root/state/invoke key sets), state-type checks
      (reject parallel/history), target resolution + dangling/ambiguous, reachability (hierarchical,
      through compound initials), dead-end non-final leaves, invoke-needs-onError-and-after, shadowed
      branches, fully-guarded-always requiring `after`/`on`/`invoke` escape or `_exhaustive`, and the
      resting-state event-completeness (`_ignores`) rules.
- [ ] Port `ReconcileMatrix` (bidirectional structural reconciliation), `NamedUnitNames`,
      `MachineUnitNames`, `MachineTransitionRows`, `MatrixTransitionRows`, and `_guard_matches`
      (the `!guard` / `(else)` / `-` fallback conventions).
- [ ] Reimplement the one lookaround regex (`_token_in`) as a manual whole-token scan over
      `[A-Za-z0-9_-]` boundaries; add a unit test that `inv-1` does not match inside `inv-12`.
- [ ] Port the CLI (`machinery lint <dir>`): per-file headers, ERROR/DRIFT/warn/note lines, the
      trailing summary, and the non-zero exit on error/drift.
- [ ] Port the relevant unit tests from `tests/test_machine_lint.py` (all 30+ cases) to Go and to
      the shared mutation corpus (Phase 10).
- [ ] Exit gate: `diff-tool.sh lint ...` PASS across the corpus; Go unit tests green.

## Phase 5 -- tla_gen (control-flow model)

- [ ] Port `classify`, `retry_states`, per-transition action synthesis, the retry template (one
      bounded counter per retry state), the `Terminated` clause for finals, and the ASSUMPTIONS
      header including the UNVERIFIED `_exhaustive` listing.
- [ ] Reproduce the hard-error paths (nested/compound states, unsupported type, multi-`after` retry)
      with identical `tla_gen:` messages and exit codes.
- [ ] Exit gate: `diff-tool.sh tla ...` PASS for every machine in the corpus (both `.tla` and
      `.cfg`), and the regenerated example `.tla` match what is committed.

## Phase 6 -- refine_gen (data refinement: three patterns)

Goal: identical reconciliation errors and identical emitted modules for linear-lifecycle,
terminal-lifecycle, and saga.

- [ ] Port `ReconcileLifecycle` (parameterized overlay names via `overlay:`, defaults
      persisting/persistRetry/rolledBack) and `EmitLifecycle` (Data/Contract/Refinement + cfgs).
- [ ] Port `ReconcileTerminal` and `EmitTerminal` (phases, success/failure terminals, optional
      retry overlays, completion flag).
- [ ] Port `ReconcileSaga` and `EmitSaga` (per-obligation compensation, FailedDirty).
- [ ] Reproduce every `RECONCILIATION FAILED:` message verbatim (the tests and the differential
      harness assert exact text), and the `unsupported pattern` exit.
- [ ] Feed each reconciler its drift mutations (wrong stage set, terminal that accepts a forbidden
      event, stale rollback routing, missing event name, reordered saga steps, dropped undo, a
      served-phase failing into a terminal): Go must die with the same message class as Python.
- [ ] Exit gate: `diff-tool.sh refine ...` PASS on all `*.semantics.yaml` in the corpus (emitted
      files identical) AND every drift case exits nonzero with matching message on both.

## Phase 7 -- compose_gen (cross-aggregate composition)

- [ ] Port `forward_chain` validation against the coordinator machine, the branching emission
      (per-step Done/Fail, per-obligation Undo, CompensateDone/Stall), the auto
      `Inv_CleanCompensation`, and the declared-invariant expansion.
- [ ] Reproduce the `VALIDATION FAILED:` messages and exits (step-order drift, coordinator failure
      reroute, missing undo).
- [ ] Exit gate: `diff-tool.sh compose ...` PASS on the fulfillment composition and its drift
      mutations.

## Phase 8 -- machinery_check (the gate suite: G2, G3, Gx, G4)

Goal: the whole `machinery check` output, `checked:` counts, findings, and exit codes match, across
clean designs and every mutation experiment.

- [ ] Port the contract locator/parser (the `Architecture Contract` heading + `contract_version`
      fence fallback), boundary/external validation, `dependency_rules` edge parsing, and the
      workspace.dsl element parser (`dsl_elements`, tag indices).
- [ ] Port G2-c4: binding, duplicate ids, allow/deny conflict, undeclared refs, and mitigation
      coverage (externals + Database/Queue/External-tagged elements; backticked first-column match).
- [ ] Port G3-machine: run the lint, diff the committed oracle against a fresh in-memory generation
      (DRIFT), reconcile the matrix, named-unit coverage, and the aggregated `_exhaustive` warn +
      count.
- [ ] Port Gx-trace: enum/state binding both directions, events-are-actions, entity-has-machine,
      the placement table (first backticked token only), the invariant enforcement split
      (unit-backed vs attested vs the total), and the orphan maps-to check.
- [ ] Port G4-import: the boundary code-glob mapping (with `**` via doublestar), the import
      extractors for Go (single + block + go.mod module), Python, TypeScript/JavaScript, Elixir
      (module + `modules:`), and Rust; `exposes` enforcement; allow/deny/undeclared edge findings;
      test-file skipping; `ignore` globs.
- [ ] Port the `Gate` accumulator, `require_nonzero`, the `emit()` formatting (ERROR/DRIFT/warn/note,
      the `checked:` join, the `ok` line), and the `--gate` subset + `--impl` gating + the final
      "N blocking" line and process exit.
- [ ] Port all cases from `tests/test_machinery_check.py` (experiments A, B, D, E, F1, G, H, and the
      placement/enforced-split cases) to Go and to the shared mutation corpus.
- [ ] Exit gate: `diff-tool.sh check ...` PASS on `go-crm --impl`, `fulfillment`, `portfolio-engine`,
      the synthetic fixture, and every mutation, with identical `checked:` counts and exit codes.

## Phase 9 -- CLI unification, verify-formal, doctor/preflight

- [ ] Wire all subcommands into the cobra root with `machinery version`.
- [ ] Reimplement `verify_formal.sh` as `machinery verify-formal <design>`: discover
      `*.machine.json`, `*.semantics.yaml`, `*.composition.yaml`; regenerate via the Go generators;
      then run TLC. Reimplement `tlc.sh` (pinned jar fetch + checksum + `states/` cleanup) as an
      internal step that shells to `java`. Keep the exact PASS/FAIL lines and the
      "No error has been found" + zero-exit acceptance.
- [ ] Reimplement `doctor`/`preflight` in Go (same labels: ok/MISSING/optional/auto; Java stays
      optional).
- [ ] Repoint the `Makefile` (`check`, `verify-formal`, `oracle`, `doctor`) to the `machinery`
      binary; keep `make install` building/placing it.
- [ ] Exit gate: `make check` and `make verify-formal` are green on all three examples using ONLY
      the Go binary; the TLC PASS/FAIL output matches the bash version line for line.

## Phase 10 -- Consolidate the regression net as shared, language-neutral fixtures

Goal: the pytest suite's real value is the vacuity/drift experiments; make them data that both
implementations run, so parity on the adversarial cases is guaranteed, not duplicated by hand.

- [ ] Extract every mutation experiment (from all four `tests/test_*.py`) into
      `testdata/experiments/<name>/` as `{input snapshot, mutation, expected-finding-substring,
      expected-exit}`.
- [ ] Build a table-driven runner (Go test) that applies each experiment to the Go tools and asserts
      the expected finding + exit.
- [ ] Add a Python shim that runs the SAME experiment table against the Python tools (temporary),
      so the differential harness covers the adversarial set too.
- [ ] Port `tests/fixtures.py` (the synthetic design + impl) to a Go fixture builder that emits the
      identical files.
- [ ] Exit gate: Go table runner green; the experiment table passes identically on Python and Go.

## Phase 11 -- Shadow burn-in in CI

Goal: accumulate evidence over time that the two implementations never disagree.

- [ ] Promote the `differential` CI job to required: on every push, `diff-all.sh` + the experiment
      table must show Go == Python everywhere, else the build fails.
- [ ] Add a nightly job that regenerates all committed oracles/`.tla` with Go and asserts an empty
      `git diff`.
- [ ] Run a real end-to-end dogfood: author a fresh small design using ONLY the Go tools through all
      four phases + verify-formal; record it under `testdata/dogfood/`.
- [ ] Exit gate: `differential` green for K consecutive CI runs (recommend K >= 20 or two weeks of
      activity), zero divergences, dogfood passes.

## Phase 12 -- Flip default to Go; Python demoted to shadow oracle

- [ ] Make the Go binary the tool everything calls (Makefile, docs, CI gate runs).
- [ ] Keep Python present but only invoked by the `differential` job as the oracle.
- [ ] Update `README.md`, `skills/machinery/SKILL.md`, `skills/machinery/tools/README.md`, and the
      agent docs to reference the `machinery` binary; drop the `python3 .../X.py` and
      `uv run --with pyyaml` invocations; state that PyYAML/Python are no longer runtime deps.
- [ ] Exit gate: a full `make test && make check && make verify-formal` is green with Go as the
      sole caller; the differential job (still running Python as oracle) stays silent.

## Deletion gate -- the criteria to remove Python entirely

Delete only when ALL of these are true (each a checkbox to tick at the end):

- [ ] Go passes its own full unit suite (the ported `test_*` cases) and the experiment table.
- [ ] `diff-all.sh` shows Go == Python byte-for-byte on stdout, stderr, emitted files, and exit code
      across the entire corpus, for K consecutive green CI runs (Phase 11).
- [ ] All committed oracles and `.tla` regenerate byte-identically under Go (nightly job empty diff).
- [ ] `make check` and `make verify-formal` green on go-crm (with `--impl`), fulfillment, and
      portfolio-engine using only Go.
- [ ] The end-to-end dogfood design (Phase 11) passes all gates and proofs with only Go.
- [ ] No open parity defect in the freeze log.

When every box above is ticked:

- [ ] Delete `skills/machinery/tools/*.py`, `tests/*.py`, `pyproject.toml`, `uv.lock`, and the
      PyYAML/pytest references, in one commit tagged `python-removed`.
- [ ] Remove the Python shim and the `differential` oracle job; keep the Go table runner + golden
      diff as the permanent regression net.
- [ ] Update `.gitignore` (`__pycache__/`, `.venv/`, `.pytest_cache/` can go), the Makefile `test`
      target (now `go test ./...`), and CI.
- [ ] Final acceptance: fresh clone, `make install` (Go binary), `make test`, `make check`,
      `make verify-formal` all green with no Python interpreter on PATH.

## Corpus (what "everywhere" means)

- [ ] The three example/experiment designs: `examples/go-crm` (incl. `--impl`),
      `examples/fulfillment`, `examples/portfolio-engine`.
- [ ] The synthetic design + impl from `tests/fixtures.py`.
- [ ] Every machine, semantics, and composition annotation therein (each generator, each pattern:
      linear-lifecycle, terminal-lifecycle, saga, composition, and control-flow).
- [ ] The full experiment/mutation table (Phase 10).
- [ ] A few hand-authored adversarial machines that stress the IR: deeply nested compound states,
      multi-retry-loop machines, list-form transitions, `{type}` actions, `_ignores`/`_exhaustive`
      edge cases, and the ordering-sensitive cases (many `on` events whose emission order matters).

## Fidelity hazards (the things that will bite; each is a test target)

- [ ] **JSON/YAML key order.** Python dicts preserve insertion order; Go maps do not. Oracle rows and
      TLA+ actions are emitted in source order. Use order-preserving decoders; the Phase 2 `ir-dump`
      test is the guard.
- [ ] **sha256 stable ids.** Exact input string, `upper()[:4]` tag, empty-string guard, `.N`
      disambiguation. Bit-exact or every cross-reference breaks.
- [ ] **Globs.** `**` and the custom `_match_glob` prefix behavior; use doublestar and unit-test
      against Python `fnmatch` results on the boundary/ignore globs.
- [ ] **Lookaround regex.** The single `_token_in` lookbehind/lookahead; reimplement manually.
- [ ] **Sorting.** `sorted()` vs `sort.Strings`; both code-point lexicographic, but verify on tokens
      with `-`, `.`, `_`.
- [ ] **Markdown separator detection** and documentation-row skips must match `parse_md_tables`
      exactly.
- [ ] **Exit codes and message strings** are part of the contract; the differential harness asserts
      full strings, not substrings.
- [ ] **Trailing newlines / blank lines** in generated files; byte-exactness includes them.
- [ ] **Number formatting** (`%d`, rank maps) and int-vs-string YAML scalars.

## Rollback

- [ ] Any phase whose exit gate cannot be met without changing Python behavior: stop, record the
      divergence, decide whether Python has a latent bug (fix in both, re-golden) or Go is wrong
      (fix Go). Never "adjust" a golden to make Go pass.
- [ ] If a defect surfaces after `python-removed`: `git revert` to `pre-go-migration` or the last
      shadow-green tag; the tags exist for exactly this.
