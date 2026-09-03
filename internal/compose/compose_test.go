package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/artifactset"
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

func TestGuardedCurrentComposeArtifactsRejectForeignTargetHalves(t *testing.T) {
	files := map[string][]byte{"Checkout.tla": []byte("generated tla\n"), "Checkout.cfg": []byte("generated cfg\n")}
	for _, target := range []string{"Checkout.tla", "Checkout.cfg"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			foreign := []byte("hand-authored sentinel\n")
			if err := os.WriteFile(filepath.Join(dir, target), foreign, 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := guardedCurrentComposeArtifacts(dir, "checkout.composition.yaml", files); err == nil {
				t.Fatalf("foreign %s was accepted for replacement", target)
			}
			if got, err := os.ReadFile(filepath.Join(dir, target)); err != nil || string(got) != string(foreign) {
				t.Fatalf("foreign target changed: %q, %v", got, err)
			}
		})
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
	comp, err := ir.LoadYAML([]byte("composition: X\ncoordinator: C\naggregates:\n  a:\n    states: [S]\n    initial: S\nsequence: []\n"))
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

func TestRunWrittenReconcilesPriorCompositionNameOwnedBySameSource(t *testing.T) {
	design := filepath.Join(t.TempDir(), "design")
	machines := filepath.Join(design, "machines")
	formal := filepath.Join(design, "formal")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	machineBody, err := os.ReadFile(filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	if err != nil {
		t.Fatal(err)
	}
	machine := filepath.Join(machines, "FulfillmentSaga.machine.json")
	if err := os.WriteFile(machine, machineBody, 0o644); err != nil {
		t.Fatal(err)
	}
	compositionBody, err := os.ReadFile(filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	composition := filepath.Join(formal, "checkout.composition.yaml")
	if err := os.WriteFile(composition, compositionBody, 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := RunWritten(composition, machine, out); err != nil {
		t.Fatal(err)
	}
	renamed := strings.Replace(string(compositionBody), "composition: checkout", "composition: checkout2", 1)
	if renamed == string(compositionBody) {
		t.Fatal("fixture composition name was not replaced")
	}
	if err := os.WriteFile(composition, []byte(renamed), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RunWritten(composition, machine, out); err != nil {
		t.Fatal(err)
	}
	for _, stale := range []string{"Checkout.tla", "Checkout.cfg"} {
		if _, err := os.Lstat(filepath.Join(out, stale)); !os.IsNotExist(err) {
			t.Fatalf("stale owned composition artifact %s remains: %v", stale, err)
		}
	}
	for _, current := range []string{"Checkout2.tla", "Checkout2.cfg"} {
		if _, err := os.Stat(filepath.Join(out, current)); err != nil {
			t.Fatalf("current composition artifact %s missing: %v", current, err)
		}
	}
	purchase := filepath.Join(formal, "purchase.composition.yaml")
	purchaseBody := strings.Replace(renamed, "composition: checkout2", "composition: purchase", 1)
	if err := os.WriteFile(purchase, []byte(purchaseBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(composition); err != nil {
		t.Fatal(err)
	}
	if _, err := RunWritten(purchase, machine, out); err != nil {
		t.Fatal(err)
	}
	for _, stale := range []string{"Checkout2.tla", "Checkout2.cfg"} {
		if _, err := os.Lstat(filepath.Join(out, stale)); !os.IsNotExist(err) {
			t.Fatalf("orphaned source-owned composition artifact %s remains: %v", stale, err)
		}
	}
	for _, current := range []string{"Purchase.tla", "Purchase.cfg"} {
		if _, err := os.Stat(filepath.Join(out, current)); err != nil {
			t.Fatalf("renamed composition artifact %s missing: %v", current, err)
		}
	}
}

func TestRunWrittenFailsClosedOnAmbiguousExternalCompositionOwnership(t *testing.T) {
	machine := filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json")
	body, err := os.ReadFile(filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name       string
		secondBase string
	}{
		{name: "same basename in another external directory", secondBase: "checkout.composition.yaml"},
		{name: "external source move and rename", secondBase: "purchase.composition.yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			firstDir := t.TempDir()
			secondDir := t.TempDir()
			first := filepath.Join(firstDir, "checkout.composition.yaml")
			second := filepath.Join(secondDir, tc.secondBase)
			if err := os.WriteFile(first, body, 0o644); err != nil {
				t.Fatal(err)
			}
			purchase := strings.Replace(string(body), "composition: checkout", "composition: purchase", 1)
			if err := os.WriteFile(second, []byte(purchase), 0o644); err != nil {
				t.Fatal(err)
			}
			out := t.TempDir()
			if _, err := RunWritten(first, machine, out); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(filepath.Join(out, "Checkout.tla"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := RunWritten(second, machine, out); err == nil || !strings.Contains(err.Error(), "cannot safely reconcile") {
				t.Fatalf("ambiguous external ownership was not rejected: %v", err)
			}
			after, err := os.ReadFile(filepath.Join(out, "Checkout.tla"))
			if err != nil || string(after) != string(before) {
				t.Fatalf("existing external-owner output changed: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(out, "Purchase.tla")); !os.IsNotExist(err) {
				t.Fatalf("ambiguous external owner published a subset: %v", err)
			}
		})
	}
}

func TestRunWrittenRejectsAtomicReplacementOfPlannedStaleComposition(t *testing.T) {
	design := filepath.Join(t.TempDir(), "design")
	machines, formal := filepath.Join(design, "machines"), filepath.Join(design, "formal")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	machineBody, _ := os.ReadFile(filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	compositionBody, _ := os.ReadFile(filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml"))
	machine := filepath.Join(machines, "FulfillmentSaga.machine.json")
	composition := filepath.Join(formal, "checkout.composition.yaml")
	if err := os.WriteFile(machine, machineBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(composition, compositionBody, 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := RunWritten(composition, machine, out); err != nil {
		t.Fatal(err)
	}
	renamed := strings.Replace(string(compositionBody), "composition: checkout", "composition: checkout2", 1)
	if err := os.WriteFile(composition, []byte(renamed), 0o644); err != nil {
		t.Fatal(err)
	}
	prior := runWrittenAfterStalePlan
	runWrittenAfterStalePlan = func() { atomicReplaceFile(t, filepath.Join(out, "Checkout.tla")) }
	defer func() { runWrittenAfterStalePlan = prior }()
	if _, err := RunWritten(composition, machine, out); err == nil || !strings.Contains(err.Error(), "ownership plan") {
		t.Fatalf("atomic stale replacement was accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "Checkout.tla")); err != nil {
		t.Fatalf("replacement was not preserved: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(out, "Checkout2.tla")); !os.IsNotExist(err) {
		t.Fatalf("new composition was partially installed: %v", err)
	}
}

func TestComposeOwnershipGrammarAndCanonicalHeader(t *testing.T) {
	for _, name := range []string{"bad,name.composition.yaml", "bad + owner.composition.yaml", "bad\nname.composition.yaml", "bad\x01name.composition.yaml"} {
		if err := validateComposeOwnerBase(name); err == nil {
			t.Errorf("hostile ownership filename accepted: %q", name)
		}
	}
	out := t.TempDir()
	source := t.TempDir()
	manual := "---- MODULE Manual ----\n\\* machinery-version: v1\nEXTENDS Naturals\nManual == \\\"\\* GENERATED by machinery compose from checkout.composition.yaml,\\\"\n====\n"
	if err := os.WriteFile(filepath.Join(out, "Manual.tla"), []byte(manual), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := staleOwnedComposeArtifacts(out, source, "checkout.composition.yaml", map[string][]byte{})
	if err != nil || len(stale) != 0 {
		t.Fatalf("handwritten quoted marker authorized deletion: stale=%v err=%v", stale, err)
	}
}

func TestStaleOwnedComposeArtifactsConvergesMissingAnchorConfig(t *testing.T) {
	out := t.TempDir()
	body := "\\* machinery-version: v1\nSPECIFICATION Spec\nINVARIANT TypeOK\nINVARIANT Inv_CleanCompensation\nPROPERTY Live_Terminates\n"
	if err := os.WriteFile(filepath.Join(out, "Gone.cfg"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := staleOwnedComposeArtifacts(out, t.TempDir(), "current.composition.yaml", map[string][]byte{})
	if err != nil || len(stale) != 1 || stale[0].Name != "Gone.cfg" {
		t.Fatalf("missing-anchor generated cfg not reconciled: stale=%v err=%v", stale, err)
	}
	if err := artifactset.ReconcilePlanned(out, map[string][]byte{}, stale); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(out, "Gone.cfg")); !os.IsNotExist(err) {
		t.Fatalf("orphan cfg retained: %v", err)
	}
}

func atomicReplaceFile(t *testing.T, path string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	temp := path + ".replacement"
	if err := os.WriteFile(temp, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temp, path); err != nil {
		t.Fatal(err)
	}
}

func TestCompositionSchemaRejectsUnknownAndWrongTypedClaims(t *testing.T) {
	base, err := os.ReadFile(filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, replace, with, want string
	}{
		{"root typo", "coordinator: FulfillmentSaga", "coordinator: FulfillmentSaga\nmax_retry: 3", "max_retry"},
		{"aggregate typo", "initial: Free", "inti" + "al: Free", "inti" + "al"},
		{"sequence typo", "aggregate: reservation", "ag" + "regate: reservation", "ag" + "regate"},
		{"undo typo", "undo: { to: Released }", "undo: { too: Released }", "too"},
		{"wrong invariant type", `'payment = "Captured" => reservation # "Free"'`, "[not, a, string]", "wrong type"},
	}
	machine, err := ir.LoadMachineJSON(filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.Replace(string(base), tc.replace, tc.with, 1)
			comp, loadErr := ir.LoadYAML([]byte(body))
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if _, _, _, err := Generate(comp, machine, "FulfillmentSaga.machine.json"); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid composition accepted: %v", err)
			}
		})
	}
}

func TestCompositionSchemaFirstUnknownKeyIsDeterministic(t *testing.T) {
	comp, err := ir.LoadYAML([]byte("composition: x\ncoordinator: C\naggregates: {}\nsequence: []\nz_bad: 1\na_bad: 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1_000; i++ {
		err := validateCompositionSchema(comp)
		if err == nil || !strings.Contains(err.Error(), "'a_bad'") {
			t.Fatalf("validation %d = %v", i, err)
		}
	}
}

func TestRunWrittenInvalidDiagnosticIsStableAndLogical(t *testing.T) {
	source := filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml")
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "checkout.composition.yaml")
	body = []byte(strings.Replace(string(body), "coordinator: FulfillmentSaga", "coordinator: FulfillmentSaga\nmax_retry: 3", 1))
	if err := os.WriteFile(input, body, 0o644); err != nil {
		t.Fatal(err)
	}
	machine := filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json")
	_, firstErr := RunWritten(input, machine, t.TempDir())
	_, secondErr := RunWritten(input, machine, t.TempDir())
	if firstErr == nil || secondErr == nil || firstErr.Error() != secondErr.Error() {
		t.Fatalf("invalid composition diagnostic is unstable:\nfirst: %v\nsecond: %v", firstErr, secondErr)
	}
	if strings.Contains(firstErr.Error(), "machinery-design-source-") || strings.Contains(firstErr.Error(), "machinery-input-snapshot-") {
		t.Fatalf("invalid composition diagnostic leaks private snapshot: %v", firstErr)
	}
}

func TestRunWrittenRejectsSymlinkedAndMutatingCompositionInput(t *testing.T) {
	machine := filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json")
	source := filepath.Join(repoRoot(), "examples/fulfillment/design/formal/checkout.composition.yaml")
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		link := filepath.Join(dir, "checkout.composition.yaml")
		if err := os.Symlink(source, link); err != nil {
			t.Fatal(err)
		}
		if _, err := RunWritten(link, machine, t.TempDir()); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("symlinked composition accepted: %v", err)
		}
	})
	t.Run("mutation", func(t *testing.T) {
		dir := t.TempDir()
		input := filepath.Join(dir, "checkout.composition.yaml")
		if err := os.WriteFile(input, body, 0o644); err != nil {
			t.Fatal(err)
		}
		prior := runWrittenAfterInputSnapshot
		runWrittenAfterInputSnapshot = func() {
			if err := os.WriteFile(input, append(body, []byte("\n# concurrent edit\n")...), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		defer func() { runWrittenAfterInputSnapshot = prior }()
		out := filepath.Join(dir, "out")
		if _, err := RunWritten(input, machine, out); err == nil || !strings.Contains(err.Error(), "external input") {
			t.Fatalf("mutating composition accepted: %v", err)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatalf("mutation wrote output: %v", err)
		}
	})
	t.Run("symlinked output directory", func(t *testing.T) {
		dir := t.TempDir()
		outside := t.TempDir()
		out := filepath.Join(dir, "out")
		if err := os.Symlink(outside, out); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, err := RunWritten(source, machine, out); err == nil || !strings.Contains(err.Error(), "unsafe output directory") {
			t.Fatalf("symlinked output directory accepted: %v", err)
		}
	})
}
