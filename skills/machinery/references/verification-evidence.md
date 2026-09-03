# Verification and evidence

Read this reference when a design uses formal annotations, external checkers,
attestations, milestone closure, or implementation import checking. These are
source formats: do not infer their shape from generated artifacts or inspect
Machinery source code.

## Formal annotations

Every lifecycle machine has `formal/<Machine>.semantics.yaml`. Choose exactly
one pattern. A machine that has no honest data-refinement claim still needs the
explicit control-flow declaration:

```yaml
machine: evaluation
pattern: control-flow-only
reason: The machine has no persistent data transition beyond its checked control flow.
```

The other supported shapes are:

```yaml
machine: deal
pattern: linear-lifecycle
stages: [Lead, Qualified, Proposal, Negotiation]
win_stage: Won
lose_stage: Lost
reopen_to: Negotiation
close_date_on: Won
advance_event: advanceStage
win_event: win
lose_event: lose
reopen_event: reopen
max_retries: 3
```

```yaml
machine: RecommendationRun
pattern: terminal-lifecycle
phases: [Collecting, Optimizing]
success_terminal: Ready
failure_terminals: [Failed]
retry: {state: collectRetry, serves: Collecting}
success_flag: portfolioReady
max_retries: 3
```

```yaml
machine: fulfillmentSaga
pattern: saga
states: [Reserving, Paying, Shipping]
obligations:
  Reserving: {sets: reserved, undo: released}
  Paying: {sets: captured, undo: refunded}
  Shipping: {sets: shipped}
max_retries: 3
```

Cross-aggregate coordination uses a separate composition source:

```yaml
composition: checkout
coordinator: FulfillmentSaga
aggregates:
  reservation: {states: [Free, Held, Released], initial: Free}
  payment: {states: [NotCaptured, Captured, Refunded], initial: NotCaptured}
sequence:
  - {step: Reserving, aggregate: reservation, to: Held, undo: {to: Released}}
  - {step: Paying, aggregate: payment, to: Captured, undo: {to: Refunded}}
invariants:
  reserve-before-pay: 'payment = "Captured" => reservation # "Free"'
```

Run `machinery verify-formal <design>` after authoring. Use `--gen-only` only
when Java is unavailable, and record that the solver run remains owed.

## External checker closure

The committed `design/checkers/<id>.checker.yaml` remains tool-neutral. The
machine-local, git-ignored `.machinery/checkers.local.yaml` supplies the exact
runtime. Ambient host interpreters are not accepted. Use this complete shape:

```yaml
checkers:
  <checker-id>:
    runtime:
      kind: oci
      engine: [docker]
      image: registry.example/checker@sha256:<64-lowercase-hex>
      platform: linux/amd64
      inputs:
        - {source: tools/adapter.py, mount: adapter.py}
    run: ["python3", "/checker/adapter.py", "{projection}", "{config}", "{out}"]
    verify: ["python3", "/checker/adapter.py", "--verify", "{out}"]
    timeout: 120s
```

`platform` is exactly `linux/amd64` or `linux/arm64`; the image must be pinned
by digest. Input sources are registry-relative regular files and mounts are
portable relative paths. Successful checker and engine commands are silent.
Run, in order:

1. `machinery project <design>`.
2. Run the adapter under the declared closure to produce committed evidence.
3. `machinery check <design> --gate gk`.
4. `machinery verify-checkers <design>`.

## Attestations cover their full subject

One current hash is not enough when a claim ranges over an inventory. Each row
must cover every required subject for that claim:

| Claim family | Required covered paths |
|---|---|
| `g2.*` | `ARCHITECTURE.md` |
| `g3.*` | every `machines/*.machine.json` and every `machines/*.matrix.md` |
| `gt.conformance-test-shape`, `g4.zero-context`, `g4.standin-coverage` | `BUILD.md` and every non-index `BUILD/*.md` packet |
| `ga.review-quality` | every `acceptance/*.yaml` file |
| `g4.pack-event-discipline` | every regular file under `pack/` |

Generate hashes with `machinery attest <path>...`. A row is:

```yaml
attestation_version: 1
attestations:
  - claim: g3.guard-semantics
    attestor: <person or reviewing agent>
    date: 2026-09-03
    covers:
      - path: machines/Order.machine.json
        hash: sha256:<hex>
      - path: machines/Order.matrix.md
        hash: sha256:<hex>
```

Re-read and re-attest every subject after any covered byte changes. Never copy
an old hash forward without repeating the judgment.

## Milestone acceptance

A milestone is closed only after its reviewed commit has evidence at
`acceptance/M<n>.yaml`:

```yaml
milestone: 3
commit: 9f3c1a2b7d4e5f60718293a4b5c6d7e8f9012345
verdict: ACCEPTED
dod_ids:
  - T-DEAL-04
  - DEAL-eb0c40
attestations:
  - ga.review-quality: the DoD was exercised against the real boundary
findings: []
reviewer: milestone acceptance review, conductor + owner sign-off
date: 2026-09-03
```

Only then set `Status: closed` in the root milestone manifest. Run
`machinery check <design>` from the evidence-bearing checkout; Machinery proves
that each reviewed commit is in that checkout's ancestry. Use `--commit <sha>`
only to select a different history endpoint. `dod_ids` must equal the real
oracle identifiers cited by that milestone's DoD, in both directions.

When a DoD requires an entire oracle table, keep the plan compact with
`ORACLESET{machines/<Machine>.oracle.md}`, `ORACLESET{formal/Policy.oracle.md}`,
or `ORACLESET{formal/Isolation.oracle.md}`. The acceptance file must still list
every expanded stable id individually, so later oracle drift invalidates the
old review instead of changing its meaning.

## Implementation dependency inventory

With `--impl`, imports of manifest-declared third-party packages must resolve
to an Architecture Contract `external.imports` entry. This applies to Go,
Rust/Cargo, Node, Python, and Elixir manifests. Cargo dotted keys such as
`serde.workspace = true`, dependency tables, workspace dependencies, target
dependencies, build dependencies, and dev dependencies are supported.
For a Cargo member subtree, pass the member or `crates/` directory as
`--impl`; Machinery snapshots the governing ancestor `Cargo.toml` separately
and verifies every `workspace = true` dependency against its exact
`[workspace.dependencies]` definition. Malformed version requirements,
warning-bearing Git URL fragments, unsupported dependency keys, and invalid
field/group combinations fail closed.

Declare each adopted library once as an external with its import roots, then
give it the same adoption-closure and mitigation treatment as other declared
dependencies. Do not add externals merely to silence the gate: confirm that the
manifest and actual imports still represent an intended dependency.
