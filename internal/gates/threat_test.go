package gates

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/threat"
)

// --- Gz-threat fixtures ---

// threatModel has two detected candidates: Session (its definition speaks of
// authentication) and Note.verifyAuthor (an action name with a verifier
// stem). Activity carries no security vocabulary.
const threatModel = `kind: modelith
version: 1
entities:
  Session:
    definition: An authenticated user session.
    actions:
      - name: open
    invariants:
      - id: session-requires-valid-credential
  Note:
    definition: A free-text remark.
    actions:
      - name: verifyAuthor
        description: checks who wrote the remark
  Activity:
    definition: A log entry.
`

// sessionSubject classifies Session completely: one tested row, the rest
// not applicable.
func sessionSubject() string {
	var b strings.Builder
	b.WriteString("  - subject: Session\n    kinds: [authenticates]\n    adversaries:\n")
	for _, a := range threat.Adversaries {
		if a == "outsider_without_key" {
			b.WriteString("      - adversary: outsider_without_key\n        invariant: session-requires-valid-credential\n        negative_test: SESS-forged-refused\n")
			continue
		}
		b.WriteString("      - adversary: " + a + "\n        not_applicable: \"no surface for this adversary on a session\"\n")
	}
	return b.String()
}

func enforceLedger(extra string) string {
	return "mode: enforce\nenforced_since: 2026-10-05\nsubjects:\n" + sessionSubject() + extra
}

func writeThreatDesign(t *testing.T, files map[string]string) string {
	t.Helper()
	design := t.TempDir()
	if _, ok := files["domain.modelith.yaml"]; !ok {
		files["domain.modelith.yaml"] = threatModel
	}
	for name, content := range files {
		mustWrite(t, filepath.Join(design, filepath.FromSlash(name)), content)
	}
	return design
}

func noteCount(g *Gate, needle string) int { return len(notesWith(g, needle)) }

// --- A1: classification, audit and enforce ---

func TestGzWithoutLedgerAuditsAsNotes(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{})
	g := CheckThreats(design, "")
	if g.Title != "Gz-threat  adversary classification" {
		t.Fatalf("title: %q", g.Title)
	}
	if len(g.Errs) != 0 || len(g.Warns) != 0 {
		t.Fatalf("audit mode blocks nothing: errs %v warns %v", g.Errs, g.Warns)
	}
	for _, want := range []string{
		"audit: Session looks security-relevant (\"authenticated\" in definition)",
		"audit: Note.verifyAuthor looks security-relevant (\"verify\" in action name)",
		"audit: set mode: enforce and enforced_since in design/threats.yaml to make these blocking; see docs/threat-driven-verification.md",
	} {
		if !hasNote(g, want) {
			t.Fatalf("missing note %q: %v", want, g.Notes)
		}
	}
	if hasNote(g, "Activity") {
		t.Fatalf("a subject without security vocabulary is no candidate: %v", g.Notes)
	}
	for _, n := range g.Notes {
		if !strings.HasPrefix(n, "audit: ") {
			t.Fatalf("every audit-mode finding is an audit: note: %q", n)
		}
	}
	if g.Counts["candidates detected"] != 2 {
		t.Fatalf("counts: %v", g.Counts)
	}
}

func TestGzEnforceUnclassifiedCandidateIsError(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": enforceLedger("")})
	g := CheckThreats(design, "")
	if !hasErr(g, "Note.verifyAuthor looks security-relevant (\"verify\" in action name) and is not classified in design/threats.yaml") {
		t.Fatalf("an unclassified candidate must error in enforce mode: %v", g.Errs)
	}
	if hasErr(g, "Session looks") {
		t.Fatalf("a classified subject must not error: %v", g.Errs)
	}
	if hasNote(g, "audit:") {
		t.Fatalf("enforce mode prints no audit notes: %v", g.Notes)
	}
	if g.Counts["subjects classified"] != 1 || g.Counts["adversary rows"] != len(threat.Adversaries) {
		t.Fatalf("counts: %v", g.Counts)
	}
}

func TestGzEnforceExemptionAndWaiverCover(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": enforceLedger(
		"not_security_relevant:\n  - subject: Note\n    reason: a remark gates nothing\n")})
	if g := CheckThreats(design, ""); len(g.Errs) != 0 {
		t.Fatalf("an entity-level exemption covers its actions: %v", g.Errs)
	}
	design = writeThreatDesign(t, map[string]string{"threats.yaml": enforceLedger(
		"waivers:\n  - subject: Note.verifyAuthor\n    owner: Product owner\n    date: 2026-10-05\n    reason: classification scheduled\n")})
	if g := CheckThreats(design, ""); len(g.Errs) != 0 {
		t.Fatalf("an owner-signed waiver covers the subject: %v", g.Errs)
	}
}

func TestGzAuditLedgerKeepsMissingClassificationsAsNotes(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": "mode: audit\nsubjects:\n" + sessionSubject()})
	g := CheckThreats(design, "")
	if len(g.Errs) != 0 {
		t.Fatalf("audit ledger blocks nothing it validated: %v", g.Errs)
	}
	if !hasNote(g, "audit: Note.verifyAuthor looks security-relevant") || hasNote(g, "audit: Session") {
		t.Fatalf("audit ledger notes only the unclassified: %v", g.Notes)
	}
}

// --- a malformed ledger is never audit-softened ---

func TestGzMalformedLedgerErrorsInAuditMode(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": "mode: audit\nsubjects:\n" +
		strings.Replace(sessionSubject(), "adversary: trusted_key_holder", "adversary: trusted_keyholder", 1)})
	g := CheckThreats(design, "")
	if !hasErr(g, "unknown adversary trusted_keyholder") {
		t.Fatalf("a parse problem is an error in audit mode: %v", g.Errs)
	}
	design = writeThreatDesign(t, map[string]string{"threats.yaml": "mode: audit\nnot_security_relevant:\n  - subject: Ghost\n    reason: gone\n"})
	g = CheckThreats(design, "")
	if !hasErr(g, "subject Ghost is not an entity or Entity.action the model declares") {
		t.Fatalf("a resolve problem is an error in audit mode: %v", g.Errs)
	}
	design = writeThreatDesign(t, map[string]string{"threats.yaml": ": not yaml ["})
	if g := CheckThreats(design, ""); !hasErr(g, "threats.yaml: invalid YAML") {
		t.Fatalf("invalid YAML is an error: %v", g.Errs)
	}
}

// --- A7: a threat invariant is never closed by a waiver note ---

func TestGzWaiverAnnexCannotCloseThreatInvariant(t *testing.T) {
	annex := "waivers:\n  - invariant: session-requires-valid-credential\n    reason: enforced somewhere\n"
	for _, mode := range []string{"audit", "enforce"} {
		ledger := "mode: " + mode + "\nenforced_since: 2026-10-05\nsubjects:\n" + sessionSubject() +
			"not_security_relevant:\n  - subject: Note\n    reason: a remark gates nothing\n"
		design := writeThreatDesign(t, map[string]string{"threats.yaml": ledger, "formal/waivers.yaml": annex})
		g := CheckThreats(design, "")
		if !hasErr(g, "formal/waivers.yaml waives threat invariant 'session-requires-valid-credential' (Session, outsider_without_key): a threat invariant cannot be closed by a waiver note; fix it or record an owner-signed accepted_risk in threats.yaml") {
			t.Fatalf("%s: a waived threat invariant must error: %v", mode, g.Errs)
		}
	}
	design := writeThreatDesign(t, map[string]string{"formal/waivers.yaml": annex})
	if g := CheckThreats(design, ""); len(g.Errs) != 0 {
		t.Fatalf("without a ledger there are no threat invariants: %v", g.Errs)
	}
}

// --- A8: verifiers are governed C4 elements ---

const threatDSL = `workspace {
  model {
    sys = softwareSystem "Sys" {
      app = container "App" "the application" "Go" {
        authz = component "Authz" "decides record visibility" "Go"
        report = component "Report" "renders summaries" "Go"
      }
    }
  }
}
`

func threatArch(boundaries string) string {
	return "# Architecture\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" + boundaries +
		"dependency_rules:\n  allow: []\n  deny: []\n```\n"
}

func verifierLedger() string {
	return enforceLedger("not_security_relevant:\n  - subject: Note\n    reason: a remark gates nothing\n" +
		"verifiers:\n  - component: authz\n    decides: record visibility\n")
}

func TestGzVerifierMustBeDslElement(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": verifierLedger(),
		"workspace.dsl": strings.Replace(threatDSL, "authz = component", "access = component", 1)})
	g := CheckThreats(design, "")
	if !hasErr(g, "verifier authz is not an element workspace.dsl declares") {
		t.Fatalf("a verifier missing from the dsl must error: %v", g.Errs)
	}
	design = writeThreatDesign(t, map[string]string{"threats.yaml": verifierLedger()})
	if g := CheckThreats(design, ""); !hasErr(g, "verifier authz is not an element workspace.dsl declares") {
		t.Fatalf("a verifier with no workspace.dsl must error: %v", g.Errs)
	}
}

func TestGzVerifierMustBeGovernedByContract(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": verifierLedger(), "workspace.dsl": threatDSL,
		"ARCHITECTURE.md": threatArch("  - id: app.report\n    element: report\n    code: [\"report/**\"]\n")})
	g := CheckThreats(design, "")
	if !hasErr(g, "verifier authz is not governed by the Architecture Contract") {
		t.Fatalf("an unbound verifier must error: %v", g.Errs)
	}
	design = writeThreatDesign(t, map[string]string{"threats.yaml": verifierLedger(), "workspace.dsl": threatDSL})
	if g := CheckThreats(design, ""); !hasErr(g, "verifier authz is not governed by the Architecture Contract") {
		t.Fatalf("a verifier with no contract must error: %v", g.Errs)
	}
	design = writeThreatDesign(t, map[string]string{"threats.yaml": verifierLedger(), "workspace.dsl": threatDSL,
		"ARCHITECTURE.md": threatArch("  - id: app.authz\n    element: authz\n    code: [\"authz/**\"]\n")})
	g = CheckThreats(design, "")
	if len(g.Errs) != 0 {
		t.Fatalf("a bound verifier is governed: %v", g.Errs)
	}
	if g.Counts["verifiers"] != 1 {
		t.Fatalf("counts: %v", g.Counts)
	}
}

func TestGzUnlistedVerifierLookingElement(t *testing.T) {
	dsl := strings.Replace(threatDSL, `report = component "Report" "renders summaries" "Go"`,
		`report = component "Report" "renders summaries" "Go"
        stamp = component "Stamp" "signing receipts before export" "Go"`, 1)
	arch := threatArch("  - id: app.authz\n    element: authz\n    code: [\"authz/**\"]\n")

	// audit: a note naming the element and its evidence
	design := writeThreatDesign(t, map[string]string{"workspace.dsl": dsl, "ARCHITECTURE.md": arch})
	g := CheckThreats(design, "")
	if len(g.Errs) != 0 || !hasNote(g, "audit: workspace.dsl element stamp speaks of \"signing\"") {
		t.Fatalf("an unlisted verifier-looking element is an audit note: errs %v notes %v", g.Errs, g.Notes)
	}
	if !hasNote(g, "audit: workspace.dsl element authz speaks of \"authz\"") {
		t.Fatalf("an undeclared authz element is a finding too: %v", g.Notes)
	}

	// enforce: an error; declaring it a verifier or exempting it by element
	// name clears it, and the element-name exemption is no resolve error
	design = writeThreatDesign(t, map[string]string{"workspace.dsl": dsl, "ARCHITECTURE.md": arch, "threats.yaml": verifierLedger()})
	g = CheckThreats(design, "")
	if !hasErr(g, "workspace.dsl element stamp speaks of \"signing\" in its name or description and is neither a declared verifier nor listed under not_security_relevant") {
		t.Fatalf("enforce mode makes it an error: %v", g.Errs)
	}
	if hasErr(g, "element authz") {
		t.Fatalf("a declared verifier is no finding: %v", g.Errs)
	}
	ledger := verifierLedger() + "  - component: stamp\n    decides: export signatures\n"
	ledger = strings.Replace(ledger, "not_security_relevant:\n", "not_security_relevant:\n  - subject: stamp\n    reason: a cosmetic stamp, not a signature\n", 1)
	ledger = strings.Replace(ledger, "  - component: stamp\n    decides: export signatures\n", "", 1)
	design = writeThreatDesign(t, map[string]string{"workspace.dsl": dsl, "ARCHITECTURE.md": arch, "threats.yaml": ledger})
	if g = CheckThreats(design, ""); len(g.Errs) != 0 {
		t.Fatalf("an element exempted by name is clean: %v", g.Errs)
	}
}

// --- --impl: every locked negative test exists ---

func TestGzImplNegativeTestToken(t *testing.T) {
	ledger := enforceLedger("not_security_relevant:\n  - subject: Note\n    reason: a remark gates nothing\n")
	design := writeThreatDesign(t, map[string]string{"threats.yaml": ledger})
	impl := t.TempDir()
	mustWrite(t, filepath.Join(impl, "go.mod"), "module example.com/m\n")
	mustWrite(t, filepath.Join(impl, "session_test.go"), "package m\n\nimport \"testing\"\n\nfunc TestOther(t *testing.T) {\n\tt.Log(\"SESS-forged-refused-later\")\n}\n")
	g := CheckThreats(design, impl)
	if !hasErr(g, "Session outsider_without_key: negative_test 'SESS-forged-refused' appears in no test file under the implementation") {
		t.Fatalf("a missing negative test must error in enforce mode: %v", g.Errs)
	}
	audit := writeThreatDesign(t, map[string]string{"threats.yaml": strings.Replace(ledger, "mode: enforce", "mode: audit", 1)})
	if g := CheckThreats(audit, impl); len(g.Errs) != 0 || !hasNote(g, "audit: Session outsider_without_key: negative_test 'SESS-forged-refused' appears in no test file") {
		t.Fatalf("a missing negative test is a note in audit mode: errs %v notes %v", g.Errs, g.Notes)
	}
	mustWrite(t, filepath.Join(impl, "session_test.go"), "package m\n\nimport \"testing\"\n\nfunc TestForged(t *testing.T) {\n\tt.Log(\"SESS-forged-refused\")\n}\n")
	g = CheckThreats(design, impl)
	if len(g.Errs) != 0 || g.Counts["negative tests found"] != 1 {
		t.Fatalf("a present negative test is clean: errs %v counts %v", g.Errs, g.Counts)
	}
	// a test file path is a valid token too
	design = writeThreatDesign(t, map[string]string{"threats.yaml": strings.Replace(ledger, "SESS-forged-refused", "session_test.go", 1)})
	if g := CheckThreats(design, impl); len(g.Errs) != 0 {
		t.Fatalf("a negative test named by its file path is found: %v", g.Errs)
	}
}

// --- the migration ratchet ---

func recordThreatDebt(t *testing.T, design string, grow bool) DebtRecord {
	t.Helper()
	r, err := LoadRatchet(design)
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		r = &Ratchet{Date: "2026-10-05"}
	}
	rec, err := RecordThreatDebt(design, r, grow)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteRatchet(design, r); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestGzRatchetBaselinesRecordedCandidates(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": "mode: enforce\nenforced_since: 2026-10-05\n"})
	if g := CheckThreats(design, ""); len(g.Errs) != 2 {
		t.Fatalf("both candidates block before the baseline: %v", g.Errs)
	}
	rec := recordThreatDebt(t, design, false)
	if rec.Gate != "Gz-threat" || rec.Observed != 2 || rec.Recorded != 2 || !rec.First {
		t.Fatalf("record: %+v", rec)
	}
	r, _ := LoadRatchet(design)
	if len(r.Threats) != 2 || r.Threats[0].Subject != "Note.verifyAuthor" || !strings.HasPrefix(r.Threats[0].Hash, "sha256:") {
		t.Fatalf("threat_debt: %+v", r.Threats)
	}
	g := CheckThreats(design, "")
	if len(g.Errs) != 0 || noteCount(g, "baselined: ") != 2 || g.Counts[BaselinedCount] != 2 {
		t.Fatalf("recorded candidates are baselined notes: errs %v notes %v counts %v", g.Errs, g.Notes, g.Counts)
	}
	if !hasNote(g, "baselined: Session looks security-relevant") {
		t.Fatalf("the baselined note carries the finding: %v", g.Notes)
	}
}

func TestGzRatchetChangedHashAndNewSubjectBlock(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": "mode: enforce\nenforced_since: 2026-10-05\n"})
	recordThreatDebt(t, design, false)
	changed := strings.Replace(threatModel, "A free-text remark.", "A free-text remark, edited.", 1) +
		"  Token:\n    definition: An API token.\n"
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), changed)
	g := CheckThreats(design, "")
	if !hasErr(g, "Note.verifyAuthor looks security-relevant") || !hasErr(g, "its model definition changed since the baseline") {
		t.Fatalf("a changed definition re-arms the candidate: %v", g.Errs)
	}
	if !hasErr(g, "Token looks security-relevant") {
		t.Fatalf("a new candidate blocks: %v", g.Errs)
	}
	if hasErr(g, "Session looks") || !hasNote(g, "baselined: Session") {
		t.Fatalf("an unchanged recorded candidate stays baselined: errs %v notes %v", g.Errs, g.Notes)
	}
	// a later baseline shrinks: the changed entry is not re-recorded without --grow
	rec := recordThreatDebt(t, design, false)
	if rec.Recorded != 1 || rec.NotRecorded != 2 || rec.Dropped != 1 {
		t.Fatalf("shrink-only record: %+v", rec)
	}
	if rec = recordThreatDebt(t, design, true); rec.Recorded != 3 {
		t.Fatalf("--grow records every current candidate: %+v", rec)
	}
}

func TestGzRatchetResolvedEntryIsNote(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": "mode: enforce\nenforced_since: 2026-10-05\n"})
	recordThreatDebt(t, design, false)
	mustWrite(t, filepath.Join(design, "threats.yaml"), enforceLedger(""))
	g := CheckThreats(design, "")
	if len(g.Errs) != 0 || !hasNote(g, "baselined threat candidate Session resolved; run machinery baseline to shrink the ratchet") {
		t.Fatalf("a classified recorded subject is resolved: errs %v notes %v", g.Errs, g.Notes)
	}
	if g.Counts[resolvedCount] != 1 || g.Counts[BaselinedCount] != 1 {
		t.Fatalf("counts: %v", g.Counts)
	}
}

func TestGzBaselineRefusesMalformedLedger(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": "mode: enforce\n"})
	if _, err := RecordThreatDebt(design, &Ratchet{Date: "2026-10-05"}, false); err == nil || !strings.Contains(err.Error(), "enforced_since") {
		t.Fatalf("baseline must refuse a malformed ledger: %v", err)
	}
	design = writeThreatDesign(t, map[string]string{"threats.yaml": "not_security_relevant:\n  - subject: Ghost\n    reason: gone\n"})
	if _, err := RecordThreatDebt(design, &Ratchet{Date: "2026-10-05"}, false); err == nil || !strings.Contains(err.Error(), "Ghost") {
		t.Fatalf("baseline must refuse a ledger that does not resolve: %v", err)
	}
}

func TestRatchetThreatDebtDecodesStrictly(t *testing.T) {
	for _, body := range []string{
		`{"date": "2026-10-05", "threat_debt": [{"subject": "A"}]}`,
		`{"date": "2026-10-05", "threat_debt": [{"subject": "A", "hash": "sha256:1", "x": 1}]}`,
		`{"date": "2026-10-05", "threat_debt": [{"subject": "A", "hash": "sha256:1"}, {"subject": "A", "hash": "sha256:2"}]}`,
		`{"date": "2026-10-05", "threat_debt": {}}`,
	} {
		if _, err := decodeRatchet([]byte(body)); err == nil {
			t.Fatalf("must reject %s", body)
		}
	}
	r, err := decodeRatchet([]byte(`{"date": "2026-10-05", "threat_debt": []}`))
	if err != nil || r.Threats == nil {
		t.Fatalf("a present, empty section decodes as recorded: %v %+v", err, r)
	}
	out, _ := RenderRatchet(&Ratchet{Date: "2026-10-05", Edges: map[string][]string{}})
	if strings.Contains(string(out), "threat_debt") {
		t.Fatalf("an absent section is omitted:\n%s", out)
	}
}

func TestFinalHandoffRefusesBaselinedThreatDebt(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{"threats.yaml": "mode: enforce\nenforced_since: 2026-10-05\n"})
	recordThreatDebt(t, design, false)
	sel := Selection{Run: map[string]bool{"gz": true}, Explicit: true}
	var final *Gate
	for _, g := range RunSelected(design, "", sel, RunOptions{Complete: true}) {
		if strings.HasPrefix(g.Title, "G!-complete") {
			final = g
		}
	}
	if final == nil {
		t.Fatal("--complete must run the final-handoff gate")
	}
	if !hasErr(final, "2 baselined finding(s) remain (Gz-threat 2)") {
		t.Fatalf("final handoff must refuse baselined threat debt: %v", final.Errs)
	}
}

// --- selection ---

func TestGzRunsByDefaultWithAModel(t *testing.T) {
	design := writeThreatDesign(t, map[string]string{})
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gz"] || !KnownGate("gz") {
		t.Fatalf("gz is in the default list: %v", sel.Run)
	}
	var titles []string
	for _, g := range RunSelected(design, "", sel, RunOptions{}) {
		titles = append(titles, strings.Fields(g.Title)[0])
	}
	joined := strings.Join(titles, ",")
	if !strings.Contains(joined, "Gc-carrier,Gz-threat,") {
		t.Fatalf("Gz runs right after Gc: %s", joined)
	}
}
