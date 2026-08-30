package gates

import (
	"path/filepath"
	"strings"
	"testing"
)

// nfrFixture builds a minimal coherent design and swaps in the given NFR
// section (or none).
func nfrFixture(t *testing.T, nfrSection string) *Gate {
	t.Helper()
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      a = component \"a\" \"logic\" \"Go\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: p.a\n    code: [\"a/**\"]\n```\n" + nfrSection
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckC4(design)
}

func TestNFRRecordPresent(t *testing.T) {
	g := nfrFixture(t, nfrStub)
	if hasErr(g, "NFR") {
		t.Fatalf("a filled NFR record must pass: %v", g.Errs)
	}
	if g.Counts["nfr topics recorded"] != 3 {
		t.Fatalf("nfr topics = %d, want 3: %+v", g.Counts["nfr topics recorded"], g.Counts)
	}
}

func TestNFRRecordAbsent(t *testing.T) {
	g := nfrFixture(t, "")
	if !hasErr(g, "has no NFR record") {
		t.Fatalf("a missing NFR record must error: %v", g.Errs)
	}
}

func TestNFRRecordMissingTopic(t *testing.T) {
	g := nfrFixture(t, "\n## 8. NFR record\n\n- security: local CLI, no network surface.\n- capacity: thousands of records.\n")
	if !hasErr(g, "never mentions observability") {
		t.Fatalf("a missing topic must error: %v", g.Errs)
	}
	if hasErr(g, "never mentions security") || hasErr(g, "never mentions capacity") {
		t.Fatalf("present topics must not error: %v", g.Errs)
	}
}

func TestNFRRecordNonFunctionalHeading(t *testing.T) {
	g := nfrFixture(t, "\n### Non-functional requirements\n\nsecurity, capacity, and observability: all out of scope, recorded as such.\n")
	if hasErr(g, "NFR") {
		t.Fatalf("a 'non-functional' heading must satisfy the check: %v", g.Errs)
	}
}

func TestNFRSectionEndsAtSameLevelHeading(t *testing.T) {
	// the topics sit AFTER the next same-level heading, so they are outside
	// the NFR section and must not count
	g := nfrFixture(t, "\n## NFR record\n\nnothing here.\n\n## Later\n\nsecurity capacity observability\n")
	found := 0
	for _, e := range g.Errs {
		if strings.Contains(e, "never mentions") {
			found++
		}
	}
	if found != 3 {
		t.Fatalf("topics outside the section must not count (got %d of 3 errors): %v", found, g.Errs)
	}
}
