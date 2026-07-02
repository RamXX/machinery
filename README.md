# machinery

[![CI](https://github.com/ramirosalas/machinery/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/ramirosalas/machinery/actions/workflows/ci.yml)
[![Formal Verification](https://github.com/ramirosalas/machinery/actions/workflows/formal.yml/badge.svg?branch=main)](https://github.com/ramirosalas/machinery/actions/workflows/formal.yml)
[![Security](https://github.com/ramirosalas/machinery/actions/workflows/security.yml/badge.svg?branch=main)](https://github.com/ramirosalas/machinery/actions/workflows/security.yml)
[![Nightly](https://github.com/ramirosalas/machinery/actions/workflows/nightly.yml/badge.svg?branch=main)](https://github.com/ramirosalas/machinery/actions/workflows/nightly.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ramirosalas/machinery.svg)](https://pkg.go.dev/github.com/ramirosalas/machinery)
[![Go Report Card](https://goreportcard.com/badge/github.com/ramirosalas/machinery)](https://goreportcard.com/report/github.com/ramirosalas/machinery)

**Design software once, as a state machine, and let everything else be derived and proven from it:
the tests, the architecture contracts, the build instructions, and machine-checked proofs of
correctness.** machinery is a design methodology and toolchain that turns a fuzzy idea into a
build-ready, formally verified blueprint that a coding agent with zero prior context can implement
under hard TDD.

## Why it exists

AI coding agents make it cheap to write software fast. They do not make it safe to write large
software. On anything past a toy, correctness degrades quietly: the design and the code drift apart,
a cross-cutting invariant gets violated three files away from where it was written, a failure mode
nobody enumerated takes down production at 3am. The usual answer is "review it carefully," which is
another way of saying "trust the model." Trust does not scale.

machinery takes the opposite stance. It treats correctness as something you **construct and check**,
not something you hope for. The design is a single source of truth. Everything downstream is either
generated from it (so it cannot drift) or checked against it by a deterministic tool or an exhaustive
proof (so a mistake is caught, not shipped). The model does the creative work; the machine holds the
line.

## The thesis

Most software is a state machine. Make that explicit and the rest follows. machinery separates a
design into three layers that compose rather than merely coexist:

- **The what**, a domain model ([Modelith](https://modelith.sh/)): the entities, the relationships,
  and the invariants that must always hold. Linted.
- **The how**, an architecture ([C4](https://c4model.com/)): the components, the deployment, and what
  every dependency does when it fails. Contract-checked.
- **The behavior**, state machines in the [XState](https://github.com/statelyai/xstate) v5 JSON
  format (the notation, not the library): every state, transition, guard, timeout, and failure mode,
  conditioned on the architecture the previous layer fixed. Model-checked.

The state machines come last because they need the other two as inputs, and half of each machine is
*derived* from the domain model rather than invented. The final artifact is a `BUILD.md` a zero-context
coder can build from, plus the machines that are simultaneously the test oracle and the formal spec.

To be clear about that XState reference: machinery uses the XState config **format** as a linted
notation and does not run the XState library. The lint is our own (a deliberately narrowed subset,
plus annotations like `_role` and `_exhaustive` that are not XState), and so are the oracle generator
and the model checking. The de-annotated config loads into [Stately](https://stately.ai/) and
`@xstate/graph` for optional visualization and covering-path generation, and a TypeScript build may
adopt XState directly, while Go, Rust, Python, and Elixir targets hand-roll the state field. The
guarantees come from machinery's tooling and [TLC](https://github.com/tlaplus/tlaplus), never from
XState.

## Agentic systems: the machine is the envelope, not the agent

"Most software is a state machine" is a claim about control flow, not about cognition, and agentic
programs are where the difference bites. An agent is a non-deterministic policy (an LLM choosing which
tool to call) running inside a deterministic envelope: the perceive-act-observe loop, tool execution,
budgets, guardrails, approval gates, side-effect compensation, sub-agent orchestration. The envelope
is the state machine; the policy is not, and machinery does not pretend otherwise.

What machinery models is the containment. Every tool is an `invoke` with an enumerated failure set, a
timeout, and an idempotency key, so the action space is bounded even when the choice within it is not;
the loop is a machine whose termination and budget are model-checked, so it provably stops; every
irreversible action is a saga with compensation and an explicit stalled-dirty residual; every
guardrail is a guard tied to an invariant; sub-agents coordinate through the event-contract table. The
LLM decision itself is a contracted, non-deterministic oracle machinery fences but never tries to
prove. You get liveness (it stops, within budget, and cleans up) without usefulness (that it chose
well), so keep the action space enumerated and the high-stakes guardrails deterministic: that is the
region machinery can prove, and whatever you leave to the model's discretion is region it cannot.
Fittingly, machinery is itself built this way, a gated pipeline around a non-deterministic conductor.

## The pipeline

```
Phase 0  Frame        what, who, purpose, target language
Phase 1  Modelith     domain model             gate: modelith lint clean
Phase 2  C4           architecture + contract   gate: every action owned, every dependency has a mitigation posture
Phase 3  XState       state machines            gate: every invoke has onError + timeout; every invariant guarded
Phase 4  BUILD.md     the blueprint             gate: a zero-context coding agent could build it
```

An interrogation, not a form. The conductor pushes on naming and on "what must always be true," and
does not advance a phase until its gate passes.

## What makes it production-grade: the correctness ladder

A domain-model linter is table stakes. The differentiator is that machinery pushes deterministic and
formal correctness into every layer, strongest first:

1. **Generate, do not co-author.** Anything derivable is generated from the machine JSON: the
   test oracle (with content-derived stable ids that survive design revisions) and the TLA+ specs.
   The gates then diff every committed generated artifact against a fresh generation, so staleness
   is caught as drift, never assumed away.
2. **Deterministic symbolic gates that cannot pass on absence.** `machinery check` verifies, with no
   LLM in the loop: machines are well-formed (reachability, unambiguous targets, no dead ends, every
   side effect has an error path and a timeout, every resting state handles or explicitly ignores
   every event), the architecture contract binds to the C4 model and every dependency has a failure
   posture, the layers trace to each other by construction (states to enum values, events to actions,
   invariants to enforcement rows), and the code respects the boundaries (Go, Python,
   TypeScript/JavaScript, Elixir, Rust). Every gate prints what it actually checked; a gate that
   finds nothing to check fails instead of passing.
3. **Model checking.** Each machine is finite, so TLC checks it exhaustively: retry loops bounded
   (each loop with its own counter), every operation terminates, nothing gets stuck half-done, no
   deadlock. Every generated spec states its assumptions in its header (guards erased soundly for
   safety, liveness conditional on guard exhaustiveness that the linter discharges, single instance,
   no data at this rung), so a green check reads as exactly what it is.
4. **Refinement and assume-guarantee.** The data-and-composition annotations are reconciled against
   the machines before anything is emitted, so a drifted annotation fails generation instead of
   proving a stale twin. Each subsystem is proven to refine the small contract its neighbors rely
   on; the composition instances that same contract module and TLC additionally checks the
   composition satisfies it. Parts are verified against contracts, never against the flattened
   system, which is the only way this scales to real size.

Rungs 1 through 3 are generated from the design automatically. Rungs 2 and 4 are generated from
short declarative annotations that the generators verify against the machines. And the toolchain
that holds all of this together is itself held: a pytest suite encodes every vacuity and drift
attack from an adversarial design review as a permanent regression, and CI runs the tests, the
gates, the proofs, and the example build on every push.

Generating every artifact (the oracle, the TLA+ specs, the reconciled models) is pure Python and
needs no [Java](https://adoptium.net/). Only the *checking* of the proofs (rungs 3 and 4, and rung 2's refinement) runs under
TLC, which is a Java program, so **Java is required only for `make verify-formal`**. That step is
optional but recommended: the deterministic gates (rung 2's generation and every symbolic check)
already catch malformed machines, drift, and boundary erosion, but they cannot tell you that a saga
can strand money, that a retry can loop forever, or that a subsystem violates the contract its
neighbors assume. The model checking is what proves those, exhaustively, with a concrete
counterexample when it fails (it caught two real defects in the examples below). A Java-free setup is
a complete, gated design; adding Java upgrades "structurally consistent" to "machine-checked."

## Proof it works: the go-crm example

`examples/go-crm` is a Go CRM with a native CLI over an embedded
[LadybugDB](https://github.com/LadybugDB/go-ladybug) graph and role- and
ownership-based access control, taken end to end:

- **Designed** through all four phases. Domain model lints clean (9 entities, 24 invariants); C4 model
  with the dependency posture that an embedded store forces (corruption is fatal-until-restore, not a
  transient); five state machines; a 1223-line `BUILD.md`.
- **Built by a zero-context coding agent** under hard TDD: a test-writer wrote the suite from the
  blueprint, the tests were locked, and an implementer made them pass without touching a test. Result:
  286 tests green, 89% coverage, architecture boundaries upheld in the source. The one impossible test
  was escalated as a design defect and fixed in the design, not the code.
- **Gated** by `machinery check`: it certified the design consistent and caught a real contract defect
  the prose review had missed. The hardened gates verify it non-vacuously: 194 transitions reconciled
  row by row against the matrices, every guard, action, and actor covered by a named-unit contract,
  every import edge checked against the architecture contract.
- **Proven** by `make verify-formal`: eight TLC proofs, all green, regenerated from source every run.

Every number above is real output in this repository, not an illustration.

And it holds on a second, deliberately different design. `examples/fulfillment` is a distributed
order-fulfillment platform: microservices, a saga orchestrator, compensation, and a transactional
outbox, with six state machines (the saga plus the Order, Payment, Reservation, Shipment, and
OutboxMessage lifecycles). The same generators produced its formal models, and TLC checked them: the
saga always terminates, and its data-refined model shows that money and stock are never silently
lost, with compensation modeled per obligation so partial compensation is a real, checked state.
Building that proof caught a real bug in the saga as first drawn, where a single failed refund could
leave a customer charged with nothing returned. TLC produced the exact counterexample and the fix is
checked. The hardened cross-layer gate then caught a second real defect: the domain model's saga
enum had drifted from the machine and was missing the FailedDirty residual entirely. Across the
example designs, `make verify-formal` checks every proof, all green, regenerated from source on each
run (and only this step needs Java).

## Brownfield systems

The pipeline reads as greenfield, but the toolchain does not care. On an existing system you run
the phases as archaeology instead of invention: write the Modelith model, the contract, and the
machines to describe the system AS IT IS, then let the gates arbitrate. G4 checks the real code
against the contract immediately, and Gx surfaces every place the code's actual states and events
disagree with your model, which is exactly the drift map you want from a legacy system. From there,
revision mode is the operating loop: design changes as diffs, stable-id oracle diffs as the
affected-test list, and state-migration notes for persisted machines, which a brownfield system has
on day one. Two things change character: oracle-derived tests start as characterization tests (when
one fails, you decide case by case whether the code or the machine is the truth), and the first
modeling pass is a real investment, roughly proportional to how undocumented the system is.

## Which model to use where

The gates check structure, not substance: a shallow domain model with the wrong invariants gates
completely clean. Extracting the right invariants, pushing on fuzzy definitions, and deciding
failure postures (Phases 0 through 2) is pure judgment with no machine backstop, so use the
strongest reasoning model you have there. Phases 3 and 4 are a different regime: half of each
machine is derived mechanically, and lint, oracle diff, reconciliation, and TLC catch most of what
a weaker model would fumble, so a mid-tier model is much safer for the synthesis. The deterministic
layer narrows the failure mode rather than removing it: with a weak interrogator you get a
structurally consistent, formally verified model of the wrong system.

## What machinery does not verify

The gates and proofs are exactly as strong as stated above, and no stronger. Not covered by any
deterministic check or proof, by construction: whether the interrogation extracted the RIGHT
invariants (a shallow domain model gates clean); guard and action semantics in code (the named-unit
contracts carry them into unit, integration, and property tests, but a wrong implementation of a
correctly-named guard is caught by tests, not proofs); races between concurrent machine instances
and message loss, duplication, or reordering between machines (the models are single-instance; the
event-contract table and the idempotency contracts govern those seams, and the tests exercise them);
coupling through shared database tables or bus topics (invisible to import analysis); and security,
capacity, and observability beyond what the Phase 2 NFR record captures. The methodology's stance is
to name every one of these residuals in the design artifacts rather than let a green check imply
they are covered.

## Install

### Prerequisites

`make preflight` (or `make doctor`) checks all of these at any time; `make install` runs the same
check and warns about anything missing.

**Required**

- **[modelith](https://modelith.sh/)** -- Phase 1 domain-model lint and render. Primary install is
  [Homebrew](https://brew.sh/) (macOS and Linux): `brew install stacklok/tap/modelith`. Secondary:
  with the [Go](https://go.dev/dl/) toolchain on any OS,
  `go install github.com/stacklok/modelith/cmd/modelith@latest` (then put `$(go env GOPATH)/bin` on
  your `PATH`); or download a prebuilt binary (macOS, Linux, Windows) from the
  [releases](https://github.com/stacklok/modelith/releases). Full options:
  [modelith.sh/cli](https://modelith.sh/cli/).
- **machinery** -- the deterministic gate tools and formal generators. A single static binary
  (no Python, no Go runtime). Two ways to install:

  **Download a prebuilt binary** (no toolchain needed — macOS arm64/x86, Linux amd64/arm64,
  Windows amd64):
  ```bash
  make install-binary                      # auto-detects OS/arch, fetches latest release
  # or pin a version:
  make install-binary MACHINERY_VERSION=v0.1.0
  ```
  Binaries are published on the [releases page](https://github.com/ramirosalas/machinery/releases);
  download `machinery-<os>-<arch>`, put it on your `PATH`.

  **Or build from source** (if you have [Go](https://go.dev/dl/) 1.22+):
  ```bash
  make build                               # builds .bin/machinery
  ```

**Optional**

- **[Java](https://adoptium.net/) 11+** -- only for `make verify-formal`, which runs
  [TLC](https://github.com/tlaplus/tlaplus) to model-check the proofs (`tlc.sh` then fetches the
  pinned [tla2tools.jar](https://github.com/tlaplus/tlaplus/releases) on first use). macOS:
  `brew install --cask temurin`; Linux: `sudo apt install default-jdk` or
  `sudo dnf install java-21-openjdk`, or [download Temurin](https://adoptium.net/temurin/releases/);
  Windows: `winget install EclipseAdoptium.Temurin.21.JDK` or
  [download](https://adoptium.net/temurin/releases/). Without Java you still get the full design and
  every deterministic gate; with it you add the machine-checked proofs (the top of the correctness
  ladder above).
- **[Structurizr CLI](https://github.com/structurizr/cli)** -- only to export C4 diagrams from
  `workspace.dsl` (the [Structurizr DSL](https://github.com/structurizr/dsl) text and every gate need
  no export); needs Java. Any OS: download a
  [release zip](https://github.com/structurizr/cli/releases), unzip, and add it to `PATH`
  (`structurizr.sh` on macOS/Linux, `structurizr.bat` on Windows); or run the
  [container](https://hub.docker.com/r/structurizr/cli): `docker pull structurizr/cli`.
- **[uv](https://docs.astral.sh/uv/)** -- runs `make test` and can resolve PyYAML on the fly. macOS
  and Linux: `curl -LsSf https://astral.sh/uv/install.sh | sh` (or `brew install uv`); Windows:
  `powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"` (or
  `winget install astral-sh.uv`).

```sh
make install       # symlink the skill and agents into every agent home (edits go live)
make doctor        # check dependencies and install status
make test          # run the toolchain's own test suite (pytest via uv)
make check         # run the deterministic gate suite on the examples
make verify-formal # regenerate and TLC-check the full formal suite
make oracle        # regenerate the transition oracles from the machine JSON
```

machinery is agent-agnostic. `make install` places the skill under `<home>/skills` and the two role
docs under `<home>/agents` for every home in `AGENT_HOMES`, which defaults to `~/.claude` (Claude
Code) and `~/.agents` (the cross-agent convention). Override it to add or restrict targets, for
example `make install AGENT_HOMES="$HOME/.agents"`. `make install-copy` does the same without
symlinks, and `make uninstall` removes machinery from every home.

The gate tools need Python 3.10+ with PyYAML (declared in `pyproject.toml`; `uv` resolves it).
`tlc.sh` uses a version-pinned, checksum-verified `tla2tools.jar`. CI runs the test suite, both
gate runs, the full formal suite, and the go-crm build on every push.

## Use

In an agent session (Claude Code, or any agent that loads skills from an installed home), from the
project you want to design:

```
Design a new <system> with machinery.
```

The conductor takes it from Phase 0. It is fully standalone: no tracker, no project settings, no
other process dependencies. Target languages it realizes: Elixir, Go, Rust, TypeScript, Python.

## How it is put together

- `skills/machinery/SKILL.md` the conductor, plus `references/` (XState format, C4 technique, BUILD.md
  template) and `tools/` (the frozen Python differential oracle during migration).
- `cmd/machinery/` the single Go binary (cobra CLI): `lint`, `oracle`, `tla`, `refine`, `compose`,
  `check`, `verify-formal`, `doctor`, `preflight`.
- `internal/` the Go toolchain: `ir/` (order-preserving machine model), `lint/`, `oracle/`, `tla/`,
  `refine/`, `compose/`, `gates/` (the G2/G3/Gx/G4 suite), `formal/` (TLC orchestration),
  `experiments/` (the shared mutation-experiment table). Every package has unit tests.
- `agents/` two synthesis subagents (the machine author and the build-doc writer).
- `examples/go-crm/` the worked example: `design/` (the blueprint and the formal models) and `impl/`
  (the verified Go build).
- `examples/fulfillment/` the distributed stress test: `design/` only (six machines, eight proofs,
  and `FINDINGS.md`, the record of what strained and what was fixed).
- `examples/portfolio-engine/` a second design-only example in a different domain and language (a
  Python drawdown portfolio recommender): exercises the terminal-lifecycle pattern and a
  persistence overlay renamed from the defaults, proving the formal layer is not hardcoded to one
  vocabulary.
- `tests/` the Python toolchain's own test suite (the frozen differential oracle during migration);
  the Go experiment table lives in `internal/experiments/`.

See `skills/machinery/tools/README.md` for the checkers and generators, and
`examples/go-crm/design/formal/README.md` for the proofs. The skill also defines a revision mode
(design changes after code exists: stable test ids, oracle diffs as the affected-test list, and a
mandatory state-migration note for persisted machines) and a sharding rule for designs beyond
roughly ten stateful components.

## Testing & CI

The Go toolchain has full unit tests in every package, plus a shared mutation-experiment table
(the vacuity/drift findings from design review, ported from the pytest suite). Current coverage:

| Package | Coverage | Role |
|---------|----------|------|
| `internal/tla` | 90% | TLA+ control-flow generator |
| `internal/oracle` | 87% | transition oracle (content-hashed ids) |
| `internal/lint` | 84% | structural lint + matrix reconciliation |
| `internal/refine` | 83% | data-refinement (3 patterns) |
| `internal/compose` | 79% | cross-aggregate composition |
| `internal/gates` | 77% | the G2/G3/Gx/G4 gate suite |
| `internal/ir` | 57% | shared IR (covered transitively via lint/gates) |
| **internal/ overall** | **~75%** | (cmd/ is thin CLI plumbing) |

Run `go test -coverprofile=cover.out ./internal/... && go tool cover -func=cover.out` locally.
CI runs `go test -race ./...`. Beyond unit tests, two stronger nets are always green in CI:

- **Differential parity** — `scripts/diff-all.sh` proves the Go binary equals the Python tools
  byte-for-byte (stdout/stderr/exit) on every example and every generator.
- **Formal verification** — `machinery verify-formal` regenerates and TLC-model-checks all 22 TLA+
  proofs across the three examples.

## Built on

machinery is a thin methodology over these external projects. It invokes or emits their notations and
bundles none of them.

- [Modelith](https://modelith.sh/) -- the domain-model language and linter (Phase 1).
- [C4 model](https://c4model.com/) -- the architecture technique (Phase 2).
- [Structurizr DSL](https://github.com/structurizr/dsl) and
  [Structurizr CLI](https://github.com/structurizr/cli) -- architecture-as-code, and optional C4
  diagram export.
- [XState](https://github.com/statelyai/xstate) and [Stately](https://stately.ai/) -- the
  state-machine JSON format (notation only; machinery does not run the library) and its visualizer.
- [TLA+ and TLC](https://github.com/tlaplus/tlaplus) -- the specification language and model checker
  (the formal layer).
- [Eclipse Temurin / Adoptium](https://adoptium.net/) -- the JVM that runs TLC.
- [Python](https://www.python.org/) and [PyYAML](https://pyyaml.org/) -- the gate tooling.
- [uv](https://docs.astral.sh/uv/) -- the test runner and dependency resolver.
- [Go](https://go.dev/) -- to install Modelith.
- [LadybugDB](https://github.com/LadybugDB/go-ladybug) -- the embedded store used only by the go-crm
  example, not a machinery dependency.

## License

Copyright 2026 Ramiro Salas. Licensed under the Apache License 2.0; see `LICENSE`. machinery invokes
`modelith` and emits XState and C4 notation; it bundles none of them, so no dependency's license binds
it. The tools it works with are permissively licensed and compatible with Apache-2.0: Modelith and
Structurizr are Apache-2.0, XState and LadybugDB are MIT, and C4 is an open notation.
