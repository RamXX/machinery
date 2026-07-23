# External checkers: plugging your own deterministic analyzer into the gate suite

This guide is for someone who has a checker machinery does not ship, and wants it to gate every
design with the same discipline as the built-in gates. The checker can be anything that reads a
description of the design and returns a verdict: a SAST or AST linter, a Datalog closed-world
solver, a probabilistic graph reasoner, a units/dimensions checker, or a rule engine that encodes a
statute or a control catalog. Machinery treats them all the same way, through one small contract.

The design goal is a boundary you can build against without changing your tool and without machinery
ever learning your tool's internals. Machinery defines two neutral JSON schemas and a resolution
registry; you supply a manifest and (usually) a one-file adapter. Nothing about any specific engine
lives in machinery.

A complete, runnable reference lives at [`examples/pii-flow/`](../examples/pii-flow/): a Datalog
closed-world checker over a small data-flow model, wired end to end. If you want to see the contract
in a real design rather than in the abstract, start there.

If you have not read them yet, the sibling built-in checkers are the
[policy layer](policy-layer.md), the [integrity layer](integrity-layer.md), and the
[isolation layer](isolation-layer.md). External checkers reuse their exact mechanics (opt-in on an
artifact, coverage as a hard rule, freshness by byte-match, a pure gate plus a separate solver
phase). If those guides make sense, this one will too.

## The one idea

Every machinery gate already splits into two halves:

- a **pure, deterministic half** that runs in `machinery check` with no external runtime: it
  reconciles an annotation against the model, checks coverage, and byte-matches committed artifacts;
- an **engine half** that runs in a separate command (`machinery verify-formal`) and shells out to
  the actual solver (TLC, the Alloy analyzer).

`machinery check` never runs a solver. It proves the design *accounts for* a check and that the
committed *evidence is fresh and bound to this exact design*. The solver runs elsewhere.

External checkers ride that same split:

```
                       machinery owns                          you own
                 (open source, tool-neutral)           (your repo, your tool)

  design/  --->  machinery project (Go, sorted,   --->  projection.json
                  byte-stable)                              |
                                                              v  (your adapter + your engine)
  gk gate  <---  machinery check --gate gk:        <---  evidence.json
  (pure)         input_hash binds, coverage holds,
                 verdict is pass
                                                              |
  verify-  --->  machinery verify-checkers:        --->  your engine's own
  checkers       resolves the binary via the             run + verify
                 registry, re-runs the adapter,
                 compares to committed evidence
```

The pure gate is `gk` (external checkers), one instance per manifest (`Gk-<id>`). The engine phase is
`machinery verify-checkers`, the sibling of `machinery verify-formal`. The next section spells out
the three commands you actually run.

## The three commands

- **`machinery project <design>`** generates and writes the committed projection for every
  `design/checkers/*.checker.yaml` manifest, one file each, to the path its `evidence.projection_out`
  names. This is the write side of the contract: run it, then `git commit` the result, before your
  adapter ever consumes the projection. It needs no registry and no engine.
- **`machinery check <design> --gate gk`** is the pure, hermetic gate (`Gk-<id>` per checker), one of
  the names in `--gate`'s vocabulary: `gm,gs,gp,gi,gn,g2,g3,gx,gk,gb,g4,gt,g5`. It never runs an
  engine and never touches the registry; it reconciles the manifest, byte-matches the committed
  projection, and checks that the committed evidence binds and covers the claim.
- **`machinery verify-checkers <design> [--registry <path>] [--checker <id>]`** is the engine phase,
  the sibling of `machinery verify-formal`. It resolves each checker's binary through the registry,
  re-runs the adapter, and confirms the freshly produced evidence is reproducible: verdict,
  `input_hash`, and coverage all match the committed copy. It then runs the optional replay `verify`
  command. It requires the committed projection to already be present, so run `machinery project`
  first. `--checker` scopes a run to one id; `--registry` overrides the default registry path,
  `.machinery/checkers.local.yaml` in the current working directory.

## The contract on one screen

| Piece | Who writes it | Where it lives | What it is |
|---|---|---|---|
| `projection.schema.json` | machinery | [`schemas/projection.schema.json`](../schemas/projection.schema.json) | the canonical, deterministic slice of the design machinery hands you |
| `evidence.schema.json` | machinery | [`schemas/evidence.schema.json`](../schemas/evidence.schema.json) | the verdict + coverage + provenance you hand back |
| the manifest | you | `design/checkers/<id>.checker.yaml` (committed, tool-neutral) | which slice you need, what you cover, where your evidence sits |
| the registry | you | `.machinery/checkers.local.yaml` (repo root, git-ignored) | which binary fulfills `<id>`, and its verify command |
| the adapter | you | anywhere your registry command points | maps projection -> your input, and your output -> evidence.json |

Machinery reads the two schemas, the manifest, and the evidence. It resolves the binary through the
registry. It never reads your adapter, your engine, or your native output format. That separation is
the whole point: your tool emits what it emits, and the adapter is the only thing that has to know
both languages.

## The two schemas

### Projection (machinery to checker)

A single JSON object, one per checker, carrying exactly the layers the manifest asked for. **v1
projects three layers only: `model` (entities and attributes), `invariants`, and `relationships`.**
Requesting `actions`, `scenarios`, `machines`, `c4`, or `oracles` in a manifest's `projection.include`
is not silently dropped: both `machinery project` and the `gk` gate fail loudly, because a checker
that believes it received a layer it never got is worse than a checker that never ran. Those five
names are reserved in the schema for a later revision; write your manifest against `model`,
`invariants`, and `relationships` today.

Every element is keyed by a stable id, so a verdict binds to an identity that survives renames, not
to a line number. v1 emits natural composite ids derived straight from your Modelith names:

| Element | Stable id shape | Example |
|---|---|---|
| entity | `entity:<Name>` | `entity:DataSubject` |
| attribute | `attr:<Entity>.<name>` | `attr:DataSubject.email` |
| invariant | `inv:<id>` | `inv:priv-no-unredacted-export` |
| relationship | `rel:<From>-><To>:<card>` | `rel:ProcessingActivity->Redactor:1:1` |

A later revision may switch to content-derived ids without changing the contract, since evidence
binds by whatever id the projection emits, not by a fixed format. Note the distinction that matters
for coverage: a manifest's `coverage.claim` globs over the short, human invariant `id` (`priv-*`
matches `priv-consent-required`), while evidence `coverage` rows reference the full `stable_id` the
projection assigns it (`inv:priv-consent-required`), never the bare id.

Two more things v1 is upfront about:

- An invariant's `text` field is Modelith's invariant `statement`, verbatim.
- `polarity` (positive / negative) is reserved on the invariant shape and always absent in v1:
  Modelith does not yet carry polarity, so there is nothing to project.

Full field reference: [`schemas/projection.schema.json`](../schemas/projection.schema.json).

Three things about the projection's determinism matter to you:

- Machinery emits the projection with sorted keys and arrays sorted by stable_id, byte-for-byte
  reproducible. The committed copy is byte-matched on every `check`; a stale copy is DRIFT.
- Machinery computes `input_hash = sha256(canonical projection bytes, excluding the generated
  object)`. That hash is the binding your evidence must echo. The `generated` object (timestamps,
  versions) is excluded so a fresh render is never spurious drift.
- `machinery project` also mirrors that same hash into the committed file, at
  `projection["generated"]["input_hash"]`, purely as a convenience: your adapter reads it and echoes
  it into `evidence.json` without ever reimplementing machinery's canonicalization. The gate never
  trusts this mirror. `machinery check --gate gk` and `machinery verify-checkers` always recompute
  `input_hash` from the projection bytes themselves; the mirror being present, absent, or wrong
  changes nothing about what they check.

### Evidence (checker to machinery)

A single JSON object you write back. The required core is small:

```json
{
  "evidence_schema": "1.0",
  "checker": { "id": "privacy-cl", "version": "2026.07.1" },
  "input_hash": "sha256:...",
  "verdict": "pass",
  "coverage": [
    { "element": "inv:priv-consent-required", "verdict": "pass" },
    { "element": "inv:priv-retention-bounded", "verdict": "pass" }
  ]
}
```

`input_hash` is the binding. `coverage` is the list of elements you actually decided, which
machinery cross-checks against your claim; each `element` is the projection's full `stable_id`, not
the bare invariant id the manifest's `claim` glob matched. Optional fields carry findings, a detached
`input_signature` machinery can verify in the pure phase, an opaque `attestation` block for your
engine's own provenance, and a `trace_ref` for replay. Full reference:
[`schemas/evidence.schema.json`](../schemas/evidence.schema.json).

## The manifest (tool-neutral, committed)

`design/checkers/<id>.checker.yaml` declares the *contract only*. It names no binary, so the design
stays portable and nothing about your tool leaks into a committed artifact.

```yaml
checker:
  id: privacy-cl                       # opaque label; must match evidence.checker.id
  description: "Static data-protection check over data-handling invariants"

# The slice you need. Machinery projects exactly this, in canonical order, and owns its freshness.
# v1 supports model, invariants, and relationships only. Naming actions, scenarios, machines, c4, or
# oracles here fails machinery project and the gate loudly; they are reserved for a later revision.
projection:
  include: [model, invariants, relationships]
  requires: [gx]                        # gates whose validated artifacts this projection assumes; informational in v1, see "Order" below

# What you are RESPONSIBLE for. Coverage is a hard rule.
coverage:
  claim: ["priv-*"]                     # globs over the human invariant id, e.g. priv-consent-required
  residuals:
    - id: priv-cross-border-transfer
      reason: "Operational control (DPA in place); not design-derivable"

# Opaque to machinery: your own domain knowledge, passed through untouched. This is how a checker
# supplies things the generic projection has no vocabulary for (which attributes are sensitive, which
# entities are export sinks) without machinery ever having to model them. During
# `machinery verify-checkers`, this block is serialized to a temp JSON file and its path is handed to
# your adapter as the {config} token.
config:
  sensitive: ["attr:DataSubject.email", "attr:DataSubject.nationalId"]
  sinks: ["entity:AnalyticsExport"]

# Where your evidence sits and how it binds. Both paths are relative to the design directory.
evidence:
  projection_out: checkers/privacy-cl/projection.json
  evidence_in: checkers/privacy-cl/evidence.json

# Optional: this checker emits an oracle-shaped decision table the implementation must conform to.
emits_oracle: false
```

`claim` is the machine that gives you "every feature, every time." When a new invariant matching
`priv-*` appears on a new entity, it enters the projection, `input_hash` changes, and stale evidence
that does not cover it fails `gk`. You cannot silently skip a control on a new feature.

v1 has no manifest-level `signature` field. A checker that signs its own artifacts carries that
provenance through the opaque `attestation` block on evidence instead (see "Determinism across an
opaque engine" and "Writing an adapter" below); `input_signature` remains reserved on the evidence
schema for a directly-verified mode machinery does not yet implement.

## The registry (local, git-ignored, resolves the binary)

`.machinery/checkers.local.yaml` at the repo root binds each `id` to a command. It is machine-local,
git-ignored config, not a design artifact: the design never names your tool, and machinery core never
does either. `machinery verify-checkers` reads it from the current working directory by default; pass
`--registry <path>` to point at a different file (a shared CI config, a per-environment variant).

```yaml
checkers:
  <checker-id>:
    run: ["<cmd>", "<args-with-tokens>"]   # required; the adapter, writes fresh evidence to {out}
    verify: ["<cmd>", "<args>"]            # optional; replays / re-verifies the engine's own trace
    timeout: "120s"                        # optional; default 120s
```

Tokens substituted into `run` and `verify` arguments:

| Token | Substituted with |
|---|---|
| `{projection}` | path to the committed projection (`evidence.projection_out`) |
| `{config}` | path to a temp JSON file holding the manifest's opaque `config` block |
| `{manifest}` | path to the checker manifest itself |
| `{out}` | path your adapter must write fresh evidence to |
| `{design}` | the design directory |

Concretely, for the manifest above:

```yaml
checkers:
  privacy-cl:
    run: ["./tools/privacy-cl-adapter.sh", "{projection}", "{config}", "{out}"]
    verify: ["./tools/privacy-cl-verify.sh", "{out}"]
    timeout: "120s"
```

`run` is required and must write fresh evidence to `{out}`; `verify` is optional and delegated
entirely to your engine's own replay or verify subcommand. A missing or unresolvable binary is
reported by `machinery verify-checkers` itself as an ERROR before it attempts to run anything; it is
never silently skipped.

`machinery doctor` reads this registry too: when it is present in the current directory, doctor lists
each configured checker and probes that its `run` binary is on PATH, so a missing engine surfaces up
front rather than as a confusing `verify-checkers` failure. With no registry present, doctor's output
is unchanged.

## The two phases in operation

### `machinery check --gate gk` (pure, hermetic, runs in any CI)

For each manifest under `design/checkers/`, in id order, machinery:

1. parses the manifest and reconciles it against the model: every claimed id and every residual id
   must resolve to a real invariant, or ERROR; a residual with no reason is also an ERROR (a waiver
   without a reason is not a waiver);
2. regenerates the projection and byte-matches the committed `projection_out`; a missing or stale
   copy is DRIFT, pointing you at `machinery project`;
3. reads the committed `evidence_in` and checks its `checker.id` matches the manifest; absence of the
   file is an ERROR, never a silent pass;
4. checks `input_hash == sha256(fresh projection)`; a mismatch is DRIFT (the verdict was computed
   over a different design);
5. checks `verdict == pass` and that every claimed invariant is in `coverage` or a declared residual;
   a coverage gap is ERROR, and a `fail` verdict (or a `pass` that still carries a blocking finding)
   surfaces as ERROR with the blocking findings shown;
6. prints `layers projected`, `invariants claimed`, `residuals`, `evidence bound to design`, and
   `elements covered` counts.

Notice what step 5 does *not* do: it never re-derives the verdict. It proves the design projects to
exactly this input and that fresh, bound evidence asserts pass over that input. That is fully
deterministic even when the engine behind it is not, which is what lets an opaque or probabilistic
checker live inside machinery's determinism guarantee.

Absence fails: a manifest with no committed evidence is an ERROR, never a silent pass, exactly like
every other gate.

### `machinery verify-checkers <design>` (the engine phase, sibling of verify-formal)

This is where the engine actually runs. It requires the committed projection to already be present
(run `machinery project` first); it does not itself re-run the `gk` gate, keeping the phases separate
exactly as `verify-formal` does, so running `machinery check --gate gk` first is the discipline that
keeps an engine from burning time on a design that has not passed the pure gate. Pass `--checker <id>`
to scope a run to one checker, or `--registry <path>` to point at a registry other than the default
`.machinery/checkers.local.yaml`. For each checker it:

1. resolves the binary through the registry;
2. re-runs the adapter (the registry's `run` command) against the committed projection and the
   manifest's `config`, in a sandbox with the declared timeout, writing fresh evidence to `{out}`;
3. confirms the freshly produced evidence is reproducible against the committed copy: verdict,
   `input_hash`, and coverage all match;
4. runs the registry's `verify` command, when declared, delegating replay and signature checking to
   your engine's own verifier. Machinery checks its exit code and that the re-derived verdict agrees;
   it does not parse your native trace.

The split keeps `check` dependency-free (no Rust, Python, Java, or your engine on the CI box) while
`verify-checkers` runs where your runtime is available, exactly as `verify-formal` needs Java only
where the proofs run.

## Determinism across an opaque engine

Machinery's guarantee is that `check` is reproducible. Three things carry it across a boundary it
cannot see through:

- **Input determinism.** The projection is a pure function of the design, generated by machinery and
  byte-matched. Your engine cannot read the design any other way.
- **Verdict binding.** `input_hash` ties the committed verdict to that exact input. A verdict over a
  changed design cannot pass.
- **Re-derivation elsewhere.** `machinery verify-checkers` re-runs or replays the engine and compares.

For a **probabilistic** checker (converging intervals over a graph, for example), determinism is
recovered at the verdict layer: the manifest declares the thresholded proposition ("every
privacy-relevant node converges to lower bound at least 0.95"), your evidence carries the converged
intervals in the trace, and replay checks intervals-against-threshold deterministically even though
the inference used floats. A non-reproducible engine still yields a reproducible *verdict*.

For a checker that **signs its own native artifacts** and cannot be modified, you do not try to make
it sign machinery's `input_hash`. You leave `input_signature` absent, put a reference to the signed
native artifacts in the opaque `attestation` block, and let the registry `verify` command run the
engine's own verify/replay in the external phase. The pure phase still gets its hash binding; the
cryptographic provenance is confirmed where the engine runs.

## Writing an adapter for a tool you cannot change

This is the common case: a mature engine that emits its own report or its own signed, replayable
artifacts. You never modify it. You write one script (the registry `run` command) that speaks both
sides:

1. read the machinery projection at `{projection}` and, if your manifest declares a `config` block,
   the temp JSON machinery wrote for it at `{config}`;
2. transform it into your engine's native input (its facts, its claim, its graph), using `config` for
   whatever domain knowledge the projection itself does not carry;
3. run the engine unchanged;
4. read the engine's native output (a SARIF file, a decision, a converged graph, a signed trace);
5. write `evidence.json` at `{out}`: set `input_hash` to `projection["generated"]["input_hash"]`, the
   mirror `machinery project` wrote for exactly this reason, so you never reimplement machinery's
   canonicalization; map the native verdict to `pass`/`fail`; map decided elements into `coverage` by
   stable id; map issues into `findings`; and for a signing engine put its artifact references under
   `attestation` and point `trace_ref` at its native trace.

The `verify` command is usually a two-liner that calls the engine's own verify or replay subcommand
on the native artifacts the adapter preserved. Machinery only reads its exit code and the re-derived
verdict.

Everything tool-specific lives in those two scripts. Machinery, the manifest, and the committed
design remain neutral.

## Worked examples

### The reference: a Datalog sensitive-data-flow checker

[`examples/pii-flow/`](../examples/pii-flow/) is a complete checker wired into a small design, worth
reading end to end rather than taking on faith:

- **the model**, `examples/pii-flow/design/pii-flow.modelith.yaml`: a `DataSubject` who supplies
  personal data, a `ProcessingActivity` that handles it, a `Redactor` it must pass through, and an
  `AnalyticsExport` sink. The invariant under test is `priv-no-unredacted-export`: no sensitive
  attribute reaches the sink without passing through redaction.
- **the manifest**, `examples/pii-flow/design/checkers/pii-flow.checker.yaml`: claims `priv-*`,
  declares `priv-consent-required` and `priv-minimal-collection` as residuals (consent and
  collection-minimality are not decidable from a static flow graph), and carries a `config` block
  naming which attributes are sensitive and which entities are the sink and the redactor, so the
  Datalog program never has to guess at domain knowledge the projection does not carry.
- **the rules**, `examples/pii-flow/design/checkers/pii-flow/rules.dl`: a Soufflé Datalog program.
  Taint propagates along `flows` edges from any entity holding a `sensitive` attribute; a `redacted`
  entity blocks propagation into it using closed-world negation (`tainted(F) :- tainted(E), flows(E,
  F), !redacted(F).`), so anything downstream of the `Redactor` is clean; a `leak` fires when tainted
  data reaches a `sink`.
- **the adapter**, `examples/pii-flow/design/checkers/pii-flow/adapter.py`: reads `{projection}` and
  `{config}`, lowers the projected entities and relationships into Soufflé EDB facts using the
  `config` block's sensitive/sink/redactor lists, runs `souffle` against `rules.dl`, and maps an empty
  `leak` relation to a `pass` (with `inv:priv-no-unredacted-export` in `coverage`) or a non-empty one
  to a `fail` with a blocking finding per leaking element.
- **the committed outputs**, `.../pii-flow/projection.json` and `.../pii-flow/evidence.json`: the
  real `machinery project` output and the adapter's real evidence for this design, so you can see an
  actual `input_hash` binding a real projection to a real verdict rather than an abstract one.
- **a sample registry entry**, `checkers.local.example.yaml`: copy it to
  `.machinery/checkers.local.yaml` and point it at your local `souffle` and Python to run
  `machinery verify-checkers` against the example yourself.

This is the shape every checker in this guide follows; the rest of this section covers short
variants where the mapping differs.

### Short conceptual variants

**An AST or SAST linter (deterministic, unsigned) -- the simplest case.** Manifest claims the
invariants your rules cover; `include: [model, invariants]`. The adapter renders the projection into
whatever your linter reads, runs it, and maps results: no issues implies `verdict: pass` with every
claimed invariant in `coverage`; each issue becomes a `blocking` finding and flips the covered
element to `fail`. No `config`, no trace. `verify` re-runs the linter.

**A probabilistic graph reasoner.** `include: [model, invariants, relationships]`, plus a declared
threshold carried in `config`. The adapter builds the graph, runs inference to convergence, and
writes the converged intervals into the trace. The verdict is the threshold test over those
intervals. `verify` replays the trace against the threshold rather than re-running inference, so a
float-sensitive engine still passes deterministically.

**A proprietary engine that signs its own artifacts, which you cannot modify.** The pattern is
covered in full above, in "Writing an adapter for a tool you cannot change" and "Determinism across
an opaque engine": the adapter maps the projection into the engine's native input, runs it unchanged,
and writes `evidence.json` with `attestation` referencing the engine's own signed decision and
`trace_ref` at its native trace; `input_signature` stays absent. Machinery gates the binding and
coverage in the pure phase; the registry's `verify` command confirms the engine's own provenance in
the external phase, having parsed none of the engine's formats.

## Failure semantics (the same everywhere)

| Situation | Result |
|---|---|
| manifest present, no committed evidence | ERROR (absence is failure) |
| projection stale or missing vs a fresh render | DRIFT |
| `input_hash` does not match the fresh projection | DRIFT (verdict over a different design) |
| a claimed element is neither covered nor a residual | ERROR (coverage hole) |
| `verdict: fail`, or a `pass` carrying a blocking finding | ERROR, with the blocking findings surfaced |
| `projection.include` names a layer v1 does not project | ERROR, from `machinery project` and from the gate; never a silent omission |
| registry entry or engine binary missing (`verify-checkers`) | ERROR, reported before anything is run |
| schema or machinery version differs from what's running | non-blocking note (skew); regenerate when convenient |

## Order, and why gk sits after Gx

`gk` runs after `Gx-trace` and before `Gb-plan`. That placement is chosen for correctness and costs
nothing:

- **Correctness.** By the time `gk` runs, the model (`Gx` and its upstream), the C4 (`G2`), the
  machines (`G3`), and any relational layers (`Gp/Gi/Gn`) have all passed their gates. A checker
  therefore projects a design whose upstream layers are already validated; it never renders a verdict
  over an inconsistent model. A manifest's `requires` list makes the dependency explicit and lets a
  checker demand only the layers it reads.
- **Efficiency.** The pure `gk` phase is a hash comparison and a handful of field reads; placing it
  late is free. The expensive work, running the engine, lives in `verify-checkers`, a separate command
  you run after `check` is green, so no engine burns time on an invalid design. Multiple checkers run
  in id order for stable output and are independent, so the engine phase can run them in parallel.

## Shipping checklist

1. Write `design/checkers/<id>.checker.yaml`: the slice you need (v1: `model`, `invariants`,
   `relationships`), what you claim, your residuals, any `config` your adapter needs, and the
   evidence paths.
2. Add `.machinery/checkers.local.yaml` (git-ignored): bind `<id>` to your `run` command and, if you
   have one, your `verify` command, using the tokens above.
3. Write the adapter: `{projection}` and `{config}` in, `evidence.json` out at `{out}`; preserve
   native artifacts for `verify`.
4. Run `machinery project <design>` and commit the generated `projection.json`.
5. Produce `evidence.json`, either by running your adapter directly or via `machinery
   verify-checkers`, and commit it.
6. Run `machinery check <design> --gate gk` to confirm the pure gate is green: hermetic, byte-matched,
   bound, covered.
7. Run `machinery verify-checkers <design>` where your engine is available; wire it into CI next to
   `verify-formal`.
8. If the checker emits an oracle-shaped decision table, set `emits_oracle: true` and add the
   conformance test to `BUILD.md`, exactly as the policy oracle does, to close design to code.
