package experiments

// Mutation experiments for the recursive-decomposition mechanism (G5-pack and
// the pack/refine generators), using examples/checkout-split as the fixture.
// Every failure mode of the recursion converts 1:1 into a case here: a
// mechanism that holds subsystems to contracts must itself be held.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramirosalas/machinery/internal/gates"
	"github.com/ramirosalas/machinery/internal/pack"
)

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// splitFixture copies examples/checkout-split to a temp dir and returns the
// parent, orders, and payments design dirs.
func splitFixture(t *testing.T) (parent, orders, payments string) {
	t.Helper()
	root := t.TempDir()
	copyTree(t, filepath.Join("..", "..", "examples", "checkout-split"), root)
	return filepath.Join(root, "parent", "design"),
		filepath.Join(root, "orders", "design"),
		filepath.Join(root, "payments", "design")
}

func TestSplitFixturePassesAllPackGates(t *testing.T) {
	parent, orders, payments := splitFixture(t)
	for _, d := range []string{parent, orders, payments} {
		g := gates.CheckPack(d)
		if len(g.Errs) != 0 || len(g.Drift) != 0 {
			t.Errorf("%s: errs=%v drift=%v", d, g.Errs, g.Drift)
		}
	}
}

func TestEditedPackFailsItsHash(t *testing.T) {
	_, orders, _ := splitFixture(t)
	editFile(t, filepath.Join(orders, "pack", "events.md"), "at-least-once", "exactly-once")
	if !containsAny(gates.CheckPack(orders).Errs, "fails its own content hash") {
		t.Error("a hand-edited pack passed G5")
	}
}

func TestStalePackPinFailsParentG5(t *testing.T) {
	parent, _, _ := splitFixture(t)
	// the parent's contract machine changes; the committed pack and the child
	// pin are now both stale
	editFile(t, filepath.Join(parent, "contracts", "OrdersContract.machine.json"),
		`"markDeclined": {"target": "Done"}`, `"markDeclined": {"target": "Paid"}`)
	g := gates.CheckPack(parent)
	if len(g.Drift) == 0 {
		t.Errorf("stale committed pack not reported as drift: errs=%v", g.Errs)
	}
	found := false
	for _, e := range g.Errs {
		if strings.Contains(e, "was built against pack") {
			found = true
		}
	}
	if !found {
		t.Errorf("stale child pin not reported: %v", g.Errs)
	}
}

func TestChildDroppingConsumedEventFailsG5(t *testing.T) {
	_, orders, _ := splitFixture(t)
	// stop handling markDeclined everywhere (rename the handler and its ignore)
	mp := filepath.Join(orders, "machines", "Order.machine.json")
	data, _ := os.ReadFile(mp)
	mustWrite(t, mp, strings.ReplaceAll(string(data), `"markDeclined"`, `"internalDecline"`))
	g := gates.CheckPack(orders)
	if !containsAny(g.Errs, "'markDeclined' (consumed) is handled or ignored by no machine") {
		t.Errorf("dropped consumed boundary event passed G5: %v", g.Errs)
	}
}

func TestChildDroppingProducedEventFailsG5(t *testing.T) {
	_, _, payments := splitFixture(t)
	data, _ := os.ReadFile(filepath.Join(payments, "machines", "Payment.machine.json"))
	mustWrite(t, filepath.Join(payments, "machines", "Payment.machine.json"),
		strings.ReplaceAll(string(data), `"markPaid"`, `"settleInternally"`))
	data2, _ := os.ReadFile(filepath.Join(payments, "machines", "Payment.matrix.md"))
	mustWrite(t, filepath.Join(payments, "machines", "Payment.matrix.md"),
		strings.ReplaceAll(string(data2), "markPaid", "settleInternally"))
	g := gates.CheckPack(payments)
	if !containsAny(g.Errs, "'markPaid' (produced) appears in no machine action") {
		t.Errorf("dropped produced boundary event passed G5: %v", g.Errs)
	}
}

func TestChildEnumDriftFailsG5(t *testing.T) {
	_, orders, _ := splitFixture(t)
	editFile(t, filepath.Join(orders, "orders.modelith.yaml"),
		`      - {name: Cancelled, definition: "Cancelled before settlement; terminal."}`,
		`      - {name: Voided, definition: "Renamed; the public shape is frozen."}`)
	if !containsAny(gates.CheckPack(orders).Errs, "enum OrderStatus drifted from the pack") {
		t.Error("public enum drift passed G5")
	}
}

func TestDelegatedInvariantUntracedFailsG5(t *testing.T) {
	_, orders, _ := splitFixture(t)
	for _, f := range []string{
		filepath.Join(orders, "BUILD.md"),
		filepath.Join(orders, "machines", "Order.matrix.md"),
	} {
		data, _ := os.ReadFile(f)
		mustWrite(t, f, strings.ReplaceAll(string(data), "no-ship-without-capture", "some-other-words"))
	}
	if !containsAny(gates.CheckPack(orders).Errs, "delegated invariant 'no-ship-without-capture'") {
		t.Error("untraced delegated invariant passed G5")
	}
}

func TestPackmapWeakeningFailsReconciliation(t *testing.T) {
	_, orders, _ := splitFixture(t)
	// map Shipped to Paid instead of Done: the map still typechecks but the
	// refinement proof would be wrong; here we drop a state entirely, which
	// reconciliation itself must reject
	editFile(t, filepath.Join(orders, "packmap.yaml"), "  Cancelled: Done\n", "")
	_, err := pack.GenerateRefinement(orders)
	if err == nil || !strings.Contains(err.Error(), "no mapping entry") {
		t.Fatalf("partial packmap accepted: %v", err)
	}
}

func TestPackmapBogusContractStateFailsReconciliation(t *testing.T) {
	_, orders, _ := splitFixture(t)
	editFile(t, filepath.Join(orders, "packmap.yaml"), "Shipped: Done", "Shipped: Warehouse")
	_, err := pack.GenerateRefinement(orders)
	if err == nil || !strings.Contains(err.Error(), "not a contract state") {
		t.Fatalf("bogus contract state accepted: %v", err)
	}
}

func TestStaleRefinementArtifactIsDrift(t *testing.T) {
	_, orders, _ := splitFixture(t)
	editFile(t, filepath.Join(orders, "formal", "OrderPackRefinement.tla"),
		`s = "Shipped" -> "Done"`, `s = "Shipped" -> "Paid"`)
	if len(gates.CheckPack(orders).Drift) == 0 {
		t.Error("hand-edited refinement module passed G5 as fresh")
	}
}

func TestDoubleOwnershipFailsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"), "owns: [Payment]", "owns: [Payment, Order]")
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "ownership must be exactly-once") {
		t.Fatalf("double ownership accepted: %v", err)
	}
}

func TestUnownedEntityFailsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"), "owns: [Payment]", "owns: []")
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "owned by no subsystem") {
		t.Fatalf("unowned entity accepted: %v", err)
	}
}

func TestContractMachineOutsideSubsetFailsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "contracts", "PaymentsContract.machine.json"),
		`"Settling": {`, `"Settling": {"after": {"t": {"target": "Done"}},`)
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "restricted to plain on-transitions") {
		t.Fatalf("contract machine with after: accepted: %v", err)
	}
}

func TestExplicitG5OnPlainDesignIsError(t *testing.T) {
	design, _ := fixture(t) // the widget fixture: no decomposition, no pack
	if !containsAny(gates.CheckPack(design).Errs, "nothing to check") {
		t.Error("G5 on a plain design must error, not silently pass")
	}
}
