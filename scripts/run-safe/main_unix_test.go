//go:build !windows

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunTimeoutKillsForkedDescendant(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	fake := fakeCommand(t, "modelith", "sleep 30 &\nprintf '%s\\n' \"$!\" >\"$MACHINERY_TEST_CHILD_PID\"\nwait\n")
	t.Setenv("MACHINERY_TEST_CHILD_PID", pidFile)
	var stdout, stderr bytes.Buffer
	started := time.Now()
	status := run([]string{"-timeout", "3s", "--", fake, "render"}, &stdout, &stderr)
	if status != 124 || !strings.Contains(stderr.String(), "timed out after 3s") {
		t.Fatalf("timeout status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("bounded command returned after %s", elapsed)
	}
	body, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read descendant PID: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = exec.CommandContext(ctx, "kill", "-0", strconv.Itoa(pid)).Run()
		cancel()
		if err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("forked command descendant %d survived timeout", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
