// Scoped custody bindings for subprocess execution: a verified processscope
// custody scope can be carried by the execution context (WithScope) or bound
// to one command (AttachScope), and Run then executes that command as an
// owned, guarded broker job instead of an unscoped direct child.

package processcontrol

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/processscope"
)

// scopeContextKey namespaces the custody scope carried by an execution
// context.
type scopeContextKey struct{}

// WithScope returns a context that carries scope as the verified custody
// authority for every subprocess launched under it. Both arguments are
// mandatory: a nil context or nil scope is a programming error and panics,
// matching context.WithValue's contract for nil parents.
func WithScope(ctx context.Context, scope processscope.Scope) context.Context {
	if ctx == nil {
		panic("processcontrol: WithScope requires a non-nil context")
	}
	if scope == nil {
		panic("processcontrol: WithScope requires a non-nil custody scope")
	}
	return ctx // RED: scope carriage is not wired; absence stays explicit.
}

// ScopeFromContext returns the custody scope carried by ctx, or nil when the
// context carries no scope.
func ScopeFromContext(ctx context.Context) processscope.Scope {
	return nil // RED: no scope is ever carried.
}

// TargetExitError reports that a scoped target process ran to a terminal
// outcome: it exited with a nonzero status, or died on a signal. It is the
// scoped counterpart of *exec.ExitError and is inspectable through
// ExitStatus. Cancellation and custody failures are never represented as
// target exits.
type TargetExitError struct {
	ExitCode int
	Signal   string
}

func (e *TargetExitError) Error() string {
	if e.Signal != "" {
		return fmt.Sprintf("scoped subprocess terminated by %s", e.Signal)
	}
	return fmt.Sprintf("scoped subprocess exited with status %d", e.ExitCode)
}

// ExitStatus is the approved accessor for target exit classification across
// both execution paths: it reports the exit code, death signal when the
// target died by signal, and whether the error represents a terminal target
// outcome at all. Ordinary (unscoped) runs surface *exec.ExitError; scoped
// runs surface *TargetExitError; cancellation, custody, and launch failures
// report exited=false.
func ExitStatus(err error) (code int, signal string, exited bool) {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if st := ee.ProcessState; st != nil {
			return st.ExitCode(), "", true
		}
		return -1, "", true
	}
	var te *TargetExitError
	if errors.As(err, &te) {
		return te.ExitCode, te.Signal, true
	}
	return 0, "", false
}

var errScopedCustodyUnimplemented = errors.New("processcontrol: custody scope attachment is not implemented in this build")

// AttachScope binds scope to cmd so the next Run of cmd executes it under
// verified native custody. Malformed commands fail closed before any launch:
// custom inherited file descriptors and any preset SysProcAttr conflict with
// guarded-group custody and are rejected explicitly, never silently reduced.
func AttachScope(cmd *exec.Cmd, scope processscope.Scope) error {
	if cmd == nil {
		return fmt.Errorf("processcontrol.AttachScope: nil command")
	}
	if scope == nil {
		return fmt.Errorf("processcontrol.AttachScope: nil custody scope")
	}
	if len(cmd.ExtraFiles) != 0 {
		return fmt.Errorf("processcontrol.AttachScope: custom inherited file descriptors are unsupported in scoped execution (ExtraFiles set)")
	}
	if cmd.SysProcAttr != nil {
		return fmt.Errorf("processcontrol.AttachScope: conflicting SysProcAttr is unsupported in scoped execution")
	}
	name := cmd.Path
	if name == "" && len(cmd.Args) > 0 {
		name = cmd.Args[0]
	}
	if name == "" {
		return fmt.Errorf("processcontrol.AttachScope: command has no executable")
	}
	if !filepath.IsAbs(name) && !strings.ContainsRune(name, filepath.Separator) {
		if _, err := exec.LookPath(name); err != nil {
			return fmt.Errorf("processcontrol.AttachScope: resolve executable: %w", err)
		}
	}
	return errScopedCustodyUnimplemented
}
