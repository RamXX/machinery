// Scoped custody bindings for subprocess execution: a verified processscope
// custody scope can be carried by the execution context (WithScope) or bound
// to one command (AttachScope), and Run then executes that command as an
// owned, guarded broker job instead of an unscoped direct child. Scoped runs
// never fall back to unscoped execution: malformed input fails closed before
// any dispatch.

package processcontrol

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
)

// scopeContextKey namespaces the custody scope carried by an execution
// context.
type scopeContextKey struct{}

// scopedChildCloseTimeout bounds the per-run child-scope close wait. The
// broker independently enforces its single shared cleanup grace; this only
// keeps a wedged control channel from hanging the caller forever.
const scopedChildCloseTimeout = 45 * time.Second

// WithScope returns a context that carries scope as the verified custody
// authority for subprocesses launched under it. Both arguments are
// mandatory: a nil context or nil scope is a programming error and panics,
// matching context.WithValue's contract for nil parents.
func WithScope(ctx context.Context, scope processscope.Scope) context.Context {
	if ctx == nil {
		panic("processcontrol: WithScope requires a non-nil context")
	}
	if scope == nil {
		panic("processcontrol: WithScope requires a non-nil custody scope")
	}
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

// ScopeFromContext returns the custody scope carried by ctx, or nil when the
// context carries no scope.
func ScopeFromContext(ctx context.Context) processscope.Scope {
	if ctx == nil {
		return nil
	}
	scope, _ := ctx.Value(scopeContextKey{}).(processscope.Scope)
	return scope
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

// scopedCancelError preserves errors.Is cancellation semantics across the
// broker boundary while keeping the originating custody error inspectable.
type scopedCancelError struct {
	cause  error
	origin error
}

func (e *scopedCancelError) Error() string {
	return fmt.Sprintf("scoped subprocess canceled by custody scope: %v", e.origin)
}

func (e *scopedCancelError) Unwrap() []error {
	if e.cause == nil {
		return []error{e.origin}
	}
	return []error{e.cause, e.origin}
}

// ExitStatus is the approved accessor for target exit classification across
// both execution paths: it reports the exit code, the death signal when the
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

// scopedBinding is one command's verified custody attachment.
type scopedBinding struct {
	scope      processscope.Scope
	executable string
	digest     string
}

// scopedBindings associates explicitly attached commands with their scope.
// Entries are consumed exactly once by Run.
var scopedBindings sync.Map // *exec.Cmd -> scopedBinding

// AttachScope binds scope to cmd so the next Run of cmd executes it under
// verified native custody. Malformed commands fail closed before any launch:
// custom inherited file descriptors and any preset SysProcAttr conflict with
// guarded-group custody and are rejected explicitly, never silently reduced.
// The runtime identity digest is computed from the resolved executable.
func AttachScope(cmd *exec.Cmd, scope processscope.Scope) error {
	binding, err := newScopedBinding(cmd, scope)
	if err != nil {
		return err
	}
	if prev, ok := scopedBindings.Load(cmd); ok {
		old := prev.(scopedBinding)
		if !sameScope(old.scope, binding.scope) || old.executable != binding.executable {
			return fmt.Errorf("processcontrol.AttachScope: command is already bound to a different custody attachment")
		}
		return nil
	}
	scopedBindings.Store(cmd, binding)
	return nil
}

func newScopedBinding(cmd *exec.Cmd, scope processscope.Scope) (scopedBinding, error) {
	if cmd == nil {
		return scopedBinding{}, fmt.Errorf("processcontrol.AttachScope: nil command")
	}
	if scope == nil {
		return scopedBinding{}, fmt.Errorf("processcontrol.AttachScope: nil custody scope")
	}
	if len(cmd.ExtraFiles) != 0 {
		return scopedBinding{}, fmt.Errorf("processcontrol.AttachScope: custom inherited file descriptors are unsupported in scoped execution (ExtraFiles set)")
	}
	if cmd.SysProcAttr != nil {
		return scopedBinding{}, fmt.Errorf("processcontrol.AttachScope: conflicting SysProcAttr is unsupported in scoped execution")
	}
	exe, err := resolveScopedExecutable(cmd)
	if err != nil {
		return scopedBinding{}, err
	}
	digest, err := scopedExecutableDigest(exe)
	if err != nil {
		return scopedBinding{}, fmt.Errorf("processcontrol.AttachScope: runtime identity for %s: %w", exe, err)
	}
	return scopedBinding{scope: scope, executable: exe, digest: digest}, nil
}

func resolveScopedExecutable(cmd *exec.Cmd) (string, error) {
	name := cmd.Path
	if name == "" && len(cmd.Args) > 0 {
		name = cmd.Args[0]
	}
	if name == "" {
		return "", fmt.Errorf("processcontrol.AttachScope: command has no executable")
	}
	if !filepath.IsAbs(name) {
		if strings.ContainsRune(name, filepath.Separator) {
			abs, err := filepath.Abs(name)
			if err != nil {
				return "", fmt.Errorf("processcontrol.AttachScope: resolve executable: %w", err)
			}
			name = abs
		} else {
			resolved, err := exec.LookPath(name)
			if err != nil {
				return "", fmt.Errorf("processcontrol.AttachScope: resolve executable: %w", err)
			}
			name = resolved
		}
	}
	st, err := os.Stat(name)
	if err != nil {
		return "", fmt.Errorf("processcontrol.AttachScope: inspect executable: %w", err)
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("processcontrol.AttachScope: %s must be a regular executable file", name)
	}
	return name, nil
}

func scopedExecutableDigest(exe string) (string, error) {
	f, err := os.Open(exe)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func sameScope(a, b processscope.Scope) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if ra.Type() != rb.Type() || !ra.Comparable() || !rb.Comparable() {
		return false
	}
	return ra.Equal(rb)
}

// takeScopedBinding consumes cmd's explicit attachment, if any.
func takeScopedBinding(cmd *exec.Cmd) (scopedBinding, bool) {
	if cmd == nil {
		return scopedBinding{}, false
	}
	if v, ok := scopedBindings.LoadAndDelete(cmd); ok {
		return v.(scopedBinding), true
	}
	return scopedBinding{}, false
}

// syncWriter serializes the broker's separate stdout/stderr pumps when a
// caller shares one writer across both streams.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

func sameWriter(a, b io.Writer) bool {
	if a == nil || b == nil {
		return false
	}
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if ra.Type() != rb.Type() || !ra.Comparable() || !rb.Comparable() {
		return false
	}
	return ra.Equal(rb)
}

// runScoped executes cmd as one owned job of the binding's custody scope.
// Each run uses a fresh logical child scope so completed jobs are retired and
// their cleanup verified before Run returns, while the wall budget stays the
// owner's single cumulative deadline.
func runScoped(ctx context.Context, cmd *exec.Cmd, binding scopedBinding) error {
	if len(cmd.ExtraFiles) != 0 {
		return fmt.Errorf("processcontrol: scoped execution rejects custom inherited file descriptors (ExtraFiles set)")
	}
	if cmd.SysProcAttr != nil {
		return fmt.Errorf("processcontrol: scoped execution rejects conflicting SysProcAttr")
	}
	if ctxScope := ScopeFromContext(ctx); ctxScope != nil && !sameScope(ctxScope, binding.scope) {
		return fmt.Errorf("processcontrol: custody mismatch between the command attachment and the execution context refuses scoped execution")
	}
	if cmd.Env == nil {
		return fmt.Errorf("processcontrol: scoped execution requires an explicitly declared command environment")
	}
	if cmd.Dir != "" && !filepath.IsAbs(cmd.Dir) {
		return fmt.Errorf("processcontrol: scoped execution requires an absolute working directory")
	}
	argv := cmd.Args
	if len(argv) == 0 {
		argv = []string{binding.executable}
	}
	child, err := binding.scope.Child(ctx)
	if err != nil {
		return fmt.Errorf("processcontrol: open scoped child custody: %w", err)
	}
	command := processscope.Command{
		Executable:    binding.executable,
		Args:          argv[1:],
		Dir:           cmd.Dir,
		Env:           cmd.Env,
		RuntimeDigest: binding.digest,
	}
	attached, err := child.Attach(command)
	if err != nil {
		return errors.Join(fmt.Errorf("processcontrol: attach scoped custody: %w", err), closeScopedChild(child))
	}
	stdout, stderr := cmd.Stdout, cmd.Stderr
	if sameWriter(stdout, stderr) {
		shared := &syncWriter{w: stdout}
		stdout, stderr = shared, shared
	}
	res, runErr := child.Run(ctx, attached, processscope.Streams{
		Stdin:  cmd.Stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	closeErr := closeScopedChild(child)
	if runErr != nil {
		var serr *processscope.Error
		if errors.As(runErr, &serr) {
			switch serr.Code {
			case processscope.CodeCanceled:
				runErr = &scopedCancelError{cause: context.Canceled, origin: serr}
			case processscope.CodeTimeout:
				runErr = &scopedCancelError{cause: context.DeadlineExceeded, origin: serr}
			}
		}
		return errors.Join(runErr, closeErr)
	}
	if res.Completed && (res.ExitCode != 0 || res.Signal != "") {
		return errors.Join(&TargetExitError{ExitCode: res.ExitCode, Signal: res.Signal}, closeErr)
	}
	return closeErr
}

func closeScopedChild(child processscope.Scope) error {
	ctx, cancel := context.WithTimeout(context.Background(), scopedChildCloseTimeout)
	defer cancel()
	rep, err := child.Close(ctx)
	if err != nil {
		return fmt.Errorf("processcontrol: close scoped child custody: %w", err)
	}
	if rep.Status != processscope.StatusCleaned {
		return fmt.Errorf("processcontrol: scoped child cleanup status %s: %+v", rep.Status, rep)
	}
	return nil
}
