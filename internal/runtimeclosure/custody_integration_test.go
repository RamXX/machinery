//go:build unix

package runtimeclosure

import (
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

func runtimeCustodyDigest(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func openRuntimeCustodyScope(t *testing.T) processscope.Scope {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	s, err := processscope.Open(ctx, processscope.Options{
		HelperExecutable: exe,
		HelperDigest:     runtimeCustodyDigest(t, exe),
		ScratchRoot:      t.TempDir(),
		Limits:           processscope.Limits{Jobs: 4, WallMS: 1200000},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func closeRuntimeCustodyScope(t *testing.T, s processscope.Scope, timeout time.Duration) processscope.CleanupReport {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rep, err := s.Close(ctx)
	if err != nil {
		t.Fatalf("close runtime custody scope: %v", err)
	}
	return rep
}

func requireRuntimeCustodyClean(t *testing.T, rep processscope.CleanupReport, wantJobs int) {
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

func runtimeCustodySentinel(t *testing.T) int {
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

func runtimeCustodyAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func runtimeCustodyWaitFile(t *testing.T, path string, timeout time.Duration) string {
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

func runtimeCustodyWaitGone(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !runtimeCustodyAlive(pid) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive after %v", pid, timeout)
}

// writeWatchProgram writes the single-file Java source used to observe a
// genuinely running pinned JVM: it publishes its own pid and then sleeps
// forever.
func writeWatchProgram(t *testing.T, dir string) string {
	t.Helper()
	body := "import java.nio.file.*;\n" +
		"public class Watch {\n" +
		"    public static void main(String[] args) throws Exception {\n" +
		"        Files.write(Paths.get(args[0]), Long.toString(ProcessHandle.current().pid()).getBytes());\n" +
		"        Thread.sleep(Long.MAX_VALUE);\n" +
		"    }\n" +
		"}\n"
	path := filepath.Join(dir, "Watch.java")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// usePinnedJava clears explicit overrides so OpenJava resolves the provisioned
// pinned runtime (downloading it through the normal pinned mechanism when the
// cache is cold).
func usePinnedJava(t *testing.T) {
	t.Helper()
	t.Setenv(JavaEnv, "")
	t.Setenv(JavaClosureSHAEnv, "")
}

// TestAttachCustodyBindsScopeAfterSanitation is the unsafe-implementation
// challenge for the sanitized-environment safe defaults: a scoped launch must
// run under a broker guardian (distinct parent) while staying sanitized.
func TestAttachCustodyBindsScopeAfterSanitation(t *testing.T) {
	s := openRuntimeCustodyScope(t)
	defer closeRuntimeCustodyScope(t, s, 30*time.Second)
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env.txt")
	ppidFile := filepath.Join(dir, "ppid.txt")
	cmd := exec.Command("/bin/sh", "-c", "env > "+envFile+"; echo $PPID > "+ppidFile)
	cmd.Env = Environment(dir, dir, "/usr/bin/true")
	t.Setenv("JAVA_TOOL_OPTIONS", "-javaagent:/hostile.jar")
	ctx := processcontrol.WithScope(context.Background(), s)
	if err := AttachCustody(ctx, cmd); err != nil {
		t.Fatalf("AttachCustody after sanitation: %v", err)
	}
	if err := processcontrol.Run(ctx, cmd); err != nil {
		t.Fatalf("scoped run: %v", err)
	}
	env := runtimeCustodyWaitFile(t, envFile, 10*time.Second)
	if strings.Contains(env, "hostile") {
		t.Fatalf("ambient hostile values survived sanitation:\n%s", env)
	}
	ppidText := strings.TrimSpace(runtimeCustodyWaitFile(t, ppidFile, 10*time.Second))
	ppid, err := strconv.Atoi(ppidText)
	if err != nil {
		t.Fatalf("guardian ppid %q: %v", ppidText, err)
	}
	if ppid == 0 || ppid == os.Getpid() || ppid == os.Getppid() {
		t.Fatalf("scoped child must run under a distinct guardian parent, saw %d (self %d, parent %d)", ppid, os.Getpid(), os.Getppid())
	}
}

// TestAttachCustodyAbsentScopeIsCompatibilityNoOp is the safe control for the
// challenge above: without a scope the ordinary compatibility path runs.
func TestAttachCustodyAbsentScopeIsCompatibilityNoOp(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("/bin/echo", "hi")
	cmd.Env = Environment(dir, dir, "/usr/bin/true")
	if err := AttachCustody(context.Background(), cmd); err != nil {
		t.Fatalf("absent scope must be a no-op: %v", err)
	}
	if err := processcontrol.Run(context.Background(), cmd); err != nil {
		t.Fatalf("ordinary run: %v", err)
	}
	if err := AttachCustody(context.Background(), nil); err == nil {
		t.Fatal("nil command must fail closed")
	}
}

// TestScopedPinnedJVMRunsBeforeCancellationAndIsReaped observes a genuinely
// running pinned nested JVM (the Watch single-file program on the provisioned
// Temurin runtime), cancels it, and proves the owned JVM is gone while an
// unrelated sentinel survives and the opened closure identity is retained.
func TestScopedPinnedJVMRunsBeforeCancellationAndIsReaped(t *testing.T) {
	sentinel := runtimeCustodySentinel(t)
	usePinnedJava(t)
	s := openRuntimeCustodyScope(t)
	dir := t.TempDir()
	watch := writeWatchProgram(t, dir)
	pidFile := filepath.Join(dir, "watch.pid")
	tmp := filepath.Join(dir, "tmp")
	if err := os.Mkdir(tmp, 0o755); err != nil {
		t.Fatal(err)
	}

	java, err := OpenJava()
	if err != nil {
		t.Fatalf("open pinned Java runtime: %v", err)
	}
	defer java.Close()
	identity := java.Identity()
	path := java.Path()

	ctx, cancel := context.WithCancel(processcontrol.WithScope(context.Background(), s))
	cmd := exec.CommandContext(ctx, path, watch, pidFile)
	cmd.Dir = dir
	cmd.Env = Environment(dir, tmp, path)
	if err := AttachCustody(ctx, cmd); err != nil {
		t.Fatalf("attach custody to pinned JVM launch: %v", err)
	}
	runDone := make(chan error, 1)
	go func() { runDone <- processcontrol.Run(ctx, cmd) }()

	pidText := runtimeCustodyWaitFile(t, pidFile, 60*time.Second)
	pid, err := strconv.Atoi(strings.TrimSpace(pidText))
	if err != nil {
		t.Fatalf("watch pid %q: %v", pidText, err)
	}
	if !runtimeCustodyAlive(pid) {
		t.Fatalf("pinned JVM %d must be observed alive before cancellation", pid)
	}
	time.Sleep(500 * time.Millisecond)
	if !runtimeCustodyAlive(pid) {
		t.Fatal("pinned JVM died before cancellation")
	}
	cancel()
	select {
	case runErr := <-runDone:
		if runErr == nil {
			t.Fatal("canceled pinned JVM launch must report an error")
		}
		if !errors.Is(runErr, context.Canceled) {
			t.Fatalf("cancellation must preserve errors.Is semantics: %v", runErr)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("canceled pinned JVM launch did not return")
	}
	runtimeCustodyWaitGone(t, pid, 30*time.Second)
	requireRuntimeCustodyClean(t, closeRuntimeCustodyScope(t, s, 60*time.Second), 1)
	if err := java.Validate(); err != nil {
		t.Fatalf("opened Java closure identity must survive scoped cancellation: %v", err)
	}
	if java.Identity() != identity {
		t.Fatalf("Java identity changed: %q -> %q", identity, java.Identity())
	}
	if !runtimeCustodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

// TestScopedJavaProbePreservesIdentityUnderCustody runs the real identity
// probe shape on the pinned runtime through scoped custody and verifies the
// probe output still binds the opened closure identity.
func TestScopedJavaProbePreservesIdentityUnderCustody(t *testing.T) {
	sentinel := runtimeCustodySentinel(t)
	usePinnedJava(t)
	s := openRuntimeCustodyScope(t)
	workdir := t.TempDir()

	java, err := OpenJava()
	if err != nil {
		t.Fatalf("open pinned Java runtime: %v", err)
	}
	defer java.Close()

	ctx, cancel := context.WithTimeout(processcontrol.WithScope(context.Background(), s), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, java.Path(), "-XshowSettings:properties", "-version")
	cmd.Env = Environment(workdir, workdir, java.Path())
	if err := AttachCustody(ctx, cmd); err != nil {
		t.Fatalf("attach custody to probe: %v", err)
	}
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := processcontrol.Run(ctx, cmd); err != nil {
		t.Fatalf("scoped pinned probe: %v", err)
	}
	if err := java.BindIdentity(out.String()); err != nil {
		t.Fatalf("probe output must still bind the opened closure identity: %v\n%s", err, out.String())
	}
	if err := java.Validate(); err != nil {
		t.Fatalf("validate pinned closure after scoped probe: %v", err)
	}
	if !strings.Contains(java.Identity(), PinnedJavaProbeVersion) {
		t.Fatalf("pinned probe identity = %q", java.Identity())
	}
	requireRuntimeCustodyClean(t, closeRuntimeCustodyScope(t, s, 60*time.Second), 1)
	if !runtimeCustodyAlive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

// TestScopedJavaEnvironmentExcludesHostileAmbient is the safe control for
// environment sanitation on the ordinary path: hostile ambient values never
// reach a sanitized child, scoped or not.
func TestScopedJavaEnvironmentExcludesHostileAmbient(t *testing.T) {
	s := openRuntimeCustodyScope(t)
	defer closeRuntimeCustodyScope(t, s, 30*time.Second)
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env.txt")
	t.Setenv("JAVA_TOOL_OPTIONS", "-javaagent:/hostile.jar")
	t.Setenv("JDK_JAVA_OPTIONS", "-Dhostile=true")
	t.Setenv("CLASSPATH", "/hostile/classpath")
	cmd := exec.Command("/bin/sh", "-c", "env > "+envFile)
	cmd.Env = Environment(dir, dir, "/usr/bin/true")
	ctx := processcontrol.WithScope(context.Background(), s)
	if err := AttachCustody(ctx, cmd); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if err := processcontrol.Run(ctx, cmd); err != nil {
		t.Fatalf("scoped run: %v", err)
	}
	env := runtimeCustodyWaitFile(t, envFile, 10*time.Second)
	for _, forbidden := range []string{"hostile", "CLASSPATH"} {
		if strings.Contains(env, forbidden) {
			t.Fatalf("sanitized scoped environment leaked %s:\n%s", forbidden, env)
		}
	}
	for _, required := range []string{"TZ=UTC", "LC_ALL=C.UTF-8", "HOME="} {
		if !strings.Contains(env, required) {
			t.Fatalf("sanitized scoped environment lacks %s:\n%s", required, env)
		}
	}
}
