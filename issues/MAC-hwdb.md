---
id: MAC-hwdb
title: "Bound OpenCode governance subprocesses"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:30:27Z
created_by: ramirosalas
updated_at: 2026-09-05T19:33:48Z
content_hash: "sha256:2dd32d3bf49366b4aac9ed7b01fc6e93f3c9bc53f92008273d89f828336524da"
blocks: [MAC-gcrr, MAC-ou97]
---

## Description
## USER INTENT
Users need Machinery's green results to establish the intended safety claim, not merely artifact shape.

## Context (Embedded)
Assessment additional finding: defaultRunner accumulates arbitrary stdout/stderr with no deadline and runMachinery accepts parsed JSON of arbitrary shape. Existing seven adapter tests inject runner functions and do not establish real subprocess behavior.

## Ownership
Own only these paths and directly associated tests: adapters/opencode/plugins/machinery.js, adapters/opencode/plugins/machinery.test.mjs. You are not alone in this codebase; preserve other edits and coordinate any shared-file changes with dispatcher.

## Boundary Map
PRODUCES:
- adapters/opencode/plugins/machinery.js -> hardened behavior and regression proof
- adapters/opencode/plugins/machinery.test.mjs -> hardened behavior and regression proof
CONSUMES:
- Existing Machinery source interfaces.
  spec: function defaultRunner(root, payload); async function runMachinery(run, root, payload)

## Acceptance Criteria
1. Default runner applies finite deadline and independent bounded stdout/stderr, terminates owned subprocesses on expiry/overflow, closes resources and settles exactly once across error/close/EPIPE races.
2. Validate allowed hook response shapes and decisions; malformed JSON, unknown/wrong-shape responses, invalid fields, nonzero exit and transport failure block. Preserve documented empty-success and valid host allow/deny protocols.
3. Positive actual machinery hook invocation passes valid governance output. Negative actual child processes hang, flood stdout/stderr, return invalid shapes, exit early or ignore termination; bounded failure is observed without retained children.
4. No shell interpolation; preserve safe argv/stdin construction and correct path handling. Test Node runtime explicitly and compatibility with supported Bun host without claiming unrun environments.
5. Add native-subprocess tests in addition to existing injected unit tests; no skip-if-missing and no mocked runner accepted as integration proof.

## Testing Requirements
- Hard TDD explicitly authorized. RED author commits tests first; intended behavioral assertions fail on unchanged production, with a passing control. Compile/import/infra errors are not RED evidence. PM independently replays RED. GREEN implementer does not edit/delete frozen RED tests or fixtures; any repair requires explicit reviewer authorization and re-RED.
- Unit tests plus Integration tests: MANDATORY (no mocks). Real CLI/filesystem/service path, no stubs, no skip-if-missing. Missing prerequisites block rather than pass.
- Commands: node --test adapters/opencode/plugins/machinery.test.mjs; real process tests use isolated temporary PATH fixtures for adversarial child behavior plus one actual built machinery binary for positive protocol.
- No full scripts/preflight.sh during this story; final epic gate owns heavy preflight. No GitHub push, sync, release, or remote mutation. Local story worktree only.

## OUT OF SCOPE
- Other assessment subsystems are separate epic stories; include small directly related fixes needed for this guarantee rather than inventing exclusions.
- Global heavy preflight, main merge and local release binary belong to final epic gate.

## DIFF BUDGET
- ~2-3 files, under 650 changed LOC; material overrun requires PM investigation, not weakened requirements.

## MANDATORY SKILLS
- developer for implementation; codebase-memory for discovery; pm_acceptor for independent acceptance.

## Delivery Requirements
Use pvg story deliver, never close. Append RED/GREEN SHAs, exact commands and outputs, test inventory, independent proof per AC, and any residual limits to shared nd. Do not use pushing pvg story merge.

## nd_contract
status: new

### evidence
- Created 2026-09-05; source signatures verified at assessment base 497419ab4512fcff765cd5feb27aed4c67b5608d.

### proof
- [ ] AC #1: independently verified
- [ ] AC #2: independently verified
- [ ] AC #3: independently verified
- [ ] AC #4: independently verified
- [ ] AC #5: independently verified

## Acceptance Criteria


## Design


## Notes
AUTHORITATIVE USER CONSTRAINTS 2026-09-05: Machinery product must be standalone, never require Paivot/pvg/nd, workflow labels or commit conventions. Local development coordination only may use Paivot. Another agent uses installed Machinery in NIL: do not replace installed binary/plugins/skills/agents; no dev-link or live install/update. Build isolated candidate only. No GitHub push/mutation during work. Full scripts/preflight.sh only final epic gate. RED author may update preexisting tests that encode superseded unsafe behavior with explicit review and genuine assertion-failure proof; after RED approval freeze exact tests/fixtures/config bytes.

## History
- 2026-09-05T19:35:09Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:36:15Z dep_added: blocks MAC-ou97

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-gcrr]], [[MAC-ou97]]

## Comments
