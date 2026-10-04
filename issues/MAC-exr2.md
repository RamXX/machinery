---
id: MAC-exr2
title: "Reopen Dependabot #13 handling once Dagger unpins otel/sdk/log v0.16.0"
status: open
priority: 3
type: chore
labels: [dagger, security, deps]
created_at: 2026-10-04T22:06:02Z
created_by: ramirosalas
updated_at: 2026-10-04T22:06:02Z
content_hash: "sha256:2f92353e019079cabcf9c12e302a2508b711d20501670cdc4acc60d392f24d56"
---

## Description
Dependabot alert #13 (go.opentelemetry.io/otel/sdk/log < 0.21.0, BatchingProcessor CPU busy-loop under exporter backpressure) was dismissed as tolerable_risk on 2026-10-04.

Why it cannot be fixed locally: the Dagger Go SDK embeds sdk/go/go.mod replaces pinning otel/log, otel/sdk/log, otlploggrpc and otlploghttp to v0.16.0, and codegen (cmd/codegen/generator/go/generate_module.go, syncModReplaceAndTidy) re-applies them with go mod edit -replace on every dagger develop and inside the engine on every dagger call. Editing .dagger/go.mod would hide the alert but the module would still build v0.16.0.

Risk: CI-only module, not shipped in the machinery binary; logs are our own CI jobs, not attacker-driven.

Done when: a Dagger release drops or raises those replaces (track dagger/dagger#13040). Then bump dagger.json engineVersion and .dagger-linux-amd64.sha256 to that release, run dagger develop, confirm .dagger/go.mod resolves otel/sdk/log >= 0.21.0 (go list -m go.opentelemetry.io/otel/sdk/log in .dagger), and make dagger-job JOB=lint passes.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
