# machinery

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

- **The what**, a domain model (Modelith): the entities, the relationships, and the invariants that
  must always hold. Linted.
- **The how**, an architecture (C4): the components, the deployment, and what every dependency does
  when it fails. Contract-checked.
- **The behavior**, state machines (XState): every state, transition, guard, timeout, and failure
  mode, conditioned on the architecture the previous layer fixed. Model-checked.

The state machines come last because they need the other two as inputs, and half of each machine is
*derived* from the domain model rather than invented. The final artifact is a `BUILD.md` a zero-context
coder can build from, plus the machines that are simultaneously the test oracle and the formal spec.

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
   transition matrix, the hard-TDD test oracle, and the TLA+ spec. Whole classes of drift become
   impossible by construction rather than caught after the fact.
2. **Deterministic symbolic gates.** `machinery check` verifies, with no LLM in the loop: machines are
   well-formed (targets resolve, no dead ends, every side effect has an error path and a timeout), the
   architecture contract is consistent and complete, the layers trace to each other, and the code
   respects the boundaries.
3. **Model checking.** Each machine is finite, so TLC proves it exhaustively: safety (bad states
   unreachable, retry loops bounded), liveness (every operation terminates, nothing gets stuck
   half-done), and deadlock-freedom.
4. **Refinement and assume-guarantee.** Each subsystem is proven to refine the small contract its
   neighbors rely on, and the contracts compose. Parts are verified against contracts, never against
   the flattened system, which is the only way this scales to real size.

Rungs 1 through 3 are generated from the design automatically. Rung 4 is generated from a short
declarative annotation. Correctness at every level, and almost none of it hand-written.

## Proof it works: the go-crm example

`examples/go-crm` is a Go CRM with a native CLI over an embedded LadybugDB graph and role- and
ownership-based access control, taken end to end:

- **Designed** through all four phases. Domain model lints clean (9 entities, 24 invariants); C4 model
  with the dependency posture that an embedded store forces (corruption is fatal-until-restore, not a
  transient); five state machines; a 1223-line `BUILD.md`.
- **Built by a zero-context coding agent** under hard TDD: a test-writer wrote the suite from the
  blueprint, the tests were locked, and an implementer made them pass without touching a test. Result:
  286 tests green, 89% coverage, architecture boundaries upheld in the source. The one impossible test
  was escalated as a design defect and fixed in the design, not the code.
- **Gated** by `machinery check`: it certified the design consistent and caught a real contract defect
  the prose review had missed.
- **Proven** by `make verify-formal`: eight TLC proofs, all green, regenerated from source every run so
  they cannot drift from the design.

Every number above is real output in this repository, not an illustration.

## Install

Requires [`modelith`](https://github.com/stacklok/modelith) on `PATH`. The formal layer needs Java 11+;
`tlc.sh` fetches the TLA+ tools on first use.

```sh
make install       # symlink the skill and agents into ~/.claude (edits go live)
make doctor        # check dependencies and install status
make check         # run the deterministic gate suite on the example
make verify-formal # regenerate and TLC-check the example's eight proofs
make oracle        # regenerate the transition oracles from the machine JSON
```

## Use

In a Claude Code session, from the project you want to design:

```
Design a new <system> with machinery.
```

The conductor takes it from Phase 0. It is fully standalone: no tracker, no project settings, no
other process dependencies. Target languages it realizes: Elixir, Go, Rust, TypeScript, Python.

## How it is put together

- `skills/machinery/SKILL.md` the conductor, plus `references/` (XState format, C4 technique, BUILD.md
  template) and `tools/` (the deterministic gate suite and the formal generators).
- `agents/` two synthesis subagents (the machine author and the build-doc writer).
- `examples/go-crm/` the worked example: `design/` (the blueprint and the formal models) and `impl/`
  (the verified Go build).

See `skills/machinery/tools/README.md` for the checkers and generators, and
`examples/go-crm/design/formal/README.md` for the proofs.

## License

Copyright 2026 Ramiro Salas. Licensed under the Apache License 2.0; see `LICENSE`. machinery invokes
`modelith` and emits XState and C4 notation; it bundles none of them, so no dependency's license binds
it. The tools it works with are permissively licensed and compatible with Apache-2.0: Modelith and
Structurizr are Apache-2.0, XState and LadybugDB are MIT, and C4 is an open notation.
