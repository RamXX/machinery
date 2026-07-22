package experiments

// 2026-07-21 adversarial review of the recursive-decomposition mechanism
// (task MAC-sadm). Every confirmed finding converts 1:1 into a permanent
// mutation test here, on top of a fixture that is regenerated and re-pinned
// with the CURRENT generator (so these tests never depend on the committed
// example's pack bytes). The classes are declared as PackReviewExperiments in
// experiments.go and registered below.
//
// Classes covered: second-event-table-dropped, duplicate-component-direction-
// flip, undelegated-invariant, child-drops-owned-attribute, child-weakens-
// entity-invariant, stale-pack-dir-after-rename, orphaned-child-green,
// hash-laundering-window (documented limit), packmap-shim-machine,
// stale-packrefinement-extras, pack-subdirectory-smuggling,
// waiver-count-injection, colon-id-roundtrip, child_design-nonexistent-dir,
// plus the pack_revision amendment counter and the scale non-design guard.

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/pack"
	"github.com/RamXX/machinery/internal/version"
)

func init() {
	RegisterRunner("pack_review_experiments_test.go",
		"second-event-table-dropped", "duplicate-component-direction-flip",
		"undelegated-invariant", "child-drops-owned-attribute",
		"child-weakens-entity-invariant", "stale-pack-dir-after-rename",
		"orphaned-child-green", "hash-laundering-window", "packmap-shim-machine",
		"stale-packrefinement-extras", "pack-subdirectory-smuggling",
		"waiver-count-injection", "colon-id-roundtrip", "child_design-nonexistent-dir")
}

var (
	packHashLineRe    = regexp.MustCompile(`(?m)^pack_hash: [0-9a-f]+$`)
	contentHashLineRe = regexp.MustCompile(`(?m)^content_hash: [0-9a-f]+$`)
)

// repinChild re-copies packs/<id>.pack to the child's pack/, updates the
// packmap pin to the fresh content hash, and regenerates the refinement
// artifacts.
func repinChild(t *testing.T, parent, id, child string) {
	t.Helper()
	src := filepath.Join(parent, "packs", id+".pack")
	dst := filepath.Join(child, "pack")
	if err := os.RemoveAll(dst); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, rerr := os.ReadFile(filepath.Join(src, e.Name()))
		if rerr != nil {
			t.Fatal(rerr)
		}
		mustWrite(t, filepath.Join(dst, e.Name()), string(data))
	}
	manifest, err := pack.LoadPackManifest(child)
	if err != nil {
		t.Fatal(err)
	}
	pmPath := filepath.Join(child, "packmap.yaml")
	data, err := os.ReadFile(pmPath)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, pmPath, packHashLineRe.ReplaceAllString(string(data), "pack_hash: "+manifest.GetString("content_hash")))
	if _, err := pack.WriteRefinement(child); err != nil {
		t.Fatal(err)
	}
}

// regenSplitFixture copies examples/checkout-split, regenerates the parent
// packs with the current generator, and re-pins both children.
func regenSplitFixture(t *testing.T) (parent, orders, payments string) {
	t.Helper()
	parent, orders, payments = splitFixture(t)
	if _, err := pack.WritePacks(parent); err != nil {
		t.Fatal(err)
	}
	repinChild(t, parent, "orders", orders)
	repinChild(t, parent, "payments", payments)
	return
}

// relaunderChildPack recomputes and rewrites the child pack's content_hash
// from the (edited) files on disk and re-pins the packmap: the laundering
// move whose reach PACK-6 bounds via the parent-side pin.
func relaunderChildPack(t *testing.T, child string) {
	t.Helper()
	files, err := pack.PackFilesOnDisk(child)
	if err != nil {
		t.Fatal(err)
	}
	h := pack.ContentHash(files)
	manifestPath := filepath.Join(child, "pack", "pack.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, manifestPath, contentHashLineRe.ReplaceAllString(string(data), "content_hash: "+h))
	pmPath := filepath.Join(child, "packmap.yaml")
	pdata, err := os.ReadFile(pmPath)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, pmPath, packHashLineRe.ReplaceAllString(string(pdata), "pack_hash: "+h))
}

func gateOutput(t *testing.T, g *gates.Gate) string {
	t.Helper()
	var buf bytes.Buffer
	g.Emit(&buf)
	return buf.String()
}

// Baseline: the regenerated, re-pinned fixture is green on all three designs.
// Everything below mutates on top of this.
func TestRegeneratedSplitFixtureIsGreen(t *testing.T) {
	parent, orders, payments := regenSplitFixture(t)
	for _, d := range []string{parent, orders, payments} {
		g := gates.CheckPack(d)
		if len(g.Errs) != 0 || len(g.Drift) != 0 {
			t.Errorf("%s: errs=%v drift=%v", d, g.Errs, g.Drift)
		}
	}
}

// ------------------------- PACK-1 / SCALE-1 -------------------------------

const secondEventTable = `
## 9. Refund event contracts (added later, second table)

| event | producer | consumer | payload | delivery | ordering | dedupe |
|---|---|---|---|---|---|---|
| refundRequested | orders | payments | Payment.id | at-least-once | none | Payment.id |
| refundSettled | payments | orders | Payment.id | at-least-once | none | Payment.id |
`

func appendSecondEventTable(t *testing.T, parent string) {
	t.Helper()
	arch := filepath.Join(parent, "ARCHITECTURE.md")
	data, err := os.ReadFile(arch)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, arch, string(data)+secondEventTable)
}

// A later event-contract table used to be silently excluded from every pack
// while the packs still claimed boundary completeness. All matching tables
// merge.
func TestSecondEventTableRowsAreIncluded(t *testing.T) {
	parent, _, _ := splitFixture(t)
	appendSecondEventTable(t, parent)
	packs, err := pack.GeneratePacks(parent)
	if err != nil {
		t.Fatal(err)
	}
	pay := packs["payments"]["events.md"]
	for _, want := range []string{
		"| refundRequested | consumes | orders |",
		"| refundSettled | produces | orders |",
	} {
		if !strings.Contains(pay, want) {
			t.Errorf("payments events.md missing %q:\n%s", want, pay)
		}
	}
	if got := pack.CountBoundaryEvents(pay); got != 5 {
		t.Errorf("payments boundary events = %d, want 5 (second table merged)", got)
	}
	if got := pack.CountBoundaryEvents(packs["orders"]["events.md"]); got != 5 {
		t.Errorf("orders boundary events = %d, want 5 (second table merged)", got)
	}
}

// machinery scale shares the locator via pack.EventRows; the size signal must
// count every table's rows.
func TestEventRowsMergeAllTables(t *testing.T) {
	parent, _, _ := splitFixture(t)
	appendSecondEventTable(t, parent)
	if got := len(pack.EventRows(parent)); got != 5 {
		t.Errorf("EventRows = %d rows, want 5", got)
	}
}

// ------------------------- PACK-2 -----------------------------------------

// A component claimed by two subsystems flipped boundary event directions in
// the duplicating pack (consumes became produces, deleting the handling
// obligation). Claims are exactly-once.
func TestDuplicateComponentClaimFailsLoad(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"),
		"components: [payments]", "components: [payments, orders]")
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "claimed by both") {
		t.Fatalf("duplicate component claim accepted: %v", err)
	}
}

// A row whose producer and consumer both belong to one subsystem has no
// boundary direction; it must fail generation, not pick one silently.
func TestIntraSubsystemEventRowFailsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "ARCHITECTURE.md"),
		"| request | orders | payments |", "| request | orders | orders |")
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "both belong to subsystem 'orders'") {
		t.Fatalf("intra-subsystem event row accepted: %v", err)
	}
}

// ------------------------- PACK-3 -----------------------------------------

const undelegatedInvariant = `invariants:
  - {id: no-double-refund, definition: "A captured payment is refunded at most once. Cross-subsystem; MUST be enforced somewhere."}
  - {id: no-ship-without-capture`

func addUndelegatedInvariant(t *testing.T, parent string) {
	t.Helper()
	editFile(t, filepath.Join(parent, "checkout.modelith.yaml"),
		"invariants:\n  - {id: no-ship-without-capture", undelegatedInvariant)
}

// A top-level invariant delegated to no subsystem enters no pack and no child
// and the machine-less parent enforces nothing: it must fail generation.
func TestUndelegatedTopLevelInvariantFailsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	addUndelegatedInvariant(t, parent)
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "delegated to no subsystem") {
		t.Fatalf("undelegated top-level invariant accepted: %v", err)
	}
	if !strings.Contains(err.Error(), "no-double-refund") {
		t.Errorf("finding does not name the invariant: %v", err)
	}
}

// retained: {id: reason} declares the invariant stays parent-enforced.
func TestRetainedInvariantAllowsGeneration(t *testing.T) {
	parent, _, _ := splitFixture(t)
	addUndelegatedInvariant(t, parent)
	editFile(t, filepath.Join(parent, "decomposition.yaml"),
		"decomposition_version: 1\n",
		"decomposition_version: 1\nretained:\n  no-double-refund: \"enforced by the parent-level refund saga until the refunds subsystem exists\"\n")
	if _, err := pack.GeneratePacks(parent); err != nil {
		t.Fatalf("retained invariant still fails generation: %v", err)
	}
}

func TestRetainedValidation(t *testing.T) {
	cases := []struct {
		name, retained, wantSub string
	}{
		{"unknown id", "retained:\n  no-such-invariant: \"reason\"\n", "unknown invariant"},
		{"empty reason", "retained:\n  no-ship-without-capture: \"\"\n", "non-empty"},
		{"also delegated", "retained:\n  no-ship-without-capture: \"parent keeps it\"\n", "both retained and delegated"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parent, _, _ := splitFixture(t)
			editFile(t, filepath.Join(parent, "decomposition.yaml"),
				"decomposition_version: 1\n", "decomposition_version: 1\n"+c.retained)
			_, err := pack.GeneratePacks(parent)
			if err == nil || !strings.Contains(err.Error(), c.wantSub) {
				t.Fatalf("retained %s accepted: %v", c.name, err)
			}
		})
	}
}

// ------------------------- PACK-4 -----------------------------------------

// A child deleting a frozen attribute (one the boundary event payload cells
// name) used to pass G5; every pack attribute must exist in the child.
func TestChildDroppingOwnedAttributeFailsG5(t *testing.T) {
	_, _, payments := regenSplitFixture(t)
	editFile(t, filepath.Join(payments, "payments.modelith.yaml"),
		"      - {name: orderId, type: string}\n", "")
	g := gates.CheckPack(payments)
	if !containsAny(g.Errs, "attribute orderId is missing") {
		t.Errorf("dropped frozen attribute passed G5: %v", g.Errs)
	}
}

// A child rewriting a frozen entity invariant's statement used to pass G5.
// Extension (the shipped example appends enforcement detail) stays legal; a
// rewrite fails.
func TestChildRewritingEntityInvariantFailsG5(t *testing.T) {
	_, _, payments := regenSplitFixture(t)
	editFile(t, filepath.Join(payments, "payments.modelith.yaml"),
		`- {id: payment-single-capture, definition: "A payment is captured at most once. Structural: capture is only handled in Requested."}`,
		`- {id: payment-single-capture, definition: "WEAKENED: a payment may be captured many times; retries are fine."}`)
	g := gates.CheckPack(payments)
	if !containsAny(g.Errs, "invariant payment-single-capture drifted from the pack") {
		t.Errorf("rewritten frozen entity invariant passed G5: %v", g.Errs)
	}
}

// Deleting the frozen entity invariant outright must fail too.
func TestChildDeletingEntityInvariantFailsG5(t *testing.T) {
	_, _, payments := regenSplitFixture(t)
	editFile(t, filepath.Join(payments, "payments.modelith.yaml"),
		"    invariants:\n      - {id: payment-single-capture", "    invariants: []\n    _was:\n      - {id: payment-single-capture")
	g := gates.CheckPack(payments)
	if !containsAny(g.Errs, "invariant payment-single-capture is missing") {
		t.Errorf("deleted frozen entity invariant passed G5: %v", g.Errs)
	}
}

// ------------------------- PACK-5 -----------------------------------------

// After a subsystem rename the old packs/<id>.pack directory survived forever
// and its orphaned child stayed green. The parent flags every pack dir with
// no current subsystem; the child alone cannot see it (which is why the
// parent must).
func TestOrphanedPackDirAfterRenameFailsParentG5(t *testing.T) {
	parent, _, payments := regenSplitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"), "  - id: payments\n", "  - id: billing\n")
	editFile(t, filepath.Join(parent, "decomposition.yaml"), "    child_design: ../../payments/design\n", "")
	g := gates.CheckPack(parent)
	if !containsAny(g.Errs, "packs/payments.pack") {
		t.Errorf("orphaned pack dir not flagged: %v", g.Errs)
	}
	if !containsAny(g.Errs, "no current subsystem") {
		t.Errorf("orphan finding does not explain itself: %v", g.Errs)
	}
	// the orphaned child keeps validating against its copied pack; only the
	// parent can flag the orphan, which is exactly what the check is for
	cg := gates.CheckPack(payments)
	if len(cg.Errs) != 0 {
		t.Errorf("orphaned child reports its own errors (fixture drift?): %v", cg.Errs)
	}
}

// ------------------------- PACK-6 (documented limit + revision) ------------

// With the parent pin in place, a child that edits its pack and recomputes
// the content hash is caught by the pin. Without child_design there is no
// pin: the child stays green and the parent only warns. That window is the
// documented hash-laundering limit; pack_revision narrows the amendment
// blindness, not the laundering itself.
func TestHashLaunderingCaughtWhenPinnedLimitWhenNot(t *testing.T) {
	parent, orders, _ := regenSplitFixture(t)
	editFile(t, filepath.Join(orders, "pack", "pack.yaml"),
		"delegated_invariants:\n  - no-ship-without-capture\n", "delegated_invariants: []\n")
	relaunderChildPack(t, orders)
	if _, err := pack.WriteRefinement(orders); err != nil {
		t.Fatal(err)
	}
	// pinned: the parent catches the laundered hash
	g := gates.CheckPack(parent)
	if !containsAny(g.Errs, "was built against pack") {
		t.Errorf("pinned laundering not caught at the parent: %v", g.Errs)
	}
	// unpinned: the documented limit; child green, parent warns
	editFile(t, filepath.Join(parent, "decomposition.yaml"), "    child_design: ../../orders/design\n", "")
	g = gates.CheckPack(parent)
	if containsAny(g.Errs, "was built against pack") {
		t.Errorf("unpinned parent still claims a pin check: %v", g.Errs)
	}
	if !containsAny(g.Warns, "declares no child_design") {
		t.Errorf("unpinned subsystem does not warn: %v", g.Warns)
	}
	cg := gates.CheckPack(orders)
	if len(cg.Errs) != 0 {
		t.Errorf("documented limit shifted: unpinned laundered child now red (update the docs note): %v", cg.Errs)
	}
}

// revision flows from decomposition.yaml into every manifest as
// pack_revision, both G5 sides verify and print it.
func TestPackRevisionFlowsToManifestAndGates(t *testing.T) {
	parent, orders, payments := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"),
		"decomposition_version: 1\n", "decomposition_version: 1\nrevision: 3\n")
	if _, err := pack.WritePacks(parent); err != nil {
		t.Fatal(err)
	}
	repinChild(t, parent, "orders", orders)
	repinChild(t, parent, "payments", payments)
	manifest, err := pack.LoadPackManifest(orders)
	if err != nil {
		t.Fatal(err)
	}
	if got := manifest.Get2("pack_revision"); got == nil || string(got.AsNumber()) != "3" {
		t.Errorf("pack_revision = %s, want 3", ir.Repr(got))
	}
	pg := gates.CheckPack(parent)
	if len(pg.Errs) != 0 || len(pg.Drift) != 0 {
		t.Fatalf("revision fixture not green at parent: errs=%v drift=%v", pg.Errs, pg.Drift)
	}
	if out := gateOutput(t, pg); !strings.Contains(out, "pack revision 3") {
		t.Errorf("parent checked line does not print the revision:\n%s", out)
	}
	cg := gates.CheckPack(orders)
	if len(cg.Errs) != 0 || len(cg.Drift) != 0 {
		t.Fatalf("revision fixture not green at child: errs=%v drift=%v", cg.Errs, cg.Drift)
	}
	if out := gateOutput(t, cg); !strings.Contains(out, "pack revision 3") {
		t.Errorf("child checked line does not print the revision:\n%s", out)
	}
}

func TestInvalidRevisionFailsLoad(t *testing.T) {
	for _, bad := range []string{"revision: 0\n", "revision: -2\n", "revision: 1.5\n"} {
		parent, _, _ := splitFixture(t)
		editFile(t, filepath.Join(parent, "decomposition.yaml"),
			"decomposition_version: 1\n", "decomposition_version: 1\n"+bad)
		if _, err := pack.GeneratePacks(parent); err == nil || !strings.Contains(err.Error(), "revision") {
			t.Errorf("%q accepted: %v", strings.TrimSpace(bad), err)
		}
	}
}

// A copied pack without pack_revision is a pre-revision artifact; the child
// gate demands regeneration rather than guessing an amendment history.
func TestChildManifestWithoutPackRevisionFailsG5(t *testing.T) {
	_, orders, _ := regenSplitFixture(t)
	editFile(t, filepath.Join(orders, "pack", "pack.yaml"), "pack_revision: 1\n", "")
	relaunderChildPack(t, orders)
	g := gates.CheckPack(orders)
	if !containsAny(g.Errs, "pack_revision") {
		t.Errorf("manifest without pack_revision passed child G5: %v", g.Errs)
	}
}

// ------------------------- PACK-7 -----------------------------------------

const shim2Machine = `{
 "id": "shim2",
 "_role": "operational",
 "initial": "Open",
 "states": {
  "Open": {"on": {"go": {"target": "Paid"}}},
  "Paid": {"on": {"finish": {"target": "Done"}}},
  "Done": {"type": "final"}
 }
}
`

func rebindPackmap(t *testing.T, child, machine string) {
	t.Helper()
	manifest, err := pack.LoadPackManifest(child)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(child, "packmap.yaml"),
		"subsystem: orders\npack_hash: "+manifest.GetString("content_hash")+
			"\nmachine: "+machine+"\nmapping:\n  Open: Open\n  Paid: Paid\n  Done: Done\n")
}

// The packmap must bind a machine with a real stake in the contract: the
// lifecycle machine of a pack-owned entity, or one that touches a boundary
// event. A machine that touches neither proves nothing about the subsystem.
func TestShimPackmapMachineFailsRefine(t *testing.T) {
	_, orders, _ := regenSplitFixture(t)
	mustWrite(t, filepath.Join(orders, "machines", "Shim2.machine.json"), shim2Machine)
	rebindPackmap(t, orders, "Shim2")
	_, err := pack.GenerateRefinement(orders)
	if err == nil || !strings.Contains(err.Error(), "neither the lifecycle machine") {
		t.Fatalf("contract-stake-free shim accepted by refine: %v", err)
	}
	if !containsAny(gates.CheckPack(orders).Errs, "neither the lifecycle machine") {
		t.Error("contract-stake-free shim passed child G5")
	}
}

// A refinement artifact a fresh generation would not produce is a stale
// binding from a previous packmap; the old one-way diff never looked.
func TestStaleRefinementExtraFailsG5(t *testing.T) {
	_, orders, _ := regenSplitFixture(t)
	mustWrite(t, filepath.Join(orders, "formal", "GhostPackRefinement.tla"), "---- MODULE GhostPackRefinement ----\n====\n")
	if !containsAny(gates.CheckPack(orders).Errs, "formal/GhostPackRefinement.tla") {
		t.Error("stale refinement artifact passed child G5")
	}
}

// Rebinding the packmap to a copied contract shim regenerates
// ShimPackRefinement.* but used to leave OrderPackRefinement.* committed and
// unchecked: the proof the neighbors rely on silently stopped being checked.
func TestReboundPackmapLeavesStaleArtifactsFlagged(t *testing.T) {
	parent, orders, _ := regenSplitFixture(t)
	data, err := os.ReadFile(filepath.Join(parent, "contracts", "OrdersContract.machine.json"))
	if err != nil {
		t.Fatal(err)
	}
	// a byte-copy of the contract machine handles boundary events, so the
	// binding check alone cannot refuse it; the stale-artifact diff must
	mustWrite(t, filepath.Join(orders, "machines", "Shim.machine.json"),
		strings.Replace(string(data), `"ordersContract"`, `"shim"`, 1))
	rebindPackmap(t, orders, "Shim")
	if _, err := pack.WriteRefinement(orders); err != nil {
		t.Fatal(err)
	}
	g := gates.CheckPack(orders)
	if !containsAny(g.Errs, "formal/OrderPackRefinement.tla") {
		t.Errorf("stale OrderPackRefinement.tla not flagged after rebinding: %v", g.Errs)
	}
	if !containsAny(g.Errs, "formal/OrderPackRefinement.cfg") {
		t.Errorf("stale OrderPackRefinement.cfg not flagged after rebinding: %v", g.Errs)
	}
}

// ------------------------- PACK-8 -----------------------------------------

// A newline in the waiver reason could forge the generated events.md count
// line; reasons are single-line by contract.
func TestWaiverReasonWithNewlineFailsLoad(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"),
		"    delegated_invariants: []\n",
		"    delegated_invariants: []\n    boundary_events:\n      none: \"legacy note follows.\\nBoundary events: 9 were retired in v1; none remain\"\n")
	_, err := pack.GeneratePacks(parent)
	if err == nil || !strings.Contains(err.Error(), "single line") {
		t.Fatalf("multi-line waiver reason accepted: %v", err)
	}
}

// Defense in depth: even with an injected count line earlier in the file, the
// counter anchors on the last line, which is the generated one.
func TestCountBoundaryEventsAnchorsOnLastLine(t *testing.T) {
	forged := "# Boundary event contracts: payments\n\nreason mentioning\nBoundary events: 9 were retired\n\nBoundary events: 0 (waived)\n"
	if got := pack.CountBoundaryEvents(forged); got != 0 {
		t.Errorf("CountBoundaryEvents anchored on an injected line: got %d, want 0", got)
	}
}

// ------------------------- PACK-9 -----------------------------------------

// A subdirectory smuggled into the frozen child pack/ escaped the content
// hash entirely; any directory entry is a hard error.
func TestPackSubdirectoryFailsG5(t *testing.T) {
	_, orders, _ := regenSplitFixture(t)
	extra := filepath.Join(orders, "pack", "extra")
	if err := os.MkdirAll(extra, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(extra, "smuggled.md"), "unhashed content the child can cite\n")
	g := gates.CheckPack(orders)
	if !containsAny(g.Errs, "directory") {
		t.Errorf("smuggled pack subdirectory passed child G5: %v", g.Errs)
	}
	if _, err := pack.PackFilesOnDisk(orders); err == nil {
		t.Error("PackFilesOnDisk silently skipped a directory")
	}
}

// ------------------------- PACK-10 ----------------------------------------

// A delegated invariant id containing ": " must survive the manifest
// round-trip as a string (writeList quotes what needs quoting).
func TestColonInvariantIDRoundTripsThroughManifest(t *testing.T) {
	parent, _, _ := splitFixture(t)
	editFile(t, filepath.Join(parent, "checkout.modelith.yaml"),
		"invariants:\n  - {id: no-ship-without-capture",
		"invariants:\n  - {id: \"cap: no-ship\", definition: \"colon id round-trip probe\"}\n  - {id: no-ship-without-capture")
	editFile(t, filepath.Join(parent, "decomposition.yaml"),
		"delegated_invariants: [no-ship-without-capture]",
		"delegated_invariants: [no-ship-without-capture, \"cap: no-ship\"]")
	packs, err := pack.GeneratePacks(parent)
	if err != nil {
		t.Fatal(err)
	}
	manifest := packs["orders"]["pack.yaml"]
	if !strings.Contains(manifest, `- "cap: no-ship"`) {
		t.Errorf("colon id not quoted in the manifest:\n%s", manifest)
	}
	v, err := ir.LoadYAML([]byte(manifest))
	if err != nil || v.AsObject() == nil {
		t.Fatal("manifest does not reparse")
	}
	var ids []string
	for _, e := range v.AsObject().Get2("delegated_invariants").AsArray() {
		if e != nil && e.Kind == ir.KindString {
			ids = append(ids, e.AsString())
		}
	}
	if len(ids) != 2 || ids[1] != "cap: no-ship" {
		t.Errorf("delegated_invariants round-trip = %v, want both string ids", ids)
	}
}

// A manifest whose delegated_invariants holds a non-string entry (the old
// unquoted colon-id emission) is an error, never a silent drop.
func TestNonStringDelegatedInvariantEntryFailsG5(t *testing.T) {
	_, orders, _ := regenSplitFixture(t)
	editFile(t, filepath.Join(orders, "pack", "pack.yaml"),
		"delegated_invariants:\n  - no-ship-without-capture\n",
		"delegated_invariants:\n  - no-ship-without-capture\n  - cap: no-ship\n")
	relaunderChildPack(t, orders)
	g := gates.CheckPack(orders)
	if !containsAny(g.Errs, "not a plain string") {
		t.Errorf("non-string delegated_invariants entry silently dropped: %v", g.Errs)
	}
}

// ------------------------- parent bookkeeping ------------------------------

// child_design pointing at a directory that does not exist is a load-visible
// error at the parent (the pin cannot be checked).
func TestChildDesignNonexistentDirFailsParentG5(t *testing.T) {
	parent, _, _ := regenSplitFixture(t)
	editFile(t, filepath.Join(parent, "decomposition.yaml"),
		"child_design: ../../payments/design", "child_design: ../../nonexistent/design")
	g := gates.CheckPack(parent)
	if !containsAny(g.Errs, "no readable packmap.yaml") {
		t.Errorf("nonexistent child_design not flagged: %v", g.Errs)
	}
}

// ------------------------- SCALE-2 ----------------------------------------

// scale must refuse a directory that is not a design instead of measuring
// emptiness and recommending a single-run design.
func TestLooksLikeDesignDirGuard(t *testing.T) {
	empty := t.TempDir()
	if pack.LooksLikeDesignDir(empty) {
		t.Error("an empty directory passes the design-dir guard")
	}
	parent, _, _ := splitFixture(t)
	if !pack.LooksLikeDesignDir(parent) {
		t.Error("the checkout-split parent fails the design-dir guard")
	}
	machinesOnly := t.TempDir()
	if err := os.MkdirAll(filepath.Join(machinesOnly, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !pack.LooksLikeDesignDir(machinesOnly) {
		t.Error("a machines/-only directory fails the design-dir guard")
	}
	decompOnly := t.TempDir()
	mustWrite(t, filepath.Join(decompOnly, "decomposition.yaml"), "decomposition_version: 1\nsubsystems: []\n")
	if !pack.LooksLikeDesignDir(decompOnly) {
		t.Error("a decomposition-only directory fails the design-dir guard")
	}
}

// ------------------- wave-1 handoff: NG-9 for the child gate ---------------

// A child artifact that exists but cannot be read is a hard ERROR naming the
// path, never silently an empty file (the readOrEmpty call sites G5's child
// checks used to share with the Python port).
func TestUnreadableChildArtifactsAreHardErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not deny reads")
	}
	unreadable := func(t *testing.T, path string) {
		t.Helper()
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	}
	t.Run("matrix", func(t *testing.T) {
		_, orders, _ := regenSplitFixture(t)
		unreadable(t, filepath.Join(orders, "machines", "Order.matrix.md"))
		g := gates.CheckPack(orders)
		if !containsAny(g.Errs, "Order.matrix.md is unreadable") {
			t.Errorf("unreadable matrix not a hard error naming the path: %v", g.Errs)
		}
	})
	t.Run("BUILD.md", func(t *testing.T) {
		_, orders, _ := regenSplitFixture(t)
		unreadable(t, filepath.Join(orders, "BUILD.md"))
		g := gates.CheckPack(orders)
		if !containsAny(g.Errs, "BUILD.md is unreadable") {
			t.Errorf("unreadable BUILD.md not a hard error naming the path: %v", g.Errs)
		}
	})
}

// ------------------- P-F10: refinement artifact version stamps -------------

// A committed refinement artifact stamped by another machinery version,
// content otherwise fresh, is not drift (G5 strips the stamp line from both
// sides of the freshness diff).
func TestRefinementVersionOnlySkewIsNotDrift(t *testing.T) {
	_, orders, _ := regenSplitFixture(t)
	path := filepath.Join(orders, "formal", "OrderPackRefinement.tla")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := strings.Replace(string(data), version.TLAStamp(), `\* machinery-version: v0.0.2`, 1)
	if rewritten == string(data) {
		t.Fatal("committed refinement artifact carries no stamp to rewrite")
	}
	mustWrite(t, path, rewritten)
	g := gates.CheckPack(orders)
	if len(g.Drift) != 0 || len(g.Errs) != 0 {
		t.Fatalf("version-only skew reported as drift: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if note := gates.VersionSkewNote([]*gates.Gate{g}); !strings.Contains(note, "v0.0.2") {
		t.Errorf("skew note = %q, want v0.0.2 named", note)
	}
}
