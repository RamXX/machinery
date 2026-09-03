//go:build !windows

package processcontrol

import (
	"bytes"
	"context"
	"os"
	"os/exec"
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
