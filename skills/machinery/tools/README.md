# machinery check: deterministic verification gates

Pure static analysis over a machinery design (and, with `--impl`, the code). No LLM.
These are the hard symbolic checks that make correctness not depend on an agent getting
every cross-reference right. Two design rules govern the whole suite:

1. **Absence is failure.** A missing artifact, an unparseable section, or a gate that
   found nothing to check is an ERROR, never a silent pass. Every gate prints a
   `checked:` line with counts of what it actually verified, so "clean" is always
   distinguishable from "checked nothing".
2. **Generate, do not co-author.** Derivable artifacts are generated (the oracle, the
   TLA+ models), and the gates verify the committed copies are byte-identical to a
   fresh generation, so staleness is DRIFT, not an assumption.

```sh
python3 machinery_check.py <design-dir> [--impl <code-dir>] [--gate g2,g3,gx,g4]
```

Exit is non-zero on any ERROR or DRIFT. `warn` and `note` do not fail the gate. Use
`--gate` to run the subset that applies mid-phase; the default runs everything.
Requires PyYAML (`uv run --with pyyaml -- python3 ...` or have it installed).

## Gates

| Gate | What it checks | Kills |
|---|---|---|
| G2-c4 | The Architecture Contract (v2: `element`, `externals`, `ignore`) parses from its marked heading; boundary ids are unique and bind to `workspace.dsl` elements; rules reference declared boundaries and never contradict; every infrastructure dependency (contract externals plus DSL elements tagged Database/Queue/External) has a backticked mitigation row. | drift between the C4 model, its contract, and the failure postures |
| G3-machine | Full structural lint per machine (targets resolve unambiguously, reachability, no dead ends, invoke has onError + after, no shadowed branches, guarded-always exhaustiveness via `_exhaustive`, event completeness on resting states via `_ignores`, only the supported XState subset). The committed oracle must be byte-identical to a fresh generation. The hand matrix, if it has a transition table, must reconcile row by row in BOTH directions (source, trigger, guard shape, target, actions). Every guard, action, and actor fired by the machine must have a named-unit contract row. | malformed machines; machine-vs-oracle staleness; machine-vs-matrix drift; unspecified units |
| Gx-trace | Machine-to-model binding is explicit (filename matches an entity, `_lifecycle_of`, or `_role: operational`; anything else is an error). For lifecycle machines: TitleCase states are exactly the entity's lifecycle-enum values (both directions), events are Modelith actions of that entity. Every entity with a lifecycle enum has a machine; every backticked placement-table component has a machine or a waiver. Every invariant id appears (whole-token) in a matrix or BUILD.md table cell; backticked kebab references in maps-to columns must exist in the model (typo catch). | silent drift between the what, the how, and the behavior |
| G4-import | (needs `--impl`) Maps each source file to a boundary via the `code` globs (unmapped source is an error unless contract-`ignore`d; test files are skipped and counted). Extracts imports for Go (single and block form, module name from go.mod), Python, TypeScript/JavaScript, Elixir (via boundary `modules:`), and Rust; resolves externals via the contract `externals` map; enforces `exposes` (importing non-exposed internals is an error); any denied or undeclared cross-boundary edge is an error. | boundary erosion in code |

Known out of scope for G4, by construction: coupling through shared database tables or
message-bus topics is invisible to import analysis. The event-contract table in
ARCHITECTURE.md is the governing artifact for those seams (see the c4 reference).

## The machine IR

`machine_lint.py` is both the standalone linter and the shared IR: `walk_states`,
`transitions_of` (on / after / always / state-level onDone / invoke onDone / onError),
`actions_of` (string and `{type: ...}` action objects), plus the matrix parser and the
structural reconciler. Every other tool (oracle_gen, tla_gen, refine_gen, machinery_check)
consumes this one IR, so the gates and the formal layer cannot diverge on how a machine is
read. Unsupported XState constructs (parallel, history, root-level `on`, array targets,
unknown keys) are hard errors, never silently narrowed.

## Formal verification

The ladder, and what each rung actually proves:

- **Rung 1, generation** (`oracle_gen.py`): the transition oracle is generated from the
  machine, with a sequential test id for humans and a content-derived STABLE id for tests
  (row numbers renumber on the first design change; stable ids survive unrelated edits).
  G3 diffs the committed oracle against a fresh generation.
- **Rung 3, control flow** (`tla_gen.py` + `tlc.sh`): a TLA+ model per machine, every
  retry loop with its own bounded counter. TLC exhaustively checks `Live_OverlayResolves`
  (no unbounded retry, nothing stuck half-done) and deadlock-freedom; `TypeOK` is a
  regression lock. Every generated module carries an ASSUMPTIONS block: guards are erased
  (sound for safety; liveness is conditional on guard-list exhaustiveness, which the lint
  discharges via fallback-or-`_exhaustive`), invokes resolve exactly once, timers fire,
  single instance, no data. `tlc.sh` runs a pinned, checksum-verified `tla2tools.jar`
  (needs Java 11+).
- **Rungs 2 and 4, data and composition** (`refine_gen.py`, `compose_gen.py`): the
  `<M>.semantics.yaml` and `<C>.composition.yaml` annotations are trust points, so they
  are RECONCILED against the machine before anything is emitted: states, transition
  structure, failure routing, and step order must match the machine JSON exactly, or
  generation fails. refine_gen emits the data-refined model (domain invariants such as
  stage-forward and no-silent-loss, with saga compensation modeled per obligation so
  partial compensation is representable), the abstract contract, and a TLC-checked
  refinement. compose_gen emits the cross-aggregate composition over the coordinator's
  real branching (failures, per-obligation compensation, the FailedDirty stall) with an
  auto-generated clean-compensation invariant. `System.tla` (go-crm) INSTANCEs the
  generated contract module, so the assumption a caller makes is the same module the
  refinement proves, and TLC additionally checks the composition satisfies the contract
  spec. What remains assumed after reconciliation is printed into each generated header.
- `verify_formal.sh` regenerates and TLC-checks the whole suite from source
  (`make verify-formal`, sixteen proofs across the two examples). A pass requires both a
  zero TLC exit code and the no-error line, and failures print the TLC output.

## Tools

- `machine_lint.py` structural machine linter and the shared IR.
- `machinery_check.py` the full deterministic gate suite.
- `oracle_gen.py` rung 1: generate the transition oracle (stable ids) from the machine JSON.
- `tla_gen.py` + `tlc.sh` rung 3: generate and model-check a control-flow TLA+ model.
- `refine_gen.py` rung 2: reconcile a `*.semantics.yaml` against the machine, then generate
  the data-refined model, abstract contract, and refinement proof (linear-lifecycle, saga).
- `compose_gen.py` rung 4: validate a `*.composition.yaml` against the coordinator machine,
  then generate the branching composition that checks cross-aggregate invariants.
- `verify_formal.sh` regenerate and TLC-check everything for a design (`make verify-formal`).

The suite itself is tested: `make test` runs the pytest suite in `tests/`, which encodes
every vacuity and drift experiment from the design review as a permanent regression.
