package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// A synthetic design carrying consistency debt: a System action with no
// admission row, an unresolved USES{} member and a matrix with no machine
// (Gy-rules), and two backticked tokens outside every group (Gl-ledger).
var debtDesignFiles = map[string]string{
	"domain.modelith.yaml": "kind: DomainModel\nversion: v1\nentities:\n  Order:\n    attributes:\n" +
		"      - {name: status, type: string}\n      - {name: total, type: integer}\n" +
		"    actions:\n      - {name: pay, actor: System}\n",
	"machines/Order.machine.json": `{"id": "order", "initial": "Placed", "context": {"total": 0}, "states": {` +
		`"Placed": {"on": {"pay": {"target": "Paid", "guard": "canPay", "actions": ["recordPay"]}}}, "Paid": {"type": "final"}}}` + "\n",
	"machines/Order.matrix.md": "| name | kind | event | pre / post |\n|---|---|---|---|\n" +
		"| `canPay` | guard | - | compares `Order.total` with `ghost_token`. USES{Order.stat} |\n" +
		"| `recordPay` | action | - | stores the payment. WRITES{} |\n",
	"machines/Ledger.matrix.md": "| name | kind | pre / post |\n|---|---|---|\n| `checkLedger` | guard | - |\n",
}

func writeDebtDesign(t *testing.T) string {
	t.Helper()
	design := filepath.Join(t.TempDir(), "design")
	for rel, body := range debtDesignFiles {
		writeText(t, filepath.Join(design, filepath.FromSlash(rel)), body)
	}
	return design
}

func runCmd(t *testing.T, cmd *cobra.Command, args ...string) (string, string, int) {
	t.Helper()
	out, errB, codes := withCapturedIO(t)
	cmd.SetArgs(args)
	code := 0
	if err := executeCapturedCommand(cmd); err != nil {
		// a returned error exits 1 from main
		errB.WriteString(err.Error())
		code = 1
	}
	if len(*codes) > 0 {
		code = (*codes)[0]
	}
	return out.String(), errB.String(), code
}

func runBaseline(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	return runCmd(t, newBaselineCmd(), args...)
}

func runCheck(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	return runCmd(t, newCheckCmd(), args...)
}

// Adoption: the design is red under strict mode, `baseline --gate gy,gl`
// records its debt, and the same strict check passes with the debt counted
// as baselined notes.
func TestBaselineGyGlMakesStrictCheckPass(t *testing.T) {
	design := writeDebtDesign(t)
	out, _, code := runCheck(t, design, "--gate", "gy,gl", "--warnings-as-errors")
	if code != 1 || !strings.Contains(out, "5 blocking (ERROR/DRIFT/warning) finding(s); warnings are errors") {
		t.Fatalf("the fixture must be red before the baseline (exit %d):\n%s", code, out)
	}
	out, errOut, code := runBaseline(t, design, "--gate", "gy,gl", "--date", "2026-09-23")
	if code != 0 {
		t.Fatalf("baseline failed (exit %d): %s\n%s", code, errOut, out)
	}
	for _, want := range []string{
		"== baseline  consistency debt snapshot ==",
		"  Gy-rules: 3 finding(s) observed; 3 recorded (first recording)",
		"  Gl-ledger: 2 undeclared-fact warning(s) observed; 2 recorded (first recording)",
		"ratchet.json: 3 Gy-rules finding(s), 2 Gl-ledger undeclared-fact warning(s) baselined",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("baseline output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "boundary debt snapshot") {
		t.Fatalf("a gy,gl run must not run the G4 scan:\n%s", out)
	}
	out, _, code = runCheck(t, design, "--gate", "gy,gl", "--warnings-as-errors")
	if code != 0 || !strings.Contains(out, "0 blocking (ERROR/DRIFT/warning) finding(s); warnings are errors") {
		t.Fatalf("a fully baselined design must pass strict mode (exit %d):\n%s", code, out)
	}
	if !strings.Contains(out, ", 3 baselined\n") || !strings.Contains(out, ", 2 baselined\n") || !strings.Contains(out, "  note   baselined: ") {
		t.Fatalf("the check must note and count the baselined findings:\n%s", out)
	}
}

// Projection errors are a broken design: the baseline refuses, names them,
// and writes nothing.
func TestBaselineRefusesProjectionErrors(t *testing.T) {
	design := writeDebtDesign(t)
	matrix := filepath.Join(design, "machines", "Order.matrix.md")
	writeText(t, matrix, strings.Replace(debtDesignFiles["machines/Order.matrix.md"], "USES{Order.stat}", "USES{Order.stat} USES{Order.total}", 1))
	out, errOut, code := runBaseline(t, design, "--gate", "gy", "--date", "2026-09-23")
	if code == 0 || !strings.Contains(out+errOut, "projection error") || !strings.Contains(out+errOut, "not debt") {
		t.Fatalf("baseline must refuse projection errors (exit %d):\n%s\n%s", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(design, "ratchet.json")); !os.IsNotExist(err) {
		t.Fatalf("a refused baseline writes no ratchet: %v", err)
	}
}

func TestBaselineFlagValidation(t *testing.T) {
	design := writeDebtDesign(t)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{design}, "machinery_baseline: --impl is required"},
		{[]string{design, "--gate", "g4,gy"}, "machinery_baseline: --impl is required"},
		{[]string{design, "--gate", "gx"}, "baseline records g4, gy and gl only"},
		{[]string{design, "--gate", "gy,"}, "empty gate name"},
		{[]string{design, "--impl", ".", "--grow"}, "--grow applies to --gate gy and gl"},
	}
	for _, tc := range cases {
		_, errOut, code := runBaseline(t, tc.args...)
		if code != 1 || !strings.Contains(errOut, tc.want) {
			t.Fatalf("%v: exit %d, stderr %q, want %q", tc.args, code, errOut, tc.want)
		}
	}
}

// The G4 baseline a caller gets without --gate prints and writes exactly what
// it did before the consistency sections existed (the text below is the
// previous release's output on this fixture), and a later G4 rerun keeps the
// recorded Gy/Gl sections.
func TestBaselineG4OutputUnchangedAndKeepsConsistencySections(t *testing.T) {
	root := t.TempDir()
	design := filepath.Join(root, "design")
	impl := filepath.Join(root, "impl")
	writeText(t, filepath.Join(design, "ARCHITECTURE.md"), "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n"+
		"  - id: alpha\n    code: [\"alpha/**\"]\n  - id: beta\n    code: [\"beta/**\"]\ndependency_rules:\n  allow: []\n  deny: []\n```\n")
	writeText(t, filepath.Join(impl, "go.mod"), "module example.com/m\n")
	writeText(t, filepath.Join(impl, "alpha", "a.go"), "package alpha\n\nimport (\n\t\"example.com/m/beta\"\n\t\"example.com/m/internal/oldpkg\"\n)\n")
	writeText(t, filepath.Join(impl, "alpha", "b.go"), "package alpha\n\nimport \"example.com/m/beta\"\n")
	writeText(t, filepath.Join(impl, "beta", "b.go"), "package beta\n")
	writeText(t, filepath.Join(impl, "legacy", "old.go"), "package old\n")
	out, errOut, code := runBaseline(t, design, "--impl", impl, "--date", "2026-09-03")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	want := `== baseline  boundary debt snapshot ==
  observed: 1 cross-boundary edge(s); 1 need a baseline rule; 1 source file(s) outside every boundary; 1 import(s) map to no boundary

add to the Architecture Contract under dependency_rules (review each edge; keep intent explicit: a deny: for the same edge is legitimate and recommended when the edge should eventually die):
  baseline:
    - "alpha -> beta"   # 2026-09-03 seen in alpha/a.go and 1 more file

suggested ignore: globs for the source files outside every boundary (each glob amnesties a whole directory; review before pasting, and remember ignored code that modeled code imports still needs an external with imports: prefixes):
    - "legacy/**"

imports that map to no contract boundary (declare an external, e.g. external.rest_of_monolith, and list these under its imports: prefixes):
    - example.com/m/internal/oldpkg (1 file(s))

wrote ` + design + `/ratchet.json: 1 edge(s), 2 offender file(s)
rerunning baseline rewrites ratchet.json and may accept newly added offender files, even when no new dependency rules are proposed. Review ratchet changes before adopting them as accepted debt.
armed: G4 now fails when a baselined edge gains a new offender file, and the machinery plugin blocks import findings at turn end
`
	if out != want {
		t.Fatalf("G4 baseline output changed:\n%s\nwant\n%s", out, want)
	}
	wantRatchet := "{\n  \"date\": \"2026-09-03\",\n  \"edges\": {\n    \"alpha -\\u003e beta\": [\n      \"alpha/a.go\",\n      \"alpha/b.go\"\n    ]\n  }\n}\n"
	data, err := os.ReadFile(filepath.Join(design, "ratchet.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != wantRatchet {
		t.Fatalf("G4 ratchet bytes changed:\n%s\nwant\n%s", data, wantRatchet)
	}

	// record consistency debt beside the edges; the date stays the G4 snapshot's
	for rel, body := range debtDesignFiles {
		writeText(t, filepath.Join(design, filepath.FromSlash(rel)), body)
	}
	if _, errOut, code := runBaseline(t, design, "--gate", "gy,gl", "--date", "2026-09-30"); code != 0 {
		t.Fatalf("gy,gl baseline failed: %s", errOut)
	}
	data, err = os.ReadFile(filepath.Join(design, "ratchet.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), strings.TrimSuffix(wantRatchet, "\n}\n")+",\n  \"rule_findings\": [") || !strings.Contains(string(data), "\"undeclared_facts\": [") {
		t.Fatalf("a gy,gl run must keep the edges and the date and add its sections:\n%s", data)
	}
	if _, errOut, code := runBaseline(t, design, "--impl", impl); code != 0 {
		t.Fatalf("g4 rerun failed: %s", errOut)
	}
	again, err := os.ReadFile(filepath.Join(design, "ratchet.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(data) {
		t.Fatalf("a g4 rerun must keep the recorded Gy/Gl sections:\n%s\nwas\n%s", again, data)
	}
}
