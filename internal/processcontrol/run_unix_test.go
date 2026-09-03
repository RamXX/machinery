//go:build !windows

package processcontrol

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestRunBoundsDetachedSessionHoldingOutputPipe(t *testing.T) {
	const key = "MACHINERY_PROCESSCONTROL_DETACHED_HELPER"
	switch os.Getenv(key) {
	case "parent":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestRunBoundsDetachedSessionHoldingOutputPipe")
		cmd.Env = replaceEnv(os.Environ(), key, "child")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	case "child":
		if _, err := syscall.Setsid(); err != nil {
			os.Exit(3)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestRunBoundsDetachedSessionHoldingOutputPipe")
	cmd.Env = replaceEnv(os.Environ(), key, "parent")
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	started := time.Now()
	err := Run(ctx, cmd)
	if err == nil {
		t.Fatal("detached output-holder was reported as success")
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("detached output-holder blocked Wait for %s: %v", elapsed, err)
	}
}

func TestRunBoundsAttachFailureWhenIndependentKillsFail(t *testing.T) {
	const key = "MACHINERY_PROCESSCONTROL_ATTACH_FAILURE_HELPER"
	switch os.Getenv(key) {
	case "parent":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRunBoundsAttachFailureWhenIndependentKillsFail$")
		cmd.Env = replaceEnv(os.Environ(), key, "child")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "child":
		if err := os.WriteFile(os.Getenv("MACHINERY_PROCESSCONTROL_READY"), []byte("ready"), 0o600); err != nil {
			os.Exit(3)
		}
		time.Sleep(2 * time.Second)
		_ = os.WriteFile(os.Getenv("MACHINERY_PROCESSCONTROL_SENTINEL"), []byte("survived"), 0o600)
		os.Exit(0)
	}

	dir := t.TempDir()
	ready := dir + string(os.PathSeparator) + "ready"
	sentinel := dir + string(os.PathSeparator) + "survived"
	attachErr := errors.New("injected attach failure")
	terminateErr := errors.New("injected tree termination failure")
	killErr := errors.New("injected direct kill failure")

	originalAttach := attachProcessTree
	originalTerminate := terminateProcessTree
	originalKill := killDirectProcess
	originalClose := closeProcessTree
	var closeCalls atomic.Int32
	attachProcessTree = func(control *treeControl, cmd *exec.Cmd) error {
		// Preserve the partially established process-group authority that a real
		// attach failure may leave behind, then wait until a descendant proves it
		// is holding the inherited output descriptors.
		if err := originalAttach(control, cmd); err != nil {
			return err
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			if _, err := os.Lstat(ready); err == nil {
				return attachErr
			}
			if time.Now().After(deadline) {
				return errors.Join(attachErr, errors.New("descendant did not become ready"))
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	terminateProcessTree = func(*treeControl, *exec.Cmd) error { return terminateErr }
	killDirectProcess = func(*exec.Cmd) error { return killErr }
	closeProcessTree = func(control *treeControl) error {
		closeCalls.Add(1)
		return originalClose(control)
	}
	t.Cleanup(func() {
		attachProcessTree = originalAttach
		terminateProcessTree = originalTerminate
		killDirectProcess = originalKill
		closeProcessTree = originalClose
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRunBoundsAttachFailureWhenIndependentKillsFail$")
	cmd.Env = replaceEnv(os.Environ(), key, "parent")
	cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_READY", ready)
	cmd.Env = replaceEnv(cmd.Env, "MACHINERY_PROCESSCONTROL_SENTINEL", sentinel)
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	started := time.Now()
	err := Run(ctx, cmd)
	if elapsed := time.Since(started); elapsed > 6*time.Second {
		t.Fatalf("attach failure cleanup exceeded fixed return bound after %s: %v", elapsed, err)
	}
	if !errors.Is(err, attachErr) || !errors.Is(err, terminateErr) || !errors.Is(err, killErr) {
		t.Fatalf("attach failure error = %v, want attach, termination, and direct-kill failures", err)
	}
	if got := closeCalls.Load(); got != 1 {
		t.Fatalf("process-tree close calls = %d, want 1", got)
	}
	if cmd.ProcessState == nil {
		t.Fatalf("direct child was not reaped after attach failure: %#v; output=%q", cmd.ProcessState, output.String())
	}
	if _, err := os.Lstat(ready); err != nil {
		t.Fatalf("descendant did not hold inherited output descriptors: %v; output=%q", err, output.String())
	}
	time.Sleep(2 * time.Second)
	if _, err := os.Lstat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("descendant survived process-tree close after attach failure: %v", err)
	}
}
