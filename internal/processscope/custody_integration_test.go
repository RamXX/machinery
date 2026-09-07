//go:build unix

package processscope_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
)

const runtimeDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

func digestOf(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func openScope(t *testing.T, scratch string, mod func(*processscope.Options)) processscope.Scope {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	opts := processscope.Options{
		HelperExecutable: exe,
		HelperDigest:     digestOf(t, exe),
		ScratchRoot:      scratch,
		Limits:           processscope.Limits{Jobs: 4},
	}
	if mod != nil {
		mod(&opts)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	s, err := processscope.Open(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func envBase(extra ...string) []string {
	env := []string{"PATH=/bin:/usr/bin"}
	return append(env, extra...)
}

func sleepCommand(secs string) processscope.Command {
	return processscope.Command{
		Executable:    "/bin/sleep",
		Args:          []string{secs},
		Env:           envBase(),
		RuntimeDigest: runtimeDigest,
	}
}

func roleCommand(t *testing.T, role string, vars ...string) processscope.Command {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env := envBase(append([]string{"MACHINERY_QLW2_ROLE=" + role}, vars...)...)
	return processscope.Command{
		Executable:    exe,
		Args:          []string{"-test.run=^TestHelperTarget$", "-test.timeout=180s"},
		Env:           env,
		RuntimeDigest: "sha256:" + digestOf(t, exe),
	}
}

func attach(t *testing.T, s processscope.Scope, cmd processscope.Command) processscope.Command {
	t.Helper()
	attached, err := s.Attach(cmd)
	if err != nil {
		t.Fatal(err)
	}
	return attached
}

func codeOf(err error) string {
	if err == nil {
		return ""
	}
	var e *processscope.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return "OTHER:" + err.Error()
}

func alive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func waitFile(t *testing.T, path string, timeout time.Duration) string {
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

func waitPidFile(t *testing.T, path string, timeout time.Duration) int {
	t.Helper()
	n, err := strconv.Atoi(waitFile(t, path, timeout))
	if err != nil {
		t.Fatalf("pid file %s: %v", path, err)
	}
	return n
}

func waitGone(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !alive(pid) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive after %v", pid, timeout)
}

func startSentinel(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("/bin/sleep", "120")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	return cmd.Process.Pid
}

func closeScope(t *testing.T, s processscope.Scope, timeout time.Duration) processscope.CleanupReport {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rep, err := s.Close(ctx)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	return rep
}

func requireClean(t *testing.T, rep processscope.CleanupReport, wantJobs int) processscope.CleanupReport {
	t.Helper()
	if rep.Status != processscope.StatusCleaned {
		t.Fatalf("cleanup status %s: %+v", rep.Status, rep)
	}
	if len(rep.Jobs) != wantJobs {
		t.Fatalf("expected %d jobs in report, got %d: %+v", wantJobs, len(rep.Jobs), rep.Jobs)
	}
	for _, j := range rep.Jobs {
		if !j.Registered || !j.Terminated || !j.Reaped {
			t.Fatalf("job state incomplete: %+v", j)
		}
	}
	return rep
}

func TestHelperTarget(t *testing.T) {
	role := os.Getenv("MACHINERY_QLW2_ROLE")
	if role == "" {
		return
	}
	file := os.Getenv("MACHINERY_QLW2_FILE")
	switch role {
	case "grandparent":
		secs := os.Getenv("MACHINERY_QLW2_SECS")
		if secs == "" {
			secs = "300"
		}
		c := exec.Command("/bin/sleep", secs)
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(strconv.Itoa(c.Process.Pid)), 0644); err != nil {
			t.Fatal(err)
		}
		return
	case "sleeper":
		if err := os.WriteFile(file, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(300 * time.Second)
	case "writer":
		buf := bytes.Repeat([]byte("x"), 4096)
		for i := 0; i < 512; i++ {
			if _, err := os.Stdout.Write(buf); err != nil {
				return
			}
		}
	case "joiner":
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cap, err := processscope.InheritedCapability(ctx)
		if err != nil {
			os.WriteFile(file, []byte("constructor:"+codeOf(err)), 0644)
			return
		}
		sc, err := processscope.Join(ctx, cap)
		if err != nil {
			os.WriteFile(file, []byte("join:"+codeOf(err)), 0644)
			return
		}
		childFile := os.Getenv("MACHINERY_QLW2_CHILD_FILE")
		exe, _ := os.Executable()
		dg := digestOf(t, exe)
		cmd := processscope.Command{
			Executable:    exe,
			Args:          []string{"-test.run=^TestHelperTarget$", "-test.timeout=180s"},
			Env:           envBase("MACHINERY_QLW2_ROLE=grandparent", "MACHINERY_QLW2_FILE="+childFile, "MACHINERY_QLW2_SECS=300"),
			RuntimeDigest: "sha256:" + dg,
		}
		attached, err := sc.Attach(cmd)
		if err != nil {
			os.WriteFile(file, []byte("attach:"+codeOf(err)), 0644)
			return
		}
		go func() { sc.Run(context.Background(), attached, processscope.Streams{}) }()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(childFile); err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		os.WriteFile(file, []byte("joiner-started"), 0644)
		time.Sleep(300 * time.Millisecond)
	case "stale-delegate":
		result := os.Getenv("MACHINERY_QLW2_RESULT")
		delay := os.Getenv("MACHINERY_QLW2_DELAY")
		if delay == "" {
			delay = "1500ms"
		}
		c := exec.Command(os.Args[0], "-test.run=^TestHelperTarget$", "-test.timeout=60s")
		c.Env = envBase(
			"MACHINERY_QLW2_ROLE=stale-child",
			"MACHINERY_QLW2_RESULT="+result,
			"MACHINERY_QLW2_DELAY="+delay,
			processscope.EnvCapability+"="+os.Getenv(processscope.EnvCapability),
		)
		c.ExtraFiles = []*os.File{os.NewFile(3, "capability")}
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("delegated"), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(300 * time.Millisecond)
	case "stale-child":
		result := os.Getenv("MACHINERY_QLW2_RESULT")
		delay := os.Getenv("MACHINERY_QLW2_DELAY")
		if d, err := time.ParseDuration(delay); err == nil && d > 0 {
			time.Sleep(d)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		msg := ""
		cap, err := processscope.InheritedCapability(ctx)
		if err != nil {
			msg = "constructor:" + codeOf(err)
		} else if _, err := processscope.Join(ctx, cap); err != nil {
			msg = "join:" + codeOf(err)
		} else {
			msg = "join:OK"
		}
		if err := os.WriteFile(result, []byte(msg), 0644); err != nil {
			t.Fatal(err)
		}
	case "owner":
		scratch := os.Getenv("MACHINERY_QLW2_SCRATCH")
		childFile := os.Getenv("MACHINERY_QLW2_CHILD_FILE")
		exe, _ := os.Executable()
		dg := digestOf(t, exe)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		s, err := processscope.Open(ctx, processscope.Options{
			HelperExecutable: exe,
			HelperDigest:     dg,
			ScratchRoot:      scratch,
			Limits:           processscope.Limits{Jobs: 2},
		})
		if err != nil {
			os.WriteFile(file, []byte("open:"+codeOf(err)), 0644)
			return
		}
		cmd := processscope.Command{
			Executable:    exe,
			Args:          []string{"-test.run=^TestHelperTarget$", "-test.timeout=180s"},
			Env:           envBase("MACHINERY_QLW2_ROLE=grandparent", "MACHINERY_QLW2_FILE="+childFile, "MACHINERY_QLW2_SECS=120"),
			RuntimeDigest: "sha256:" + dg,
		}
		attached, err := s.Attach(cmd)
		if err != nil {
			os.WriteFile(file, []byte("attach:"+codeOf(err)), 0644)
			return
		}
		go func() { s.Run(ctx, attached, processscope.Streams{}) }()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(childFile); err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		b, _ := os.ReadFile(childFile)
		os.WriteFile(file, b, 0644)
		time.Sleep(300 * time.Millisecond)
		syscall.Kill(os.Getpid(), syscall.SIGKILL)
	default:
		t.Fatalf("unknown helper role %q", role)
	}
}

func TestConsumerConformanceLifecycle(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, nil)

	echo := processscope.Command{
		Executable:    "/bin/echo",
		Args:          []string{"hi"},
		Env:           envBase(),
		RuntimeDigest: runtimeDigest,
	}
	attached := attach(t, s, echo)
	if attached.DeadlineMS <= 0 {
		t.Fatalf("attachment must compute remaining deadline, got %d", attached.DeadlineMS)
	}
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := s.Run(ctx, attached, processscope.Streams{Stdout: &out, StdoutLimit: 1 << 16})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.Started || !res.Completed || res.ExitCode != 0 || res.Signal != "" || res.JobID == "" {
		t.Fatalf("result fields: %+v", res)
	}
	if out.String() != "hi\n" {
		t.Fatalf("stdout capture: %q", out.String())
	}

	child := func() processscope.Scope {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		c, err := s.Child(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}()
	attachedChild := attach(t, child, echo)
	if _, err := child.Run(ctx, attachedChild, processscope.Streams{}); err != nil {
		t.Fatalf("child run: %v", err)
	}
	requireClean(t, closeScope(t, child, 30*time.Second), 1)

	attached2 := attach(t, s, echo)
	if _, err := s.Run(ctx, attached2, processscope.Streams{}); err != nil {
		t.Fatalf("run after child close: %v", err)
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 3)
	rep2 := closeScope(t, s, 30*time.Second)
	if rep2.Status != processscope.StatusCleaned {
		t.Fatalf("close must be idempotent: %+v", rep2)
	}
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestRunAssertionFailureExitCode(t *testing.T) {
	sentinel := startSentinel(t)
	s := openScope(t, t.TempDir(), nil)
	attached := attach(t, s, processscope.Command{
		Executable:    "/usr/bin/false",
		Env:           envBase(),
		RuntimeDigest: runtimeDigest,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := s.Run(ctx, attached, processscope.Streams{})
	if err != nil {
		t.Fatalf("expected assertion-failure exit to be a result, not error: %v", err)
	}
	if !res.Completed || res.ExitCode != 1 {
		t.Fatalf("expected exit 1: %+v", res)
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 1)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestRunTimeoutCleanup(t *testing.T) {
	sentinel := startSentinel(t)
	s := openScope(t, t.TempDir(), nil)
	attached := attach(t, s, sleepCommand("300"))
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	res, err := s.Run(ctx, attached, processscope.Streams{})
	asCode(t, err, processscope.CodeTimeout)
	if !res.Started || !res.Completed || res.Signal == "" {
		t.Fatalf("timeout result must observe signal death: %+v", res)
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 1)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestRunOutputOverflow(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, nil)
	f := filepath.Join(scratch, "writer.out")
	attached := attach(t, s, roleCommand(t, "writer", "MACHINERY_QLW2_FILE="+f))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var out bytes.Buffer
	res, err := s.Run(ctx, attached, processscope.Streams{Stdout: &out, StdoutLimit: 4096})
	asCode(t, err, processscope.CodeOutputLimit)
	if !res.Started {
		t.Fatalf("overflow result: %+v", res)
	}
	if out.Len() > 4096+4096 {
		t.Fatalf("overflow bound violated: %d", out.Len())
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 1)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestGrandchildEarlyIntermediateCleanup(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, nil)
	f := filepath.Join(scratch, "grandchild.pid")
	attached := attach(t, s, roleCommand(t, "grandparent", "MACHINERY_QLW2_FILE="+f, "MACHINERY_QLW2_SECS=300"))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := s.Run(ctx, attached, processscope.Streams{})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("intermediate must exit cleanly: %v %+v", err, res)
	}
	pid := waitPidFile(t, f, 5*time.Second)
	if !alive(pid) {
		t.Fatal("grandchild must remain alive while scope is open")
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 1)
	waitGone(t, pid, 5*time.Second)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestNestedJoinedSiblingAndIntermediateExit(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, nil)
	j := filepath.Join(scratch, "joiner.txt")
	g := filepath.Join(scratch, "sibling-grandchild.pid")
	joiner := roleCommand(t, "joiner", "MACHINERY_QLW2_FILE="+j, "MACHINERY_QLW2_CHILD_FILE="+g)
	joiner.Env = append(joiner.Env, processscope.EnvChildRequest+"=join")
	attached := attach(t, s, joiner)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := s.Run(ctx, attached, processscope.Streams{})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("joiner intermediate must exit cleanly: %v %+v", err, res)
	}
	if got := waitFile(t, j, 5*time.Second); got != "joiner-started" {
		t.Fatalf("joiner evidence: %q", got)
	}
	pid := waitPidFile(t, g, 5*time.Second)
	if !alive(pid) {
		t.Fatal("sibling grandchild must survive intermediate joiner exit")
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 2)
	waitGone(t, pid, 5*time.Second)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestInterruptionSIGINTClose(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, nil)
	f := filepath.Join(scratch, "grandchild.pid")
	attached := attach(t, s, roleCommand(t, "grandparent", "MACHINERY_QLW2_FILE="+f, "MACHINERY_QLW2_SECS=300"))
	runCtx, runCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer runCancel()
	go func() { s.Run(runCtx, attached, processscope.Streams{}) }()
	pid := waitPidFile(t, f, 5*time.Second)
	if !alive(pid) {
		t.Fatal("grandchild must be alive before interruption")
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	<-sigCh
	requireClean(t, closeScope(t, s, 30*time.Second), 1)
	waitGone(t, pid, 5*time.Second)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestOwnerLossBrokerSelfCleanup(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	h := filepath.Join(scratch, "handoff.pid")
	f2 := filepath.Join(scratch, "owner-grandchild.pid")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	owner := exec.Command(exe, "-test.run=^TestHelperTarget$", "-test.timeout=60s")
	owner.Env = envBase(
		"MACHINERY_QLW2_ROLE=owner",
		"MACHINERY_QLW2_FILE="+h,
		"MACHINERY_QLW2_CHILD_FILE="+f2,
		"MACHINERY_QLW2_SCRATCH="+scratch,
	)
	owner.Stdout, _ = os.OpenFile(filepath.Join(scratch, "owner.log"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	owner.Stderr = owner.Stdout
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	pid := waitPidFile(t, h, 20*time.Second)
	if !alive(pid) {
		t.Fatal("owned grandchild must be alive while owner lives")
	}
	if err := owner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	owner.Wait()
	ledgerPath := ""
	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		matches, _ := filepath.Glob(filepath.Join(scratch, "processscope-*", "ledger.json"))
		if len(matches) == 1 {
			b, err := os.ReadFile(matches[0])
			if err == nil && len(b) > 0 {
				ledgerPath = matches[0]
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if ledgerPath == "" {
		t.Fatal("broker did not write terminal ledger after owner loss")
	}
	waitGone(t, pid, 10*time.Second)
	b, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	var ledger struct {
		Schema string `json:"schema"`
		Status string `json:"status"`
		Jobs   []struct {
			Registered bool `json:"registered"`
			Terminated bool `json:"terminated"`
			Reaped     bool `json:"reaped"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(b, &ledger); err != nil {
		t.Fatalf("ledger parse: %v", err)
	}
	if len(ledger.Jobs) < 1 || !ledger.Jobs[0].Registered || !ledger.Jobs[0].Terminated || !ledger.Jobs[0].Reaped {
		t.Fatalf("owner-loss ledger must report terminal job state: %s", b)
	}
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestClosedScopeRefusesLaunchAndIdempotentClose(t *testing.T) {
	sentinel := startSentinel(t)
	s := openScope(t, t.TempDir(), nil)
	requireClean(t, closeScope(t, s, 30*time.Second), 0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.Run(ctx, sleepCommand("1"), processscope.Streams{})
	asCode(t, err, processscope.CodeScopeClosed)
	_, err = s.Attach(sleepCommand("1"))
	asCode(t, err, processscope.CodeScopeClosed)
	child, err := s.Child(ctx)
	if err == nil {
		asCode(t, func() error { _, err := child.Run(ctx, sleepCommand("1"), processscope.Streams{}); return err }(), processscope.CodeScopeClosed)
	} else {
		asCode(t, err, processscope.CodeScopeClosed)
	}
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestChildScopeCloseCancelsOnlySubtree(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, nil)
	fr := filepath.Join(scratch, "root-grandchild.pid")
	fc := filepath.Join(scratch, "child-grandchild.pid")
	attachedRoot := attach(t, s, roleCommand(t, "grandparent", "MACHINERY_QLW2_FILE="+fr, "MACHINERY_QLW2_SECS=300"))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := s.Run(ctx, attachedRoot, processscope.Streams{}); err != nil {
		t.Fatal(err)
	}
	pidRoot := waitPidFile(t, fr, 5*time.Second)

	childCtx, childCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer childCancel()
	child, err := s.Child(childCtx)
	if err != nil {
		t.Fatal(err)
	}
	attachedChild := attach(t, child, roleCommand(t, "grandparent", "MACHINERY_QLW2_FILE="+fc, "MACHINERY_QLW2_SECS=300"))
	if _, err := child.Run(ctx, attachedChild, processscope.Streams{}); err != nil {
		t.Fatal(err)
	}
	pidChild := waitPidFile(t, fc, 5*time.Second)

	requireClean(t, closeScope(t, child, 30*time.Second), 1)
	waitGone(t, pidChild, 5*time.Second)
	if !alive(pidRoot) {
		t.Fatal("root-owned job must survive child scope close")
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 2)
	waitGone(t, pidRoot, 5*time.Second)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestStaleCapabilityAfterClose(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, nil)
	sd := filepath.Join(scratch, "delegated.txt")
	rs := filepath.Join(scratch, "stale-result.txt")
	delegate := roleCommand(t, "stale-delegate", "MACHINERY_QLW2_FILE="+sd, "MACHINERY_QLW2_RESULT="+rs, "MACHINERY_QLW2_DELAY=5000ms")
	delegate.Env = append(delegate.Env, processscope.EnvChildRequest+"=join")
	attached := attach(t, s, delegate)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := s.Run(ctx, attached, processscope.Streams{}); err != nil {
		t.Fatal(err)
	}
	if got := waitFile(t, sd, 5*time.Second); got != "delegated" {
		t.Fatalf("delegate evidence: %q", got)
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 1)
	got := waitFile(t, rs, 15*time.Second)
	if got != "join:"+processscope.CodeStaleCapability {
		t.Fatalf("stale capability must be rejected with %s, got %q", processscope.CodeStaleCapability, got)
	}
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestForgedCapabilityTransportRejected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	t.Setenv(processscope.EnvCapability, "")
	if _, err := processscope.InheritedCapability(ctx); err == nil {
		t.Fatal("absent capability must fail")
	} else {
		asCode(t, err, processscope.CodeCustodyError)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	future := time.Now().Add(time.Hour).UnixNano()
	t.Setenv(processscope.EnvCapability, fmt.Sprintf("fd=%d;scope=s;root=r;deadline=%d", w.Fd(), future))
	if _, err := processscope.InheritedCapability(ctx); err == nil {
		t.Fatal("pipe transport must be rejected")
	} else {
		asCode(t, err, processscope.CodeInvalidSchema)
	}
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	dead := os.NewFile(uintptr(fds[0]), "dead-socket")
	peer := os.NewFile(uintptr(fds[1]), "peer-socket")
	dead.Close()
	t.Setenv(processscope.EnvCapability, fmt.Sprintf("fd=%d;scope=s;root=r;deadline=%d", peer.Fd(), future))
	cap, err := processscope.InheritedCapability(ctx)
	if err != nil {
		t.Fatal(err)
	}
	joinCtx, joinCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer joinCancel()
	if _, err := processscope.Join(joinCtx, cap); err == nil {
		t.Fatal("dead broker channel must be rejected")
	} else {
		code := codeOf(err)
		if code != processscope.CodeStaleCapability && code != processscope.CodeCustodyError {
			t.Fatalf("unexpected error code: %s", code)
		}
	}
	peer.Close()
}

func TestUnrelatedSameNameProcessUntouched(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, nil)
	f := filepath.Join(scratch, "owned.pid")
	owned := roleCommand(t, "grandparent", "MACHINERY_QLW2_FILE="+f, "MACHINERY_QLW2_SECS=97")
	attached := attach(t, s, owned)
	runCtx, runCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer runCancel()
	go func() { s.Run(runCtx, attached, processscope.Streams{}) }()
	pid := waitPidFile(t, f, 5*time.Second)
	requireClean(t, closeScope(t, s, 30*time.Second), 1)
	waitGone(t, pid, 5*time.Second)
	if !alive(sentinel) {
		t.Fatal("unrelated same-name /bin/sleep 97 sentinel died")
	}
}

func TestJobLimitRegistrationRefusal(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, func(o *processscope.Options) { o.Limits.Jobs = 1 })
	f := filepath.Join(scratch, "sleeper.pid")
	first := attach(t, s, roleCommand(t, "sleeper", "MACHINERY_QLW2_FILE="+f))
	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()
	go func() { s.Run(runCtx, first, processscope.Streams{}) }()
	waitPidFile(t, f, 5*time.Second)
	time.Sleep(200 * time.Millisecond)
	second := attach(t, s, sleepCommand("1"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.Run(ctx, second, processscope.Streams{})
	asCode(t, err, processscope.CodeRegistrationFailed)
	runCancel()
	time.Sleep(300 * time.Millisecond)
	requireClean(t, closeScope(t, s, 30*time.Second), 1)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestBudgetExhaustionRefusesLaunch(t *testing.T) {
	sentinel := startSentinel(t)
	s := openScope(t, t.TempDir(), func(o *processscope.Options) { o.Limits.WallMS = 350 })
	time.Sleep(450 * time.Millisecond)
	attached, aerr := s.Attach(sleepCommand("1"))
	if aerr == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := s.Run(ctx, attached, processscope.Streams{})
		asCode(t, err, processscope.CodeBudgetExhausted)
	} else {
		asCode(t, aerr, processscope.CodeBudgetExhausted)
	}
	requireClean(t, closeScope(t, s, 30*time.Second), 0)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func TestCleanupBudgetExhaustionReported(t *testing.T) {
	sentinel := startSentinel(t)
	scratch := t.TempDir()
	s := openScope(t, scratch, func(o *processscope.Options) { o.Limits.CleanupMS = 1; o.Limits.Jobs = 2 })
	f := filepath.Join(scratch, "sleeper.pid")
	attached := attach(t, s, roleCommand(t, "sleeper", "MACHINERY_QLW2_FILE="+f))
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := s.Run(ctx, attached, processscope.Streams{})
	asCode(t, err, processscope.CodeTimeout)
	pid := waitPidFile(t, f, 5*time.Second)
	rep := closeScope(t, s, 30*time.Second)
	if rep.Status != processscope.StatusCleanupFailed {
		t.Fatalf("exhausted cleanup budget must be reported honestly: %+v", rep)
	}
	if len(rep.Diagnostics) == 0 {
		t.Fatal("cleanup-failed must carry diagnostics")
	}
	waitGone(t, pid, 10*time.Second)
	if !alive(sentinel) {
		t.Fatal("unrelated sentinel died")
	}
}

func asCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", want)
	}
	var e *processscope.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected *processscope.Error, got %T: %v", err, err)
	}
	if e.Code != want {
		t.Fatalf("expected error code %s, got %s (%v)", want, e.Code, err)
	}
}
