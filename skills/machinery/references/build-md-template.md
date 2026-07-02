# BUILD.md template

`BUILD.md` is the single deliverable. It must be self-contained: a coding agent with **zero** prior
context builds the system from this file alone, under hard TDD. Assume the reader has never seen the
domain model, the architecture, or the state machines. Inline what matters; reference the `design/`
files for the full source.

Fill every section. Omit a section only by writing "N/A" with a reason.

---

```markdown
# BUILD: <System Name>

## 1. Purpose and scope
One paragraph: what this system does, who uses it, and the one-sentence reason it exists.
In scope / out of scope as two short lists.

## 2. Glossary
The ubiquitous language (from the Modelith glossary and entity names). Define every term the
rest of the document uses. The reader has no other source for these words.

## 3. Domain model (the what)
- The entities, their definitions, and the relationships (paste the ER Mermaid from `modelith render`).
- The data dictionary: each entity's attributes and types. This is the ONE canonical schema; the
  architecture and the machines reference it, they do not restate it.
- The invariants, by id, each with its statement and its owner. These are non-negotiable rules.
- Source of truth: `design/domain.modelith.yaml` (lints clean).

## 4. Architecture (the how)
- System context and container diagrams (Mermaid C4 or the Structurizr export).
- Technology stack per container, and why.
- Deployment topology (replicas, operators, HA), from the deployment view.
- The Architecture Contract (boundaries + allow/deny dependency rules). The coding agent must not
  introduce cross-boundary dependencies outside `allow`.
- Interface contracts at each boundary: request/response shape, enumerated errors, idempotency keys.
- Persistence and placement per stateful component (actor vs persisted aggregate; how concurrent
  events are serialized).
- Source of truth: `design/workspace.dsl` and `design/ARCHITECTURE.md`.

## 5. Behavior: the state machines (the logic)
For each stateful component, one subsection:
- A one-paragraph narration of its lifecycle in plain language.
- The XState machine (paste the JSON from `design/machines/<Component>.machine.json`).
- The named-unit contract table: every guard, action, and actor with its signature, its pre/post,
  and what it maps to (invariant id or C4 relationship). These are the units the coding agent implements.
- The failure catalog for this component: per failure, the detection (which invoke error or timeout),
  the transition, the recovery, and the C4 mitigation that bounds it (or the residual risk if none).

## 6. Traceability matrix
One table proving nothing was dropped between layers:

| invariant id | enforced by (guard / structural) | in component | interface contract | test id(s) |
|---|---|---|---|---|

Every invariant from section 3 appears here. Any invariant not enforced by a guard and not
structurally impossible is called out explicitly as a known risk.

## 7. Test specification (the hard-TDD oracle)
The transition matrix, flattened to test cases. One row per transition and per guard branch:

| test id | component | given state + context | event | expected next state | expected actions | derived from |
|---|---|---|---|---|---|---|

Plus contract tests per boundary (from section 4) and property tests for each invariant.
This section is the input to the test-writer agent. It writes tests from here; it does not invent them.
Reference `@xstate/graph` covering paths for completeness.

## 8. Build plan
- Walking skeleton first: the thinnest end-to-end slice that exercises one real transition through one
  real boundary. Prove the topology before adding breadth.
- Then vertical slices, one component lifecycle at a time, each slice green before the next.
- Milestone list with a definition of done per milestone (all transitions covered, all invariants
  property-tested, contract tests green, no cross-boundary violations).

## 9. Language realization notes
Target language(s): <...>. How the machines become code:
- Elixir: `gen_statem` or a GenServer per aggregate under a Registry and a supervisor.
- Go: explicit state field + a transition switch, persisted state + optimistic lock; a library only if it earns it.
- Rust: typestate or an enum + match; persistence + lock as above.
- TypeScript: XState directly; the machine JSON is nearly the implementation.
- Python: an explicit state field + a transition table; persistence + lock as above.

## 10. Hard-TDD protocol (read this before writing any code)
1. A test-writer agent reads sections 6 and 7 and writes the full test suite from the spec.
2. The tests are then LOCKED. The implementer agent may not modify them to make them pass.
3. The implementer agent writes the code until the locked tests pass.
4. Every transition in section 5 has a test in section 7. Every invariant in section 3 is
   property-tested. Coverage target and gates as stated in the project conventions.
5. If a test is wrong, that is a design defect: stop, fix the design and this BUILD.md, regenerate
   the affected tests. Do not "adjust" a test to pass.

## 11. Open questions and residual risks
Anything deferred, any dependency with no mitigation, any invariant not structurally guaranteed.
Be explicit. A named risk is cheaper than a surprise.
```
