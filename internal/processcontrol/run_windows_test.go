//go:build windows

package processcontrol

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestRunTerminatesSuspendedChildWhenOpenProcessFails(t *testing.T) {
	injectedErr := errors.New("injected OpenProcess failure")
	attachReached := make(chan struct{})
	closedHandles := countClosedWindowsHandles(t)
	original := openProcessForJob
	openProcessForJob = func(uint32) (syscall.Handle, error) {
		close(attachReached)
		return 0, injectedErr
	}
	t.Cleanup(func() { openProcessForJob = original })

	assertAttachFailureTerminatesSuspendedChild(t, attachReached, injectedErr)
	if got := closedHandles.Load(); got != 1 {
		t.Fatalf("closed native handles = %d, want job handle", got)
	}
}

func TestRunTerminatesSuspendedChildWhenJobAssignmentFails(t *testing.T) {
	injectedErr := errors.New("injected AssignProcessToJobObject failure")
	attachReached := make(chan struct{})
	closedHandles := countClosedWindowsHandles(t)
	original := assignProcessToJob
	assignProcessToJob = func(syscall.Handle, syscall.Handle) error {
		close(attachReached)
		return injectedErr
	}
	t.Cleanup(func() { assignProcessToJob = original })

	assertAttachFailureTerminatesSuspendedChild(t, attachReached, injectedErr)
	if got := closedHandles.Load(); got != 2 {
		t.Fatalf("closed native handles = %d, want process and job handles", got)
	}
}

func countClosedWindowsHandles(t *testing.T) *atomic.Int32 {
	t.Helper()
	var count atomic.Int32
	original := closeWindowsHandle
	closeWindowsHandle = func(handle syscall.Handle) error {
		count.Add(1)
		return original(handle)
	}
	t.Cleanup(func() { closeWindowsHandle = original })
	return &count
}

func assertAttachFailureTerminatesSuspendedChild(t *testing.T, attachReached <-chan struct{}, injectedErr error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWindowsAttachFailureChild$")
	cmd.Env = append(os.Environ(), "MACHINERY_WINDOWS_ATTACH_FAILURE_CHILD=1")
	done := make(chan error, 1)
	go func() { done <- Run(ctx, cmd) }()

	select {
	case <-attachReached:
	case <-time.After(5 * time.Second):
		t.Fatal("subprocess attach was not attempted")
	}

	select {
	case err := <-done:
		if !errors.Is(err, injectedErr) {
			t.Fatalf("Run error = %v, want injected attach error", err)
		}
		if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
			t.Fatalf("suspended child was not reaped: %#v", cmd.ProcessState)
		}
	case <-time.After(5 * time.Second):
		// Start completed before attachReached was closed, so Process is stable
		// here. Kill it to prevent a failed regression test from leaking the
		// suspended helper or blocking the package test indefinitely.
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		t.Fatal("Run blocked waiting for an unassigned suspended child")
	}
}

func TestWindowsAttachFailureChild(t *testing.T) {
	if os.Getenv("MACHINERY_WINDOWS_ATTACH_FAILURE_CHILD") != "1" {
		t.Skip("helper subprocess only")
	}
	// Run starts this test binary suspended. Reaching the body would mean the
	// attach-failure path accidentally resumed a child it does not control.
	t.Fatal("attach-failure child unexpectedly resumed")
}
