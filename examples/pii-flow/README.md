# pii-flow: a Datalog sensitive-data-flow checker

This example is the reference implementation of machinery's
[external-checker contract](../../docs/external-checkers.md). Its rules are a
Datalog program, `rules.dl`, and Souffle 2.5 evaluates them inside a
digest-pinned OCI image with no ambient host runtime. The Python adapter beside
the rules only translates between machinery's JSON contracts and Souffle's
native files; it computes no part of the verdict. The checker enforces one
invariant machinery's built-in gates do not express: a sensitive-data-flow
property over the domain model's relationship graph.

## What it demonstrates

- **A real Datalog engine behind the contract.** `rules.dl` derives
  `tainted(E)` for every entity that holds a sensitive attribute and everything
  it flows to, except through a `redacted` node. Closed-world negation
  (`!redacted(F)`) makes redaction a taint boundary: nothing downstream of a
  redactor is ever tainted. `leak(E)` is derived only when a tainted entity is
  also a declared sink. The negation is stratified (`redacted` is an input,
  never derived), so the fixed point is unique and deterministic. Souffle
  computes it; `adapter.py` writes the five input relations as `.facts` files,
  runs `souffle --no-preprocessor -F <facts> -D <out> rules.dl`, and maps the
  two output relations, `leak.csv` and `tainted.csv`, back to evidence.
- **A complete runtime closure.** The local registry pins the image by
  manifest digest and its exact `linux/amd64` implementation, then declares
  `adapter.py` and `rules.dl` as exact read-only inputs. The image digest,
  platform, run and replay argv, mount names, and input bytes derive one closure
  digest carried by both the manifest and the evidence.
- **A replay that re-derives the verdict.** The registry's `verify` command is
  `adapter.py verify`: it re-runs Souffle over the committed projection and
  requires the committed evidence and its trace to be byte-identical to what
  the engine derives now.
- **The coverage-claim-plus-residuals mechanism.** The manifest claims all
  six `priv-*` invariants (`coverage.claim: ["priv-*"]`) but the checker
  only decides `priv-no-unredacted-export` from the flow graph. The other five
  are declared **residuals** with specific reasons: consent and minimality are
  process/design controls, while purpose validation, transform effectiveness,
  and export idempotency are runtime properties rather than relationship-graph
  properties. Coverage is a hard rule
  in machinery: a claimed invariant that is neither in the evidence nor a
  declared residual fails the gate. This example shows both halves of that
  rule working together on the same manifest.
- **A domain designed to pass.** The model is a straight-line pipeline,
  `DataSubject -> ProcessingActivity -> Redactor -> AnalyticsExport`, where
  the export sink is reachable only by passing through the redactor. Souffle
  derives an empty `leak` relation, so the verdict is `pass`, the coverage row
  says so explicitly, and the evidence carries zero findings. An empty output
  relation is a decided `pass`, not a coverage gap.

## A complete design, not a bare checker harness

The checker is the point of this example, but the design around it is
complete enough that the **full default gate suite passes with zero blocking findings**
(`machinery check examples/pii-flow/design`, no `--gate` narrowing). One warning remains by
design: `Gv-attest` reports `gt.conformance-test-shape: plan only; current implementation review
missing`, because this example ships no implementation, and a current-review claim can only be
discharged against one. Promoting warnings to failures therefore fails on that single line until an
implementation exists; every other gate is warning-free:

- the model carries a `DataSubject` lifecycle (`SubjectStatus`:
  Active/Erased) with `register`/`erase` actions and the
  `subject-erased-terminal` invariant, realized by
  `machines/DataSubject.machine.json` (Erased is final, so post-erasure
  processing is structurally impossible) and its named-unit matrix;
- `ARCHITECTURE.md` carries the Architecture Contract: the pipeline's only
  exit is the export adapter, the pipeline never talks to the analytics
  destination directly (a deny rule), and the export layer never reaches
  back into subject data, which is a checked `assert: no_path` claim over
  the allow-graph closure, not a prose note;
- Gc-carrier reconciles every invariant to a carrier: the flow invariant is
  carried by the checker's own coverage claim, the two process controls by
  its declared residuals, and the lifecycle invariant by `preserves` plus
  the machine matrix.

The split of labor is the lesson: the architecture asserts what a dependency
graph can prove (no reverse path), the machine asserts what structure can
prove (erasure is terminal), and the Datalog checker decides the one claim
neither can express (no sensitive attribute reaches the sink unredacted).

## File layout

```
examples/pii-flow/
  design/
    pii-flow.modelith.yaml               # the domain model (Modelith), with the DataSubject lifecycle
    workspace.dsl                        # the C4 model (Structurizr DSL)
    ARCHITECTURE.md                      # the Architecture Contract, incl. the no_path assertion
    BUILD.md                             # complete Phase 4 implementation and hard-TDD plan
    surfaces.yaml                        # complete Controller persona-to-interface ledger
    attestations.yaml                    # content-bound review evidence for every owed claim
    machines/
      DataSubject.machine.json           # the erasure lifecycle (Erased is final)
      DataSubject.matrix.md              # named-unit contracts + failure catalog
      DataSubject.oracle.md              # generated by `machinery oracle` (committed)
    formal/
      DataSubject.semantics.yaml         # strict control-flow-only lifecycle declaration
      Datasubject.tla                    # generated lifecycle control-flow proof
      Datasubject.cfg                    # TLC model configuration
    checkers/
      pii-flow.checker.yaml              # the tool-neutral manifest (committed)
      pii-flow/
        rules.dl                         # the Datalog program Souffle evaluates (committed)
        adapter.py                       # projection -> .facts -> souffle -> evidence (translation only)
        projection.json                  # generated by `machinery project` (committed)
        evidence.json                    # produced in the pinned image (committed)
        generated/
          souffle-outputs.json           # the trace: every output relation Souffle derived (committed)
  souffle-image/
    Dockerfile                           # the checker userspace: CPython 3.14.7 + Souffle 2.5, all inputs pinned
  checkers.local.example.yaml            # sample local registry
  README.md
```

`pii-flow.checker.yaml` names no binary and no engine; it declares only the
projection slice it needs (`model`, `invariants`, `relationships`), what it
claims, its residuals, and where its evidence lives. The `config` block
(`enforces_invariant`, `sensitive`, `sinks`, `redacted`) is opaque to
machinery and passed through verbatim to `adapter.py`, which turns it into the
`sensitive`, `sink`, and `redacted` fact relations. That is how the checker
supplies its own domain knowledge without machinery ever needing to understand
it.

The manifest stays on projection 1.0. The rules read entities, their
attributes, and the relationship graph, all of which the 1.0 `model` block
carries; projection 2.0 layers such as `machines` or `c4` would add inputs the
rules never join. The adapter accepts exactly the 1.0 three-layer projection
and rejects anything else (a `layers` object, a 2.0 schema, an extra key),
because deciding the invariant while silently ignoring a layer the manifest
asked for would be a verdict over a design the checker never read.

## The adapter fails closed

`adapter.py` exits non-zero with one diagnostic and writes no evidence when:

- `souffle` is not on the image's PATH (the diagnostic names it);
- Souffle exits non-zero (a syntax error in `rules.dl`, for example) or prints
  anything on an exit-zero run, since a dropped warning is a hidden one;
- a relation `rules.dl` declares with `.output` (`leak`, `tainted`) has no
  output file, or Souffle writes an output the adapter does not map;
- a fact value is empty or holds a tab, carriage return, or newline, which the
  tab-separated facts format cannot represent (stable ids never should);
- the projection is not exactly the 1.0 three-layer contract, or
  `config.enforces_invariant` names no invariant in the projection.

`cmd/machinery/pii_flow_souffle_test.go` exercises each of these in the pinned
image, together with a design whose sensitive data reaches the sink unredacted
(verdict `fail`, one blocking finding naming `entity:AnalyticsExport`).

## Why `machinery check --gate gk` is hermetic

The pure gate never starts Docker or an image and never shells out. It:

1. regenerates the projection from `pii-flow.modelith.yaml` and byte-matches
   it against the committed `projection.json`;
2. reads the committed `evidence.json` and checks that its `input_hash`
   equals the hash of that fresh projection and its `runtime_closure` equals
   the manifest's closure (the verdict was computed over exactly this design
   and checker userspace, not stale inputs);
3. checks `verdict == pass` and that every claimed `priv-*` invariant is
   either in `coverage` or a declared residual.

Because the projection and the evidence are both committed artifacts, this
check runs anywhere machinery runs, with no OCI engine or image
required. That is the split this example exists to show: the deterministic
half lives in `machinery check`; the engine half is a separate, explicit
step.

## Enabling the engine phase

The gate above only verifies that committed evidence is fresh and covers
what it claims; it does not start the OCI runtime. To regenerate the evidence,
or to have machinery re-run the checker and confirm the verdict was earned:

1. Provision the pinned image. No upstream registry publishes a Souffle image,
   so it is built from `souffle-image/Dockerfile`, whose base image, Souffle
   release, and two runtime libraries are all pinned by content:

   ```
   scripts/pii-flow-image.sh
   ```

   The script builds with a digest-pinned BuildKit (`docker buildx`,
   docker-container driver), `SOURCE_DATE_EPOCH=1742825974` (the Souffle 2.5
   release time), and rewritten layer timestamps, so the manifest digest is a
   pure function of the Dockerfile. It pushes the result to a throwaway
   loopback registry and pulls it back, because only a registry round trip
   gives the local engine the `RepoDigests` entry verify-checkers checks, and
   it fails if the rebuilt digest is not the pinned one:

   ```
   localhost:5959/machinery/pii-flow-souffle@sha256:51981e17aef416020a1faa778042473a45dda347eab30cbbd9648a264f6f0df7
   ```

   On an arm64 host the `linux/amd64` build and every later run execute under
   emulation (Rosetta or qemu). That is emulated reproduction of the pinned
   amd64 userspace, not native arm64 evidence. The digest does not depend on
   the host: see [Reproducibility](#reproducibility).

2. Run the engine phase with the sample registry in place. Its input
   `source` paths resolve against the registry file's directory, so they name
   the committed adapter and rules directly:

   ```
   machinery verify-checkers examples/pii-flow/design --registry examples/pii-flow/checkers.local.example.yaml
   ```

   To use the default location instead, copy the registry to
   `.machinery/checkers.local.yaml` and copy `adapter.py` and `rules.dl` to
   `.machinery/design/checkers/pii-flow/` beside it; the closure binds the
   same bytes either way. The verifier snapshots a regular, non-symlink engine
   executable: if the `docker` launcher on PATH is a symlink, replace
   `engine: [docker]` with the canonical executable path.

   verify-checkers regenerates the projection in memory, verifies the exact
   local image `RepoDigests` and `linux/amd64` identity, runs `adapter.py run`
   with the same explicit platform, `--pull=never`, no network, a read-only
   root, and only the declared read-only inputs, confirms the fresh evidence and
   trace match the committed ones, then runs `adapter.py verify` as the replay.

## Reproducibility

The image digest is the same on every host that runs the script, native
amd64 or arm64 under emulation. What makes it so is that the Dockerfile's
final stage executes nothing: `dpkg-deb`, `ldconfig`, and the `souffle`
smoke run all execute in the fetch stage, and the final stage only copies the
staged files (the souffle binary, the unpacked libgomp1 and libncurses6
payloads, and a regenerated `/etc/ld.so.cache`). An emulated process leaves
host state in whatever layer it runs in: the earlier Dockerfile ran `dpkg -i`
and `souffle --version` in the final stage, and under Rosetta that layer
gained an empty `/root/.cache/rosetta` directory (three extra tar entries,
`root`, `root/.cache`, `root/.cache/rosetta`, every other entry byte-identical),
so an arm64 Mac pinned `sha256:0fc676da...` while native amd64 hosts, hosted CI
included, built `sha256:a316c0bb...`. `TestPiiFlowImageFinalStageRunsNothing`
fails if a `RUN` returns to the final stage. The two libraries are unpacked
rather than installed through dpkg, so the image holds their files and
copyright notices but no dpkg database records for them.

Recorded builds of the current Dockerfile, each through
`scripts/pii-flow-image.sh` (`--no-cache`, fresh builder and registry, pinned
image removed first):

| Host | Build | Digest |
|---|---|---|
| amd64 Linux, Docker 29.8.0, native | 1 | `sha256:51981e17aef416020a1faa778042473a45dda347eab30cbbd9648a264f6f0df7` |
| amd64 Linux, Docker 29.8.0, native | 2 | `sha256:51981e17aef416020a1faa778042473a45dda347eab30cbbd9648a264f6f0df7` |
| arm64 macOS, Docker Desktop 29.8.0, emulated | 1 | `sha256:51981e17aef416020a1faa778042473a45dda347eab30cbbd9648a264f6f0df7` |
| arm64 macOS, Docker Desktop 29.8.0, emulated | 2 | `sha256:51981e17aef416020a1faa778042473a45dda347eab30cbbd9648a264f6f0df7` |

## Deriving `runtime_closure`

`checker.runtime_closure` in the manifest (and in the evidence) is not written
by hand. Machinery derives it from the registry: the image digest, the
platform, every `run` and `verify` argument, and each declared input's mount
name plus the sha256 of its bytes. When any of those change, run
`machinery verify-checkers` as above: it refuses to execute and names the
digest it derived (`local registry runtime closure sha256:... does not match
manifest checker.runtime_closure ...`). After reviewing the change, copy that
digest into the manifest, regenerate the evidence in the pinned image, and
re-run verify-checkers to confirm it reproduces. `TestPiiFlowRuntimeClosureGolden`
recomputes the same digest from the sample registry and fails if the manifest
or evidence disagree.

## Prerequisites

- Docker, or a command-compatible OCI engine whose executable is a regular,
  non-symlink file, with the buildx plugin to build the pinned image
- the pinned image provisioned locally for `linux/amd64` (step 1 above)

Neither is required to run `machinery check --gate gk`; both are required
only for `machinery verify-checkers`.
