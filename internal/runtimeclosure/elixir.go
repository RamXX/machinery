package runtimeclosure

import (
	"context"
	"errors"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/tdd"
)

// The exact approved Elixir closure of the first-release catalog
// (docs/test-assurance-contract.md section 6): elixir-exunit/v1 executes
// only under Elixir/ExUnit/Mix 1.20.4 with Erlang/OTP 29.0.6 (ERTS 17.0.6)
// on the pinned native platforms darwin/arm64 and linux/amd64. This file
// owns that RuntimeHandle; there is no user-selectable adapter runtime API.
const (
	// ElixirProfile is the runtime profile identity of the
	// elixir-exunit/v1 closure.
	ElixirProfile = elixirRuntimeProfile
	// RequiredElixirVersion is the exact first-release Elixir/Mix/ExUnit
	// version.
	RequiredElixirVersion = "1.20.4"
	// RequiredOTPVersion is the exact first-release OTP version.
	RequiredOTPVersion = "29.0.6"
	// RequiredErtsVersion is the exact first-release ERTS version.
	RequiredErtsVersion = "17.0.6"
	// ElixirIdentityVersion is the RuntimeRef version spelling of the
	// three-part pinned closure identity.
	ElixirIdentityVersion = "1.20.4/29.0.6/17.0.6"
)

// elixirRuntimeProfile is unexported so no other package can conjure the
// identity without this file's pinned constants.
const elixirRuntimeProfile = "elixir-exunit/v1"

// ErrElixirClosureNotImplemented is the RED-phase placeholder returned
// while the MAC-8yai GREEN implementation is pending; every frozen test
// that needs a real handle fails on it instead of silently passing.
var ErrElixirClosureNotImplemented = errors.New("UNSUPPORTED_VERSION: the pinned Elixir " + RequiredElixirVersion + " / OTP " + RequiredOTPVersion + " / ERTS " + RequiredErtsVersion + " runtime closure is not implemented yet")

// ElixirRequest is the closed open request of the pinned Elixir runtime
// closure.
type ElixirRequest struct {
	// ElixirPath optionally names the elixir launcher. Empty resolves the
	// host runtime from PATH through its real symlink chain.
	ElixirPath string
	// ErlangPath optionally names the erl launcher. Empty resolves the
	// host runtime from PATH through its real symlink chain.
	ErlangPath string
	// ExpectedClosure, when nonempty, is an exact lowercase sha256 the
	// opened closure must reproduce; a mismatch fails closed.
	ExpectedClosure string
}

// Elixir is the pinned Elixir 1.20.4 / OTP 29.0.6 (ERTS 17.0.6) runtime
// handle implementing tdd.RuntimeHandle.
type Elixir struct{ pending struct{} }

// Compile-time contract check: the closed RuntimeHandle boundary.
var _ tdd.RuntimeHandle = (*Elixir)(nil)

// OpenElixir opens and binds the pinned Elixir runtime closure: the Elixir
// launcher and its complete lib tree, the mix launcher, and the Erlang/OTP
// installation (launcher, ERTS and lib tree), all by exact bytes, without
// launching anything. The runtime version identity is proven by the scoped
// probes in Validate.
func OpenElixir(ctx context.Context, req ElixirRequest) (*Elixir, error) {
	return nil, ErrElixirClosureNotImplemented
}

// Identity implements tdd.RuntimeHandle.
func (e *Elixir) Identity() tdd.RuntimeRef { return tdd.RuntimeRef{} }

// ElixirPath returns the resolved real elixir launcher of the closure.
func (e *Elixir) ElixirPath() string { return "" }

// MixPath returns the resolved real mix launcher of the closure.
func (e *Elixir) MixPath() string { return "" }

// ErlangPath returns the resolved real erl launcher of the closure.
func (e *Elixir) ErlangPath() string { return "" }

// ElixirBinDir returns the directory holding the elixir/mix launchers.
func (e *Elixir) ElixirBinDir() string { return "" }

// ErlangBinDir returns the directory holding the erl launcher.
func (e *Elixir) ErlangBinDir() string { return "" }

// Validate implements tdd.RuntimeHandle: it performs the runtime's own
// identity probes (elixir --version, mix --version) as real scoped
// subprocesses while the supplied scope is live, then re-checks the
// retained byte identities.
func (e *Elixir) Validate(ctx context.Context, scope processscope.Scope) error {
	return ErrElixirClosureNotImplemented
}

// Close implements tdd.RuntimeHandle: the final process-free
// byte/topology/root-identity revalidation before releasing the retained
// handles. It never launches a process.
func (e *Elixir) Close() error { return ErrElixirClosureNotImplemented }
