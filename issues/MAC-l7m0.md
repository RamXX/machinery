---
id: MAC-l7m0
title: "Define standalone executable test-assurance contract"
status: in_progress
priority: 1
type: task
parent: MAC-ui8a
created_at: 2026-09-05T19:35:06Z
created_by: ramirosalas
updated_at: 2026-09-06T10:17:35Z
content_hash: "sha256:6d34f2fd82d57061af5df9982c1bc27752c3a22fbf98e22b0e3170abe8869bd6"
blocks: [MAC-vx24, MAC-ou97, MAC-qlw2]
follows: [MAC-uzxr, MAC-p7jd]
assignee: dev-MAC-l7m0
---

## Description
## USER INTENT
Provide strongest honest standalone Machinery guarantees for LLM-generated software, including test sensitivity to unsafe behavior.

## Context (Embedded)
New mechanically enforced negative hard-TDD is requested, but no shipped runtime protocol currently exists. Existing buildplan.go only checks narrative section names. Architecture author is independently assessing the closed runner adapter and trust boundary; do not invent interfaces before approval. Product must never depend on Paivot.

## Ownership
docs/test-assurance-contract.md. Not alone in codebase; preserve other edits. Newly listed files are explicitly owned outputs, not pre-existing interfaces.

## Boundary Map
PRODUCES:
- docs/test-assurance-contract.md -> approved closed protocol
CONSUMES:
- Existing Machinery source.
  source: internal/gates/buildplan.go CheckBuildPlan(design string) *Gate; gate suite and hook contracts.

### Story Acceptance Criteria
1. Resolve and record user choices on initial native runner languages and trusted-host versus hostile-code boundary. Record approved concrete CLI, versioned closed manifest/evidence schemas, limits, error statuses, activation/compatibility and ownership before implementation dispatch.
2. Specify Machinery-captured or independently replayed real native test outcomes on immutable snapshots, expected assertion witnesses (native nonzero alone is insufficient), named passing control, no empty/skip/xfail/build/import/panic/timeout/malformed/truncated/fabricated-summary success.
3. Bind exact bytes and complete inventories of tests/helpers/fixtures/runner config/dependency locks; prohibit tokens-equal/whitespace equivalence for frozen evidence. Unsafe challenge variants may alter implementation only, and every required negative behavior needs sensitivity proof against a reviewed unsafe variant, not merely a passing health control.
4. Require replay of retained RED/challenge snapshots plus current GREEN before strong final assurance. Static receipt hashes prove binding, not execution/authenticity. Cheap check/hook freshness must explicitly state replay not performed; historical milestone acceptance remains distinct.
5. Make protocol standalone Machinery only; ordinary Git snapshots are allowed, Paivot metadata/labels/commands are not. Define current implementation staleness, runtime residual obligations, schema migration and unsupported-runner behavior failclosed.
6. Independent architecture review approves contract and dispatcher copies exact new field/API names into blocked implementation story before it is executable.

## Testing Requirements
Read source-verified APIs; adversarial scenario table and standalone contract review only. No production edits or heavy preflight. Architecture task has no RED because it resolves contracts rather than executable behavior.
Integration tests: MANDATORY (no mocks) for executable implementation, no skip-if-missing. Architecture review is not execution proof.
No push/sync/GitHub mutation. No installed binary/skills/plugins/agents replacement or dev-link. Isolated candidate only. Full scripts/preflight.sh only final epic gate.

## OUT OF SCOPE
- Mathematical proof of arbitrary test semantic adequacy or malicious host safety beyond explicitly approved trust boundary: expose these as residual limits, never silently claim them.
- Any Paivot runtime dependency: prohibited by user.

## DIFF BUDGET
- 1 document, under 700 lines.

## MANDATORY SKILLS
- architect for contract authoring/review; developer and pm_acceptor for implementation; codebase-memory for source discovery.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from user requirements and independent read-only design analysis.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified
- [ ] AC #6: independently verified

Observable outcome: the maintainer can review a closed standalone contract that displays its trust boundary and blocks unapproved implementation.

## Acceptance Criteria


## Design


## Notes
BLOCKED — USER CHOICE / ARCHITECTURE REVIEW. Do not dispatch a generic developer. Pending user decisions: supported initial native runner languages and trusted-host versus adversarial-code execution boundary. Independent architect owns exact contract. Only after answers plus reviewed contract may Sr PM repair implementation interfaces and release this blocker.
ANCHOR ROUND-1 RUNTIME CLASSIFICATION: Architecture is read-only contract work, not an executable runtime suite. Its accepted contract must explicitly identify any native adapter runtimes consumed by implementation; MAC-vx24 must register/provision those using the required closed integration lane. No execution assurance may be claimed from schema or architectural review alone.

## History
- 2026-09-05T19:35:06Z dep_added: blocks MAC-vx24
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97
- 2026-09-05T19:36:47Z status: open -> blocked
- 2026-09-06T09:09:57Z dep_added: blocks MAC-qlw2
- 2026-09-06T10:07:13Z status: blocked -> in_progress
- 2026-09-06T10:07:13Z auto-follows: linked to predecessor MAC-uzxr
- 2026-09-06T10:07:13Z claimed by dev-MAC-l7m0
- 2026-09-06T10:12:18Z status: in_progress -> open
- 2026-09-06T10:12:18Z released by ramirosalas
- 2026-09-06T10:12:24Z status: open -> in_progress
- 2026-09-06T10:12:24Z auto-follows: linked to predecessor MAC-p7jd
- 2026-09-06T10:12:24Z claimed by dev-MAC-l7m0

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-vx24]], [[MAC-ou97]], [[MAC-qlw2]]
- Follows: [[MAC-uzxr]], [[MAC-p7jd]]

## Comments

### 2026-09-06T08:47:43Z ramirosalas
AUTHORITATIVE SCOPE REVISION 2026-09-06 — approved architecture landing only
This append-only revision supersedes earlier unresolved-user-choice wording for this story without erasing it. Root confirms user choices and independent CHALLENGE-3 approval (zero blocking findings). Architecture approval is NOT implementation, native feasibility proof or permission to dispatch unreviewed successor stories. Root's backlog-review hold remains until independent Anchor approval.

USER INTENT: land one permanent exact standalone Machinery contract so bounded implementation stories consume reviewed schemas/APIs rather than a temporary proposed report.

PRODUCES:
- docs/test-assurance-contract.md -> exact 589-line v3-final architecture bytes, SHA256 e467a3b6c6f657e7a4cb28e85c688be0df24c56a7e3c76fdcfa224f7ff5d2624
- docs/test-assurance-review.md -> exact 32-line independent CHALLENGE-3 bytes, SHA256 0205b50a8eb2e135230173bbf4a7b7eccdde5934fabab5aa4b2dde95d474ac66
- docs/test-assurance-status.md -> short canonical approval/status index: approved architecture pair above, no implementation/native proof yet; its index explicitly resolves historical PROPOSED wording in the immutable source without rewriting that source
CONSUMES:
- Approved immutable architecture and independent review provided by root.
  source: /tmp/machinery-assurance-architecture.fXW3vg/test-assurance-contract-v3-final.md and CHALLENGE-3.md; verified hashes above.

AC A1: copy the complete architecture/review to owned canonical paths unchanged; exact hashes and line counts match. Status index links both and says review-approved contract, not software acceptance or execution evidence.
AC A2: preserve four first-release closed native adapters and exact catalogs: Go1.27.1; Node26.8.1+TypeScript7.0.2; CPython3.14.7; Elixir/Mix/ExUnit1.20.4+OTP29.0.6/ERTS17.0.6, plus scoped gate Git2.55.0; supported native Linuxamd64/Darwinarm64. No arbitrary shell adapters, Paivot dependency, audit exception or installed replacement.
AC A3: validate retained exact CLI/API/schema/root topology/external-store/explicit-register/expected-head CAS/empty-head/history/failed-RED contracts and entire lifecycle/budget text; no omission or semantic paraphrase.
AC A4: status index carries all three review advisories as unfulfilled implementation obligations: actual two-platform custody/assertion proof; real first-use init/scaffold/capture/register/RED/GREEN/verify/complete without prior PASS receipt; actual process-producing call-graph audit plus cumulative deadlines.
AC A5: no source/test/runtime/CI/installed edits under this documentation-only story. Preserve accepted p7 full-root/Gv custody and legacy frozen tests/96-case hpqp union; new implementation requires separately reviewed stories. No heavy preflight; hash/link/doc checks only.
AC A6: independent PM verifies hashes and canonical references, then delivers/accepts documentation through normal local workflow. No contract imported as a pass result.

DIFF BUDGET: 3 documentation files, ~650 lines. MANDATORY SKILLS: developer for literal landing, pm_acceptor for independent hash/link review. No hard-TDD label is required for immutable documentation landing.

## nd_contract
status: new

### evidence
- Sr PM fully read all589architecture lines and all32review lines; SHA256 and line counts independently matched on2026-09-06.
- Canonical tracker body read completely before this true-EOF scope revision; no old contract overwritten.

### proof
- [x] Approved immutable sources and exact landing scope identified.
- [ ] AC A1-A6: canonical docs landed and independently accepted.


### 2026-09-06T08:49:12Z ramirosalas
PUBLIC CONTRACT PROJECTION CORRECTION 2026-09-06 — supersedes literal-publication AC A1/status-index ownership above
Root requires the open-source document to be standalone and free of private tracker/workspace/user coordination. Private original architecture, challenge files, hashes and audit history remain in nd only. DO NOT publish CHALLENGE-3, private source-note path, tracker handoffs or an index of them. Current owned source output is ONLY docs/test-assurance-contract.md, a deterministic projection of immutable e467a3b6c6f657e7a4cb28e85c688be0df24c56a7e3c76fdcfa224f7ff5d2624. docs/test-assurance-review.md and docs/test-assurance-status.md from prior scope are withdrawn proposed outputs, never created; historical note retained.

Exact projection algorithm:
1. Keep original source lines1–575 inclusive; drop576–589 (private revision/source inspection chronology only).
2. Replace the following entire original lines with exact text:
L1: # Standalone executable test assurance — architecture contract
L3: Status: approved architecture contract, version 1. Approval establishes the required behavior, not implementation completion or native execution evidence.
L4: Scope: executable hard-TDD assurance for Machinery and its consumers, plus native custody for the contributor integration lane. The implementation must prove every applicable obligation below.
L8: Native final verification supports Go, TypeScript, Python, and Elixir in the first release through four CLOSED adapters, not an arbitrary shell-command protocol. A supported language does not mean every framework in that language is supported.
L477: ## 8. Native custody contract and contributor integration
L519: Contributor integration requires the reviewed `internal/processscope`, `internal/processcontrol`, and explicit formal/runtimeclosure call-site wiring, plus separately owned supplemental tests and fragments. Final-Ga scoped execution additionally requires `internal/gates/accept.go`, suite RunOptions, runtimeclosure.Git and assuranceflow wiring. Preserve every original frozen RED/config/test inventory and failed record byte-for-byte. A provisioning-only wrapper is insufficient. Approve exact supplemental test names and ownership before authoring their RED.
L537: The strict Go profile is NOT automatically compatible with existing Go CRM or Machinery contributor tests: direct T.Fatal patterns remain valid under their EXISTING native contributor/test contracts, but do not establish strict assertion-specific RED. Do not rewrite frozen tests under unrelated change authority, hide them through selectors, or claim compatibility from an old native PASS. Prospective adapter/example migration requires a new reviewed test revision and full required-test mapping. Retain original approved/failing RED and accepted history, preserve every existing required regression (running legacy native regressions independently until properly migrated), and provide the NEW strict tests' own same-assertion calibration. Existing BUILD/evidence/golden/helper maintenance does not authorize wholesale Go CRM test rewrites. Machinery's contributor registry remains independently authoritative; it need not masquerade as a consumer TDD manifest or accept imported TDD receipts.
L539: Implement this contract in bounded, explicitly owned units: core schema/inventory/storage and authored-revision registration; native custody; four adapter implementations; replay/state machine; CLI/gate/hook integration including scoped final-Ga Git; example/runtime-residual migration; standalone final capstones. Every interface must have real producer/consumer wiring and frozen positive/negative tests. No helper or adapter may ship uncalled by normal complete verification.
L541: The contributor runtime inventory must include provisioned Python, TypeScript compiler, Elixir/OTP, the separately scoped pinned Git closure for final Ga, and their exact source/runtime identities through separately owned fragments rather than editing a frozen pilot. Preserve existing lane semantics; if a frozen schema cannot express a new adapter requirement, retain it as v1 history and introduce an independently reviewed compatible v2 lane contract plus complete union migration. Never silently edit frozen v1 and call it the same RED.
L545: ## 11. Implementation acceptance obligations
L547: These obligations require executable acceptance evidence; the document alone is not sufficient:
L568: - The first release requires all four languages. Supporting every framework immediately would be an unbounded promise; unsupported frameworks receive actionable migration diagnostics.
L575: Technical feasibility requires actual implementation evidence, especially guardian custody and adapter assertion/event/source correlation. If real tests cannot meet these contracts, stop and revise the architecture through independent review rather than weaken a required outcome to match available code.
3. Apply these exact literal replacements to retained text:
"accepted p7 " -> "existing "
"Accepted p7 " -> "Existing "
"P7 behavior" -> "Existing attestation behavior"
"Never touch the user's existing Dagger container or other unrelated resources." -> "Never touch unrelated containers or other user-owned resources."
"Global final preflight remains the user's explicitly final heavy operation." -> "Global final preflight remains the final integrated heavy operation."
"These were present locally during architecture feasibility inspection; availability is not adapter verification." -> "Availability of a runtime is not adapter verification."
"no provisioning or Git installation is authorized by this architecture task." -> "runtime provisioning is an explicit implementation operation, not evidence supplied by this document."
4. Replace each Unicode em dash with ASCII space-hyphen-space for repository documentation style. Preserve every other byte and one final newline. No normative record/field/argv/signature/limit/lifecycle obligation changes.
Expected public projection: 575lines, 109367UTF8bytes, SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8. Independent PM must reproduce this projection from the private approved source, compare exact bytes AND verify all schemas/interfaces/lifecycle/limits/guarantees remain intact; hash matching alone is not normative review.
Public AC: document contains no private MAC tracker IDs, /tmp paths, NIL/Dagger/user-specific coordination or private challenge chronology; a negative standalone-dependency statement is permissible. All three review advisories remain executable story obligations, not source-document success claims. No production/tests/runtime changes. Earlier user-choice blocker is resolved, but root backlog-review/landing hold remains.
## nd_contract
status: new
### evidence
- Approved589-line source fully read; deterministic public575-line projection defined without changing source or repository.
### proof
- [x] Private/public provenance boundary explicitly repaired.
- [ ] Canonical public projection landed and independently verified.


### 2026-09-06T08:55:41Z ramirosalas
CURRENT COMPLETE ONE-DOCUMENT ACCEPTANCE MAP 2026-09-06

This is the complete current scope/AC/proof map for MAC-l7m0, superseding every earlier landing requirement for a status index, public review file, literal private report publication, or architecture implementation. Historical contracts remain provenance only.

USER INTENT: Publish the approved standalone executable assurance contract faithfully, so implementation and independent acceptance share one permanent, reviewable source.

PRODUCES:
- docs/test-assurance-contract.md -> exact 575-line public projection specified by the preceding PUBLIC CONTRACT PROJECTION CORRECTION; SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8.
CONSUMES:
- Approved immutable source and independent challenge recorded in the preceding notes.
  source: 589-line source SHA256 e467a3b6c6f657e7a4cb28e85c688be0df24c56a7e3c76fdcfa224f7ff5d2624; challenge SHA256 0205b50a8eb2e135230173bbf4a7b7eccdde5934fabab5aa4b2dde95d474ac66.

CURRENT ACCEPTANCE CRITERIA
A1. Developer adds exactly the one public document above using the recorded deterministic line/word projection, verifies 575 lines and exact SHA256, and shows the diff against the approved source with every transformation accounted for. No second public status/review artifact is required or authorized.
A2. All normative schemas, exact interfaces/argv/runtime catalog, source/control/store identities, registration/CAS/history, native assertion/custody, frozen topology/modes, replay/state, scoped Git/Ga, finalization/no-launch order, aggregate deadlines/limits, compatibility and unsupported behavior remain unchanged. No architecture extension or silent weakening is allowed.
A3. Public content is standalone: private tracker IDs, private filesystem/workspace provenance, review chronology, user/NIL/Dagger-specific coordination are not product prerequisites or disclosures. No Paivot product/runtime/build/test dependency. Independence disclaimers are allowed.
A4. Documentation distinguishes architecture approval from implemented behavior and actual native evidence; first-release four-language scope, two-platform proof, true first-use without PASS receipt, actual process-producer audit and cumulative deadlines remain implementation acceptance obligations, not claimed results.
A5. Targeted document validation establishes exact-byte projection and normative preservation, with negative checks for omitted/renamed schema keys or interfaces, softened requirements, altered budgets, private-reference reintroduction, and false implementation-complete language. No heavy preflight, runtime provisioning, source/test edits outside this one document, installed replacement, remote mutation, or merge authority.
A6. Developer delivers the document with hash/diff and AC evidence; an independent PM compares every projected normative section to the approved immutable source and accepts or rejects. Sr PM authoring and architecture approval are not implementation acceptance. Root dispatch hold remains until independent backlog review.

DIFF BUDGET: 1 file, 575 added lines; investigate any overrun.
MANDATORY SKILLS: developer for delivery, pm_acceptor for independent acceptance; no additional domain skill identified.
OUT OF SCOPE: all runtime/code/tests implementation belongs to bounded downstream assurance stories; private source/challenge provenance stays in nd notes.

## nd_contract
status: new

### evidence
- Current one-document scope reconciled with approved architecture and root public-boundary correction.
- No source files changed and no implementation or native proof claimed.

### proof
- [ ] A1: Exact public projection bytes/hash and accounted diff.
- [ ] A2: Every normative requirement preserved.
- [ ] A3: Public standalone boundary verified.
- [ ] A4: Approval/implementation/evidence truthfulness verified.
- [ ] A5: Targeted positive and negative document validation.
- [ ] A6: Developer delivery followed by independent PM acceptance.

### 2026-09-06T10:17:35Z ramirosalas
## Implementation Evidence (DELIVERED)

PROOF:

### Document validation
- Approved source checked: 589 lines, 113493 bytes, SHA256 e467a3b6c6f657e7a4cb28e85c688be0df24c56a7e3c76fdcfa224f7ff5d2624. Independent challenge checked: 32 lines, 3378 bytes, SHA256 0205b50a8eb2e135230173bbf4a7b7eccdde5934fabab5aa4b2dde95d474ac66.
- Public projection: `wc -l -c docs/test-assurance-contract.md` => 575 lines, 109367 bytes; `shasum -a 256` => 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; mode => 0644.
- Projection review: retained source lines 1-575; accounted line replacements 1, 3, 4, 8, 477, 519, 537, 539, 541, 545, 547, 568, 575; applied the seven recorded literal substitutions and repository-wide em-dash normalization. The resulting exact expected byte hash validates the complete deterministic projection.
- Negative validation: no private tracker IDs, private `/tmp/machinery` or workspace paths, NIL/Dagger, challenge chronology, or implementation-complete claim; required schemas/API/argv/catalog/error/custody/replay/Ga keys remained present. `git diff --check` passed.
- `pvg verify docs/test-assurance-contract.md --format text` passed; it reports 0 scanned files because this documentation path is excluded by the verifier, so hash/diff/key/negative checks above are the substantive document validation.

### Commit
- Branch: story/MAC-l7m0
- SHA: 0feaebf725267f9d0045d653ef57a90af0c9d946
- Scope: one new documentation file; no source, test, runtime, CI, installation, remote, merge, or acceptance mutation.

### AC Verification
| AC | Evidence | Status |
|---|---|---|
| A1 | Exact 575-line, 109367-byte public projection and SHA256; no second artifact | PASS |
| A2 | Deterministic full projection and required normative interface/catalog/lifecycle key checks | PASS |
| A3 | Private tracker/workspace/challenge coordination negative checks; standalone dependency statement retained | PASS |
| A4 | Status and acceptance-obligation text explicitly distinguish approval from implementation/native evidence | PASS |
| A5 | Bounded exact-byte, key-presence, private-leak, false-completion and diff-whitespace checks; no heavy/runtime work | PASS |
| A6 | Committed evidence supplied for independent PM byte and normative review | PASS |

### Provenance
- A prior discarded candidate was not accepted or validated: its omitted line-477 transformation produced 109372 bytes/SHA256 4a73aa0bc6a205b3c0d672d73c398c55d2f90d1f819c342f20bafa77e775e41c and readonly mode prevented a valid patch. This fresh worktree creates the correct 0644 document and does not represent that failed attempt as proof.

LEARNINGS:
- Exact public projections need both byte verification and explicit semantic/key-preservation checks; a matching document hash is strongest when paired with an accounted transformation map.
- Documentation-only validation is appropriately bounded: the repository verifier excludes docs, so its clean result is not a substitute for projection review.
- The contract intentionally leaves native feasibility, two-platform custody/assertion proof, first-use lifecycle proof, process-producer audit, and cumulative deadlines to separately owned executable acceptance work.

## nd_contract
status: delivered

### evidence
- Commit 0feaebf725267f9d0045d653ef57a90af0c9d946 on story/MAC-l7m0.
- Exact public SHA256 22b74bc01afec68c67a9c2a4ec13b12eef910caddd4dd5e0f82464ca92bec3c8; source/challenge hashes independently checked.

### proof
- [x] A1: Exact single public projection landed.
- [x] A2: Normative contract preserved by deterministic projection and key checks.
- [x] A3: Public standalone boundary checked.
- [x] A4: Approval is not claimed as implementation/native evidence.
- [x] A5: Bounded positive and negative document validation completed.
- [x] A6: Committed for independent PM acceptance.
