package runtimeclosure

import (
	"context"
	"errors"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/tdd"
)

// The exact approved Go closure of the first-release catalog
// (docs/test-assurance-contract.md section 6): go-testing/v1 executes only
// under Go 1.27.1 on the pinned native platforms. This file owns that
// RuntimeHandle; there is no user-selectable adapter runtime API.
const (
	// GoProfile is the runtime profile identity of the Go toolchain closure.
	GoProfile = "go"
	// RequiredGoVersion is the exact first-release Go toolchain version.
	RequiredGoVersion = "1.27.1"
)

// ErrGoClosureNotImplemented is the RED-phase placeholder returned while the
// MAC-wi2u GREEN implementation is pending; every frozen test that needs a
// real handle fails on it instead of silently passing.
var ErrGoClosureNotImplemented = errors.New("UNSUPPORTED_VERSION: the pinned Go " + RequiredGoVersion + " runtime closure is not implemented yet")

// GoRequest is the closed open request of the pinned Go runtime closure.
type GoRequest struct {
	// RuntimeRoot names the GOROOT directory that owns bin/go. Empty
	// resolves the host toolchain from PATH through its real symlink chain.
	RuntimeRoot string
	// ExpectedClosure, when nonempty, is an exact lowercase sha256 the
	// opened closure must reproduce; a mismatch fails closed.
	ExpectedClosure string
}

// Go is the pinned Go 1.27.1 runtime handle of tdd.RuntimeHandle.
type Go struct{ pending struct{} }

// Compile-time contract check: the closed RuntimeHandle boundary.
var _ tdd.RuntimeHandle = (*Go)(nil)

// OpenGo opens and binds the pinned Go runtime closure without launching any
// process: identity is pre-bound from the toolchain's own VERSION file and
// the complete GOROOT tree is fingerprinted.
func OpenGo(ctx context.Context, req GoRequest) (*Go, error) {
	return nil, ErrGoClosureNotImplemented
}

// Identity returns the pinned runtime identity of the opened closure.
func (g *Go) Identity() tdd.RuntimeRef { return tdd.RuntimeRef{} }

// Binary returns the resolved absolute go executable of the closure.
func (g *Go) Binary() string { return "" }

// Validate performs the native identity probes (go version, go env GOROOT)
// under the supplied live scope and revalidates the closure bytes.
func (g *Go) Validate(ctx context.Context, scope processscope.Scope) error {
	return ErrGoClosureNotImplemented
}

// Close performs the final process-free byte/topology/root-identity
// revalidation and then releases the retained handles.
func (g *Go) Close() error { return ErrGoClosureNotImplemented }
