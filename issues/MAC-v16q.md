---
id: MAC-v16q
title: "Remove prompt prose that restates deterministic gate checks"
status: open
priority: 1
type: chore
created_at: 2026-08-30T08:34:37Z
created_by: ramirosalas
updated_at: 2026-08-30T08:34:37Z
content_hash: "sha256:fe3967d878887b99666757e894998ff14bd8a22da86bd1d15334345bbdb80101"
blocks: [MAC-ug4h]
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


## History
- 2026-08-30T08:34:40Z dep_added: blocks MAC-ug4h

## Links
- Blocks: [[MAC-ug4h]]

## Comments
