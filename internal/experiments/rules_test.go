package experiments

// Runners for the consistency-layer Stage 3 experiments: one fixture per
// shipped Gy-rules rule. Each applies its mutation to the synthetic fixture
// design and asserts the rule's finding, then applies the near-neighbour that
// must not fire (the false-positive control), so an over-eager rule fails here
// as loudly as a missing one.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
)

func init() {
	RegisterRunner("rules_test.go",
		"rules-authz-missing", "rules-authz-orphan", "rules-authz-unknown-capability",
		"rules-fact-unresolved", "rules-values-disagree", "rules-payload-twin",
		"rules-supersession-cycle", "rules-effect-uncarried",
		"rules-produces-owes-admission", "rules-produces-unknown-action")
}

// rulesFindings runs Gy-rules and returns its SHADOW and warning lines. The
// gate itself must not fail: every fixture projects and every rule runs.
func rulesFindings(t *testing.T, design string) []string {
	t.Helper()
	g := gates.CheckRules(design, false)
	if len(g.Errs) != 0 {
		t.Fatalf("Gy-rules errored: %v", g.Errs)
	}
	return append(append([]string(nil), g.Shadow...), g.Warns...)
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
// action with no CARRIES{} is a warning.
func TestRulesEffectUncarried(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "commit status", "commit status. WRITES{Widget.status}")
	requireRuleFinding(t, "rules-effect-uncarried", design)
	if g := gates.CheckRules(design, false); !containsAny(g.Warns, "effect_uncarried") {
		t.Fatalf("effect_uncarried is warning tier, not shadow: %v", g.Warns)
	}

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
