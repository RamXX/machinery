package install

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
	pluginProcessFloodToken      = "machinery-install-process-flood"
	pluginProcessDescendantToken = "machinery-install-process-descendant"
	pluginProcessHoldToken       = "machinery-install-process-hold"
	pluginProcessWarningToken    = "machinery-install-process-warning"
)

// The flood helper writes forever, so the deadline always fires; it only has
// to be long enough for the child to overrun the 1 MiB capture ceiling on a
// loaded CI runner under the race detector (100ms was not: the suite has
// failed at 720896 bytes, and at exactly the ceiling with no marker yet).
const pluginFloodTimeout = time.Second

func TestRunCombinedBoundsInfiniteOutputAndTimeout(t *testing.T) {
	setPluginProcessTestBounds(t, pluginFloodTimeout, 100*time.Millisecond)
	output, err := runCombined(os.Args[0], "-test.run=^TestPluginProcessHelper$", "--", pluginProcessFloodToken)
	if err == nil || !strings.Contains(err.Error(), "timed out after "+pluginFloodTimeout.String()) || !strings.Contains(err.Error(), "process tree was terminated") {
		t.Fatalf("unbounded plugin process diagnostic = %v", err)
	}
	wantSuffix := fmt.Sprintf("\n[output truncated at %d bytes]\n", pluginCommandOutputLimit)
	if !strings.HasSuffix(output, wantSuffix) || len(output) != pluginCommandOutputLimit+len(wantSuffix) {
		t.Fatalf("bounded plugin output length/suffix = %d/%q", len(output), output[len(output)-min(len(output), 80):])
	}
}

func TestRunCombinedRejectsSuccessfulStderr(t *testing.T) {
	output, err := runCombined(os.Args[0], "-test.run=^TestPluginProcessHelper$", "--", pluginProcessWarningToken)
	if output != "canonical stdout\n" {
		t.Fatalf("stdout = %q", output)
	}
	if err == nil || !strings.Contains(err.Error(), `wrote to stderr: "warning on stderr\n"`) {
		t.Fatalf("successful warning diagnostic = %v", err)
	}
}

func TestRunCombinedBoundsDescendantPipeRetention(t *testing.T) {
	setPluginProcessTestBounds(t, 2*time.Second, 100*time.Millisecond)
	started := time.Now()
	_, err := runCombined(os.Args[0], "-test.run=^TestPluginProcessHelper$", "--", pluginProcessDescendantToken)
	if err == nil || !strings.Contains(err.Error(), "descendant held output pipes open beyond 100ms") || !strings.Contains(err.Error(), "process tree was terminated") {
		t.Fatalf("plugin descendant diagnostic = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("plugin descendant retained pipes for %s", elapsed)
	}
}

func TestRunCombinedWithInstallLockCapabilityBoundsOutput(t *testing.T) {
	setPluginProcessTestBounds(t, pluginFloodTimeout, 100*time.Millisecond)
	run := runCombinedWithInstallLockCapability(installLockCapability{path: "fixture", token: "fixture", pid: os.Getpid()})
	output, err := run(os.Args[0], "-test.run=^TestPluginProcessHelper$", "--", pluginProcessFloodToken)
	if err == nil || !strings.Contains(err.Error(), "timed out after "+pluginFloodTimeout.String()) {
		t.Fatalf("delegated plugin process diagnostic = %v", err)
	}
	wantSuffix := fmt.Sprintf("\n[output truncated at %d bytes]\n", pluginCommandOutputLimit)
	if !strings.HasSuffix(output, wantSuffix) || len(output) != pluginCommandOutputLimit+len(wantSuffix) {
		t.Fatalf("delegated plugin output length/suffix = %d/%q", len(output), output[len(output)-min(len(output), 80):])
	}
}

func setPluginProcessTestBounds(t *testing.T, timeout, waitDelay time.Duration) {
	t.Helper()
	oldTimeout, oldWait := pluginCommandTimeout, pluginCommandWaitDelay
	pluginCommandTimeout = timeout
	pluginCommandWaitDelay = waitDelay
	t.Cleanup(func() {
		pluginCommandTimeout = oldTimeout
		pluginCommandWaitDelay = oldWait
	})
}

func TestPluginProcessHelper(t *testing.T) {
	if len(os.Args) == 0 {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case pluginProcessFloodToken:
		block := []byte(strings.Repeat("x", 32*1024))
		for {
			if _, err := os.Stdout.Write(block); err != nil {
				os.Exit(0)
			}
			if _, err := os.Stderr.Write(block); err != nil {
				os.Exit(0)
			}
		}
	case pluginProcessDescendantToken:
		command := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestPluginProcessHelper$", "--", pluginProcessHoldToken)
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Start(); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	case pluginProcessHoldToken:
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case pluginProcessWarningToken:
		_, _ = fmt.Fprintln(os.Stdout, "canonical stdout")
		_, _ = fmt.Fprintln(os.Stderr, "warning on stderr")
		os.Exit(0)
	}
}
