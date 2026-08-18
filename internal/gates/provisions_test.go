package gates

import (
	"path/filepath"
	"strings"
	"testing"
)

// A neutral three-boundary contract carrying capability keys. The dsl declares
// the elements the contract's boundaries map to.
func provisionsDesign(t *testing.T, boundariesYAML string) string {
	t.Helper()
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      api = component \"Api\" \"edge\" \"Go\"\n" +
		"      store = component \"Store\" \"state\" \"Go\"\n" +
		"      auth = component \"Auth\" \"tokens\" \"Go\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		boundariesYAML + "```\n"
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return design
}

const cleanProvisions = "  - id: api\n    code: [\"api/**\"]\n    consumes: [session-tokens]\n" +
	"  - id: store\n    code: [\"store/**\"]\n    provides: [record-store]\n" +
	"  - id: auth\n    code: [\"auth/**\"]\n    provides: [session-tokens]\n" +
	"dependency_rules:\n  allow:\n    - api -> auth\n    - api -> store\n"

func TestProvisionsCleanContractPasses(t *testing.T) {
	g := CheckC4(provisionsDesign(t, cleanProvisions))
	for _, e := range g.Errs {
		if strings.Contains(e, "capability") || strings.Contains(e, "provides") || strings.Contains(e, "consumes") {
			t.Fatalf("clean provisions must not error: %v", g.Errs)
		}
	}
	if g.Counts["provided keys"] != 2 || g.Counts["consumed keys"] != 1 {
		t.Fatalf("counts: %+v", g.Counts)
	}
}

func TestTwoProvidersForOneKeyIsAnError(t *testing.T) {
	y := strings.Replace(cleanProvisions, "provides: [record-store]", "provides: [session-tokens]", 1)
	g := CheckC4(provisionsDesign(t, y))
	if !hasErr(g, "has 2 providers") {
		t.Fatalf("disjoint-provisions law must fire: %v", g.Errs)
	}
}

func TestConsumingAnUnprovidedKeyIsAnError(t *testing.T) {
	y := strings.Replace(cleanProvisions, "provides: [session-tokens]", "provides: [other-thing]", 1)
	g := CheckC4(provisionsDesign(t, y))
	if !hasErr(g, "no boundary provides it") {
		t.Fatalf("satisfied-consumption law must fire: %v", g.Errs)
	}
}

func TestConsumingWithoutTheAllowEdgeIsAnError(t *testing.T) {
	y := strings.Replace(cleanProvisions, "    - api -> auth\n", "", 1)
	g := CheckC4(provisionsDesign(t, y))
	if !hasErr(g, "the capability view and the dependency view disagree") {
		t.Fatalf("capability-vs-dependency coherence must fire: %v", g.Errs)
	}
}

func TestSelfBindingAndMalformedKeysAreErrors(t *testing.T) {
	y := "  - id: api\n    code: [\"api/**\"]\n    provides: [session-tokens]\n    consumes: [session-tokens]\n" +
		"  - id: store\n    code: [\"store/**\"]\n    provides: [\"Bad_Key\"]\n"
	g := CheckC4(provisionsDesign(t, y))
	if !hasErr(g, "self-binding") {
		t.Fatalf("self-binding must fire: %v", g.Errs)
	}
	if !hasErr(g, "kebab-case keys") {
		t.Fatalf("key format must be validated: %v", g.Errs)
	}
}

func TestContractsWithoutProvisionsAreUntouched(t *testing.T) {
	y := "  - id: api\n    code: [\"api/**\"]\n  - id: store\n    code: [\"store/**\"]\n" +
		"  - id: auth\n    code: [\"auth/**\"]\n"
	g := CheckC4(provisionsDesign(t, y))
	if g.Counts["provided keys"] != 0 || g.Counts["consumed keys"] != 0 {
		t.Fatalf("no provisions declared, none may be counted: %+v", g.Counts)
	}
	for _, e := range g.Errs {
		if strings.Contains(e, "capability") {
			t.Fatalf("no provisions declared, no capability errors: %v", g.Errs)
		}
	}
}
