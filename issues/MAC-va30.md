---
id: MAC-va30
title: "Native test adapters: consumer-declared runtime identity instead of a machinery-global pin"
status: open
priority: 1
type: feature
labels: [assurance, adapters, consumer]
created_at: 2026-09-08T19:32:45Z
created_by: ramirosalas
updated_at: 2026-09-08T19:32:45Z
content_hash: "sha256:12bc46c0b407bb48373e2db4629bff69933d99ed041a1a72525b285dd92ac147"
blocks: [MAC-sd7g, MAC-cup9]
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


## History
- 2026-09-24T21:33:56Z dep_added: blocks MAC-sd7g
- 2026-09-24T21:33:56Z dep_added: blocks MAC-cup9

## Links
- Blocks: [[MAC-sd7g]], [[MAC-cup9]]

## Comments
