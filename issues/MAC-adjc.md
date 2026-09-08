---
id: MAC-adjc
title: "gates: ci.yml and the Makefile never mirrored the accepted design-only example gate policy"
status: open
priority: 1
type: bug
labels: [ci, gates, mirror-drift]
created_at: 2026-09-08T12:46:59Z
created_by: ramirosalas
updated_at: 2026-09-08T12:46:59Z
content_hash: "sha256:233913fb0b13e4921e16f6afb95c4e6ddd16003146f7e2515ae4f1c84e770b2c"
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


## Links


## Comments
