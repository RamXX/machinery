//go:build !windows

package formal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRunBoundedProcessKillsForkedChild(t *testing.T) {
	pidFile := t.TempDir() + "/child.pid"
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `sleep 30 & echo $! > "$1"; wait`, "sh", pidFile)
	if _, err := runBoundedProcess(ctx, cmd, 100*time.Millisecond); err == nil {
		t.Fatal("forking process did not time out")
	}
	body, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err = syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("forked child %d survived process-tree cancellation: %v", pid, err)
}
