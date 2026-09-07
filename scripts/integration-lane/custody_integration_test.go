package main

// Supplemental custody contract for the required integration lane: every
// process the lane launches executes under verified native processscope
// custody (registration-before-launch, terminal group kill before reap,
// cumulative wall budget, owner liveness), formal provisioning joins the
// inherited capability so its nested JVMs are guarded jobs of the same root,
// and custody that cannot be established or verified fails closed. Residual
// behavior inherited from processscope, stated honestly: an uncatchable owner
// death or a descendant that deliberately escapes registration and its own
// process group (for example a direct setsid) is reported as cleanup-failed
// evidence, never as false success; no broad process-name or PID matching is
// used as cleanup authority. These black-box tests exercise the real lane
// binary and the in-process run() entry with the Go toolchain only; the
// Java/Docker provisioning paths are covered by the tagged custody suite in
// cmd/machinery/integration_lane_custody_test.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
)

// TestMain gives this test binary its internal activation shape so the
// processscope broker can launch it as a broker/guardian helper when run()
// opens custody in-process.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	io, ok, err := processscope.InheritedInternalIO(ctx)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "machinery: invalid internal channel claim:", err)
		os.Exit(2)
	}
	if ok {
		handled, code := processscope.ServeInternal(os.Args[1:], io)
		if !handled {
			fmt.Fprintln(os.Stderr, "machinery: internal protocol failure")
			os.Exit(2)
		}
		os.Exit(code)
	}
	os.Exit(m.Run())
}

var (
	custodyBinaryOnce sync.Once
	custodyBinaryPath string
	custodyBinaryErr  error
)

// custodyLaneBinary builds the real lane CLI once for subprocess assertions.
func custodyLaneBinary(t *testing.T) string {
	t.Helper()
	custodyBinaryOnce.Do(func() {
		bin := filepath.Join(t.TempDir(), "integration-lane")
		// The temp dir belongs to the first requesting test; keep the binary
		// in a shared location that outlives individual tests.
		dir, err := os.MkdirTemp("", "machinery-custody-binary-")
		if err != nil {
			custodyBinaryErr = err
			return
		}
		bin = filepath.Join(dir, "integration-lane")
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./scripts/integration-lane")
		cmd.Dir = laneRepo(t)
		b, err := cmd.CombinedOutput()
		if err != nil {
			custodyBinaryErr = fmt.Errorf("lane build: %v %s", err, b)
			return
		}
		custodyBinaryPath = bin
	})
	if custodyBinaryErr != nil {
		t.Fatal(custodyBinaryErr)
	}
	return custodyBinaryPath
}

type custodyProc struct {
	pid     int
	ppid    int
	command string
}

// custodyProcessTable captures the exact native process table without any
// pattern-based cleanup; it is observation-only evidence.
func custodyProcessTable(t *testing.T) map[int]custodyProc {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "ps", "-eo", "pid=,ppid=,command=").CombinedOutput()
	if err != nil {
		t.Fatalf("process table: %v %s", err, b)
	}
	table := map[int]custodyProc{}
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
		table[pid] = custodyProc{pid: pid, ppid: ppid, command: strings.Join(fields[2:], " ")}
	}
	return table
}

// custodyAncestors walks the exact PID ancestry and reports whether the chain
// from start reaches the owner PID and contains a custody guardian process.
func custodyAncestors(t *testing.T, table map[int]custodyProc, start, owner int) (reachesOwner, hasGuardian bool) {
	t.Helper()
	pid := start
	for i := 0; i < 64 && pid > 1; i++ {
		proc, ok := table[pid]
		if !ok {
			break
		}
		if pid == owner {
			reachesOwner = true
		}
		if strings.Contains(proc.command, processscope.InternalMarker+" guardian") {
			hasGuardian = true
		}
		if pid == owner {
			break
		}
		pid = proc.ppid
	}
	return reachesOwner, hasGuardian
}

func custodyWaitForFile(t *testing.T, path string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
			return string(b)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("file %s never appeared", path)
	return ""
}

func custodyWaitGone(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, ok := custodyProcessTable(t)[pid]; !ok {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("owned process %d survived past lane return", pid)
}

// The lane binary must authenticate internal activation before any ordinary
// execution: a malformed or forged internal channel claim is refused instead
// of silently running normal lane behavior, and an unknown internal role is
// rejected by the authenticated dispatcher.
func TestLaneCustodyActivationIsAuthenticatedAndExclusive(t *testing.T) {
	bin := custodyLaneBinary(t)
	for _, tc := range []struct {
		name string
		args []string
		env  []string
		want string
	}{
		{
			name: "unknown-internal-role",
			args: []string{processscope.InternalMarker, "not-a-role", "--dir", t.TempDir()},
			want: "machinery:",
		},
		{
			name: "internal-claim-without-marker",
			args: []string{"--lane", "required"},
			env:  []string{"MACHINERY_INTERNAL_CTL=9"},
			want: "internal activation",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, tc.args...)
			cmd.Env = append(os.Environ(), tc.env...)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("untrusted internal activation was accepted: %s", out)
			}
			if !strings.Contains(string(out), tc.want) {
				t.Fatalf("refusal diagnostic missing %q: %s", tc.want, out)
			}
		})
	}
}

// A present-but-malformed inherited custody capability must fail the lane
// closed; it may never be silently discarded in favor of an unscoped run.
func TestLaneRefusesMalformedInheritedCapability(t *testing.T) {
	bin := custodyLaneBinary(t)
	root, _ := laneFixture(t, `func TestPilot(t *testing.T) { if 6*7 != 42 { t.Fatal("wrong") } }`)
	work := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--root", root, "--lane", "required", "--report", filepath.Join(work, "report.json"), "--work-dir", filepath.Join(work, "owned"), "--cache-dir", filepath.Join(work, "cache"))
	cmd.Env = append(os.Environ(), processscope.EnvCapability+"=forged-not-a-capability")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("malformed inherited custody was ignored and the lane passed: %s", out)
	}
	if !strings.Contains(strings.ToLower(string(out)), "custody") {
		t.Fatalf("failure was not the intended custody refusal: %s", out)
	}
}

// A successful run must bind verifiable custody evidence into the report:
// root mode, a nonzero guarded job count, and every job registered,
// terminated and reaped.
func TestLaneCustodyReportBindsVerifiedCleanup(t *testing.T) {
	root, _ := laneFixture(t, `func TestPilot(t *testing.T) { if 6*7 != 42 { t.Fatal("wrong") } }`)
	work := t.TempDir()
	report := filepath.Join(work, "report.json")
	args := []string{"--root", root, "--lane", "required", "--report", report, "--work-dir", filepath.Join(work, "owned"), "--cache-dir", filepath.Join(work, "cache")}
	var out, errout strings.Builder
	if status := run(args, &out, &errout); status != 0 {
		t.Fatalf("custody must not weaken a passing lane: %d %s %s", status, out.String(), errout.String())
	}
	var raw struct {
		Status  string `json:"status"`
		Custody struct {
			Status string `json:"status"`
			Mode   string `json:"mode"`
			Jobs   int    `json:"jobs"`
		} `json:"custody"`
	}
	b, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("report missing: %v", err)
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Status != "passed" || raw.Custody.Status != "passed" || raw.Custody.Mode != "root" || raw.Custody.Jobs < 1 {
		t.Fatalf("report lacks verified native custody evidence: %+v report=%s", raw.Custody, b)
	}
}

// Every suite execution must be a guarded custody job: while a suite test
// body is live, its exact PID ancestry passes through a processscope
// guardian process before reaching the lane. Cancelling the lane terminates
// the suite process before the lane returns while custody cleanup still
// verifies.
func TestLaneSuitesExecuteAsGuardedCustodyJobs(t *testing.T) {
	bin := custodyLaneBinary(t)
	root, _ := laneFixture(t, `func TestPilot(t *testing.T) {
  if e := os.WriteFile("suite-live", []byte("live"), 0600); e != nil { t.Fatal(e) }
  time.Sleep(90 * time.Second)
}`)
	work := t.TempDir()
	owned := filepath.Join(work, "owned")
	if err := os.MkdirAll(owned, 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(owned, "user-sentinel")
	if err := os.WriteFile(sentinel, []byte("caller-owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(work, "report.json")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--root", root, "--lane", "required", "--report", report, "--work-dir", owned, "--cache-dir", filepath.Join(work, "cache"))
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	lanePID := cmd.Process.Pid
	live := filepath.Join(root, "sample", "suite-live")
	first := custodyWaitForFile(t, live, 45*time.Second)
	if first != "live" {
		t.Fatalf("suite never became live: %q", first)
	}
	// The fixture writes from inside the test binary; identify it through the
	// process table by walking ancestry from every candidate that reaches the
	// lane binary, and require the running suite job to be guardian-guarded.
	table := custodyProcessTable(t)
	suitePID := 0
	for pid, proc := range table {
		if !strings.Contains(proc.command, "test") || strings.Contains(proc.command, "integration-lane") {
			continue
		}
		reaches, _ := custodyAncestors(t, table, pid, lanePID)
		if reaches {
			suitePID = pid
			break
		}
	}
	if suitePID == 0 {
		t.Fatalf("no live suite process under the lane was found: %v", table)
	}
	_, hasGuardian := custodyAncestors(t, table, suitePID, lanePID)
	if !hasGuardian {
		t.Fatalf("suite process %d is not a guarded custody job; ancestry lacks a processscope guardian", suitePID)
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	err := cmd.Wait()
	if err == nil {
		t.Fatalf("cancelled lane reported success: %s", output.String())
	}
	custodyWaitGone(t, suitePID, 15*time.Second)
	if b, err := os.ReadFile(sentinel); err != nil || string(b) != "caller-owned" {
		t.Fatalf("caller sentinel lost: %v %q", err, b)
	}
	entries, err := os.ReadDir(owned)
	if err != nil || len(entries) != 1 || entries[0].Name() != "user-sentinel" {
		t.Fatalf("owned work residue: %v %v", err, entries)
	}
	b, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("report missing after cancellation: %v %s", err, output.String())
	}
	var raw struct {
		Status  string `json:"status"`
		Cleanup struct {
			Status string `json:"status"`
		} `json:"cleanup"`
		Custody struct {
			Status string `json:"status"`
		} `json:"custody"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Status != "failed" || raw.Custody.Status != "passed" {
		t.Fatalf("cancelled lane must fail honestly with verified owned-job cleanup: %+v", raw)
	}
}

// The formal provisioning helper must open its own verified custody root
// before doing any work: it runs only as a guarded job of the lane, and a
// direct invocation that cannot establish custody fails closed instead of
// provisioning unscoped.
func TestLaneProvisionFormalRequiresCustodyCapability(t *testing.T) {
	bin := custodyLaneBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	notADir := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(notADir, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, bin, "provision-formal", notADir)
	cmd.Env = append(os.Environ(),
		"HOME="+t.TempDir(),
		"HTTPS_PROXY=http://127.0.0.1:1",
		"HTTP_PROXY=http://127.0.0.1:1",
		"NO_PROXY=",
		processscope.EnvCapability+"=",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("helper without verifiable custody succeeded: %s", out)
	}
	if !strings.Contains(strings.ToLower(string(out)), "custody") {
		t.Fatalf("helper failure was not the custody fail-closed outcome: %s", out)
	}
}
