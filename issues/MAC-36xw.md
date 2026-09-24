---
id: MAC-36xw
title: "cmd/machinery: TestProvisionStructurizrWaiterRecoversFromCrashedWinner flaked once on the hosted macOS runner (waiter returned a provisioned path while the helper held the lock)"
status: open
priority: 2
type: bug
labels: [flaky-under-load, macos, ci, cmd-machinery, structurizr]
created_at: 2026-09-23T22:35:13Z
created_by: ramirosalas
updated_at: 2026-09-24T05:13:12Z
content_hash: "sha256:821e0def39ab7d01ae4229eb97e7182a61af8ceed5d4e8ca35966feeb40e21e9"
---

## Description
Observed 2026-09-23 on ci run 35924889820 (commit 8ab48a15), job native-tests (macos-latest, go test -count=1 ./... -timeout=30m): 'structurizr_provision_race_test.go:381: waiter did not wait for the live winner: path=<launcher under the test cache root>'. The re-run of the same job on the same commit passed, and the test passes locally on arm64 macOS six times in a row filtered and once in the full cmd/machinery package. The Java and formal-jar twins of this test did not flake. Facts established: the helper re-applies the cache root from its env (HOME, XDG_CACHE_HOME, LOCALAPPDATA), marks 'holding-' only inside the download hook, which runs after filelock.AcquireFileWaitContext took the flock on <cache>/machinery/structurizr/.structurizr-provision.lock, so at mark time the lock is held by a live process; darwin uses BSD flock (internal/filelock/flock_unix.go) with an in-process identity reservation and no stale-lock breaking. Open question: how the waiter in the test process acquired the flock or found the target warm within 500 ms while the child held it. Candidates to check with hosted-runner evidence: an inherited descriptor that shares the child's open file description (flock is per open-file-description, and a shared description makes the parent and child co-owners of the same lock), the 500 ms assertion window versus a slow child on a loaded runner, and /var versus /private/var path identity on the runner's TMPDIR. Reproduce by re-running the native-tests job several times or by adding a diagnostic that records the lock file's device:inode and the flock result in both processes. Not a release blocker: the run is green on re-run and the fix under test (one provisioner per cache) is verified deterministically by the barrier tests on Linux and macOS.

## Acceptance Criteria


## Design


## Notes
Second occurrence 2026-09-24 on ci run 35955372364 (the cancellation variant, same shape). 4bb85b78 replaces the helpers' bare select {} with a sleep loop (a process whose goroutines are all blocked with no timer pending is a runtime deadlock and dies, releasing the flock; the helper binary runs with no -test.timeout so no timer exists) and makes every waiter assertion print the helper's liveness, exit status and output. ci on 4bb85b78 passed first try. Keep open until the instrumentation has seen one more hosted macOS run under load; close if the sleep loop explanation holds.

## History


## Links


## Comments
