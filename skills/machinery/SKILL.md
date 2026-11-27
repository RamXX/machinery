---
name: machinery
description: Use when a project has a machinery.yaml, or when the user asks to design, plan, build, verify or adopt with machinery. machinery joins a design (Modelith model, XState machines, contracts, failure modes, risk tiers) to evidence from real end-to-end runs, turns what is unproven into backlog stories, and gates each story by its risk tier. Names the role a context is in (planner, modeler, builder, verifier, checker, delegate) and the command that says what to do next. Covers greenfield and adoption in an existing project.
---

# machinery

machinery is a command-line tool. Whatever it can check is a command: run it and read its findings
instead of recalling a rule. This page tells you which role you are in and which command says what
to do next. `docs/formats.md` is the contract; open it at the section a finding names, not before.

## What must never happen

- Evidence from anything but the real system: no unit tests, no mocks, no stand-ins.
- Stories or code before the design. A requirement changes the design first
  (`machinery intake new`).
- A test, a property, a tier or a baseline weakened to get a pass. A stage that is
  `not demonstrated` blocks exactly like `failed`.
- A generated file edited by hand (the gen directory, formal skeletons, traces, result files).
- A claim beyond the evidence. Proven means observed on the running system at this commit.

## Which role you are in

One role per context. A many-agent run drifts when roles blur: the builder who adjusts the design,
the planner who patches code, the verifier who read the proof first. `docs/roles.md` says who may
write what, what the tool verifies about each role, and why each rule exists.

| Role | Owns | Never | Prompt |
|---|---|---|---|
| Planner | The backlog: intake, tiers, waves, landing. | Builds. | `agents/machinery-pm.md` |
| Modeler | The design sources. | Writes code or stories. | `agents/machinery-modeler.md` |
| Builder | One story in one worktree. | Changes the design. | `agents/machinery-builder.md` |
| Verifier | The verdict on a story it did not build. | Edits anything. | `agents/machinery-verifier.md` |
| Checker | A pass or fail review at set points. | Fixes what it finds. | `agents/machinery-checker.md` |
| Delegate (experimental) | An answer under the owner's policy, or a hold. | Anything else. | `agents/machinery-delegate.md` |

Alone in one session you still take the roles in turn, and a consequential, critical or vital story
still needs a verifier in a fresh context.

## The command that says what is next

| Question | Command |
|---|---|
| Where does the project stand? | `machinery status` |
| What does this requirement still lack? | `machinery intake status <id>` |
| What can run now, and in parallel? | `machinery waves` |
| What is my assignment? | `machinery brief <story>`, with `--role verifier` or `--role modeler` |
| What does this story still lack? | `machinery story <id>` |
| What waits for the owner? | `machinery questions` |
| What does a stage protect? | `machinery checks` |

## Where to start

| Situation | Start with |
|---|---|
| No `machinery.yaml`, new project | `machinery init`. Guide: `docs/getting-started.md`. |
| No `machinery.yaml`, existing code and backlog | `machinery assess --config <a hand-written yaml outside the project>` (a read-only report), then `machinery adopt --rehearse --config <the same yaml>` (adoption on a copy: what it writes, owes and what blocks it), then `machinery adopt --paths '<area>/**'` (a dry run), then `--write`. Guide: `docs/brownfield.md`. |
| A Go suite whose tests name their ids | `machinery run --adapter go-test-json -- go test ./...`; behind a wrapper, add `--streams`. It never proves a `tc` id. Guide: `docs/brownfield.md`. |
| A new requirement | `machinery intake new "<title>" --intent "..." --refuses "..."`, before any story or code. Guide: `docs/pm.md`. |
| You were handed a story | `machinery brief <story>`, and run its lines in order. |
| You are asked to verify a story | `machinery brief <story> --role verifier`. Predict before you read the proof. |
| Something is owed and has no story | `machinery owed list`, then `machinery plan --apply`. |

## The loop, one line per role

1. Planner: record the requirement (`machinery intake new`), choose the tiers.
2. Modeler: change the model, the machines, the failure modes, the risk file and the formal model of
   a critical area; `machinery lint`, `machinery gen`; commit the design alone.
3. Checker: design review against the requirement (omission, invention, contradiction), recorded
   with `machinery review design <id> --verdict approved|rejected`.
4. Planner: `machinery intake plan <id> --apply` writes every story and the capstone at once. Review
   the backlog; the checker reviews it (`machinery review backlog`); then `machinery waves --freeze`
   and one brief per worker.
5. Builder: see the end-to-end check fail, make it pass, `machinery verify --story <id>`, then
   `machinery proof --story <id> --post` with limits, learnings and observations.
6. Verifier, for a consequential, critical or vital story: `machinery predict`, read the proof,
   observe, `machinery accept`.
7. Planner: `machinery waves --thaw`, `machinery landing --base <base>`, close the intake once its
   capstone is accepted, then read `machinery learnings` and run `machinery observations --file`
   before the next plan. Stories all closed and the goal not met means a new intake.

A decision is recorded with `machinery decide`; a choice only the owner can make is asked with
`machinery ask`, and the work goes on elsewhere meanwhile.

## Model routing

Planner, modeler, checker and delegate run on the strongest class of model: a wrong plan or design
is repeated by every story built on it. Builders run on a standard class. A verifier runs on a class
at least as strong as its builder's. The `route:` line of a brief and of `machinery waves` gives the
text for the story's tier; a project changes it under `routing` in `machinery.yaml`.

## When a command fails

Read the finding: it names the id or the file and what to do. If the rule is unclear, the answer is
in `docs/formats.md` under the section the finding belongs to. A finding outside your role or your
story is not yours to fix: say so and hand back (a builder posts it as an observation).
