//go:build machinery_integration

package main

// Required-lane Docker bind-mount recovery cases for `machinery recover`
// (MAC-2n83 AC5). A real container from the lane's pinned immutable image
// performs a real designlock publication against a bind-mounted host design
// root and dies by real process death. The host then inspects and recovers
// across the process/host identity boundary:
//
//   - an already-complete cross-identity publication is finalized by
//     `machinery recover --apply` after full revalidation, and
//   - a partial cross-identity publication is NEVER rolled back: the refusal
//     preserves the journal and every user byte and explains the conflict.
//
// Container ownership follows the lane contract: cidfile under the suite work
// root's containers/ directory plus the exact per-run label; containers are
// reclaimed individually after label verification; no broad Docker cleanup.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/designlock"
)

const recoveryDockerDesignDir = "design"

func recoveryDockerExec(t *testing.T, timeout time.Duration, argv ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real process %s failed: %v output=%s", argv[0], err, out)
	}
	return string(out)
}

// recoveryDockerOwnContainer runs one bounded offline container from the
// pinned image with the lane's ownership witnesses and reclaims exactly it.
func recoveryDockerOwnContainer(t *testing.T, bindHost, boundary string) {
	t.Helper()
	work := os.Getenv("MACHINERY_INTEGRATION_WORK")
	if work == "" {
		work = t.TempDir()
	}
	runID := os.Getenv("MACHINERY_INTEGRATION_RUN_ID")
	if runID == "" {
		runID = fmt.Sprintf("machinery-recovery-%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	if err := os.MkdirAll(filepath.Join(work, "containers"), 0o700); err != nil {
		t.Fatal(err)
	}
	cidfile := filepath.Join(work, "containers", fmt.Sprintf("recovery-%d.cid", time.Now().UnixNano()))
	t.Cleanup(func() {
		cid, readErr := os.ReadFile(cidfile)
		if os.IsNotExist(readErr) {
			return
		}
		if readErr != nil {
			t.Errorf("read owned recovery cid: %v", readErr)
			return
		}
		id := strings.TrimSpace(string(cid))
		if len(id) != 64 {
			t.Errorf("refuse recovery cleanup for malformed cid %q", id)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		b, err := exec.CommandContext(ctx, "docker", "inspect", "--format", `{{index .Config.Labels "dev.machinery.integration-run"}}`, id).CombinedOutput()
		if err != nil && strings.Contains(strings.ToLower(string(b)), "no such object") {
			return
		}
		if err != nil || strings.TrimSpace(string(b)) != runID {
			t.Errorf("refuse recovery cleanup: ownership of %s changed: %v %s", id, err, b)
			return
		}
		b, err = exec.CommandContext(ctx, "docker", "rm", "-f", id).CombinedOutput()
		if err != nil {
			t.Errorf("owned recovery container cleanup %s failed: %v %s", id, err, b)
		}
	})
	shared, err := filepath.EvalSymlinks(bindHost)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "run",
		"--pull=never",
		"--platform", integrationPilotPlatform,
		"--network=none",
		"--memory=128m",
		"--cpus=0.5",
		"--pids-limit=32",
		"--label", "dev.machinery.integration-run="+runID,
		"--cidfile", cidfile,
		"--volume", shared+":/work",
		"--workdir", "/work",
		// The writer runs as the host user: on a Linux host a root container
		// leaves root-owned residue in the bind mount that the test process can
		// neither inspect nor remove (macOS Docker Desktop maps ownership, which
		// hid this). The publication under test is about interruption, not
		// privilege, so the fixture needs no root.
		"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		"--env", "MACHINERY_RECOVERY_CRASH_DESIGN=/work/"+recoveryDockerDesignDir,
		"--env", "MACHINERY_RECOVERY_CRASH_BOUNDARY="+boundary,
		integrationPilotImage,
		"/work/recovery-helper.test", "-test.run=^TestRecoveryCrashPublishHelper$", "-test.timeout=90s",
	)
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err == nil {
		t.Fatalf("container publication unexpectedly succeeded: %s", output.String())
	} else {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 9 {
			t.Fatalf("container writer did not die by process death: %v output=%s", err, output.String())
		}
	}
	cid, err := os.ReadFile(cidfile)
	if err != nil {
		t.Fatalf("container cidfile missing: %v %s", err, output.String())
	}
	id := strings.TrimSpace(string(cid))
	if len(id) != 64 {
		t.Fatalf("noncanonical owned cid %q", id)
	}
	if running := strings.TrimSpace(recoveryDockerExec(t, 15*time.Second, "docker", "inspect", "--format", "{{.State.Running}}", id)); running == "true" {
		t.Fatalf("container writer %s is still live; refusing host-side recovery evidence", id)
	}
	if code := strings.TrimSpace(recoveryDockerExec(t, 15*time.Second, "docker", "inspect", "--format", "{{.State.ExitCode}}", id)); code != "9" {
		t.Fatalf("container writer exit code %s, want 9", code)
	}
}

// recoveryDockerBuildHelper cross-compiles the internal/designlock test
// binary (which carries TestRecoveryCrashPublishHelper) for the container's
// pinned platform. It is test-owned output in a private temp directory.
func recoveryDockerBuildHelper(t *testing.T, dest string) {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-c", "-o", dest, "./internal/designlock")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cross-compile recovery helper failed: %v %s", err, out)
	}
	info, err := os.Stat(dest)
	if err != nil || info.Size() < 1<<20 {
		t.Fatalf("recovery helper binary missing implausibly small: %v %v", err, info)
	}
}

func recoveryDockerDesign(t *testing.T, shared string) string {
	t.Helper()
	design := filepath.Join(shared, recoveryDockerDesignDir)
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "source.md"), []byte("design inputs must survive every recovery path\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "stale.txt"), []byte("obsolete output\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "generated.txt"), []byte("obsolete output\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return design
}

// TestRecoveryDockerBindMountPublication proves the already-complete
// cross-identity case: the container writer's lock identity is unreachable
// from the host, the writer is proven dead, and the host finalizes the
// content-and-mode complete publication only after full revalidation.
func TestRecoveryDockerBindMountPublication(t *testing.T) {
	shared := t.TempDir()
	recoveryDockerBuildHelper(t, filepath.Join(shared, "recovery-helper.test"))
	recoveryDockerDesign(t, shared)
	hostShared, err := filepath.EvalSymlinks(shared)
	if err != nil {
		t.Fatal(err)
	}
	hostDesign := filepath.Join(hostShared, recoveryDockerDesignDir)
	recoveryDockerOwnContainer(t, shared, "pre-clear")
	out, errB, codes := runRecoverCommand(t, hostDesign)
	if len(codes) != 0 || errB != "" {
		t.Fatalf("host inspection failed: codes=%v stderr=%q stdout=\n%s", codes, errB, out)
	}
	for _, want := range []string{
		"writer: recover-fixture",
		"live writer: no",
		"input inventory: verified",
		"action: finalize",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("host inspection report missing %q:\n%s", want, out)
		}
	}
	out, errB, codes = runRecoverCommand(t, hostDesign, "--apply")
	if len(codes) != 0 || errB != "" {
		t.Fatalf("host apply failed: codes=%v stderr=%q stdout=\n%s", codes, errB, out)
	}
	if _, err := os.Stat(filepath.Join(hostDesign, ".machinery-design-publish.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cross-identity publication sentinel survived host apply: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(hostDesign, "generated.txt"))
	if err != nil || string(body) != "generated by the interrupted writer\n" {
		t.Fatalf("cross-identity recovered output wrong: %v %q", err, body)
	}
	if _, err := os.Stat(filepath.Join(hostDesign, "stale.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("declared-absent output survived cross-identity apply: %v", err)
	}
	reader, err := designlock.AcquireReader(hostDesign)
	if err != nil {
		t.Fatalf("reader remained blocked after cross-identity apply: %v", err)
	}
	if err := reader.Release(); err != nil {
		t.Fatal(err)
	}
	out, errB, codes = runRecoverCommand(t, hostDesign, "--apply")
	if len(codes) != 0 || errB != "" || !strings.Contains(out, "no interrupted publication") {
		t.Fatalf("idempotent second host apply not harmless: codes=%v stderr=%q stdout=\n%s", codes, errB, out)
	}
}

// TestRecoveryDockerPartialRefusesRollback proves the partial cross-identity
// case is distinguished from completion: the host refuses, rolls nothing
// back, preserves the journal and every user byte, and never performs broad
// sentinel or directory deletion as remediation.
func TestRecoveryDockerPartialRefusesRollback(t *testing.T) {
	shared := t.TempDir()
	recoveryDockerBuildHelper(t, filepath.Join(shared, "recovery-helper.test"))
	recoveryDockerDesign(t, shared)
	hostShared, err := filepath.EvalSymlinks(shared)
	if err != nil {
		t.Fatal(err)
	}
	hostDesign := filepath.Join(hostShared, recoveryDockerDesignDir)
	recoveryDockerOwnContainer(t, shared, "mid-callback")
	before := recoverTreeFingerprint(t, hostDesign)
	out, errB, codes := runRecoverCommand(t, hostDesign)
	if len(codes) != 0 || errB != "" {
		t.Fatalf("host inspection failed: codes=%v stderr=%q stdout=\n%s", codes, errB, out)
	}
	if !strings.Contains(out, "action: rerun-writer") {
		t.Fatalf("partial cross-identity publication not classified for writer rerun:\n%s", out)
	}
	out, errB, codes = runRecoverCommand(t, hostDesign, "--apply")
	if len(codes) == 0 || codes[0] != 1 {
		t.Fatalf("partial cross-identity apply must refuse: codes=%v stdout=\n%s", codes, out)
	}
	if !strings.Contains(errB, "nothing was modified") {
		t.Fatalf("refusal must state preservation: %q", errB)
	}
	after := recoverTreeFingerprint(t, hostDesign)
	if len(before) != len(after) {
		t.Fatalf("refusal changed the design inventory: %d -> %d entries", len(before), len(after))
	}
	for key, value := range before {
		if after[key] != value {
			t.Fatalf("refusal mutated %s: %s -> %s", key, value, after[key])
		}
	}
	if body, err := os.ReadFile(filepath.Join(hostDesign, "generated.txt")); err != nil || string(body) != "generated by the interrupted writer\n" {
		t.Fatalf("partial cross-identity output was rolled back or damaged: %v %q", err, body)
	}
	if _, err := os.Stat(filepath.Join(hostDesign, ".machinery-design-publish.json")); err != nil {
		t.Fatalf("refusal deleted the cross-identity publication journal: %v", err)
	}
	if reader, err := designlock.AcquireReader(hostDesign); err == nil {
		_ = reader.Release()
		t.Fatal("partial cross-identity residue stopped blocking readers without recovery")
	}
}
