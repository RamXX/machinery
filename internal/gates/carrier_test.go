package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Gc-carrier fixtures ---

// carrierModel declares three invariants: one carried by preserves, two not.
const carrierModel = `kind: modelith
version: 1
entities:
  Widget:
    actions:
      - name: publish
        preserves: [widget-owned]
    invariants:
      - id: widget-owned
      - id: widget-immutable
invariants:
  - id: audit-verifiable
`

func writeCarrierDesign(t *testing.T, files map[string]string) string {
	t.Helper()
	design := t.TempDir()
	if _, ok := files["domain.modelith.yaml"]; !ok {
		files["domain.modelith.yaml"] = carrierModel
	}
	for name, content := range files {
		mustWrite(t, filepath.Join(design, name), content)
	}
	return design
}

func hasWarn(g *Gate, needle string) bool {
	for _, w := range g.Warns {
		if strings.Contains(w, needle) {
			return true
		}
	}
	return false
}

// --- Gc: the reconciliation itself ---

func TestGcUncarriedInvariantIsAnError(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{})
	g := CheckCarriers(design)
	if !hasErr(g, "'widget-immutable' has no carrier") || !hasErr(g, "'audit-verifiable' has no carrier") {
		t.Fatalf("uncarried invariants must error: %v", g.Errs)
	}
	if hasErr(g, "'widget-owned'") {
		t.Fatalf("a preserves-carried invariant must not error: %v", g.Errs)
	}
	if g.Counts["carried by preserves"] != 1 {
		t.Fatalf("preserves carrier not counted: %v", g.Counts)
	}
}

func TestGcPreservesUnknownInvariantIsAnError(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{
		"domain.modelith.yaml": "kind: modelith\nversion: 1\nentities:\n  Widget:\n    actions:\n      - name: publish\n        preserves: [widget-onwed]\n    invariants:\n      - id: widget-owned\n",
	})
	g := CheckCarriers(design)
	if !hasErr(g, "Widget.publish preserves unknown invariant 'widget-onwed'") {
		t.Fatalf("a preserves typo must error (it is a carrier claim binding to nothing): %v", g.Errs)
	}
}

func TestGcWaiverAnnexCarries(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{
		"formal/waivers.yaml": "waivers:\n  - invariant: widget-immutable\n    reason: DB row is append-only; enforced by the storage layer\n  - invariant: audit-verifiable\n    reason: hash chain held by the ledger store; verified by chain tests at impl\n",
	})
	g := CheckCarriers(design)
	if len(g.Errs) != 0 {
		t.Fatalf("waived invariants must not error: %v", g.Errs)
	}
	if g.Counts["waived with reason"] != 2 {
		t.Fatalf("waivers not counted: %v", g.Counts)
	}
}

func TestGcWaiverValidation(t *testing.T) {
	cases := []struct {
		name, annex, needle string
	}{
		{"unknown invariant", "waivers:\n  - invariant: ghost\n    reason: r\n", "waives unknown invariant 'ghost'"},
		{"missing reason", "waivers:\n  - invariant: widget-immutable\n", "needs both an invariant id and a reason"},
		{"waived twice", "waivers:\n  - invariant: widget-immutable\n    reason: a\n  - invariant: widget-immutable\n    reason: b\n", "waived twice"},
		{"unknown entry key", "waivers:\n  - invariant: widget-immutable\n    reason: r\n    rationale: wrong key\n", "unknown key 'rationale'"},
		{"unknown top key", "exceptions:\n  - invariant: widget-immutable\n    reason: r\n", "unknown key 'exceptions'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			design := writeCarrierDesign(t, map[string]string{"formal/waivers.yaml": c.annex})
			g := CheckCarriers(design)
			if !hasErr(g, c.needle) {
				t.Fatalf("want error containing %q, got %v", c.needle, g.Errs)
			}
		})
	}
}

func TestGcStaleWaiverWarns(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{
		"formal/waivers.yaml": "waivers:\n  - invariant: widget-owned\n    reason: no longer true; preserves carries it\n",
	})
	g := CheckCarriers(design)
	if !hasWarn(g, "waives invariant 'widget-owned', but it already has a carrier") {
		t.Fatalf("a waiver for a carried invariant must warn as stale: %v", g.Warns)
	}
}

func TestGcRelationalResidualCountsAsWaiver(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{
		"formal/policy.relational.yaml": "subjects:\n  entity: Widget\nresiduals:\n  - invariant: audit-verifiable\n    reason: cryptographic property, not an access-control rule\n",
	})
	g := CheckCarriers(design)
	if hasErr(g, "'audit-verifiable'") {
		t.Fatalf("a layer residual is a waiver-with-reason: %v", g.Errs)
	}
	if g.Counts["waived with reason"] != 1 {
		t.Fatalf("residual waiver not counted: %v", g.Counts)
	}
}

func TestGcMachineUnitCarries(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{
		"machines/Widget.matrix.md": "# Widget\n\n| name | kind | signature | contract (pre / post) | maps to | test type | fixture |\n|---|---|---|---|---|---|---|\n| isImmutable | guard | (w) -> bool | frozen / frozen | widget-immutable | unit | w |\n",
	})
	g := CheckCarriers(design)
	if hasErr(g, "'widget-immutable'") {
		t.Fatalf("a matrix maps-to cell is a checked carrier: %v", g.Errs)
	}
	if g.Counts["carried by a machine unit"] != 1 {
		t.Fatalf("machine-unit carrier not counted: %v", g.Counts)
	}
}

func TestGcCheckerClaimCarriesAndResidualWaives(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{
		"checkers/aud.checker.yaml": "checker:\n  id: aud\n  description: audit flow\nprojection:\n  include: [model, invariants]\ncoverage:\n  claim: [\"audit-*\"]\n  residuals:\n    - id: widget-immutable\n      reason: not decidable from the flow graph\nevidence:\n  projection_out: checkers/aud/projection.json\n  evidence_in: checkers/aud/evidence.json\n",
	})
	g := CheckCarriers(design)
	if hasErr(g, "'audit-verifiable'") || hasErr(g, "'widget-immutable'") {
		t.Fatalf("checker claim and residual must carry/waive: %v", g.Errs)
	}
	if g.Counts["carried by an external checker"] != 1 || g.Counts["waived with reason"] != 1 {
		t.Fatalf("checker carrier/waiver not counted: %v", g.Counts)
	}
}

func TestGcNotesAbsentLayers(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{})
	g := CheckCarriers(design)
	if !hasNote(g, "relational layers not opted in: policy, integrity, isolation") {
		t.Fatalf("absent layers are a coverage fact: %v", g.Notes)
	}
}

func TestGcNoInvariantsFails(t *testing.T) {
	design := writeCarrierDesign(t, map[string]string{
		"domain.modelith.yaml": "kind: modelith\nversion: 1\nentities:\n  Widget:\n    actions:\n      - name: publish\n",
	})
	g := CheckCarriers(design)
	if !hasErr(g, "declares no invariants") {
		t.Fatalf("an empty reconciliation is a failure, not a pass: %v", g.Errs)
	}
}

// --- G2: no_path assertions ---

// assertFixture writes a contract with boundaries alpha, beta, gamma and the
// allow chain alpha -> beta -> gamma (no direct alpha -> gamma edge).
func writeAssertFixture(t *testing.T, rules string) string {
	t.Helper()
	design := t.TempDir()
	arch := "# Architecture\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: alpha\n    code: [\"alpha/**\"]\n  - id: beta\n    code: [\"beta/**\"]\n  - id: gamma\n    code: [\"gamma/**\"]\n" +
		"dependency_rules:\n" + rules + "\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return design
}

func TestG2NoPathAssertionHolds(t *testing.T) {
	design := writeAssertFixture(t, "  allow: [\"alpha -> beta\"]\n  assert:\n    - no_path: gamma -> alpha")
	g := CheckC4(design)
	if hasErr(g, "no_path") || hasWarn(g, "no_path") {
		t.Fatalf("a held assertion must be silent: %v %v", g.Errs, g.Warns)
	}
	if g.Counts["no_path assertions"] != 1 {
		t.Fatalf("assertion not counted: %v", g.Counts)
	}
}

func TestG2NoPathAssertionFailsTransitively(t *testing.T) {
	design := writeAssertFixture(t, "  allow: [\"alpha -> beta\", \"beta -> gamma\"]\n  assert:\n    - no_path: alpha -> gamma")
	g := CheckC4(design)
	if !hasErr(g, "no_path assertion alpha -> gamma fails: the allow graph reaches it (alpha -> beta -> gamma)") {
		t.Fatalf("the closure, not the direct edges, decides an assertion; want the witness path, got %v", g.Errs)
	}
}

func TestG2NoPathAssertionBaselineDebtWarns(t *testing.T) {
	design := writeAssertFixture(t, "  allow: [\"alpha -> beta\"]\n  deny: [\"beta -> gamma\"]\n  baseline: [\"beta -> gamma\"]\n  assert:\n    - no_path: alpha -> gamma")
	g := CheckC4(design)
	if hasErr(g, "no_path") {
		t.Fatalf("a baseline-closed path is debt, not declared intent: %v", g.Errs)
	}
	if !hasWarn(g, "no_path assertion alpha -> gamma holds in allow rules but closes through baseline edges (alpha -> beta -> gamma)") {
		t.Fatalf("baseline-closed assertion paths must warn: %v", g.Warns)
	}
}

func TestG2AssertValidation(t *testing.T) {
	cases := []struct {
		name, rules, needle string
	}{
		{"unknown kind", "  allow: [\"alpha -> beta\"]\n  assert:\n    - no_cycle: alpha -> beta", "unknown assertion kind 'no_cycle'"},
		{"wildcard", "  allow: [\"alpha -> beta\"]\n  assert:\n    - no_path: \"* -> gamma\"", "contains a wildcard"},
		{"undeclared boundary", "  allow: [\"alpha -> beta\"]\n  assert:\n    - no_path: alpha -> ghost", "references undeclared boundary 'ghost'"},
		{"unparseable", "  allow: [\"alpha -> beta\"]\n  assert:\n    - no_path: \"alpha beta\"", "unparseable no_path assertion"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := CheckC4(writeAssertFixture(t, c.rules))
			if !hasErr(g, c.needle) {
				t.Fatalf("want error containing %q, got %v", c.needle, g.Errs)
			}
		})
	}
}

func TestG2UnusedMechanismNotes(t *testing.T) {
	design := writeAssertFixture(t, "  allow: [\"alpha -> beta\"]")
	g := CheckC4(design)
	for _, needle := range []string{
		"exposes: declared by no boundary",
		"baseline: no rules",
		"assert: no dependency assertions",
	} {
		if !hasNote(g, needle) {
			t.Fatalf("want unused-mechanism note %q, got %v", needle, g.Notes)
		}
	}
}

// --- Gx: waiver credit and the attested residual ---

func writeGxCarrierFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	design := t.TempDir()
	if _, ok := files["domain.modelith.yaml"]; !ok {
		files["domain.modelith.yaml"] = carrierModel
	}
	// one operational machine so Gx runs; a BUILD.md table attesting the
	// two invariants the model does not machine-map
	files["machines/Ops.machine.json"] = `{"id":"ops","_role":"operational","initial":"A","states":{"A":{}}}`
	files["BUILD.md"] = "# Build\n\nMode: full\n\n## Toolchain\n\ngo\n\n## Traceability\n\n| invariant | enforcement |\n|---|---|\n| widget-owned | service layer |\n| widget-immutable | storage layer |\n| audit-verifiable | ledger store |\n"
	for name, content := range files {
		mustWrite(t, filepath.Join(design, name), content)
	}
	return design
}

func TestGxAttestedOnlyWarnsAndDoesNotCountAsEnforced(t *testing.T) {
	design := writeGxCarrierFixture(t, map[string]string{})
	g := CheckTraceability(design)
	if !hasWarn(g, "'widget-immutable' is attested only by prose table rows") {
		t.Fatalf("prose-only attestation must warn: %v", g.Warns)
	}
	if !hasWarn(g, "'widget-owned' is carried by preserves but realized by no machine unit") {
		t.Fatalf("preserves-carried prose attestation gets its own text (a waiver would be stale): %v", g.Warns)
	}
	if g.Counts["invariants attested only (prose)"] != 3 {
		t.Fatalf("attested-only not counted: %v", g.Counts)
	}
	if g.Counts["invariants enforced"] != 0 {
		t.Fatalf("prose rows must not count as enforced: %v", g.Counts)
	}
}

func TestGxWaivedInvariantIsCreditedNotWarned(t *testing.T) {
	design := writeGxCarrierFixture(t, map[string]string{
		"formal/waivers.yaml": "waivers:\n  - invariant: audit-verifiable\n    reason: hash chain held by the ledger store\n",
	})
	g := CheckTraceability(design)
	if hasWarn(g, "'audit-verifiable'") || hasErr(g, "'audit-verifiable'") {
		t.Fatalf("a waived invariant is a recorded disposition: %v %v", g.Warns, g.Errs)
	}
	if g.Counts["invariants waived with reason"] != 1 {
		t.Fatalf("waiver not counted at Gx: %v", g.Counts)
	}
}

func TestGxCheckerClaimCountsAsEnforced(t *testing.T) {
	design := writeGxCarrierFixture(t, map[string]string{
		"checkers/aud.checker.yaml": "checker:\n  id: aud\n  description: audit flow\nprojection:\n  include: [model, invariants]\ncoverage:\n  claim: [\"audit-*\"]\nevidence:\n  projection_out: checkers/aud/projection.json\n  evidence_in: checkers/aud/evidence.json\n",
	})
	g := CheckTraceability(design)
	if hasWarn(g, "'audit-verifiable'") {
		t.Fatalf("a checker-claimed invariant is enforced (Gk reconciles the evidence): %v", g.Warns)
	}
	if g.Counts["invariants checker-claimed (external checker)"] != 1 {
		t.Fatalf("checker claim not credited: %v", g.Counts)
	}
}

func TestCarriersDuplicateDeclarations(t *testing.T) {
	// S13 compensating check: modelith lint upstream tolerates duplicate
	// attribute names; a duplicate invariant id or enum value shares the gap.
	d := t.TempDir()
	model := `kind: modelith
version: 1
enums:
  Status:
    values:
      - {name: Open, definition: open}
      - {name: Open, definition: again}
invariants:
  - id: inv-one
    statement: s
entities:
  Thing:
    definition: d
    attributes:
      - {name: content_hash, type: string}
      - {name: content_hash, type: string}
    invariants:
      - id: inv-one
        statement: duplicate
    actions:
      - name: touch
        actor: System
        description: d
        preserves: [inv-one]
`
	if err := os.WriteFile(filepath.Join(d, "domain.modelith.yaml"), []byte(model), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckCarriers(d)
	wantSubstrings := []string{
		"invariant id 'inv-one' is declared more than once",
		"attribute 'content_hash' is declared more than once",
		"enum Status: value 'Open' is declared more than once",
	}
	for _, want := range wantSubstrings {
		found := false
		for _, e := range g.Errs {
			if strings.Contains(e, want) {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing error %q in %v", want, g.Errs)
		}
	}
}
