# Adopting machinery on a brownfield codebase, as a team

This guide is for a small team (roughly 2 to 8 developers) adopting machinery on an
existing repository that has grown messy, with the goal of clawing back to a sustainable,
gated model. The greenfield pipeline in `skills/machinery/SKILL.md` still applies; this
document covers what changes when the code already exists and when more than one person
works the design at once. Every recipe here was verified against machinery v0.1.0.

## 1. Calibrate expectations first: what the tool sees, and what it never sees

Adoption fails fastest on miscalibrated expectations, so start here.

| Concern | Who checks it |
|---|---|
| Design internally consistent (machines vs domain model vs contract) | `machinery check` (G2, G3, Gx), deterministic |
| Committed oracles and formal artifacts fresh | G3 and `verify-formal`, deterministic |
| Code respects declared import boundaries | G4, deterministic, **imports only** |
| Code behavior matches the machines (states, events, guards) | **Characterization tests you write from the oracles. No gate reads your code's behavior.** |
| Guard semantics, action ownership, NFRs, zero-context claim | A person or agent attests; the tool cannot |

Three consequences:

- A fully green `machinery check --impl .` on day one means the design is coherent and the
  import graph matches the contract. It says nothing about whether the code behaves like
  the machines. Behavioral drift surfaces only when you run oracle-derived tests against
  the real code (section 3, stage 3).
- G4 cannot see coupling through shared database tables or bus topics. The event-contract
  table in ARCHITECTURE.md is the governing artifact for those seams, and it is maintained
  by hand.
- Every gate has an LLM-attested half, spelled out per gate in SKILL.md. On a team, those
  attestations need a named owner (section 6), or they silently become checked by nobody.

## 2. Fit: model the stateful core, not the whole repo

machinery earns its cost on lifecycle, protocol, retry, saga, and workflow logic: the
order lifecycle, the payment saga, the background job runner, the sync engine, the
operational envelope of a service. Pure transforms, CRUD screens, and UI flows get a
contract spec and ordinary tests, not a machine and not the four-phase pipeline. Pick the
two or three state-bearing subsystems where bugs actually hurt, and start there. If a
subsystem has no interesting states, machinery is the wrong tool for that subsystem, and
using it anyway will read as ceremony and burn the team's goodwill.

## 3. The adoption ladder

Do not attempt all gates at once. Each stage below is independently green in CI, so the
team gets a ratchet, not a big bang. The stage you are on is encoded in one place: the
`--gate` list in your CI invocation.

### Stage 0: pin the toolchain, create the design directory

- Install a pinned machinery release (`MACHINERY_VERSION=v0.1.0`) or
  `go install github.com/RamXX/machinery/cmd/machinery@v0.1.0`. Pin modelith too
  (`v0.4.0`; `make install-modelith` installs the pinned release, and
  `machinery preflight` warns when the installed version does not match the pin. Keep the
  pin in your CI file as well).
- Create `design/` at the repo root, per the SKILL.md output layout. It is versioned with
  the code, in the same repo, and changes to it go through the same PR review as code.

### Stage 1: boundary baseline (gates g2 and g4, day one)

The goal of this stage: the intended clean boundaries are written down, today's violations
are explicitly recorded, and CI fails on any NEW kind of violation. This is achievable in
one day and needs no domain model and no machines.

1. Author `design/workspace.dsl` and `design/ARCHITECTURE.md` with an Architecture
   Contract (v2) describing the boundaries you INTEND, one boundary per package or package
   group you are claiming. Ids are dot-separated segments, each starting with a letter or
   underscore and continuing with letters, digits, underscores, and hyphens.
2. Everything you have not modeled yet needs BOTH of these, not one or the other:
   - `ignore:` globs covering the unmodeled packages' own source files, so G4 does not
     error on every file that maps to no boundary.
   - For unmodeled internal packages that modeled code still imports: declare a single
     external, for example `external.rest_of_monolith`, list those packages under its
     `imports:` prefixes, add `allow: [yourboundary -> external.rest_of_monolith]`, and
     give it a mitigation row whose posture is recorded as "own unmodeled code, out of
     scope until modeled". G2 requires the row even for your own code; write it truthfully
     rather than fighting it. The `imports:` prefixes govern only the edges INTO that code;
     they do not exempt its files, which is what the `ignore:` globs are for.
3. Baseline today's real violations as explicit `allow:` rules, each tagged with a
   comment such as `# BASELINE 2026-07: orders reaches into billing directly`. Do not use
   a matching `deny:` to record intent for the same literal edge: G2 rejects an edge that
   is both allowed and denied. Intent lives in the comment and in the ratchet review.
4. CI runs `machinery check design --impl . --gate g2,g4`. From this moment, a new
   undeclared cross-boundary edge fails the build.

Know the baseline's one hole: an `allow` rule amnesties the whole edge, including files
written next year. A new file that repeats a baselined violation passes silently. The
ratchet therefore needs a human cadence: put a recurring calendar slot (monthly works) on
shrinking the BASELINE list and the `ignore:` globs, and treat a quarter with zero
shrinkage as a warning sign that the gate has become wallpaper.

### Stage 2: domain archaeology and the first machines (add gate g3)

- Model the domain AS IT IS, not as you wish it were. The entities, statuses, and actions
  come out of the code and the database, not out of a product conversation. Where the code
  is incoherent (two meanings for one word), record the incoherence as an open question in
  the model rather than silently picking a winner.
- Keep the domain model trimmed to the slice you are gating. Gx has no per-entity waiver:
  every entity with a lifecycle enum must have a machine before Gx passes. Machine-at-a-time
  adoption therefore means the domain model grows entity by entity alongside the machines.
  Add `gx` to the CI gate list only when every lifecycle enum in the model has its machine.
- After each machine: run `machinery oracle design/machines` and commit the generated
  oracle in the same PR. G3 fails on stale oracles, which is the drift protection working.
- Keep matrix markdown well-formed. A table that looks like a transition table but fails
  to parse is a hard G3 error, so rot in a hand-maintained matrix is caught loudly rather
  than silently skipping the row-by-row reconciliation. The `checked:` line always states
  how many matrix rows were actually reconciled.

### Stage 3: characterization tests (the behavioral loop)

This is the loop that actually maps model-vs-code behavioral drift, and it is manual by
design; the tool's contribution is the oracle and its stable ids.

1. A test writer derives table-driven tests from the oracle rows, keyed on stable ids.
2. Run them against the legacy code. Failures are the drift map.
3. Adjudicate each failing row, case by case, with an explicit verdict:
   - **Code is the truth**: the model is wrong archaeology. Fix the design artifact,
     regenerate the oracle, and the row changes or disappears.
   - **Model is the truth**: the code has a latent defect. File it, quarantine the test
     with a marker naming the stable id and the ticket, and fix the code on its own
     schedule.
4. A test becomes LOCKED (the hard-TDD rule: implementers never edit it) at the moment its
   row is adjudicated, not at the moment it is generated. Record verdicts in the PR that
   introduces the tests. An unadjudicated red test is a question, not a gate.

This is the one place this guide deliberately extends SKILL.md: the hard-TDD handoff
describes greenfield, where every oracle row is normative from birth. On brownfield, rows
start descriptive and become normative through adjudication. Do not skip the recording
step; five developers each "deciding case by case" without a written verdict converges on
nobody knowing which tests are load-bearing.

### Stage 4: full gates and revision mode as the operating loop

Once a slice is fully modeled (machines, matrices, oracles, formal annotations if you want
the proofs), drop the `--gate` narrowing for that design and run the full
`machinery check design --impl .`. New work inside the slice now follows the ordinary
greenfield protocol: design change first, gates green, oracle diff as the affected-test
list, then implementation. BUILD.md's zero-context claim applies to the new work you carve
out of the modeled slice.

### Stage 5 (rare): sharding and recursion

Run `machinery scale design` before reaching for either. Shard at roughly ten stateful
components; recurse (contract packs) only when the domain model itself no longer fits one
conversation. If you do decompose across repos, note that a stale child is only detectable
from the PARENT side (the child pins its own copied pack and is green by construction), so
the parent's `machinery check` must run in CI somewhere, on a schedule if the repos are
separate.

## 4. Team workflow: ownership, PRs, merges

machinery's docs assume one conductor and one user. With five people, add these rules.

**Ownership.** Each design (or bounded context) has a named steward. The steward owns the
domain vocabulary, arbitrates contested definitions (disputes are settled in the design
PR and recorded in a `design/DECISIONS.md`, one dated line each), runs the ratchet review,
and is the default attestor for that design's LLM-attested gate halves.

**PR discipline.**

- A design change and its regenerated artifacts (`*.oracle.md`, `formal/*.tla`,
  `formal/*.cfg`, `packs/`) land atomically in ONE PR. A PR that edits a machine without
  its regenerated oracle is malformed; CI's G3 will catch it, but reviewers should reject
  it on sight.
- Design PR before implementation PR when the change is behavioral. The oracle diff in the
  design PR is the implementation PR's test plan.
- CI runs `machinery check` on every PR and again on main after merge. The post-merge run
  matters: two individually green design PRs can merge into a stale combination.

**Merge protocol for generated files.** Never hand-resolve a conflict in a generated file.
On any conflict in `*.oracle.md`, `formal/*.tla|cfg`, or `packs/`: take either side,
regenerate (`machinery oracle design/machines`, `machinery verify-formal design`, or
`machinery pack generate`), commit, and let `machinery check` arbitrate the result. The
sources (machine JSON, matrix, contract, domain model) merge like ordinary text and their
conflicts are resolved by humans as usual; the generated layer is always reconstructed,
never merged.

**STATE.md.** The session ledger is single-writer: only the steward updates it, and only
on the branch where an interrogation session is actually running. Cross-branch status
lives in PRs, not in STATE.md, or it becomes a permanent merge conflict.

## 5. CI recipe for a consuming repo

The machinery repo's own CI gates machinery; your repo needs its own, and none ships in
the box. Skeleton (GitHub Actions shown; the shape ports anywhere):

```yaml
jobs:
  design-gates:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@<pinned-sha>
      - name: Install machinery (pinned)
        run: |
          curl -fsSLO https://github.com/RamXX/machinery/releases/download/v0.1.0/machinery-linux-amd64
          curl -fsSLO https://github.com/RamXX/machinery/releases/download/v0.1.0/checksums-sha256.txt
          grep machinery-linux-amd64 checksums-sha256.txt | sha256sum -c -
          install -m 0755 machinery-linux-amd64 /usr/local/bin/machinery
      - name: Install modelith (pinned)
        run: go install github.com/stacklok/modelith/cmd/modelith@v0.4.0
      - name: Domain model lint
        run: modelith lint design/domain.modelith.yaml
      - name: Deterministic gates (stage-scoped)
        run: machinery check design --impl . --gate g2,g4   # widen as you climb the ladder

  formal:            # optional but recommended; needs Java
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@<pinned-sha>
      - uses: actions/setup-java@<pinned-sha>
        with: { distribution: temurin, java-version: "21" }
      - run: machinery verify-formal design
```

Rules of thumb: the `--gate` list is the single source of truth for your adoption stage;
widen it in a PR so the whole team sees the ratchet click. Run `design-gates` on every PR
and on main. Run `formal` nightly if it is slow for you, on PR if it is fast. If you have
a decomposed design, add a job that runs the parent design's check.

## 6. Attestation: the non-deterministic half of every gate

SKILL.md marks, per gate, what the tool verifies and what "you" attest (guard semantics
match the invariants they name, every action has an owning component, the NFR record is
real, BUILD.md is buildable with zero context). On a team, "you" must resolve to a name:

- The design PR description carries an attestation checklist per gate, copied from
  SKILL.md's LLM-attested lists, each item checked by a named reviewer (steward by
  default, any second dev otherwise).
- An attestation is invalidated by any change to the artifact it covered; re-attest in the
  PR that changes it. Cheap rule: if the PR touches a machine, the guard-semantics line
  gets re-checked for that machine, and nothing else.

This is bookkeeping, but it is the difference between "the deterministic half is green and
the other half is somebody's memory" and a design history you can audit.

## 7. Revision mode: renames and state migration on day one

- The oracle diff is the affected-test list, and it works exactly as documented for
  ordinary edits (verified: one edit produced exactly one new stable id, all others
  preserved).
- **Renames are the exception.** Stable ids hash the machine name and source state, so
  renaming an entity or state churns every affected id even though nothing behavioral
  changed. Handle a rename as a dedicated mapping PR: rename, regenerate, and include an
  old-id to new-id mapping table in the PR (the diff pairs up by everything except the
  renamed token). Do not process a rename as "all tests deleted, all tests new", and do
  not hand-edit oracles to avoid the churn; G3 flags hand edits as DRIFT.
- **State migration runs in reverse on brownfield.** The template's protocol covers
  renaming states in a deployed machinery design. Your day-one problem is the other
  direction: legacy persisted values onto freshly modeled enums. So the FIRST version of
  any machine whose states are persisted already carries a state-migration note: a mapping
  table from every value observed in production storage to a modeled state, plus a rule
  for unmapped strays (fail loudly beats silent coercion). Gx's
  state-migration-when-persisted conformance check will hold you to having the section;
  the brownfield content is on you.

## 8. Sharp edges (verified, work around until fixed)

- No built-in baseline or ratchet mechanism: the BASELINE comment convention in section 3
  is manual, and allow rules amnesty future files on the same edge.
- Gx requires a machine for every lifecycle enum in the domain model, with no per-entity
  waiver; trim the model to the gated slice instead.
- G4 reports violations per edge, naming one witness file plus a count of additional
  offenders; budget remediation by the count, not by the number of error lines.
- A stale child in a decomposed design is only visible to the parent's check (the parent
  does warn about subsystems with no `child_design` link, whose pins it cannot verify).
  The same holds for tampering: a child-side edit that rewrites the pack copy and
  recomputes its hash is self-consistent and passes the child's own gate; the parent's
  check is the authority, so it must run in CI wherever the parent design lives.
- `deny:` rules cannot reference boundaries that do not exist yet; planned-but-unbuilt
  boundaries live in comments until they have DSL elements.

## 9. What "sustainable" looks like (the exit criteria)

You are done adopting when, for each design: the BASELINE allow list is empty; `ignore:`
covers test scaffolding only; every lifecycle enum has a machine and Gx is in the CI gate
list; the characterization suite is fully adjudicated and locked; CI runs the full check
on every PR plus formal verification at least nightly; and a new hire cannot merge a
boundary violation, a stale oracle, or an unadjudicated behavioral change even if nobody
reviews carefully. At that point the repo is operating the same loop a greenfield
machinery project runs from birth, which was the goal: not a documented mess, but a system
whose design and code are held together by gates instead of memory.
