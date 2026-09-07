//go:build unix

package processcontrol

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
)

// TestMain gives this test binary its internal activation shape: when the
// processscope broker launches it as a guardian, the internal marker is
// served before any test runs.
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

// TestCustodyScopeHelper is the in-binary helper target for scoped-run tests.
// It is a no-op unless MACHINERY_CUSTODY_ROLE selects a role.
func TestCustodyScopeHelper(t *testing.T) {
	role := os.Getenv("MACHINERY_CUSTODY_ROLE")
	if role == "" {
		return
	}
	file := os.Getenv("MACHINERY_CUSTODY_FILE")
	switch role {
	case "envdump":
		body := strings.Join(os.Environ(), "\n") + "\nPPID=" + strconv.Itoa(os.Getppid())
		if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	case "marker":
		if err := os.WriteFile(file, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
			t.Fatal(err)
		}
	case "sleeper":
		if err := os.WriteFile(file, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(300 * time.Second)
	default:
		t.Fatalf("unknown custody helper role %q", role)
	}
}

func custodyDigest(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func openCustodyScope(t *testing.T, mod func(*processscope.Options)) processscope.Scope {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	opts := processscope.Options{
		HelperExecutable: exe,
		HelperDigest:     strings.TrimPrefix(custodyDigest(t, exe), "sha256:"),
		ScratchRoot:      t.TempDir(),
		Limits:           processscope.Limits{Jobs: 4, WallMS: 1200000},
	}
	if mod != nil {
		mod(&opts)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, err := processscope.Open(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func closeCustodyScope(t *testing.T, s processscope.Scope, timeout time.Duration) processscope.CleanupReport {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rep, err := s.Close(ctx)
	if err != nil {
		t.Fatalf("close custody scope: %v", err)
	}
	return rep
}

func requireCustodyClean(t *testing.T, rep processscope.CleanupReport, wantJobs int) {
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

func custodyHelperCommand(t *testing.T, role string, vars ...string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env := append([]string{"PATH=/bin:/usr/bin", "MACHINERY_CUSTODY_ROLE=" + role}, vars...)
	cmd := exec.CommandContext(t.Context(), exe, "-test.run=^TestCustodyScopeHelper$", "-test.timeout=290s")
	cmd.Env = env
	return cmd
}

func custodySentinel(t *testing.T) int {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "/bin/sleep", "300")
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

func custodyWaitFile(t *testing.T, path string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(path)
		if err == nil && len(b) > 0 {
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
		if !custodyAlive(pid) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive after %v", pid, timeout)
}

func TestWithScopeBindsScopeIntoContext(t *testing.T) {
	s := openCustodyScope(t, nil)
	defer closeCustodyScope(t, s, 30*time.Second)
	ctx := context.Background()
	if got := ScopeFromContext(ctx); got != nil {
		t.Fatalf("plain context must carry no scope, got %v", got)
	}
	scoped := WithScope(ctx, s)
	if got := ScopeFromContext(scoped); got == nil {
		t.Fatal("WithScope must carry the custody scope for scoped execution")
	}
}

func TestWithScopeRejectsNilArguments(t *testing.T) {
	s := openCustodyScope(t, nil)
	defer closeCustodyScope(t, s, 30*time.Second)
	mustPanic := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s must panic on nil arguments", name)
			}
		}()
		f()
	}
	mustPanic("nil context", func() { WithScope(nil, s) })
	mustPanic("nil scope", func() { WithScope(context.Background(), nil) })
}

func TestAttachScopeFailClosedOnMalformedCommand(t *testing.T) {
	s := openCustodyScope(t, nil)
	defer closeCustodyScope(t, s, 30*time.Second)
	valid := custodyHelperCommand(t, "marker", "MACHINERY_CUSTODY_FILE=/dev/null")

	if err := AttachScope(nil, s); err == nil || !strings.Contains(err.Error(), "nil command") {
		t.Fatalf("nil command must fail closed: %v", err)
	}
	if err := AttachScope(valid, nil); err == nil || !strings.Contains(err.Error(), "nil custody scope") {
		t.Fatalf("nil scope must fail closed: %v", err)
	}
	withFiles := custodyHelperCommand(t, "marker", "MACHINERY_CUSTODY_FILE=/dev/null")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	withFiles.ExtraFiles = []*os.File{w}
	if err := AttachScope(withFiles, s); err == nil || !strings.Contains(err.Error(), "ExtraFiles") {
		t.Fatalf("custom inherited descriptors must fail closed: %v", err)
	}
	withAttr := custodyHelperCommand(t, "marker", "MACHINERY_CUSTODY_FILE=/dev/null")
	withAttr.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := AttachScope(withAttr, s); err == nil || !strings.Contains(err.Error(), "SysProcAttr") {
		t.Fatalf("conflicting SysProcAttr must fail closed: %v", err)
	}
	// The safe control: an ordinary well-formed command attaches cleanly.
	if err := AttachScope(valid, s); err != nil {
		t.Fatalf("valid scoped command must attach: %v", err)
	}
}

func TestScopedRunPreservesOrdinaryRunCompatibility(t *testing.T) {
	sentinel := custodySentinel(t)
	s := openCustodyScope(t, nil)
	cmd := exec.CommandContext(t.Context(), "/bin/echo", "hi")
	cmd.Env = []string{"PATH=/bin:/usr/bin"}
	if err := AttachScope(cmd, s); err != nil {
		t.Fatalf("attach: %v", err)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := Run(WithScope(context.Background(), s), cmd); err != nil {
		t.Fatalf("scoped run of a successful command must behave like ordinary Run: %v", err)
	}
	if out.String() != "hi\n" {
		t.Fatalf("scoped stdout capture: %q", out.String())
	}
	if code, _, exited := ExitStatus(nil); exited || code != 0 {
		t.Fatalf("nil error must classify as no target exit: %d %v", code, exited)
	}
	requireCustodyClean(t, closeCustodyScope(t, s, 30*time.Second), 1)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestScopedRunClassifiesTargetExitStatus(t *testing.T) {
	sentinel := custodySentinel(t)
	s := openCustodyScope(t, nil)
	cmd := exec.CommandContext(t.Context(), "/usr/bin/false")
	cmd.Env = []string{"PATH=/bin:/usr/bin"}
	if err := AttachScope(cmd, s); err != nil {
		t.Fatalf("attach: %v", err)
	}
	err := Run(WithScope(context.Background(), s), cmd)
	if err == nil {
		t.Fatal("assertion-failure exit must surface as an error")
	}
	code, signal, exited := ExitStatus(err)
	if !exited || code != 1 || signal != "" {
		t.Fatalf("ExitStatus of a scoped nonzero exit = (%d, %q, %v), want (1, \"\", true)", code, signal, exited)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("target exit must not classify as cancellation: %v", err)
	}
	requireCustodyClean(t, closeCustodyScope(t, s, 30*time.Second), 1)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestScopedRunCancellationPreservesErrorsIsSemantics(t *testing.T) {
	sentinel := custodySentinel(t)
	s := openCustodyScope(t, nil)
	defer func() {
		rep := closeCustodyScope(t, s, 45*time.Second)
		if rep.Status != processscope.StatusCleaned {
			t.Fatalf("cleanup after cancellation: %+v", rep)
		}
	}()
	for _, tc := range []struct {
		name string
		arm  func() (context.Context, context.CancelFunc)
		want error
	}{
		{"timeout", func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 400*time.Millisecond)
		}, context.DeadlineExceeded},
		{"cancel", func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		}, context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), "/bin/sleep", "300")
			cmd.Env = []string{"PATH=/bin:/usr/bin"}
			if err := AttachScope(cmd, s); err != nil {
				t.Fatalf("attach: %v", err)
			}
			ctx, cancel := tc.arm()
			if tc.name == "cancel" {
				go func() {
					time.Sleep(200 * time.Millisecond)
					cancel()
				}()
			} else {
				defer cancel()
			}
			err := Run(WithScope(ctx, s), cmd)
			if err == nil {
				t.Fatal("canceled scoped run must fail")
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("cancellation must preserve errors.Is(%v): %v", tc.want, err)
			}
			if _, _, exited := ExitStatus(err); exited {
				t.Fatalf("cancellation must not classify as a target exit: %v", err)
			}
		})
	}
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

// TestScopedRunExecutesChildUnderGuardianCustody is the immutable
// unsafe-implementation challenge for the fail-closed safe defaults: it fails
// unless the scoped child actually ran under a broker guardian (distinct
// parent) with a sanitized declared environment.
func TestScopedRunExecutesChildUnderGuardianCustody(t *testing.T) {
	sentinel := custodySentinel(t)
	s := openCustodyScope(t, nil)
	envFile := filepath.Join(t.TempDir(), "scoped-env.txt")
	// Hostile values stay ambient: the declared environment must be the only
	// channel into the scoped child.
	t.Setenv("JAVA_TOOL_OPTIONS", "-javaagent:/hostile.jar")
	cmd := custodyHelperCommand(t, "envdump", "MACHINERY_CUSTODY_FILE="+envFile)
	if err := AttachScope(cmd, s); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if err := Run(WithScope(context.Background(), s), cmd); err != nil {
		t.Fatalf("scoped run: %v", err)
	}
	env := custodyWaitFile(t, envFile, 10*time.Second)
	ppid := 0
	for _, line := range strings.Split(env, "\n") {
		if v, ok := strings.CutPrefix(line, "PPID="); ok {
			ppid, _ = strconv.Atoi(v)
		}
	}
	if ppid == 0 || ppid == os.Getpid() || ppid == os.Getppid() {
		t.Fatalf("scoped child must run under a distinct guardian parent, saw ppid %d (self %d, parent %d):\n%s", ppid, os.Getpid(), os.Getppid(), env)
	}
	if strings.Contains(env, "hostile") {
		t.Fatalf("declared hostile environment reached the scoped child:\n%s", env)
	}
	requireCustodyClean(t, closeCustodyScope(t, s, 30*time.Second), 1)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestScopedRunClosedScopeFailsClosedWithoutFallback(t *testing.T) {
	sentinel := custodySentinel(t)
	s := openCustodyScope(t, nil)
	requireCustodyClean(t, closeCustodyScope(t, s, 30*time.Second), 0)
	marker := filepath.Join(t.TempDir(), "marker.txt")
	cmd := custodyHelperCommand(t, "marker", "MACHINERY_CUSTODY_FILE="+marker)
	if err := AttachScope(cmd, s); err != nil {
		t.Fatalf("attach must bind before the scope closes: %v", err)
	}
	err := Run(context.Background(), cmd)
	if err == nil {
		t.Fatal("run on a closed scope must fail closed")
	}
	var serr *processscope.Error
	if !errors.As(err, &serr) {
		t.Fatalf("closed-scope failure must preserve the custody error type: %v", err)
	}
	if serr.Code != processscope.CodeScopeClosed {
		t.Fatalf("closed-scope failure code = %s, want %s", serr.Code, processscope.CodeScopeClosed)
	}
	if _, statErr := os.Lstat(marker); statErr == nil {
		t.Fatal("closed scope executed the command unscoped: marker file exists")
	}
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestScopedRunAmbiguousCustodyFailsClosed(t *testing.T) {
	first := openCustodyScope(t, nil)
	second := openCustodyScope(t, nil)
	defer closeCustodyScope(t, second, 30*time.Second)
	defer closeCustodyScope(t, first, 30*time.Second)
	cmd := exec.CommandContext(t.Context(), "/bin/echo", "hi")
	cmd.Env = []string{"PATH=/bin:/usr/bin"}
	if err := AttachScope(cmd, first); err != nil {
		t.Fatalf("attach: %v", err)
	}
	err := Run(WithScope(context.Background(), second), cmd)
	if err == nil || !strings.Contains(err.Error(), "custody") {
		t.Fatalf("binding/context custody mismatch must fail closed with a custody diagnostic: %v", err)
	}
}

func TestScopedRunReportsCleanupFailureHonestly(t *testing.T) {
	sentinel := custodySentinel(t)
	s := openCustodyScope(t, func(o *processscope.Options) { o.Limits.CleanupMS = 1; o.Limits.Jobs = 2 })
	pidFile := filepath.Join(t.TempDir(), "sleeper.pid")
	cmd := custodyHelperCommand(t, "sleeper", "MACHINERY_CUSTODY_FILE="+pidFile)
	if err := AttachScope(cmd, s); err != nil {
		t.Fatalf("attach: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	err := Run(WithScope(ctx, s), cmd)
	if err == nil {
		t.Fatal("cancellation with an exhausted cleanup budget must be an error")
	}
	var serr *processscope.Error
	if !errors.As(err, &serr) || serr.Code == "" {
		t.Fatalf("cancellation must preserve the custody error classification: %v", err)
	}
	pid, perr := strconv.Atoi(custodyWaitFile(t, pidFile, 10*time.Second))
	if perr != nil {
		t.Fatal(perr)
	}
	rep := closeCustodyScope(t, s, 45*time.Second)
	if rep.Status != processscope.StatusCleanupFailed {
		t.Fatalf("exhausted cleanup budget must be reported honestly: %+v", rep)
	}
	custodyWaitGone(t, pid, 15*time.Second)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestExitStatusCoversUnscopedExitError(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "/usr/bin/false")
	cmd.Env = []string{"PATH=/bin:/usr/bin"}
	err := Run(context.Background(), cmd)
	if err == nil {
		t.Fatal("unscoped false must fail")
	}
	code, _, exited := ExitStatus(err)
	if !exited || code != 1 {
		t.Fatalf("unscoped exit classification = (%d, %v), want (1, true)", code, exited)
	}
	if _, _, exited := ExitStatus(errors.New("not an exit")); exited {
		t.Fatal("non-exit errors must not classify as target exits")
	}
}

func TestScopedRunScopedGrandchildTerminationAndReap(t *testing.T) {
	sentinel := custodySentinel(t)
	s := openCustodyScope(t, nil)
	scratch := t.TempDir()
	grandchild := filepath.Join(scratch, "grandchild.pid")
	// The helper spawns a long-lived /bin/sleep grandchild and exits as an
	// intermediate parent; custody must still own the descendant and reap it
	// before Run returns.
	cmd := exec.CommandContext(t.Context(), "/bin/sh", "-c", "/bin/sleep 300 & echo $! > "+grandchild+"; sleep 2")
	cmd.Env = []string{"PATH=/bin:/usr/bin"}
	if err := AttachScope(cmd, s); err != nil {
		t.Fatalf("attach: %v", err)
	}
	runDone := make(chan error, 1)
	go func() { runDone <- Run(WithScope(context.Background(), s), cmd) }()
	pid, err := strconv.Atoi(strings.TrimSpace(custodyWaitFile(t, grandchild, 10*time.Second)))
	if err != nil {
		t.Fatal(err)
	}
	if !custodyAlive(pid) {
		t.Fatal("owned grandchild must be alive while its intermediate parent runs")
	}
	select {
	case runErr := <-runDone:
		if runErr != nil {
			t.Fatalf("scoped intermediate must exit cleanly: %v", runErr)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("scoped intermediate did not return")
	}
	custodyWaitGone(t, pid, 15*time.Second)
	requireCustodyClean(t, closeCustodyScope(t, s, 45*time.Second), 1)
	if !custodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}
