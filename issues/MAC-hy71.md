---
id: MAC-hy71
title: "Gate release publication on exact-commit verification"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:33:47Z
created_by: ramirosalas
updated_at: 2026-09-05T19:45:33Z
content_hash: "sha256:c72eb7d4f4ff25b6c3521f191e7de1213a145c58ca6cf1166c043669087c2dc9"
blocks: [MAC-gcrr, MAC-ou97]
blocked_by: [MAC-hpqp]
---

## Description
## USER INTENT
Strengthen Machinery mission-critical assurance with observable fail-closed behavior and precise limits.

## Context (Embedded)
Assessment F5 release v0.6.11 published before failed same-SHA CI; current branch protection lacks required checks. F7 nightly golden checkout is shallow and acceptance ancestors unavailable. User prohibits any remote mutations during work.

## Ownership
Own only .github/workflows/release.yml, .github/workflows/nightly.yml, .github/workflows/ci.yml, .github/workflows/formal.yml, .github/workflows/security.yml, scripts/release-policy/main.go, scripts/release-policy/main_test.go, docs/release-policy.json and directly associated tests. You are not alone: preserve other edits and coordinate shared paths.

## Boundary Map
PRODUCES:
- .github/workflows/release.yml -> hardened contract and regression evidence
- .github/workflows/nightly.yml -> hardened contract and regression evidence
- .github/workflows/ci.yml -> hardened contract and regression evidence
- .github/workflows/formal.yml -> hardened contract and regression evidence
- .github/workflows/security.yml -> hardened contract and regression evidence
- scripts/release-policy/main.go -> hardened contract and regression evidence
- scripts/release-policy/main_test.go -> hardened contract and regression evidence
- docs/release-policy.json -> hardened contract and regression evidence
CONSUMES:
- Existing Machinery implementation.
  source: GitHub workflows on push/workflow_dispatch; existing release build verifies tag SHA = HEAD = GITHUB_SHA; publish currently needs only build. New standalone local release-policy verifier is owned output, not pre-existing API.

### Story Acceptance Criteria
1. Release publication deterministically requires CI, formal and security success for exact release commit plus permitted main ancestry/tag version. Pending, failed, cancelled, skipped, stale SHA, wrong workflow identity and missing required checks block; no unsafe timeout fallback.
2. Both tag and manual dispatch obey the same prerequisites with least required token permissions and no executable interpolation of untrusted event strings. Gate build/publish artifact identity as well as status identity.
3. Nightly jobs that perform acceptance ancestry checks fetch sufficient full history; local shallow-clone negative and repaired positive exercise the actual git ancestry behavior.
4. Commit a ready-to-apply repository required-checks policy payload and documented application/verification commands, but do not apply it, dispatch runs, publish, push or mutate GitHub. Clearly distinguish locally tested policy from remotely enforced state.
5. Unit policy tests enumerate all reject states; integration uses real local git histories/tag ancestry and real process policy invocation with captured API response fixtures (not claimed as live remote enforcement); actionlint and existing workflow contract tests pass. Read-only live API verification may confirm names only.

## Testing Requirements
- Explicit hard TDD: separate RED test author, expected behavioral assertion failure plus passing control on unmodified production, independent PM replay, frozen RED test/fixture bytes retained through GREEN. Compilation, import, timeout or unavailable infrastructure are not valid RED. Existing tests encoding the unsafe contract may be corrected during reviewed RED; GREEN cannot silently weaken them.
- Integration tests: MANDATORY (no mocks). Exercise real process/filesystem/runtime boundaries; no stubs, no skip-if-missing. Fixture policy inputs may test parser logic but are not live remote proof.
- go test ./scripts/release-policy; go test ./cmd/machinery -run 'Repository|Workflow|Release|Shallow'; actionlint; real disposable shallow git clone then fetch history; full pipeline deferred to final preflight.
- Full scripts/preflight.sh ONLY at end of epic. No pushes, sync, remote mutation, installed binary/plugin/agent/skill replacement, or dev-link. Another agent uses installed Machinery in NIL; builds/tests use isolated output and homes.
- Shipped product enforcement is standalone Machinery: never requires pvg, nd, Paivot labels/metadata or commit conventions. Paivot is local development coordination only.

## OUT OF SCOPE
- Other assessment areas have sibling stories; final preflight/local main merge/isolated candidate binary are final gate responsibilities. Do not omit small directly related safety fixes.

## DIFF BUDGET
- ~7-10 files, under 1100 changed LOC; overrun triggers PM investigation rather than weaker proof.

## MANDATORY SKILLS
- developer; codebase-memory; pm_acceptor.

## Delivery Requirements
Append exact RED/GREEN SHAs, commands/output, ownership inventory and per-AC proof. Use pvg story deliver; PM accepts separately. Never invoke pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05 from assessment and source-verified interfaces.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

## Acceptance Criteria


## Design


## Notes
ANCHOR ROUND-1 AUTHORITATIVE CI OWNERSHIP REPAIR
CONSUMES:
- MAC-hpqp: .github/workflows/ci.yml
  source: Required provisioned integration-lane job and service-free native suites.
- MAC-hpqp: .github/workflows/formal.yml
  source: Shared closed runtime inventory and formal engine provisioning.
- MAC-hpqp: .github/workflows/nightly.yml
  source: Explicit native versus required-lane selection.
This story modifies these shared workflow files only after MAC-hpqp acceptance. Release required-check policy must include the required integration-lane job's actual exact identity, so a missing/skipped/failed lane prevents publication even if ordinary native tests pass. Preserve provisioning before invocation, selected-test accounting, full history and no-orphan validation. actionlint remains provisioned by existing local/CI pinned tool contract; local git policy tests remain service-free and never invoke GitHub mutations. Add targeted release-policy regression for absent/failed integration status. Do not reintroduce bare unconditional Docker tests into ordinary macOS or Linux suites.

## History
- 2026-09-05T19:35:08Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:16Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:33Z dep_added: blocked_by MAC-hpqp

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]
- Blocked by: [[MAC-hpqp]]

## Comments
