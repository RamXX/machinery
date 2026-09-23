---
id: MAC-yoxb
title: "formal: pinned Java provisioning races itself when several packages cold-start one cache in the Dagger test job"
status: open
priority: 2
type: bug
labels: [formal, runtimeclosure, flaky-under-load, dagger, ci]
created_at: 2026-09-23T08:39:54Z
created_by: ramirosalas
updated_at: 2026-09-23T08:39:54Z
content_hash: "sha256:ab115b70b03db758e877b90d0eb776a2d3855dae83f25088be551da9b638e3c7"
---

## Description
Observed 2026-09-23 in the Dagger test job (arm64 host, native container, fresh /home/ci cache) at 0c36e72c: internal/formal fails in TestVerifyFormalInScopePortfolioDesignUnderCustody, TestScopedPinnedJVMRunsBeforeCancellationAndIsReaped and TestScopedJavaProbePreservesIdentityUnderCustody with 'provision pinned Java runtime: open .../java/.java-stage-<n>/runtime.archive: no such file or directory', 'mkdir .../.java-stage-<n>/extracted: no such file or directory' and 'witness reserved cache stage .java-stage-<n>: file grew beyond its exact 201932480-byte snapshot'. The same package passes natively on the host (ok 106s) where the Java cache is already warm. Cause hypothesis: go test ./... runs packages in parallel and more than one package (formal, alloy, tla, cmd/machinery, cachestage) provisions the pinned Temurin into the same cache stage at once; the custody witness correctly refuses the concurrent writer but the loser does not retry or wait on the winner. Not introduced by the consistency-layer branch; it surfaced because the Dagger test job could not run before the shared-temp EACCES fix (commit e24dd7e1 on proposal/consistency-layer). Reproduce: make dagger-job JOB=test on a cold cache. Deliver: one provisioner per cache stage (a lock or a wait-for-winner path) so concurrent cold starts serialize, plus a test with two provisioners racing on an empty cache. Related: MAC-y5ho, MAC-1jhn (flaky-under-load family).

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
