// Command run-safe executes one external command with a bounded process tree,
// independently bounded output streams, and closed success-output semantics.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/RamXX/machinery/internal/processcontrol"
)

const (
	defaultStreamLimit = 1 << 20
	maximumStreamLimit = 64 << 20
	maximumTimeout     = 30 * time.Minute
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "snapshot-executable" {
		return snapshotExecutableCommand(args[1:], stderr)
	}
	if len(args) > 0 && args[0] == "verify-executable" {
		return verifyExecutableCommand(args[1:], stderr)
	}
	flags := flag.NewFlagSet("run-safe", flag.ContinueOnError)
	flags.SetOutput(stderr)
	timeout := flags.Duration("timeout", 2*time.Minute, "complete command-tree timeout")
	stdoutLimit := flags.Int("stdout-limit", defaultStreamLimit, "maximum captured stdout bytes")
	stderrLimit := flags.Int("stderr-limit", defaultStreamLimit, "maximum captured stderr bytes")
	expectStdout := flags.String("expect-stdout-file", "", "file containing the exact permitted successful stdout")
	expectStderr := flags.String("expect-stderr-file", "", "file containing the exact permitted successful stderr")
	executableReceiptPath := flags.String("executable-receipt", "", "receipt binding command[0] to an immutable executable snapshot")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	command := flags.Args()
	if len(command) == 0 || *timeout <= 0 || *timeout > maximumTimeout ||
		*stdoutLimit <= 0 || *stdoutLimit > maximumStreamLimit || *stderrLimit <= 0 || *stderrLimit > maximumStreamLimit {
		fmt.Fprintln(stderr, "run-safe: require a command, timeout in (0,30m], and stream limits in (0,64MiB]")
		return 2
	}

	wantStdout, err := readExpected(*expectStdout, *stdoutLimit, "stdout")
	if err != nil {
		fmt.Fprintf(stderr, "run-safe: %v\n", err)
		return 1
	}
	wantStderr, err := readExpected(*expectStderr, *stderrLimit, "stderr")
	if err != nil {
		fmt.Fprintf(stderr, "run-safe: %v\n", err)
		return 1
	}
	var executableReceipt *executableSnapshotReceipt
	if *executableReceiptPath != "" {
		executableReceipt, err = loadExecutableReceipt(*executableReceiptPath)
		if err != nil {
			fmt.Fprintf(stderr, "run-safe: load executable receipt: %v\n", err)
			return 1
		}
		commandPath, pathErr := filepath.Abs(command[0])
		if pathErr != nil || commandPath != executableReceipt.Snapshot {
			fmt.Fprintf(stderr, "run-safe: command executable does not match receipt snapshot: %v\n", pathErr)
			return 1
		}
		if err := executableReceipt.validateSnapshot(); err != nil {
			fmt.Fprintf(stderr, "run-safe: executable snapshot preflight failed: %v\n", err)
			return 1
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	commandStdout, commandStderr, runErr := processcontrol.RunCapturedStreamLimits(ctx, cmd, *stdoutLimit, *stderrLimit)
	if executableReceipt != nil {
		runErr = errors.Join(runErr, executableReceipt.validateSnapshot())
	}
	if ctx.Err() != nil {
		writeCaptured(stdout, stderr, commandStdout, commandStderr)
		fmt.Fprintf(stderr, "run-safe: command timed out after %s\n", timeout.String())
		return 124
	}
	if runErr != nil {
		writeCaptured(stdout, stderr, commandStdout, commandStderr)
		fmt.Fprintf(stderr, "run-safe: command failed: %v\n", runErr)
		return 1
	}
	if wantStdout != nil && !bytes.Equal([]byte(commandStdout), wantStdout) {
		_, _ = io.WriteString(stdout, commandStdout)
		fmt.Fprintln(stderr, "run-safe: successful command stdout did not exactly match its permitted receipt")
		return 1
	}
	if wantStderr == nil && commandStderr != "" {
		_, _ = io.WriteString(stderr, commandStderr)
		fmt.Fprintln(stderr, "run-safe: successful command emitted stderr")
		return 1
	}
	if wantStderr != nil && !bytes.Equal([]byte(commandStderr), wantStderr) {
		_, _ = io.WriteString(stderr, commandStderr)
		fmt.Fprintln(stderr, "run-safe: successful command stderr did not exactly match its permitted receipt")
		return 1
	}
	if _, err := io.WriteString(stdout, commandStdout); err != nil {
		fmt.Fprintf(stderr, "run-safe: write stdout: %v\n", err)
		return 1
	}
	if _, err := io.WriteString(stderr, commandStderr); err != nil {
		fmt.Fprintf(stderr, "run-safe: write stderr: %v\n", err)
		return 1
	}
	return 0
}

func readExpected(path string, limit int, stream string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve expected %s receipt: %w", stream, err)
	}
	before, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("inspect expected %s receipt: %w", stream, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() > int64(limit) {
		return nil, fmt.Errorf("expected %s receipt must be a regular non-symlink file no larger than its stream limit", stream)
	}
	file, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("open expected %s receipt: %w", stream, err)
	}
	opened, statErr := file.Stat()
	body, readErr := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	closeErr := file.Close()
	after, pathErr := os.Lstat(abs)
	if err := errors.Join(statErr, readErr, closeErr, pathErr); err != nil {
		return nil, fmt.Errorf("read expected %s receipt: %w", stream, err)
	}
	if len(body) > limit || !os.SameFile(before, opened) || !os.SameFile(before, after) ||
		opened.Mode() != before.Mode() || after.Mode() != before.Mode() || opened.Size() != before.Size() ||
		after.Size() != before.Size() || !opened.ModTime().Equal(before.ModTime()) || !after.ModTime().Equal(before.ModTime()) {
		return nil, fmt.Errorf("expected %s receipt changed identity, metadata, or size while reading", stream)
	}
	return body, nil
}

func writeCaptured(stdout, stderr io.Writer, commandStdout, commandStderr string) {
	_, _ = io.WriteString(stdout, commandStdout)
	_, _ = io.WriteString(stderr, commandStderr)
}
