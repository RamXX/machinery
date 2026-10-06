package gates

import (
	"strings"
	"testing"
)

// threatModel is a minimal domain model: Session reads as security-relevant
// (its definition authenticates), Note does not.
const threatModel = `kind: DomainModel
version: v1
title: Thing
entities:
  Session:
    definition: "A local credential that authenticates the acting user."
    actions:
      - name: login
    invariants:
      - id: session-requires-valid-credential
  Note:
    definition: "A free-text remark."
`

// threatLedgerRows is the Session adversary set: two tested rows, one
// accepted risk, and five not_applicable rows Gb must not demand.
const threatLedgerRows = `subjects:
  - subject: Session
    kinds: [authenticates]
    adversaries:
      - adversary: outsider_without_key
        invariant: session-requires-valid-credential
        negative_test: SESS-forged-credential-refused
      - adversary: trusted_key_holder
        invariant: session-key-role-bound
        negative_test: SESS-wrong-role-key-refused
      - adversary: unsigned_document_controller
        not_applicable: "a session carries no unsigned digest"
      - adversary: valid_signature_wrong_meaning
        not_applicable: "no signed meaning"
      - adversary: optional_input_omission
        not_applicable: "no optional input"
      - adversary: duplicate_replay_reorder
        accepted_risk:
          owner: Product owner
          date: 2026-10-05
          reason: "local single-user store"
          visible_in: "session status prints replay-unchecked"
      - adversary: signed_field_injection
        not_applicable: "no signed text fields"
      - adversary: cross_bundle_substitution
        not_applicable: "no bundles"
`

const threatLedgerEnforce = "mode: enforce\nenforced_since: 2026-10-01\n" + threatLedgerRows
const threatLedgerAudit = "mode: audit\n" + threatLedgerRows

// threatCriteria is a compliant security-relevant milestone body: paired
// criteria, a complete threat table, and the third RED quality check.
const threatCriteria = `Accept: a valid credential opens an active session.
Refuse: a forged credential is refused with "credential signature invalid".

| adversary | negative test | accepted risk |
|---|---|---|
| outsider_without_key | SESS-forged-credential-refused | - |
| trusted_key_holder | SESS-wrong-role-key-refused | - |
| duplicate_replay_reorder | - | Product owner, 2026-10-05 |

Pass-wrongly: a check that accepts any well-formed token passes every positive test.
`

func threatPlan(m1Body string) string {
	return "# B\n\nMode: full\n\n## 9. Build plan\n\n" +
		"**M0 - Walking skeleton.** One thread. NFR: error envelope. DoD: T-CMD-01 green.\n\n" +
		"**M1 - Session slice.** Login and resume for Session.\nDoD: T-CMD-02 green.\n" + m1Body
}

func threatFixture(t *testing.T, build, ledger string, extra map[string]string) string {
	t.Helper()
	files := map[string]string{"domain.modelith.yaml": threatModel}
	if ledger != "" {
		files["threats.yaml"] = ledger
	}
	for k, v := range extra {
		files[k] = v
	}
	return writeBuildPlanFixture(t, build, files)
}

func TestGbThreatFullModeCompliantMilestoneIsClean(t *testing.T) {
	design := threatFixture(t, threatPlan(threatCriteria), threatLedgerEnforce, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 || len(g.Notes) != 0 {
		t.Fatalf("a compliant security-relevant milestone must be clean: errs=%v notes=%v", g.Errs, g.Notes)
	}
	if g.Counts["security-relevant milestones"] != 1 {
		t.Errorf("checked line must count the one security-relevant milestone: %+v", g.Counts)
	}
}

func TestGbThreatEnforceFindings(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"only positive criteria",
			strings.Replace(threatCriteria, "Refuse: a forged credential is refused with \"credential signature invalid\".\n", "", 1),
			"only positive criteria"},
		{"no criteria at all",
			strings.Replace(strings.Replace(threatCriteria, "Accept: a valid credential opens an active session.\n", "", 1),
				"Refuse: a forged credential is refused with \"credential signature invalid\".\n", "", 1),
			"no paired acceptance criteria"},
		{"accept without its refuse",
			"Accept: a second positive property.\nSome prose.\n" + threatCriteria,
			"an Accept: line not followed by its Refuse: line"},
		{"refuse without an accept",
			"Refuse: an orphan refusal.\n\n" + threatCriteria,
			"a Refuse: line with no Accept: line before it"},
		{"no threat table",
			strings.Replace(threatCriteria, "| adversary | negative test | accepted risk |", "| who | what | why |", 1),
			"carries no threat table"},
		{"missing adversary row",
			strings.Replace(threatCriteria, "| trusted_key_holder | SESS-wrong-role-key-refused | - |\n", "", 1),
			"has no row for Session adversary trusted_key_holder"},
		{"wrong negative test",
			strings.Replace(threatCriteria, "SESS-wrong-role-key-refused", "SESS-some-other-test", 1),
			"does not name its locked negative test SESS-wrong-role-key-refused"},
		{"accepted risk without its owner",
			strings.Replace(threatCriteria, "Product owner, 2026-10-05", "accepted", 1),
			"does not name the accepted risk's owner Product owner"},
		{"documented cell",
			strings.Replace(threatCriteria, "SESS-wrong-role-key-refused", "documented", 1),
			"\"documented\" is not a closure"},
		{"agreement-only cell",
			strings.Replace(threatCriteria, "SESS-wrong-role-key-refused", "output matches the reference implementation", 1),
			"only claims agreement"},
		{"no pass-wrongly line",
			strings.Replace(threatCriteria, "Pass-wrongly: a check that accepts any well-formed token passes every positive test.\n", "", 1),
			"has no Pass-wrongly: line"},
		{"empty pass-wrongly line",
			strings.Replace(threatCriteria, "Pass-wrongly: a check that accepts any well-formed token passes every positive test.", "Pass-wrongly:", 1),
			"empty Pass-wrongly: line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := threatFixture(t, threatPlan(tc.body), threatLedgerEnforce, nil)
			g := CheckBuildPlan(design)
			if !hasErr(g, tc.want) {
				t.Fatalf("want ERROR containing %q, got errs=%v notes=%v", tc.want, g.Errs, g.Notes)
			}
			if !hasErr(g, "BUILD.md: milestone M1 (Session slice)") {
				t.Errorf("the finding must name the milestone and its document: %v", g.Errs)
			}
		})
	}
}

func TestGbThreatAuditModeNotesOnly(t *testing.T) {
	design := threatFixture(t, threatPlan("Nothing else.\n"), threatLedgerAudit, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("audit mode blocks nothing: %v", g.Errs)
	}
	for _, want := range []string{"audit: BUILD.md: milestone M1 (Session slice)", "no paired acceptance criteria", "carries no threat table", "has no Pass-wrongly: line"} {
		if !hasNote(g, want) {
			t.Errorf("want audit note containing %q, got %v", want, g.Notes)
		}
	}
}

func TestGbThreatDocumentedCellBlocksInAuditMode(t *testing.T) {
	body := strings.Replace(threatCriteria, "SESS-wrong-role-key-refused", "documented", 1)
	design := threatFixture(t, threatPlan(body), threatLedgerAudit, nil)
	g := CheckBuildPlan(design)
	if !hasErr(g, "\"documented\" is not a closure") {
		t.Fatalf("a written documented closure is an ERROR in every mode: errs=%v notes=%v", g.Errs, g.Notes)
	}
}

func TestGbThreatNoLedgerUsesDetectedCandidates(t *testing.T) {
	design := threatFixture(t, threatPlan("Nothing else.\n"), "", nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("without a ledger Gb only audits: %v", g.Errs)
	}
	if !hasNote(g, "audit: BUILD.md: milestone M1 (Session slice)") || !hasNote(g, "carries no threat table") {
		t.Fatalf("a detected candidate makes the milestone security-relevant: %v", g.Notes)
	}
	// without a ledger any threat table with one row satisfies the structure
	design = threatFixture(t, threatPlan(threatCriteria), "", nil)
	g = CheckBuildPlan(design)
	if len(g.Errs) != 0 || len(g.Notes) != 0 {
		t.Fatalf("a structurally complete milestone is clean without a ledger: errs=%v notes=%v", g.Errs, g.Notes)
	}
	empty := strings.Replace(strings.Replace(strings.Replace(threatCriteria,
		"| outsider_without_key | SESS-forged-credential-refused | - |\n", "", 1),
		"| trusted_key_holder | SESS-wrong-role-key-refused | - |\n", "", 1),
		"| duplicate_replay_reorder | - | Product owner, 2026-10-05 |\n", "", 1)
	design = threatFixture(t, threatPlan(empty), "", nil)
	g = CheckBuildPlan(design)
	if !hasNote(g, "threat table has no rows") {
		t.Fatalf("an empty threat table is not a threat table: %v", g.Notes)
	}
}

func TestGbThreatNothingSecurityRelevantWithoutModel(t *testing.T) {
	design := writeBuildPlanFixture(t, threatPlan("Nothing else.\n"), nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 || len(g.Notes) != 0 || g.Counts["security-relevant milestones"] != 0 {
		t.Fatalf("no ledger and no model: nothing is security-relevant: errs=%v notes=%v counts=%v", g.Errs, g.Notes, g.Counts)
	}
}

func TestGbThreatUnparseableLedgerAddsNothing(t *testing.T) {
	design := threatFixture(t, threatPlan("Nothing else.\n"), "subjects: [unclosed\n", nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 || len(g.Notes) != 0 {
		t.Fatalf("Gz owns a broken ledger; Gb adds nothing: errs=%v notes=%v", g.Errs, g.Notes)
	}
}

func TestGbThreatClosedMilestoneIsHistory(t *testing.T) {
	design := threatFixture(t, threatPlan("Status: closed\n"), threatLedgerEnforce, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 || g.Counts["security-relevant milestones"] != 0 {
		t.Fatalf("a closed milestone is history: errs=%v counts=%v", g.Errs, g.Counts)
	}
	design = threatFixture(t, threatPlan("Status: open\n"), threatLedgerEnforce, nil)
	if g := CheckBuildPlan(design); !hasErr(g, "carries no threat table") {
		t.Fatalf("an explicitly open milestone is checked: %v", g.Errs)
	}
}

func TestGbThreatInvariantIDMakesMilestoneRelevant(t *testing.T) {
	plan := strings.Replace(threatPlan("Nothing else.\n"), "Login and resume for Session.", "Holds session-requires-valid-credential.", 1)
	plan = strings.Replace(plan, "Session slice", "Login slice", 1)
	design := threatFixture(t, plan, threatLedgerEnforce, nil)
	g := CheckBuildPlan(design)
	if !hasErr(g, "milestone M1 (Login slice)") || !hasErr(g, "has no row for Session adversary outsider_without_key") {
		t.Fatalf("naming a threat invariant names its subject: %v", g.Errs)
	}
}

func TestThreatSubjectTokenMatching(t *testing.T) {
	cases := []struct {
		subject, text string
		want          bool
	}{
		{"Session", "resume a Session.", true},
		{"Session", "the Session.login action", true},
		{"Session", "SessionStore", false},
		{"Session", "CheckingSession", false},
		{"Session", "session", false},
		{"Session", "Session-scoped", false},
		{"Session.login", "call Session.login.", true},
		{"Session.login", "call Session.login now", true},
		{"Session.login", "call Session.loginAll", false},
		{"Session.login", "call Session.login.retry", false},
		{"Session.login", "XSession.login", false},
		{"Session.login", "Session", false},
	}
	for _, tc := range cases {
		if got := subjectTokenIn(tc.subject, tc.text); got != tc.want {
			t.Errorf("subjectTokenIn(%q, %q) = %v, want %v", tc.subject, tc.text, got, tc.want)
		}
	}
}

func TestGbThreatPairwisePacket(t *testing.T) {
	plan := "# B\n\nMode: manifest\n\n## 9. Build plan\n\n" +
		"**M0 - Walking skeleton.**\nPacket: [M0](BUILD/M0-walking-skeleton.md)\n" +
		"Demo: one thread.\nNFR: error envelope.\nDoD: T-CMD-01 green.\n\n" +
		"**M1 - Login slice.**\nPacket: [M1](BUILD/M1-login.md)\n" +
		"Demo: log in.\nDoD: all rows green.\n"
	bare := executionPacket("1", "Login slice") + "\nResume a Session.\n"
	design := threatFixture(t, plan, threatLedgerEnforce, map[string]string{
		"BUILD/M0-walking-skeleton.md": executionPacket("0", "Walking skeleton"),
		"BUILD/M1-login.md":            bare,
	})
	g := CheckBuildPlan(design)
	if !hasErr(g, "M1-login.md: milestone M1 (Login slice)") || !hasErr(g, "carries no threat table") {
		t.Fatalf("the packet carries the threat rules in pairwise mode: %v", g.Errs)
	}
	design = threatFixture(t, plan, threatLedgerEnforce, map[string]string{
		"BUILD/M0-walking-skeleton.md": executionPacket("0", "Walking skeleton"),
		"BUILD/M1-login.md":            bare + "\n" + threatCriteria,
	})
	g = CheckBuildPlan(design)
	if len(g.Errs) != 0 || g.Counts["security-relevant milestones"] != 1 {
		t.Fatalf("a compliant packet is clean: errs=%v counts=%v", g.Errs, g.Counts)
	}
}

func TestGbThreatMatrixShard(t *testing.T) {
	design := threatFixture(t, matrixPlan(), threatLedgerEnforce, map[string]string{
		"BUILD/core.md":  matrixShard("core", "M0, M1"),
		"BUILD/trust.md": matrixShard("trust", "M0") + "\nThe Session lifecycle.\n",
		"BUILD/ops.md":   matrixShard("ops", "M1"),
	})
	g := CheckBuildPlan(design)
	if !hasErr(g, "trust.md: milestone M0 (Walking skeleton)") || !hasErr(g, "no paired acceptance criteria") {
		t.Fatalf("a security-relevant shard carries the threat rules: %v", g.Errs)
	}
	for _, e := range g.Errs {
		if strings.Contains(e, "core.md") || strings.Contains(e, "ops.md") {
			t.Errorf("a shard that names no subject owes nothing: %s", e)
		}
	}
	design = threatFixture(t, matrixPlan(), threatLedgerEnforce, map[string]string{
		"BUILD/core.md":  matrixShard("core", "M0, M1"),
		"BUILD/trust.md": matrixShard("trust", "M0") + "\nThe Session lifecycle.\n\n" + threatCriteria,
		"BUILD/ops.md":   matrixShard("ops", "M1"),
	})
	g = CheckBuildPlan(design)
	if len(g.Errs) != 0 || g.Counts["security-relevant milestones"] != 1 {
		t.Fatalf("a compliant shard is clean: errs=%v counts=%v", g.Errs, g.Counts)
	}
}

func TestGbThreatScansAreFenceMasked(t *testing.T) {
	body := "```text\nAccept: fenced\nRefuse: fenced\nPass-wrongly: fenced\n```\n"
	design := threatFixture(t, threatPlan(body), threatLedgerEnforce, nil)
	g := CheckBuildPlan(design)
	if !hasErr(g, "no paired acceptance criteria") || !hasErr(g, "has no Pass-wrongly: line") {
		t.Fatalf("fenced example lines are not criteria: %v", g.Errs)
	}
}
