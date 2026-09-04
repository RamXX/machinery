# Manifest execution packets and shards

Read this when the implementation is large enough that one zero-context
`BUILD.md` would overload an execution model, or where domain-oriented teams
need reusable context across several milestones. The design declares which
linkage contract it uses.

## Layout

Use full mode only while one self-contained `BUILD.md` remains genuinely
narrow. Otherwise use manifest mode:

```text
design/
  BUILD.md
  BUILD/
    M0-walking-skeleton.md
    M1-orders.md
    M2-payments.md
```

The root is the single milestone, demo, ordering, and acceptance manifest. It
contains the system purpose, global ordering constraints, and one block per
milestone.

```markdown
# BUILD: Example

Mode: manifest
Linkage: pairwise

## Build plan

**M0 - Walking skeleton**
Status: open
Packet: [M0 execution packet](BUILD/M0-walking-skeleton.md)
Demo: Submit one valid order and observe its durable accepted result.
NFR: Stable error envelope, trace id, bounded request and retry budgets.
DoD: `ORDER-a1b2c3` passes at the real datastore boundary.

**M1 - Order lifecycle**
Status: open
Packet: [M1 execution packet](BUILD/M1-orders.md)
Demo: Exercise create, cancel, and timeout recovery from the public surface.
DoD: `ORDER-d4e5f6` and its falsifying-clause tests pass.
```

## Linkage choice

Use `Linkage: pairwise` when each milestone can be handed to an executor as one
bounded, self-contained file. Each milestone has exactly one `Packet:` link,
and every non-index file under `BUILD/` is linked exactly once. If `Linkage:` is
absent, Machinery preserves this original pairwise behavior.

Use `Linkage: matrix` when milestones are build stages that legitimately span
domain handoff documents, or one domain shard serves several stages. Each root
milestone has one or more `Shard:` links, and every shard has exactly one
reciprocal, canonical `Milestones:` line:

```markdown
# BUILD: Example

Mode: manifest
Linkage: matrix

## Build plan

**M0 - Walking skeleton**
Shard: [core](BUILD/core.md)
Shard: [trust](BUILD/trust.md)
Demo: Submit one valid order and observe its durable accepted result.
NFR: Stable error envelope and trace id.
DoD: `ORDER-a1b2c3` passes at the real datastore boundary.

**M1 - Operations**
Shard: [core](BUILD/core.md)
Shard: [operations](BUILD/ops.md)
Demo: Observe the completed operational workflow.
DoD: all operations rows pass.
```

`BUILD/core.md` then carries `Milestones: M0, M1`; `BUILD/trust.md` carries
`Milestones: M0`; and `BUILD/ops.md` carries `Milestones: M1`. Ids are unique,
numerically ascending, and separated by comma-space. The root-to-shard and
shard-to-root graphs must match exactly. Duplicate pairs, missing files, orphan
shards, and unknown milestone ids are errors.

## Pairwise packet contract

Each packet is narrow, self-contained Markdown with these sections:

1. `Outcome`: the user-visible result and demo.
2. `Domain context`: exact entities, attributes, invariants, actions, and
   legacy dispositions needed for this milestone.
3. `Architecture context`: owned components, allowed interfaces, dependencies,
   persistence, concurrency, and failure mitigations.
4. `Behavior and oracles`: states, transitions, guards, refusal behavior,
   oracle paths and exact stable ids.
5. `TDD and implementation`: test order, locked tests, files to create/change,
   APIs, algorithms, and commands.
6. `Risks and recovery`: faults, bounds, rollback, crash recovery, and residual
   risks.
7. `Acceptance`: executable DoD, evidence to capture, and the acceptance file.

Copy the necessary rows into the packet and hold copied tables with
`machinery:embed`; do not tell the executor to read the root, architecture,
domain model, or another packet for missing context. Links may provide provenance
but never carry an implementation prerequisite.

Keep a packet below 64 KiB. If it cannot fit, split the milestone by a user-
demonstrable boundary and give each new milestone its own packet. Never solve
size by deleting risks, negative paths, oracle ids, or acceptance obligations.

## Matrix shard contract

A matrix shard is a reusable domain handoff document, not a zero-context
milestone packet. It carries exactly one `Milestones: M1, M3` declaration and
enough domain-specific implementation context to be used with the root
manifest. Pairwise filename, title, seven-section, milestone-marker, and 64 KiB
rules do not apply. This is deliberate: the execution context is the root plus
one selected workstream shard, the execution unit is one milestone-shard pair,
and milestone acceptance remains keyed by the root milestone number.

## Execution discipline

In pairwise mode, the execution model reads one packet plus the files it
explicitly owns. In matrix mode, it reads the root manifest plus the selected
workstream shard and works only their milestone-shard intersection. It writes
RED tests from that context and committed oracles, proves coverage with
Machinery, then implements GREEN without altering locked tests. It returns
evidence, not a narrative claim of completion. Acceptance remains a separate
review bound to one commit.
