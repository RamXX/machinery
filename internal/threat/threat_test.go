package threat

import (
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

func fullRows(indent string) string {
	var b strings.Builder
	for _, a := range Adversaries {
		b.WriteString(indent + "- adversary: " + a + "\n")
		b.WriteString(indent + "  invariant: inv-" + a + "\n")
		b.WriteString(indent + "  negative_test: T-" + a + "\n")
	}
	return b.String()
}

func validLedger() string {
	return "mode: enforce\nenforced_since: 2026-10-05\nsubjects:\n  - subject: Session\n    kinds: [authenticates]\n    adversaries:\n" +
		fullRows("      ")
}

func mustParse(t *testing.T, src string) (*Ledger, []string) {
	t.Helper()
	l, problems := Parse([]byte(src))
	if l == nil {
		t.Fatalf("Parse returned nil ledger; problems: %v", problems)
	}
	return l, problems
}

func wantProblem(t *testing.T, problems []string, fragment string) {
	t.Helper()
	for _, p := range problems {
		if strings.Contains(p, fragment) {
			return
		}
	}
	t.Fatalf("want a problem containing %q, got %v", fragment, problems)
}

func TestParseValidLedger(t *testing.T) {
	l, problems := mustParse(t, validLedger())
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if l.Mode != ModeEnforce || l.EnforcedSince != "2026-10-05" {
		t.Fatalf("mode/since = %q/%q", l.Mode, l.EnforcedSince)
	}
	if len(l.Subjects) != 1 || len(l.Subjects[0].Rows) != len(Adversaries) {
		t.Fatalf("subjects = %+v", l.Subjects)
	}
	if !l.Covers("Session") || !l.Covers("Session.login") || l.Covers("Sessions") {
		t.Fatal("coverage must be exact on the entity and extend to its actions")
	}
}

func TestParseRejectsMissingAndDuplicateAdversary(t *testing.T) {
	src := strings.Replace(validLedger(), "      - adversary: cross_bundle_substitution\n        invariant: inv-cross_bundle_substitution\n        negative_test: T-cross_bundle_substitution\n", "", 1)
	src = strings.Replace(src, "adversary: replay_never", "", 1)
	_, problems := mustParse(t, src)
	wantProblem(t, problems, "Session: adversary cross_bundle_substitution is not classified")

	dup := validLedger() + "      - adversary: outsider_without_key\n        not_applicable: twice\n"
	_, problems = mustParse(t, dup)
	wantProblem(t, problems, "outsider_without_key is classified twice")

	unknown := validLedger() + "      - adversary: martians\n        not_applicable: no\n"
	_, problems = mustParse(t, unknown)
	wantProblem(t, problems, "unknown adversary martians")
}

func TestParseRowNeedsExactlyOneDisposition(t *testing.T) {
	src := strings.Replace(validLedger(), "        negative_test: T-outsider_without_key\n", "        negative_test: T-outsider_without_key\n        not_applicable: also\n", 1)
	_, problems := mustParse(t, src)
	wantProblem(t, problems, "exactly one disposition")

	src = strings.Replace(validLedger(), "        negative_test: T-outsider_without_key\n", "", 1)
	_, problems = mustParse(t, src)
	wantProblem(t, problems, "needs both invariant and negative_test")
}

func TestParseAcceptedRiskMustBeVisibleAndSigned(t *testing.T) {
	src := strings.Replace(validLedger(),
		"        invariant: inv-duplicate_replay_reorder\n        negative_test: T-duplicate_replay_reorder\n",
		"        accepted_risk: {owner: Owner, date: 2026-10-05, reason: local only}\n", 1)
	_, problems := mustParse(t, src)
	wantProblem(t, problems, "accepted_risk needs visible_in")

	src = strings.Replace(validLedger(),
		"        invariant: inv-duplicate_replay_reorder\n        negative_test: T-duplicate_replay_reorder\n",
		"        accepted_risk: {date: 2026-10-05, reason: r, visible_in: v}\n", 1)
	_, problems = mustParse(t, src)
	wantProblem(t, problems, "accepted_risk needs owner")
}

func TestParseRejectsDocumentedClosureAndAgreementOnly(t *testing.T) {
	src := strings.Replace(validLedger(), "not_applicable_placeholder", "", 1)
	src = strings.Replace(src,
		"        invariant: inv-signed_field_injection\n        negative_test: T-signed_field_injection\n",
		"        not_applicable: documented\n", 1)
	_, problems := mustParse(t, src)
	wantProblem(t, problems, "\"documented\" is not a closure")

	src = strings.Replace(validLedger(), "negative_test: T-trusted_key_holder", "negative_test: matches the reference implementation", 1)
	_, problems = mustParse(t, src)
	wantProblem(t, problems, "agreement with a reference is not correctness")
}

func TestParseModeAndSince(t *testing.T) {
	_, problems := mustParse(t, "mode: strictest\n")
	wantProblem(t, problems, "mode must be audit or enforce")
	_, problems = mustParse(t, "mode: enforce\n")
	wantProblem(t, problems, "enforced_since")
	l, problems := mustParse(t, "mode: audit\n")
	if len(problems) != 0 || l.Mode != ModeAudit {
		t.Fatalf("audit ledger: %v %v", l.Mode, problems)
	}
	l, problems = mustParse(t, "subjects: []\n")
	if len(problems) != 0 || l.Mode != ModeAudit {
		t.Fatalf("absent mode defaults to audit: %v %v", l.Mode, problems)
	}
	_, problems = mustParse(t, "mode: audit\nsurprise: 1\n")
	wantProblem(t, problems, "unknown key")
}

func TestParseWaiversExemptionsAndVerifiers(t *testing.T) {
	src := validLedger() + `not_security_relevant:
  - subject: Activity
    reason: free text
  - subject: Session
    reason: clash
waivers:
  - subject: LegacyImport
    owner: Owner
    reason: later
verifiers:
  - component: Authz
    decides: visibility
  - component: ""
`
	l, problems := mustParse(t, src)
	wantProblem(t, problems, "Session is listed more than once")
	wantProblem(t, problems, "waiver for LegacyImport needs owner, date, and reason")
	wantProblem(t, problems, "verifiers[1] needs component and decides")
	if !l.Covers("Activity") {
		t.Fatal("an exemption covers its subject")
	}
}

func TestWordsAndSecurityDetection(t *testing.T) {
	if got := strings.Join(Words("passwordHash verify_token APIKey"), ","); got != "password,hash,verify,token,api,key" {
		t.Fatalf("Words = %s", got)
	}
	for _, w := range []string{"verify", "verifier", "authenticates", "login", "signature", "signed", "countersign", "hash", "digest", "approval", "attests", "admission", "authorize", "token", "secret", "certificate", "credential"} {
		if !SecurityWord(w) {
			t.Errorf("%s should be security-relevant", w)
		}
	}
	for _, w := range []string{"design", "assign", "consign", "designer", "admin", "order", "key", "status", "hashtag"} {
		if SecurityWord(w) {
			t.Errorf("%s should not be security-relevant", w)
		}
	}
}

const model = `
entities:
  User:
    definition: A person who logs in.
    attributes:
      - {name: passwordHash, type: string}
  Order:
    definition: A purchase.
    actions:
      - {name: approve, actor: Manager}
      - {name: ship, actor: System}
  Note:
    definition: Free text.
`

func loadModel(t *testing.T, src string) *ir.Object {
	t.Helper()
	v, err := ir.LoadYAML([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return v.AsObject()
}

func TestDetectCandidates(t *testing.T) {
	got := Detect(loadModel(t, model))
	var subjects []string
	for _, c := range got {
		subjects = append(subjects, c.Subject)
		if c.Evidence == "" || c.Hash == "" {
			t.Fatalf("candidate %+v lacks evidence or hash", c)
		}
	}
	if strings.Join(subjects, ",") != "Order.approve,User" {
		t.Fatalf("candidates = %v", subjects)
	}
}

func TestEntityHashTracksDefinitionChanges(t *testing.T) {
	a := EntityHash(loadModel(t, model), "Order")
	b := EntityHash(loadModel(t, strings.Replace(model, "A purchase.", "A purchase order.", 1)), "Order")
	c := EntityHash(loadModel(t, strings.Replace(model, "Free text.", "Other text.", 1)), "Order")
	if a == "" || a == b || a != c {
		t.Fatalf("hash must change with the entity and only with it: %s %s %s", a, b, c)
	}
}

func TestResolveAgainstModel(t *testing.T) {
	m := loadModel(t, model+`
invariants:
  - {id: inv-outsider_without_key, statement: s}
`)
	src := strings.Replace(validLedger(), "subject: Session", "subject: User", 1)
	src += "  - subject: Ghost\n    kinds: [verifies]\n    adversaries:\n" + fullRows("      ")
	src += "waivers:\n  - {subject: Order.cancel, owner: O, date: 2026-10-05, reason: r}\n"
	l, problems := mustParse(t, src)
	if len(problems) != 0 {
		t.Fatalf("parse problems: %v", problems)
	}
	res := l.Resolve(m)
	wantProblem(t, res, "subject Ghost is not an entity or Entity.action the model declares")
	wantProblem(t, res, "subject Order.cancel is not an entity or Entity.action the model declares")
	wantProblem(t, res, "User: adversary trusted_key_holder names invariant inv-trusted_key_holder, which the model does not declare")
	for _, p := range res {
		if strings.Contains(p, "inv-outsider_without_key") {
			t.Fatalf("declared invariant reported: %s", p)
		}
	}
	if ids := l.ThreatInvariants(); !ids["inv-outsider_without_key"] || len(ids) != len(Adversaries) {
		t.Fatalf("ThreatInvariants = %v", ids)
	}
}
