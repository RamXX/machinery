//go:build machinery_integration

package main

// Tagged custody supplement for the required lane: real Java provisioning and
// real suite execution must run under verified native processscope custody.
// While a nested pinned JVM (provisioning path) or a suite test process
// (execution path) is independently observed live, its exact PID ancestry
// must pass through a processscope guardian before reaching the lane binary,
// and cancelling the lane must terminate every owned process before the lane
// returns with an honest failed report whose owned-job cleanup still
// verifies. Residual behavior is stated, not hidden: descendants that
// deliberately escape registration and their own process group (a direct
// setsid, or an unscoped engine launch with a fresh background context inside
// foreign test code) are beyond transitive custody and are reported honestly
// as cleanup-failed evidence rather than false success; no process-name or
// PID-pattern matching is ever used as cleanup authority.

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
)

func custodyBuildLane(t *testing.T) (string, string) {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "integration-lane")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binary, "./scripts/integration-lane")
	cmd.Dir = repo
	b, err := cmd.CombinedOutput()
	cancel()
	if err != nil {
		t.Fatalf("lane build failure is not custody evidence: %v %s", err, b)
	}
	return binary, repo
}

func custodyTable(t *testing.T) map[int][2]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "ps", "-eo", "pid=,ppid=,command=").CombinedOutput()
	if err != nil {
		t.Fatalf("process table: %v %s", err, b)
	}
	table := map[int][2]string{}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		table[pid] = [2]string{strconv.Itoa(ppid), strings.Join(fields[2:], " ")}
	}
	return table
}

func atoiOr(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

// custodyAssertGuardedAncestry walks the exact ancestry of pid and requires a
// processscope guardian between it and the lane owner process.
func custodyAssertGuardedAncestry(t *testing.T, pid, lanePID int, context string) {
	t.Helper()
	table := custodyTable(t)
	hasGuardian := false
	reaches := false
	cur := pid
	for i := 0; i < 64 && cur > 1; i++ {
		entry, ok := table[cur]
		if !ok {
			break
		}
		if cur == lanePID {
			reaches = true
			break
		}
		if strings.Contains(entry[1], processscope.InternalMarker+" guardian") {
			hasGuardian = true
		}
		cur = atoiOr(entry[0], 0)
	}
	if !reaches {
		t.Fatalf("%s process %d does not belong to the lane process tree", context, pid)
	}
	if !hasGuardian {
		t.Fatalf("%s process %d is not under transitive custody: no processscope guardian in its ancestry", context, pid)
	}
}

func custodyWaitVanish(t *testing.T, predicate func(command string) bool, timeout time.Duration, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		found := false
		for _, entry := range custodyTable(t) {
			if predicate(entry[1]) {
				found = true
				break
			}
		}
		if !found {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("owned %s survived past lane return", what)
}

func custodyWriteRoot(t *testing.T, repo, root string, runtimes []string, body string) map[string]any {
	t.Helper()
	for _, name := range []string{"schema.json", "runtime-pins.json"} {
		b, e := os.ReadFile(filepath.Join(repo, "testdata/integration-lanes", name))
		if e != nil {
			t.Fatal(e)
		}
		integrationWrite(t, filepath.Join(root, "testdata/integration-lanes", name), string(b))
	}
	javaPin, e := os.ReadFile(filepath.Join(repo, ".java-runtime-pin"))
	if e != nil {
		t.Fatal(e)
	}
	integrationWrite(t, filepath.Join(root, ".java-runtime-pin"), string(javaPin))
	integrationWrite(t, filepath.Join(root, "go.mod"), "module lane.example/custody\n\ngo 1.27.0\n")
	integrationWrite(t, filepath.Join(root, "sample/native.go"), "package sample\n")
	integrationWrite(t, filepath.Join(root, "sample/pilot_integration_test.go"), body)
	suite := map[string]any{"id": "custody", "lane": "required", "adapter": "go-json", "package": "./sample", "source_files": []string{"sample/pilot_integration_test.go"}, "tests": []string{"TestPilot"}, "runtimes": runtimes, "timeout": "5m", "stdout_limit": 1 << 20, "stderr_limit": 1 << 20}
	b, err := json.Marshal(map[string]any{"version": 1, "suites": []any{suite}})
	if err != nil {
		t.Fatal(err)
	}
	integrationWrite(t, filepath.Join(root, "testdata/integration-lanes/pilot.json"), string(b))
	return suite
}

func TestIntegrationLaneCustodyTransitiveTermination(t *testing.T) {
	binary, repo := custodyBuildLane(t)
	t.Run("provisioning", func(t *testing.T) {
		root := t.TempDir()
		cache := filepath.Join(t.TempDir(), "fresh-cache")
		if entries, e := os.ReadDir(cache); !os.IsNotExist(e) && (e != nil || len(entries) != 0) {
			t.Fatalf("formal cache was not empty: %v %v", e, entries)
		}
		work := filepath.Join(t.TempDir(), "owned")
		integrationWrite(t, filepath.Join(work, "user-sentinel"), "caller-owned bytes must survive\n")
		marker := filepath.Join(root, "executed")
		custodyWriteRoot(t, repo, root, []string{"go", "java", "tlc"}, integrationFormalGo(marker, filepath.Join(root, "executed-closure.json")))
		report := filepath.Join(t.TempDir(), "report.json")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "--root", root, "--lane", "required", "--report", report, "--cache-dir", cache, "--work-dir", work)
		var output strings.Builder
		cmd.Stdout = &output
		cmd.Stderr = &output
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		})
		lanePID := cmd.Process.Pid
		// Observe an actually active nested pinned JVM before cancelling. The
		// private cache path scopes the observation to processes this lane
		// provisioned; nothing is terminated by pattern.
		deadline := time.Now().Add(4 * time.Minute)
		jvmSeen := false
		for time.Now().Before(deadline) {
			for _, entry := range custodyTable(t) {
				if strings.Contains(entry[1], cache) && (strings.Contains(entry[1], "tlc2.TLC") || strings.Contains(filepath.Base(entry[1]), "java")) {
					jvmSeen = true
					break
				}
			}
			if jvmSeen {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if !jvmSeen {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			t.Fatalf("no live nested pinned JVM was ever observed under the lane: %s", output.String())
		}
		jvmPID := 0
		for pid, entry := range custodyTable(t) {
			if strings.Contains(entry[1], cache) && strings.Contains(entry[1], "tlc2.TLC") {
				jvmPID = pid
				break
			}
		}
		if jvmPID == 0 {
			for pid, entry := range custodyTable(t) {
				if strings.Contains(entry[1], cache) && strings.Contains(filepath.Base(entry[1]), "java") {
					jvmPID = pid
					break
				}
			}
		}
		custodyAssertGuardedAncestry(t, jvmPID, lanePID, "provisioning JVM")
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Wait(); err == nil {
			t.Fatalf("cancelled provisioning lane reported success: %s", output.String())
		}
		custodyWaitVanish(t, func(command string) bool { return strings.Contains(command, cache) }, 15*time.Second, "provisioned JVM")
		if b, e := os.ReadFile(filepath.Join(work, "user-sentinel")); e != nil || string(b) != "caller-owned bytes must survive\n" {
			t.Fatalf("caller sentinel lost: %v %q", e, b)
		}
		b, e := os.ReadFile(report)
		if e != nil {
			t.Fatalf("report missing: %v %s", e, output.String())
		}
		var raw struct {
			Status  string `json:"status"`
			Custody struct {
				Status string `json:"status"`
			} `json:"custody"`
		}
		if e := json.Unmarshal(b, &raw); e != nil {
			t.Fatal(e)
		}
		if raw.Status != "failed" || raw.Custody.Status != "passed" {
			t.Fatalf("cancelled provisioning must fail honestly with verified owned-job cleanup: %s", b)
		}
	})
	t.Run("suite", func(t *testing.T) {
		root := t.TempDir()
		cache := filepath.Join(t.TempDir(), "cache")
		work := filepath.Join(t.TempDir(), "owned")
		integrationWrite(t, filepath.Join(work, "user-sentinel"), "caller-owned bytes must survive\n")
		marker := filepath.Join(root, "executed")
		pidMarker := filepath.Join(root, "created-pid")
		t.Cleanup(func() { integrationHostCleanup(t, pidMarker) })
		custodyWriteRoot(t, repo, root, []string{"go"}, integrationProcessGo(marker, pidMarker, "process-cancellation"))
		report := filepath.Join(t.TempDir(), "report.json")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "--root", root, "--lane", "required", "--report", report, "--cache-dir", cache, "--work-dir", work)
		var output strings.Builder
		cmd.Stdout = &output
		cmd.Stderr = &output
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		})
		lanePID := cmd.Process.Pid
		deadline := time.Now().Add(45 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(pidMarker); err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		pid, script := integrationReadPID(t, pidMarker)
		if !integrationHostAlive(t, pid, script) {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			t.Fatalf("suite descendant never became live: %s", output.String())
		}
		custodyAssertGuardedAncestry(t, pid, lanePID, "suite descendant")
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Wait(); err == nil {
			t.Fatalf("cancelled suite lane reported success: %s", output.String())
		}
		if integrationHostAlive(t, pid, script) {
			t.Fatal("owned suite descendant survived lane return")
		}
		b, e := os.ReadFile(report)
		if e != nil {
			t.Fatalf("report missing: %v %s", e, output.String())
		}
		var raw struct {
			Status  string `json:"status"`
			Custody struct {
				Status string `json:"status"`
			} `json:"custody"`
		}
		if e := json.Unmarshal(b, &raw); e != nil {
			t.Fatal(e)
		}
		if raw.Status != "failed" || raw.Custody.Status != "passed" {
			t.Fatalf("cancelled suite must fail honestly with verified owned-job cleanup: %s", b)
		}
	})
}
