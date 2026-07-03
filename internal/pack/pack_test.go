package pack

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
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

func TestContentHashCoversManifestMinusHashLine(t *testing.T) {
	manifest := "subsystem: s\ndelegated_invariants:\n  - inv-1\ncontent_hash: aaa\n"
	files := map[string]string{"a.txt": "x", "b.txt": "y", "pack.yaml": manifest}
	h1 := ContentHash(files)
	// changing only the content_hash line must not change the hash (the hash
	// is written into the manifest, so it cannot feed back into itself)
	files["pack.yaml"] = strings.Replace(manifest, "content_hash: aaa", "content_hash: bbb", 1)
	if ContentHash(files) != h1 {
		t.Fatal("the content_hash line fed back into the hash")
	}
	// editing any OTHER manifest line (e.g. deleting a delegated invariant)
	// must change the hash: the manifest is covered
	files["pack.yaml"] = strings.Replace(manifest, "  - inv-1\n", "", 1)
	if ContentHash(files) == h1 {
		t.Fatal("manifest edit did not change the hash; the manifest is not covered")
	}
	// file contents stay covered, and the hash is order-independent by name
	files["pack.yaml"] = manifest
	files["a.txt"] = "z"
	if ContentHash(files) == h1 {
		t.Fatal("file change did not change the hash")
	}
}

// The hash a fresh generation writes into the manifest must verify against
// the very files it wrote (the child-side check reads them back from disk).
func TestGeneratedManifestHashVerifies(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	for id, files := range packs {
		v, err := ir.LoadYAML([]byte(files["pack.yaml"]))
		if err != nil || v.AsObject() == nil {
			t.Fatalf("pack %s manifest does not parse", id)
		}
		if got := v.AsObject().GetString("content_hash"); got != ContentHash(files) {
			t.Fatalf("pack %s manifest hash %s does not verify against its own files (%s)", id, got, ContentHash(files))
		}
	}
}

func TestSliceModelithQuotesTitleAndPreservesVersion(t *testing.T) {
	src := `kind: modelith
version: 1
title: "Checkout: Split"
enums: {}
entities:
  Order:
    attributes:
      - name: status
        type: OrderStatus
invariants: []
scenarios: []
`
	dm, err := ir.LoadYAML([]byte(src))
	if err != nil || dm.AsObject() == nil {
		t.Fatal(err)
	}
	out := sliceModelith(dm, Subsystem{ID: "orders", Owns: []string{"Order"}})
	v, err := ir.LoadYAML([]byte(out))
	if err != nil || v.AsObject() == nil {
		t.Fatalf("slice with a colon title does not round-trip: %v\n%s", err, out)
	}
	o := v.AsObject()
	if got := o.GetString("title"); got != "Checkout: Split / orders" {
		t.Errorf("title mangled: %q", got)
	}
	ver := o.Get2("version")
	if ver == nil || ver.Kind != ir.KindNumber || string(ver.AsNumber()) != "1" {
		t.Errorf("numeric version not preserved: %s", ir.Repr(ver))
	}
}

func TestYamlQuoteQuotesNumberLikeStrings(t *testing.T) {
	for _, s := range []string{"1e5", "1.5", "42", "-3", "0x1F"} {
		if got := yamlQuote(s); got != `"`+s+`"` {
			t.Errorf("yamlQuote(%q) = %s; a number-like string type-flips on reparse", s, got)
		}
	}
	if got := yamlQuote("plainWord"); got != "plainWord" {
		t.Errorf("plain string quoted needlessly: %s", got)
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
