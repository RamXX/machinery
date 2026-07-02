# machinery

A design agent for greenfield software. It interrogates you until the *what* is precise, the *how* is
fixed, and the *behavior* is a state machine, then emits a `BUILD.md` that a coding agent with zero
prior context can implement under hard TDD.

## Thesis

Most software is a state machine. machinery makes that explicit across three layers that compose:

1. **Domain model (the what)** via [Modelith](https://modelith.sh): entities, relationships, invariants,
   actions, scenarios. Lints clean before anything else proceeds.
2. **Architecture (the how)** via C4 (Structurizr DSL) plus a machine-checkable Architecture Contract:
   deployment, tech stack, boundaries, and for every dependency a failure-and-mitigation posture.
3. **State machine (the behavior)** in [XState](https://github.com/statelyai/xstate) v5 JSON: every
   state, transition, guard, timeout, and failure mode.

The state machine is authored **last** because it needs the other two as inputs. But half of it is
derived, not invented: the domain lifecycle is already latent in the Modelith model (states are status
enum values, events are actions, guards enforce invariants). The FSM phase weaves that domain lifecycle
together with a failure-and-recovery overlay that only the architecture can inform. A C4 mitigation
(for example Postgres on Kubernetes behind an operator) reclassifies a failure from "fatal" to
"transient and bounded"; it does not delete the failure transition.

## Pipeline

```
Phase 0  Frame        one paragraph: what, who, purpose, target language(s)
Phase 1  Modelith     domain model            gate: modelith lint clean
Phase 2  C4           architecture + contract  gate: every action owned, every dep has a mitigation posture
Phase 3  XState       state machines           gate: every invoke has onError + timeout; every invariant guarded
Phase 4  BUILD.md     the blueprint            gate: a zero-context coding agent could build it
```

Every phase has an exit gate. The conductor does not advance until it passes.

## What it produces

Everything lands in a single `design/` directory in the target project:

```
design/
  domain.modelith.yaml   domain.md
  workspace.dsl          ARCHITECTURE.md
  machines/<C>.machine.json   machines/<C>.matrix.md
  BUILD.md
```

`BUILD.md` carries the domain model, the architecture, the machines, a traceability matrix (every
invariant to its guard to its component to its tests), and a **test specification** derived from the
transition matrices. That test spec is the hard-TDD oracle: a test-writer agent writes the tests from
it, the tests are locked, and the implementer makes them pass without editing them.

## Components

- `skills/machinery/SKILL.md` - the conductor. Runs in the main session so it can interrogate you
  turn by turn. Reuses the `domain-model-author` skill for Phase 1 and embeds the C4 technique.
- `skills/machinery/references/` - the XState serializable subset, the standalone C4 technique, and
  the BUILD.md template.
- `agents/machinery-fsm-author.md` - synthesis subagent for Phase 3 (authors the machines).
- `agents/machinery-build-writer.md` - synthesis subagent for Phase 4 (assembles BUILD.md).

## Install

Requires [`modelith`](https://github.com/stacklok/modelith) on `PATH`
(`go install github.com/stacklok/modelith/cmd/modelith@latest`).

```sh
make install       # symlink into ~/.claude (edits in this repo go live immediately)
make install-copy  # copy instead of symlink
make doctor        # check modelith + install status
make uninstall     # remove from ~/.claude
```

## Use

In a Claude Code session, from the project you want to design:

```
Design a new <system> with machinery.
```

The conductor takes it from Phase 0. It is fully standalone: no tracker, no project settings, no
other processes dependencies. Target languages it realizes: Elixir, Go, Rust, TypeScript, Python.

## License

Copyright 2026 Ramiro Salas. Licensed under the Apache License 2.0; see `LICENSE`.

machinery does not bundle or redistribute any of the tools it works with. It invokes the `modelith`
CLI and emits text in the XState and C4 notations, so it is an independent work and no dependency's
license constrains it. For reference, those tools are permissively licensed and compatible with
Apache-2.0: Modelith and Structurizr are Apache-2.0, XState (`@xstate/graph`) and LadybugDB are MIT,
and C4 is an open notation.
