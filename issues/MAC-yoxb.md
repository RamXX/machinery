---
id: MAC-yoxb
title: "formal: pinned Java provisioning races itself when several packages cold-start one cache in the Dagger test job"
status: open
priority: 2
type: bug
labels: [formal, runtimeclosure, flaky-under-load, dagger, ci]
created_at: 2026-09-23T08:39:54Z
created_by: ramirosalas
updated_at: 2026-09-23T16:36:44Z
content_hash: "sha256:265bc4ef1a7b065feb045e9d9468b0abfa73f7f140dd781fa4c3f9b3c2c2fe98"
---

## Description
Observed 2026-09-23 in the Dagger test job (arm64 host, native container, fresh /home/ci cache) at 0c36e72c: internal/formal fails in TestVerifyFormalInScopePortfolioDesignUnderCustody, TestScopedPinnedJVMRunsBeforeCancellationAndIsReaped and TestScopedJavaProbePreservesIdentityUnderCustody with 'provision pinned Java runtime: open .../java/.java-stage-<n>/runtime.archive: no such file or directory', 'mkdir .../.java-stage-<n>/extracted: no such file or directory' and 'witness reserved cache stage .java-stage-<n>: file grew beyond its exact 201932480-byte snapshot'. The same package passes natively on the host (ok 106s) where the Java cache is already warm. Cause hypothesis: go test ./... runs packages in parallel and more than one package (formal, alloy, tla, cmd/machinery, cachestage) provisions the pinned Temurin into the same cache stage at once; the custody witness correctly refuses the concurrent writer but the loser does not retry or wait on the winner. Not introduced by the consistency-layer branch; it surfaced because the Dagger test job could not run before the shared-temp EACCES fix (commit e24dd7e1 on proposal/consistency-layer). Reproduce: make dagger-job JOB=test on a cold cache. Deliver: one provisioner per cache stage (a lock or a wait-for-winner path) so concurrent cold starts serialize, plus a test with two provisioners racing on an empty cache. Related: MAC-y5ho, MAC-1jhn (flaky-under-load family).

## Acceptance Criteria


## Design


## Notes
2026-09-23 root cause and fix on branch fix/java-provision-race (base main 0a444b8d + cherry-pick e24dd7e1 as 510905fc; not pushed).

Root cause: provisionedJavaPath locked the Java cache with filelock.AcquireWait(base), a scope lock. Under go test, filelock.lockCacheBase places scope-lock files beside the test executable (.machinery-test-cache next to each package's test binary), so the package test binaries that go test ./... runs in parallel each held a private lock while all sharing one os.UserCacheDir()/machinery/java. On a cold cache every provisioner ran cachestage.Recover(base, .java-stage-) at entry and again in its deferred cleanup; that recovery quarantined or witnessed the other provisioners' live .java-stage-<n> directories, producing the observed 'open .../runtime.archive: no such file or directory', 'mkdir .../extracted: no such file or directory', and 'file grew beyond its exact snapshot'. Installed binaries were not affected (their scope locks share the user cache lock directory).

Fix: 8254e299 adds filelock.AcquireFileWaitContext (lock file inside a caller-owned private directory; same flock/LockFileEx acquisition, path-identity revalidation, bounded wait = earlier of ctx and the 10-minute acquisition limit). 1ec9aaf1 makes provisioning take <cache>/java/.java-provision.lock through it: one holder provisions under the unchanged witness discipline, the rest wait and then validate the published runtime through the warm-cache receipt/closure path; OpenJavaContext carries ctx (formal scoped opener and integration lane pass theirs). Kernel releases the lock on holder death; only the next holder recovers the stale stage. Tests: 3e4387e7 (RED, cross-process barrier race), 1ec9aaf1 (crashed winner, cancellation while waiting, 5-way byte-identical runtime and receipt). Follow-ups ded89fcb (noctx lint), 4dd76ecb (CHANGELOG without tracker id, required by TestAssuranceDocsStandaloneMachineryOnly).

Latent siblings with the same scope-lock-on-shared-cache shape, not changed here: internal/formal/formal.go fetchJar (filelock.AcquireWait(formalJarLockScope(parent))) and cmd/machinery/structurizr_provision.go.

## History


## Links


## Comments
