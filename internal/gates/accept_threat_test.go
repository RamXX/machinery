package gates

import (
	"strings"
	"testing"
)

// acceptThreatPlan closes M0 and a security-relevant M1 (it names Session).
const acceptThreatPlan = `# BUILD: thing

Mode: full

## 9. Build plan

**M0 - Walking skeleton.** One real transition through one real boundary.
DoD: T-CMD-01 and CMD-abc123 green.
Status: closed

**M1 - Session slice.** Login and resume for Session. DoD: T-CMD-02 green.
Status: closed
`

const acceptThreatReviewYAML = `threat_review:
  - adversary: outsider_without_key
    subject: Session
    probe: "forged credential file passed to the real CLI"
    expected_reason: "credential signature invalid"
    observed_reason: "credential signature invalid"
  - adversary: trusted_key_holder
    subject: Session
    probe: "credential signed by the audit key"
    expected_reason: "key not authorized for the session role"
    observed_reason: "key not authorized for the session role"
  - adversary: duplicate_replay_reorder
    subject: Session
    probe: "replayed session file"
    expected_reason: "replay-unchecked printed by session status"
    observed_reason: "replay-unchecked printed by session status"
reopened_limits: []
`

func acceptEvidenceM1(date, extra string) string {
	return "milestone: 1\ncommit: " + acceptedCommit + "\nverdict: ACCEPTED\ndod_ids:\n  - T-CMD-02\n" +
		"attestations:\n  - the threat table was probed first\nfindings: []\nreviewer: independent review\ndate: " + date + "\n" + extra
}

func acceptThreatFixture(t *testing.T, ledger, evidence string) string {
	t.Helper()
	files := map[string]string{
		"BUILD.md":             acceptThreatPlan,
		"domain.modelith.yaml": planThreatModel,
		"acceptance/M1.yaml":   evidence,
	}
	if ledger != "" {
		files["threats.yaml"] = ledger
	}
	return writeAcceptFixture(t, files)
}

func TestGaThreatReviewCompleteIsClean(t *testing.T) {
	design := acceptThreatFixture(t, threatLedgerEnforce, acceptEvidenceM1("2026-10-02", acceptThreatReviewYAML))
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 || len(g.Notes) != 0 {
		t.Fatalf("a complete threat-first review is clean: errs=%v notes=%v", g.Errs, g.Notes)
	}
	if g.Counts["threat reviews bound"] != 3 {
		t.Errorf("checked line must count the bound threat_review rows: %+v", g.Counts)
	}
}

func TestGaThreatReviewEnforceFindings(t *testing.T) {
	cases := []struct {
		name, review, want string
	}{
		{"missing row",
			strings.Replace(acceptThreatReviewYAML, "  - adversary: trusted_key_holder\n    subject: Session\n    probe: \"credential signed by the audit key\"\n    expected_reason: \"key not authorized for the session role\"\n    observed_reason: \"key not authorized for the session role\"\n", "", 1),
			"records no threat_review row for Session adversary trusted_key_holder"},
		{"no threat_review at all", "",
			"records no threat_review row for Session adversary outsider_without_key"},
		{"mismatched reasons",
			strings.Replace(acceptThreatReviewYAML, "observed_reason: \"credential signature invalid\"", "observed_reason: \"file not found\"", 1),
			"expected_reason \"credential signature invalid\" but observed_reason \"file not found\""},
		{"exit status only",
			strings.Replace(strings.Replace(acceptThreatReviewYAML, "expected_reason: \"credential signature invalid\"", "expected_reason: \"non-zero exit\"", 1),
				"observed_reason: \"credential signature invalid\"", "observed_reason: \"exit 1\"", 1),
			"a non-zero exit is not a reason"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := acceptThreatFixture(t, threatLedgerEnforce, acceptEvidenceM1("2026-10-02", tc.review))
			g := CheckAcceptance(design, acceptedCommit)
			if !hasErr(g, tc.want) {
				t.Fatalf("want ERROR containing %q, got errs=%v notes=%v", tc.want, g.Errs, g.Notes)
			}
		})
	}
}

func TestGaThreatReviewBeforeEnforcedSinceIsHistory(t *testing.T) {
	design := acceptThreatFixture(t, threatLedgerEnforce, acceptEvidenceM1("2026-09-30", ""))
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 || len(g.Notes) != 0 {
		t.Fatalf("an acceptance dated before enforced_since is history: errs=%v notes=%v", g.Errs, g.Notes)
	}
}

func TestGaThreatReviewAuditModes(t *testing.T) {
	for _, tc := range []struct{ name, ledger, want string }{
		{"audit ledger", threatLedgerAudit, "audit: acceptance/M1.yaml: records no threat_review row for Session adversary outsider_without_key"},
		{"no ledger", "", "audit: acceptance/M1.yaml: milestone M1 names security-relevant subject(s) Session but records no threat_review"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := acceptThreatFixture(t, tc.ledger, acceptEvidenceM1("2026-10-02", ""))
			g := CheckAcceptance(design, acceptedCommit)
			if len(g.Errs) != 0 {
				t.Fatalf("audit blocks nothing: %v", g.Errs)
			}
			if !hasNote(g, tc.want) {
				t.Fatalf("want note containing %q, got %v", tc.want, g.Notes)
			}
		})
	}
	// a recorded review satisfies the no-ledger audit
	design := acceptThreatFixture(t, "", acceptEvidenceM1("2026-10-02", acceptThreatReviewYAML))
	if g := CheckAcceptance(design, acceptedCommit); len(g.Errs) != 0 || len(g.Notes) != 0 {
		t.Fatalf("a recorded review satisfies the audit: errs=%v notes=%v", g.Errs, g.Notes)
	}
}

func TestGaThreatReviewMalformedIsAlwaysAnError(t *testing.T) {
	cases := []struct {
		name, extra, want string
	}{
		{"unknown adversary",
			"threat_review:\n  - adversary: insider\n    subject: Session\n    probe: p\n    expected_reason: r\n    observed_reason: r\n",
			"threat_review[0] adversary \"insider\" is not a standard adversary"},
		{"empty field",
			"threat_review:\n  - adversary: outsider_without_key\n    subject: Session\n    probe: \"\"\n    expected_reason: r\n    observed_reason: r\n",
			"threat_review[0] needs a non-empty probe"},
		{"missing field",
			"threat_review:\n  - adversary: outsider_without_key\n    subject: Session\n    expected_reason: r\n    observed_reason: r\n",
			"threat_review[0] needs a non-empty probe"},
		{"extra key",
			"threat_review:\n  - adversary: outsider_without_key\n    subject: Session\n    probe: p\n    expected_reason: r\n    observed_reason: r\n    exit: 1\n",
			"threat_review[0] has unknown key \"exit\""},
		{"not a list", "threat_review: yes\n", "threat_review must be a list of mappings"},
		{"entry not a mapping", "threat_review:\n  - outsider_without_key\n", "threat_review[0] is not a mapping"},
		{"reopened_limits not a list", "reopened_limits: widened\n", "reopened_limits must be a list of strings"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// no ledger and no model: the shape alone is held
			design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": acceptEvidenceM0 + tc.extra})
			g := CheckAcceptance(design, acceptedCommit)
			if !hasErr(g, tc.want) {
				t.Fatalf("want ERROR containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
	design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": acceptEvidenceM0 + "reopened_limits:\n  - the offline cache limit widens\n"})
	if g := CheckAcceptance(design, acceptedCommit); len(g.Errs) != 0 {
		t.Fatalf("reopened_limits is a known key: %v", g.Errs)
	}
}

func TestExitStatusOnlyReason(t *testing.T) {
	for _, s := range []string{"exit 1", "non-zero exit", "exit code", "failed", "Exited with status 2.", "nonzero exit code 1", "error"} {
		if !exitStatusOnly(s) {
			t.Errorf("%q is only an exit status", s)
		}
	}
	for _, s := range []string{"credential signature invalid", "error: key not authorized", "failed: digest mismatch on manifest"} {
		if exitStatusOnly(s) {
			t.Errorf("%q names a reason", s)
		}
	}
}
