package pack

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramirosalas/machinery/internal/ir"
)

func parentDesign() string {
	return filepath.Join("..", "..", "examples", "checkout-split", "parent", "design")
}

func TestLoadDecompositionValidates(t *testing.T) {
	d, err := LoadDecomposition(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Subsystems) != 2 {
		t.Fatalf("subsystems=%d", len(d.Subsystems))
	}
	if d.Subsystems[0].ID != "orders" || d.Subsystems[1].ID != "payments" {
		t.Fatalf("ids=%v,%v", d.Subsystems[0].ID, d.Subsystems[1].ID)
	}
	if d.Subsystems[0].DelegatedInvariants[0] != "no-ship-without-capture" {
		t.Fatal("delegated invariant lost")
	}
}

func TestGeneratePacksIsDeterministic(t *testing.T) {
	a, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	b, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	for id := range a {
		if ContentHash(a[id]) != ContentHash(b[id]) {
			t.Fatalf("pack %s not byte-deterministic", id)
		}
		for name, body := range a[id] {
			if b[id][name] != body {
				t.Fatalf("pack %s file %s differs across runs", id, name)
			}
		}
	}
}

func TestContentHashIgnoresManifestAndOrder(t *testing.T) {
	files := map[string]string{"a.txt": "x", "b.txt": "y", "pack.yaml": "anything"}
	h1 := ContentHash(files)
	files["pack.yaml"] = "something else entirely"
	if ContentHash(files) != h1 {
		t.Fatal("manifest content leaked into its own hash")
	}
	files["a.txt"] = "z"
	if ContentHash(files) == h1 {
		t.Fatal("file change did not change the hash")
	}
}

func TestDomainSliceRoundTripsAndFreezesShape(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	slice := packs["orders"]["domain.modelith.yaml"]
	v, err := ir.LoadYAML([]byte(slice))
	if err != nil || v.AsObject() == nil {
		t.Fatalf("slice does not round-trip through the yaml loader: %v", err)
	}
	o := v.AsObject()
	if !o.GetObject("entities").Has("Order") {
		t.Fatal("owned entity missing from the slice")
	}
	if o.GetObject("entities").Has("Payment") {
		t.Fatal("foreign entity leaked into the slice")
	}
	if !o.GetObject("enums").Has("OrderStatus") {
		t.Fatal("referenced enum missing from the slice")
	}
	if o.GetObject("enums").Has("PaymentStatus") {
		t.Fatal("unreferenced enum leaked into the slice")
	}
	if !strings.Contains(slice, "no-ship-without-capture") {
		t.Fatal("delegated invariant missing from the slice")
	}
}

func TestEventsSliceDirections(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	ev := packs["payments"]["events.md"]
	if !strings.Contains(ev, "| request | consumes | orders |") {
		t.Fatalf("payments should consume request:\n%s", ev)
	}
	if !strings.Contains(ev, "| markPaid | produces | orders |") {
		t.Fatalf("payments should produce markPaid:\n%s", ev)
	}
}
