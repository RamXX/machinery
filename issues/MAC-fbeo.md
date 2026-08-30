---
id: MAC-fbeo
title: "Zero-artifact checks A: G2/G3 cluster (DSL edges, event wiring, every-match, cell quality)"
status: in_progress
priority: 1
type: feature
created_at: 2026-08-30T18:54:10Z
created_by: ramirosalas
updated_at: 2026-08-30T19:28:52Z
content_hash: "sha256:ddf00058c654b11366bfedad078beb4e90f64acc9ac75defedc2736bc03746bb"
assignee: ramirosalas
---

## Description
# Zero-artifact engine checks, part A: the G2/G3 cluster (DSL edges, event wiring, every-match locators, event cell quality)

## Context (from the 2026-08-30 determinism audit; line cites verified at f1dc685, may have drifted slightly after 64bbd17/b235fc7/3b4fc6f landed; find by content)

User decision 2026-08-30: implement the eight zero-artifact checks. This story carries the four that live in the G2/G3 area of internal/gates/. All four parse data already present in every design; none adds authoring burden. The other four (Ga/Gu/G5/Gs) are a sibling story; do not touch internal/gates/accept.go, targetsurface.go, pack.go, or surface.go.

## Check 1: DSL relationship arrows vs dependency_rules (G2)

Today dslElements (internal/gates/gates.go, near :318-345 at audit time) parses only element declarations from workspace.dsl; the DSL's own `a -> b` relationship lines are never read. The C4 diagram can draw an edge the Architecture Contract denies and G2 passes. Required: parse relationship lines (the quoteStrings machinery already handles the line shape), resolve both endpoints against declared elements, and hold each drawn edge to the contract's dependency_rules with the SAME allow/deny/baseline semantics the existing contract checks use (study checkDependencyRules and the baseline handling before writing anything; a baselined violation must behave the same way it does elsewhere). A drawn edge the contract denies is an ERROR. Do NOT require the converse (every allowed dependency drawn); diagrams are legitimately partial. Unresolvable endpoints: ERROR, consistent with how G2 treats unknown names elsewhere.

## Check 2: event wiring reconciliation on non-decomposed designs

The pack gate already verifies boundary events against machines (internal/pack or internal/gates/pack.go, the checkBoundaryEvents area near :455-457 at audit time) but only for decomposed designs. For ordinary designs the same event-contract table and the same machines exist and nobody cross-checks them; the FSM author merely attests it. Required: for designs with machines and an event-contract table, every consumer cell must resolve to a machine that handles the event or declares it via the `_ignores` idiom, and every machine event whose source is external (per the table's source notes / producer column) must have a row. Mirror the pack-side semantics exactly; where the pack check and this check would disagree, match the pack check and say so in the story notes. Decide the host gate (G2 owns the event table; G3 owns machines) and record the rationale. Tier: ERROR, same as the pack side.

## Check 3: every-match table locators (G2 mitigation, Gx placement)

The mitigation-table locator takes only the FIRST header match (a `break` near gates.go:636) and the placement locator likewise (near gates.go:1768), while the event-contract and interface-contract scans deliberately take EVERY match after the PACK-1 lesson (near gates.go:704-711, :795-797). A second mitigation or placement table in the tree is silently ignored, so its rows carry no obligation. Required: convert both locators to every-match semantics, aggregating rows across all matched tables exactly as the event/interface scans do. Watch for duplicate-row semantics once aggregation happens; resolve the same way the event scan does.

## Check 4: event-contract cell quality (G2)

Rows are counted (near gates.go:710) but per-cell completeness is unchecked: payload, delivery, ordering, dedupe cells may be empty, and producer/consumer names are never resolved against declared elements/externals, though mitigation rows already get exactly that resolution (near gates.go:644-653). Required: required cells non-empty (ERROR), producer/consumer names resolve against declared elements and externals using the same resolution the mitigation rows get (ERROR on no match). If the table format legitimately allows a placeholder in some column (check SKILL.md's event-table spec first), honor the documented placeholder rather than erroring on it.

## Hard constraints for all four

- Fixture-based tests in the same pass, no mocks, following the existing gate-test idioms (pass case + each failure class).
- The preflight (scripts/preflight.sh) runs `machinery check` over all 8 bundled example designs plus the golden corpus. New checks MAY fire on the examples. When they do, that is a real defect the check just caught: fix the EXAMPLE DESIGN (and regenerate whatever the fix obligates), never weaken the check to keep an example green. Record every such example fix in the story notes. If the golden corpus needs re-capture, use `make golden-update`, review the diff line by line, and include the reviewed summary in your notes.
- Update the gate descriptions in lockstep wherever they are documented (skills/machinery/SKILL.md gate sections, README.md gate tables, docs/ pages that enumerate G2/Gx checks). SKILL.md frontmatter `version:` must remain untouched.
- No em dashes, no emojis, anywhere (code, comments, prose, commit messages).
- Do not touch internal/hook/hook.go, accept.go, targetsurface.go, pack.go (except reading), surface.go, or cmd/ (the sibling story owns those; for check 2, if the cleanest implementation extracts a shared helper used by the pack side, extraction that only MOVES pack-side code without behavior change is allowed; note it clearly).

## Acceptance criteria

1. All four checks implemented at the specified tiers with the specified semantics; each has fixture tests for pass and every failure class.
2. `go test ./... -count=1` green; `scripts/preflight.sh` fully green including the 8 example suites and golden corpus.
3. Any example-design fixes and golden re-captures recorded and justified in the story notes.
4. Docs updated in lockstep; no em dashes or emojis introduced; SKILL.md version untouched.

## Proof required on delivery

Test and preflight summaries; for each check, one induced-failure sample (the finding text) from a fixture or example; the list of example-design fixes if any.

## Acceptance Criteria


## Design


## Notes
DELIVERED on branch story/MAC-fbeo-g2-cluster at a675ca9 (worktree left in place, not merged, not pushed).

DECISIONS AND DISAGREEMENT RESOLUTIONS (the story asked these be recorded):

Check 2 host gate: Gx-trace, not G2. Rationale: Gx is where every existing check that joins an
ARCHITECTURE.md table to the committed machines already lives (checkPlacement's machine obligation,
checkResidualHandling's "handled by" column, which the skill documents as running "in Gx-trace once
machines exist"). G2 must stay green at Phase 2, when the architecture is authored and no machines
exist yet; a machine-dependent check there would fail a design that is correctly mid-phase. Gx also
narrows itself away on a decomposed parent with no machines, which is exactly where the pack side
owns the question.

Check 2 vs the pack check, three documented differences (pack wins each time, per the story's rule):
1. DIRECTION. The pack reads an explicit direction column because a pack is written from one
   subsystem's point of view. An ordinary design has none and needs none: every participant in its
   table is a component of that design, so it owes BOTH the emission and the reaction on every row.
   This is the pack rule with its point of view set to "all of it".
2. EVENT NAMES. The pack CleanCell's the event column, exact because its format contract forbids
   anything but a bare name. An ordinary table's event cell is prose that names its events
   backticked ("`reserved` / `released` events"), the same idiom mitigation and placement rows use,
   so backticked tokens are read when present with CleanCell as the fallback. On a pack-format cell
   this reduces to exactly the pack rule.
3. MACHINE SCOPE. The pack asks whether ANY machine handles the event, never which one; so does
   this. Requiring the machine NAMED by the consumer cell would be stricter than the pack.
Additionally, a design that carries a pack is SKIPPED here: G5 reconciles the same rows from the
generated events.md where direction is explicit, so one defect never earns two findings.

Check 2, the half NOT implemented and why: "every machine event whose source is external must have
a row." Nothing in a machine marks an event as externally sourced, the pack check has no such
sweep, and the only inferable rule (an event no local action fires) is false by construction: it
would demand an event-contract row for every human-initiated event. Measured on the examples, it
would have demanded rows for markDelivered, deliver, commit, authorize, markInTransit, markLost,
markConsumed, publish, cancel, and confirm on fulfillment alone, and on go-crm (which has no event
table at all) for every one of ~194 transitions' events. The rule stays attested, as the skill's
choreography section states it, and both SKILL.md and docs/attestation-evidence.md now say so
explicitly rather than leaving it implied.

Check 4 vs the pack check: the pack is the stricter sibling (it also requires an event column and
rejects a declared external as a participant). Nothing G2 accepts can make pack generation pass
silently, so the looser G2 rule cannot hide a pack-format defect. Recorded in
references/c4-standalone.md next to the machine-checkable format.

Check 4 placeholders: SKILL.md and the c4 reference document NO placeholder token for the event
table's columns, so only an EMPTY cell is a finding; an explicit "none" or "n/a" is an answer and
passes. Pinned by TestEventCellsExplicitNoneIsAnAnswer.

Check 3 duplicate-row semantics: resolved the way the event scan resolves them. Counts accumulate
across tables (a row counted twice is two rows of obligation) and coverage is a set (a dependency
or entity named in either table is covered once). No dedup of identical findings, matching the
event scan, which counts a repeated row twice as well.

EXAMPLE-DESIGN FIXES (each is a defect a new check caught; no check was weakened):

1. examples/fulfillment/design/ARCHITECTURE.md, event table participants. The producer and consumer
   cells named DISPLAY names ("Order Service", "Inventory Service"), which the documented format
   never allowed (c4-standalone: the cell holds an Architecture Contract boundary `element`).
   Rewritten to element identifiers (`orderSvc`, `inventorySvc`, `paymentSvc`, `shippingSvc`),
   16 cells. The table now conforms to the format that pack generation would enforce if this design
   ever decomposed.
2. examples/fulfillment/design/ARCHITECTURE.md, four `(no machine: <reason>)` waivers. Rows 1, 6, 7,
   and 8 state a reaction the behavior layer does not implement, and rows 6, 7, 8 an emission it
   does not fire: fulfillment is an ORCHESTRATION design whose saga consumes bus replies through
   invoke onDone, not as machine events, and whose wire vocabulary (`reserved`, `captured`,
   `dispatched`) is deliberately distinct from its machine vocabulary (`markReserved`, `markPaid`,
   `markShipped`). The waivers state that in the row, with the reason. This is what FINDINGS.md
   already admits in prose ("that phase of this design is not yet authored"); the gate now makes it
   visible in the counts (4 reactions waived, 3 emissions waived) instead of invisible.
3. examples/checkout-split/orders and .../payments, peer subsystem declared. Each child's embedded
   event rows name the peer subsystem (`payments` / `orders`) while the child's own workspace.dsl
   and contract declared it nowhere: the coupling was entirely undeclared. Both now declare the peer
   as a contract external with a bound DSL element and a mitigation posture row (outage behavior,
   residual, bound). The embedded table itself is untouched, so Ge-embed's byte-identical claim
   still holds.

GOLDEN RE-CAPTURE (make golden-update), reviewed line by line: 9 changed lines across 8 files, all
of them `checked:` count lines. Eight are G2 lines gaining `N drawn relationships`, `N drawn edges
verified`, `N drawn edges outside the contract vocabulary`, and (where a design has an event table)
`N event-contract cells answered` and `N event-contract participants resolved`. One is
fulfillment's Gx line gaining the event-wiring counters. NO finding text changed anywhere, no
warn/note/error appeared or disappeared, and no gate changed verdict. Arithmetic verified per
design: go-crm 14 = 5 unbound + 9 verified; surreal-crm 15 = 6 + 9; portfolio 15 = 2 + 13; pii-flow
2 = 2; checkout parent 5 = 1 + 4; orders 3 = 1 + 2; payments 2 = 2; fulfillment 20 = 20 unbound (its
boundaries are containers while its relationships connect components and infra). Event cells:
fulfillment 48 = 8 rows x 6 columns, 16 participants = 8 x 2; checkout children 18 = 3 x 6, 6 = 3 x
2. Gx: fulfillment 4 reactions waived + 4 traced = 8 rows, 3 emissions waived + 5 traced = 8 rows.

## History
- 2026-08-30T18:54:34Z status: open -> in_progress
- 2026-08-30T18:54:34Z claimed by ramirosalas

## Links


## Comments
