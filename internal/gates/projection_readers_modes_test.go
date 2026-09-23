package gates

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/checker"
)

const factsModel = `kind: DomainModel
version: v1
title: Facts
description: "A model whose prose carries a tab (	) and a newline\nso a leak would split a fact line: PROSE-MARKER-TOP."
enums:
  OrderStatus:
    values:
      - {name: Placed, definition: "PROSE-MARKER-ENUM\twith a tab."}
      - {name: Paid, definition: "Paid."}
entities:
  Order:
    definition: >
      PROSE-MARKER-DEFINITION spans
      two lines.
    attributes:
      - {name: status, type: OrderStatus, description: "PROSE-MARKER-ATTR\tand a tab"}
      - {name: total, type: integer}
    invariants:
      - {id: order-paid-final, statement: "PROSE-MARKER-STATEMENT\tPaid is final."}
    actions:
      - {name: pay, actor: System, description: "PROSE-MARKER-ACTION"}
`

const factsMachine = `{
  "id": "order",
  "initial": "Placed",
  "context": {"total": 0},
  "states": {
    "Placed": {"on": {"pay": {"target": "Paid", "guard": "canPay", "actions": ["recordPay"]}}},
    "Paid": {"type": "final"}
  }
}
`

// writeFactsDesign writes a minimal design under root and returns its path.
func writeFactsDesign(t *testing.T, root string, files map[string]string) string {
	t.Helper()
	base := map[string]string{"domain.modelith.yaml": factsModel}
	for rel, body := range files {
		base[rel] = body
	}
	for rel, body := range base {
		path := filepath.Join(root, filepath.FromSlash(rel))
		must(t, os.MkdirAll(filepath.Dir(path), 0o755))
		must(t, os.WriteFile(path, []byte(body), 0o644))
	}
	return root
}

func factRows(facts *checker.DesignFacts, relation string) []string {
	var out []string
	for _, row := range facts.Rows(relation) {
		out = append(out, strings.Join(row.Values, "|"))
	}
	return out
}

func TestFactsDesignWithoutMachinesHasNoMachineLayers(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), nil)
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	for _, layer := range []string{"machines", "matrices", "oracles", "events", "c4", "authorization", "milestones", "supersession"} {
		if facts.HasLayer(layer) {
			t.Fatalf("layer %s must be absent, not empty-and-claimed", layer)
		}
	}
	files := mustFactsFiles(t, facts)
	for _, absent := range []string{"machine.facts", "state.facts", "unit.facts", "oracle_row.facts"} {
		if _, ok := files[absent]; ok {
			t.Fatalf("%s must not be written for a design with no machines", absent)
		}
	}
	if _, ok := files["entity.facts"]; !ok {
		t.Fatal("the model layer is always present")
	}
	if strings.Contains(string(files[checker.RelationsIndexFile]), "machine\t") {
		t.Fatalf("relations.txt must not list an absent layer's relations:\n%s", files[checker.RelationsIndexFile])
	}
	model, err := checker.LoadModel(filepath.Join(design, "domain.modelith.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	man := &checker.Manifest{}
	man.Checker.ID = "absent"
	man.Projection.Include = []string{"model", "machines"}
	if _, err := checker.GenerateWithFacts(model, facts, man, "sha256:"+strings.Repeat("0", 64), "v"); err == nil || !strings.Contains(err.Error(), `layer "machines" is absent`) {
		t.Fatalf("a manifest asking for an absent layer must fail loudly, got %v", err)
	}
}

func TestFactsProjectMachineAndOrphanMatrix(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md": "| name | kind | signature | pre / post |\n|---|---|---|---|\n" +
			"| `canPay` | guard | `(ctx) -> bool` | true iff paid. CLAUSES{amount-covered} USES{Order.total} |\n" +
			"| `persistOrder` | actor | `(id) -> ok \\| err` | writes the row. WRITES{Order.status} CARRIES{column:Order.status} derived: pay_window (computed, never stored) |\n",
		"machines/Ledger.matrix.md": "| name | kind | signature | pre / post |\n|---|---|---|---|\n" +
			"| `postEntry` | action | `(ctx) -> ctx` | VALUES entryKind{debit, credit} |\n",
	})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string][]string{
		"machine":      {"Order"},
		"state":        {"Order.Paid|Order|final", "Order.Placed|Order|atomic"},
		"context_key":  {"Order|total"},
		"unit":         {"Ledger.postEntry||postEntry|action", "Order.canPay|Order|canPay|guard", "Order.persistOrder|Order|persistOrder|actor"},
		"unit_clauses": {"Order.canPay|amount-covered|active"},
		"unit_values":  {"Ledger.postEntry|entryKind|credit", "Ledger.postEntry|entryKind|debit"},
		"unit_derived": {"Order.persistOrder|pay_window"},
		"unit_writes":  {"Order.persistOrder|Order.status"},
		"unit_uses":    {"Order.canPay|Order.total"},
		"unit_carries": {"Order.persistOrder|column|Order.status"},
		"action_on":    {"Order.ORDE-" + oracleSuffix(t, facts) + "|recordPay"},
	}
	for relation, want := range checks {
		if got := factRows(facts, relation); strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("%s = %v, want %v", relation, got, want)
		}
	}
	transitions := facts.Rows("transition")
	if len(transitions) != 1 {
		t.Fatalf("one transition expected, got %v", factRows(facts, "transition"))
	}
	tr := transitions[0]
	if tr.Values[1] != "Order.Placed" || tr.Values[2] != "on:pay" || tr.Values[3] != "Order.Paid" || !strings.HasPrefix(tr.StableID, "tr:Order.ORDE-") {
		t.Fatalf("transition must carry its oracle stable id and resolved states: %+v", tr)
	}
	if tr.Source.Path != "machines/Order.machine.json" || tr.Source.Line != 6 {
		t.Fatalf("transition source must be the line of its event key: %+v", tr.Source)
	}
	if got := factRows(facts, "guard_on"); len(got) != 1 || !strings.HasSuffix(got[0], "|canPay") {
		t.Fatalf("guard_on = %v", got)
	}
	unit := facts.Rows("unit")[0]
	if unit.StableID != "unit:Ledger.postEntry" || unit.Source.Path != "machines/Ledger.matrix.md" || unit.Source.Line != 3 {
		t.Fatalf("orphan matrix unit keeps its matrix id and source: %+v", unit)
	}
}

// Stage 1's unknown-group rule rejects RETIRED{...} on a matrix row, so the
// retired half of unit_clauses is pinned on the row projector directly.
func TestFactsClauseDeclarationProjectsActiveAndRetired(t *testing.T) {
	b := &factBuilder{design: t.TempDir(), facts: checker.NewDesignFacts()}
	header := []string{"name", "kind", "signature", "pre / post"}
	row := tableRow{line: 3, header: header,
		cells: []string{"`canPay`", "guard", "`(ctx) -> bool`", "CLAUSES{amount-covered, fresh} RETIRED{legacy-check}"}}
	b.matrixRowFacts(matrixRow{rel: "machines/Order.matrix.md", matrix: "Order", row: row,
		subjects: []string{"Order.canPay"}, unitNames: []string{"canPay"}, stableIDs: []string{"unit:Order.canPay"}})
	if len(b.errs) != 0 {
		t.Fatal(b.errs)
	}
	want := "Order.canPay|amount-covered|active Order.canPay|fresh|active Order.canPay|legacy-check|retired"
	if got := strings.Join(factRows(b.facts, "unit_clauses"), " "); got != want {
		t.Fatalf("unit_clauses = %s, want %s", got, want)
	}
	b.matrixRowFacts(matrixRow{rel: "machines/Order.matrix.md", matrix: "Order",
		row:      tableRow{line: 4, header: header, cells: []string{"`x`", "guard", "-", "CLAUSES{a} and CLAUSES{"}},
		subjects: []string{"Order.x"}, unitNames: []string{"x"}, stableIDs: []string{"unit:Order.x"}})
	if len(b.errs) != 1 || !strings.Contains(b.errs[0], "malformed CLAUSES") {
		t.Fatalf("an unterminated CLAUSES group must be refused: %v", b.errs)
	}
}

// oracleSuffix returns the hash part of the one transition's stable id.
func oracleSuffix(t *testing.T, facts *checker.DesignFacts) string {
	t.Helper()
	rows := facts.Rows("transition")
	if len(rows) != 1 {
		t.Fatalf("one transition expected, got %d", len(rows))
	}
	return strings.TrimPrefix(rows[0].Values[0], "Order.ORDE-")
}

func TestFactsDuplicateStableIDNamesBothSources(t *testing.T) {
	oracle := "| test id | stable id | source | trigger | guard | target | actions |\n|---|---|---|---|---|---|---|\n" +
		"| T-DUP-01 | DUP-abcdef | A | on:x | - | B | - |\n"
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Alpha.oracle.md": oracle,
		"machines/Beta.oracle.md":  "\n" + oracle,
	})
	_, err := LoadDesignFacts(design)
	if err == nil {
		t.Fatal("a stable id defined in two files must fail")
	}
	for _, want := range []string{`"orc:DUP-abcdef"`, "machines/Alpha.oracle.md:3", "machines/Beta.oracle.md:4"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("duplicate error must name %s: %v", want, err)
		}
	}
}

const factsArchitecture = "# Architecture\n\n## Event-contract table\n\nSource: the emit call sites.\n\n" +
	"| event | producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|---|\n" +
	"| `paid` | `orders` (via outbox) | `billing` | `Order.id`, `Order.total` | at-least-once | none | message id |\n" +
	"| `nudged` | `orders` | `billing` |  | at-least-once | none | message id |\n" +
	"| `prosed` | `orders` | `billing` | the order id and some prose | at-least-once | none | message id |\n"

func TestFactsEventRowWithEmptyPayloadProjectsNoFields(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{"ARCHITECTURE.md": factsArchitecture})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(factRows(facts, "event"), " "); got != "nudged|orders paid|orders prosed|orders" {
		t.Fatalf("every event row projects, empty payload or not: %s", got)
	}
	if got := strings.Join(factRows(facts, "event_payload_field"), " "); got != "paid|Order.id paid|Order.total" {
		t.Fatalf("an empty or prose payload cell yields no fields: %s", got)
	}
	if got := strings.Join(factRows(facts, "event_consumer"), " "); got != "nudged|billing paid|billing prosed|billing" {
		t.Fatalf("consumers: %s", got)
	}
	if !facts.HasLayer("supersession") || facts.HasLayer("c4") {
		t.Fatal("an ARCHITECTURE.md with no contract has the supersession layer and no c4 layer")
	}
}

func TestFactsMilestoneCitingUnknownOracleIDIsProjectedAsStated(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"BUILD.md": "# Build\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: ZZZZ-abcdef green and T-NOPE-01 green.\nStatus: open\n\n" +
			"**M1 - Rest.** DoD: ORACLESET{machines/Order.oracle.md} green.\nStatus: closed\n",
		"machines/Order.oracle.md": "| test id | stable id | source | trigger | guard | target | actions |\n|---|---|---|---|---|---|---|\n" +
			"| T-ORDE-01 | ORDE-123456 | Placed | on:pay | canPay | Paid | recordPay |\n",
	})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(factRows(facts, "milestone"), " "); got != "M0|open M1|closed" {
		t.Fatalf("milestones: %s", got)
	}
	if got := strings.Join(factRows(facts, "dod_id"), " "); got != "M0|T-NOPE-01 M0|ZZZZ-abcdef M1|ORDE-123456" {
		t.Fatalf("dod ids must be projected as stated and ORACLESET expanded: %s", got)
	}
	ms := facts.Rows("milestone")[0]
	if ms.Source.Path != "BUILD.md" || ms.Source.Line != 5 {
		t.Fatalf("milestone source must be its marker line: %+v", ms.Source)
	}
}

func TestFactsDesignPathWithSpaces(t *testing.T) {
	root := filepath.Join(t.TempDir(), "my design dir", "design with spaces")
	design := writeFactsDesign(t, root, map[string]string{"machines/Order.machine.json": factsMachine})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	rows := facts.Rows("state")
	if len(rows) != 2 {
		t.Fatalf("both states must project: %v", factRows(facts, "state"))
	}
	for _, row := range rows {
		if row.Source.Path != "machines/Order.machine.json" {
			t.Fatalf("source paths are design-relative whatever the design path is: %+v", row.Source)
		}
	}
}

func TestFactsNeverProjectProse(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md": "| name | kind | signature | pre / post |\n|---|---|---|---|\n" +
			"| `persistOrder` | actor | `(id) -> ok` | PROSE-MARKER-CONTRACT writes. derived: pay_window (PROSE-MARKER-REASON) |\n",
		"AUTHORIZATION.md": "<!-- machinery:authorization-inventory -->\n\n| authorization subject | admission |\n|---|---|\n" +
			"| `Order.pay` | (no authorization: PROSE-MARKER-WAIVER) |\n",
	})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	blob := string(factsBytes(t, facts))
	if strings.Contains(blob, "PROSE-MARKER") {
		t.Fatalf("free prose leaked into the facts:\n%s", blob)
	}
	for name, body := range mustFactsFiles(t, facts) {
		for i, line := range strings.Split(strings.TrimSuffix(string(body), "\n"), "\n") {
			if strings.ContainsRune(line, '\r') {
				t.Fatalf("%s:%d carries a carriage return", name, i+1)
			}
		}
	}
	if got := factRows(facts, "no_authorization"); strings.Join(got, " ") != "Order.pay" {
		t.Fatalf("the waiver projects its subject only: %v", got)
	}
	if got := factRows(facts, "unit_derived"); strings.Join(got, " ") != "Order.persistOrder|pay_window" {
		t.Fatalf("the derived waiver projects its fact only: %v", got)
	}
	if got := factRows(facts, "action"); strings.Join(got, " ") != "Order.pay|Order|pay|System" {
		t.Fatalf("an action projects its identifiers only: %v", got)
	}
}

func mustFactsFiles(t *testing.T, facts *checker.DesignFacts) map[string][]byte {
	t.Helper()
	files, err := facts.FactsFiles()
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestFactsSourcePathsUseForwardSlashes(t *testing.T) {
	if got := checker.PortableSourcePath(`machines\Order.matrix.md`); got != "machines/Order.matrix.md" {
		t.Fatalf("a Windows separator must render as a forward slash: %q", got)
	}
	facts := checker.NewDesignFacts()
	if err := facts.Add("entity", "entity:X", checker.Source{Path: `sub\x.yaml`, Line: 1}, "X"); err == nil {
		t.Fatal("a source path with a backslash must be refused")
	}
	b := &factBuilder{design: t.TempDir(), facts: checker.NewDesignFacts()}
	b.add(`machines\Order.matrix.md`, 2, "entity", "entity:Y", "Y")
	if len(b.errs) != 0 {
		t.Fatal(b.errs)
	}
	if got := b.facts.Rows("entity")[0].Source.Path; got != "machines/Order.matrix.md" {
		t.Fatalf("the reader normalizes separators: %q", got)
	}
	for _, rel := range bundledDesigns(t) {
		facts, err := LoadDesignFacts(filepath.Join("..", "..", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range checker.RelationCatalog() {
			for _, row := range facts.Rows(spec.Name) {
				if strings.Contains(row.Source.Path, `\`) || strings.HasPrefix(row.Source.Path, "/") {
					t.Fatalf("%s %s: source path %q", rel, spec.Name, row.Source.Path)
				}
			}
		}
	}
}

func TestFactsSecondRunIsByteIdentical(t *testing.T) {
	for _, rel := range bundledDesigns(t) {
		design := filepath.Join("..", "..", filepath.FromSlash(rel))
		first, err := LoadDesignFacts(design)
		if err != nil {
			t.Fatal(err)
		}
		second, err := LoadDesignFacts(design)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(factsBytes(t, first), factsBytes(t, second)) {
			t.Fatalf("%s: two reads of an unchanged design differ", rel)
		}
	}
}

// unit_declares marks every declaration group on a unit row, so an empty
// WRITES{} (declared read-only) stays distinguishable from no WRITES group:
// the first yields a marker and no unit_writes row, the second neither.
func TestFactsUnitDeclaresMarksEveryGroupIncludingEmptyWrites(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md": "| name | kind | event | pre / post |\n|---|---|---|---|\n" +
			"| `readOnly` | action | - | WRITES{} USES{Order.total} |\n" +
			"| `silent` | action | - | reads the order and says nothing structured |\n" +
			"| `persist` | actor | `order.paid` | WRITES{Order.status} CARRIES{column:Order.status} CLAUSES{a} READS{Order.total} VALUES kind{x, y} payload {Order.status} |\n",
	})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Order.persist|CARRIES", "Order.persist|CLAUSES", "Order.persist|READS",
		"Order.persist|VALUES", "Order.persist|WRITES", "Order.persist|payload",
		"Order.readOnly|USES", "Order.readOnly|WRITES",
	}
	if got := factRows(facts, "unit_declares"); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unit_declares = %v, want %v", got, want)
	}
	if got := factRows(facts, "unit_writes"); strings.Join(got, " ") != "Order.persist|Order.status" {
		t.Fatalf("WRITES{} must add no unit_writes row: %v", got)
	}
	for _, row := range facts.Rows("unit_declares") {
		if row.Source.Path != "machines/Order.matrix.md" || !strings.HasPrefix(row.StableID, "unit:Order.") {
			t.Fatalf("unit_declares keeps the unit's stable id and row source: %+v", row)
		}
	}
}

// PRODUCES{} projects one unit_produces row per member and a PRODUCES marker.
// A row may carry PRODUCES and WRITES together (a consumer arm that writes
// its own row and performs another entity's action): both project, neither
// shadows the other. A consumed-event row names no unit, so its PRODUCES
// attaches to the matrix id, like every declaration on such a row.
func TestFactsProducesProjectsBesideWrites(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md": "| name | kind | event | pre / post |\n|---|---|---|---|\n" +
			"| `settle` | actor | - | WRITES{Order.status} PRODUCES{Order.pay} CARRIES{column:Order.status} |\n\n" +
			"| consumed event | reaction |\n|---|---|\n" +
			"| `payment.captured` | PRODUCES{Order.pay} |\n",
	})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string][]string{
		"unit_produces": {"Order|Order.pay", "Order.settle|Order.pay"},
		"unit_writes":   {"Order.settle|Order.status"},
		"unit_declares": {"Order|PRODUCES", "Order.settle|CARRIES", "Order.settle|PRODUCES", "Order.settle|WRITES"},
	}
	for relation, want := range checks {
		if got := factRows(facts, relation); strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("%s = %v, want %v", relation, got, want)
		}
	}
}
