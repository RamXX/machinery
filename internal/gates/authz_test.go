package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const authzModel = `kind: DomainModel
version: v1
enums:
  OrderState:
    values: [{name: Placed}, {name: Paid}]
entities:
  Order:
    attributes: [{name: state, type: OrderState}]
    actions:
      - {name: markPaid, actor: System}
    invariants:
      - {id: order-paid-final, statement: a paid order stays paid}
`

func authzFixture(t *testing.T, inventory, extraMatrix string) *Gate {
	t.Helper()
	design := t.TempDir()
	for _, dir := range []string{"machines", "formal"} {
		if err := os.MkdirAll(filepath.Join(design, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), authzModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"),
		"machine: Order\npattern: control-flow-only\nreason: authorization fixture\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | maps to |\n|---|---|---|\n| `saveOrder` | actor | `order-paid-final` |\n"+extraMatrix)
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n| `Order` | in-memory |\n\n" + inventory
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	mustWrite(t, filepath.Join(design, "workspace.dsl"), "workspace \"Orders\" {\n  model {\n    orders = container \"Orders\" \"Owner\" \"Go\"\n  }\n}\n")
	return CheckTraceability(design)
}

const authzHeader = `<!-- machinery:authorization-inventory -->

| authorization subject | admission |
|---|---|
`

func TestAuthorizationInventoryMissingSystemActionErrors(t *testing.T) {
	g := authzFixture(t, "", "")
	if !hasErr(g, "System action 'Order.markPaid' has no authorization row") {
		t.Fatalf("a System write without an inventory row must fail: %v", g.Errs)
	}
}

func TestAuthorizationInventoryAdmitsSystemAction(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", "")
	if hasErr(g, "authorization") {
		t.Fatalf("an admitted System action must pass: %v", g.Errs)
	}
	if g.Counts["authorization obligations admitted"] != 1 {
		t.Fatalf("the admission must be counted: %+v", g.Counts)
	}
}

func TestAuthorizationInventoryWaiverRequiresReason(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | (no authorization: immutable internal replay) |\n", "")
	if hasErr(g, "authorization") {
		t.Fatalf("a reasoned waiver must pass: %v", g.Errs)
	}
	g = authzFixture(t, authzHeader+"| `Order.markPaid` | (no authorization: ) |\n", "")
	if !hasErr(g, "waiver names no reason") {
		t.Fatalf("an empty waiver must fail: %v", g.Errs)
	}
}

func TestAuthorizationInventoryProducerArmCreatesObligation(t *testing.T) {
	matrix := "\n| cascade | producer | outcome |\n|---|---|---|\n| persist | `Order.reindex` | emitted |\n"
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", matrix)
	if !hasErr(g, "matrix producer 'Order.reindex' has no authorization row") {
		t.Fatalf("a named cascade producer must be admitted independently: %v", g.Errs)
	}
	g = authzFixture(t, authzHeader+
		"| `Order.markPaid` | `orders.write` |\n| `Order.reindex` | `orders.reindex` |\n", matrix)
	if hasErr(g, "authorization") {
		t.Fatalf("both obligations are admitted: %v", g.Errs)
	}
}

func TestAuthorizationInventoryRejectsDuplicateAndOrphanRows(t *testing.T) {
	g := authzFixture(t, authzHeader+
		"| `Order.markPaid` | `orders.write` |\n| `Order.markPaid` | `orders.write` |\n| `Order.ghost` | `orders.ghost` |\n", "")
	joined := strings.Join(g.Errs, "\n")
	for _, want := range []string{"duplicate authorization row", "names no System action or matrix producer"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("inventory closure must report %q: %v", want, g.Errs)
		}
	}
}

func TestAuthorizationInventoryIgnoresCombinedProducerConsumerProse(t *testing.T) {
	matrix := "\n| cascade | producer / consumer | outcome |\n|---|---|---|\n| persist | `Order.reindex` and `Order.repair` | emitted |\n"
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", matrix)
	if hasErr(g, "matrix producer") {
		t.Fatalf("combined prose column must not arm producer obligations: %v", g.Errs)
	}
}

func TestAuthorizationInventoryUsesOneSubjectPerProducerCell(t *testing.T) {
	matrix := "\n| cascade | producer | outcome |\n|---|---|---|\n| persist | `Order.reindex` (via `Order.repair`) | emitted |\n"
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n| `Order.reindex` | `orders.reindex` |\n", matrix)
	if hasErr(g, "matrix producer 'Order.repair'") || hasErr(g, "matrix producer 'Order.reindex'") {
		t.Fatalf("one producer cell names one cleaned subject: %v", g.Errs)
	}
}

func TestAuthorizationInventoryRejectsTwoProducerSubjectsInOneCell(t *testing.T) {
	matrix := "\n| cascade | producer | outcome |\n|---|---|---|\n| persist | `Order.reindex` and `Order.repair` | emitted |\n"
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", matrix)
	if !hasErr(g, "producer cell must name one subject") {
		t.Fatalf("two subjects must not turn into a phantom obligation: %v", g.Errs)
	}
}

func TestAuthorizationInventoryMalformedMarkersAndAdmissions(t *testing.T) {
	cases := []struct{ name, inventory, want string }{
		{"no marker", "", "no <!-- machinery:authorization-inventory --> marker"},
		{"two markers", authzHeader + "| `Order.markPaid` | cap |\n<!-- machinery:authorization-inventory -->\n", "marker appears 2 times"},
		{"no table", "<!-- machinery:authorization-inventory -->\n", "marker has no table"},
		{"empty admission", authzHeader + "| `Order.markPaid` | |\n", "admission is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := authzFixture(t, tc.inventory, "")
			if !hasErr(g, tc.want) {
				t.Fatalf("want %q: %v", tc.want, g.Errs)
			}
		})
	}
}

func TestAuthorizationInventoryFindingNamesSourcePath(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.ghost` | stale |\n", "")
	if !hasErr(g, "ARCHITECTURE.md:") {
		t.Fatalf("row finding must name its source artifact: %v", g.Errs)
	}
}

func TestAuthorizationInventoryRejectsProseAdmission(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | TODO |\n", "")
	if !hasErr(g, "admission must name one capability") {
		t.Fatalf("free text cannot prove an admitting capability: %v", g.Errs)
	}
}

func TestAuthorizationH2MachineWrittenInventory(t *testing.T) {
	// H2 keeps both forms in Principal.matrix.md, without a marker document.
	matrix := "\n| resource | platform_admin | tenant_admin | every other preset |\n|---|---|---|---|\n" +
		"| `Order` (residual `order-paid-final`, MACHINE-WRITTEN-BY{markPaid: settlement_consumer}) | `read` | `read` | `read` |\n" +
		"\n| resource | machine-written actions | what the verb columns still decide |\n|---|---|---|\n" +
		"| `Order` | MACHINE-WRITTEN{advance} | markPaid is producer narrowed |\n"
	g := authzFixture(t, "", matrix)
	if hasErr(g, "no <!-- machinery:authorization-inventory -->") || hasErr(g, "System action 'Order.markPaid'") {
		t.Fatalf("H2 producer mark must be an inventory admission: %v", g.Errs)
	}
	if g.Counts["authorization obligations admitted"] == 0 {
		t.Fatalf("H2 admission must be counted: %+v", g.Counts)
	}
}

func TestAuthorizationH2MachineWrittenListIsResourceScoped(t *testing.T) {
	matrix := "\n| resource | machine-written actions | what the verb columns still decide |\n|---|---|---|\n" +
		"| `Other` | MACHINE-WRITTEN{markPaid} | none |\n"
	g := authzFixture(t, "", matrix)
	if !hasErr(g, "System action 'Order.markPaid' has no authorization row") {
		t.Fatalf("another resource's list cannot admit this action: %v", g.Errs)
	}
}

func TestAuthorizationH2ResidualSeatGrant(t *testing.T) {
	matrix := "\n| resource | platform_admin | tenant_admin | every other preset |\n|---|---|---|---|\n" +
		"| `Order` (residual `order-paid-final`) | `read` | `update`, WITHIN ITS OWN `Tenant` | `read` |\n"
	g := authzFixture(t, "", matrix)
	if !hasErr(g, "no <!-- machinery:authorization-inventory -->") {
		t.Fatalf("a seat grant cannot authorize a System dispatch: %v", g.Errs)
	}
}

func TestAuthorizationAdmissionResolvesC4Owner(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `fictional.capability` |\n", "")
	if !hasErr(g, "admission owner 'fictional' is not a C4 element") {
		t.Fatalf("a dotted token without a declared owner cannot admit a write: %v", g.Errs)
	}
	g = authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", "")
	if hasErr(g, "admission owner") {
		t.Fatalf("declared C4 owner must pass: %v", g.Errs)
	}
}

func TestAuthorizationH2RejectsConflictingAndMalformedProducerMarks(t *testing.T) {
	cases := []struct{ mark, list, want string }{
		{"MACHINE-WRITTEN-BY{markPaid: settlement_consumer, other: wrong}", "MACHINE-WRITTEN{advance}", "one action followed by producers"},
		{"MACHINE-WRITTEN-BY{markPaid: settlement_consumer}", "MACHINE-WRITTEN{markPaid, advance}", "both name-admitted and producer-narrowed"},
		{"MACHINE-WRITTEN-BY{markPaid: }", "MACHINE-WRITTEN{advance}", "invalid MACHINE-WRITTEN-BY"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			matrix := "\n| resource | platform_admin | tenant_admin | every other preset |\n|---|---|---|---|\n" +
				"| `Order` (residual `order-paid-final`, " + tc.mark + ") | `read` | `read` | `read` |\n" +
				"\n| resource | machine-written actions | what the verb columns still decide |\n|---|---|---|\n" +
				"| `Order` | " + tc.list + " | prose |\n"
			g := authzFixture(t, "", matrix)
			if !hasErr(g, tc.want) {
				t.Fatalf("invalid H2 inventory must fail: %v", g.Errs)
			}
		})
	}
}

type h2Action struct {
	name, description string
}

func h2ResidualFixture(t *testing.T, resource string, actions []h2Action, residualRow, actionMatrix string, machineWrittenList ...string) *Gate {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	var model strings.Builder
	fmt.Fprintf(&model, "kind: DomainModel\nversion: v1\nentities:\n  %s:\n    attributes: [{name: state, type: string}]\n    actions:\n", resource)
	for _, action := range actions {
		fmt.Fprintf(&model, "      - {name: %s, actor: System, description: %q}\n", action.name, action.description)
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model.String())
	mustWrite(t, filepath.Join(design, "machines", resource+".machine.json"),
		fmt.Sprintf(`{"id":%q,"initial":"Open","states":{"Open":{"type":"final"}}}`, resource))
	principal := "| resource | platform_admin | tenant_admin | every other preset |\n|---|---|---|---|\n" + residualRow + "\n"
	if len(machineWrittenList) > 0 {
		principal += "\n| resource | machine-written actions | what the verb columns still decide |\n|---|---|---|\n" +
			"| `" + resource + "` | " + machineWrittenList[0] + " | the verb columns decide the other actions |\n"
	}
	mustWrite(t, filepath.Join(design, "machines", "Principal.matrix.md"), principal)
	if actionMatrix != "" {
		mustWrite(t, filepath.Join(design, "machines", resource+".matrix.md"), actionMatrix)
	}
	return CheckTraceability(design)
}

func TestAuthorizationH2ResidualWithheldWrites(t *testing.T) {
	cases := []struct {
		resource, row, matrix string
		actions               []h2Action
		admitted              int
	}{
		{
			resource: "Verdict",
			row:      "| `Verdict` (residual `unknown-never-compliant`) | `read` | `read`, WITHIN ITS OWN `Tenant` | `read`. NO preset holds `create`, `update` or `delete`: `evaluate` and `replay` are `System` and a verdict is insert-only |",
			matrix:   "## Model actions\n\n- `evaluate` (actor System): the kernel records the outcome. Insert-only.\n- `replay` (actor System): re-run a recorded verdict and recompute hashes; any divergence is an integrity incident.\n\n| name | kind | contract (pre / post) |\n|---|---|---|\n| `replayAndCompareHashes` | actor | re-evaluates pinned inputs and compares hashes; the stored row is never adjusted to match replay |\n",
			actions: []h2Action{
				{"evaluate", "Run NIL over a fact bundle and an approved release; record the decision with its full hash set."},
				{"replay", "Re-execute and verify byte-identical results against recorded hashes; divergence is an integrity incident."},
			},
			admitted: 1,
		},
		{
			resource: "Trial",
			row:      "| `Trial` (residual `trial-provision-requires-release`) | read. `provision`, `notify_expiry` and `erase` are System producers through the machine, never a preset act, and no preset creates, updates or deletes a trial. | read, WITHIN ITS OWN `Tenant` | read, the prospect's clock within the trial tenant. |",
			matrix:   "| action | reason |\n|---|---|\n| `provision` | The creation action. A released submission mints the row. |\n",
			actions: []h2Action{
				{"provision", "Create the trial tenant with the funnel's capability set and caps. Postcondition: status provisioned."},
				{"notify_expiry", "Enter the notice window before expiry. Postcondition: status expiring."},
				{"erase", "Close the window and drive the trial tenant's erasure cascade. Postcondition: status erased."},
			},
			admitted: 3,
		},
		{
			resource: "TenantRoleRevision",
			row:      "| `TenantRoleRevision` | read | read, WITHIN ITS OWN `Tenant`. Nothing writes it but `appendRoleRevision`, in the transaction of the change it records | denied |",
			actions:  []h2Action{{"record", "Append the revision record in the SAME TRANSACTION as the role change that caused it."}},
			admitted: 1,
		},
		{
			resource: "ReviewTask",
			row:      "| `ReviewTask` (residual `rbac-reviewer-approval`) | `read` and `update` | `read` and `update`, WITHIN ITS OWN `Tenant` | `read` and `update`. NEVER `create`: `enqueue` is `System`, the producing assembly's guarded branch. NEVER `delete`: a resolution is append-only |",
			actions:  []h2Action{{"enqueue", "Create with exactly one typed subject and route to the scope-appropriate queue. Postcondition: status open."}},
			admitted: 1,
		},
		{
			resource: "ConversionReport",
			row:      "| `ConversionReport` (residual `conversion-report-first-class`) | `read` | `read` and `update`, WITHIN ITS OWN `Tenant` | `read` and `update`: human accept and reject; read alone for every other tenant preset. NEVER `create`: `produce` is the run's own act through the machine |",
			actions:  []h2Action{{"produce", "In the run's page-accounting stage, write the report with every per-page count. Postcondition: status pending."}},
			admitted: 1,
		},
		{
			resource: "ExtractionRun",
			row:      "| `ExtractionRun` (residual `run-single-live`) | `create` and `read`: `enqueue` is a platform act; no `update`, no `delete`, every stage being System's arm | `create` and `read`, WITHIN ITS OWN `Tenant`: `enqueue`; no `update`, no `delete` | `create` and `read`: `enqueue` for full_except_account_management; read alone for every other tenant preset |",
			actions: []h2Action{
				{"start", "Begin extraction over pinned input versions with a recorded configuration. Precondition: no other live run."},
				{"complete", "Finish with artifacts and recorded quality metrics."},
			},
			admitted: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.resource, func(t *testing.T) {
			g := h2ResidualFixture(t, tc.resource, tc.actions, tc.row, tc.matrix)
			for _, action := range tc.actions {
				if hasErr(g, "System action '"+tc.resource+"."+action.name+"' has no authorization row") {
					t.Fatalf("H2 residual row must settle %s.%s: %v", tc.resource, action.name, g.Errs)
				}
			}
			if hasErr(g, "no <!-- machinery:authorization-inventory -->") {
				t.Fatalf("H2 residual table is an authorization source: %v", g.Errs)
			}
			if got := g.Counts["authorization obligations admitted"]; got != tc.admitted {
				t.Fatalf("admitted %d, want %d: %v", got, tc.admitted, g.Errs)
			}
		})
	}
}

func TestAuthorizationH2ReadOnlyActionUsesDeclaration(t *testing.T) {
	row := "| `Verdict` (residual `unknown-never-compliant`) | `read` | `read` | `read`. NO preset holds `create`, `update` or `delete` |"
	matrix := "## Model actions\n\n- `verify_again` (actor System): re-run a recorded verdict and recompute hashes; any divergence is an integrity incident.\n\n| name | kind | contract (pre / post) |\n|---|---|---|\n| `verifyAndCompareHashes` | actor | compares the pinned hashes; the stored row is never adjusted to match verification |\n"
	g := h2ResidualFixture(t, "Verdict", []h2Action{{"verify_again", "Re-execute and verify byte-identical results against recorded hashes; divergence is an integrity incident."}}, row, matrix)
	if hasErr(g, "System action 'Verdict.verify_again'") || g.Counts["authorization obligations admitted"] != 0 {
		t.Fatalf("a declared verification with no Verdict write owes no write admission: %v, %+v", g.Errs, g.Counts)
	}
	g = h2ResidualFixture(t, "Verdict", []h2Action{{"replay", "Create a successor verdict row with the compared hashes."}}, row,
		"## Model actions\n\n- `replay` (actor System): creates a successor verdict row.\n")
	if g.Counts["authorization obligations admitted"] != 1 {
		t.Fatalf("the action name replay must not suppress a declared write: %v, %+v", g.Errs, g.Counts)
	}
}

func TestAuthorizationH2MatrixNoWriteStatement(t *testing.T) {
	row := "| `Verdict` (residual `unknown-never-compliant`) | `read` | `read` | `read`. NO preset holds `create`, `update` or `delete` |"
	matrix := "| action | reason |\n|---|---|\n| `compare_pins` | Verifies the five stored hashes and writes nothing to this Verdict row. |\n"
	g := h2ResidualFixture(t, "Verdict", []h2Action{{"compare_pins", "Compare pinned results against the existing verdict."}}, row, matrix)
	if hasErr(g, "System action 'Verdict.compare_pins'") || g.Counts["authorization obligations admitted"] != 0 {
		t.Fatalf("an action-specific matrix no-write statement owes no write admission: %v, %+v", g.Errs, g.Counts)
	}
}

func TestAuthorizationH2ConditionalNoWriteStillOwesAdmission(t *testing.T) {
	row := "| `Verdict` (residual `unknown-never-compliant`) | `read` | `read` | `read`. NO preset holds `create`, `update` or `delete` |"
	g := h2ResidualFixture(t, "Verdict", []h2Action{{"compare_pins", "Verify pinned hashes; no write occurs on a match, but append a successor Verdict on divergence."}}, row, "")
	if got := g.Counts["authorization obligations admitted"]; got != 1 {
		t.Fatalf("a conditional no-write statement still declares a write path: %v, %+v", g.Errs, g.Counts)
	}
}

func TestAuthorizationH2InvalidBranchWritesNothingStillOwesAdmission(t *testing.T) {
	row := "| `ApplicabilityElection` (residual `election-blanket-inherits`) | `read` | `read` | `read` |"
	description := "Record that a norm family arrived under this standing blanket election and inherited its posture. Precondition: the named release serves the arriving family; anything else writes nothing and is refused by name. Postcondition: one entry in inherited_family_arrivals and its rerun obligation recorded in the same transaction."
	g := h2ResidualFixture(t, "ApplicabilityElection", []h2Action{{"record_family_arrival", description}}, row, "", "MACHINE-WRITTEN{record_family_arrival}")
	if got := g.Counts["authorization obligations admitted"]; got != 1 {
		t.Fatalf("a refusal branch that writes nothing does not erase the admitted write: %v, %+v", g.Errs, g.Counts)
	}
}

func TestAuthorizationH2ResidualListClosesFallback(t *testing.T) {
	row := "| `Norm` (residual `rbac-reviewer-approval`) | `read` and `update` | `read` | `read` and `update`. NEVER `create`: `draft` is `System` |"
	g := h2ResidualFixture(t, "Norm", []h2Action{{"draft", "Create a draft norm from extracted source material."}}, row, "", "MACHINE-WRITTEN{advance}")
	if !hasErr(g, "System action 'Norm.draft' has no authorization row") {
		t.Fatalf("a closed machine-written list must block the residual fallback: %v", g.Errs)
	}
}

func TestAuthorizationH2RelationalRuleDoesNotUseResidualFallback(t *testing.T) {
	row := "| `Connector` (RULE `rbac-view-only-read`) | `read` | `read` | `read` |"
	g := h2ResidualFixture(t, "Connector", []h2Action{{"provision", "Create a connector."}}, row, "")
	if !hasErr(g, "System action 'Connector.provision' has no authorization row") {
		t.Fatalf("a RULE-owned row belongs to the relational policy, not the residual fallback: %v", g.Errs)
	}
}

func TestAuthorizationH2ResidualMixedVerbKeepsOtherDebt(t *testing.T) {
	row := "| `ConversionReport` (residual `conversion-report-first-class`) | `read` | `read` and `update` | `read` and `update`. NEVER `create`: `produce` is the run's own act through the machine |"
	g := h2ResidualFixture(t, "ConversionReport", []h2Action{
		{"produce", "Write the report for a live run. Postcondition: status pending."},
		{"reconcile", "Update the report after human review. Postcondition: status reconciled."},
	}, row, "")
	if hasErr(g, "System action 'ConversionReport.produce'") || !hasErr(g, "System action 'ConversionReport.reconcile' has no authorization row") {
		t.Fatalf("withheld create admits produce but a granted update does not admit reconcile: %v", g.Errs)
	}
	if got := g.Counts["authorization obligations admitted"]; got != 1 {
		t.Fatalf("only produce is admitted, got %d: %v", got, g.Errs)
	}
}
