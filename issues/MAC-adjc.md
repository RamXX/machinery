---
id: MAC-adjc
title: "gates: ci.yml and the Makefile never mirrored the accepted design-only example gate policy"
status: closed
priority: 1
type: bug
labels: [ci, gates, mirror-drift]
created_at: 2026-09-08T12:46:59Z
created_by: ramirosalas
updated_at: 2026-09-08T12:52:21Z
content_hash: "sha256:2b9aaafb2216f417e34dd857da1772e85099a7b96a0a087e636724637a0bf374"
closed_at: 2026-09-08T12:52:21Z
close_reason: "Fixed at f681ebd; the three mirrors now share scripts/example-gates.sh and 8/8 examples pass"
---

## Description
## Symptom

The hosted `gates` job fails on every design-only example. The accepted
example-gate policy (preflight phase 11, commit a3b9dc5) was never mirrored
into `.github/workflows/ci.yml` or the Makefile `check` target, so the three
mirrors of the same loop drifted: preflight implements the two-policy rule and
CI still runs `machinery check <design> --warnings-as-errors` for every row.

## Verbatim CI line

Run https://github.com/RamXX/machinery/actions/runs/34225004428 job `gates`
("Gate every registered example with its exact capabilities"):

```
== Gv-attest  attestation evidence ==
  warn   gt.conformance-test-shape: plan only; current implementation review missing
  warn   g4.standin-coverage: plan only; current implementation review missing
  warn   g4.pack-event-discipline: plan only; current implementation review missing
  checked: 23 covered artifacts current, 14 attested claims, 14 plan judgments, 14 claims owed

3 blocking (ERROR/DRIFT/warning) finding(s); warnings are errors
##[error]Process completed with exit code 1.
```

## Cause

Seven of the eight registered examples are design-only. Their behavioral
current-class claims are honestly attested as plans in the committed
`attestations.yaml`, so `machinery check` emits exactly three plan-only
warnings for them. `scripts/preflight.sh` phase 11 already implements the
accepted policy (go-crm strict with `--impl --complete --warnings-as-errors`;
design-only examples run a plain `check` whose warning set must equal the set
derived from their own `attestations.yaml`, diffed rather than grepped). The
`gates` job in `ci.yml` and the Makefile `check` target still carry the old
one-policy loop. Three hand-copied loops over the same inventory can drift
again the moment any one of them is edited.

## Expected

One executable script owns the policy, and preflight phase 11, the `ci.yml`
`gates` job and the Makefile `check` target all call it, so the mirrors cannot
drift. Running it locally passes all 8 registered examples, the shape
validator `laneValidatePreflight` in `scripts/integration-lane/main_test.go`
still accepts `scripts/preflight.sh`, and the `cmd/machinery` repository
contract tests that pin the CI/preflight/Makefile example wiring stay green.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-08T12:52:21Z status: open -> closed

## Links


## Comments

### 2026-09-08T12:52:21Z ramirosalas
Fixed on fix/ci-mirror at f681ebd.

New scripts/example-gates.sh owns the accepted two-policy gate. Callers: scripts/preflight.sh:181 (phase 11), .github/workflows/ci.yml gates job step 'Gate every registered example with its exact capabilities', Makefile check target via EXAMPLE_GATES. scripts/shellcheck-files.txt gained the new script.

Contract test TestExampleGatePolicyHasOneOwnerSharedByEveryMirror (cmd/machinery/repository_contract_test.go) pins the delegation in all three mirrors, forbids any mirror from re-spelling the policy, and proves the derived expectation fails closed when the check binary is silent.

Verification:
- scripts/example-gates.sh -> 'example gates: 8 registered example design suites passed' (3.0s)
- go test -count=1 -run 'Preflight|Lane' ./scripts/integration-lane -> ok 18.0s (laneValidatePreflight still accepts preflight.sh)
- go test -count=1 -run 'TestRepositoryDeterminismSurfaceContracts|TestExampleInventory|TestExampleGatePolicy|TestShellcheckInventory|TestPreflight' ./cmd/machinery -> ok 52.2s
- go test -count=1 ./scripts/release-policy -> ok 13.0s
- shellcheck over scripts/shellcheck-inventory.sh output -> clean; actionlint .github/workflows/*.yml -> clean; bash -n scripts/preflight.sh -> ok
