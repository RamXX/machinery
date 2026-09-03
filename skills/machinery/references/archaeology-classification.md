# Archaeology and legacy corpus classification

Read this before using an existing repository as design evidence. Code shows
what exists; it does not decide what the target should preserve.

## Classify before architecture

Inventory each coherent legacy corpus area, such as a subsystem, service,
schema family, public API, workflow, test corpus, migration, or operational
tool. Record exactly one intent classification:

- `preserve-port`: behavior and contract are intentional; preserve them while
  porting or rebuilding the implementation.
- `rearchitect-adapt`: the capability remains, but its current boundaries or
  behavior are not the target; adapt it to an explicit target decision.
- `learning-only`: use it as evidence and vocabulary, never as an obligation.
- `historical-only`: retain for history; it supplies no target requirement.
- `unresolved`: the owner has not decided. Record the question and owner.

Missing classification never means preserve or port. `unresolved` blocks the
architecture handoff because architecture cannot honestly fix boundaries while
the desired destination is unknown.

For each area record: path or stable identifier, classification, evidence
source, rationale, target owner/component when applicable, and the deciding
person/date. Keep capability disposition in `legacy/surface.yaml`; this corpus
classification is the intent layer that tells the conductor how to interpret
the evidence behind those rows.

## Conversation boundary

Do not ask the user about gates or classifications as internal jargon. Explain
the concrete choice: what behavior survives, what is redesigned, what is kept
only for learning, what is merely historical, and what remains undecided. Name
the risks and deliverables. Record the result in the ledger vocabulary only
after the user decides.

Use codebase-memory graph tools for the opening and closing inventories when
available. Source discovery may reveal another area, but it may not assign that
area's intent.
