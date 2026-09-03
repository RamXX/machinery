package processcontrol

import (
	"context"
	"os"
	"os/exec"
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
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestRunCancelsDirectChild")
	cmd.Env = replaceEnv(os.Environ(), helperEnv, "sleep")
	if err := Run(ctx, cmd); err == nil {
		t.Fatal("timed-out subprocess returned success")
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
