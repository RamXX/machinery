// Gt-tests: oracle coverage in the test suite. G3 proves the committed
// oracles match the machines and G4 proves the code respects the contract;
// neither proves the tests actually key on the oracle rows. Gt closes that
// gap as a DISCOVERY gate: every stable id from the committed transition
// oracles (and from the formal decision oracles, when the design carries
// them) must appear whole-token in genuinely executable test code, or the
// oracle must be parsed by a test that actually runs: for Go, a valid test
// function whose bounded call graph opens the oracle file and drives its
// parsed content into testing assertions; for the script languages, a
// conformance-parse citation inside an extracted active test body. Gt never
// executes anything: static references prove discovery, not that assertions
// ran, and the checked line says so.

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
	"strconv"
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
	// Gt is static: it discovers references, it does not run the suite. The
	// disclosure keeps every green output honest about that limit.
	g.CheckedExtra("static discovery; tests not executed; unsupported parser structures remain uncovered")
	return g
}

// coverOracle checks one committed oracle against the test corpus: covered
// wholesale when some test file earns the conformance-parse credit (an
// active, executable parser; see fileNameCited), otherwise row by row on the
// stable-id column with the whole-token id semantics. Returns whether the
// wholesale idiom applied and whether the oracle ended fully covered.
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

// testCorpusData is the scanned test suite reduced to its EXECUTABLE part:
// per-file active test bodies (comment- and docstring-stripped; Go files
// contribute the literal/identifier extracts of their valid test functions,
// Rust files their #[test] item spans), plus the join of those bodies for
// whole-token id lookups. Comments, docstrings, build-disabled files and
// non-test declarations prove nothing.
type testCorpusData struct {
	files      []corpusFile
	joinedCode string // active test bodies only; row ids must occur there
}

type corpusFile struct {
	rel    string
	goFile *goTestFile // non-nil for parsed .go test files
	bodies []string    // active test bodies / Go extracts
}

// goBuildIgnoreLine matches the header lines that exclude a Go file from
// every build: the canonical //go:build ignore and the legacy // +build
// ignore. Opt-in build tags (ladybug, race, ...) select extra suites; the
// ignore constraint marks files the toolchain never runs.
var goBuildIgnoreLine = regexp.MustCompile(`(?m)^//go:build\s+ignore\s*$|^//\s*\+build\s+ignore\s*$`)

// testCorpus gathers the impl's test files, classified per language exactly
// as G4 classifies them to SKIP (one classifier, two gates), and honors the
// contract's ignore globs the same way G4 does when the contract loads; a
// missing or broken contract just means no ignore filtering here, because
// contract findings belong to G2/G4, not Gt. A production .rs file
// contributes ONLY its #[cfg(test)] spans (NG-7), and a Go file carrying the
// ignore build constraint is not corpus at all: its tests never run.
func testCorpus(design, impl string, g *Gate) testCorpusData {
	var ignore []string
	if c := loadContract(design, filepath.Join(design, "ARCHITECTURE.md"), NewGate("_")); c != nil {
		for _, ig := range objSlice(c.AsObject().Get2("ignore")) {
			ignore = append(ignore, ig.AsString())
		}
	}
	inventory, _, walkWarns, walkErr := walkSourceFilesBounded(impl, ignore, implementationDirectoryMaxEntries, implementationDirectoryMaxDepth)
	if walkErr != nil {
		g.Errs = append(g.Errs, "walking "+impl+": "+walkErr.Error())
	}
	if inventory != nil {
		defer func() {
			if closeErr := inventory.Close(); closeErr != nil {
				g.Errs = append(g.Errs, "walking "+impl+": source inventory changed before traversal completed: "+closeErr.Error())
			}
		}()
	}
	for _, w := range walkWarns {
		g.Errs = append(g.Errs, "walk incomplete, subtree skipped: "+w)
	}
	var files []string
	if inventory != nil {
		files = inventory.Files()
	}
	sort.Strings(files)
	var corpus testCorpusData
	var codeTexts []string
	for _, rel := range files {
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
		body, readErr := inventory.ReadFile(rel)
		if readErr != nil {
			g.Errs = append(g.Errs, filepath.Join(impl, rel)+" is unreadable: "+readErr.Error())
			continue
		}
		text := string(body)
		ext := filepath.Ext(rel)
		switch {
		case isTestFile(rel) && ext == ".go":
			if goBuildIgnoreLine.MatchString(goHeader(text)) {
				continue // excluded from every build; its tests never run
			}
			g.Count("test files scanned")
			cf := corpusFile{rel: rel, goFile: parseGoTestFile(text, rel)}
			if cf.goFile != nil {
				cf.bodies = cf.goFile.extracts()
			}
			corpus.files = append(corpus.files, cf)
		case isTestFile(rel):
			g.Count("test files scanned")
			clean := stripTestComments(stripDocstrings(text, ext), ext)
			corpus.files = append(corpus.files, corpusFile{rel: rel, bodies: activeTestBodies(clean, ext)})
		case strings.HasSuffix(rel, ".rs"):
			_, spans := rustSplitTests(text)
			if len(spans) == 0 {
				continue
			}
			g.Count("test files scanned")
			corpus.files = append(corpus.files, corpusFile{rel: rel, bodies: activeTestBodies(stripTestComments(strings.Join(spans, "\n"), ext), ext)})
		default:
			continue
		}
		codeTexts = append(codeTexts, corpus.files[len(corpus.files)-1].bodies...)
	}
	corpus.joinedCode = strings.Join(codeTexts, "\n")
	return corpus
}

// goHeader returns the text before the package clause, where build
// constraints live.
func goHeader(text string) string {
	if i := goPackageLine.FindStringIndex(text); i != nil {
		return text[:i[0]]
	}
	return text
}

var goPackageLine = regexp.MustCompile(`(?m)^package\s+\w+`)

// fileNameCited reports whether some SINGLE test file carries ACTIVE
// executable evidence that it parses the oracle named by base:
//   - a Go file: a valid test function whose bounded call graph reads the
//     oracle through a resolvable literal/const path and drives the parsed
//     content into testing assertions (goTestFile.coversOracle);
//   - a script-language file: an extracted active test body holding both a
//     whole-token string mention of base and a markdown table-row delimiter
//     literal, the fingerprint of a real row parser.
//
// Comments, docstrings, unused declarations, disabled files, uncalled
// helpers and mentions outside test bodies earn nothing. The evidence must
// all live in the same file (the Gt citation rule).
func fileNameCited(base string, corpus testCorpusData) bool {
	for _, f := range corpus.files {
		if f.goFile != nil {
			if f.goFile.coversOracle(base) {
				return true
			}
			continue
		}
		for _, b := range f.bodies {
			if fileNameMentionedInString(base, b) && hasParseEvidence(b) {
				return true
			}
		}
	}
	return false
}

// fileNameMentionedInString finds a whole-token, string-literal mention of
// base in text (the boundary and quote conditions of the citation rule).
func fileNameMentionedInString(base, text string) bool {
	idx := 0
	for {
		i := strings.Index(text[idx:], base)
		if i < 0 {
			return false
		}
		pos := idx + i
		idx = pos + 1
		if wholeTokenAt(text, pos, len(base)) && mentionInsideQuotes(text, pos, pos+len(base)) {
			return true
		}
	}
}

// wholeTokenAt reports whether text[pos:pos+n] has no gluing [A-Za-z0-9_.-]
// byte on either side.
func wholeTokenAt(text string, pos, n int) bool {
	if pos > 0 && isFileNameChar(text[pos-1]) {
		return false
	}
	if end := pos + n; end < len(text) && isFileNameChar(text[end]) {
		return false
	}
	return true
}

// wholeTokenIn reports whether base occurs in text as a whole token under
// the file-name boundary class.
func wholeTokenIn(base, text string) bool {
	idx := 0
	for {
		i := strings.Index(text[idx:], base)
		if i < 0 {
			return false
		}
		pos := idx + i
		if wholeTokenAt(text, pos, len(base)) {
			return true
		}
		idx = pos + 1
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
// the markdown table-row delimiter.
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

// stripDocstrings blanks documentation prose that quoting would otherwise
// preserve as data: Python module/def/class docstrings and Elixir
// @moduledoc/@doc strings. Blanking is length- and line-preserving.
func stripDocstrings(text, ext string) string {
	switch ext {
	case ".py", ".pyw":
		return stripPyDocstrings(text)
	case ".exs", ".ex":
		return stripExDocstrings(text)
	}
	return text
}

var pyDocOpen = regexp.MustCompile(`^[ \t]*[rRbBuUfF]{0,2}("""|''')`)

// stripPyDocstrings blanks the first statement of the module and of every
// def/class body when that statement is a triple-quoted string (the
// docstring positions of the language).
func stripPyDocstrings(text string) string {
	out := []byte(text)
	blank := func(from, to int) {
		for i := from; i < to && i < len(out); i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}
	type block struct {
		min, indent int
		first       bool
	}
	blocks := []block{{min: 0, indent: -1, first: true}}
	i := 0
	for i < len(text) {
		eol := strings.IndexByte(text[i:], '\n')
		if eol < 0 {
			eol = len(text) - i
		}
		line := text[i : i+eol]
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			i += eol + 1
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		for len(blocks) > 1 {
			top := blocks[len(blocks)-1]
			if top.indent >= 0 && indent <= top.indent || top.indent < 0 && indent < top.min {
				blocks = blocks[:len(blocks)-1]
				continue
			}
			break
		}
		b := &blocks[len(blocks)-1]
		if b.indent < 0 {
			b.indent = indent
		}
		if b.first {
			b.first = false
			if m := pyDocOpen.FindStringSubmatch(line); m != nil && len(m[1]) > 0 {
				quote := m[1]
				if strings.Count(line, quote) >= 2 {
					blank(i, i+eol) // opens and closes on one line
					i += eol + 1
					continue
				}
				j := i + eol + 1
				for {
					if j >= len(text) {
						blank(i, len(text))
						i = len(text)
						break
					}
					je := strings.IndexByte(text[j:], '\n')
					if je < 0 {
						je = len(text) - j
					}
					if strings.Contains(text[j:j+je], quote) {
						blank(i, j+je+1)
						i = j + je + 1
						break
					}
					j += je + 1
				}
				continue
			}
		}
		if strings.HasPrefix(t, "def ") || strings.HasPrefix(t, "class ") || strings.HasPrefix(t, "async def ") {
			blocks = append(blocks, block{min: indent + 1, indent: -1, first: true})
		}
		i += eol + 1
	}
	return string(out)
}

var exDocHeredoc = regexp.MustCompile(`^(\s*)@(moduledoc|doc)\s+("""|''')`)
var exDocSingle = regexp.MustCompile(`^(\s*)@(moduledoc|doc)\s+(?:"([^"\n]*)"|'([^'\n]*)')`)

// stripExDocstrings blanks @moduledoc/@doc string contents, heredocs
// included.
func stripExDocstrings(text string) string {
	out := []byte(text)
	for i := 0; i < len(text); {
		eol := strings.IndexByte(text[i:], '\n')
		if eol < 0 {
			eol = len(text) - i
		}
		line := text[i : i+eol]
		if m := exDocHeredoc.FindStringSubmatch(line); m != nil {
			quote := m[3]
			j := i + len(m[0])
			end := len(text)
			if k := strings.Index(text[j:], quote); k >= 0 {
				end = j + k + len(quote)
			}
			for p := i; p < end && p < len(out); p++ {
				if out[p] != '\n' {
					out[p] = ' '
				}
			}
			i = end
			continue
		}
		if loc := exDocSingle.FindStringSubmatchIndex(line); loc != nil && loc[3] > 0 {
			for p := loc[3]; p < loc[4] && p < len(out); p++ {
				if out[p] != '\n' {
					out[p] = ' '
				}
			}
		}
		i += eol + 1
	}
	return string(out)
}

// stripTestComments is a length-preserving lexical pass for the comment
// forms used by supported test languages. Strings are retained because table
// driven tests legitimately store stable ids as data; comments are removed
// so listing ids in prose cannot manufacture coverage. Hash-comment
// languages include Elixir; Ruby additionally folds =begin/=end blocks.
func stripTestComments(text, ext string) string {
	src := []byte(text)
	out := append([]byte(nil), src...)
	hashComments := ext == ".py" || ext == ".rb" || ext == ".ex" || ext == ".exs"
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
		if ext == ".rb" && strings.HasPrefix(string(src[i:]), "=begin") && (i == 0 || src[i-1] == '\n') {
			for i < len(src) {
				if strings.HasPrefix(string(src[i:]), "=end") && (i == 0 || src[i-1] == '\n') {
					for j := i; j < len(src) && src[j] != '\n'; j++ {
						out[j] = ' '
					}
					break
				}
				if src[i] != '\n' {
					out[i] = ' '
				}
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

// activeTestBodies extracts the executable test bodies of the script
// languages: test()/it() callback spans (JS/TS), def test_* blocks (Python),
// it/specify blocks (Ruby), test do blocks (Elixir), and #[test] item spans
// (Rust). Anything outside those bodies is suite scaffolding or production
// source and earns nothing.
func activeTestBodies(text, ext string) []string {
	switch ext {
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx":
		return jsTestBodies(text)
	case ".py", ".pyw":
		return pyTestBodies(text)
	case ".rb":
		return rbTestBodies(text)
	case ".exs", ".ex":
		return exTestBodies(text)
	case ".rs":
		return rustTestSpans(text)
	}
	return nil
}

var jsTestCall = regexp.MustCompile(`\b(?:test|it)(?:\.each)?\s*\(`)

// jsTestBodies captures each test()/it()/test.each() call expression
// (following chained call groups) and keeps, inside it, the callback body
// after the last top-level "=>", or the last top-level brace group, or the
// arguments themselves (data-driven forms like test.each([...]) carry the
// ids in the argument data).
func jsTestBodies(text string) []string {
	var bodies []string
	for _, m := range jsTestCall.FindAllStringIndex(text, -1) {
		i, depth, end := m[1]-1, 0, -1
		for i < len(text) {
			switch text[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					j := i + 1
					for j < len(text) && (text[j] == ' ' || text[j] == '\t') {
						j++
					}
					if j < len(text) && text[j] == '(' {
						i = j // chained call group: test.each(...)(...)
						continue
					}
					end = i
				}
			}
			if end >= 0 {
				break
			}
			i++
		}
		if end < 0 {
			continue
		}
		span := text[m[1]:end]
		if strings.TrimSpace(span) != "" {
			bodies = append(bodies, span)
		}
	}
	return bodies
}



var pyTestDef = regexp.MustCompile(`(?m)^([ \t]*)(?:async\s+)?def\s+test_\w*\s*\(`)

// pyTestBodies returns the indented bodies of def test_* functions.
func pyTestBodies(text string) []string {
	var bodies []string
	for _, m := range pyTestDef.FindAllStringSubmatchIndex(text, -1) {
		defIndent := m[3] - m[2] // group 1: the def line's leading whitespace
		rest := text[m[1]:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			rest = rest[nl+1:] // the body starts on the line after the def
		} else {
			rest = ""
		}
		var body strings.Builder
		for _, line := range strings.Split(rest, "\n") {
			if strings.TrimSpace(line) == "" {
				body.WriteString(line + "\n")
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if indent <= defIndent {
				break
			}
			body.WriteString(line + "\n")
		}
		if b := body.String(); strings.TrimSpace(b) != "" {
			bodies = append(bodies, b)
		}
	}
	return bodies
}

var rbItLine = regexp.MustCompile(`(?m)^[ \t]*(it|specify)\b`)
var rbDoOrBrace = regexp.MustCompile(`\bdo\b|\{`)
var rubyBlockOpen = regexp.MustCompile(`\b(?:do|def|class|module|begin|case)\b`)
var rubyLineBlockOpen = regexp.MustCompile(`(?:^|\n)[ \t]*(?:if|unless|while|until|for)\b`)
var rubyEndToken = regexp.MustCompile(`\bend\b`)

// rbTestBodies returns it/specify blocks in either do...end or brace form.
func rbTestBodies(text string) []string {
	var bodies []string
	for _, m := range rbItLine.FindAllStringIndex(text, -1) {
		rest := text[m[1]:]
		first := rbDoOrBrace.FindStringIndex(rest)
		if first == nil {
			continue
		}
		if rest[first[0]] == '{' {
			if end := matchBrace(rest, first[0]); end > first[0] {
				bodies = append(bodies, rest[first[0]+1:end])
			}
			continue
		}
		if end := matchRubyEnd(rest, first[0]); end > first[0] {
			bodies = append(bodies, rest[:end])
		}
	}
	return bodies
}

// matchBrace returns the offset of the brace matching the one at open.
func matchBrace(text string, open int) int {
	depth := 0
	for i := open; i < len(text); i++ {
		switch text[i] {
		case '"', '\'':
			q := text[i]
			for i++; i < len(text) && text[i] != q; i++ {
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// matchRubyEnd walks do...end block structure from the opening do at
// openDo and returns the offset just past the matching end. Line-start
// if/unless/while/until/for count as block openers; their modifier (trailing)
// forms do not.
func matchRubyEnd(text string, openDo int) int {
	type event struct {
		pos, delta int
	}
	var events []event
	collect := func(re *regexp.Regexp, delta int) {
		for _, m := range re.FindAllStringIndex(text, -1) {
			events = append(events, event{m[0], delta})
		}
	}
	collect(rubyBlockOpen, 1)
	collect(rubyLineBlockOpen, 1)
	collect(rubyEndToken, -1)
	sort.Slice(events, func(a, b int) bool { return events[a].pos < events[b].pos })
	depth := 0
	for _, ev := range events {
		if ev.pos < openDo {
			continue
		}
		depth += ev.delta
		if depth == 0 {
			return ev.pos + 3
		}
	}
	return -1
}

var exTestLine = regexp.MustCompile(`(?m)^[ \t]*test\s+`)
var exBlockToken = regexp.MustCompile(`\b(?:do|def|defp|defmodule|case|cond|if|unless|with|receive|fn|try|end)\b`)

// exTestBodies returns Elixir `test ... do ... end` blocks. One-line `do:`
// forms open no block and are not spans.
func exTestBodies(text string) []string {
	var bodies []string
	for _, m := range exTestLine.FindAllStringIndex(text, -1) {
		rest := text[m[1]:]
		toks := exBlockToken.FindAllStringIndex(rest, -1)
		depth, end := 0, -1
		for _, tk := range toks {
			word := rest[tk[0]:tk[1]]
			if word == "do" && tk[1] < len(rest) && rest[tk[1]] == ':' {
				continue // do: one-liner
			}
			if word == "end" {
				depth--
				if depth == 0 {
					end = tk[1]
					break
				}
				if depth < 0 {
					break
				}
			} else {
				depth++
			}
		}
		if end > 0 {
			bodies = append(bodies, rest[:end])
		}
	}
	return bodies
}

// rustTestSpans carves the #[test] item spans out of an already
// comment-stripped Rust test file: tests/ and benches/ files are wholly test
// code, but only the #[test] items run; helpers do not.
func rustTestSpans(text string) []string {
	var parts []string
	for start := 0; start < len(text); {
		i := strings.Index(text[start:], "#[test]")
		if i < 0 {
			break
		}
		i += start
		end := rustItemEnd(text, i+len("#[test]"))
		parts = append(parts, text[i:end])
		start = end
	}
	return parts
}

// executableTestText reduces one test file to its executable test text (the
// same reduction the corpus applies): for Go, the extracts of the valid test
// functions; for the script languages, the extracted active test bodies.
// Package-level declarations and non-test text are invisible.
func executableTestText(text, rel string) string {
	ext := filepath.Ext(rel)
	switch ext {
	case ".go":
		if goBuildIgnoreLine.MatchString(goHeader(text)) {
			return ""
		}
		gf := parseGoTestFile(text, rel)
		if gf == nil {
			return ""
		}
		return strings.Join(gf.extracts(), "\n")
	case ".rs":
		return strings.Join(rustTestSpans(stripTestComments(text, ext)), "\n")
	}
	return strings.Join(activeTestBodies(stripTestComments(stripDocstrings(text, ext), ext), ext), "\n")
}

// goTestFile is one parsed Go test file with its valid test functions, its
// package-level string constants (the only path names a parser may resolve
// through), and its plain package-level helpers.
type goTestFile struct {
	file   *ast.File
	consts map[string]string
	fns    map[string]*ast.FuncDecl // plain package-level funcs by name
	tests  []*ast.FuncDecl          // functions the go test harness runs
}

// parseGoTestFile parses text as Go; nil when it does not parse.
func parseGoTestFile(text, rel string) *goTestFile {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, text, 0)
	if err != nil {
		return nil
	}
	gf := &goTestFile{file: file, consts: map[string]string{}, fns: map[string]*ast.FuncDecl{}}
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.CONST {
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
					continue
				}
				if lit, ok := vs.Values[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if v, err := strconv.Unquote(lit.Value); err == nil {
						gf.consts[vs.Names[0].Name] = v
					}
				}
			}
		}
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil {
			continue
		}
		gf.fns[fd.Name.Name] = fd
		if validGoTestFunc(fd) {
			gf.tests = append(gf.tests, fd)
		}
	}
	return gf
}

// validGoTestFunc reports whether the go test harness would run fd: the
// Test/Benchmark/Fuzz/Example families with canonical names and signatures,
// no receivers, no TestMain (a harness hook, not a test), no Test<lowercase>
// names the runner skips.
func validGoTestFunc(fd *ast.FuncDecl) bool {
	if fd.Body == nil || fd.Recv != nil {
		return false
	}
	name := fd.Name.Name
	params := fd.Type.Params.List
	one := func(typ string) bool {
		if len(params) != 1 {
			return false
		}
		star, ok := params[0].Type.(*ast.StarExpr)
		if !ok {
			return false
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != typ {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		return ok && id.Name == "testing"
	}
	switch {
	case name == "TestMain":
		return false
	case strings.HasPrefix(name, "Test"):
		if len(name) > 4 {
			if c := name[4]; c >= 'a' && c <= 'z' {
				return false // Testhelper: never run by the harness
			}
		}
		return one("T")
	case strings.HasPrefix(name, "Benchmark"):
		return one("B")
	case strings.HasPrefix(name, "Fuzz"):
		return one("F")
	case strings.HasPrefix(name, "Example"):
		return len(params) == 0
	}
	return false
}

// extracts returns the executable test text of the file: per valid test
// function, its name plus every identifier and literal in its body. This is
// what whole-token id lookups see; production declarations are invisible.
func (gf *goTestFile) extracts() []string {
	var out []string
	for _, fd := range gf.tests {
		var parts []string
		parts = append(parts, fd.Name.Name)
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BasicLit:
				parts = append(parts, x.Value)
			case *ast.Ident:
				parts = append(parts, x.Name)
			}
			return true
		})
		out = append(out, strings.Join(parts, "\n"))
	}
	return out
}

// Analysis bounds: parser credit follows helpers at most maxHelperDepth call
// frames, analyses at most maxCallSites call sites in the test body, and
// treats ambiguity, recursion, shadowing and exhaustion as refusal (never as
// credit).
const (
	maxHelperDepth = 6
	maxCallSites   = 128
	taintRounds    = 4
)

// coversOracle reports whether any valid test function in the file carries
// a bounded, connected read-parse-assert flow over the oracle named by base.
func (gf *goTestFile) coversOracle(base string) bool {
	for _, fd := range gf.tests {
		if gf.testProvesParse(fd, base) {
			return true
		}
	}
	return false
}

// goEnv is the per-function taint environment: which identifiers carry
// oracle content, which struct fields of a tainted identifier were NOT
// derived from that content, which names are locally assigned (path
// resolution refuses locals), and the return-value summary of the walk.
type goEnv struct {
	tainted  map[string]bool
	excepts  map[string]map[string]bool
	locals   map[string]bool
	retCount int
	retDirty bool
	retEx    map[string]bool
}

// fnSummary is the interprocedural summary of one helper: whether its
// result safely carries oracle content (single return, no recursion) and
// which struct fields of that result were not derived from it.
type fnSummary struct {
	safe       bool
	retTainted bool
	exceptions map[string]bool
}

// goAnalysis is one bounded abstract interpretation of a test file against
// one oracle base.
type goAnalysis struct {
	gf        *goTestFile
	base      string
	reach     map[string]bool // helpers reachable within maxHelperDepth
	recursion map[string]bool // helpers inside a call cycle: unsafe
	summ      map[string]*fnSummary
}

// testProvesParse decides wholesale credit for one test function: (1) the
// test's bounded call graph reads a file whose resolvable path mentions the
// oracle base as a whole token, (2) the parsed content taints values, and
// (3) a testing assertion in the test body or an executed subtest closure is
// guarded by a condition the content influences.
func (gf *goTestFile) testProvesParse(test *ast.FuncDecl, base string) bool {
	calls := 0
	ast.Inspect(test.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.CallExpr); ok {
			calls++
		}
		return true
	})
	if calls > maxCallSites {
		return false
	}
	a := &goAnalysis{gf: gf, base: base, reach: map[string]bool{}, recursion: map[string]bool{}, summ: map[string]*fnSummary{}}
	a.mapReachable(test)
	for range taintRounds {
		stable := true
		for name, fd := range gf.fns {
			if !a.reach[name] {
				continue
			}
			env := a.walkFunc(fd)
			s := &fnSummary{safe: env.retCount <= 1 && !a.recursion[name], retTainted: env.retDirty, exceptions: env.retEx}
			if old, seen := a.summ[name]; !seen || old.safe != s.safe || old.retTainted != s.retTainted || !sameFields(old.exceptions, s.exceptions) {
				stable = false
			}
			a.summ[name] = s
		}
		if stable {
			break
		}
	}
	env := a.walkFunc(test)
	tainted := false
	for _, v := range env.tainted {
		tainted = tainted || v
	}
	if !tainted {
		return false
	}
	recv := test.Type.Params.List[0].Names[0].Name
	executed := executedClosures(test.Body)
	if a.guarded(test.Body, recv, env, nil, executed) {
		return true
	}
	for fl := range executed {
		if p := testingParam(fl); p != "" && fl.Body != nil && a.guarded(fl.Body, p, env, nil, executed) {
			return true
		}
	}
	return false
}

// mapReachable DFS-walks the plain-function call graph from the test body.
// Helpers deeper than maxHelperDepth and helpers inside call cycles are left
// out of reach (their results taint nothing: bounded analysis refuses).
func (a *goAnalysis) mapReachable(test *ast.FuncDecl) {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	state := map[string]int{}
	var visit func(fd *ast.FuncDecl, depth int)
	visit = func(fd *ast.FuncDecl, depth int) {
		name := fd.Name.Name
		if state[name] == grey {
			a.recursion[name] = true
			return
		}
		if state[name] == black {
			return
		}
		state[name] = grey
		if depth < maxHelperDepth {
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); ok {
					if callee, ok := a.gf.fns[id.Name]; ok {
						visit(callee, depth+1)
					}
				}
				return true
			})
		}
		state[name] = black
		if name != test.Name.Name {
			a.reach[name] = true
		}
	}
	visit(test, 0)
}

// walkFunc walks fd twice in source order (loop bodies settle in the second
// pass) propagating oracle-content taint and returns the final environment.
// Return-statistics count once per pass: a pass resets the counter first.
func (a *goAnalysis) walkFunc(fd *ast.FuncDecl) *goEnv {
	env := &goEnv{tainted: map[string]bool{}, excepts: map[string]map[string]bool{}, locals: map[string]bool{}}
	for range 2 {
		env.retCount = 0
		a.walkStmts(fd.Body.List, env)
	}
	return env
}

func (a *goAnalysis) walkStmts(list []ast.Stmt, env *goEnv) {
	for _, st := range list {
		a.walkStmt(st, env)
	}
}

// walkLoop walks a loop body twice so loop-carried taint settles, counting
// return statements once.
func (a *goAnalysis) walkLoop(body *ast.BlockStmt, env *goEnv) {
	if body == nil {
		return
	}
	rc := env.retCount
	a.walkStmts(body.List, env)
	env.retCount = rc
	a.walkStmts(body.List, env)
}

func (a *goAnalysis) walkStmt(st ast.Stmt, env *goEnv) {
	switch s := st.(type) {
	case *ast.AssignStmt:
		taints := make([]bool, len(s.Lhs))
		excepts := make([]map[string]bool, len(s.Lhs))
		for i, rhs := range s.Rhs {
			if i >= len(s.Lhs) {
				break // multi-value assign: only the first name can carry the value
			}
			taints[i], excepts[i] = a.eval(rhs, env)
		}
		for i, lhs := range s.Lhs {
			if id, ok := lhs.(*ast.Ident); ok {
				env.locals[id.Name] = true
				env.tainted[id.Name] = taints[i]
				if len(excepts[i]) > 0 {
					if env.excepts[id.Name] == nil {
						env.excepts[id.Name] = map[string]bool{}
					}
					for f := range excepts[i] {
						env.excepts[id.Name][f] = true
					}
				} else {
					delete(env.excepts, id.Name)
				}
			}
		}
	case *ast.DeclStmt:
		if gd, ok := s.Decl.(*ast.GenDecl); ok {
			for _, spec := range gd.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for i, name := range vs.Names {
						env.locals[name.Name] = true
						env.tainted[name.Name] = false
						if i < len(vs.Values) {
							env.tainted[name.Name], _ = a.eval(vs.Values[i], env)
						}
					}
				}
			}
		}
	case *ast.IfStmt:
		if s.Init != nil {
			a.walkStmt(s.Init, env)
		}
		a.walkStmts(s.Body.List, env)
		if s.Else != nil {
			a.walkStmt(s.Else, env)
		}
	case *ast.ForStmt:
		if s.Init != nil {
			a.walkStmt(s.Init, env)
		}
		a.walkLoop(s.Body, env)
	case *ast.RangeStmt:
		if s.X != nil {
			t, ex := a.eval(s.X, env)
			for _, id := range []*ast.Ident{identOf(s.Key), identOf(s.Value)} {
				if id == nil || id.Name == "_" {
					continue
				}
				env.locals[id.Name] = true
				env.tainted[id.Name] = t
				if len(ex) > 0 {
					env.excepts[id.Name] = ex
				} else {
					delete(env.excepts, id.Name)
				}
			}
		}
		a.walkLoop(s.Body, env)
	case *ast.SwitchStmt:
		a.walkStmts(s.Body.List, env)
	case *ast.TypeSwitchStmt:
		a.walkStmts(s.Body.List, env)
	case *ast.SelectStmt:
		a.walkStmts(s.Body.List, env)
	case *ast.BlockStmt:
		a.walkStmts(s.List, env)
	case *ast.ExprStmt:
		a.eval(s.X, env)
	case *ast.DeferStmt, *ast.GoStmt:
	case *ast.CaseClause:
		a.walkStmts(s.Body, env)
	case *ast.ReturnStmt:
		env.retCount++
		for _, r := range s.Results {
			t, ex := a.eval(r, env)
			if t {
				env.retDirty = true
				if env.retEx == nil {
					env.retEx = map[string]bool{}
				}
				for f := range ex {
					env.retEx[f] = true
				}
			}
		}
	case *ast.LabeledStmt:
		a.walkStmt(s.Stmt, env)
	case *ast.IncDecStmt:
	case *ast.SendStmt:
		a.eval(s.Chan, env)
	}
}

// eval computes whether expr carries oracle content and which struct fields
// of that value were not derived from it.
func (a *goAnalysis) eval(rhs ast.Expr, env *goEnv) (bool, map[string]bool) {
	switch e := rhs.(type) {
	case *ast.CallExpr:
		if a.isOracleRead(e, env) {
			return true, nil
		}
		if id, ok := e.Fun.(*ast.Ident); ok && id.Name == "append" && len(e.Args) > 0 {
			any, ex := false, map[string]bool{}
			for _, arg := range e.Args {
				t, ex1 := a.eval(arg, env)
				any = any || t
				for f := range ex1 {
					ex[f] = true
				}
			}
			return any, ex
		}
		if id, ok := e.Fun.(*ast.Ident); ok && a.gf.fns[id.Name] != nil {
			if s := a.summ[id.Name]; s != nil && s.safe && s.retTainted {
				return true, s.exceptions
			}
			return false, nil
		}
		for _, arg := range e.Args {
			if t, _ := a.eval(arg, env); t {
				return true, nil
			}
		}
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			// a method's receiver is an implicit argument: sc.Text() carries
			// the scanner's taint without any explicit argument
			if t, _ := a.eval(sel.X, env); t {
				return true, nil
			}
		}
		return false, nil
	case *ast.Ident:
		return env.tainted[e.Name], env.excepts[e.Name]
	case *ast.SelectorExpr:
		if id, ok := e.X.(*ast.Ident); ok {
			if env.excepts[id.Name][e.Sel.Name] {
				return false, nil
			}
			return env.tainted[id.Name], nil
		}
		return a.eval(e.X, env)
	case *ast.CompositeLit:
		any, ex := false, map[string]bool{}
		for _, elt := range e.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				t, _ := a.eval(kv.Value, env)
				any = any || t
				if !t {
					if id, ok := kv.Key.(*ast.Ident); ok {
						ex[id.Name] = true
					}
				}
				continue
			}
			t, _ := a.eval(elt, env)
			any = any || t
		}
		if !any {
			return false, nil
		}
		return true, ex
	case *ast.BinaryExpr:
		t1, _ := a.eval(e.X, env)
		t2, _ := a.eval(e.Y, env)
		return t1 || t2, nil
	case *ast.UnaryExpr:
		return a.eval(e.X, env)
	case *ast.ParenExpr:
		return a.eval(e.X, env)
	case *ast.StarExpr:
		return a.eval(e.X, env)
	case *ast.IndexExpr:
		return a.eval(e.X, env)
	case *ast.SliceExpr:
		return a.eval(e.X, env)
	case *ast.TypeAssertExpr:
		return a.eval(e.X, env)
	}
	return false, nil
}

// isOracleRead reports whether call is os.ReadFile/os.Open/os.OpenFile (or
// ioutil.ReadFile) whose first argument resolves, through literals and
// package constants only, to a path mentioning the oracle base whole-token.
func (a *goAnalysis) isOracleRead(call *ast.CallExpr, env *goEnv) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) == 0 {
		return false
	}
	pkg, _ := sel.X.(*ast.Ident)
	if pkg == nil {
		return false
	}
	fn := sel.Sel.Name
	if !((pkg.Name == "os" && (fn == "ReadFile" || fn == "Open" || fn == "OpenFile")) ||
		(pkg.Name == "ioutil" && fn == "ReadFile")) {
		return false
	}
	path := a.resolvePath(call.Args[0], env)
	return path != "" && wholeTokenIn(a.base, path)
}

// resolvePath resolves expr to a concrete string using only string literals,
// package-level constants, and pure path combinators; locals, parameters and
// anything mutable resolve to "" (refused).
func (a *goAnalysis) resolvePath(expr ast.Expr, env *goEnv) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			if v, err := strconv.Unquote(e.Value); err == nil {
				return v
			}
		}
	case *ast.Ident:
		if env.locals[e.Name] {
			return ""
		}
		return a.gf.consts[e.Name]
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return ""
		}
		x, y := a.resolvePath(e.X, env), a.resolvePath(e.Y, env)
		if x == "" || y == "" {
			return ""
		}
		return x + y
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "filepath" {
				switch sel.Sel.Name {
				case "Join":
					parts := make([]string, 0, len(e.Args))
					for _, arg := range e.Args {
						p := a.resolvePath(arg, env)
						if p == "" {
							return ""
						}
						parts = append(parts, p)
					}
					return strings.Join(parts, "/")
				case "FromSlash", "Clean", "Abs", "ToSlash":
					if len(e.Args) == 1 {
						return a.resolvePath(e.Args[0], env)
					}
				}
			}
		}
	case *ast.ParenExpr:
		return a.resolvePath(e.X, env)
	}
	return ""
}

// executedClosures returns the function literals that actually run: those
// passed directly as call arguments (t.Run callbacks) and those whose
// assigned identifier is called somewhere in the body. A closure merely
// defined (check := func(){...}; _ = check) never runs and proves nothing.
func executedClosures(body ast.Node) map[*ast.FuncLit]bool {
	run := map[*ast.FuncLit]bool{}
	called := map[string]bool{}
	closureOf := map[string]*ast.FuncLit{}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			for _, arg := range x.Args {
				if fl, ok := arg.(*ast.FuncLit); ok {
					run[fl] = true
				}
			}
			if id, ok := x.Fun.(*ast.Ident); ok {
				called[id.Name] = true
			}
		case *ast.AssignStmt:
			for i, rhs := range x.Rhs {
				if fl, ok := rhs.(*ast.FuncLit); ok && i < len(x.Lhs) {
					if id, ok := x.Lhs[i].(*ast.Ident); ok {
						closureOf[id.Name] = fl
					}
				}
			}
		}
		return true
	})
	for name, fl := range closureOf {
		if called[name] {
			run[fl] = true
		}
	}
	return run
}

// testingParam returns the name of a FuncLit's *testing.T parameter, or "".
func testingParam(fl *ast.FuncLit) string {
	for _, field := range fl.Type.Params.List {
		if star, ok := field.Type.(*ast.StarExpr); ok {
			if sel, ok := star.X.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "testing" && sel.Sel.Name == "T" && len(field.Names) > 0 {
					return field.Names[0].Name
				}
			}
		}
	}
	return ""
}

// guarded walks statements tracking the active if-conditions on the path; a
// testing-receiver assertion credits when some active condition carries a
// content-derived operand.
func (a *goAnalysis) guarded(n ast.Node, recv string, env *goEnv, conds []ast.Expr, executed map[*ast.FuncLit]bool) bool {
	switch x := n.(type) {
	case *ast.BlockStmt:
		for _, st := range x.List {
			if a.guarded(st, recv, env, conds, executed) {
				return true
			}
		}
	case *ast.IfStmt:
		next := append(conds[:len(conds):len(conds)], x.Cond)
		if a.guarded(x.Body, recv, env, next, executed) {
			return true
		}
		if x.Else != nil {
			return a.guarded(x.Else, recv, env, conds, executed)
		}
	case *ast.ForStmt:
		return x.Body != nil && a.guarded(x.Body, recv, env, conds, executed)
	case *ast.RangeStmt:
		return x.Body != nil && a.guarded(x.Body, recv, env, conds, executed)
	case *ast.SwitchStmt:
		return x.Body != nil && a.guarded(x.Body, recv, env, conds, executed)
	case *ast.TypeSwitchStmt:
		return x.Body != nil && a.guarded(x.Body, recv, env, conds, executed)
	case *ast.SelectStmt:
		return x.Body != nil && a.guarded(x.Body, recv, env, conds, executed)
	case *ast.ExprStmt:
		return a.guardedExpr(x.X, recv, env, conds)
	case *ast.AssignStmt:
		for _, rhs := range x.Rhs {
			if a.guardedExpr(rhs, recv, env, conds) {
				return true
			}
		}
	case *ast.CaseClause:
		for _, st := range x.Body {
			if a.guarded(st, recv, env, conds, executed) {
				return true
			}
		}
	}
	return false
}

func (a *goAnalysis) guardedExpr(e ast.Expr, recv string, env *goEnv, conds []ast.Expr) bool {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == recv {
			switch sel.Sel.Name {
			case "Error", "Errorf", "Fatal", "Fatalf", "Fail", "FailNow":
				return a.anyDerived(conds, env)
			}
		}
	}
	return false
}

// anyDerived reports whether the guard conditions carry oracle content: at
// least one content-derived operand, and no poisoned field access (a field
// of a tainted row value that was NOT derived from the oracle text).
func (a *goAnalysis) anyDerived(conds []ast.Expr, env *goEnv) bool {
	found, poison := false, false
	for _, c := range conds {
		ast.Inspect(c, func(n ast.Node) bool {
			e, ok := n.(ast.Expr)
			if !ok {
				return true
			}
			t, _ := a.eval(e, env)
			if t {
				found = true
				return true
			}
			if sel, ok := e.(*ast.SelectorExpr); ok {
				if xt, _ := a.eval(sel.X, env); xt {
					poison = true // tainted row read through an underived field
				}
			}
			return true
		})
	}
	return found && !poison
}

func sameFields(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func identOf(e ast.Expr) *ast.Ident {
	if e == nil {
		return nil
	}
	if id, ok := e.(*ast.Ident); ok {
		return id
	}
	return nil
}
