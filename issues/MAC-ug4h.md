---
id: MAC-ug4h
title: "Class C: attestation-evidence gate (generalize Ga pattern)"
status: open
priority: 1
type: feature
created_at: 2026-08-30T08:34:37Z
created_by: ramirosalas
updated_at: 2026-08-30T08:34:37Z
content_hash: "sha256:b7ad75df9b4bc8a2e01b2dea141a6cb17f1b18a5ae9c4ab669271c5dfad64467"
---

## Description
# Class C: attestation-evidence gate (generalize the Ga pattern to every attested gate half)

## Context (all you need; verified at HEAD f1dc685)

Machinery's gates deliberately split each domain into a deterministic half (the tool checks) and an attested half (the LLM judges). The 2026-08-29/30 audits found the attested halves leave no committed record: about 15 attestations live only in conversation. The families:

1. G2's LLM-attested block (skills/machinery/SKILL.md:438-450): interface-contract rightness, placement rightness, closure discovery, event-table completeness, NFR content truth.
2. G3 / fsm-author attestations (SKILL.md:513-518; agents/machinery-fsm-author.md:158-163): guard semantics, residual transitions, event-contract rows, redelivery stories. "Include the verdicts in your summary" and the summary is ephemeral.
3. Gate 4 zero-context claim + isolated-child attestations (SKILL.md:594-595, 966-971, 997-998).
4. Gt conformance-test-shape attestation (SKILL.md:590-593).
5. Ga acceptance `attestations:` strings, checked only for non-emptiness (internal/gates/accept.go:312-313).

Decision (user, 2026-08-30): adopt the generalization. The design sketch (from NEXT.md, "Class C, attestation-evidence generalization"): a small evidence schema, named attestor + content hash of the covered artifact per attested half, staleness-checked the way Ga checks --commit, so "judged by whom, and is the judgment still current?" becomes deterministic even though the judgment never is. Gate letter free; activation on evidence-file presence, like Ga/Gj. Existing templates in-repo: Gk binds external-checker evidence by input_hash (internal/gates/checkers.go, docs/external-checkers.md:264-289); Ga parses committed acceptance evidence (internal/gates/accept.go); Gj parses adjudication verdicts (internal/gates/adjudication.go). Also see docs/brownfield-team-guide.md:307-325 (the PR-checklist sketch this supersedes) and docs/decision-lifecycle-pattern.md (draft, related but distinct; do not implement it).

## Design decisions delegated to you (record each in the story notes and DECISIONS if the repo convention asks)

- One evidence file per design vs per gate: pick one, justify briefly. Leaning from the audit: one file per design (design/attestations.yaml) with rows keyed by a typed claim id, since activation-on-presence then stays a single check; but follow whatever Ga/Gj precedent fits the codebase best.
- Attested-half vocabulary: enumerate the claim ids in code (a closed set per gate) so the schema can hold coverage closed; unknown claim ids are errors. Sources for the vocabulary: the four SKILL.md marker blocks (:438, :513, :591, :640) and the fsm-author list.
- Gate letter: any free letter following existing naming taste (e.g. Gv-attest or Gh-evidence); check the used set gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,g4,gt,g5 and the CLI/hook wiring.

## Required behavior

1. Schema: per attestation row: claim id (from the closed vocabulary), attestor (non-empty name), covered artifact path(s), content hash of the covered artifact at attestation time, date. Follow Ga/Gk YAML conventions.
2. Deterministic checks: schema validity; claim ids resolve to the vocabulary; covered paths exist; hash matches the current artifact bytes (mismatch = STALE finding, an error, in the spirit of Ga's commit binding and Gk's input_hash); duplicate claim ids flagged.
3. Coverage posture: absence of the evidence file = gate inactive (activation-on-artifact, like Ga/Gj; hook auto-selection wiring at internal/hook/hook.go:486-552 must gain the new gate). When the file exists, missing claims for gates whose artifacts exist are findings; pick warn vs error deliberately and record why.
4. Content of the judgment is NOT checked. Existence, attribution, referents, freshness only.
5. SKILL.md integration: at each attested block (the four markers) instruct the conductor/subagent to WRITE the attestation rows instead of (or in addition to) stating verdicts in a summary. Keep the register consistent with surrounding prose. Do not touch SKILL.md frontmatter `version:` (pinned by TestPluginManifests, internal/hook/hook_test.go:793-799).
6. Agents: machinery-fsm-author.md and machinery-build-writer.md updated so their attestations land in the evidence file.
7. Ga tie-in: Ga's free-prose `attestations:` list stays valid, but where a string corresponds to a Class C claim id, prefer the reference; do not break existing acceptance files.
8. Docs: a short doc (docs/attestation-evidence.md or an extension of docs/acceptance-gate.md, your call) describing the schema, the vocabulary, staleness, and re-attestation on artifact change. Note it supersedes brownfield-team-guide.md section 6's PR-checklist idea and update that section with a pointer.
9. adapters/opencode: mirror any command/prompt text changes in lockstep if it paraphrases the touched blocks.
10. Tooling ergonomics: a helper to compute the content hash the gate expects (e.g. `machinery attest --hash <path>` or documented shell equivalent) so attestors are not hand-rolling hashes; smallest thing that works.

## Non-goals

- No dialog-register work (separate open item).
- No docs/decision-lifecycle-pattern.md implementation.
- No enforcement that attestations are TRUE; that is the point of the design.

## Acceptance criteria

1. New gate implemented, wired into CLI gate list, suite, and hook auto-selection; runs only when the evidence artifact exists.
2. Gate tests: valid file passes; unknown claim id, missing attestor, missing covered path, hash mismatch (STALE), duplicate claim each produce the intended finding. Fixture-based like existing gate tests (real files, no mocks).
3. SKILL.md, both agents, and docs updated as above; adapter checked.
4. `make test` and `make lint` green at repo root.
5. No em dashes or emojis in any added text or code comments.

## Proof required on delivery

Paste go test + lint summaries and a sample gate run (pass + one induced STALE failure) into the story notes.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
