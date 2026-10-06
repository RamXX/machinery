---
id: MAC-exr2
title: "Reopen Dependabot #13 handling once Dagger unpins otel/sdk/log v0.16.0"
status: open
priority: 4
type: chore
labels: [dagger, security, deps]
created_at: 2026-10-04T22:06:02Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:51Z
content_hash: "sha256:4b05f86ab653c42450c42a7ef6c3410cf711f731d0ca771579cf5f6732345554"
---

## Description
Dependabot alert #13 (go.opentelemetry.io/otel/sdk/log < 0.21.0, BatchingProcessor CPU busy-loop under exporter backpressure) was dismissed as tolerable_risk on 2026-10-04.

Why it cannot be fixed locally: the Dagger Go SDK embeds sdk/go/go.mod replaces pinning otel/log, otel/sdk/log, otlploggrpc and otlploghttp to v0.16.0, and codegen (cmd/codegen/generator/go/generate_module.go, syncModReplaceAndTidy) re-applies them with go mod edit -replace on every dagger develop and inside the engine on every dagger call. Editing .dagger/go.mod would hide the alert but the module would still build v0.16.0.

Risk: CI-only module, not shipped in the machinery binary; logs are our own CI jobs, not attacker-driven.

Done when: a Dagger release drops or raises those replaces (track dagger/dagger#13040). Then bump dagger.json engineVersion and .dagger-linux-amd64.sha256 to that release, run dagger develop, confirm .dagger/go.mod resolves otel/sdk/log >= 0.21.0 (go list -m go.opentelemetry.io/otel/sdk/log in .dagger), and make dagger-job JOB=lint passes.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: When a Dagger release drops the replace, bump engineVersion and sha256, run dagger develop, confirm sdk/log >= 0.21.0, run make dagger-job JOB=lint. Evidence: Still pinned: .dagger/go.mod:53 replace go.opentelemetry.io/otel/sdk/log => v0.16.0; dagger.json engineVersion v0.21.10 (commit d74354dc, 2026-10-04). Upstream dagger/dagger#13040 status not verified from here. Notes: Waiting-on-upstream chore; CI-only module, risk accepted. Could be deferred with a re-check at each Dagger bump.

## History


## Links


## Comments
