# Milestone execution packets

Read this when the implementation is large enough that one zero-context
`BUILD.md` would overload a small execution model. The design may be authored
with a frontier model; each milestone must remain executable by a smaller model
without reading the rest of the repository or reconstructing design intent.

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

The root is a milestone and demo manifest. It contains the system purpose,
global ordering constraints, and one block per milestone. It does not carry the
implementation detail that belongs to packets.

```markdown
# BUILD: Example

Mode: manifest

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

Each milestone links to exactly one packet. Every packet is linked once. Do not
reuse a context shard across milestones: that forces a small model to separate
unrelated work and makes acceptance ambiguous.

## Packet contract

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

## Execution discipline

The execution model reads one packet plus the files it explicitly owns. It
writes RED tests from the packet and committed oracles, proves coverage with
Machinery, then implements GREEN without altering locked tests. It returns the
packet's evidence, not a narrative claim of completion. Acceptance remains a
separate review bound to one commit.
