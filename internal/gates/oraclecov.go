// Gt-tests: oracle coverage in the test suite. G3 proves the committed
// oracles match the machines and G4 proves the code respects the contract;
// neither proves the tests actually key on the oracle rows. Gt closes that
// gap: every stable id from the committed transition oracles (and from the
// formal decision oracles, when the design carries them) must appear
// whole-token in some test file, or the oracle must be parsed at runtime,
// which a test proves by naming the oracle file literally and then covers
// every row by construction (the conformance-parse idiom).

package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// formalOracleNames are the relational decision oracles Gt covers when they
// exist; Integrity.als generates no oracle, so it is not Gt's concern.
var formalOracleNames = []string{"Policy.oracle.md", "Isolation.oracle.md"}

// oracleTableIDs parses a committed oracle's decision table (the one with
// 'test id' and 'stable id' columns) and returns both columns as read from
// the file; the id shapes are never assumed.
func oracleTableIDs(text string) (testIDs, stableIDs []string) {
	for _, tbl := range ir.ParseMdTables(text) {
		ti := ir.FindCol(tbl.Header, "test id")
		si := ir.FindCol(tbl.Header, "stable id")
		if ti < 0 || si < 0 {
			continue
		}
		for _, r := range tbl.Rows {
			if ti < len(r) {
				if v := strings.TrimSpace(r[ti]); v != "" && v != "-" {
					testIDs = append(testIDs, v)
				}
			}
			if si < len(r) {
				if v := strings.TrimSpace(r[si]); v != "" && v != "-" {
					stableIDs = append(stableIDs, v)
				}
			}
		}
	}
	return testIDs, stableIDs
}

// CheckOracleCoverage implements Gt-tests.
func CheckOracleCoverage(design, impl string) *Gate {
	g := NewGate("Gt-tests  oracle ids in the test suite")
	g.startOrder()
	if fi, err := os.Stat(impl); err != nil || !fi.IsDir() {
		g.Errs = append(g.Errs, fmt.Sprintf("--impl %s is not a directory", ir.Repr(impl)))
		return g
	}
	corpus := testCorpus(design, impl, g)
	testFiles := g.Counts["test files scanned"]
	if testFiles == 0 {
		// the zero must stay visible: Count suppresses zeros in the checked line
		g.CheckedExtra("0 test files scanned")
	}

	mdir := filepath.Join(design, "machines")
	oraclePaths := sortedGlob(mdir, "*.oracle.md")
	machineFiles := sortedGlob(mdir, "*.machine.json")
	var formalPaths []string
	for _, name := range formalOracleNames {
		path := filepath.Join(design, "formal", name)
		if fi, err := os.Stat(path); err != nil || fi.IsDir() {
			continue // the relational layers are opt-in; Gp/Gn own their health
		}
		formalPaths = append(formalPaths, path)
	}
	if len(oraclePaths) == 0 {
		if len(machineFiles) > 0 {
			g.Errs = append(g.Errs, fmt.Sprintf("%d machine(s) under %s but no committed *.oracle.md; Gt has nothing to hold the tests to (run machinery oracle and commit the tables)", len(machineFiles), mdir))
		} else {
			// a machine-less design with an impl carries no transition-test
			// obligation; the zero must stay visible in every run
			g.CheckedExtra("0 machines")
		}
	} else {
		// once ANY oracle exists, a machine missing its own would otherwise
		// be invisible here: every machine needs its committed oracle
		for _, path := range machineFiles {
			base := filepath.Base(path)
			obase := filepath.Base(machineSibling(path, ".oracle.md"))
			if fi, err := os.Stat(machineSibling(path, ".oracle.md")); err != nil || fi.IsDir() {
				g.Errs = append(g.Errs, fmt.Sprintf("%s: no committed oracle (%s); run machinery oracle and commit the table so Gt can hold the tests to it", base, obase))
			}
		}
	}
	if testFiles == 0 && len(oraclePaths)+len(formalPaths) > 0 {
		// one loud error instead of per-machine missing-id errors whose
		// remedy ("key the tests on the ids") is impossible without tests
		g.Errs = append(g.Errs, fmt.Sprintf("no test files under %s; Gt has nothing to hold to the oracles (recognized test files: %s, any .rs under a tests/ or benches/ directory, or the #[cfg(test)] modules of any .rs file)", impl, strings.Join(testFilePatterns, ", ")))
	}
	for _, path := range oraclePaths {
		g.Count("machines")
		if testFiles == 0 {
			continue // the single no-test-files error above already blocks
		}
		base := filepath.Base(path)
		wholesale, _ := coverOracle(g, base, base, readFileOrErr(path, g), corpus)
		if wholesale {
			g.Count("machines covered by conformance parse")
		}
	}
	for _, path := range formalPaths {
		g.Count("formal oracles")
		if testFiles == 0 {
			continue // covered by the single no-test-files error
		}
		name := filepath.Base(path)
		if _, covered := coverOracle(g, "formal/"+name, name, readFileOrErr(path, g), corpus); covered {
			g.Count("formal oracles covered")
		}
	}
	return g
}

// coverOracle checks one committed oracle against the test corpus: covered
// wholesale when some test file earns the conformance-parse citation (see
// fileNameCited: the test then reads the committed table at runtime and
// covers every row by construction), otherwise row by row on the stable-id
// column with the whole-token id semantics. Returns whether the wholesale
// idiom applied and whether the oracle ended fully covered.
func coverOracle(g *Gate, label, base, text string, corpus testCorpusData) (wholesale, covered bool) {
	_, stableIDs := oracleTableIDs(text)
	g.Count("oracle rows", len(stableIDs))
	if len(stableIDs) == 0 {
		g.Errs = append(g.Errs, label+": committed oracle has no id rows (no 'test id'/'stable id' table); an empty oracle covers nothing")
		return false, false
	}
	if fileNameCited(base, corpus) {
		return true, true
	}
	var missing []string
	for _, id := range stableIDs {
		if idTokenIn(id, corpus.joined) {
			g.Count("ids covered by literal")
		} else {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		show := strings.Join(missing, ", ")
		if len(missing) > 10 {
			show = strings.Join(missing[:10], ", ") + fmt.Sprintf(" and %d more", len(missing)-10)
		}
		g.Errs = append(g.Errs, fmt.Sprintf("%s: %d of %d stable ids appear in no test file (%s); key the tests on the stable ids, or parse the committed table at runtime by naming %s in a test", label, len(missing), len(stableIDs), show, base))
		return false, false
	}
	return false, true
}

// testCorpusData is the scanned test suite: per-file texts (the wholesale
// citation is judged per file) plus the joined text for whole-token id
// lookups.
type testCorpusData struct {
	files  []corpusFile
	joined string
}

type corpusFile struct {
	rel  string
	text string
}

// testCorpus gathers the impl's test files, classified per language exactly
// as G4 classifies them to SKIP (one classifier, two gates), and honors the
// contract's ignore globs the same way G4 does when the contract loads; a
// missing or broken contract just means no ignore filtering here, because
// contract findings belong to G2/G4, not Gt. A production .rs file
// contributes ONLY its #[cfg(test)] spans: its production text (transition
// tables, constants) proves nothing about the tests (NG-7).
func testCorpus(design, impl string, g *Gate) testCorpusData {
	var ignore []string
	if c := loadContract(filepath.Join(design, "ARCHITECTURE.md"), NewGate("_")); c != nil {
		for _, ig := range objSlice(c.AsObject().Get2("ignore")) {
			ignore = append(ignore, ig.AsString())
		}
	}
	files, walkErr := walkSourceFiles(impl)
	if walkErr != nil {
		g.Errs = append(g.Errs, "walking "+impl+": "+walkErr.Error())
	}
	sort.Strings(files)
	var corpus testCorpusData
	var texts []string
	for _, path := range files {
		rel, _ := filepath.Rel(impl, path)
		ignored := false
		for _, ig := range ignore {
			if matchGlob(rel, ig) {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}
		text := readFileOrErr(path, g)
		switch {
		case isTestFile(rel):
			// the whole file is test text
		case strings.HasSuffix(rel, ".rs"):
			_, spans := rustSplitTests(text)
			if len(spans) == 0 {
				continue
			}
			text = strings.Join(spans, "\n")
		default:
			continue
		}
		g.Count("test files scanned")
		corpus.files = append(corpus.files, corpusFile{rel: rel, text: text})
		texts = append(texts, text)
	}
	corpus.joined = strings.Join(texts, "\n")
	return corpus
}

// fileNameCited reports whether some SINGLE test file cites base
// ("<Component>.oracle.md") as a real conformance-parse target. Three
// conditions, all in the same file (the Gt citation rule):
//  1. word boundaries on BOTH sides of the file name: an adjacent
//     [A-Za-z0-9_.-] byte disqualifies, so purchase-order.oracle.md never
//     covers order.oracle.md and order.oracle.md.bak covers nothing;
//  2. the mention lies inside a string literal on its line (an odd number of
//     the same quote character ' " ` before it, that character again after
//     it): a parser holds the path as a string, prose in a comment does not;
//  3. the file carries parse evidence: some string literal containing the |
//     character, the markdown table-row delimiter every conformance parser
//     splits on (go-crm's oracle_test.go and tenant_oracle_test.go satisfy
//     this via their row-splitting literals; a bare comment cannot).
func fileNameCited(base string, corpus testCorpusData) bool {
	for _, f := range corpus.files {
		if fileNameMentionedInString(base, f.text) && hasParseEvidence(f.text) {
			return true
		}
	}
	return false
}

// fileNameMentionedInString finds a whole-token, string-literal mention of
// base in text (conditions 1 and 2 of the citation rule).
func fileNameMentionedInString(base, text string) bool {
	idx := 0
	for {
		i := strings.Index(text[idx:], base)
		if i < 0 {
			return false
		}
		pos := idx + i
		idx = pos + 1
		if pos > 0 && isFileNameChar(text[pos-1]) {
			continue
		}
		if end := pos + len(base); end < len(text) && isFileNameChar(text[end]) {
			continue
		}
		if mentionInsideQuotes(text, pos, pos+len(base)) {
			return true
		}
	}
}

// mentionInsideQuotes reports whether text[start:end] lies inside a
// single-line string literal: on its line, an odd number of some quote
// character precedes start and that character appears again at or after end.
func mentionInsideQuotes(text string, start, end int) bool {
	ls := strings.LastIndexByte(text[:start], '\n') + 1
	le := len(text)
	if i := strings.IndexByte(text[end:], '\n'); i >= 0 {
		le = end + i
	}
	before, after := text[ls:start], text[end:le]
	for _, q := range []string{`"`, "'", "`"} {
		if strings.Count(before, q)%2 == 1 && strings.Contains(after, q) {
			return true
		}
	}
	return false
}

var parseEvidenceRes = []*regexp.Regexp{
	regexp.MustCompile(`"[^"\n]*\|[^"\n]*"`),
	regexp.MustCompile(`'[^'\n]*\|[^'\n]*'`),
	regexp.MustCompile("`[^`\n]*\\|[^`\n]*`"),
}

// hasParseEvidence reports whether text carries a string literal containing
// the markdown table-row delimiter (condition 3 of the citation rule).
func hasParseEvidence(text string) bool {
	for _, re := range parseEvidenceRes {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// isFileNameChar is the boundary class of the citation rule: characters that
// glue a mention into a longer file name.
func isFileNameChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') ||
		b == '_' || b == '.' || b == '-'
}
