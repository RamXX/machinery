---
id: MAC-v16q
title: "Remove prompt prose that restates deterministic gate checks"
status: in_progress
priority: 1
type: chore
created_at: 2026-08-30T08:34:37Z
created_by: ramirosalas
updated_at: 2026-08-30T08:42:03Z
content_hash: "sha256:cd425ceed0dcdbf55aa962523063aee0f067bcc7f1f8a0800b62696c5e4f1e56"
blocks: [MAC-ug4h]
assignee: ramirosalas
---

## Description
# Remove prompt prose that restates deterministic gate checks (fidelity-preserving)

## Context (all you need; verified at HEAD f1dc685)

The 2026-08-30 audit found six spots where agent/command prose restates checks a gate already performs. Duplicated prose can only drift from the code, and every deleted line shrinks the LLM instruction surface. Decision (user, 2026-08-30): delete them, on the condition that NO fidelity is lost. That condition drives item 6 below.

## The six edits

1. agents/machinery-build-writer.md:145 "state-migration section and toolchain subsection present" is checked by Gx (internal/gates/gates.go:1565-1574). Replace the restatement with a pointer: the gate checks this; run `machinery check`.
2. agents/machinery-build-writer.md:142 "traceability matrix covers every invariant, ids as whole tokens" is verbatim Gx. Same treatment.
3. agents/machinery-build-writer.md:151-155 restates the build-plan format that Gb parses (internal/gates/buildplan.go). Same treatment: keep any AUTHORING guidance that tells the writer how to produce the plan, remove only the re-verification instructions that duplicate Gb findings.
4. agents/machinery-fsm-author.md:146-156, the "Deterministic (the tools check these)" list, re-enumerates G3's lint items and can only drift from internal/lint. Replace the enumeration with a one-liner of equal force: run the tools; every lint ERROR is yours to fix before delivering.
5. commands/check.md:19-20 "a gate that checked nothing failed" restates engine behavior (gates fail on absence; see skills/machinery/SKILL.md:151-152). Remove.
6. skills/machinery/SKILL.md:194-195, the perl em-dash strip instruction over the modelith render. This one exists BECAUSE Gl only warns on em dashes in generated modelith renders (internal/gates/ledger.go:193-196). To delete it without losing fidelity, first promote that specific check: in Gl, em dashes in `*.modelith.md` (a generated file with a known post-processing obligation) become an ERROR instead of a warn; house-style scanning elsewhere stays at its current tier. Then replace the perl instruction with a short note that Gl errors on em dashes in the render. Add/extend a Gl test fixture for the error tier.

## Fidelity rule (hard)

For each edit, the obligation must remain fully enforced or fully stated after the change: either a gate errors on it, or a single pointer sentence remains. Do not delete guidance that teaches HOW to author something; only delete instructions to re-verify what a gate verifies. If, while editing, you find the cited prose does not actually duplicate the gate (line numbers may have drifted), leave it and note the discrepancy in the story instead.

## Non-goals / cautions

- Do not touch skills/machinery/SKILL.md frontmatter `version:` (pinned by TestPluginManifests, internal/hook/hook_test.go:793-799).
- adapters/opencode paraphrases commands; check whether it mirrors any of the six spots and apply the same edit there in lockstep (known convention).
- No other Gl behavior changes.

## Acceptance criteria

1. All six spots edited per above; grep confirms the removed restatements are gone.
2. Gl errors on em dashes in *.modelith.md with a test proving it; warn tier elsewhere unchanged, existing Gl tests green.
3. `make test` and `make lint` green.
4. Adapter checked; edits mirrored or a note recorded that no mirror exists.
5. No em dashes or emojis introduced in any edited prose.

## Proof required on delivery

Paste go test + lint summaries and the grep(s) showing removal into the story notes.

## Acceptance Criteria


## Design


## Notes
PROOF (commit 64bbd17, branch story/MAC-v16q-prompt-dedup)

Files changed: README.md, agents/machinery-build-writer.md, agents/machinery-fsm-author.md, commands/check.md, skills/machinery/SKILL.md, internal/gates/ledger.go, internal/gates/ledger_test.go. internal/hook/hook.go NOT touched (concurrent story owns it).

TESTS -- make test: all 16 packages ok, 0 failures.
  cmd/machinery ok 7.3s; internal/alloy ok; internal/checker ok; internal/compose ok; internal/experiments ok; internal/formal ok; internal/gates ok 1.5s; internal/hook ok; internal/install ok; internal/ir ok; internal/lint ok; internal/oracle ok; internal/pack ok; internal/refine ok; internal/tla ok; internal/version ok.
  Gl subset: go test ./internal/gates -run TestLedger -v => 23 PASS, 0 FAIL (19 pre-existing + 4 new/extended... 3 new tests, 23 total including subtests).

LINT -- the repo has no 'make lint' target; the CI mirror is 'make preflight' (gofmt gate, go vet, golangci-lint pinned, race suite, golden corpus, bundled-example gate suite, go-crm impl module). Result: 'preflight OK: local gates match ci.yml, safe to push.' 0 blocking (ERROR/DRIFT) findings across every bundled example; Gl-ledger reported ok on all of them (7 and 8 files style-scanned).

RED-BEFORE-GREEN evidence for the Gl promotion: with internal/gates/ledger.go stashed, the two new render tests fail with the finding on g.Warns instead of g.Errs:
  --- FAIL: TestLedgerModelithRenderEmDashErrors: em dash in the modelith render must error: errs=[] warns=[domain.modelith.md:2: em dash (U+2014); house style forbids it ...]
  --- FAIL: TestLedgerNestedModelithRenderEmDashErrors: nested render em dash must error: errs=[] warns=[legacy/domain.modelith.md:1: ...]

GREPS -- removed restatements (searched agents/ commands/ skills/ adapters/):
  'The traceability matrix covers every invariant'                     0 hits
  'The state-migration section and the toolchain-and-versions subsection are present'  0 hits
  'The build plan follows the checkable format Gb-plan holds'          0 hits
  'Deterministic (the tools check these'                               0 hits
  'a gate that checked nothing failed'                                 0 hits
  'perl -CSD -i -pe'                                                   0 hits (moved into the Gl error message in internal/gates/ledger.go)

AC5 -- em dashes in edited files: build-writer 0, fsm-author 0, check.md 0, SKILL.md 0, README.md 0, ledger.go 0. ledger_test.go has 9 lines with em dashes: those are the fixture strings under test (pre-existing pattern). No emojis added.

Non-goal honored: SKILL.md frontmatter version untouched (git diff shows 0 changed 'version:' lines); TestPluginManifests green in the internal/hook package run.

AC-BY-AC
1. Six spots edited; greps above confirm removal. All six genuinely duplicated their gate at HEAD f1dc685 (verified in source: Gx toolchain/state-migration headings at internal/gates/gates.go:1562-1573 and whole-token invariant coverage at :1614-1650; Gb milestone/DoD/skeleton/NFR/oracle-citation at internal/gates/buildplan.go:391-507; internal/lint/lint.go:275-465 and deadline.go for the fsm list; SKILL.md 'Gates fail on absence' for check.md). No discrepancy found, nothing left in place.
2. Gl errors on em dashes in *.modelith.md (isModelithRender suffix test), with TestLedgerModelithRenderEmDashErrors and TestLedgerNestedModelithRenderEmDashErrors proving it. Warn tier elsewhere unchanged, proven by TestLedgerHouseStyleWarnTierUnchanged (emoji in a render, em dash in ARCHITECTURE.md, in notes-modelith.md, in machines/Deal.md, in domain.modelith.yaml -- all warns, 0 errors) plus the untouched pre-existing TestLedgerHouseStyleEmDashAndEmoji / SkipsGenerated / UnicodeSymbolsAreFine.
3. make test green; make preflight (the CI mirror, in place of a nonexistent make lint) green.
4. adapters/opencode checked: it carries commands/{check,design,init,status}.md plus plugins/machinery.js, and paraphrases only. grep for 'checked nothing|traceability matrix|state-migration|perl -CSD|Deterministic (the tools|M<n>' and for 'em dash|modelith|2014|lint ERROR|self-check' returns 0 hits there. No mirror exists; no lockstep edit needed.
5. Confirmed above.

FIDELITY LEDGER (what replaced what)
- build-writer: the three deleted bullets are replaced by one pointer paragraph naming Gx-trace and Gb-plan. The authoring guidance in method steps 5, 7, 9, 10 is untouched and still states the two things Gb does NOT check (milestone numbers global across shards; the milestone-acceptance protocol), so no obligation was dropped.
- fsm-author: the enumeration is replaced by a one-liner of equal force ('do not deliver with a lint ERROR outstanding or an oracle ungenerated'), sitting directly under the non-negotiable tool-run block that already names both commands. The oracle DRIFT rule remains stated at the artifact list.
- check.md: only the engine-behavior clause was cut; 'Read the checked: counts and state what was actually verified' remains, and 'Gates fail on absence' remains in SKILL.md.
- SKILL.md: the perl instruction became a note that Gl errors, and the one-liner now lives in Gl's own error text, so the operator still gets the exact fix at the moment it matters. Gl tier descriptions updated in lockstep at SKILL.md:144 and :1038 and README.md:160 and :629.

SIDE OBSERVATION (no edit made, out of scope by dispatcher instruction): internal/hook/hook.go:508-509 carries the comment 'ledger-format and house-style findings ... never block on their own (warn tier plus rare format errors)'. Gl behavior for gl-always activation is unchanged, and the comment already acknowledges format errors, so it is not wrong; it could be sharpened to mention the render error tier when that file is next free.

LEARNINGS
- Verifying 'does this prose duplicate the gate?' means reading the gate, not the gate's name: Gb checks milestone-number uniqueness per document, not across a sharded design, so the self-check bullet was broader than the gate. Deleting it was still lossless only because the authoring section above it says the same thing; had that section not existed, the bullet would have had to stay under the fidelity rule.
- Promoting a warn to an error is cheap only if the error message absorbs the guidance the prose was carrying. Moving the perl one-liner into Gl's finding is what made the SKILL.md deletion fidelity-preserving rather than a quiet loss.
- Tier changes ripple into docs: Gl is described in four places (SKILL.md twice, README.md twice). A grep for the old tier wording ('warn tier', 'house-style') found all of them; without it the docs would have drifted the same way the prompts had.
- The working tree was switched back to main under me mid-session (another agent shares this checkout; story/MAC-rbje-wave-sentinel also exists here). Re-verify 'git branch --show-current' before every stage and commit when agents share one checkout.

## History
- 2026-08-30T08:34:40Z dep_added: blocks MAC-ug4h
- 2026-08-30T08:35:09Z status: open -> in_progress
- 2026-08-30T08:35:09Z claimed by ramirosalas

## Links
- Blocks: [[MAC-ug4h]]

## Comments
