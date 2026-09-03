package processcontrol

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
)

// RunCaptured runs cmd with bounded stdout capture. The limit is deliberately
// supplied by each caller because a compiler probe and a plugin inventory have
// different useful output sizes.
// RunCaptured always reports truncation while continuing to drain the pipe so
// a noisy child cannot deadlock on a full pipe or grow the parent unboundedly.
func RunCaptured(ctx context.Context, cmd *exec.Cmd, limit int, combined bool) (string, error) {
	if limit <= 0 {
		return "", fmt.Errorf("process output limit must be positive")
	}
	capture := &boundedCapture{limit: limit}
	cmd.Stdout = capture
	if combined {
		cmd.Stderr = capture
	}
	err := Run(ctx, cmd)
	if capture.Truncated() {
		err = errors.Join(err, fmt.Errorf("process output exceeded %d-byte capture limit", limit))
	}
	return capture.String(), err
}

// RunCapturedStreams runs cmd with independently bounded stdout and stderr
// capture. Keeping the streams separate lets callers reject diagnostics on
// stderr even when the child exits successfully, without giving either stream
// an unbounded pipe or memory budget.
func RunCapturedStreams(ctx context.Context, cmd *exec.Cmd, limit int) (stdout, stderr string, err error) {
	return RunCapturedStreamLimits(ctx, cmd, limit, limit)
}

// RunCapturedStreamLimits runs cmd with separately configured stdout and
// stderr limits. Both streams continue to drain after reaching their limit so
// a noisy process cannot deadlock while process-tree cancellation completes.
func RunCapturedStreamLimits(ctx context.Context, cmd *exec.Cmd, stdoutLimit, stderrLimit int) (stdout, stderr string, err error) {
	if stdoutLimit <= 0 || stderrLimit <= 0 {
		return "", "", fmt.Errorf("process stdout and stderr limits must be positive")
	}
	stdoutCapture := &boundedCapture{limit: stdoutLimit}
	stderrCapture := &boundedCapture{limit: stderrLimit}
	cmd.Stdout = stdoutCapture
	cmd.Stderr = stderrCapture
	err = Run(ctx, cmd)
	if stdoutCapture.Truncated() {
		err = errors.Join(err, fmt.Errorf("process stdout exceeded %d-byte capture limit", stdoutLimit))
	}
	if stderrCapture.Truncated() {
		err = errors.Join(err, fmt.Errorf("process stderr exceeded %d-byte capture limit", stderrLimit))
	}
	return stdoutCapture.String(), stderrCapture.String(), err
}

type boundedCapture struct {
	mu        sync.Mutex
	data      []byte
	limit     int
	truncated bool
}

func (capture *boundedCapture) Write(p []byte) (int, error) {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	overflow := len(p) > capture.limit-len(capture.data)
	if room := capture.limit - len(capture.data); room > 0 {
		take := len(p)
		if take > room {
			take = room
		}
		capture.data = append(capture.data, p[:take]...)
	}
	if overflow {
		capture.truncated = true
	}
	return len(p), nil
}

func (capture *boundedCapture) String() string {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	output := string(capture.data)
	if capture.truncated {
		output += fmt.Sprintf("\n[output truncated at %d bytes]\n", capture.limit)
	}
	return output
}

func (capture *boundedCapture) Truncated() bool {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return capture.truncated
}
