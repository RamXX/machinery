// Package adapters owns the closed first-release native adapters of
// docs/test-assurance-contract.md section 6. This file is the
// python-unittest/v1 adapter (MAC-imtz): it embeds the byte-pinned
// machinery-check/v1 assertion helper and the byte-pinned bootstrap/harness
// from the owned assets directory, prepares captured Python suites
// (verified bundle materialization, embedded-harness static AST validation
// of every registered assertion call site, native TestLoader discovery with
// full TestCase.id() reconciliation, exact frozen argv under CPython's
// isolated mode with a closed environment and no bytecode) and executes
// them under processscope custody, normalizing the harness JSONL report to
// machinery.tdd.event/v1.
package adapters

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

//go:embed assets/python/machinery_check.py
var pythonHelperSource []byte

//go:embed assets/python/bootstrap.py
var pythonBootstrapSource []byte

// Closed constants of the python-unittest/v1 adapter.
const (
	// AdapterPythonUnittest is the closed adapter identity.
	AdapterPythonUnittest = protocol.AdapterPythonUnittest

	// PythonEventSchema is the normalized event identity of every emitted
	// event.
	PythonEventSchema = "machinery.tdd.event/v1"

	// PythonWitnessSchema is the closed helper witness identity.
	PythonWitnessSchema = "machinery.tdd.witness/v1"

	// PythonHarnessSchema is the closed embedded-harness record identity.
	PythonHarnessSchema = "machinery.tdd.harness.python/v1"

	// PythonHelperPinnedSHA256 and PythonBootstrapPinnedSHA256 pin the
	// exact embedded asset bytes; each pin is verified at materialization
	// and re-verified after every native run.
	PythonHelperPinnedSHA256    = "a6dbeaecd11d7207bf9aa00ef9804dc8173e5378eb2ca9c38ee6a1e9924feb81"
	PythonBootstrapPinnedSHA256 = "90ef6c4c77c7b71b6879ef4152bf96e7016baa8ac506018b93fe621ce3a6e8f5"

	// PythonConformanceFixtureSHA256 pins the frozen native conformance
	// fixture bytes executed by the required contributor lane fragment
	// testdata/integration-lanes/assurance-python.json.
	PythonConformanceFixtureSHA256 = "53151c61000022e78a34d1874ab3581bb0a37821d05eb0fa7ca96dad564b5038"

	// PythonHelperFile and PythonBootstrapFile are the materialized asset
	// names inside every prepared suite root.
	PythonHelperFile    = "machinery_check.py"
	PythonBootstrapFile = "_machinery_bootstrap.py"

	pythonRequiredVersion = "3.14.7"
)

// errPythonAdapterPending is the RED-phase placeholder returned before the
// closed adapter is implemented.
var errPythonAdapterPending = errors.New("PYTHON_ADAPTER_PENDING: the python-unittest/v1 adapter is not implemented yet")

// PythonAdapter is the closed python-unittest/v1 Adapter.
type PythonAdapter struct{}

// Compile-time contract check of the closed Adapter boundary.
var _ tdd.Adapter = (*PythonAdapter)(nil)

// Python returns the python-unittest/v1 adapter.
func Python() *PythonAdapter { return &PythonAdapter{} }

// ID implements tdd.Adapter.
func (a *PythonAdapter) ID() string { return AdapterPythonUnittest }

// PythonHelperSource returns a copy of the embedded byte-pinned assertion
// helper transport.
func PythonHelperSource() []byte { return append([]byte(nil), pythonHelperSource...) }

// PythonBootstrapSource returns a copy of the embedded byte-pinned
// bootstrap/harness.
func PythonBootstrapSource() []byte { return append([]byte(nil), pythonBootstrapSource...) }

// PythonAssets returns the frozen asset inventory: embedded asset name to
// its pinned exact sha256.
func PythonAssets() map[string]string {
	return map[string]string{
		PythonHelperFile:    PythonHelperPinnedSHA256,
		PythonBootstrapFile: PythonBootstrapPinnedSHA256,
	}
}

// Prepare implements tdd.Adapter: it validates the closed request, opens and
// validates the pinned CPython 3.14.7 runtime closure under the live scope,
// materializes and re-verifies the captured source bundle plus the embedded
// byte-pinned helper and bootstrap, validates every registered assertion as
// a typed helper call at its exact frozen line through the embedded harness
// under the pinned closure, proves the native TestLoader discovery of the
// complete TestCase.id() inventory and freezes the exact argv and closed
// environment.
func (a *PythonAdapter) Prepare(ctx context.Context, req tdd.SuiteRequest) (tdd.PreparedSuite, error) {
	return tdd.PreparedSuite{}, errPythonAdapterPending
}

// Run implements tdd.Adapter: it executes the prepared suite as one guarded
// custody job through the pinned CPython interpreter in isolated mode,
// decodes the closed harness JSONL report vocabulary, reconciles the
// complete lifecycle against the frozen inventory with witness-bound
// assertion outcomes, enforces exit concordance and re-verifies the
// prepared sources against late mutation.
func (a *PythonAdapter) Run(ctx context.Context, prepared tdd.PreparedSuite, sink tdd.EventSink) (tdd.Execution, error) {
	return tdd.Execution{}, errPythonAdapterPending
}

// pythonWitness is one decoded machinery-check/v1 witness record.
type pythonWitness struct {
	assertion string
	test      string
	file      string
	line      int64
	condition bool
	thrown    string
}

// parsePythonWitness decodes one closed witness record line.
func parsePythonWitness(line string) (pythonWitness, bool) {
	_ = fmt.Sprintf
	return pythonWitness{}, false
}

// pythonReportRecord is one decoded harness JSONL record.
type pythonReportRecord struct {
	Schema              string
	Kind                string
	Tests               []string
	Test                string
	Outcome             string
	Exception           string
	Feature             string
	Code                string
	Message             string
	TestsRun            int64
	Failures            int64
	Errors              int64
	Skipped             int64
	ExpectedFailures    int64
	UnexpectedSuccesses int64
	Success             bool
}

// parsePythonReport decodes the closed harness JSONL stream; unknown
// schemas/kinds/keys, malformed lines and missing, duplicated or
// non-terminal sentinels fail closed.
func parsePythonReport(raw []byte) ([]pythonReportRecord, error) {
	return nil, errPythonAdapterPending
}

// pythonReconciliation is the reconciled normalized result of one harness
// report.
type pythonReconciliation struct {
	Events     []tdd.Event
	Outcome    string
	Discovered []tdd.NativeID
	Started    []tdd.NativeID
	Completed  []tdd.NativeID
	Assertions []tdd.AssertionOutcome
}

// reconcilePythonReport reconciles the harness lifecycle against the
// complete frozen inventory with the registered assertion witnesses.
func reconcilePythonReport(records []pythonReportRecord, suite *tdd.Suite) (pythonReconciliation, error) {
	return pythonReconciliation{}, errPythonAdapterPending
}

// buildPythonEnv builds the closed native environment: private HOME/TMPDIR,
// UTC, deterministic hash seed, bytecode disabled, minimal PATH and the
// explicitly declared suite variables; PYTHON* and MACHINERY* injection is
// never inherited.
func buildPythonEnv(scratch string, extra []tdd.EnvironmentVar) ([]string, error) {
	return nil, errPythonAdapterPending
}

// pythonNativeID renders the exact TestCase.id() spelling of a declared
// native identity.
func pythonNativeID(n tdd.NativeID) string {
	return n.Module + "." + n.Class + "." + n.Method
}
