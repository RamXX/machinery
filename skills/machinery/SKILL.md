---
name: machinery
description: Use when a project has a machinery.yaml, or when the user asks to design, plan, verify or adopt with machinery. machinery joins a design (Modelith model, XState machines, contracts) to evidence from real end-to-end runs, turns what is unproven into backlog stories, and gates each story by its risk tier. Covers greenfield and adoption in an existing project.
---

# machinery

machinery is a command-line tool. Every check is a command: run it and read its findings. Do not
restate its rules from memory, and do not read the long documents unless a step sends you there.

## The rules that never bend

- Evidence comes from the real system. No unit tests, no mocks, no stand-ins. Tests live under
  the project's e2e roots and drive the real binary or service.
- Design first: change the model and the machine, then the code.
- Never weaken a test, a property or a baseline to get a pass. A stage that is
  `not demonstrated` blocks exactly like `failed`.
- Never edit generated files (the gen directory, formal skeletons, traces, result files).

## Which situation are you in

| Situation | Start with |
|---|---|
| No `machinery.yaml`, new project | `machinery init`, then "Design" below. Guide: `docs/getting-started.md`. |
| No `machinery.yaml`, existing code and backlog | `machinery assess --config <hand-written yaml outside the project>` (read-only report), then `machinery adopt --paths '<area>/**'` (dry run), then `--write`. Guide: `docs/brownfield.md`. |
| You are planning work (PM) | "Plan" below. Guide: `docs/pm.md`. |
| You were handed a story | `machinery brief <story>` and follow it. |
| You are verifying a story | `machinery brief <story> --role verifier` and follow it, in order. |
| You are writing a formal model | `machinery brief <story> --role modeler`. Guide: `docs/formal.md`. |

## Design

1. Edit the Modelith model (`*.modelith.yaml`) and, for anything whose legal next step depends on
   its past, a machine (`*.machine.json`, format: `docs/machine-format.md`).
2. `machinery lint` until it passes. `machinery gen` writes tables and the obligation list.
3. Every new obligation is either proven or owed: `machinery owed sync` (or
   `machinery adopt --write` in an adopted project).

## Plan (the PM role owns the backlog and the risk file)

1. Assign risk tiers in `design/risk.yaml`: routine by default, consequential for authorization,
   money, published contracts and migrations, critical for races, recovery and structural rules.
   When unsure, the higher tier.
2. `machinery plan --apply` turns owed ids into backlog stories with dependency edges.
3. `machinery waves` gives the parallel schedule. Before fan-out: `machinery waves --freeze`.
4. Hand each story out with `machinery brief <story>`. Route by tier: the brief says which class
   of model fits. Give workers the brief, not your context.

## Build one story

1. Write the end-to-end check first; see it fail for the right reason.
2. Make it pass. Per change: `machinery check --base <base>`, then
   `machinery run --affected --base <base> -- <e2e command>` and `machinery verify --story <id>`.
3. `machinery owed sync --prune-only`, commit, `machinery proof --story <id> --post`.

## Accept and land

- `machinery story <id>` prints the gate for the story's tier and what is missing.
- Consequential and critical stories need a fresh verifier: `machinery predict` before reading
  the proof, then `machinery accept`.
- Run the whole suite once, at the head that is ready: `machinery landing --base <base>`. At most
  two fix-forwards; then return the failing stories to the backlog.
- `machinery probe -- <e2e command>` per batch: a surviving mutation is a weak test, fix the test.

## When a command fails

Read the finding: it names the id or file and what to do. If the contract is unclear, the answer
is in `docs/formats.md` under the section the finding belongs to (`machinery checks` lists the
stages and what each protects). If a finding is outside your story, say so and hand back.
