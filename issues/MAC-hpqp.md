---
id: MAC-hpqp
title: "Execute every required infrastructure test deterministically"
status: open
priority: 0
type: bug
labels: [hard-tdd]
parent: MAC-ui8a
created_at: 2026-09-05T19:44:27Z
created_by: ramirosalas
updated_at: 2026-09-05T19:46:12Z
content_hash: "sha256:339213f39ea5893de9b76f0bd442051806e27a8537cb98fb74f0e1e8e693f035"
blocks: [MAC-hlae, MAC-yhg5, MAC-2n83, MAC-hwdb, MAC-vx24, MAC-gcrr, MAC-ou97, MAC-hy71]
---

## Description
### USER INTENT
The maintainer can run one mandatory integration lane locally and in CI and know every required real-runtime safety test actually executed with correct prerequisites and no leaked resources.

### Context (Embedded)
Anchor round 1 found a cross-story execution gap: new Docker/Java/Node cases were assigned ordinary package suites, but macOS native CI lacks Docker and preflight provisions checker image after its race suite. Required tests need explicit lane selection, not skip-if-missing. This is repository contributor infrastructure, standalone Go/Node/Java/Docker only; it never requires Paivot.

### Ownership
Own scripts/integration-lane/main.go, scripts/integration-lane/main_test.go, testdata/integration-lanes/schema.json, testdata/integration-lanes/pilot.json, testdata/integration-lanes/runtime-pins.json, cmd/machinery/integration_lane_test.go, scripts/preflight.sh, .github/workflows/ci.yml, .github/workflows/formal.yml, .github/workflows/nightly.yml, Makefile, scripts/shellcheck-files.txt. Preserve concurrent edits. Runtime stories own separate lane fragments and their tests, never the common inventory engine. The release workflow story modifies shared workflows afterward.

### Boundary Map
PRODUCES:
- scripts/integration-lane/main.go -> new contributor CLI: go run ./scripts/integration-lane --lane required; closed inventory validation/provisioning/native execution/accounting/cleanup
- scripts/integration-lane/main_test.go -> native unit and actual process runner validation
- testdata/integration-lanes/schema.json -> closed versioned per-suite fragment contract: exact source/test identities, native adapter, runtime requirements, command selection, timeout/output/resource bounds
- testdata/integration-lanes/pilot.json -> nonempty initial suite covering actual provisioned runtime path
- testdata/integration-lanes/runtime-pins.json -> explicit immutable OCI platform/digest and pinned engine/runtime identities
- cmd/machinery/integration_lane_test.go -> build-tagged real runtime pilot tests, excluded only from named ordinary native lane
- scripts/preflight.sh -> mandatory integration lane after prerequisite provisioning, before any phase consuming service-backed evidence
- .github/workflows/ci.yml -> required Linux runtime-lane job plus explicit service-free native portability suites
- .github/workflows/formal.yml -> formal lane/engine provisioning consistent with shared inventory
- .github/workflows/nightly.yml -> declared service-free versus provisioned required-lane execution
- Makefile -> test-integration entrypoint invokes exact same required lane
- scripts/shellcheck-files.txt -> registers any introduced executable shell surface, or verified unchanged closed inventory if implementation remains Go-only
CONSUMES:
- Existing scripts/run-safe/main.go.
  spec: run(args []string, stdout, stderr io.Writer) int; CLI -timeout, -stdout-limit, -stderr-limit, -expect-stdout-file, -expect-stderr-file followed by -- and command argv.
- Existing scripts/example-inventory.sh.
  source: rows/formal/checkers modes validate closed example manifest and emit selected subjects.
- Existing scripts/preflight.sh.
  source: checker image python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc on linux/amd64; bounded docker pull, RepoDigests/platform inspect and offline execution establish closure.

### Story Acceptance Criteria
1. A required named lane discovers and validates every committed suite fragment using a closed schema and matches exact selected native test identities to its registered source files. Unknown/duplicate/orphan fragments, unregistered integration test files, invalid paths and empty lane fail. Every fragment names actual runnable cases, never future placeholders.
2. Ordinary go test ./... native lanes deliberately exclude build-tagged runtime tests and do not probe/skip absent services. Required lane explicitly selects them and fails if Docker/Node/Java prerequisites are unavailable. Runtime tests cannot become dormant via env flags, t.Skip, skipped cases, xfail, filtering or empty package selection.
3. Provision and verify immutable OCI digest/platform, pinned formal/Java closure and supported Node runtime BEFORE their tests. A cold-cache positive provisions then executes; absent/offline/unavailable runtime and wrong digest/platform reject with clear diagnostics. Existing pinned example image may be reused; never trust a mutable tag.
4. Capture real Go JSON and Node native structured execution output under bounded processes and verify every registered test started, terminated and passed exactly as required. Missing/incomplete/duplicate/skipped tests, cached or malformed/truncated events, extra unexpected failures and fabricated aggregate-only summaries block. Success records actual nonzero execution counts per suite and runtime.
5. Required Linux hosted CI and final local preflight invoke the same lane/inventory. macOS native portability remains service-free while local macOS preflight with Docker Desktop executes the same required safety coverage using pinned platform. Job status must be available for release required-check policy; no remote mutation occurs during implementation.
6. Eliminate preflight ordering hazard: service-backed tests are never executed in the early ordinary race suite before image/runtime provisioning. Preserve existing formal/C4/checker gates, full-history needs and cheapest-first safe ordering; no weakening via SKIP_PREFLIGHT or optional missing-runtime fallback.
7. Initial story is independently executable: include a nonempty actual bounded OCI success/termination pilot plus actual formal/Node invocation where classified as required. Later stories add their own fragment only alongside executable tests. Shared runner discovers their union deterministically without each touching the root manifest.
8. Teardown validates all owned containers/processes/temp roots are removed after success, failed assertion, timeout, cancellation and provisioning error, or fails with actionable cleanup error. Never remove unrelated containers or user files; preserve existing dagger-engine-v0.21.9.
9. RED tests demonstrate existing missing-lane/incorrect-order or wrong-accounting behavior through genuine runtime assertions with passing controls; freeze tests/config/fixtures after approval. No compile/infra-only RED. Targeted lane/provisioning/accounting tests and actionlint pass; do NOT run full scripts/preflight.sh before final epic completion.
10. Real full-path no-mock tests cover provisioned success, fresh cache, missing runtime, zero execution, skipped required test, wrong pin, partial output and leaked owned resource. Run native suites without services to prove separation and required lane with services to prove coverage. Machine-readable report identifies exactly what ran; it cannot stand in for native events.

### Testing Requirements
Hard TDD explicitly authorized. Unit plus Integration tests: MANDATORY (no mocks), actual bounded processes and real Docker daemon; no skip-if-missing. Unit parser fixtures may test malformed events but cannot replace real execution proof. Targeted go test ./scripts/integration-lane and go run ./scripts/integration-lane --lane required; actionlint and native ordinary-lane selection checks. Full heavy preflight ONLY at end.
No GitHub push/sync/mutation, no installed binary/plugin/skill/agent replacement. Isolated builds/homes only. No Paivot product dependency.

### OUT OF SCOPE
- Individual saga/OCI/recovery/OpenCode/process/capstone behavior: sibling stories own their real tests and inventory fragments, consuming this lane.
- Final full-preflight execution belongs to epic completion after all siblings.

### DIFF BUDGET
~10-13 files, under 1800 changed LOC; investigate overrun rather than weaken execution guarantees.

### MANDATORY SKILLS
developer; codebase-memory; pm_acceptor.

### nd_contract
status: new

#### evidence
Created 2026-09-05 to repair Anchor round-1 general execution-lane gap; source interfaces and provisioning sequence read directly.

#### proof
- [ ] AC #1-10: independently verified with real execution.


## Acceptance Criteria


## Design


## Notes
## nd_contract
status: new

### evidence
- Backlog-only integration-lane contract authored from actual CI/preflight ordering and bounded runner interfaces.
- No production edits or heavy preflight; parent-authorized Anchor round-1 repair.

### proof
- [ ] AC #1: independent execution-lane verification pending
- [ ] AC #2: independent execution-lane verification pending
- [ ] AC #3: independent execution-lane verification pending
- [ ] AC #4: independent execution-lane verification pending
- [ ] AC #5: independent execution-lane verification pending
- [ ] AC #6: independent execution-lane verification pending
- [ ] AC #7: independent execution-lane verification pending
- [ ] AC #8: independent execution-lane verification pending
- [ ] AC #9: independent execution-lane verification pending
- [ ] AC #10: independent execution-lane verification pending

## History
- 2026-09-05T19:45:31Z dep_added: blocks MAC-hlae
- 2026-09-05T19:45:31Z dep_added: blocks MAC-yhg5
- 2026-09-05T19:45:32Z dep_added: blocks MAC-2n83
- 2026-09-05T19:45:32Z dep_added: blocks MAC-hwdb
- 2026-09-05T19:45:32Z dep_added: blocks MAC-vx24
- 2026-09-05T19:45:33Z dep_added: blocks MAC-gcrr
- 2026-09-05T19:45:33Z dep_added: blocks MAC-ou97
- 2026-09-05T19:45:33Z dep_added: blocks MAC-hy71

## Links
- Parent: [[MAC-ui8a]]
- Blocks: [[MAC-hlae]], [[MAC-yhg5]], [[MAC-2n83]], [[MAC-hwdb]], [[MAC-vx24]], [[MAC-gcrr]], [[MAC-ou97]], [[MAC-hy71]]

## Comments
