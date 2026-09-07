//go:build unix

package formal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/processcontrol"
	"github.com/RamXX/machinery/internal/processscope"
)

// TestMain gives this test binary its internal activation shape so the
// processscope broker can launch it as a broker/guardian helper.
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

// --- shared custody helpers ---

func custodyTestDigest(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func openFormalCustodyScope(t *testing.T) processscope.Scope {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	s, err := processscope.Open(ctx, processscope.Options{
		HelperExecutable: exe,
		HelperDigest:     custodyTestDigest(t, exe),
		ScratchRoot:      t.TempDir(),
		Limits:           processscope.Limits{Jobs: 4, WallMS: 2400000, CleanupMS: 30000},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func closeFormalCustodyScope(t *testing.T, s processscope.Scope, timeout time.Duration) processscope.CleanupReport {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rep, err := s.Close(ctx)
	if err != nil {
		t.Fatalf("close formal custody scope: %v", err)
	}
	return rep
}

func requireFormalCustodyClean(t *testing.T, rep processscope.CleanupReport, wantJobs int) {
	t.Helper()
	if rep.Status != processscope.StatusCleaned {
		t.Fatalf("cleanup status %s: %+v", rep.Status, rep)
	}
	if len(rep.Jobs) != wantJobs {
		t.Fatalf("expected %d jobs in cleanup report, got %d: %+v", wantJobs, len(rep.Jobs), rep.Jobs)
	}
	for _, j := range rep.Jobs {
		if !j.Registered || !j.Terminated || !j.Reaped {
			t.Fatalf("job state incomplete: %+v", j)
		}
	}
}

func formalCustodySentinel(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("/bin/sleep", "300")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	return cmd.Process.Pid
}

func custodyAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// waitFormalJavaProcess polls pgrep for a live JVM matching pattern and
// returns its pid. Observation only: ownership proof stays with the broker's
// cleanup report.
func waitFormalJavaProcess(t *testing.T, pattern string, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("/usr/bin/pgrep", "-f", pattern).Output()
		if err == nil {
			for _, field := range strings.Fields(string(out)) {
				pid, err := strconv.Atoi(field)
				if err != nil || pid == os.Getpid() {
					continue
				}
				if custodyAlive(pid) {
					return pid
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("no live JVM matched %q within %v", pattern, timeout)
	return 0
}

func waitFormalJavaProcessGone(t *testing.T, pattern string, pid int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, _ := exec.Command("/usr/bin/pgrep", "-f", pattern).Output()
		gone := !custodyAlive(pid)
		if !strings.Contains(string(out), strconv.Itoa(pid)) && gone {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("JVM pid %d matching %q still alive after %v", pid, pattern, timeout)
}

// useRealFormalTools points the formal chain at the provisioned pinned Java
// runtime and the checksum-pinned engine jars in the user cache.
func useRealFormalTools(t *testing.T) {
	t.Helper()
	t.Setenv("MACHINERY_JAVA", "")
	t.Setenv("MACHINERY_JAVA_CLOSURE_SHA256", "")
	t.Setenv("TLA_TOOLS_JAR", "")
	t.Setenv("TLA_TOOLS_JAR_SHA256", "")
	t.Setenv("ALLOY_TOOLS_JAR", "")
	t.Setenv("ALLOY_TOOLS_JAR_SHA256", "")
}

// writeManualLongTLCPair writes a design whose single manual TLA pair
// model-checks a deliberately long-running state space (401^3 states).
func writeManualLongTLCPair(t *testing.T, design string) {
	t.Helper()
	for _, dir := range []string{filepath.Join(design, "formal")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	tla := "\\* machinery:manual\n" +
		"---- MODULE LongRun ----\n" +
		"EXTENDS Naturals\n" +
		"VARIABLES x, y, z\n" +
		"Init == x = 0 /\\ y = 0 /\\ z = 0\n" +
		"Next == \\/ /\\ x' = (x + 1) % 401\n" +
		"           /\\ y' = y /\\ z' = z\n" +
		"        \\/ /\\ y' = (y + 1) % 401\n" +
		"           /\\ x' = x /\\ z' = z\n" +
		"        \\/ /\\ z' = (z + 1) % 401\n" +
		"           /\\ x' = x /\\ y' = y\n" +
		"Spec == Init /\\ [][Next]_<<x, y, z>>\n" +
		"Inv == TRUE\n" +
		"====\n"
	if err := os.WriteFile(filepath.Join(design, "formal", "LongRun.tla"), []byte(tla), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := "SPECIFICATION Spec\nINVARIANT Inv\n"
	if err := os.WriteFile(filepath.Join(design, "formal", "LongRun.cfg"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- tests ---

func TestVerifyFormalInScopeRequiresCustodyScope(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	rc := VerifyFormalInScope(context.Background(), design, false, &stdout, &stderr)
	if rc != 1 {
		t.Fatalf("full verification without a custody scope must fail closed, got exit %d", rc)
	}
	if !strings.Contains(stderr.String(), "custody") {
		t.Fatalf("fail-closed diagnostic must name custody: %q", stderr.String())
	}
}

func TestVerifyFormalInScopeGenOnlyControlWithoutScope(t *testing.T) {
	design := t.TempDir()
	machines := filepath.Join(design, "machines")
	if err := os.Mkdir(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	machine := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(machines, "Toy.machine.json"), []byte(machine), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	rc := VerifyFormalInScope(context.Background(), design, true, &stdout, &stderr)
	if rc != 0 {
		t.Fatalf("gen-only mode launches no engine and needs no scope: exit %d stderr %q", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "TLC skipped (--gen-only)") {
		t.Fatalf("gen-only control output: %q", stdout.String())
	}
}

// TestVerifyFormalInScopeScopedEngineRunsUnderGuardianCustody is the
// immutable unsafe-implementation challenge for the scope fail-closed safe
// default: a scoped engine launch must actually run under a broker guardian
// with the sanitized declared environment.
func TestVerifyFormalInScopeScopedEngineRunsUnderGuardianCustody(t *testing.T) {
	s := openFormalCustodyScope(t)
	defer closeFormalCustodyScope(t, s, 30*time.Second)
	design := t.TempDir()
	machines := filepath.Join(design, "machines")
	if err := os.Mkdir(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	machine := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(machines, "Toy.machine.json"), []byte(machine), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	jar := filepath.Join(dir, "tla2tools.jar")
	if err := os.WriteFile(jar, []byte("verified"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, err := fileSHA256(jar)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLA_TOOLS_JAR", jar)
	t.Setenv("TLA_TOOLS_JAR_SHA256", sha)
	envFile := filepath.Join(dir, "engine-env.txt")
	ppidFile := filepath.Join(dir, "engine-ppid.txt")
	engine := "env > \"" + envFile + "\"; echo $PPID > \"" + ppidFile + "\"; echo 'No error has been found'; exit 0\n"
	javaPath := filepath.Join(dir, "runtime", "bin", "java")
	writeJavaRuntime(t, javaPath, supportedJavaScript(engine))
	t.Setenv("JAVA_TOOL_OPTIONS", "-javaagent:/hostile.jar")

	var stdout, stderr bytes.Buffer
	rc := VerifyFormalInScope(processcontrol.WithScope(context.Background(), s), design, false, &stdout, &stderr)
	if rc != 0 {
		t.Fatalf("scoped fake-engine verification must pass: exit %d stdout %q stderr %q", rc, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS  Toy") {
		t.Fatalf("scoped run output: %q", stdout.String())
	}
	env := ""
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(envFile); err == nil && len(b) > 0 {
			env = string(b)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if env == "" {
		t.Fatal("engine never observed its environment")
	}
	if strings.Contains(env, "hostile") {
		t.Fatalf("hostile ambient environment reached the scoped TLC engine:\n%s", env)
	}
	ppidText := ""
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(ppidFile); err == nil && len(b) > 0 {
			ppidText = strings.TrimSpace(string(b))
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	ppid, err := strconv.Atoi(ppidText)
	if err != nil {
		t.Fatalf("engine guardian ppid %q: %v", ppidText, err)
	}
	if ppid == 0 || ppid == os.Getpid() || ppid == os.Getppid() {
		t.Fatalf("scoped engine must run under a distinct guardian parent, saw %d (self %d, parent %d)", ppid, os.Getpid(), os.Getppid())
	}
	// One identity probe + one engine ran as owned scoped jobs.
	requireFormalCustodyClean(t, closeFormalCustodyScope(t, s, 30*time.Second), 2)
}

func TestVerifyFormalInScopePortfolioDesignUnderCustody(t *testing.T) {
	sentinel := formalCustodySentinel(t)
	useRealFormalTools(t)
	s := openFormalCustodyScope(t)
	design := filepath.Join(t.TempDir(), "design")
	copyTree(t, filepath.Join("..", "..", "examples", "portfolio-engine", "design"), design)
	var stdout, stderr bytes.Buffer
	rc := VerifyFormalInScope(processcontrol.WithScope(context.Background(), s), design, false, &stdout, &stderr)
	if rc != 0 {
		t.Fatalf("portfolio verification under custody must pass: exit %d\nstdout:\n%s\nstderr:\n%s", rc, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "7 passed, 0 failed") {
		t.Fatalf("portfolio pass summary missing: %q", stdout.String())
	}
	// 7 TLC engines + 7 identity probes ran as owned scoped jobs.
	requireFormalCustodyClean(t, closeFormalCustodyScope(t, s, 60*time.Second), 14)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestVerifyFormalInScopeFulfillmentDesignRunsTLCAndAlloyUnderCustody(t *testing.T) {
	sentinel := formalCustodySentinel(t)
	useRealFormalTools(t)
	s := openFormalCustodyScope(t)
	design := filepath.Join(t.TempDir(), "design")
	copyTree(t, filepath.Join("..", "..", "examples", "fulfillment", "design"), design)
	var stdout, stderr bytes.Buffer
	rc := VerifyFormalInScope(processcontrol.WithScope(context.Background(), s), design, false, &stdout, &stderr)
	if rc != 0 {
		t.Fatalf("fulfillment verification under custody must pass: exit %d\nstdout:\n%s\nstderr:\n%s", rc, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS  Integrity/") {
		t.Fatalf("Alloy integrity verdicts missing from scoped run: %q", stdout.String())
	}
	// 8 TLC engines + 8 probes + 1 Alloy engine + 1 probe.
	requireFormalCustodyClean(t, closeFormalCustodyScope(t, s, 60*time.Second), 18)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestScopedFormalTLCCancellationReapsOwnedJVM(t *testing.T) {
	sentinel := formalCustodySentinel(t)
	useRealFormalTools(t)
	s := openFormalCustodyScope(t)
	design := t.TempDir()
	writeManualLongTLCPair(t, design)

	ctx, cancel := context.WithCancel(processcontrol.WithScope(context.Background(), s))
	defer cancel() // a failed observation must never strand the engine
	type outcome struct {
		rc     int
		stdout string
		stderr string
	}
	done := make(chan outcome, 1)
	go func() {
		var stdout, stderr bytes.Buffer
		rc := VerifyFormalInScope(ctx, design, false, &stdout, &stderr)
		done <- outcome{rc: rc, stdout: stdout.String(), stderr: stderr.String()}
	}()

	// Observe the actually running pinned TLC JVM before cancelling.
	pid := waitFormalJavaProcess(t, `tlc2\.TLC .*machinery-tlc-`, 60*time.Second)
	if !custodyAlive(pid) {
		t.Fatalf("observed TLC JVM %d is not alive", pid)
	}
	time.Sleep(2 * time.Second)
	if !custodyAlive(pid) {
		t.Fatal("TLC JVM died before cancellation")
	}
	cancel()
	var res outcome
	select {
	case res = <-done:
	case <-time.After(120 * time.Second):
		t.Fatal("canceled verification did not return")
	}
	if res.rc != 1 {
		t.Fatalf("canceled verification must fail, got exit %d", res.rc)
	}
	if !strings.Contains(res.stdout, "FAIL  LongRun") {
		t.Fatalf("canceled pair must be reported as failed: %q", res.stdout)
	}
	// The owned JVM is terminated and reaped before return; the unrelated
	// sentinel survives.
	waitFormalJavaProcessGone(t, `tlc2\.TLC .*machinery-tlc-`, pid, 30*time.Second)
	requireFormalCustodyClean(t, closeFormalCustodyScope(t, s, 60*time.Second), 2)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

// writeHeavyAlloyModel writes an .als whose single check is a Ramsey
// unsatisfiability proof that runs far beyond the cancellation point.
func writeHeavyAlloyModel(t *testing.T, dir string) string {
	t.Helper()
	body := "sig Node { adj: set Node }\n" +
		"fact Graph {\n" +
		"  all n: Node | not n in n.adj\n" +
		"  all disj a, b: Node | a in b.adj iff b in a.adj\n" +
		"}\n" +
		"check Ramsey {\n" +
		"  some disj a, b, c: Node | a in b.adj and b in c.adj and c in a.adj\n" +
		"  or some disj v1, v2, v3, v4, v5, v6, v7, v8, v9: Node |\n" +
		"    let S = v1 + v2 + v3 + v4 + v5 + v6 + v7 + v8 + v9 |\n" +
		"      no disj p, q: S | p in q.adj\n" +
		"} for exactly 36 Node\n"
	als := filepath.Join(dir, "Heavy.als")
	if err := os.WriteFile(als, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return als
}

func TestScopedAlloyEngineCancellationReapsOwnedJVM(t *testing.T) {
	sentinel := formalCustodySentinel(t)
	useRealFormalTools(t)
	s := openFormalCustodyScope(t)
	als := writeHeavyAlloyModel(t, t.TempDir())

	ctx, cancel := context.WithCancel(processcontrol.WithScope(context.Background(), s))
	defer cancel() // a failed observation must never strand the engine
	type outcome struct {
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		_, _, err := runAlloyScoped(ctx, als, []alloy.Command{{Kind: "check", Name: "Ramsey"}})
		done <- outcome{err: err}
	}()

	pid := waitFormalJavaProcess(t, `machinery-alloy-tool-.*verified-tool\.jar`, 60*time.Second)
	if !custodyAlive(pid) {
		t.Fatalf("observed Alloy JVM %d is not alive", pid)
	}
	time.Sleep(2 * time.Second)
	if !custodyAlive(pid) {
		t.Fatal("Alloy JVM died before cancellation")
	}
	cancel()
	select {
	case res := <-done:
		if res.err == nil {
			t.Fatal("canceled Alloy engine must report an error")
		}
	case <-time.After(120 * time.Second):
		t.Fatal("canceled Alloy engine did not return")
	}
	waitFormalJavaProcessGone(t, `machinery-alloy-tool-.*verified-tool\.jar`, pid, 30*time.Second)
	// One identity probe + one Alloy engine ran as owned scoped jobs.
	requireFormalCustodyClean(t, closeFormalCustodyScope(t, s, 60*time.Second), 2)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}
