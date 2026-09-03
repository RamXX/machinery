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
	diagnosticProcessFloodToken      = "machinery-diagnostic-process-flood"
	diagnosticProcessDescendantToken = "machinery-diagnostic-process-descendant"
	diagnosticProcessHoldToken       = "machinery-diagnostic-process-hold"
	diagnosticProcessWarningToken    = "machinery-diagnostic-process-warning"
)

func TestRunCommandBoundsInfiniteOutputAndTimeout(t *testing.T) {
	setDiagnosticProcessTestBounds(t, 100*time.Millisecond, 100*time.Millisecond)
	output, err := runCommand(os.Args[0], true, "-test.run=^TestDiagnosticProcessHelper$", "--", diagnosticProcessFloodToken)
	if err == nil || !strings.Contains(err.Error(), "timed out after 100ms") || !strings.Contains(err.Error(), "process tree was terminated") {
		t.Fatalf("unbounded diagnostic process diagnostic = %v", err)
	}
	wantSuffix := fmt.Sprintf("\n[output truncated at %d bytes]\n", diagnosticCommandOutputLimit)
	if !strings.HasSuffix(output, wantSuffix) || len(output) != diagnosticCommandOutputLimit+len(wantSuffix) {
		t.Fatalf("bounded diagnostic output length/suffix = %d/%q", len(output), output[len(output)-min(len(output), 80):])
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
	case diagnosticProcessFloodToken:
		block := []byte(strings.Repeat("x", 32*1024))
		for {
			if _, err := os.Stdout.Write(block); err != nil {
				os.Exit(0)
			}
			if _, err := os.Stderr.Write(block); err != nil {
				os.Exit(0)
			}
		}
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
