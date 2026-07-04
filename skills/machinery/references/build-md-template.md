# BUILD.md template

`BUILD.md` is the single deliverable. It operates in one of two modes, and must say which:

- **Full mode** (the default): one self-contained document. A coding agent with **zero** prior
  context builds the system from this file alone, under hard TDD. Assume the reader has never seen
  the domain model, the architecture, or the state machines. Inline what matters; reference the
  `design/` files for the full source.
- **Manifest mode** (sharded designs, see the skill's "Sharding large designs"): the root BUILD.md
  is an entry-point manifest over `design/` and the shards `design/BUILD/<context>.md`. The root
  carries the glossary, the Architecture Contract, the traceability matrix, and the cross-context
  test spec; each shard carries sections 5 to 9 for its context. The zero-context claim then applies
  to the design tree as a whole, and self-containment applies per shard.

Two artifacts are never pasted by hand: the machine JSON (section 5 references the machine files)
and the transition tables (section 7 references the generated oracles). Pasted copies drift; the
files are what the deterministic gates check.

Fill every section. Omit a section only by writing "N/A" with a reason.

---

```markdown
# BUILD: <System Name>

Mode: full (self-contained) | manifest (root of a sharded design; shards in design/BUILD/<context>.md)

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
  introduce cross-boundary dependencies outside `allow`; G4-import enforces this against the code.
- Interface contracts at each boundary: request/response shape, enumerated errors, idempotency keys.
- The event-contract table for multi-component designs (producer, consumer, payload by Modelith
  attribute, delivery, ordering, dedupe key). DB and bus coupling is invisible to import analysis;
  this table governs it.
- Persistence and placement per stateful component (actor vs persisted aggregate; how concurrent
  events are serialized).
- The NFR record: security posture, capacity assumptions, observability requirements (operator
  signal per residual failure state).
- Source of truth: `design/workspace.dsl` and `design/ARCHITECTURE.md`.

## 5. Behavior: the state machines (the logic)
For each stateful component, one subsection:
- A one-paragraph narration of its lifecycle in plain language.
- A reference to `design/machines/<Component>.machine.json`. Do NOT paste the JSON; the file is
  the source and the gates lint it there.
- The named-unit contract table: every guard, action, and actor with its signature, its pre/post,
  what it maps to (invariant id or C4 relationship), its test type (unit / integration / property),
  and its fixture (real dependency or fake, and which). These are the units the coding agent
  implements. Idempotency and side-effect contracts (the "charges once" class) are integration or
  property tests against the real dependency or a contract-tested fake, never derivable from
  transition tests: say so per row.
- The failure catalog for this component: per failure, the detection (which invoke error or timeout),
  the transition, the recovery, and the C4 mitigation that bounds it (or the residual risk if none).
The named-unit table and failure catalog live in `design/machines/<Component>.matrix.md`; inline
them or reference them, but the matrix file remains what G3 checks.

## 6. Traceability matrix
One table proving nothing was dropped between layers:

| invariant id | enforced by (guard / structural) | in component | interface contract | test id(s) |
|---|---|---|---|---|

Every invariant from section 3 appears here. Any invariant not enforced by a guard and not
structurally impossible is called out explicitly as a known risk. Invariant ids must appear inside
table cells as whole tokens: Gx-trace matches them structurally, so `inv-1` buried inside `inv-12`
does not count, and prose outside a table does not count.

## 7. Test specification (the hard-TDD oracle)
The transition test spec IS the generated `design/machines/<Component>.oracle.md` files. Do not
restate the transition tables here; reference the oracles. Tests key on each row's STABLE id, never
the sequential test id: row numbers renumber when the design changes, stable ids survive unrelated
insertions and change only when that transition's stimulus changes.

BUILD.md adds only what the oracles cannot derive:
- The guard-branch completeness analysis: one test per falsifying clause of each conjunction guard
  (the T-XXX-04a/b/c pattern). A guard `A AND B AND C` needs one test with only A false, one with
  only B false, one with only C false, each expecting the rejection path.
- The named-unit test plan: for each guard/action/actor row from section 5, its test type and
  fixture. Idempotency and side-effect contracts are integration or property tests against the real
  dependency or a contract-tested fake.
- Contract tests per boundary (from section 4) and property tests for each invariant.

This section is the input to the test-writer agent. It writes tests from here; it does not invent
them. `@xstate/graph` covering paths remain available for multi-step path tests.

## 8. State migration
For every machine whose placement row (section 4) says its state is persisted, the migration
protocol for future state changes: when a state is renamed, split, or removed, ship a mapping table
from old persisted values to new states, or an explicit drain rule for in-flight instances. If
nothing is deployed, state exactly that: "no persisted instances yet".
Brownfield runs this in reverse on day one: when a machine models an ALREADY-persisted legacy
lifecycle, the first version of this document must include the mapping table from every observed
legacy persisted value to a modeled state, plus an explicit rule for unmapped values (fail loudly,
never silently coerce).

## 9. Build plan
- Walking skeleton first: the thinnest end-to-end slice that exercises one real transition through one
  real boundary. Prove the topology before adding breadth.
- Then vertical slices, one component lifecycle at a time, each slice green before the next.
- Milestone list with a definition of done per milestone (all transitions covered, all invariants
  property-tested, contract tests green, no cross-boundary violations).

## 10. Language realization notes
Target language(s): <...>. How the machines become code:
- Elixir: `gen_statem` or a GenServer per aggregate under a Registry and a supervisor.
- Go: explicit state field + a transition switch, persisted state + optimistic lock; a library only if it earns it.
- Rust: typestate or an enum + match; persistence + lock as above.
- TypeScript: XState directly; the machine JSON is nearly the implementation.
- Python: an explicit state field + a transition table; persistence + lock as above.

### Toolchain and versions
Pin the environment so two implementing agents cannot diverge on it: language version, library
versions with exact pins or a lockfile instruction, test framework and version, and any codegen
tools (including how to run `machinery oracle` and `machinery check`).

## 11. Hard-TDD protocol (read this before writing any code)
1. RED precondition: run `machinery check design` (with this project's staged `--gate` list if it
   declares one) and require ZERO blocking findings before deriving any test. The oracles are the
   test spec; a red design means the spec itself cannot be trusted, and tests derived from it test
   the wrong things with confidence. Fix the design first, never the tests.
2. A test-writer agent reads sections 6 and 7 and writes the full test suite from the spec, keying
   transition tests on the oracle stable ids. A runtime that cannot spawn a fresh-context
   test-writer (no subagents) runs RED then GREEN sequentially with the same single agent; the
   derivation rule is unchanged (tests come from sections 6 and 7 and the oracles, never from
   implementation intentions), and the gate runs in steps 1 and 3 are what separate the phases in
   place of context isolation.
3. RED exit gate, all three deterministic checks required before anything locks:
   a. Coverage of the spec: every oracle row's stable id appears whole-token somewhere in the
      suite (verify with a grep per id; a missing id is a missing test), every guard-conjunction
      clause has its falsifying test (section 7), every invariant in section 3 has its property
      test.
   b. Architecture: `machinery check design --impl <impl-dir>` is green. G4-import skips test
      files but checks everything they import and every support file, so the compile skeleton,
      stubs, and scaffolding the tests stand on already respect the Architecture Contract (put
      test scaffolding under the contract's `ignore:` paths). A suite that only compiles against
      a boundary violation would force the implementer to reproduce that violation to go green.
   c. The suite RUNS and is red for the right reason: failing assertions on missing behavior,
      never compile or import errors inside the tests themselves.
   Together these are the guarantee: the spec is gate-checked, the suite's coverage of the spec is
   id-checked, and the suite's own skeleton respects the architecture, so the implementer has no
   correct move except delivering the designed behavior inside the designed boundaries.
4. The tests are then LOCKED. The implementer agent may not modify them to make them pass.
5. The implementer agent writes the code until the locked tests pass.
6. GREEN acceptance bar, both together: the locked suite passes AND
   `machinery check design --impl <impl-dir>` is green again. Code that passes the tests by
   crossing a boundary fails the gate; code that respects the boundaries but fails a test is not
   done. Coverage target and any further gates as stated in the project conventions.
7. Generated tests live apart from hand-written tests (a marker comment or a directory), so
   regenerating them on a design change never clobbers hand-written ones.
8. If a test is wrong, that is a design defect: stop, fix the design and this BUILD.md, rerun
   `machinery oracle`, and regenerate the affected tests (the stable-id diff is the affected-test list).
   Do not "adjust" a test to pass.

## 12. Open questions and residual risks
Anything deferred, any dependency with no mitigation, any invariant not structurally guaranteed.
Be explicit. A named risk is cheaper than a surprise.
```
