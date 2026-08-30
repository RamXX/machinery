---
id: MAC-ug4h
title: "Class C: attestation-evidence gate (generalize Ga pattern)"
status: closed
priority: 1
type: feature
created_at: 2026-08-30T08:34:37Z
created_by: ramirosalas
updated_at: 2026-08-30T09:15:36Z
content_hash: "sha256:bcda2d74cd78c0358f025b6bd8e02c8e906af23bc841409b55128d358a0d45a1"
was_blocked_by: [MAC-v16q]
assignee: ramirosalas
follows: [MAC-v16q]
labels: [accepted]
closed_at: 2026-08-30T09:15:36Z
close_reason: "Accepted via pvg story accept"
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
DESIGN DECISIONS (delegated by the story; the repo has no DECISIONS.md for its own development, so they live here and in the code comments they annotate)

1. Evidence-file granularity: ONE FILE PER DESIGN, design/attestations.yaml, rows keyed by claim id.
   Rationale: the attested halves are halves of the design's own gates and are keyed by nothing else.
   Ga keys per milestone number and Gj per machine name because those are natural partitions of their
   evidence; there is no analogous key here, so a per-gate split would buy five files to keep in sync
   and five activation stats instead of one. It also matches the single-top-level-file taste of
   migration.yaml, surfaces.yaml, and decomposition.yaml. Recorded at internal/gates/attest.go, the
   AttestationsFileName doc comment.

2. Claim-id vocabulary: CLOSED, enumerated in code as attestVocabulary (internal/gates/attest.go),
   exported for the CLI and docs via AttestationClaimIDs() so it is never transcribed. 15 ids:
     g2.action-ownership, g2.interface-contract-rightness, g2.placement-rightness,
     g2.adoption-closure-discovery, g2.event-contract-completeness, g2.nfr-content,
     g3.guard-semantics, g3.invariant-enforcement, g3.residual-transitions, g3.event-redelivery,
     gt.conformance-test-shape, g4.zero-context, g4.standin-coverage, g4.pack-event-discipline,
     ga.review-quality
   Sources: the four SKILL.md LLM-attested blocks (G2, G3, Gt/Gate 4, Ga) plus the isolated-child and
   pack-event items, plus the attestation list in agents/machinery-fsm-author.md. Unknown ids are
   ERRORs: an open vocabulary would let a design invent a claim, attest it, and pass a gate that never
   asked for it, which records diligence instead of holding it.

3. Gate letter: Gv-attest. Free in the used set (gm,gs,gu,gp,gi,gn,gc,g2,g3,gd,gl,gx,gk,gb,ge,ga,gj,
   g4,gt,g5); a for accept and t for tests were already taken, so v (vouching) carries the mnemonic.
   Placed after gj and before g4 in the canonical order: it is evidence like Ga/Gj, and it belongs
   before the impl-facing gates.

4. Staleness severity: ERROR, not DRIFT, with the literal word STALE in the message. DRIFT in
   machinery means a GENERATED artifact fell behind its source and is fixed by regenerating; nothing
   regenerates a judgment. Calling it DRIFT would misdescribe the remedy, which is a person reading
   the changed artifact and deciding again.

5. Coverage posture: WARN for an owed-but-unattested claim, ERROR for a record that is WRONG (unknown
   claim, missing attestor, dangling referent, stale hash, duplicate claim), ERROR for an evidence
   file with zero rows. Rationale: the file is opt-in and adopted mid-design. If its first commit
   failed the gate for every claim not yet re-judged, adopting the record would cost more than not
   adopting it, which guarantees the attested halves stay in conversation forever. The absence rule
   still bites at the file level (an empty check is a failure, not a pass), and a misleading record
   always blocks, because it is worse than a missing one. Recorded at checkAttestationCoverage.

6. Owed-when predicates: g2.* once ARCHITECTURE.md exists; g3.* once machines/*.machine.json exist;
   gt.conformance-test-shape and g4.zero-context once BUILD.md exists; g4.standin-coverage once the
   build document declares a "Neighbor stand-ins" section (the posture itself, so a full-environment
   child owes nothing); g4.pack-event-discipline once pack/ exists; ga.review-quality once
   acceptance/ exists.

7. Hash helper: `machinery attest <path> [<path> ...]` prints "sha256:<hex>  <path>" per file;
   `machinery attest --claims` prints the vocabulary from the binary. Positional paths rather than a
   --hash flag: one form, least surprise, smallest thing that works. gates.ContentHash is the single
   definition of the digest, used by both the gate and the command, so the value the attestor pastes
   is the value the gate compares.

8. Ga tie-in: acceptance/M<n>.yaml keeps its free-prose attestations: list unchanged (no existing
   acceptance file breaks). Documented preference: where an entry restates a Class C claim, write the
   claim id and carry the detail in attestations.yaml. Noted in docs/acceptance-gate.md and SKILL.md.

9. adapters/opencode: checked. adapters/opencode/commands/check.md carries no gate list and no
   paraphrase of the attested blocks (it delegates to `machinery check` and .machinery.json), and
   adapters/opencode/plugins/machinery.js has no gate vocabulary. Nothing to mirror; TestOpenCodeAdapterContracts
   stays green.

PROOF (commit 08f7f85, branch story/MAC-ug4h-class-c-attest, worktree off main 64bbd17)

Commands run and results:

1) go test ./... -count=1
   16/16 packages ok, 0 FAIL. Verbose pass count: 1286 --- PASS lines, 0 failures, 0 skips.

2) scripts/preflight.sh  (the repo's own CI mirror; 11 stages)
   [1] git diff --check                   clean
   [2] gofmt                              clean
   [3] go vet ./...                       clean
   [4] golangci-lint (pinned v2.13.2)     0 issues
   [5..7] schema/manifest checks + build .bin/machinery   ok
   [8] go test -race ./...                16/16 ok
   [9] golden corpus + gate-experiment suite   ok (cmd/machinery, internal/experiments)
   [10] machinery check on all 8 example design suites    0 blocking findings
   [11] go-crm impl tests (separate module)                6/6 ok
   Final line: "preflight OK: local gates match ci.yml, safe to push."

3) New tests added (all passing):
   internal/gates/attest_test.go   TestAttestationClean, TestAttestationInactiveWithoutFile,
     TestAttestationMutations (16 subtests), TestAttestationGoesStaleWhenTheArtifactMoves,
     TestAttestationRejectsDirectoryReferent, TestAttestationCoverageWarnsRatherThanBlocks,
     TestAttestationOwesOnlyWhatTheDesignReached, TestAttestationVocabularyIsClosedAndOrdered,
     TestContentHashShapeAndStability, TestAttestationSuiteWiring,
     TestAttestationSurvivesDecomposedParentNarrowing
   cmd/machinery/attest_test.go    TestAttestPrintsTheHashTheGateDemands, TestAttestHashesEveryPath,
     TestAttestClaimsPrintsTheVocabulary, TestAttestFailsOnAnAbsentPath,
     TestAttestWithoutArgumentsFails, TestCheckGateGvRunsAndCatchesStaleness
   internal/hook/hook_test.go      TestSelectGatesActivatesGvOnAttestationEvidence
   Fixture-based, real files under t.TempDir(), no mocks.

SAMPLE GATE RUN (real binary built from 08f7f85, real design dir on disk)

Helper:
  $ machinery attest $D/ARCHITECTURE.md
  sha256:82d657598c1bf13f3e3698b94b244a67580e703bb03f8d9f2f480b1b9f602ffa  $D/ARCHITECTURE.md

Pass (six g2 rows bound to that hash):
  $ machinery check $D --gate gv ; echo exit=$?
  == Gv-attest  attestation evidence ==
    checked: 6 covered artifacts current, 6 attested claims, 6 claims owed
    ok

  0 blocking (ERROR/DRIFT) finding(s)
  exit=0

Induced STALE (one sentence appended to ARCHITECTURE.md, nothing else touched):
  $ printf '\nOne more sentence.\n' >> $D/ARCHITECTURE.md
  $ machinery check $D --gate gv ; echo exit=$?
  == Gv-attest  attestation evidence ==
    ERROR  attestations.yaml: g2.action-ownership is STALE: ARCHITECTURE.md changed since it was
           attested (recorded sha256:82d6575..., current sha256:4d7528d...); re-read the artifact,
           judge it again, and update the row with 'machinery attest <design>/ARCHITECTURE.md'
    ERROR  attestations.yaml: g2.interface-contract-rightness is STALE: ... (same shape)
    ERROR  attestations.yaml: g2.placement-rightness is STALE: ...
    ERROR  attestations.yaml: g2.adoption-closure-discovery is STALE: ...
    ERROR  attestations.yaml: g2.event-contract-completeness is STALE: ...
    ERROR  attestations.yaml: g2.nfr-content is STALE: ...
    checked: 6 attested claims, 6 claims owed

  6 blocking (ERROR/DRIFT) finding(s)
  exit=1

ACCEPTANCE CRITERIA

AC1 gate implemented + wired: internal/gates/attest.go (CheckAttestations, AttestationActive);
    suite.go knownGateSet, default list gm..gj,gv,g4,gt,g5, RunSelected, and the machine-less-parent
    narrowing; internal/hook/hook.go selectGates. Activates only on design/attestations.yaml.
    Pinned by TestAttestationSuiteWiring, TestAttestationSurvivesDecomposedParentNarrowing,
    TestSelectGatesActivatesGvOnAttestationEvidence, TestCheckGateGvRunsAndCatchesStaleness.
AC2 gate tests: valid file passes (TestAttestationClean); unknown claim id, missing attestor,
    missing covered path, hash mismatch STALE, duplicate claim each produce the intended finding
    (TestAttestationMutations subtests "unknown claim id", "missing attestor"/"empty attestor",
    "missing covered path", "hash mismatch is STALE", "duplicate claim"), plus 11 further shape
    mutations. Fixture-based, real files, no mocks.
AC3 SKILL.md (design tree, activation paragraph, gate roll-call, the four attested blocks, and a
    new "Attestation evidence (Gv-attest)" section), agents/machinery-fsm-author.md,
    agents/machinery-build-writer.md, docs/attestation-evidence.md (new),
    docs/acceptance-gate.md (Ga tie-in), docs/brownfield-team-guide.md section 6 (PR-checklist
    superseded, pointer added), docs/claude-plugin.md (stop-hook selection),
    skills/machinery/tools/README.md, commands/check.md, README.md (gate table + phase map + docs
    index). SKILL.md frontmatter version: untouched (TestPluginManifests green). Adapter checked:
    adapters/opencode carries no gate list or attested-block paraphrase, so nothing to mirror.
AC4 make test and make lint green (see preflight stages 3, 4, 8 above).
AC5 no em dashes or emojis: scanned every changed and added file, zero hits.

WIRING EVIDENCE: Gv is mounted in three places, each with a test that exercises it THROUGH the
wiring, not in isolation. (1) CLI gate list: TestCheckGateGvRunsAndCatchesStaleness drives
newCheckCmd() with --gate gv end to end, green then blocking. (2) Suite default selection and the
machine-less-parent narrowing: TestAttestationSuiteWiring runs Select+RunSelected and asserts the
gate is absent without the artifact and present with it; TestAttestationSurvivesDecomposedParentNarrowing
asserts the narrowing note lists gv. (3) Stop-time hook auto-selection:
TestSelectGatesActivatesGvOnAttestationEvidence calls selectGates the same way the hook does.

LEARNINGS
- The generalization was cheaper than expected because Ga and Gj had already settled the shape:
  activation-on-artifact, ir.LoadYAML plus a closed key set, per-row findings that name the file and
  the row index. The only genuinely new decision was the severity of staleness, and reading Gk's
  input_hash (Drift) next to Ga's commit binding (Errs) is what settled it: DRIFT is for artifacts a
  generator can refresh, and a judgment has no generator.
- Coverage posture was the one place where machinery's usual 'absence is an ERROR' rule had to be
  argued against rather than applied. The deciding question was adoption cost: a gate whose first
  commit turns the tree red is a gate nobody commits. Warn on missing coverage, error on a wrong
  record, error on an empty file, keeps the principle where it bites and removes it where it would
  have prevented the record from ever existing.
- Exporting the vocabulary (AttestationClaimIDs) instead of documenting it twice paid for itself
  immediately: the CLI --claims flag, the gate's own 'the ids are ...' error text, and the CLI test
  all read the same slice, so the docs table is the only transcription and it is the one a reader
  can check against 'machinery attest --claims'.
- Gotcha for future gate stories: golangci-lint's staticcheck QF1002 rejects a bare 'switch { case
  x == "": ... default: ... }' over a single variable and wants a tagged switch or an if/else. Cheap
  to fix, but it only surfaces at preflight stage 4, well after go test is green, so run
  scripts/preflight.sh before believing a gate is done.
- The gate-list string 'gm,gs,...,g5' is duplicated across seven files (CLI Use line, CLI flag help,
  suite.go default, three docs, one command frontmatter). A single perl -pi over the exact old
  substring caught them all; grepping for the whole list rather than for individual letters is the
  reliable way to find every copy.

## History
- 2026-08-30T08:34:40Z dep_added: blocked_by MAC-v16q
- 2026-08-30T08:48:02Z dep_removed: was_blocked_by MAC-v16q
- 2026-08-30T08:52:03Z status: open -> in_progress
- 2026-08-30T08:52:03Z auto-follows: linked to predecessor MAC-v16q
- 2026-08-30T08:52:03Z claimed by ramirosalas
- 2026-08-30T09:10:12Z status: in_progress -> in_progress
- 2026-08-30T09:15:36Z status: in_progress -> closed

## Links
- Was blocked by: [[MAC-v16q]]
- Follows: [[MAC-v16q]]

## Comments

### 2026-08-30T09:15:31Z ramirosalas
ACCEPTED: Reviewed diff 64bbd17..08f7f85 (single commit) in worktree wt-ug4h. Tier 1: pvg verify clean (the 6 hook.go 'return ""' stub hits are pre-existing lines outside the diff, not part of this delivery); pvg gates PASS (7 file_loc WARN, 0 BLOCK). Tier 2: go test ./... 16/16 packages ok; scripts/preflight.sh full 11-stage run green, matching proof exactly. Re-ran the sample gate independently against a fresh copy of examples/go-crm/design: machinery attest printed the hash, machinery check --gate gv passed clean (6 covered artifacts current, 6 attested claims, exit 0), then an appended sentence to ARCHITECTURE.md reproduced the STALE ERROR on all 6 covering rows with exit 1, matching the developer's proof transcript. Design-call rulings: Gv-attest letter, one-file-per-design (attestations.yaml), the 15-id closed vocabulary, STALE-as-ERROR-not-DRIFT, and WARN-for-owed-missing/ERROR-for-wrong-record/ERROR-for-empty-file coverage posture are all well justified and consistently implemented (checkAttestationCoverage, checkAttestationFreshness). Vocabulary spot-check: all ids referenced in SKILL.md's four attested blocks (g2.*, g3.*, gt.conformance-test-shape, g4.zero-context, g4.standin-coverage, g4.pack-event-discipline, ga.review-quality) and in agents/machinery-fsm-author.md and agents/machinery-build-writer.md match attestVocabulary in internal/gates/attest.go exactly; nothing dropped or invented. Wiring confirmed in three places with tests exercising each: CLI --gate flag (cmd/machinery/check.go + TestCheckGateGvRunsAndCatchesStaleness), suite.go default list/activation (TestAttestationSuiteWiring, TestAttestationSurvivesDecomposedParentNarrowing), and hook.go auto-selection (TestSelectGatesActivatesGvOnAttestationEvidence). Confirmed via a throwaway local rebase onto current main (b235fc7, the wave-sentinel story) that the hook.go hunk (an 8-line addition inside selectGates only) rebases cleanly with zero conflicts against the wave-sentinel deny-code changes in pre()/editedOps(), and the rebased tree builds and passes internal/hook, internal/gates, cmd/machinery tests. Grepped the whole repo for the old 19-gate list string: zero stale copies remain, all 7 known copies (CLI Use/flag help, suite.go, 3 docs, 1 command frontmatter) updated to include gv. No em dashes or emojis in any added/changed line (regex-scanned the full diff). SKILL.md frontmatter version: line untouched. adapters/opencode confirmed to carry no gate list or attested-block paraphrase (no diff needed, none produced). Test suite is fixture-based (t.TempDir, real files), no mock libraries found. Docs (attestation-evidence.md, acceptance-gate.md Ga tie-in, brownfield-team-guide.md section 6 pointer, claude-plugin.md, external-checkers.md, tools/README.md, commands/check.md, README.md) are complete, consistent, and cross-referenced correctly. No out-of-scope changes found (dialog-register and decision-lifecycle-pattern.md untouched, as required). Decision: ACCEPT.
