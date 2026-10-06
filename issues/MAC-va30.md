---
id: MAC-va30
title: "Native test adapters: consumer-declared runtime identity instead of a machinery-global pin"
status: open
priority: 1
type: feature
labels: [assurance, adapters, consumer]
created_at: 2026-09-08T19:32:45Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:53Z
content_hash: "sha256:5459633975ed6cebac732782d60bd28bf907756112d5dc63a26fe7b806218fd3"
blocks: [MAC-sd7g]
---

## Description
## Problem
The native test adapters pin one runtime identity each (Elixir 1.20.4 on OTP 29.0.6, Node 26.8.1 / TypeScript 7.0.2, CPython 3.14.7, Go 1.27.1) as machinery-global constants in the assurance catalog. A consumer design that runs a newer runtime (H2 has ruled: latest stable Elixir/OTP at bootstrap) cannot pass the adapter's identity check, so executable assurance would fail for a reason unrelated to the tests.

## Ask
Let the consumer declare its runtime identity in its assurance declaration (design/assurance/plan.json or the adapter manifest): the adapter verifies the declared identity byte-for-byte (same fail-closed discipline) instead of a global constant, while machinery's own lane keeps its pinned catalog for its fixtures. Document the supported floor per adapter and what changes when a consumer pins a newer runtime.

## Consumer
H2 (Elixir modular monolith), first affected at M1 RED.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Let design/assurance/plan.json (or the adapter manifest) declare the consumer runtime identity, verified byte for byte fail-closed; keep machinery's pinned catalog for its own lane; document the per-adapter supported floor. Evidence: Runtime identity is still machinery-global: internal/runtimeclosure/elixir.go:41 RequiredElixirVersion = "1.20.4" (OTP 29.1.1); testdata/integration-lanes/assurance-runtime-pins.json:8 pins elixir-exunit/v1 1.20.4; no consumer-declared identity field in the assurance plan or adapter manifest. Notes: Blocks MAC-sd7g (real: exact replay state needs the declared identity). The blocks on MAC-cup9 (self-design epic) and MAC-9azz (table oracles) look soft: neither needs a non-global runtime pin to be built. Candidate stale blocks: MAC-cup9, MAC-9azz.

## History
- 2026-09-24T21:33:56Z dep_added: blocks MAC-sd7g
- 2026-09-24T21:33:56Z dep_added: blocks MAC-cup9
- 2026-09-24T21:33:56Z dep_added: blocks MAC-9azz
- 2026-10-06T04:03:38Z dep_removed: no_longer_blocks MAC-9azz
- 2026-10-06T04:03:39Z dep_removed: no_longer_blocks MAC-cup9

## Links
- Blocks: [[MAC-sd7g]]

## Comments
