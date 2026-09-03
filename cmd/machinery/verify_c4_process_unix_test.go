//go:build !windows

package main

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

func TestRunC4ProcessKillsForkedChild(t *testing.T) {
	old := verifyC4Timeout
	verifyC4Timeout = 100 * time.Millisecond
	t.Cleanup(func() { verifyC4Timeout = old })
	pidFile := t.TempDir() + "/child.pid"
	ctx, cancel := context.WithTimeout(context.Background(), verifyC4Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `sleep 30 & echo $! > "$1"; wait`, "sh", pidFile)
	if _, err := runC4Process(ctx, cmd, verifyC4Timeout); err == nil {
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
