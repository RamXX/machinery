package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	diagnosticProcessOverflowToken   = "machinery-diagnostic-process-overflow"
	diagnosticProcessDescendantToken = "machinery-diagnostic-process-descendant"
	diagnosticProcessHoldToken       = "machinery-diagnostic-process-hold"
	diagnosticProcessWarningToken    = "machinery-diagnostic-process-warning"
)

func TestRunCommandBoundsOutput(t *testing.T) {
	setDiagnosticProcessTestBounds(t, 5*time.Second, 100*time.Millisecond)
	output, err := runCommand(os.Args[0], true, "-test.run=^TestDiagnosticProcessHelper$", "--", diagnosticProcessOverflowToken)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("process output exceeded %d-byte capture limit", diagnosticCommandOutputLimit)) {
		t.Fatalf("overflow diagnostic = %v", err)
	}
	wantSuffix := fmt.Sprintf("\n[output truncated at %d bytes]\n", diagnosticCommandOutputLimit)
	if !strings.HasSuffix(output, wantSuffix) || len(output) != diagnosticCommandOutputLimit+len(wantSuffix) {
		t.Fatalf("bounded diagnostic output length/suffix = %d/%q", len(output), output[len(output)-min(len(output), 80):])
	}
}

func TestRunCommandBoundsTimeout(t *testing.T) {
	setDiagnosticProcessTestBounds(t, 100*time.Millisecond, 100*time.Millisecond)
	_, err := runCommand(os.Args[0], true, "-test.run=^TestDiagnosticProcessHelper$", "--", diagnosticProcessHoldToken)
	if err == nil || !strings.Contains(err.Error(), "timed out after 100ms") || !strings.Contains(err.Error(), "process tree was terminated") {
		t.Fatalf("timeout diagnostic = %v", err)
	}
}

func TestRunCommandRejectsSuccessfulStderr(t *testing.T) {
	output, err := runCommand(os.Args[0], false, "-test.run=^TestDiagnosticProcessHelper$", "--", diagnosticProcessWarningToken)
	if output != "canonical stdout\n" {
		t.Fatalf("stdout = %q", output)
	}
	if err == nil || !strings.Contains(err.Error(), `wrote to stderr: "warning on stderr\n"`) {
		t.Fatalf("successful warning diagnostic = %v", err)
	}
}

func TestRunCommandBoundsDescendantPipeRetention(t *testing.T) {
	setDiagnosticProcessTestBounds(t, 2*time.Second, 100*time.Millisecond)
	started := time.Now()
	_, err := runCommand(os.Args[0], false, "-test.run=^TestDiagnosticProcessHelper$", "--", diagnosticProcessDescendantToken)
	if err == nil || !strings.Contains(err.Error(), "descendant held output pipes open beyond 100ms") || !strings.Contains(err.Error(), "process tree was terminated") {
		t.Fatalf("diagnostic descendant diagnostic = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("diagnostic descendant retained pipes for %s", elapsed)
	}
}

func setDiagnosticProcessTestBounds(t *testing.T, timeout, waitDelay time.Duration) {
	t.Helper()
	oldTimeout, oldWait := diagnosticCommandTimeout, diagnosticCommandWaitDelay
	diagnosticCommandTimeout = timeout
	diagnosticCommandWaitDelay = waitDelay
	t.Cleanup(func() {
		diagnosticCommandTimeout = oldTimeout
		diagnosticCommandWaitDelay = oldWait
	})
}

func TestDiagnosticProcessHelper(t *testing.T) {
	if len(os.Args) == 0 {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case diagnosticProcessOverflowToken:
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", diagnosticCommandOutputLimit+1)))
		os.Exit(0)
	case diagnosticProcessDescendantToken:
		command := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestDiagnosticProcessHelper$", "--", diagnosticProcessHoldToken)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Start(); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	case diagnosticProcessHoldToken:
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case diagnosticProcessWarningToken:
		_, _ = fmt.Fprintln(os.Stdout, "canonical stdout")
		_, _ = fmt.Fprintln(os.Stderr, "warning on stderr")
		os.Exit(0)
	}
}
