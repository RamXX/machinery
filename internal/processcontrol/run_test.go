package processcontrol

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const helperEnv = "MACHINERY_PROCESSCONTROL_HELPER"

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}

func TestRunCancelsDirectChild(t *testing.T) {
	switch os.Getenv(helperEnv) {
	case "sleep":
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "residual-parent":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestRunCancelsDirectChild")
		cmd.Env = replaceEnv(os.Environ(), helperEnv, "residual-child")
		if err := cmd.Start(); err != nil {
			os.Exit(2)
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			if _, err := os.Lstat(os.Getenv("MACHINERY_PROCESSCONTROL_READY")); err == nil {
				break
			}
			if time.Now().After(deadline) {
				os.Exit(3)
			}
			time.Sleep(5 * time.Millisecond)
		}
		os.Exit(0)
	case "residual-child":
		_ = os.WriteFile(os.Getenv("MACHINERY_PROCESSCONTROL_READY"), []byte("ready"), 0o600)
		time.Sleep(5 * time.Second)
		_ = os.WriteFile(os.Getenv("MACHINERY_PROCESSCONTROL_SENTINEL"), []byte("survived"), 0o600)
		os.Exit(0)
	case "stdout-overflow":
		_, _ = os.Stdout.Write([]byte("0123456789"))
		os.Exit(0)
	case "terminate-failure-parent":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRunCancelsDirectChild$")
		cmd.Env = replaceEnv(os.Environ(), helperEnv, "terminate-failure-child")
		cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_READY", os.Getenv("MACHINERY_PROCESSCONTROL_READY"))
		cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_SENTINEL", os.Getenv("MACHINERY_PROCESSCONTROL_SENTINEL"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			os.Exit(4)
		}
		if err := cmd.Wait(); err != nil {
			os.Exit(5)
		}
		os.Exit(0)
	case "terminate-failure-child":
		_ = os.WriteFile(os.Getenv("MACHINERY_PROCESSCONTROL_READY"), []byte("ready"), 0o600)
		time.Sleep(2 * time.Second)
		_ = os.WriteFile(os.Getenv("MACHINERY_PROCESSCONTROL_SENTINEL"), []byte("survived"), 0o600)
		os.Exit(0)
	case "terminate-failure-overflow":
		_, _ = os.Stdout.Write([]byte("0123456789"))
		_ = os.WriteFile(os.Getenv("MACHINERY_PROCESSCONTROL_READY"), []byte("ready"), 0o600)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestRunCancelsDirectChild")
	cmd.Env = replaceEnv(os.Environ(), helperEnv, "sleep")
	if err := Run(ctx, cmd); err == nil {
		t.Fatal("timed-out subprocess returned success")
	}
}

func TestRunCapturedJoinsCancellationTerminationAndTruncationErrors(t *testing.T) {
	ready := filepath.Join(t.TempDir(), "ready")
	injectedErr := errors.New("injected tree termination failure")
	original := terminateProcessTree
	terminateProcessTree = func(*treeControl, *exec.Cmd) error { return injectedErr }
	t.Cleanup(func() { terminateProcessTree = original })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRunCancelsDirectChild$")
	cmd.Env = replaceEnv(os.Environ(), helperEnv, "terminate-failure-overflow")
	cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_READY", ready)
	type result struct {
		stdout string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		stdout, _, err := RunCapturedStreams(ctx, cmd, 4)
		done <- result{stdout: stdout, err: err}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Lstat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatal("overflow helper did not become ready")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, injectedErr) || !strings.Contains(got.err.Error(), "stdout exceeded 4-byte capture limit") {
			t.Fatalf("joined cancellation error = %v", got.err)
		}
		if got.stdout != "0123\n[output truncated at 4 bytes]\n" {
			t.Fatalf("bounded cancellation output = %q", got.stdout)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("captured cancellation did not reap after termination failure")
	}
}

func TestRunCancellationSurvivesInjectedTreeTerminationFailure(t *testing.T) {
	dir := t.TempDir()
	ready := dir + string(os.PathSeparator) + "ready"
	sentinel := dir + string(os.PathSeparator) + "survived"
	injectedErr := errors.New("injected tree termination failure")
	original := terminateProcessTree
	terminateProcessTree = func(*treeControl, *exec.Cmd) error { return injectedErr }
	t.Cleanup(func() { terminateProcessTree = original })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRunCancelsDirectChild$")
	cmd.Env = replaceEnv(os.Environ(), helperEnv, "terminate-failure-parent")
	cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_READY", ready)
	cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_SENTINEL", sentinel)
	var output strings.Builder
	cmd.Stdout, cmd.Stderr = &output, &output
	done := make(chan error, 1)
	go func() { done <- Run(ctx, cmd) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Lstat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatalf("descendant did not become ready; output=%q", output.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
	started := time.Now()
	cancel()
	var err error
	select {
	case err = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("termination failure stranded process reap")
	}
	if !errors.Is(err, injectedErr) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v, want injected termination failure and cancellation", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("termination failure stranded process reap for %s: %v", elapsed, err)
	}
	if cmd.ProcessState == nil {
		t.Fatalf("direct child was not reaped after termination failure: %#v", cmd.ProcessState)
	}
	if _, err := os.Lstat(ready); err != nil {
		t.Fatalf("descendant did not hold inherited output descriptors before cancellation: %v; output=%q", err, output.String())
	}
	time.Sleep(2 * time.Second)
	if _, err := os.Lstat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("descendant survived independent cancellation cleanup: %v", err)
	}
}

func TestRunCapturedStreamsRejectsSuccessfulOutputOverflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRunCancelsDirectChild$")
	cmd.Env = replaceEnv(os.Environ(), helperEnv, "stdout-overflow")
	stdout, stderr, err := RunCapturedStreams(ctx, cmd, 4)
	if err == nil || !strings.Contains(err.Error(), "stdout exceeded 4-byte capture limit") {
		t.Fatalf("overflow error = %v", err)
	}
	if stdout != "0123\n[output truncated at 4 bytes]\n" || stderr != "" {
		t.Fatalf("bounded streams = %q / %q", stdout, stderr)
	}
}

func TestRunKillsResidualChildAfterSuccessfulParent(t *testing.T) {
	dir := t.TempDir()
	sentinel := dir + string(os.PathSeparator) + "survived"
	ready := dir + string(os.PathSeparator) + "ready"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestRunCancelsDirectChild")
	cmd.Env = replaceEnv(os.Environ(), helperEnv, "residual-parent")
	cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_SENTINEL", sentinel)
	cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_READY", ready)
	if err := Run(ctx, cmd); err != nil {
		t.Fatalf("successful parent failed: %v", err)
	}
	time.Sleep(time.Second)
	if _, err := os.Lstat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("residual child survived process-tree close: %v", err)
	}
}
