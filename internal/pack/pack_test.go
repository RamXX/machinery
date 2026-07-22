package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

func parentDesign() string {
	return filepath.Join("..", "..", "examples", "checkout-split", "parent", "design")
}

// copyParentDesign copies the checkout-split parent design into a temp dir so
// a test can mutate the sources (ARCHITECTURE.md table cells, decomposition
// waivers) without touching the shipped example.
func copyParentDesign(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := parentDesign()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// editDesignFile replaces old with new in one design file, failing when the
// needle is absent (a silent no-op mutation would vacuously pass the test).
func editDesignFile(t *testing.T, design, name, old, new string) {
	t.Helper()
	p := filepath.Join(design, name)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", name, old)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
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

// The event-contract extraction is strict: a producer or consumer cell that
// does not resolve to exactly one known participant fails generation with a
// finding naming the row and the offending cell text. Silent drops shipped
// packs asserting boundary completeness over an almost-empty table once; the
// cases below are the observed lossy shapes.
func TestStrictEventCellValidation(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		wantSubs []string
	}{
		{
			name: "comma multi-consumer",
			old:  "| request | orders | payments |",
			new:  "| request | orders | orders (drafts), payments (audit) |",
			wantSubs: []string{
				"event 'request'",
				"consumer cell 'orders (drafts), payments (audit)'",
				"names more than one component",
				"one row per producer-consumer pair",
			},
		},
		{
			name:     "slash multi-consumer",
			old:      "| request | orders | payments |",
			new:      "| request | orders | orders/payments |",
			wantSubs: []string{"names more than one component"},
		},
		{
			name:     "arrow in cell",
			old:      "| request | orders | payments |",
			new:      "| request | orders | orders -> payments |",
			wantSubs: []string{"names more than one component"},
		},
		{
			name:     "fan-out phrase",
			old:      "| request | orders | payments |",
			new:      "| request | orders | ALL components (fan-out) |",
			wantSubs: []string{"names more than one component"},
		},
		{
			name:     "event name embedded in producer cell",
			old:      "| markPaid | payments | orders |",
			new:      "| markPaid | payments `markPaid` | orders |",
			wantSubs: []string{"event 'markPaid'", "names more than one component"},
		},
		{
			name: "unknown single name",
			old:  "| request | orders | payments |",
			new:  "| request | warehouse | payments |",
			wantSubs: []string{
				"producer cell 'warehouse'",
				"is not a known component",
				"orders, payments", // the known-participant list teaches the fix
			},
		},
		{
			name:     "empty producer cell",
			old:      "| request | orders | payments |",
			new:      "| request |  | payments |",
			wantSubs: []string{"event 'request'", "empty producer cell"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			design := copyParentDesign(t)
			editDesignFile(t, design, "ARCHITECTURE.md", c.old, c.new)
			_, err := GeneratePacks(design)
			if err == nil {
				t.Fatal("lossy event-contract cell accepted; extraction must fail loudly")
			}
			for _, sub := range c.wantSubs {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("error %q\nmissing %q", err.Error(), sub)
				}
			}
		})
	}
}

// Parenthetical annotations are the sanctioned annotation channel: the cell
// resolves to the component and the pack row keeps the pairwise direction.
func TestParentheticalAnnotationsAccepted(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| request | orders | payments |",
		"| request | orders (command) | payments (worker) |")
	packs, err := GeneratePacks(design)
	if err != nil {
		t.Fatal(err)
	}
	if ev := packs["payments"]["events.md"]; !strings.Contains(ev, "| request | consumes | orders |") {
		t.Errorf("annotated producer cell lost the pairwise contract:\n%s", ev)
	}
	if ev := packs["orders"]["events.md"]; !strings.Contains(ev, "| request | produces | payments |") {
		t.Errorf("annotated consumer cell lost the pairwise contract:\n%s", ev)
	}
}

// A row between two non-pack participants (declared only as Architecture
// Contract boundary elements) validates like any other row and emits no pack
// rows.
func TestNonPackToNonPackRowValidatesAndEmitsNothing(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "ARCHITECTURE.md",
		"  - id: payments.svc",
		"  - id: gw.svc\n    kind: container\n    element: gw\n    code: [ \"gw/**\" ]\n  - id: payments.svc")
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| markDeclined | payments | orders | Payment.orderId | at-least-once | none | Payment.id |",
		"| markDeclined | payments | orders | Payment.orderId | at-least-once | none | Payment.id |\n| ping | gw | gw | n/a | at-least-once | none | n/a |")
	packs, err := GeneratePacks(design)
	if err != nil {
		t.Fatalf("non-pack-to-non-pack row rejected: %v", err)
	}
	for id, files := range packs {
		if strings.Contains(files["events.md"], "ping") {
			t.Errorf("pack %s picked up a row between two non-pack participants:\n%s", id, files["events.md"])
		}
	}
}

// zeroEventPayments rewrites the table so no row names payments: extraction
// yields zero boundary events for that subsystem. The rows route through a
// declared non-pack boundary element (gw) because a row whose producer and
// consumer belong to one subsystem is itself a generation error now.
func zeroEventPayments(t *testing.T, design string) {
	t.Helper()
	editDesignFile(t, design, "ARCHITECTURE.md",
		"  - id: payments.svc",
		"  - id: gw.svc\n    kind: container\n    element: gw\n    code: [ \"gw/**\" ]\n  - id: payments.svc")
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| request | orders | payments |", "| request | orders | gw |")
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| markPaid | payments | orders |", "| markPaid | gw | orders |")
	editDesignFile(t, design, "ARCHITECTURE.md",
		"| markDeclined | payments | orders |", "| markDeclined | gw | orders |")
}

// waivePayments declares the payments boundary_events waiver.
func waivePayments(t *testing.T, design, reason string) {
	t.Helper()
	editDesignFile(t, design, "decomposition.yaml",
		"    delegated_invariants: []\n",
		"    delegated_invariants: []\n    boundary_events:\n      none: \""+reason+"\"\n")
}

func TestZeroBoundaryEventsIsGenerationError(t *testing.T) {
	design := copyParentDesign(t)
	zeroEventPayments(t, design)
	_, err := GeneratePacks(design)
	if err == nil {
		t.Fatal("a zero-boundary-event subsystem generated silently")
	}
	for _, sub := range []string{"'payments'", "extracts zero boundary events", "boundary_events"} {
		if !strings.Contains(err.Error(), sub) {
			t.Errorf("error %q\nmissing %q", err.Error(), sub)
		}
	}
}

func TestZeroBoundaryEventsWaiverGeneratesWithReason(t *testing.T) {
	design := copyParentDesign(t)
	zeroEventPayments(t, design)
	reason := "payments is store-driven in this variant; no events cross its boundary"
	waivePayments(t, design, reason)
	packs, err := GeneratePacks(design)
	if err != nil {
		t.Fatalf("waived zero-event subsystem still fails: %v", err)
	}
	ev := packs["payments"]["events.md"]
	if !strings.Contains(ev, reason) {
		t.Errorf("waiver reason missing from events.md:\n%s", ev)
	}
	if strings.Contains(ev, "there are no other cross-boundary events") {
		t.Errorf("waived events.md still emits the completeness claim:\n%s", ev)
	}
	if !strings.Contains(ev, "Boundary events: 0 (waived)") {
		t.Errorf("waived events.md missing the visible zero count:\n%s", ev)
	}
	if got := CountBoundaryEvents(ev); got != 0 {
		t.Errorf("CountBoundaryEvents(waived) = %d, want 0", got)
	}
}

func TestStaleWaiverIsGenerationError(t *testing.T) {
	design := copyParentDesign(t)
	waivePayments(t, design, "stale: rows still name payments")
	_, err := GeneratePacks(design)
	if err == nil || !strings.Contains(err.Error(), "remove the stale waiver") {
		t.Fatalf("stale waiver accepted: %v", err)
	}
}

func TestMalformedWaiverIsLoadError(t *testing.T) {
	design := copyParentDesign(t)
	editDesignFile(t, design, "decomposition.yaml",
		"    delegated_invariants: []\n",
		"    delegated_invariants: []\n    boundary_events: nonsense\n")
	_, err := LoadDecomposition(design)
	if err == nil || !strings.Contains(err.Error(), "boundary_events must be a mapping") {
		t.Fatalf("malformed boundary_events accepted: %v", err)
	}
}

func TestCountBoundaryEvents(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	if got := CountBoundaryEvents(packs["payments"]["events.md"]); got != 3 {
		t.Errorf("payments boundary events = %d, want 3", got)
	}
	if got := CountBoundaryEvents("not a generated events file"); got != -1 {
		t.Errorf("absent count line = %d, want -1", got)
	}
}

// A parent model with no title yields just the subsystem id; the old
// unconditional concatenation emitted the nonsense title " / core".
func TestSliceModelithTitleWithoutParentTitle(t *testing.T) {
	src := `kind: modelith
version: 1
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
		t.Fatalf("slice does not round-trip: %v\n%s", err, out)
	}
	if got := v.AsObject().GetString("title"); got != "orders" {
		t.Errorf("title = %q, want %q", got, "orders")
	}
}

// P-F10: pack files are covered by the content hash, so they must NEVER carry
// a version stamp (it would churn the hash every release and re-red every
// pinned child).
func TestGeneratedPackFilesAreUnstamped(t *testing.T) {
	packs, err := GeneratePacks(parentDesign())
	if err != nil {
		t.Fatal(err)
	}
	for id, files := range packs {
		for name, body := range files {
			if strings.Contains(body, "machinery-version:") {
				t.Errorf("pack %s file %s carries a version stamp; the content hash covers it", id, name)
			}
		}
	}
}

// P-F10: the refinement modules the child commits to design/formal DO carry
// the stamp (G5 strips it before the freshness diff); the contract module
// stays a byte-copy of the pack's hash-covered file.
func TestGenerateRefinementStampsGeneratedModules(t *testing.T) {
	design := filepath.Join("..", "..", "examples", "checkout-split", "orders", "design")
	files, err := GenerateRefinement(design)
	if err != nil {
		t.Fatal(err)
	}
	stamped := 0
	for name, body := range files {
		switch {
		case strings.HasSuffix(name, "PackRefinement.tla"), strings.HasSuffix(name, "PackRefinement.cfg"):
			if got := strings.Count(body, "machinery-version:"); got != 1 {
				t.Errorf("%s carries %d stamp lines, want exactly 1", name, got)
			}
			if strings.HasSuffix(name, ".tla") && !strings.HasPrefix(body, "---- MODULE ") {
				t.Errorf("%s no longer opens with the MODULE line", name)
			}
			stamped++
		default:
			// the contract module copied from the pack
			if strings.Contains(body, "machinery-version:") {
				t.Errorf("%s must stay byte-identical to the pack's copy (no stamp)", name)
			}
			packCopy, rerr := os.ReadFile(filepath.Join(design, "pack", name))
			if rerr != nil {
				t.Fatal(rerr)
			}
			if body != string(packCopy) {
				t.Errorf("%s differs from the pack's copy", name)
			}
		}
	}
	if stamped != 2 {
		t.Errorf("expected 2 stamped refinement artifacts, got %d", stamped)
	}
}
