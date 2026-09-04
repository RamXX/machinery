---
name: machinery
metadata:
  version: "0.6.7"
description: >
  Design software as a build-ready blueprint for greenfield, brownfield,
  rebuild, or hybrid work. Use for domain modeling, C4 architecture, state
  machines, formal verification, legacy adoption, and zero-context BUILD
  handoffs under hard TDD. Do not use for ordinary implementation-only tasks.
---

# machinery

Turn product intent and existing evidence into a checked design that an
implementation agent can execute under hard TDD. The deliverable is the design,
not production code.

This entrypoint is deliberately short. Read only the references selected by the
current phase; do not load the entire methodology at once.

## Runtime contract

Use capabilities, not host names. The skill, artifacts, and `machinery` CLI are
portable. Subagents and hooks are optional accelerators:

- Use fresh-context `machinery-fsm-author` and `machinery-build-writer` roles
  when available; otherwise execute their installed instructions inline.
- Hooks may protect generated files and run checks. Without hooks, treat
  generated artifacts as read-only and run checks explicitly.
- In archaeology, use codebase-memory graph tools before broad filesystem
  discovery whenever they are available.
- Keep artifacts free of host-specific commands, models, sockets, or metadata.

The design contract is identical across Claude Code, Codex, OpenCode, and other
Agent Skills runtimes.

## Start and route

1. Read `design/STATE.md` and `design/DECISIONS.md` if they exist.
2. Classify the run: greenfield, brownfield, rebuild, or hybrid.
3. State the system purpose, users, target languages, and current open decision.
4. Read the reference for the work now in front of you:

| Work | Required reference |
|---|---|
| Domain model | the installed `domain-model-author` instructions; use `modelith lint` |
| Architecture and boundaries | [references/c4-standalone.md](references/c4-standalone.md) |
| Human-facing actions | [references/target-surfaces.md](references/target-surfaces.md) |
| Machine authoring | [references/xstate-format.md](references/xstate-format.md) |
| Formal, checker, attestation, acceptance, or import findings | [references/verification-evidence.md](references/verification-evidence.md) |
| Small build handoff | [references/build-md-template.md](references/build-md-template.md) |
| Large or small-model execution handoff | [references/execution-packets.md](references/execution-packets.md) |
| Existing-code intent | [references/archaeology-classification.md](references/archaeology-classification.md) |
| Rebuild or hybrid migration | [references/rebuild-guide.md](references/rebuild-guide.md) |
| Legacy capability inventory | [references/surface-ledger.md](references/surface-ledger.md) |
| CLI commands and gate names | [tools/README.md](tools/README.md) |

Do not inspect Machinery source to discover an artifact schema. The selected
reference must be sufficient. If it is not, report the missing instruction as a
Machinery defect instead of reverse-engineering a private contract into the
client design.

## Traceability spine

Hold these exact relationships through every phase:

| Domain | Behavior | Architecture |
|---|---|---|
| status enum value | state | owning component |
| action | event and transition | component performing it |
| invariant id | guard or structurally impossible edge | enforcing component |
| attribute | typed context reference | datastore schema |
| side effect | invoke with success, failure, and timeout | dependency plus mitigation |
| scenario | path and test case | participating interfaces |

Modelith is the single source of truth for data. C4 and machines reference it;
they do not redefine it.

## The four handoffs

### Domain

Interrogate breadth-first: vocabulary and entities, then invariants/actions,
then happy and negative scenarios. Sweep ubiquitous, event-driven,
state-driven, optional, and unwanted behavior. Every lifecycle has a status
enum; every invariant has an owner and a carrier or reasoned waiver.

Run `modelith lint` and `machinery check <design> --gate gc`. Do not advance on
errors or warnings.

### Architecture

Author `workspace.dsl`, `ARCHITECTURE.md`, and `surfaces.yaml`. Fix component
ownership, allowed and denied crossings, interface shapes, persistence,
concurrency, dependency failures, mitigations, NFRs, and human surfaces. Treat
technology adoption as a closure, not a product name.

For existing systems, finish the intent classifications before this handoff.
An unresolved corpus area blocks architecture; missing classification never
means must-port.

Run `machinery check <design> --gate g2,gu` plus each artifact-activated gate,
and `machinery verify-c4 <design>`. Record the required attestation rows.

### Behavior

Author one machine per lifecycle or operational envelope, its named-unit
matrix, its oracle, and its formal semantics. Every dependency failure remains
a transition even when architecture mitigates it. Every fully guarded handler
states refusal behavior; every resting state states ignored events.

Run `machinery oracle`, `machinery check <design> --gate g3`, and
`machinery verify-formal <design>`. Read the verification reference for all
four semantics patterns, including `control-flow-only`.

### Build handoff

The build plan must be executable without design archaeology. Full mode is for
a genuinely narrow project. Large work uses a root milestone/demo manifest and
declares its linkage. Pairwise linkage gives every milestone exactly one bounded,
self-contained packet for a smaller execution model. Matrix linkage gives
milestones and reusable domain shards an exact reciprocal many-to-many graph;
each execution unit is one milestone-shard pair and reads the root plus that
workstream shard.

Run the full `machinery check <design>` and, once code exists,
`machinery check <design> --impl <dir>`. A green design is the RED precondition;
the locked tests and the same check are the GREEN acceptance boundary.

## Evidence and closure

- Generated artifacts are committed with their sources and never hand-edited.
- `attestations.yaml` covers the full subject inventory for every judgment.
- A milestone closes only with `acceptance/M<n>.yaml` bound to a reviewed commit.
- Brownfield oracle failures are adjudicated as code-is-truth or
  model-is-truth; neither is silently normalized.
- Run `machinery check` with zero errors, drift, or warnings before handoff.
- If a Paivot nd story owns the work, append the current `nd_contract` with
  command evidence and per-AC proof; use the standard story transition commands.

## Conversation and decisions

Speak to the user about intent, design choices, risks, and deliverables in plain
language. Keep phase numbers, gate ids, schemas, and CLI detail in artifacts and
operator output unless the user asks for them.

Batch questions, echo load-bearing interpretations, and stop interrogating when
the current handoff is decided and clean. Record decisions at answer cadence in
`DECISIONS.md` as `<date> <who>: <decision>`. Author-proposed decisions remain
explicitly unconfirmed. Record phase status, open questions, check evidence, and
the five-part adversarial self-review in `STATE.md`.

## Scale and decomposition

Use milestone packets before recursive decomposition. Recurse only when the
domain language forks, a subsystem team needs an isolated contract, or formal
composition no longer scales. Contract-pack children remain complete Machinery
designs. Boundary changes are parent-owned pack amendments, never unilateral
child edits.

For very large designs, `machinery scale <design>` informs the choice; team
isolation remains a human decision it cannot infer.

## Upgrade discipline

Upgrade the binary, skill, roles, plugin manifests, and generated artifacts as
one versioned change. Run `machinery doctor`; any stale cache, invalid receipt,
or mismatched skill version is a release blocker. Regenerate artifacts in a
dedicated upgrade change and classify stable-id churn before design changes.

`machinery update [--version <tag>]` refreshes the binary and recorded direct
agent homes. Host plugin caches remain host-owned and must also be refreshed
through the host plugin manager.
