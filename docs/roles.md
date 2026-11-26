# Roles

A many-agent run drifts when one context plans, designs, builds and judges its own work. machinery
keeps six roles apart. Each has a prompt under `agents/`; `skills/machinery/SKILL.md` is the entry
point. This page says who may write what, what the tool verifies about each role, and which rules
are judgment. The contract is `docs/formats.md`, sections 15, 21 and 23.

## The roles

| Role | Prompt | Goal | May write | Never |
|---|---|---|---|---|
| Planner | `agents/machinery-pm.md` | The owner's goal, observed on the running system. | The backlog: intake, stories, decisions, questions, the wave freeze, the landing. It chooses tiers and statuses. | Code, tests, design sources. |
| Modeler | `agents/machinery-modeler.md` | A design that states the requirement, all of it and no more. | Design sources: model, machines, failure modes, the risk and status files, formal models. | Code, tests, stories. |
| Builder | `agents/machinery-builder.md` | The ids of one story observed on the real system. | Code and end-to-end tests in the story's scope, in one worktree; its proof. | The design, generated files, other stories' paths. |
| Verifier | `agents/machinery-verifier.md` | A verdict from its own observation. | Its prediction and its verdict. | Anything else. It did not build the story. |
| Checker | `agents/machinery-checker.md` | Pass or fail on a backlog, a design change, or drift. | Its review, with `machinery review`. | Fixes. |
| Delegate | `agents/machinery-delegate.md` | The owner's written answer to a question, or a hold. | An answer under a clause of the owner's policy. | Building, verifying, planning; the policy file. |

The delegate is experimental. The owner has not settled the concept, the policy file is the owner's
and lies outside the project, and section 15.9 may change in any release.

## What the tool verifies

| Role | Held by the tool | Not held: rests on the role |
|---|---|---|
| Planner | The order of intake: `machinery intake plan` refuses before the design is changed, sound, generated, tiered and modeled, and while a wave is frozen; it creates the stories and the capstone whole or not at all. `machinery waves` fails on a cycle, competing writers and a story without ids. `--freeze` waits for an approved backlog review and `intake plan --apply` for a design review, each from another context. A failed `machinery landing` marks the branch red. The ratchet reports a lowered tier and a review switched off. | That it wrote no code. That it reviewed its own backlog first. That the learnings were read (`intake plan` only reminds). That the stories are vertical slices. |
| Modeler | `machinery lint`, stage `gen-drift`, stage `formal` (reach predicates, mutations, baseline, sync), stage `risk`, and the ratchet on a vital area that states less. | That the design says what the requirement asked. This is the checker's design review. |
| Builder | Stages `tests`, `trace` and `arch`; stage `scope` (`machinery check --base <base> --story <id>`, and the commit hook on a branch named after the story): every changed file inside the story's scope, no design source outside the intake's design story; stage `contract` on a design source changed during a frozen wave; `machinery verify` on evidence at the commit; the tier gate of `machinery story`; the commit and push hooks. | That the scope it was given is the right one. That it ran the check with its story where no hook does. That it posted what it saw. |
| Verifier | A prediction for the commit before the verdict, by an actor who posted no proof for it, and not from the builder's session and agent (`docs/hooks.md`). | What it read before predicting. That it made the observation. |
| Checker | Its verdict, once recorded with `machinery review`: a rejection names one to five issues, each with its rule and scope, and a review from the planner's session and agent is refused. | What it read, what it missed, and whether each issue names its whole class. |
| Delegate | Every condition of the clause: topics, reserved topics, effect, limit, the sum of earlier amounts. The answer comes from an agent named `*-delegate` that did not ask the question, and is a vetoable decision. | That the agent named a delegate is the delegate: whoever spawns it sets the name. That a hold was preferred in doubt. |

A cell in the right column is where a run can drift without a finding. The rules below cover those
cells.

## When the checker is called

At set points, by the planner, each time in a fresh context, and each review is recorded with
`machinery review` (`docs/formats.md` section 15.10):

- before a wave is dispatched, on the backlog: `machinery review backlog`, which
  `machinery waves --freeze` waits for;
- after a design change, on the change against its requirement: `machinery review design <id>`,
  which `machinery intake plan <id> --apply` waits for; after the plan, on its stories:
  `machinery review intake <id>`;
- periodically, on drift: `machinery assess` on the project itself, with backlog, design and code
  compared. After a landing, and before a release, are the natural moments.

Not continuously. A reviewer that follows the work becomes its co-author and stops seeing it.

## When the delegate is called

Sparingly: for a question `machinery questions` lists that holds up a wave while the owner is away,
and only when `delegate.policy` is set. Without a clause that covers the question it holds, and the
question waits for the owner.

## Model routing

The design roles (planner, modeler, checker, delegate) run on the strongest class of model. Builders
run on a standard class; the core of a vital area is built on the strongest. A verifier runs on a
class at least as strong as its builder's. `routing` in `machinery.yaml` holds the text per tier
that `machinery brief` and `machinery waves` print. The prompts name no vendor model: the harness
maps a class to a model.

## Rules that are judgment

The tool enforces none of these. Each stays in a prompt while its failure still happens. Retire one,
here and in the prompt, when the failure is no longer seen, or when a command comes to enforce it.

| Rule | Role | Failure it prevents |
|---|---|---|
| Never builds. | Planner | Nobody watches the plan; one author's code is checked by nobody. |
| Reads learnings and files observations before planning. | Planner | The same mistake in the next wave; a defect seen once and lost with a context. |
| Walking skeleton in the first wave. | Planner | Integration fails at landing, after every story was accepted. |
| Stories are vertical slices. | Planner | A layer story has nothing to observe, so its proof is deferred to the last story. |
| Reviews its own backlog, then calls the checker, before a freeze. | Planner | One planning error multiplied by the width of the wave. |
| Briefs by pointer; pastes no context. | Planner | The planner's assumptions reach builder and verifier alike, and the verifier is no longer independent. |
| A goal not met with every story closed is a new intake. | Planner | "All closed" reported as done; stories patched away from the design. |
| Designs from the caller's side. | Modeler | A design that mirrors the implementation and lints clean. |
| Settles the riskiest shared premise before fan-out. | Modeler | One wrong assumption built into every worktree at once. |
| Every id has a sentence of the requirement behind it. | Modeler | Invented scope that becomes owed stories and looks legitimate from then on. |
| Never changes the design; a design change is an intake. | Builder | The design follows the code; the contract moves under parallel work. |
| Sees the check fail first. | Builder | A test that observes nothing. |
| Posts limits, learnings and observations. See something, say something. | Builder | A proof read as more than it shows; a defect fixed silently out of scope, or not reported at all. |
| Predicts before reading the proof. | Verifier | Agreement with the builder's framing in place of a test of the claim. |
| Writes nothing. | Verifier, Checker | A change that no one verified, made by the one who was to verify. |
| Pass or fail, nothing between. | Checker | A conditional pass treated as a pass. |
| At most five issues a round. | Checker | A long list patched line by line, the important issue buried. |
| Each issue states the general rule and its scope. | Checker | The instance fixed and the class left; review rounds that never converge. |
| A design change is read for omission, invention and contradiction. | Checker | Gaps and extras that enter as ids and are then proven faithfully. |
| Called at set points, in a fresh context. | Checker | A reviewer anchored on the work it watched being made. |
| Holds unless a clause covers the question. | Delegate | An agent deciding what is the owner's to decide. |
| Design roles on the strongest class; a verifier at least at its builder's. | All | Cheap design errors that every story repeats; a verifier that cannot disprove. |
