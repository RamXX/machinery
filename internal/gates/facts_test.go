package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

const factModel = `kind: DomainModel
version: v1
enums:
  OrderState:
    values: [{name: Placed}, {name: Paid}]
entities:
  Order:
    attributes:
      - {name: state, type: OrderState}
      - {name: order_total, type: integer}
    actions: [{name: markPaid}]
    invariants:
      - {id: order-paid-final, statement: a paid order stays paid}
`

func factFixture(t *testing.T, contract, payload string) *Gate {
	t.Helper()
	return CheckTraceability(factDesign(t, contract, payload))
}

func factDesign(t *testing.T, contract, payload string) string {
	t.Helper()
	design := t.TempDir()
	for _, dir := range []string{"machines", "formal"} {
		if err := os.MkdirAll(filepath.Join(design, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	machine := strings.Replace(wiringMachine, `"initial": "Placed",`, `"initial": "Placed", "context": {"retry_count": 0},`, 1)
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), factModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), machine)
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"),
		"machine: Order\npattern: control-flow-only\nreason: fact fixture\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n"+
			"| `saveOrder` | actor | "+contract+" | `order-paid-final` |\n")
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n| `Order` | in-memory |\n\n" +
		wiringHeader + "| markPaid | orders | payments | " + payload + " | at-least-once | none | event_id |\n" +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| order-paid-final | `Order` guard |\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return design
}

func TestFactsH2ClassBFalsePositives(t *testing.T) {
	t.Run("class C content knob", func(t *testing.T) {
		design := factDesign(t, "the `target_artifact_lexicon` knob's Class C re-dating formula; persists `ghost_fact`", "`event_id`")
		if err := os.MkdirAll(filepath.Join(design, "content"), 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(design, "content", "knob-register.yaml"), "knobs:\n  - {key: target_artifact_lexicon, class: C}\n")
		g := CheckTraceability(design)
		if hasErr(g, "unresolved fact 'target_artifact_lexicon'") || !hasErr(g, "unresolved fact 'ghost_fact'") {
			t.Fatalf("knob declaration must resolve only its key: %v", g.Errs)
		}
	})
	t.Run("rejected former heuristic", func(t *testing.T) {
		g := factFixture(t, "The earlier reading, that a finding carrying an `interpretation_id` stands independently, proves only a citation; persists `ghost_fact`", "`event_id`")
		if hasErr(g, "unresolved fact 'interpretation_id'") || !hasErr(g, "unresolved fact 'ghost_fact'") {
			t.Fatalf("rejected heuristic must not assert a fact: %v", g.Errs)
		}
	})
	for _, tc := range []struct{ name, row string }{
		{"DocumentVersion outbox", "records `version.withdrawn` in the transaction, idempotent by `withdrawn_at`"},
		{"GapAnalysis consumer", "retires the run with the withdrawal and `withdrawn_at`, never a successor"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := factFixture(t, tc.row, "withdrawn version ref, withdrawn_at, and prior status")
			if hasErr(g, "unresolved fact 'withdrawn_at'") {
				t.Fatalf("payload cell declares the field: %v", g.Errs)
			}
		})
	}
	t.Run("relationship name", func(t *testing.T) {
		design := factDesign(t, "subject resolves inside `Order.tenant`", "`event_id`")
		model := strings.Replace(factModel, "entities:\n", "entities:\n  Tenant: {}\n", 1)
		model = strings.Replace(model, "    actions: [{name: markPaid}]", "    relationships: [{entity: Tenant, cardinality: 'n:1'}]\n    actions: [{name: markPaid}]", 1)
		mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
		g := CheckTraceability(design)
		if hasErr(g, "unresolved fact 'Order.tenant'") {
			t.Fatalf("relationship-backed name must resolve: %v", g.Errs)
		}
	})
	t.Run("vertical field", func(t *testing.T) {
		design := factDesign(t, "rule whose `deadline_confidence` is capped", "`event_id`")
		if err := os.MkdirAll(filepath.Join(design, "content", "verticals"), 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(design, "content", "verticals", "pipeline.vertical.yaml"), "rules:\n  - id: rule-1\n    deadline_confidence: verify_with_customer\n")
		g := CheckTraceability(design)
		if hasErr(g, "unresolved fact 'deadline_confidence'") {
			t.Fatalf("vertical field must resolve: %v", g.Errs)
		}
	})
	t.Run("relationship key", func(t *testing.T) {
		design := factDesign(t, "read the seat's keyed `identity_id` reference", "`event_id`")
		model := strings.Replace(factModel, "entities:\n", "entities:\n  Identity: {}\n", 1)
		model = strings.Replace(model, "    actions: [{name: markPaid}]", "    relationships: [{entity: Identity, cardinality: 'n:1'}]\n    actions: [{name: markPaid}]", 1)
		mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
		g := CheckTraceability(design)
		if hasErr(g, "unresolved fact 'identity_id'") {
			t.Fatalf("keyed relation must resolve: %v", g.Errs)
		}
	})
	t.Run("architecture join key", func(t *testing.T) {
		design := factDesign(t, "join record keyed by (`principal_id`, `tenant_role_id`)", "`event_id`")
		f := filepath.Join(design, "ARCHITECTURE.md")
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, f, string(b)+"\n| `PrincipalTenantRole` | the pair (`principal_id`, `tenant_role_id`) |\n")
		g := CheckTraceability(design)
		if hasErr(g, "unresolved fact 'principal_id'") || hasErr(g, "unresolved fact 'tenant_role_id'") {
			t.Fatalf("architecture join keys must resolve: %v", g.Errs)
		}
	})
}

func TestFactsH2UntypedPinnedInputsIsOneGap(t *testing.T) {
	design := factDesign(t, "writes the `pinned_inputs` attribute under `document_version_shas`, `case_member_refs`, and `release_ir_hash`; persists `ghost_fact`", "`event_id`")
	model := strings.Replace(factModel, "      - {name: order_total, type: integer}", "      - {name: order_total, type: integer}\n      - name: pinned_inputs\n        type: string\n        description: 'THE KEY NAMES ARE CLOSED AND STATED HERE: `document_version_shas`, `case_member_refs`, `release_ir_hash`. A key not carried is missing.'", 1)
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
	g := CheckTraceability(design)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "pinned_inputs needs a typed map") || strings.Count(joined, "pinned_inputs needs a typed map") != 1 {
		t.Fatalf("closed prose keys are one model typing gap: %v", g.Errs)
	}
	for _, key := range []string{"document_version_shas", "case_member_refs", "release_ir_hash"} {
		if hasErr(g, "unresolved fact '"+key+"'") {
			t.Fatalf("duplicate key finding for %s: %v", key, g.Errs)
		}
	}
	if !hasErr(g, "unresolved fact 'ghost_fact'") {
		t.Fatalf("unrelated fact must still fail: %v", g.Errs)
	}
}

func TestFactsResolveAcrossModelContextAndEventPayload(t *testing.T) {
	g := factFixture(t, "reads `order_total`, increments `retry_count`, records `event_id`", "`event_id`")
	if hasErr(g, "unresolved fact") {
		t.Fatalf("facts declared by the model, context, or event contract must resolve: %v", g.Errs)
	}
}

func TestFactsRejectUnknownNamedUnitFact(t *testing.T) {
	g := factFixture(t, "persists `ghost_fact`", "`event_id`")
	if !hasErr(g, "unresolved fact 'ghost_fact'") {
		t.Fatalf("a named unit fact with no declaration must fail: %v", g.Errs)
	}
}

func TestFactsEventPayloadDeclaresFact(t *testing.T) {
	g := factFixture(t, "persists `event_fact`", "`event_fact`")
	if hasErr(g, "unresolved fact 'event_fact'") {
		t.Fatalf("an event-contract payload field is a readable fact declaration: %v", g.Errs)
	}
}

func TestFactsDerivedWaiverRequiresReason(t *testing.T) {
	g := factFixture(t, "computes `risk_score`; derived: risk_score (weighted fraud inputs)", "`event_id`")
	if hasErr(g, "unresolved fact 'risk_score'") {
		t.Fatalf("a reasoned row-local derived waiver must close the fact: %v", g.Errs)
	}
	g = factFixture(t, "computes `risk_score`; derived: risk_score ()", "`event_id`")
	if !hasErr(g, "derived waiver for 'risk_score' names no reason") {
		t.Fatalf("an empty derived reason must fail: %v", g.Errs)
	}
}

func TestFactsResolveModelEnumActionAndSameRowValues(t *testing.T) {
	g := factFixture(t, "`Paid`, `markPaid`, VALUES{review_pending, release_ready}, then `release_ready`", "`event_id`")
	if hasErr(g, "unresolved fact") {
		t.Fatalf("declared enum, action, and row vocabulary members resolve: %v", g.Errs)
	}
}

func TestFactsRejectQualifiedUnknownFact(t *testing.T) {
	g := factFixture(t, "persists `Order.ghost_field`", "`event_id`")
	if !hasErr(g, "unresolved fact 'Order.ghost_field'") {
		t.Fatalf("qualified unknown fact must fail: %v", g.Errs)
	}
}

func TestFactsRejectSingleWordFactInSingleWordModel(t *testing.T) {
	g := factFixture(t, "persists `stage`", "`event_id`")
	if !hasErr(g, "unresolved fact 'stage'") {
		t.Fatalf("single-word fact must be checked when the model uses that grammar: %v", g.Errs)
	}
}

func TestFactsDerivedWaiverMustBeWellFormedAndNamed(t *testing.T) {
	g := factFixture(t, "uses `risk_score`; derived: risk_score (calculated) and derived: ghost_fact", "`event_id`")
	if !hasErr(g, "malformed derived waiver") {
		t.Fatalf("one valid waiver must not hide a malformed sibling: %v", g.Errs)
	}
	g = factFixture(t, "uses `risk_score`; derived: ghost_fact (unrelated)", "`event_id`")
	if !hasErr(g, "waiver for 'ghost_fact' names no fact") {
		t.Fatalf("a waiver must name a fact on the row: %v", g.Errs)
	}
}

func TestFactUniverseIncludesModelVocabularyActionsAndEvents(t *testing.T) {
	dm, err := ir.LoadYAML([]byte(`kind: DomainModel
version: v1
enums:
  RefusalReason:
    values: [{name: wrong_tenant}]
entities:
  Finding:
    attributes: [{name: standing_basis, type: string}, {name: stage, type: string}, {name: reviewStatus, type: string}]
    actions: [{name: raise_remediation}]
`))
	if err != nil {
		t.Fatal(err)
	}
	u := declaredFacts(dm, t.TempDir(), wiringHeader+"| review_started | a | b | stage | once | none | key |\n")
	for _, name := range []string{"wrong_tenant", "raise_remediation", "Finding.raise_remediation", "Finding.standing_basis", "review_started"} {
		if !u.declared[name] {
			t.Errorf("%s missing from fact universe", name)
		}
	}
	if !u.snake || !u.camel || !u.single {
		t.Fatalf("attribute spelling styles missing: %+v", u)
	}
	if !u.candidate("Finding.standing_basis", "`Finding.standing_basis`", 0) || !u.candidate("reviewStatus", "`reviewStatus`", 0) || !u.candidate("stage", "persists `stage`", 9) {
		t.Fatal("model-derived grammar does not recognize the three fact shapes")
	}
}

func TestFactsDoNotTreatReasonMembersOrProducerNamesAsFacts(t *testing.T) {
	g := factFixture(t,
		"records reason class (`wrong_tenant`, `not_verified`) and closure basis `source_observing_again`; closed producer is `Order.reindex`",
		"`event_id`")
	for _, name := range []string{"wrong_tenant", "not_verified", "source_observing_again", "Order.reindex"} {
		if hasErr(g, "unresolved fact '"+name+"'") {
			t.Fatalf("%s is a value or producer, not a fact: %v", name, g.Errs)
		}
	}
}

func TestFactsDoNotRequireNegatedOrRejectedColumns(t *testing.T) {
	g := factFixture(t,
		"There is no `correlation_absent` basis. The `promotion_requested_at` and `promotion_requesting_principal_id` columns were weighed and refused; persists `ghost_fact`.",
		"`event_id`")
	for _, name := range []string{"correlation_absent", "promotion_requested_at", "promotion_requesting_principal_id"} {
		if hasErr(g, "unresolved fact '"+name+"'") {
			t.Fatalf("negative statement about %s is not a fact requirement: %v", name, g.Errs)
		}
	}
	if !hasErr(g, "unresolved fact 'ghost_fact'") {
		t.Fatalf("positive write must still be checked: %v", g.Errs)
	}
}

func TestFactsDoNotTreatClassificationAndFailureCodesAsFields(t *testing.T) {
	g := factFixture(t,
		"for a `study_item` derivation, marks the call `unvalidated_by_platform`; routes work only for `defective_output`; classifies a `controlled_upstream` with no basis; records the reason class `refused` or `insert_failed` on error; persists `ghost_fact`",
		"`event_id`")
	for _, name := range []string{"study_item", "unvalidated_by_platform", "defective_output", "controlled_upstream", "insert_failed"} {
		if hasErr(g, "unresolved fact '"+name+"'") {
			t.Fatalf("%s is a quoted value, not a field: %v", name, g.Errs)
		}
	}
	if !hasErr(g, "unresolved fact 'ghost_fact'") {
		t.Fatalf("positive fact use must still fail: %v", g.Errs)
	}
}

func TestFactsDoNotTreatKindAndFailureClassMembersAsFields(t *testing.T) {
	g := factFixture(t, "derivation source (kind `study_item`, ref input); records the failure class `refused` or `insert_failed` on error", "`event_id`")
	for _, name := range []string{"study_item", "insert_failed"} {
		if hasErr(g, "unresolved fact '"+name+"'") {
			t.Fatalf("%s is a vocabulary member: %v", name, g.Errs)
		}
	}
}
