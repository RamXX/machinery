//go:build machinery_integration

// MAC-yhg5 deterministic checker-container lifetime ownership. Every test in
// this file is a required real-integration case registered in
// testdata/integration-lanes/oci.json; the set of Test functions and the
// fragment must stay in exact both-directions sync. Docker lifecycle cases run
// against the real daemon and the pinned immutable checker image; a missing or
// unreachable daemon fails the case, it is never skipped. Shell-backed stub
// engines are failure-injection controls for engine-cleanup argv semantics
// only; they never substitute for the daemon-backed lifecycle proofs.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/checker"
)

// lifecycleImage is the repo's pinned immutable checker userspace, identical to
// testdata/integration-lanes/runtime-pins.json. Runs prove ownership, budgets
// and cleanup against the real daemon using only this digest-addressed image.
const lifecycleImage = "python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc"

// lifecycleCIDFileName is the per-run container identity contract: runCheckerOCI
// must register the daemon-side container ID at <workDir>/checker.cid before the
// checker starts, so timeout/cancellation can force-remove exactly that ID.
const lifecycleCIDFileName = "checker.cid"

var lifecycleHexContainerID = regexp.MustCompile(`^[a-f0-9]{64}$`)

// lifecycleWorkDir creates a checker work directory whose runtime-inputs bind
// source exists, matching the mount contract runCheckerOCI must satisfy.
func lifecycleWorkDir(t *testing.T) string {
	t.Helper()
	work := t.TempDir()
	if err := os.Mkdir(filepath.Join(work, "runtime-inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	return work
}

// lifecycleRealEngine resolves the real docker CLI exactly like the frozen OCI
// golden: a snapshot-safe hard-linked regular file, with DOCKER_HOST pinned to
// the resolved context endpoint so the deterministic checker environment
// transports one explicit endpoint. A missing CLI or unresponsive daemon fails
// the required integration.
func lifecycleRealEngine(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Fatalf("required checker lifecycle lane has no Windows daemon provisioning")
	}
	docker, err := exec.LookPath("docker")
	if err != nil {
		t.Fatalf("required checker lifecycle integration has no Docker engine: %v", err)
	}
	docker, err = filepath.EvalSymlinks(docker)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(docker)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("Docker does not resolve to a snapshot-safe regular file: %v", err)
	}
	engineDir := t.TempDir()
	engine := filepath.Join(engineDir, "docker")
	if err := os.Link(docker, engine); err != nil {
		body, readErr := os.ReadFile(docker)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if writeErr := os.WriteFile(engine, body, info.Mode().Perm()); writeErr != nil {
			t.Fatalf("copy Docker executable after hard-link failure %v: %v", err, writeErr)
		}
	}
	if os.Getenv("DOCKER_HOST") == "" {
		endpoint, inspectErr := dockerContextEndpoint(docker)
		if inspectErr != nil {
			t.Fatalf("required checker lifecycle integration cannot resolve Docker endpoint: %v", inspectErr)
		}
		t.Setenv("DOCKER_HOST", endpoint)
	}
	return engine
}

// lifecycleRequirePinnedImage proves the exact immutable identity is present
// locally, provisioning it once via a bounded digest pull when missing. Failure
// is fatal: the required integration never silently skips.
func lifecycleRequirePinnedImage(t *testing.T, engine string) {
	t.Helper()
	digest, err := checker.OCIImageDigest(lifecycleImage)
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	if err := verifyLocalOCIImage([]string{engine}, lifecycleImage, digest, testRuntimePlatform, checkerOCIControlPlaneTimeout, work); err == nil {
		return
	}
	out, errOut, pullErr := lifecycleDockerOutput(t, 10*time.Minute, "pull", "--quiet", "--platform", testRuntimePlatform, lifecycleImage)
	if pullErr != nil {
		t.Fatalf("required pinned checker image %s is not provisioned and cannot be pulled: %v\n%s\n%s", lifecycleImage, pullErr, out, errOut)
	}
	if err := verifyLocalOCIImage([]string{engine}, lifecycleImage, digest, testRuntimePlatform, checkerOCIControlPlaneTimeout, work); err != nil {
		t.Fatalf("required pinned checker image %s is unusable after provisioning: %v", lifecycleImage, err)
	}
}

// lifecycleDockerRun runs one bounded docker CLI probe under ctx. Cleanup
// paths must pass a context.Background() derivative: t.Context() is already
// canceled once cleanups run, which would kill the probe before it reaches
// the daemon.
func lifecycleDockerRun(ctx context.Context, timeout time.Duration, docker string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, docker, args...)
	cmd.Env = os.Environ()
	stdout := boundedCheckerOutput{limit: 1 << 20}
	stderr := boundedCheckerOutput{limit: 1 << 20}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	return stdout.String(), stderr.String(), runErr
}

// lifecycleDockerOutput runs one bounded docker CLI probe with the ambient
// daemon endpoint and returns its stdout and stderr.
func lifecycleDockerOutput(t *testing.T, timeout time.Duration, args ...string) (string, string, error) {
	t.Helper()
	docker, err := exec.LookPath("docker")
	if err != nil {
		t.Fatalf("required checker lifecycle integration has no Docker engine: %v", err)
	}
	return lifecycleDockerRun(t.Context(), timeout, docker, args...)
}

func lifecycleCIDPath(work string) string {
	return filepath.Join(work, lifecycleCIDFileName)
}

// lifecycleReadContainerID reads the captured container identity. An absent or
// empty file yields "" so callers can distinguish "no identity captured".
func lifecycleReadContainerID(t *testing.T, work string) string {
	t.Helper()
	body, err := os.ReadFile(lifecycleCIDPath(work))
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(body))
}

// lifecycleWaitForContainerID blocks until runCheckerOCI registers the
// container identity, proving the run owns an unambiguous daemon-side identity
// while the checker is still executing.
func lifecycleWaitForContainerID(t *testing.T, work string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if id := lifecycleReadContainerID(t, work); id != "" {
			return id
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("runCheckerOCI captured no container identity at %s within %s", lifecycleCIDPath(work), timeout)
	return ""
}

func lifecycleAssertHexContainerID(t *testing.T, id string) {
	t.Helper()
	if !lifecycleHexContainerID.MatchString(id) {
		t.Fatalf("captured checker container identity %q is not a full 64-hex daemon ID", id)
	}
}

// lifecycleAssertContainerAbsent proves by actual docker inspect that the exact
// owned container ID no longer exists on the daemon.
func lifecycleAssertContainerAbsent(t *testing.T, id string) {
	t.Helper()
	out, errOut, err := lifecycleDockerOutput(t, 30*time.Second, "inspect", id)
	if err == nil {
		t.Fatalf("owned checker container %s still exists after runCheckerOCI returned: %s", id, out)
	}
	if !strings.Contains(strings.ToLower(out+errOut), "no such object") {
		t.Fatalf("absence proof for checker container %s was inconclusive: %v\n%s\n%s", id, err, out, errOut)
	}
}

// lifecycleLeakedRunningContainerIDs scans the daemon for containers of the
// pinned checker image that are both RUNNING and created after started. The
// ownership contract is "no owned RUNNING container remains": a pre-existing
// dead container of the pinned image (a foreign fixture left by an unrelated
// suite) is out of scope and must not fail unrelated lifecycle runs, while a
// genuinely leaked test-owned container is still running (timeout leak) or
// freshly created within the run window and stays detectable.
func lifecycleLeakedRunningContainerIDs(t *testing.T, started time.Time) []string {
	t.Helper()
	out, errOut, err := lifecycleDockerOutput(t, 30*time.Second, "ps", "--no-trunc", "--format", "{{.ID}}", "--filter", "status=running", "--filter", "ancestor="+lifecycleImage)
	if err != nil {
		t.Fatalf("owned checker container inventory failed: %v\n%s\n%s", err, out, errOut)
	}
	leaked := []string{}
	for _, id := range strings.Fields(out) {
		created, errOut, err := lifecycleDockerOutput(t, 30*time.Second, "inspect", "--format", "{{.Created}}", id)
		if err != nil {
			if strings.Contains(strings.ToLower(created+errOut), "no such object") {
				continue
			}
			t.Fatalf("running checker container %s could not be inspected: %v\n%s\n%s", id, err, created, errOut)
		}
		createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(created))
		if err != nil {
			t.Fatalf("running checker container %s has an unparsable creation time %q: %v", id, created, err)
		}
		if !createdAt.Before(started) {
			leaked = append(leaked, id)
		}
	}
	return leaked
}

// lifecycleAssertNoOwnedCheckerContainers is the post-return proof: every
// captured identity must be absent, and no container of the pinned checker
// image that is RUNNING and created after the test started may remain on the
// daemon. Containers of the pinned image that predate the test (including
// dead foreign fixtures) are outside this suite's ownership.
func lifecycleAssertNoOwnedCheckerContainers(t *testing.T, started time.Time, works ...string) {
	t.Helper()
	for _, work := range works {
		if id := lifecycleReadContainerID(t, work); id != "" {
			lifecycleAssertHexContainerID(t, id)
			lifecycleAssertContainerAbsent(t, id)
		}
	}
	if ids := lifecycleLeakedRunningContainerIDs(t, started); len(ids) != 0 {
		t.Fatalf("leaked running checker container(s) of the pinned image created during the test remain on the daemon: %v", ids)
	}
}

// lifecycleForeignContainer starts an unrelated control container the checker
// lifecycle must never touch, returning its unique disposable name.
func lifecycleForeignContainer(t *testing.T) string {
	t.Helper()
	name := fmt.Sprintf("machinery-yhg5-foreign-%d-%d", os.Getpid(), time.Now().UnixNano())
	out, errOut, err := lifecycleDockerOutput(t, 60*time.Second, "run", "-d", "--name", name, "busybox:latest", "sleep", "300")
	if err != nil {
		t.Fatalf("foreign survival control container could not start: %v\n%s\n%s", err, out, errOut)
	}
	t.Cleanup(func() {
		docker, err := exec.LookPath("docker")
		if err != nil {
			return
		}
		_, _, _ = lifecycleDockerRun(context.Background(), 30*time.Second, docker, "rm", "-f", name)
	})
	return name
}

func lifecycleAssertForeignRunning(t *testing.T, name string) {
	t.Helper()
	out, errOut, err := lifecycleDockerOutput(t, 30*time.Second, "inspect", "--format", "{{.State.Status}}", name)
	if err != nil || strings.TrimSpace(out) != "running" {
		t.Fatalf("unrelated container %s did not survive checker cleanup: %v\n%s\n%s", name, err, out, errOut)
	}
}

// lifecycleWriteStubEngine writes a POSIX sh stub that emulates the engine argv
// contract runCheckerOCI must satisfy: on "run" it records cid under the
// --cidfile path it was given (hold: sleep; quick: exit 0), and on "rm" it
// records the marker and exits with the configured status.
func lifecycleWriteStubEngine(t *testing.T, cid, mode, rmMarker string, rmExit int) string {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "freebsd" {
		t.Fatalf("stub engine failure-injection control requires a POSIX /bin/sh host")
	}
	body := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "rm" ]; then
  printf 'stub-rm-invoked\n' > '%s'
  printf 'stub rm exit %d\n' >&2
  exit %d
fi
cidfile=''
prev=''
for arg in "$@"; do
  if [ "$prev" = "--cidfile" ]; then cidfile="$arg"; fi
  prev="$arg"
done
if [ -z "$cidfile" ]; then
  printf 'stub engine: no --cidfile in argv\n' >&2
  exit 31
fi
printf '%%s' '%s' > "$cidfile"
if [ '%s' = 'hold' ]; then
  sleep 30
fi
exit 0
`, rmMarker, rmExit, rmExit, cid, mode)
	path := filepath.Join(t.TempDir(), "stub-engine.sh")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func lifecycleStubMarkerAbsent(t *testing.T, marker string) {
	t.Helper()
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("engine cleanup was invoked unexpectedly; marker %s exists: %v", marker, err)
	}
}

// TestCheckerOCISuccessRunOwnsAndRemovesContainer is the passing control: the
// current successful checker path completes silently with no owned container
// remaining on the daemon.
func TestCheckerOCISuccessRunOwnsAndRemovesContainer(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	work := lifecycleWorkDir(t)
	started := time.Now()
	out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", "pass"}, testRuntimeClosure, 60*time.Second, work)
	if err != nil {
		t.Fatalf("successful pinned checker run failed: %v\n%s", err, out)
	}
	if out != "" {
		t.Fatalf("successful checker run emitted output despite the file-only contract: %q", out)
	}
	if elapsed := time.Since(started); elapsed > 60*time.Second {
		t.Fatalf("successful checker run was not bounded: %s", elapsed)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
}

// TestCheckerOCIMissingIdentityBeforeCreationIsSafe covers cancellation before
// container creation: a timeout that fires first must fail cleanly without
// attempting daemon-side removal of an identity that was never captured.
func TestCheckerOCIMissingIdentityBeforeCreationIsSafe(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	work := lifecycleWorkDir(t)
	started := time.Now()
	out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", "import time; time.sleep(60)"}, testRuntimeClosure, time.Millisecond, work)
	if err == nil {
		t.Fatalf("pre-creation cancellation unexpectedly succeeded: %q", out)
	}
	if elapsed := time.Since(started); elapsed > 30*time.Second {
		t.Fatalf("pre-creation cancellation was not bounded: %s", elapsed)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
}

// TestCheckerOCIAlreadyExitedContainerIsRemoved covers a workload that exits
// on its own before the timeout: the stopped container must still be removed by
// its captured identity.
func TestCheckerOCIAlreadyExitedContainerIsRemoved(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	work := lifecycleWorkDir(t)
	started := time.Now()
	out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", "raise SystemExit(7)"}, testRuntimeClosure, 30*time.Second, work)
	if err == nil || !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf("already-exited workload diagnostic was not propagated: %v\n%s", err, out)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
}

// TestCheckerOCIForeignContainersSurviveCleanup proves cleanup only ever
// removes the exact owned identity: an unrelated concurrent container must be
// untouched while the owned timeout case is reclaimed.
func TestCheckerOCIForeignContainersSurviveCleanup(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	foreign := lifecycleForeignContainer(t)
	work := lifecycleWorkDir(t)
	started := time.Now()
	out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", "import time; time.sleep(120)"}, testRuntimeClosure, 4*time.Second, work)
	if err == nil || !strings.Contains(err.Error(), "timed out after 4s") {
		t.Fatalf("owned timeout run was not reported: %v\n%s", err, out)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
	lifecycleAssertForeignRunning(t, foreign)
}

// TestCheckerOCITimeoutForceRemovesOwnedContainer is the headline regression
// for assessment F4: a timeout kills the engine CLI, and the still-running
// daemon-side container must be force-removed by its captured identity before
// runCheckerOCI reports completion.
func TestCheckerOCITimeoutForceRemovesOwnedContainer(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	work := lifecycleWorkDir(t)
	started := time.Now()
	out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", "import time; time.sleep(120)"}, testRuntimeClosure, 4*time.Second, work)
	if err == nil || !strings.Contains(err.Error(), "timed out after 4s") {
		t.Fatalf("timeout was not reported deterministically: %v\n%s", err, out)
	}
	if id := lifecycleReadContainerID(t, work); id != "" {
		lifecycleAssertHexContainerID(t, id)
	}
	if elapsed := time.Since(started); elapsed > 45*time.Second {
		t.Fatalf("timeout cleanup exceeded the bounded daemon-side grace: %s", elapsed)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
}

// TestCheckerOCITimeoutForceRemovesSIGTERMIgnoringWorkload proves the
// force-removal path survives a workload that ignores SIGTERM: cleanup must not
// depend on cooperative signal handling.
func TestCheckerOCITimeoutForceRemovesSIGTERMIgnoringWorkload(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	work := lifecycleWorkDir(t)
	started := time.Now()
	ignoresSIGTERM := "import signal; signal.signal(signal.SIGTERM, signal.SIG_IGN); import time; time.sleep(300)"
	out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", ignoresSIGTERM}, testRuntimeClosure, 4*time.Second, work)
	if err == nil || !strings.Contains(err.Error(), "timed out after 4s") {
		t.Fatalf("timeout over a SIGTERM-ignoring workload was not reported: %v\n%s", err, out)
	}
	if elapsed := time.Since(started); elapsed > 45*time.Second {
		t.Fatalf("SIGTERM-ignoring workload outlived the bounded cleanup: %s", elapsed)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
}

// TestCheckerOCIBudgetsAreFiniteAndEnforced observes the live container while
// the checker runs: the daemon must apply the finite pinned memory, CPU and PID
// budgets together with the preserved network=none, read-only root, capability
// and no-new-privileges restrictions.
func TestCheckerOCIBudgetsAreFiniteAndEnforced(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	work := lifecycleWorkDir(t)
	started := time.Now()
	type runResult struct {
		out string
		err error
	}
	done := make(chan runResult, 1)
	go func() {
		out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", "import time; time.sleep(90)"}, testRuntimeClosure, 8*time.Second, work)
		done <- runResult{out: out, err: err}
	}()
	id := lifecycleWaitForContainerID(t, work, 30*time.Second)
	hostConfig := "{{.HostConfig.Memory}} {{.HostConfig.NanoCpus}} {{.HostConfig.PidsLimit}} {{.HostConfig.NetworkMode}} {{.HostConfig.ReadonlyRootfs}}"
	out, errOut, err := lifecycleDockerOutput(t, 30*time.Second, "inspect", "--format", hostConfig, id)
	if err != nil || strings.TrimSpace(out) != "134217728 500000000 32 none true" {
		t.Fatalf("live checker container does not carry the finite pinned budgets and sandbox: %v\n%s\n%s", err, out, errOut)
	}
	out, errOut, err = lifecycleDockerOutput(t, 30*time.Second, "inspect", "--format", "{{json .HostConfig.CapDrop}} {{json .HostConfig.SecurityOpt}}", id)
	if err != nil || strings.TrimSpace(out) != `["ALL"] ["no-new-privileges"]` {
		t.Fatalf("live checker container lost capability/no-new-privileges restrictions: %v\n%s\n%s", err, out, errOut)
	}
	select {
	case result := <-done:
		if result.err == nil || !strings.Contains(result.err.Error(), "timed out after 8s") {
			t.Fatalf("budgeted checker run did not end in its bounded timeout: %v\n%s", result.err, result.out)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("budgeted checker run did not return")
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
}

// TestCheckerOCIResourceExhaustionIsBoundedAndCleaned proves the daemon-side
// PID budget actually rejects exhaustion attempts and that the failed run is
// cleaned like any other failure.
func TestCheckerOCIResourceExhaustionIsBoundedAndCleaned(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	work := lifecycleWorkDir(t)
	exhaust := "import subprocess\ntry:\n    for _ in range(64):\n        subprocess.Popen(['sleep', '300'])\nexcept OSError:\n    raise SystemExit(3)\nraise SystemExit(0)"
	started := time.Now()
	out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", exhaust}, testRuntimeClosure, 60*time.Second, work)
	if err == nil {
		t.Fatalf("PID-budget exhaustion unexpectedly succeeded; the daemon-side budget is not finite: %q", out)
	}
	if elapsed := time.Since(started); elapsed > 60*time.Second {
		t.Fatalf("resource exhaustion handling exceeded the run bound: %s", elapsed)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
}

// TestCheckerOCIOutputBreachAbortsAndCleans proves an unbounded-output
// workload is aborted at the capture limit and force-removed, instead of
// streaming until the timeout with the container left on the daemon.
func TestCheckerOCIOutputBreachAbortsAndCleans(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	work := lifecycleWorkDir(t)
	started := time.Now()
	out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"yes", "flood"}, testRuntimeClosure, 45*time.Second, work)
	if err == nil {
		t.Fatalf("output-flooding checker unexpectedly succeeded")
	}
	if !strings.Contains(out, "checker output truncated") {
		t.Fatalf("output breach was not diagnosed as truncation: %q", out)
	}
	if !strings.Contains(err.Error(), "exceeded the bounded output capture limit") {
		t.Fatalf("output breach did not surface the bounded capture diagnostic: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 30*time.Second {
		t.Fatalf("output breach was not aborted promptly: %s", elapsed)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, work)
}

// TestCheckerOCIRepeatedTimeoutsLeaveNoOrphans runs the timeout path three
// times and proves no owned container accumulates on the daemon.
func TestCheckerOCIRepeatedTimeoutsLeaveNoOrphans(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	var works []string
	started := time.Now()
	for range 3 {
		work := lifecycleWorkDir(t)
		works = append(works, work)
		out, err := runCheckerOCI([]string{engine}, lifecycleImage, testRuntimePlatform, []string{"python3", "-c", "import time; time.sleep(120)"}, testRuntimeClosure, 3*time.Second, work)
		if err == nil || !strings.Contains(err.Error(), "timed out after 3s") {
			t.Fatalf("repeated timeout run was not reported: %v\n%s", err, out)
		}
		lifecycleAssertNoOwnedCheckerContainers(t, started, work)
	}
	lifecycleAssertNoOwnedCheckerContainers(t, started, works...)
}

// TestCheckerOCIStubEngineReportsCleanupFailure injects an engine whose rm
// fails: runCheckerOCI must explicitly report the cleanup failure alongside the
// run failure instead of silently losing the container.
func TestCheckerOCIStubEngineReportsCleanupFailure(t *testing.T) {
	stub := lifecycleWriteStubEngine(t, strings.Repeat("a", 64), "hold", filepath.Join(t.TempDir(), "rm-marker"), 1)
	work := lifecycleWorkDir(t)
	out, err := runCheckerOCI([]string{stub}, testRuntimeImage, testRuntimePlatform, []string{"true"}, testRuntimeClosure, 2*time.Second, work)
	if err == nil {
		t.Fatalf("failing cleanup stub run unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "checker container cleanup failed") {
		t.Fatalf("engine cleanup failure was not explicitly reported: %v\n%s", err, out)
	}
}

// TestCheckerOCIStubEngineRefusesInvalidIdentity injects a malformed captured
// identity: cleanup must refuse to remove it and report the invalid identity
// instead of passing an ambiguous ID to the engine.
func TestCheckerOCIStubEngineRefusesInvalidIdentity(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "rm-marker")
	stub := lifecycleWriteStubEngine(t, "not-a-container-id", "hold", marker, 0)
	work := lifecycleWorkDir(t)
	out, err := runCheckerOCI([]string{stub}, testRuntimeImage, testRuntimePlatform, []string{"true"}, testRuntimeClosure, 2*time.Second, work)
	if err == nil {
		t.Fatalf("invalid identity stub run unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "invalid checker container identity") {
		t.Fatalf("invalid captured identity was not reported: %v\n%s", err, out)
	}
	lifecycleStubMarkerAbsent(t, marker)
}

// TestCheckerOCIStubEngineCleanupSucceedsSilently proves a successful run with
// successful cleanup stays silent: the file-only evidence contract must not see
// cleanup noise, and the captured identity must be well-formed.
func TestCheckerOCIStubEngineCleanupSucceedsSilently(t *testing.T) {
	stub := lifecycleWriteStubEngine(t, strings.Repeat("b", 64), "quick", filepath.Join(t.TempDir(), "rm-marker"), 0)
	work := lifecycleWorkDir(t)
	out, err := runCheckerOCI([]string{stub}, testRuntimeImage, testRuntimePlatform, []string{"true"}, testRuntimeClosure, 5*time.Second, work)
	if err != nil {
		t.Fatalf("successful stub run with successful cleanup failed: %v\n%s", err, out)
	}
	if out != "" {
		t.Fatalf("successful stub run was not silent: %q", out)
	}
	lifecycleAssertHexContainerID(t, lifecycleReadContainerID(t, work))
}

// TestCheckerOCICLITimeoutPropagatesAndCleans drives the full
// `machinery verify-checkers` command context with the real daemon: a timing-out
// checker must propagate exit code 1 with the run-failure diagnostic, and no
// owned container may remain afterwards.
func TestCheckerOCICLITimeoutPropagatesAndCleans(t *testing.T) {
	engine := lifecycleRealEngine(t)
	lifecycleRequirePinnedImage(t, engine)
	d := setupVerifyDesign(t)
	registry := writeRawRegistryFile(t, "",
		"checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: ["+engine+"]\n      image: "+lifecycleImage+"\n      platform: "+testRuntimePlatform+"\n    run: [\"python3\", \"-c\", \"import time; time.sleep(120)\"]\n    timeout: \"4s\"\n")
	started := time.Now()
	out, errS, code := runVC(t, d.dir, registry)
	if code != 1 || !strings.Contains(errS, "checker 'test' run failed") || !strings.Contains(errS, "timed out after 4s") {
		t.Fatalf("checker timeout did not propagate through the verify-checkers CLI: code=%d\nstdout=%s\nstderr=%s", code, out, errS)
	}
	if ids := lifecycleLeakedRunningContainerIDs(t, started); len(ids) != 0 {
		t.Fatalf("verify-checkers CLI left running checker container(s) of the pinned image created during the test behind: %v", ids)
	}
}
