package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const attestArch = `# Architecture

## Architecture Contract

Nothing to see; Gv never reads the content, only the bytes.
`

// attestFixture builds a design carrying an ARCHITECTURE.md (so the Gate 2
// claims are owed) plus whatever evidence the case supplies. The returned
// hash is ARCHITECTURE.md's, so a case can bind correctly or deliberately
// not.
func attestFixture(t *testing.T, evidence string) (string, string) {
	t.Helper()
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), attestArch)
	hash, err := ContentHash(filepath.Join(design, "ARCHITECTURE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if evidence != "" {
		mustWrite(t, filepath.Join(design, AttestationsFileName), strings.ReplaceAll(evidence, "<ARCH_HASH>", hash))
	}
	return design, hash
}

// attestEvidence renders a full evidence file over the six Gate 2 claims plus
// one row per extra line the case adds, so a clean fixture is clean on
// coverage too (no warnings) and a mutation case changes exactly one thing.
func attestEvidence(rows ...string) string {
	b := strings.Builder{}
	b.WriteString("attestation_version: 1\nattestations:\n")
	for _, r := range rows {
		b.WriteString(r)
	}
	return b.String()
}

// attestRowFor renders one well-formed row over ARCHITECTURE.md.
func attestRowFor(claim string) string {
	return "  - claim: " + claim + "\n" +
		"    attestor: Ramiro Salas\n" +
		"    date: 2026-08-30\n" +
		"    covers:\n" +
		"      - {path: ARCHITECTURE.md, hash: <ARCH_HASH>}\n"
}

// attestAllG2 is every claim ARCHITECTURE.md makes owed, so the clean case
// carries no coverage warning.
func attestAllG2() []string {
	var rows []string
	for _, id := range AttestationClaimIDs() {
		if strings.HasPrefix(id, "g2.") {
			rows = append(rows, attestRowFor(id))
		}
	}
	return rows
}

func TestAttestationClean(t *testing.T) {
	design, _ := attestFixture(t, attestEvidence(attestAllG2()...))
	if !AttestationActive(design) {
		t.Fatal("AttestationActive must be true once the evidence file exists")
	}
	g := CheckAttestations(design)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("clean evidence must pass: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if len(g.Warns) != 0 {
		t.Fatalf("evidence covering every owed claim must not warn: %v", g.Warns)
	}
	if g.Counts["attested claims"] != 6 {
		t.Fatalf("attested claims = %d, want 6: %+v", g.Counts["attested claims"], g.Counts)
	}
	if g.Counts["covered artifacts current"] != 6 {
		t.Fatalf("covered artifacts current = %d, want 6: %+v", g.Counts["covered artifacts current"], g.Counts)
	}
	if g.Counts["claims owed"] != 6 {
		t.Fatalf("claims owed = %d, want 6 (the Gate 2 block): %+v", g.Counts["claims owed"], g.Counts)
	}
}

// The gate is inactive without its artifact, exactly like Ga and Gj: absence
// of the record is not adoption of the record.
func TestAttestationInactiveWithoutFile(t *testing.T) {
	design, _ := attestFixture(t, "")
	if AttestationActive(design) {
		t.Fatal("AttestationActive must be false without the evidence file")
	}
	g := CheckAttestations(design)
	if !hasErr(g, "no "+AttestationsFileName) {
		t.Fatalf("an explicitly requested gate with no artifact must fail: %v", g.Errs)
	}
}

func TestAttestationMutations(t *testing.T) {
	good := attestRowFor("g2.nfr-content")
	cases := []struct {
		name, evidence, want string
	}{
		{
			"unknown claim id",
			attestEvidence(attestRowFor("g2.invented-claim")),
			"is not in the attested-claim vocabulary",
		},
		{
			"missing attestor",
			attestEvidence("  - claim: g2.nfr-content\n    date: 2026-08-30\n    covers:\n      - {path: ARCHITECTURE.md, hash: <ARCH_HASH>}\n"),
			"attestor is required",
		},
		{
			"empty attestor",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: \"  \"\n    date: 2026-08-30\n    covers:\n      - {path: ARCHITECTURE.md, hash: <ARCH_HASH>}\n"),
			"attestor is required",
		},
		{
			"missing covered path",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-08-30\n    covers:\n      - {path: NOPE.md, hash: <ARCH_HASH>}\n"),
			"which the design does not carry",
		},
		{
			"hash mismatch is STALE",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-08-30\n    covers:\n      - {path: ARCHITECTURE.md, hash: sha256:" + strings.Repeat("0", 64) + "}\n"),
			"is STALE: ARCHITECTURE.md changed since it was attested",
		},
		{
			"duplicate claim",
			attestEvidence(good, good),
			"attests g2.nfr-content twice",
		},
		{
			"no covers",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-08-30\n"),
			"covers must be a non-empty list",
		},
		{
			"bad date",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-13-01\n    covers:\n      - {path: ARCHITECTURE.md, hash: <ARCH_HASH>}\n"),
			"is not a real YYYY-MM-DD date",
		},
		{
			"malformed hash",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-08-30\n    covers:\n      - {path: ARCHITECTURE.md, hash: deadbeef}\n"),
			"is not sha256:<64 lower-case hex>",
		},
		{
			"absolute covered path",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-08-30\n    covers:\n      - {path: /etc/hosts, hash: <ARCH_HASH>}\n"),
			"is absolute",
		},
		{
			"escaping covered path",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-08-30\n    covers:\n      - {path: ../elsewhere.md, hash: <ARCH_HASH>}\n"),
			"escapes the design directory",
		},
		{
			"unknown root key",
			attestEvidence(good) + "bogus: true\n",
			"unsupported key",
		},
		{
			"unknown row key",
			attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-08-30\n    verdict: ACCEPTED\n    covers:\n      - {path: ARCHITECTURE.md, hash: <ARCH_HASH>}\n"),
			"unsupported key",
		},
		{
			"wrong version",
			"attestation_version: 2\nattestations: []\n",
			"attestation_version must be the integer 1",
		},
		{
			"empty list",
			"attestation_version: 1\nattestations: []\n",
			"an empty record is a failure, not a pass",
		},
		{
			"not a mapping",
			"- just a list\n",
			"not a yaml mapping",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design, _ := attestFixture(t, tc.evidence)
			g := CheckAttestations(design)
			if !hasErr(g, tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

// A covered artifact edited after the attestation is the case the gate exists
// for: the record still parses, still names a real claim and a real attestor,
// and is worthless. The finding must say so and must block.
func TestAttestationGoesStaleWhenTheArtifactMoves(t *testing.T) {
	design, _ := attestFixture(t, attestEvidence(attestRowFor("g2.nfr-content")))
	if g := CheckAttestations(design); len(g.Errs) != 0 {
		t.Fatalf("the fixture must start clean: %v", g.Errs)
	}
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), attestArch+"\nOne more sentence, and the judgment is over different bytes.\n")
	g := CheckAttestations(design)
	if !hasErr(g, "is STALE") {
		t.Fatalf("an edited covered artifact must go STALE: %v", g.Errs)
	}
	if !hasErr(g, "machinery attest ") {
		t.Fatalf("the STALE finding must name the helper that recomputes the hash: %v", g.Errs)
	}
	if g.Counts["covered artifacts current"] != 0 {
		t.Fatalf("a stale row must not count as current: %+v", g.Counts)
	}
}

// A directory cannot carry a content hash; naming one is a schema mistake
// with a clear remedy, not a mismatch.
func TestAttestationRejectsDirectoryReferent(t *testing.T) {
	design, _ := attestFixture(t, attestEvidence("  - claim: g2.nfr-content\n    attestor: R\n    date: 2026-08-30\n    covers:\n      - {path: machines, hash: sha256:"+strings.Repeat("a", 64)+"}\n"))
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	g := CheckAttestations(design)
	if !hasErr(g, "which is a directory") {
		t.Fatalf("a directory referent must be its own finding: %v", g.Errs)
	}
}

// Coverage is a WARN: a design adopting the record mid-flight must not be
// failed for the claims it has not re-judged yet. The warning still has to
// name the claim and the artifact that makes it owed.
func TestAttestationCoverageWarnsRatherThanBlocks(t *testing.T) {
	design, _ := attestFixture(t, attestEvidence(attestRowFor("g2.nfr-content")))
	g := CheckAttestations(design)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("partial coverage must not block: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if len(g.Warns) != 5 {
		t.Fatalf("want one warn per unattested owed Gate 2 claim (5), got %d: %v", len(g.Warns), g.Warns)
	}
	joined := strings.Join(g.Warns, "\n")
	for _, want := range []string{"g2.interface-contract-rightness", "g2.placement-rightness", "ARCHITECTURE.md"} {
		if !strings.Contains(joined, want) {
			t.Errorf("coverage warning must name %q: %v", want, g.Warns)
		}
	}
	if strings.Contains(joined, "g2.nfr-content") {
		t.Errorf("the attested claim must not warn: %v", g.Warns)
	}
}

// Claims are owed by the artifacts that exist, never by the vocabulary as a
// whole: a design with no machines owes no guard-semantics attestation.
func TestAttestationOwesOnlyWhatTheDesignReached(t *testing.T) {
	design, _ := attestFixture(t, attestEvidence(attestAllG2()...))
	if g := CheckAttestations(design); g.Counts["claims owed"] != 6 {
		t.Fatalf("a design with only ARCHITECTURE.md owes exactly the Gate 2 claims: %+v", g.Counts)
	}
	mustWrite(t, filepath.Join(design, "machines", "Deal.machine.json"), "{}\n")
	g := CheckAttestations(design)
	if g.Counts["claims owed"] != 10 {
		t.Fatalf("machines must make the Gate 3 claims owed: %+v", g.Counts)
	}
	if !strings.Contains(strings.Join(g.Warns, "\n"), "g3.guard-semantics") {
		t.Fatalf("the unattested guard-semantics claim must warn: %v", g.Warns)
	}
}

// The vocabulary is the gate's own, exported for the CLI and the docs so it
// cannot be transcribed into a second, drifting list.
func TestAttestationVocabularyIsClosedAndOrdered(t *testing.T) {
	ids := AttestationClaimIDs()
	if len(ids) != len(attestVocabulary) || len(ids) == 0 {
		t.Fatalf("AttestationClaimIDs must mirror the vocabulary, got %d", len(ids))
	}
	seen := map[string]bool{}
	for i, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate claim id %q in the vocabulary", id)
		}
		seen[id] = true
		if id != attestVocabulary[i].id {
			t.Fatalf("order drifted at %d: %q vs %q", i, id, attestVocabulary[i].id)
		}
		if attestVocabulary[i].what == "" || attestVocabulary[i].owedBy == "" || attestVocabulary[i].owed == nil {
			t.Fatalf("claim %q must carry a description and an owed predicate", id)
		}
	}
	for _, want := range []string{"g2.nfr-content", "g3.guard-semantics", "gt.conformance-test-shape", "g4.zero-context", "ga.review-quality"} {
		if !seen[want] {
			t.Errorf("vocabulary omits %q, which SKILL.md attests", want)
		}
	}
}

// ContentHash is the single definition of the digest the schema records; the
// gate compares against it and `machinery attest` prints it.
func TestContentHashShapeAndStability(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	mustWrite(t, path, "hello\n")
	first, err := ContentHash(path)
	if err != nil {
		t.Fatal(err)
	}
	if !attestHashRe.MatchString(first) {
		t.Fatalf("ContentHash = %q, want sha256:<64 lower-case hex>", first)
	}
	second, err := ContentHash(path)
	if err != nil || second != first {
		t.Fatalf("ContentHash is not stable: %q then %q (%v)", first, second, err)
	}
	mustWrite(t, path, "hello world\n")
	third, err := ContentHash(path)
	if err != nil {
		t.Fatal(err)
	}
	if third == first {
		t.Fatal("ContentHash must change with the bytes")
	}
	if _, err := ContentHash(filepath.Join(dir, "absent.md")); err == nil {
		t.Fatal("ContentHash on an absent file must error")
	}
}

// Gv joins the vocabulary, the default list, and the artifact-activated set,
// so `machinery check` runs it exactly when the evidence exists.
func TestAttestationSuiteWiring(t *testing.T) {
	design, _ := attestFixture(t, attestEvidence(attestAllG2()...))
	if !KnownGate("gv") {
		t.Fatal("gv must be a known gate")
	}
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gv"] {
		t.Fatalf("the default suite must include gv: %v", sel.Run)
	}
	titles := func(d string) string {
		s, serr := Select(d, "", "")
		if serr != nil {
			t.Fatal(serr)
		}
		var out []string
		for _, g := range RunSelected(d, "", s, RunOptions{}) {
			out = append(out, g.Title)
		}
		return strings.Join(out, "\n")
	}
	bare := t.TempDir()
	if strings.Contains(titles(bare), "Gv-attest") {
		t.Error("Gv must not run on a design that committed no attestation evidence")
	}
	got := titles(design)
	if !strings.Contains(got, "Gv-attest") {
		t.Errorf("committed attestation evidence must activate Gv:\n%s", got)
	}
}

// A machine-less decomposed parent keeps every artifact-activated gate; Gv is
// one of them, and the narrowing note has to say so.
func TestAttestationSurvivesDecomposedParentNarrowing(t *testing.T) {
	design, _ := attestFixture(t, attestEvidence(attestAllG2()...))
	mustWrite(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gv"] {
		t.Fatalf("the narrowing dropped gv although the evidence exists: %v", sel.Run)
	}
	if !strings.Contains(sel.Note, "gv") {
		t.Fatalf("the narrowing note must list gv: %q", sel.Note)
	}
}
