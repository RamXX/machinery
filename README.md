# machinery

machinery joins a project's design to evidence from its running system, plans what is unproven
as a backlog graph, and adds formality where the risk calls for it.

This is **machinery 2.0.0**. It is a rewrite, not an in-place upgrade of v1: the design is data,
evidence comes only from end-to-end runs, and the backlog lives in `bd`. v1, the 0.x releases
(last 0.11.2), lives on branch [`v1`](https://github.com/RamXX/machinery/tree/v1).
[CHANGELOG.md](CHANGELOG.md), "Upgrading from v1", says how to move a project.

## Install

With Go 1.27.4 and git:

```
go install github.com/RamXX/machinery/v2/cmd/machinery@v2.0.0
go install github.com/stacklok/modelith/cmd/modelith@v0.6.0   # the model linter it runs
```

Optional, by feature: Node and `xstate` 5.33.4 (stage `xstate`), Java 21 and the pinned TLA+
and Alloy jars (`machinery formal tools` fetches them by checksum), and `bd` (the backlog
bridge).

`machinery --version` prints `machinery <version> (<commit>, <build date>)`. A `go install`
build prints `machinery dev (unknown, unknown)`: Go records no revision in a build from the
module cache, and `go version -m "$(command -v machinery)"` shows the module version. In a
checkout of the tag, `make tools && make build` puts the pinned tools and a binary that prints
`machinery 2.0.0 (<commit>, <build date>)` in `.bin/`.

## Ten minutes

A ticket desk, from an empty directory to a proven design. `MACHINERY_REPO` is a checkout of
this repository at the tag (`git clone --branch v2.0.0 https://github.com/RamXX/machinery`);
`docs/getting-started.md` explains each step. The test suite runs this block
(`docs_readme.txtar`):

```sh
machinery init
cp -R "$MACHINERY_REPO/e2e/testdata/fixtures/starter/." .
machinery gen
git init -q -b main
git add -A
git commit -q -m "ticket desk"
machinery check
machinery run -- sh e2e/tickets_test.sh
machinery verify
machinery status
```

`verify` ends with `machinery verify: 1 passed, 0 failed, 0 not demonstrated, 0 not applicable`:
six obligations, each observed on the running system at that commit.

## Limits and what v2 does not do

| Area | Limit |
|---|---|
| Hooks | Wired in `hooks/hooks.json`, and `machinery hook` is tested with hook JSON on stdin. They are not yet confirmed in a live harness session: try them in one of yours before you rely on them. The Codex plugin wires no hooks. |
| Vital cores | Denied imports, function sizes and entry symbols of a core are read in Go files only. A core in another language is not demonstrated. |
| Adapter evidence | Weaker than a receipt a test emits: it shows that a test which names the id passed, not that the test checked it. It never proves a `tc` or `cx` id, and by default no id of a critical or vital area. |
| Proof obligations | None is discharged. Each property of a vital area prints as awaiting proof, and an id it is over reads `(by test and model; proof awaited)`. |
| Delegate | Experimental: its commands and the policy format may change in any release. |
| Evidence rules | They catch mistakes and casual edits, not a party who can run commands as the tool or plant code in the project (`docs/formats.md` sections 10.2 and 10.3). |
| Backlog | `bd` is the only backlog tool. |

## What it does

A project keeps a small, structured design: a domain model ([Modelith](https://github.com/stacklok/modelith)
0.6.0), state machines (the XState v5 JSON subset, checked against XState 5.33.4), API contracts,
failure modes and a risk file. One command-line tool does the rest:

| Step | Command | What you get |
|---|---|---|
| Lint and cross-check the design | `machinery lint` | Every state answers every event; model, machines and contracts agree. |
| Generate what follows from it | `machinery gen` | Runtime transition tables and a list of obligations with stable ids. |
| Check the code against it | `machinery check` | Static stages: trace, layering, tests policy, drift; with `--base`, the ratchet. |
| Prove it on the real system | `machinery run`, `machinery verify` | Receipts from real end-to-end runs, joined to the obligations at one commit. |
| Take in a new requirement | `machinery intake new` | The design changes first; then every story of the change is written at once, with a capstone that shows it on the running system. |
| Plan what is unproven | `machinery plan`, `machinery waves` | Backlog stories with a dependency graph, scheduled in parallel waves. |
| Accept by risk | `machinery story`, `machinery landing` | A tier gate per story; one full verification at the head that is ready. |
| Adopt an existing project | `machinery assess`, `machinery adopt` | A read-only report, a rehearsal, then one area at a time without a rewrite. |

Three rules shape it:

- **Formality follows risk.** Routine work gets machines and generated end-to-end cases.
  Consequential work adds a fresh verifier who predicts before reading the proof. Critical work
  adds a formal model: TLA+ for ordering, concurrency and recovery, Alloy for structure. Vital
  work, the few paths that must never go wrong, adds exact properties over its ids, a small core
  shaped for a later machine-checked proof, and its assumptions with evidence.
- **Evidence comes from the real system.** No unit tests and no test doubles, in machinery or in
  a project that uses it. A receipt counts only from a run of `machinery run` and only for a
  test file that claims the id. Test quality is measured by `machinery probe`, which gives the
  system a wrong transition table and expects the tests to notice.
- **A blocking check protects a named product behavior.** `machinery checks` lists every stage
  and what it protects. Nothing checks that a ritual was followed.

## Where to read next

| You have | Read |
|---|---|
| A new project | [docs/getting-started.md](docs/getting-started.md) |
| Existing code and a backlog | [docs/brownfield.md](docs/brownfield.md): assess, rehearse, adopt one area at a time, evidence from the suite you have |
| A backlog to plan and schedule | [docs/pm.md](docs/pm.md) |
| A critical or vital area to model | [docs/formal.md](docs/formal.md) |
| Machines to write | [docs/machine-format.md](docs/machine-format.md) |
| A deterministic checker of your own | [docs/external-checkers.md](docs/external-checkers.md) |
| A question about any format, command, output line or exit code | [docs/formats.md](docs/formats.md), the contract |

The reference project, a small CRM on a real Postgres built with this method from the design up,
is in [examples/crm](examples/crm).

## For coding agents

The plugin ships one small skill (`skills/machinery/SKILL.md`), one prompt per role under
`agents/` (planner, modeler, builder, verifier, checker and an experimental delegate; see
[docs/roles.md](docs/roles.md)), and three hooks (`hooks/hooks.json`, see
[docs/hooks.md](docs/hooks.md)): the status when a session starts, the static check before
`git commit` and `git push`, and after a turn that changed files. Checks are commands, not prose;
the hooks only call them. Workers are briefed by pointer with `machinery brief <story>`.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md). `make` lists the targets; `make verify` is the full
verification.

## License

Apache-2.0. See [LICENSE](LICENSE).
