// Package adapters owns the closed first-release native adapters of
// docs/test-assurance-contract.md section 6. This file is the go-testing/v1
// adapter (MAC-wi2u): it embeds the byte-pinned machinery-check/v1 helper
// transport from the owned assets directory, prepares captured Go suites
// (verified bundle materialization, typed assertion call-site validation,
// native build selection proof, exact frozen argv) and executes them under
// processscope custody through the pinned Go 1.27.1 runtime closure,
// normalizing native events to machinery.tdd.event/v1.
//
// RED note: until MAC-wi2u GREEN lands, every behavior below is a fail-closed
// stub so the frozen test suites fail on the absent adapter rather than
// silently passing; only the frozen asset bytes, the closed constants and the
// byte-pin accessors are real.
package adapters

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

//go:embed assets/go/machinerycheck/machinerycheck.go
var goHelperSource []byte

// Closed constants of the go-testing/v1 adapter.
const (
	// AdapterGoTesting is the closed adapter identity.
	AdapterGoTesting = protocol.AdapterGoTesting

	// GoEventSchema is the normalized event identity of every emitted event.
	GoEventSchema = "machinery.tdd.event/v1"

	// GoHelperPinnedSHA256 pins the exact embedded helper bytes; the pin is
	// verified at materialization and re-verified after every native run.
	GoHelperPinnedSHA256 = "5b0b1c48f6a5dd95d257b499d6b76fc22d97048f4746c5cc35862287e35bef53"

	// GoConformanceTestSHA256 and GoConformanceGoModSHA256 pin the frozen
	// native conformance fixture bytes executed by the required contributor
	// lane fragment testdata/integration-lanes/assurance-go.json.
	GoConformanceTestSHA256  = "1900b2ae10201d956c20f20242dace862baff4a537b200d87b8a4fc3fbbfb1d1"
	GoConformanceGoModSHA256 = "60ac10e392e35c417b5f78480803124a01750e634c9ef1494c85798322d7a734"

	// GoHelperDir is the directory the helper is materialized into inside
	// every prepared module; frozen suite sources import it as
	// <module-path>/machinerycheck.
	GoHelperDir = "machinerycheck"
)

// errGoAdapterPending is the RED placeholder for the pending GREEN
// implementation.
var errGoAdapterPending = errors.New("UNSUPPORTED_ADAPTER: the go-testing/v1 native adapter is not implemented yet")

// GoAdapter is the closed go-testing/v1 Adapter.
type GoAdapter struct{ pending struct{} }

// Compile-time contract check of the closed Adapter boundary.
var _ tdd.Adapter = (*GoAdapter)(nil)

// Go returns the go-testing/v1 adapter.
func Go() *GoAdapter { return &GoAdapter{} }

// Lookup resolves one closed adapter identity.
func Lookup(id string) (tdd.Adapter, error) {
	if id != AdapterGoTesting {
		return nil, fmt.Errorf("UNSUPPORTED_ADAPTER: %q is not a closed first-release native adapter", id)
	}
	return Go(), nil
}

// ID implements tdd.Adapter.
func (a *GoAdapter) ID() string { return AdapterGoTesting }

// HelperSource returns a copy of the embedded byte-pinned helper transport.
func HelperSource() []byte { return append([]byte(nil), goHelperSource...) }

// Prepare implements tdd.Adapter. RED: always fails with errGoAdapterPending.
func (a *GoAdapter) Prepare(ctx context.Context, req tdd.SuiteRequest) (tdd.PreparedSuite, error) {
	return tdd.PreparedSuite{}, errGoAdapterPending
}

// Run implements tdd.Adapter. RED: always fails with errGoAdapterPending.
func (a *GoAdapter) Run(ctx context.Context, prepared tdd.PreparedSuite, sink tdd.EventSink) (tdd.Execution, error) {
	return tdd.Execution{}, errGoAdapterPending
}

// goWitness is one parsed helper witness line.
type goWitness struct {
	id    string
	test  string
	site  string
	value bool
	line  int64
}

// goJSONEvent is one decoded native test2json event.
type goJSONEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
}

// parseWitnessLine parses one machinery-check/v1 witness line.
func parseWitnessLine(line string) (goWitness, bool) { return goWitness{}, false }

// parseGoTestStream decodes the closed test2json stream vocabulary.
func parseGoTestStream(raw []byte) ([]goJSONEvent, error) {
	return nil, errGoAdapterPending
}

// buildGoRunPattern builds the anchored native selection pattern from the
// declared identities.
func buildGoRunPattern(tests []tdd.Test) (string, error) { return "", errGoAdapterPending }

// buildGoEnv builds the closed native environment.
func buildGoEnv(scratch, goroot string, extra []tdd.EnvironmentVar) ([]string, error) {
	return nil, errGoAdapterPending
}

// validateGoSuiteSources validates frozen suite sources: typed helper import
// and Check call sites at the declared file/line/id, declared fixed subtest
// names, and absence of TestMain/benchmark/fuzz entry points.
func validateGoSuiteSources(moduleDir, modulePath string, suite *tdd.Suite) error {
	return errGoAdapterPending
}

// goReconciliation is the reconciled normalized result of one native stream.
type goReconciliation struct {
	Events     []tdd.Event
	Outcome    string
	Discovered []tdd.NativeID
	Started    []tdd.NativeID
	Completed  []tdd.NativeID
	Assertions []tdd.AssertionOutcome
}

// reconcileGoStream reconciles native lifecycle events against the complete
// frozen leaf inventory with the registered assertion witnesses.
func reconcileGoStream(events []goJSONEvent, suite *tdd.Suite) (goReconciliation, error) {
	return goReconciliation{}, errGoAdapterPending
}
