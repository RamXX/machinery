package compose

import (
	"path/filepath"
	"testing"

	"github.com/ramirosalas/machinery/internal/ir"
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
