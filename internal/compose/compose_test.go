package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/version"
)

func repoRoot() string { return "../.." }

func TestCompositionValidatesAndModelsBranching(t *testing.T) {
	compPath := filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml")
	coordPath := filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json")
	data, _ := osReadFile(compPath)
	comp, err := ir.LoadYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := ir.LoadMachineJSON(coordPath)
	if err != nil {
		t.Fatal(err)
	}
	name, tla, cfg, err := Generate(comp, machine, "FulfillmentSaga.machine.json")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Checkout" {
		t.Errorf("name=%s", name)
	}
	for _, want := range []string{"Fail_Paying", "Undo_payment", "Undo_reservation", "CompensateStall"} {
		if !contains(tla, want) {
			t.Errorf("missing %q in tla", want)
		}
	}
	if !contains(cfg, "Inv_CleanCompensation") {
		t.Error("missing Inv_CleanCompensation in cfg")
	}
}

func TestCompositionRejectsMissingUndo(t *testing.T) {
	compPath := filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml")
	coordPath := filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json")
	data, _ := osReadFile(compPath)
	comp, _ := ir.LoadYAML(data)
	machine, _ := ir.LoadMachineJSON(coordPath)
	// drop the undo from a non-final step
	seq := comp.AsObject().Get2("sequence").AsArray()
	seq[1].AsObject().Delete("undo")
	_, _, _, err := Generate(comp, machine, "m")
	if err == nil || !contains(err.Error(), "undo") {
		t.Fatalf("expected undo error, got %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestComposeRejectsUnmodeledCoordinatorTransition(t *testing.T) {
	// FORMAL-F1 (compose side, reviewer mutation exp-g): a chain step gains an
	// on: route (Shipping.abort -> Failed) the composition model does not
	// carry; it must be a hard validation error, never silently unmodeled.
	compPath := filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml")
	coordPath := filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json")
	data, _ := osReadFile(compPath)
	comp, err := ir.LoadYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := ir.LoadMachineJSON(coordPath)
	if err != nil {
		t.Fatal(err)
	}
	node := machine.AsObject().Get2("states").AsObject().Get2("Shipping").AsObject()
	tr := ir.NewObject()
	tr.Set("target", ir.StringValue("Failed"))
	on := ir.NewObject()
	on.Set("abort", ir.ObjectValue(tr))
	node.Set("on", ir.ObjectValue(on))
	_, _, _, genErr := Generate(comp, machine, "FulfillmentSaga.machine.json")
	if genErr == nil || !contains(genErr.Error(), "abort") {
		t.Fatalf("unmodeled coordinator transition accepted: %v", genErr)
	}
}

func TestComposeRejectsEmptyForwardChain(t *testing.T) {
	comp, err := ir.LoadYAML([]byte("name: X\ncoordinator: C\naggregates:\n  a:\n    states: [S]\nsequence: []\n"))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := ir.LoadMachineJSONStr("w", `{"id":"c","initial":"Completed","states":{"Completed":{"type":"final"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, genErr := Generate(comp, machine, "C.machine.json")
	if genErr == nil || !contains(genErr.Error(), "no forward chain") {
		t.Fatalf("empty forward chain accepted: %v", genErr)
	}
}

func TestComposeRejectsDuplicateUndoForAggregate(t *testing.T) {
	compPath := filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml")
	coordPath := filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json")
	data, _ := osReadFile(compPath)
	comp, err := ir.LoadYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := ir.LoadMachineJSON(coordPath)
	if err != nil {
		t.Fatal(err)
	}
	// point the second step's aggregate at the first step's aggregate
	seq := comp.AsObject().Get2("sequence").AsArray()
	if len(seq) < 2 {
		t.Skip("fixture too small")
	}
	first := seq[0].AsObject().GetString("aggregate")
	// copy first step's aggregate and undo to the second step
	seq[1].AsObject().Set("aggregate", ir.StringValue(first))
	seq[1].AsObject().Set("to", ir.StringValue(seq[0].AsObject().GetString("to")))
	if u := seq[0].AsObject().Get2("undo"); u != nil {
		seq[1].AsObject().Set("undo", u)
	}
	_, _, _, genErr := Generate(comp, machine, "FulfillmentSaga.machine.json")
	if genErr == nil || !contains(genErr.Error(), "exactly one step") {
		t.Fatalf("duplicate undo accepted: %v", genErr)
	}
}

// P-F10: the written .tla/.cfg pair carries exactly one version stamp line
// each; the in-memory Generate output stays unstamped.
func TestRunWrittenStampsGeneratorVersion(t *testing.T) {
	outdir := t.TempDir()
	names, err := RunWritten(
		filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml"),
		filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"),
		outdir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("names = %v", names)
	}
	for _, n := range names {
		data, rerr := os.ReadFile(filepath.Join(outdir, n))
		if rerr != nil {
			t.Fatal(rerr)
		}
		body := string(data)
		if !strings.Contains(body, version.TLAStamp()) {
			t.Errorf("%s carries no version stamp", n)
		}
		if got := strings.Count(body, "machinery-version:"); got != 1 {
			t.Errorf("%s carries %d stamp lines, want exactly 1", n, got)
		}
		if strings.HasSuffix(n, ".tla") && !strings.HasPrefix(body, "---- MODULE ") {
			t.Errorf("%s no longer opens with the MODULE line", n)
		}
	}
}
