package experiments

// Runners for the consistency-layer Stage 3 experiments: one fixture per
// shipped Gy-rules rule. Each applies its mutation to the synthetic fixture
// design and asserts the rule's finding, then applies the near-neighbour that
// must not fire (the false-positive control), so an over-eager rule fails here
// as loudly as a missing one.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
)

func init() {
	RegisterRunner("rules_test.go",
		"rules-authz-missing", "rules-authz-orphan", "rules-authz-unknown-capability",
		"rules-fact-unresolved", "rules-values-disagree", "rules-payload-twin",
		"rules-supersession-cycle", "rules-effect-uncarried",
		"rules-produces-owes-admission", "rules-produces-unknown-action",
		"rules-actor-uncarried", "rules-carrier-misplaced", "rules-values-conflict",
		"rules-stale-reservation", "rules-superseded-in-packet",
		"rules-milestone-binding-stale", "rules-milestone-binding-phantom")
}

// rulesFindings runs Gy-rules and returns its ERROR and warning lines. The
// gate itself must run: every fixture projects, the shipped rules load, and
// no rule file stops at a limit; every ERROR is then a rule's finding.
func rulesFindings(t *testing.T, design string) []string {
	t.Helper()
	return rulesFindingsImpl(t, design, "")
}

// rulesFindingsImpl is rulesFindings with an implementation root (--impl).
func rulesFindingsImpl(t *testing.T, design, impl string) []string {
	t.Helper()
	g := gates.CheckRulesImpl(design, impl, false)
	for _, e := range g.Errs {
		if strings.HasPrefix(e, "cannot project") || strings.HasPrefix(e, "cannot read the implementation") || strings.HasPrefix(e, "the shipped consistency rules") || strings.HasPrefix(e, "rules/consistency/") {
			t.Fatalf("Gy-rules did not run: %v", g.Errs)
		}
	}
	return append(append([]string(nil), g.Errs...), g.Warns...)
}

func requireRuleFinding(t *testing.T, name, design string) {
	t.Helper()
	e := experimentNamed(t, name)
	if got := rulesFindings(t, design); !containsAny(got, e.ExpectSubstr) {
		t.Fatalf("%s escaped Gy-rules: %v", e.Name, got)
	}
}

func refuteRuleFinding(t *testing.T, design, code string) {
	t.Helper()
	for _, f := range rulesFindings(t, design) {
		if strings.Contains(f, ": "+code) {
			t.Fatalf("near-neighbour fired %s: %s", code, f)
		}
	}
}

const authorizationInventory = "# Authorization\n\n<!-- machinery:authorization-inventory -->\n\n" +
	"| authorization subject | admission |\n|---|---|\n"

func systemPublish(t *testing.T, design string) {
	t.Helper()
	editFile(t, filepath.Join(design, "widget.modelith.yaml"), "      - name: publish\n", "      - name: publish\n        actor: System\n")
}

func admit(t *testing.T, design, subject, admission string) {
	t.Helper()
	mustWrite(t, filepath.Join(design, "AUTHORIZATION.md"), authorizationInventory+"| "+subject+" | "+admission+" |\n")
}

func contractOf(t *testing.T, design, old, contract string) {
	t.Helper()
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"), "| "+old+" |", "| "+contract+" |")
}

func TestRulesAuthzMissing(t *testing.T) {
	design, _ := fixture(t)
	systemPublish(t, design)
	requireRuleFinding(t, "rules-authz-missing", design)

	near, _ := fixture(t)
	systemPublish(t, near)
	admit(t, near, "Widget.publish", "`app`")
	refuteRuleFinding(t, near, "authz_")
}

func TestRulesAuthzOrphan(t *testing.T) {
	design, _ := fixture(t)
	admit(t, design, "Widget.publish", "`app`")
	requireRuleFinding(t, "rules-authz-orphan", design)

	near, _ := fixture(t)
	systemPublish(t, near)
	admit(t, near, "Widget.publish", "(no authorization: the fixture's batch publisher)")
	refuteRuleFinding(t, near, "authz_")
}

// An admission naming a real workspace.dsl element resolves; the same name
// with a fabricated suffix does not.
func TestRulesAuthzUnknownCapability(t *testing.T) {
	design, _ := fixture(t)
	systemPublish(t, design)
	admit(t, design, "Widget.publish", "`appX`")
	requireRuleFinding(t, "rules-authz-unknown-capability", design)

	near, _ := fixture(t)
	systemPublish(t, near)
	admit(t, near, "Widget.publish", "`app`")
	refuteRuleFinding(t, near, "authz_unknown_capability")
}

// USES{Widget.status} joins the attr id Widget.status (fact columns carry the
// stable id without its attr: prefix); USES{Widget.stat} resolves nowhere.
func TestRulesFactUnresolved(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "actor may publish", "actor may publish. USES{Widget.stat}")
	requireRuleFinding(t, "rules-fact-unresolved", design)

	near, _ := fixture(t)
	contractOf(t, near, "actor may publish", "actor may publish. USES{Widget.status}")
	refuteRuleFinding(t, near, "fact_unresolved")
}

// A named vocabulary with no enum is declared by its groups, which must agree;
// the same set spelled twice is silent.
func TestRulesValuesConflict(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "commit status", "commit status. VALUES reason{late, lost}")
	contractOf(t, design, "stash pending", "stash pending. VALUES reason{late, early}")
	requireRuleFinding(t, "rules-values-conflict", design)

	near, _ := fixture(t)
	contractOf(t, near, "commit status", "commit status. VALUES reason{late, lost}")
	contractOf(t, near, "stash pending", "stash pending. VALUES reason{lost, late}")
	refuteRuleFinding(t, near, "values_conflict")
}

func TestRulesValuesDisagree(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "actor may publish", "actor may publish. VALUES WidgetStatus{Draft, Published, Archived}")
	requireRuleFinding(t, "rules-values-disagree", design)

	near, _ := fixture(t)
	contractOf(t, near, "actor may publish", "actor may publish. VALUES WidgetStatus{Draft, Published}")
	refuteRuleFinding(t, near, "values_disagree")
}

const eventContract = "\n## 5. Event contracts\n\n| event | producer | consumer | delivery | payload |\n|---|---|---|---|---|\n" +
	"| `widget.published` | app | storelib | at-least-once | `Widget.id`, `Widget.status` |\n"

func payloadTwin(t *testing.T, design, fields string) {
	t.Helper()
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"), "\n## 6. Dependency mitigation posture", eventContract+"\n## 6. Dependency mitigation posture")
	matrix := filepath.Join(design, "machines", "Widget.matrix.md")
	editFile(t, matrix, "## (c) Transition matrix", "## (b) Emitted events\n\n| name | kind | event | pre / post |\n|---|---|---|---|\n"+
		"| `announce` | action | `widget.published` | payload {"+fields+"} |\n\n## (c) Transition matrix")
}

func TestRulesPayloadTwin(t *testing.T) {
	design, _ := fixture(t)
	payloadTwin(t, design, "Widget.id")
	requireRuleFinding(t, "rules-payload-twin", design)

	near, _ := fixture(t)
	payloadTwin(t, near, "Widget.id, Widget.status")
	refuteRuleFinding(t, near, "payload_twin")
}

func supersessionTable(t *testing.T, design string, rows ...string) {
	t.Helper()
	table := "\n## 10. Types\n\n| type | replaces |\n|---|---|\n" + strings.Join(rows, "\n") + "\n"
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"), "\n## 9. NFR record", table+"\n## 9. NFR record")
}

func TestRulesSupersessionCycle(t *testing.T) {
	design, _ := fixture(t)
	supersessionTable(t, design,
		"| WidgetA | SUPERSEDES{type:WidgetB} |",
		"| WidgetB | SUPERSEDES{type:WidgetC} |",
		"| WidgetC | SUPERSEDES{type:WidgetA} |")
	requireRuleFinding(t, "rules-supersession-cycle", design)

	near, _ := fixture(t)
	supersessionTable(t, near,
		"| WidgetA | SUPERSEDES{type:WidgetB} |",
		"| WidgetB | SUPERSEDES{type:WidgetC} |",
		"| WidgetC | the original type |")
	refuteRuleFinding(t, near, "supersession_cycle")
}

// WRITES{} declares a read-only action, which owes no carrier; a writing
// action with no CARRIES{} is a finding.
func TestRulesEffectUncarried(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "commit status", "commit status. WRITES{Widget.status}")
	requireRuleFinding(t, "rules-effect-uncarried", design)

	near, _ := fixture(t)
	contractOf(t, near, "commit status", "commit status. WRITES{}")
	refuteRuleFinding(t, near, "effect_uncarried")
}

// A produced action owes an admission though its Modelith actor is not
// System; an admission discharges the produced obligation.
func TestRulesProducesOwesAdmission(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "commit status", "commit status. PRODUCES{Widget.publish}")
	requireRuleFinding(t, "rules-produces-owes-admission", design)

	near, _ := fixture(t)
	contractOf(t, near, "commit status", "commit status. PRODUCES{Widget.publish}")
	admit(t, near, "Widget.publish", "`app`")
	refuteRuleFinding(t, near, "authz_")
}

// A PRODUCES member the model does not declare is its own finding, and owes
// no admission (there is no action to admit); a declared one is silent.
func TestRulesProducesUnknownAction(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "commit status", "commit status. PRODUCES{Widget.unpublish}")
	requireRuleFinding(t, "rules-produces-unknown-action", design)
	refuteRuleFinding(t, design, "authz_missing")

	near, _ := fixture(t)
	contractOf(t, near, "commit status", "commit status. PRODUCES{Widget.publish}")
	refuteRuleFinding(t, near, "produces_unknown_action")
}

// An actor owes a carrier whether or not it lists WRITES: the fixture's
// saveWidget actor carries its column, and dropping the group is a finding.
func TestRulesActorUncarried(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"), "atomic persist CARRIES{column:Widget.status}", "atomic persist")
	requireRuleFinding(t, "rules-actor-uncarried", design)

	near, _ := fixture(t)
	refuteRuleFinding(t, near, "effect_uncarried")
}

// Carriers belong to actions and actors: CARRIES{} on a guard row is a
// finding, and the same group on the actor row is not.
func TestRulesCarrierMisplaced(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "actor may publish", "actor may publish. CARRIES{signal:published}")
	requireRuleFinding(t, "rules-carrier-misplaced", design)

	near, _ := fixture(t)
	refuteRuleFinding(t, near, "carrier_misplaced")
}

// A reservation claims a type is not defined yet; once a contract row owns
// the type, the claim is stale. A reservation of a type nobody owns, and one
// of a different type beside an owner, are silent.
func TestRulesStaleReservation(t *testing.T) {
	design, _ := fixture(t)
	supersessionTable(t, design,
		"| WireDraft | RESERVED{type:WidgetReceipt} |",
		"| WidgetReceipt | SUPERSEDES{type:WidgetReceiptV0} |")
	requireRuleFinding(t, "rules-stale-reservation", design)

	near, _ := fixture(t)
	supersessionTable(t, near, "| WireDraft | RESERVED{type:WidgetReceipt} |")
	refuteRuleFinding(t, near, "stale_reservation")

	other, _ := fixture(t)
	supersessionTable(t, other,
		"| WireDraft | RESERVED{type:WidgetManifest} |",
		"| WidgetReceipt | SUPERSEDES{type:WidgetReceiptV0} |")
	refuteRuleFinding(t, other, "stale_reservation")
}

func slicesCiting(t *testing.T, design string, rows ...string) {
	t.Helper()
	cites := ""
	for _, row := range rows {
		cites += "          - row:ARCHITECTURE.md#types#" + row + "\n"
	}
	mustWrite(t, filepath.Join(design, "slices.yaml"), "milestones:\n  - id: M1\n    slices:\n      - id: M1-S1\n        cites:\n"+cites)
}

// A packet must not carry a superseded definition as its current contract:
// citing the replaced row is a finding, citing the replacing row is not.
func TestRulesSupersededInPacket(t *testing.T) {
	design, _ := fixture(t)
	supersessionTable(t, design, "| WidgetV2 | SUPERSEDES{type:WidgetV1} |", "| WidgetV1 | the original type |")
	slicesCiting(t, design, "WidgetV1")
	requireRuleFinding(t, "rules-superseded-in-packet", design)

	near, _ := fixture(t)
	supersessionTable(t, near, "| WidgetV2 | SUPERSEDES{type:WidgetV1} |", "| WidgetV1 | the original type |")
	slicesCiting(t, near, "WidgetV2")
	refuteRuleFinding(t, near, "superseded_in_packet")
}

// widgetOracleIDs returns the first two stable ids of the fixture's
// committed Widget oracle.
func widgetOracleIDs(t *testing.T, design string) (string, string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(design, "machines", "Widget.oracle.md"))
	if err != nil {
		t.Fatal(err)
	}
	ids := regexp.MustCompile(`\b[A-Z][A-Z0-9]*-[0-9a-f]{6}\b`).FindAllString(string(body), -1)
	if len(ids) < 2 || ids[0] == ids[1] {
		t.Fatalf("fixture drift: the Widget oracle has ids %v", ids)
	}
	return ids[0], ids[1]
}

// bindingFixture writes a locked suite that binds the first Widget oracle id,
// and a BUILD.md binding table with the given rows (%1 and %2 stand for the
// first two stable ids).
func bindingFixture(t *testing.T, rows ...string) (design, impl string) {
	t.Helper()
	design, impl = fixture(t)
	first, second := widgetOracleIDs(t, design)
	mustWrite(t, filepath.Join(impl, "internal", "app", "app_test.go"),
		"package app\n\nimport \"testing\"\n\nfunc TestPublish(t *testing.T) { t.Log(\""+first+"\") }\n")
	table := "\n## Oracle bindings\n\n| oracle | bound at |\n|---|---|\n"
	for _, r := range rows {
		table += strings.NewReplacer("%1", first, "%2", second).Replace(r) + "\n"
	}
	editFile(t, filepath.Join(design, "BUILD.md"), "\n## State migration", table+"\n## State migration")
	return design, impl
}

func requireImplRuleFinding(t *testing.T, name, design, impl string) {
	t.Helper()
	e := experimentNamed(t, name)
	if got := rulesFindingsImpl(t, design, impl); !containsAny(got, e.ExpectSubstr) {
		t.Fatalf("%s escaped Gy-rules under --impl: %v", e.Name, got)
	}
}

func refuteImplRuleFinding(t *testing.T, design, impl, code string) {
	t.Helper()
	for _, f := range rulesFindingsImpl(t, design, impl) {
		if strings.Contains(f, ": "+code) {
			t.Fatalf("near-neighbour fired %s: %s", code, f)
		}
	}
}

// BUILD.md says an oracle is unbound while the locked suite binds it. The
// agreeing table is silent, and so is the disagreeing one without --impl.
func TestRulesMilestoneBindingStale(t *testing.T) {
	design, impl := bindingFixture(t, "| %1 | unbound |")
	requireImplRuleFinding(t, "rules-milestone-binding-stale", design, impl)
	refuteRuleFinding(t, design, "milestone_binding_stale")

	near, nearImpl := bindingFixture(t, "| %1 | `internal/app/app_test.go` |", "| %2 | unbound |")
	refuteImplRuleFinding(t, near, nearImpl, "milestone_binding_")
}

// BUILD.md names a path that binds nothing for the oracle: a file that binds
// another oracle, and a file that does not exist. The agreeing row is
// silent, and the disagreeing ones are silent without --impl.
func TestRulesMilestoneBindingPhantom(t *testing.T) {
	design, impl := bindingFixture(t, "| %2 | `internal/app/app_test.go` |", "| %1 | internal/app/gone_test.go |")
	requireImplRuleFinding(t, "rules-milestone-binding-phantom", design, impl)
	if got := rulesFindingsImpl(t, design, impl); !containsAny(got, "milestone_binding_phantom (path 'internal/app/gone_test.go')") {
		t.Fatalf("a bound-at path naming no test file must be a phantom: %v", got)
	}
	refuteRuleFinding(t, design, "milestone_binding_phantom")

	near, nearImpl := bindingFixture(t, "| %1 | internal/app/app_test.go |")
	refuteImplRuleFinding(t, near, nearImpl, "milestone_binding_")
}
