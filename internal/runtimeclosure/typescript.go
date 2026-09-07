package runtimeclosure

import (
	"context"
	"errors"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/tdd"
)

// The exact approved TypeScript closure of the first-release catalog
// (docs/test-assurance-contract.md section 6): node-test-typescript/v1
// executes only under Node 26.8.1 with the TypeScript compiler 7.0.2 on the
// pinned native platforms darwin/arm64 and linux/amd64. This file owns that
// RuntimeHandle; there is no user-selectable adapter runtime API.
const (
	// TypeScriptProfile is the runtime profile identity of the
	// node-test-typescript/v1 closure.
	TypeScriptProfile = tddRuntimeProfile
	// RequiredNodeVersion is the exact first-release Node runtime version.
	RequiredNodeVersion = "v26.8.1"
	// RequiredTypeScriptVersion is the exact first-release TypeScript
	// compiler version.
	RequiredTypeScriptVersion = "7.0.2"
	// TypeScriptIdentityVersion is the RuntimeRef version spelling of the
	// two-part pinned closure identity.
	TypeScriptIdentityVersion = "26.8.1/7.0.2"
)

// tddRuntimeProfile is unexported so no other package can conjure the
// identity without this file's pinned constants.
const tddRuntimeProfile = "node-test-typescript/v1"

// ErrTypeScriptClosureNotImplemented is the RED-phase placeholder returned
// while the MAC-avfp GREEN implementation is pending; every frozen test that
// needs a real handle fails on it instead of silently passing.
var ErrTypeScriptClosureNotImplemented = errors.New("UNSUPPORTED_VERSION: the pinned Node " + RequiredNodeVersion + " / TypeScript " + RequiredTypeScriptVersion + " runtime closure is not implemented yet")

// TypeScriptRequest is the closed open request of the pinned TypeScript
// runtime closure.
type TypeScriptRequest struct {
	// NodePath optionally names the node executable. Empty resolves the
	// host runtime from PATH through its real symlink chain.
	NodePath string
	// ExpectedClosure, when nonempty, is an exact lowercase sha256 the
	// opened closure must reproduce; a mismatch fails closed.
	ExpectedClosure string
}

// TypeScript is the pinned Node 26.8.1 / TypeScript 7.0.2 runtime handle of
// tdd.RuntimeHandle.
type TypeScript struct{ pending struct{} }

// Compile-time contract check: the closed RuntimeHandle boundary.
var _ tdd.RuntimeHandle = (*TypeScript)(nil)

// OpenTypeScript opens and binds the pinned TypeScript runtime closure: the
// node executable, the TypeScript package tree and the platform native
// compiler binary, all by exact bytes, without launching anything.
func OpenTypeScript(ctx context.Context, req TypeScriptRequest) (*TypeScript, error) {
	if err := ctx.Err(); err != nil {
		return nil, ErrTypeScriptClosureNotImplemented
	}
	return nil, ErrTypeScriptClosureNotImplemented
}

// Identity implements tdd.RuntimeHandle.
func (ts *TypeScript) Identity() tdd.RuntimeRef {
	return tdd.RuntimeRef{Profile: TypeScriptProfile, Version: TypeScriptIdentityVersion}
}

// NodePath returns the resolved real node executable path.
func (ts *TypeScript) NodePath() string { return "" }

// CompilerPath returns the resolved real native TypeScript compiler binary.
func (ts *TypeScript) CompilerPath() string { return "" }

// NodeDigest returns the sha256 of the verified node executable bytes.
func (ts *TypeScript) NodeDigest() string { return "" }

// CompilerDigest returns the sha256 of the verified native compiler bytes.
func (ts *TypeScript) CompilerDigest() string { return "" }

// Validate implements tdd.RuntimeHandle: it performs the runtime's own
// identity probes (node --version, the compiler's --version) as real
// subprocesses under the supplied live scope.
func (ts *TypeScript) Validate(ctx context.Context, scope processscope.Scope) error {
	return ErrTypeScriptClosureNotImplemented
}

// Close implements tdd.RuntimeHandle: the final process-free
// byte/topology/root-identity revalidation before releasing retained
// handles. It never launches a process.
func (ts *TypeScript) Close() error { return ErrTypeScriptClosureNotImplemented }
