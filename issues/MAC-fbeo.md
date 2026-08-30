---
id: MAC-fbeo
title: "Zero-artifact checks A: G2/G3 cluster (DSL edges, event wiring, every-match, cell quality)"
status: open
priority: 1
type: feature
created_at: 2026-08-30T18:54:10Z
created_by: ramirosalas
updated_at: 2026-08-30T18:54:10Z
content_hash: "sha256:6b81fe65ee5552a0f1a3f15a72ebc0df2cf8e9d133f7dfac4024df3064fdbfbf"
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


## History


## Links


## Comments
