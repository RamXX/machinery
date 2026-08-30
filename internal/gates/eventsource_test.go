package gates

import (
	"path/filepath"
	"testing"
)

func eventSourceFixture(t *testing.T, eventSection string) *Gate {
	t.Helper()
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      app = component \"App\" \"logic\" \"Go\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: app\n    code: [\"app/**\"]\n```\n" + eventSection + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckC4(design)
}

// eventTable is a COMPLETE event-contract table (every column the format
// contract names, every cell answered, participants declared), so a fixture
// built to exercise the source-note check reports only that check's findings.
const eventTable = "| event | producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|---|\n" +
	"| paid | app | app | Order.id | at-least-once | none | Order.id |\n"

func TestEventTableSourceNotePasses(t *testing.T) {
	g := eventSourceFixture(t, "\n## Events\n\nSource: swept the emit call sites and the topic bindings.\n\n"+eventTable)
	if hasErr(g, "enumeration source") {
		t.Fatalf("a sourced table must pass: %v", g.Errs)
	}
	if g.Counts["event tables with sources"] != 1 {
		t.Fatalf("sourced table not counted: %+v", g.Counts)
	}
}

func TestEventTableEmbedMarkerPasses(t *testing.T) {
	g := eventSourceFixture(t, "\n## Events\n\n<!-- machinery:embed from=\"../parent/ARCHITECTURE.md\" table=\"producer,consumer,delivery\" claims=\"subset\" -->\n"+eventTable)
	if hasErr(g, "enumeration source") {
		t.Fatalf("an embed marker is a source statement: %v", g.Errs)
	}
}

func TestEventTableNoSourceErrors(t *testing.T) {
	g := eventSourceFixture(t, "\n## Events\n\n"+eventTable)
	if !hasErr(g, "enumeration source") {
		t.Fatalf("an unsourced event table must error: %v", g.Errs)
	}
}

func TestEventTableSourceTooFarAbove(t *testing.T) {
	g := eventSourceFixture(t, "\n## Events\n\nSource: swept.\n\nfiller\nfiller\nfiller\nfiller\nfiller\n\n"+eventTable)
	if !hasErr(g, "enumeration source") {
		t.Fatalf("a note beyond the lookback must not satisfy the check: %v", g.Errs)
	}
}

func TestNonEventTablesCarryNoSourceObligation(t *testing.T) {
	g := eventSourceFixture(t, "\n## Other\n\n| thing | note |\n|---|---|\n| x | y |\n")
	if hasErr(g, "enumeration source") {
		t.Fatalf("a non-event table carries no source obligation: %v", g.Errs)
	}
}
