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

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/pack"
)

func init() {
	RegisterRunner("packsuite_test.go",
		"edited-pack-fails-hash", "stale-child-pin", "dropped-consumed-event",
		"dropped-produced-event", "frozen-enum-drift", "delegated-invariant-untraced",
		"partial-packmap", "stale-refinement-artifact", "double-ownership",
		"unowned-entity", "contract-outside-subset")
}

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

// ------------------- 2026-07-02 review: G5 hardening -----------------------

// The content hash covers the manifest: deleting a delegated invariant from
// the child's copied pack/pack.yaml alone must fail the child's G5.
func TestChildEditingManifestDelegatedInvariantFailsHash(t *testing.T) {
	_, orders, _ := splitFixture(t)
	editFile(t, filepath.Join(orders, "pack", "pack.yaml"), "  - no-ship-without-capture\n", "")
	if !containsAny(gates.CheckPack(orders).Errs, "fails its own content hash") {
		t.Error("editing the child's pack manifest passed G5")
	}
}

// A produced boundary event needs an EMITTER; a mere on: handler proves
// consumption, not emission, and used to satisfy the produced check.
func TestChildProducedEventHandlerOnlyFailsG5(t *testing.T) {
	_, _, payments := splitFixture(t)
	mp := filepath.Join(payments, "machines", "Payment.machine.json")
	editFile(t, mp,
		`"capture": {"target": "Captured", "actions": "markPaid"}`,
		`"capture": {"target": "Captured", "actions": "recordCapture"}, "markPaid": {"target": "Captured"}`)
	data, _ := os.ReadFile(filepath.Join(payments, "machines", "Payment.matrix.md"))
	mustWrite(t, filepath.Join(payments, "machines", "Payment.matrix.md"),
		strings.ReplaceAll(string(data), "markPaid", "recordCapture"))
	g := gates.CheckPack(payments)
	if !containsAny(g.Errs, "'markPaid' (produced) appears in no machine action") {
		t.Errorf("handler-only produced event passed G5: %v", g.Errs)
	}
}

// Subsystem ids become path segments under packs/; traversal must fail.
func TestSubsystemIDTraversalFailsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"), "- id: orders", "- id: ../../../escaped/orders")
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "bare name") {
		t.Fatalf("traversal subsystem id accepted: %v", err)
	}
}

// contract_machine must resolve inside the design directory.
func TestContractMachineEscapeFailsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"),
		"contract_machine: contracts/OrdersContract.machine.json",
		"contract_machine: ../../../../contracts/OrdersContract.machine.json")
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "outside the design directory") {
		t.Fatalf("escaping contract_machine accepted: %v", err)
	}
}

// The pack boundary lookup uses the same heading-anchored contract locator as
// G2; prose mentioning contract_version above the heading must not break it.
func TestProseContractVersionMentionDoesNotBreakG5(t *testing.T) {
	parent, _, _ := splitFixture(t)
	arch := filepath.Join(parent, "ARCHITECTURE.md")
	data, err := os.ReadFile(arch)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, arch, "The contract_version discipline is explained below.\n\n"+string(data))
	g := gates.CheckPack(parent)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Errorf("prose contract_version mention broke G5: errs=%v drift=%v", g.Errs, g.Drift)
	}
}

// Parent freshness findings iterate the pack files in sorted order, so the
// gate output is byte-stable across runs (map-order iteration was not).
func TestParentPackFindingsAreDeterministicallyOrdered(t *testing.T) {
	parent, _, _ := splitFixture(t)
	for _, name := range []string{"events.md", "domain.modelith.yaml"} {
		if err := os.Remove(filepath.Join(parent, "packs", "orders.pack", name)); err != nil {
			t.Fatal(err)
		}
	}
	g := gates.CheckPack(parent)
	var missing []string
	for _, e := range g.Errs {
		if strings.Contains(e, "missing committed file") {
			missing = append(missing, e)
		}
	}
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing-file findings, got %v", g.Errs)
	}
	if !strings.Contains(missing[0], "domain.modelith.yaml") || !strings.Contains(missing[1], "events.md") {
		t.Errorf("findings not in sorted (deterministic) order: %v", missing)
	}
}

// A committed pack file a fresh generation never produces is stale trust bait.
func TestExtraStaleFileInPackFailsParentG5(t *testing.T) {
	parent, _, _ := splitFixture(t)
	mustWrite(t, filepath.Join(parent, "packs", "orders.pack", "leftover.tla"), "---- MODULE stale ----\n====\n")
	if !containsAny(gates.CheckPack(parent).Errs, "not part of a fresh generation") {
		t.Error("extra stale committed pack file passed G5")
	}
}

// A subsystem with no child_design is non-blocking (parent-first workflow)
// but must WARN and show up in the checked counts.
func TestUnpinnedSubsystemWarnsButDoesNotBlock(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"), "    child_design: ../../payments/design\n", "")
	g := gates.CheckPack(parent)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Errorf("unpinned subsystem must not block: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if !containsAny(g.Warns, "declares no child_design") {
		t.Errorf("missing unpinned-subsystem warning: %v", g.Warns)
	}
	if g.Counts["children pinned"] != 1 || g.Counts["children unpinned"] != 1 {
		t.Errorf("pin counts wrong: %v", g.Counts)
	}
}

// An absolute child_design is rejected at decomposition load.
func TestAbsoluteChildDesignFailsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"),
		"child_design: ../../payments/design", "child_design: /etc/payments/design")
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "must be a relative path") {
		t.Fatalf("absolute child_design accepted: %v", err)
	}
}
