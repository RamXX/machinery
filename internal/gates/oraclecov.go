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
	"go/ast"
	"go/parser"
	"go/token"
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
	oraclePaths, _ := strictSortedGlob(g, mdir, "*.oracle.md", "committed oracle")
	machineFiles, _ := strictSortedGlob(g, mdir, "*.machine.json", "machine source")
	machineStems := map[string]bool{}
	for _, path := range machineFiles {
		machineStems[strings.TrimSuffix(filepath.Base(path), ".machine.json")] = true
	}
	validOracles := oraclePaths[:0]
	for _, path := range oraclePaths {
		stem := strings.TrimSuffix(filepath.Base(path), ".oracle.md")
		if !machineStems[stem] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: orphan oracle has no corresponding %s.machine.json; it cannot establish test coverage", filepath.Base(path), stem))
			continue
		}
		validOracles = append(validOracles, path)
	}
	oraclePaths = validOracles
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
		wholesale, _ := coverOracle(g, base, base, readDesignFileOrErr(design, path, g), corpus)
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
		if _, covered := coverOracle(g, "formal/"+name, name, readDesignFileOrErr(design, path, g), corpus); covered {
			g.Count("formal oracles covered")
		}
	}
	// opt-in via CLAUSES{...} declarations: the falsifying-clause tests the
	// oracle cannot derive, held per governed row (see clausecov.go)
	if testFiles > 0 {
		checkClauseCoverage(g, design, corpus)
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
		if idTokenIn(id, corpus.joinedCode) {
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
	files      []corpusFile
	joined     string
	joinedCode string // comments removed; row ids must occur in test code/data
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
	if c := loadContract(design, filepath.Join(design, "ARCHITECTURE.md"), NewGate("_")); c != nil {
		for _, ig := range objSlice(c.AsObject().Get2("ignore")) {
			ignore = append(ignore, ig.AsString())
		}
	}
	files, _, walkWarns, walkErr := walkSourceFiles(impl, ignore)
	if walkErr != nil {
		g.Errs = append(g.Errs, "walking "+impl+": "+walkErr.Error())
	}
	for _, w := range walkWarns {
		g.Errs = append(g.Errs, "walk incomplete, subtree skipped: "+w)
	}
	sort.Strings(files)
	var corpus testCorpusData
	var texts, codeTexts []string
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
		codeTexts = append(codeTexts, executableTestText(text, rel))
	}
	corpus.joined = strings.Join(texts, "\n")
	corpus.joinedCode = strings.Join(codeTexts, "\n")
	return corpus
}

func executableTestText(text, rel string) string {
	clean := stripTestComments(text, filepath.Ext(rel))
	switch filepath.Ext(rel) {
	case ".go":
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, rel, clean, 0)
		if err != nil {
			return ""
		}
		var parts []string
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || (!strings.HasPrefix(fn.Name.Name, "Test") && !strings.HasPrefix(fn.Name.Name, "Benchmark") && !strings.HasPrefix(fn.Name.Name, "Fuzz") && !strings.HasPrefix(fn.Name.Name, "Example")) {
				continue
			}
			// The test declaration itself is executable test identity. This is
			// especially important for migration contracts, whose `tests:` rows
			// cite test function names rather than literals in their bodies.
			parts = append(parts, fn.Name.Name)
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.BasicLit:
					parts = append(parts, x.Value)
				case *ast.Ident:
					parts = append(parts, x.Name)
				}
				return true
			})
		}
		return strings.Join(parts, "\n")
	case ".rs":
		var parts []string
		for start := 0; start < len(clean); {
			i := strings.Index(clean[start:], "#[test]")
			if i < 0 {
				break
			}
			i += start
			end := rustItemEnd(clean, i+len("#[test]"))
			parts = append(parts, clean[i:end])
			start = end
		}
		return strings.Join(parts, "\n")
	default:
		return clean
	}
}

// stripTestComments is a length-preserving lexical pass for the comment
// forms used by supported test languages. Strings are retained because table
// driven tests legitimately store stable ids as data; comments are removed so
// listing ids in prose cannot manufacture coverage.
func stripTestComments(text, ext string) string {
	src := []byte(text)
	out := append([]byte(nil), src...)
	hashComments := ext == ".py" || ext == ".rb"
	lineComments := !hashComments
	quote := byte(0)
	escaped := false
	for i := 0; i < len(src); {
		if quote != 0 {
			if escaped {
				escaped = false
				i++
				continue
			}
			if src[i] == '\\' {
				escaped = true
				i++
				continue
			}
			if src[i] == quote {
				quote = 0
			}
			i++
			continue
		}
		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote = src[i]
			i++
			continue
		}
		if hashComments && src[i] == '#' {
			for i < len(src) && src[i] != '\n' {
				out[i] = ' '
				i++
			}
			continue
		}
		if lineComments && src[i] == '/' && i+1 < len(src) && src[i+1] == '/' {
			for i < len(src) && src[i] != '\n' {
				out[i] = ' '
				i++
			}
			continue
		}
		if lineComments && src[i] == '/' && i+1 < len(src) && src[i+1] == '*' {
			out[i], out[i+1] = ' ', ' '
			i += 2
			for i < len(src) {
				if src[i] == '*' && i+1 < len(src) && src[i+1] == '/' {
					out[i], out[i+1] = ' ', ' '
					i += 2
					break
				}
				if src[i] != '\n' {
					out[i] = ' '
				}
				i++
			}
			continue
		}
		i++
	}
	return string(out)
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
