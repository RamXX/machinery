package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// indirections so tests/the real main can observe exit + stderr (and feed stdin).
var (
	stdinR  io.Reader = os.Stdin
	stdoutW io.Writer = os.Stdout
	stderrW io.Writer = os.Stderr
)

// exitStatusError carries a command's requested process status back through
// Cobra without terminating the process.  The cause is a diagnostic the
// command has already rendered; keeping it attached preserves errors.Is/As
// for direct callers while allowing the outermost runner to avoid printing it
// twice.  Only main is allowed to translate this value into os.Exit.
type exitStatusError struct {
	code  int
	cause error
}

func (e *exitStatusError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return fmt.Sprintf("command exited with status %d", e.code)
}

func (e *exitStatusError) Unwrap() error { return e.cause }

func commandExit(code int) error { return commandExitBecause(code, nil) }

func commandExitBecause(code int, cause error) error {
	if code <= 0 {
		panic("command exit status must be positive")
	}
	return &exitStatusError{code: code, cause: cause}
}

// commandResult separates already-rendered command failures from independent
// errors joined by deferred cleanup/revalidation.  The latter must still be
// surfaced after Cobra has unwound.  When several statuses are joined, the
// first one in deterministic error-tree order wins.
func commandResult(err error) (code int, remaining error) {
	if err == nil {
		return 0, nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var residuals []error
		for _, child := range joined.Unwrap() {
			childCode, childRemaining := commandResult(child)
			if code == 0 && childCode != 0 {
				code = childCode
			}
			if childRemaining != nil {
				residuals = append(residuals, childRemaining)
			}
		}
		return code, errors.Join(residuals...)
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		childCode, childRemaining := commandResult(wrapped.Unwrap())
		if childCode != 0 {
			return childCode, childRemaining
		}
	}
	var status *exitStatusError
	if errors.As(err, &status) {
		return status.code, nil
	}
	return 0, err
}

// trackedOutput preserves an output error until a command's RunE returns.
// fmt helpers and io.Copy callers are commonly written for an io.Writer and
// otherwise make it easy to discard a short/broken stdout or stderr write.
type trackedOutput struct {
	name string
	dest io.Writer
	err  error
}

func (w *trackedOutput) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.dest.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.err = fmt.Errorf("write machinery %s: %w", w.name, err)
		return n, w.err
	}
	return n, nil
}

type commandOutput struct {
	stdout *trackedOutput
	stderr *trackedOutput
}

func trackCommandOutput() *commandOutput {
	return trackOutput(stdoutW, stderrW)
}

func trackOutput(stdout, stderr io.Writer) *commandOutput {
	return &commandOutput{
		stdout: &trackedOutput{name: "stdout", dest: stdout},
		stderr: &trackedOutput{name: "stderr", dest: stderr},
	}
}

func (o *commandOutput) join(err error) error {
	return errors.Join(err, o.stdout.err, o.stderr.err)
}
