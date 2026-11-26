# machinery

machinery joins a project's design to evidence from its running system, plans what is unproven
as a backlog graph, and adds formality where the risk calls for it.

This branch is **v2, experimental**. Nothing is released from it. v1 is on `main`.

## The idea

A project keeps a small, structured design: a domain model ([Modelith](https://github.com/stacklok/modelith)
0.6.0), state machines (the XState v5 JSON subset, checked against XState 5.33.4), API contracts,
failure modes and a risk file. One command-line tool does the rest:

| Step | Command | What you get |
|---|---|---|
| Lint and cross-check the design | `machinery lint` | Every state answers every event; model, machines and contracts agree. |
| Generate what follows from it | `machinery gen` | Runtime transition tables and a list of obligations with stable ids. |
| Check the code against it | `machinery check` | Static stages: trace, layering, tests policy, drift, ratchet. |
| Prove it on the real system | `machinery run`, `machinery verify` | Receipts from real end-to-end runs, joined to the obligations at one commit. |
| Take in a new requirement | `machinery intake` | The design changes first; then every story of the change is written at once, with a capstone that shows it on the running system. |
| Plan what is unproven | `machinery plan`, `machinery waves` | Backlog stories with a dependency graph, scheduled in parallel waves. |
| Accept by risk | `machinery story`, `machinery landing` | A tier gate per story; one full verification at the head that is ready. |

Three rules shape it:

- **Formality follows risk.** Routine work gets machines and generated end-to-end cases.
  Consequential work adds a fresh verifier who predicts before reading the proof. Critical work
  adds a formal model: TLA+ for ordering, concurrency and recovery, Alloy for structure. Vital
  work, the few paths that must never go wrong, adds exact properties over its ids, a small core
  shaped for a later machine-checked proof, and its assumptions with evidence; nothing reports
  it as more proven than its tests and model make it.
- **Evidence comes from the real system.** No unit tests and no test doubles, in machinery or in
  a project that uses it. Test quality is measured by `machinery probe`, which gives the system
  a wrong transition table and expects the tests to notice.
- **A blocking check protects a named product behavior.** `machinery checks` lists every stage
  and what it protects. Nothing checks that a ritual was followed.

Formal models carry the checks that keep them from lying: reach predicates (no vacuous pass),
numbered assumptions, a mutation list per spec, a baseline that makes weakening visible, a sync
marker against the source, and replay of every counterexample against the real code.

## Start

```
make tools && make build        # in this repository: pinned tools into .bin/, then the binary
```

| You have | Read |
|---|---|
| A new project | [docs/getting-started.md](docs/getting-started.md) |
| Existing code and a backlog | [docs/brownfield.md](docs/brownfield.md): assess read-only first, then adopt one area at a time, no rewrite |
| A backlog to plan and schedule | [docs/pm.md](docs/pm.md) |
| A critical or vital area to model | [docs/formal.md](docs/formal.md) |
| A deterministic checker of your own | [docs/external-checkers.md](docs/external-checkers.md) |
| A question about any format or output line | [docs/formats.md](docs/formats.md), the contract |

The reference project, a small CRM on a real Postgres built with this method from the design up,
is in [examples/crm](examples/crm).

## For coding agents

The plugin ships one small skill (`skills/machinery/SKILL.md`), three prompts: planner,
verifier and modeler (`agents/`), and three hooks (`hooks/hooks.json`, see
[docs/hooks.md](docs/hooks.md)): the status when a session starts, the static check before
`git commit` and `git push`, and after a turn that changed files. Checks are commands, not prose;
the hooks only call them. Workers are briefed by pointer with `machinery brief <story>`.

## Requirements

Go 1.27, git, `modelith` 0.6.0. Optional, by feature: Node and `xstate` 5.33.4 (stage
`xstate`), Java 21 with the pinned `tla2tools` and Alloy jars (formal models), `bd` (the backlog
bridge). `make tools` installs the pinned ones into `.bin/`.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md). `make` lists the targets; `make verify` is the full
verification.

## License

Apache-2.0. See [LICENSE](LICENSE).
