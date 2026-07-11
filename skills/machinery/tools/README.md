# machinery tools: the CLI and the TLC shell wrappers

The toolchain is the single `machinery` Go binary (install it with the one-line installer, or
`go build ./cmd/machinery`). This directory holds the only pieces that are not the binary:
two shell wrappers around TLC, the TLA+ model checker, for environments that prefer the shell path.
`machinery verify-formal` embeds the same orchestration logic, so the wrappers are a convenience,
not a requirement.

## What lives here

- `tlc.sh <spec.tla>` runs TLC on one `.tla`/`.cfg` pair. On first use it fetches
  `tla2tools.jar` v1.7.4 from the TLA+ releases, verifies it against the pinned sha256
  (`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`), caches it under
  `~/.cache/machinery/`, and cleans TLC's `states/` working directory on exit. Override the pin
  with `TLA_TOOLS_VERSION` plus `TLA_TOOLS_SHA256` (both, deliberately). Needs Java 11+.
- `verify_formal.sh <design-dir>` regenerates the whole formal suite from source (via the
  `machinery` binary) into `design/formal/` and TLC-checks every `.tla`/`.cfg` pair through
  `tlc.sh`. A pass requires both a zero TLC exit code and the no-error line, so a Java crash, a
  download failure, or a TLC message change can never read as PASS.

## The CLI

One line per subcommand:

- `machinery lint <machines-dir>` structural lint of every `*.machine.json` (reachability, invoke
  error paths and timeouts, shadowed branches, event completeness, the enforced XState subset).
- `machinery oracle <machines-dir>` writes the canonical `<M>.oracle.md` transition oracle next to
  each machine (sequential test ids for humans, content-derived stable ids for tests). G3 byte-diffs
  the committed oracle against a fresh in-memory generation, so a stale or edited oracle is DRIFT.
- `machinery tla <machine.json> [out-dir]` generates the control-flow TLA+ model and `.cfg` for one
  machine (bounded retry counters, liveness, deadlock-freedom; assumptions printed in the header).
- `machinery alloy <design-dir> [out-dir]` reconciles every present relational annotation against
  the domain model and emits its generated proof artifacts: policy (`Policy.als` plus the
  authorization oracle), integrity (`Integrity.als`, including inverse exclusivity for `1:1` and
  `1:n` relationships), and isolation (`Isolation.als` plus the field-qualified tenant oracle).
  Reconciliation failure is a hard error; `verify-formal` runs the pinned Alloy analyzer, and the
  Gp/Gi/Gn gates byte-diff committed artifacts against fresh generation.
- `machinery refine <machine.json> <semantics.yaml> [out-dir]` reconciles a `<M>.semantics.yaml`
  annotation against the machine, then generates the data-refined model, the abstract contract, and
  the refinement proof. Patterns: `linear-lifecycle`, `terminal-lifecycle`, `saga` (state names come
  from the annotation, nothing is hardcoded to a domain). Reconciliation failure is a hard error.
- `machinery compose <composition.yaml> <coordinator.machine.json> [out-dir]` validates a
  `<name>.composition.yaml` against the coordinator machine, then generates the cross-aggregate
  composition (failures, per-obligation compensation, the FailedDirty stall) with its invariants.
- `machinery check <design-dir> [--impl <code-dir>] [--gate gm,gs,gp,gi,gn,g2,g3,gx,gb,g4,gt,g5]` the deterministic
  gate suite (Gm-transition on rebuild/hybrid contracts; Gs-surface on legacy surface ledgers;
  Gp/Gi/Gn relational gates; G2-c4,
  G3-machine, Gx-trace; Gb-plan on build plan structure, artifact-activated on `design/BUILD.md`;
  G4-import; Gt-tests on oracle ids in the test suite, runs only with `--impl`; G5-pack on
  decomposed designs). Gates fail on absence; every gate prints a `checked:`
  line. Exit is non-zero on any ERROR or DRIFT.
- `machinery pack generate <parent-design>` emits the frozen per-subsystem contract packs
  (`design/packs/<id>.pack/`) from `decomposition.yaml`: the owned domain slice, the boundary event
  rows, the contract machine plus its TLA+ module, the delegated invariants, and a content hash.
- `machinery pack refine <child-design>` reconciles the child's `packmap.yaml` against the pack's
  contract machine and the child's exposed machine, then generates the pack-refinement proof that
  `verify-formal` TLC-checks. Reconciliation failure is a hard error.
- `machinery scale <design>` measures a design's size (stateful components, bounded contexts,
  synthesis input) and recommends sharding or recursive decomposition.
- `machinery verify-formal <design-dir>` regenerates the tla/refine/compose specs into
  `design/formal/` and TLC-checks every `.tla`/`.cfg` pair. It fails on generator errors and on
  zero pairs, so an empty formal directory can never read as a green suite. `--gen-only`
  regenerates without running TLC (no Java needed): the freshness half of the check for
  Java-free environments such as a nightly regen gate.
- `machinery doctor` checks dependencies and install status.
- `machinery preflight` the same check, for use before a design session.
- `machinery version` prints the build version.

The binary is the whole generation and deterministic-gate toolchain: no interpreter or runtime
dependencies. Java 11+ is needed only for solver execution by `verify-formal` (TLC and, when a
relational annotation exists, Alloy) and the TLC shell wrappers.

## Two design rules govern the gate suite

1. **Absence is failure.** A missing artifact, an unparseable section, or a gate that found nothing
   to check is an ERROR, never a silent pass. "Clean" is always distinguishable from "checked
   nothing".
2. **Generate, do not co-author.** Derivable artifacts are generated (the oracle, the TLA+ models),
   and the gates verify the committed copies match a fresh generation, so staleness is DRIFT, not an
   assumption.

## The formal suite

`make verify-formal` runs the full ladder across the example designs: 26 TLA+ proofs, all green
(8 in `examples/go-crm`, 8 in `examples/fulfillment`, 6 in `examples/portfolio-engine`, and 4 in
`examples/checkout-split`, two per child incl. the contract-refinement proofs), regenerated
from source on every run. The `*.semantics.yaml` and `*.composition.yaml` annotations are trust
points, so they are reconciled against the machine before anything is emitted: a design change that
invalidates an annotation fails generation rather than proving a stale twin. What remains assumed
after reconciliation is printed into each generated header.
