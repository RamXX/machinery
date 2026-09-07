package runtimeclosure

// The exact approved CPython runtime closure of the first-release catalog
// (docs/test-assurance-contract.md section 6): python-unittest/v1 executes
// only under CPython 3.14.7 on the pinned native platforms darwin/arm64 and
// linux/amd64. OpenPython binds the complete closure by exact bytes without
// launching anything: the interpreter executable, the interpreter's own
// stdlib library tree and the static patchlevel.h version identity.
// Validate performs the runtime's own identity probe (python --version)
// under a live processscope scope; Close is the final process-free
// byte/topology/root-identity revalidation. There is no user-selectable
// adapter runtime API.

import (
	"context"
	"errors"
	"fmt"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/tdd"
)

const (
	// PythonProfile is the runtime profile identity of the
	// python-unittest/v1 closure.
	PythonProfile = pythonRuntimeProfile
	// RequiredPythonVersion is the exact first-release CPython version.
	RequiredPythonVersion = "3.14.7"

	pythonClosureDomain = "machinery.runtime.python/v1"
)

// pythonRuntimeProfile is unexported so no other package can conjure the
// identity without this file's pinned constants.
const pythonRuntimeProfile = "python-unittest/v1"

// ErrPythonClosureNotImplemented is the RED-phase placeholder returned
// before the pinned closure is implemented.
var ErrPythonClosureNotImplemented = errors.New("UNSUPPORTED_VERSION: the pinned CPython " + RequiredPythonVersion + " runtime closure is not implemented yet")

// PythonRequest is the closed open request of the pinned CPython runtime
// closure.
type PythonRequest struct {
	// PythonPath optionally names the python3 interpreter. Empty resolves
	// the host runtime from PATH through its real symlink chain.
	PythonPath string
	// ExpectedClosure, when nonempty, is an exact lowercase sha256 the
	// opened closure must reproduce; a mismatch fails closed.
	ExpectedClosure string
}

// Python is the pinned CPython 3.14.7 runtime handle implementing
// tdd.RuntimeHandle.
type Python struct{}

// Compile-time contract check: the closed RuntimeHandle boundary.
var _ tdd.RuntimeHandle = (*Python)(nil)

// OpenPython opens and binds the pinned CPython runtime closure. It never
// launches a process.
func OpenPython(ctx context.Context, req PythonRequest) (*Python, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("UNSUPPORTED_VERSION: cannot open the pinned CPython %s runtime closure: %w", RequiredPythonVersion, err)
	}
	return nil, ErrPythonClosureNotImplemented
}

// Identity implements tdd.RuntimeHandle.
func (py *Python) Identity() tdd.RuntimeRef { return tdd.RuntimeRef{} }

// Binary returns the resolved absolute python3 interpreter of the closure.
func (py *Python) Binary() string { return "" }

// StdlibRoot returns the resolved absolute stdlib library root.
func (py *Python) StdlibRoot() string { return "" }

// Validate implements tdd.RuntimeHandle.
func (py *Python) Validate(ctx context.Context, scope processscope.Scope) error {
	return ErrPythonClosureNotImplemented
}

// Close implements tdd.RuntimeHandle.
func (py *Python) Close() error { return ErrPythonClosureNotImplemented }
