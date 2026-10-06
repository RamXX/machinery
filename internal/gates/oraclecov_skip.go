package gates

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func skippedOraclePaths(base, id string, corpus testCorpusData) []string {
	var paths []string
	for _, f := range corpus.skipped {
		if f.parsesOracle(base) || idTokenIn(id, strings.Join(f.bodies, "\n")) {
			paths = append(paths, filepath.ToSlash(f.rel))
		}
	}
	return paths
}

// splitSkippedTests preserves skipped bindings for diagnostics, separately
// from the executable corpus used by both Gt and OracleBindings.
func splitSkippedTests(cf corpusFile, text, ext string, excludes map[string]string) (active, skipped corpusFile) {
	active, skipped = cf, corpusFile{rel: cf.rel}
	if cf.goFile != nil {
		live, dead := *cf.goFile, *cf.goFile
		live.tests, dead.tests = nil, nil
		for _, fd := range cf.goFile.tests {
			if goTestSkips(fd) {
				dead.tests = append(dead.tests, fd)
			} else {
				live.tests = append(live.tests, fd)
			}
		}
		active.goFile, skipped.goFile = &live, &dead
		active.bodies, skipped.bodies = live.extracts(), dead.extracts()
		return active, skipped
	}
	text = stripTestComments(stripDocstrings(text, ext), ext)
	var starts [][]int
	var flags []bool
	switch ext {
	case ".ex", ".exs":
		starts = exTestLine.FindAllStringIndex(text, -1)
		flags = exSkippedTests(text, excludes)
	case ".py", ".pyw":
		starts = pyTestDef.FindAllStringIndex(text, -1)
		for _, m := range starts {
			flags = append(flags, pyTestSkipped(text, m[0]))
		}
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx":
		starts = jsAllTestCall.FindAllStringIndex(text, -1)
		for _, m := range starts {
			flags = append(flags, strings.Contains(text[m[0]:m[1]], ".skip") || strings.Contains(text[m[0]:m[1]], ".todo"))
		}
	default:
		return active, skipped
	}
	anySkipped := false
	for _, flag := range flags {
		anySkipped = anySkipped || flag
	}
	if !anySkipped {
		return active, skipped
	}
	active.bodies = nil
	for i, m := range starts {
		part := text[m[0]:]
		if i+1 < len(starts) {
			part = text[m[0]:starts[i+1][0]]
		}
		if jsExtension(ext) {
			prefix := m[1] - m[0]
			part = jsSkippedModifier.ReplaceAllString(part[:prefix], "$1") + part[prefix:]
		}
		bodies := activeTestBodies(part, ext)
		if flags[i] {
			skipped.bodies = append(skipped.bodies, bodies...)
		} else {
			active.bodies = append(active.bodies, bodies...)
		}
	}
	return active, skipped
}

func goTestSkips(fd *ast.FuncDecl) bool {
	if len(fd.Body.List) == 0 || len(fd.Type.Params.List) != 1 || len(fd.Type.Params.List[0].Names) != 1 {
		return false
	}
	stmt, ok := fd.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	recv, ok := sel.X.(*ast.Ident)
	return ok && recv.Name == fd.Type.Params.List[0].Names[0].Name && (sel.Sel.Name == "Skip" || sel.Sel.Name == "Skipf" || sel.Sel.Name == "SkipNow")
}

var (
	jsAllTestCall     = regexp.MustCompile(`\b(?:test|it)(?:\.(?:skip|todo))?(?:\.each)?\s*\(`)
	jsSkippedModifier = regexp.MustCompile(`\b(test|it)\.(?:skip|todo)\b`)
	pySkipMarker      = regexp.MustCompile(`\bpytest\.mark\.skip\b`)
	pySkipIfTrue      = regexp.MustCompile(`\bpytest\.mark\.skipif\s*\(\s*True\b`)
	pyModuleMark      = regexp.MustCompile(`(?m)^pytestmark\s*=([^\n]*)`)
	exTagLine         = regexp.MustCompile(`^\s*@(moduletag|describetag|tag)\s+(.+)$`)
	exKeywordTag      = regexp.MustCompile(`\b([a-zA-Z_]\w*):\s*(.*)`)
	exAtomTag         = regexp.MustCompile(`^:([a-zA-Z_]\w*)$`)
	exExclude         = regexp.MustCompile(`(?s)\bExUnit\.(?:start|configure)\s*\([^)]*?\bexclude:\s*\[([^]]*)\]`)
	exScopeToken      = regexp.MustCompile(`\b(?:do|fn|end)\b`)
)

func jsExtension(ext string) bool {
	switch ext {
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx":
		return true
	}
	return false
}

func pyTestSkipped(text string, start int) bool {
	marker := func(s string) bool { return pySkipMarker.MatchString(s) || pySkipIfTrue.MatchString(s) }
	for _, m := range pyModuleMark.FindAllStringSubmatch(text, -1) {
		if marker(m[1]) {
			return true
		}
	}
	prefix := strings.TrimRight(text[:start], " \t\n")
	for prefix != "" {
		i := strings.LastIndexByte(prefix, '\n')
		line := strings.TrimSpace(prefix[i+1:])
		if !strings.HasPrefix(line, "@") {
			break
		}
		if marker(line) {
			return true
		}
		if i < 0 {
			break
		}
		prefix = strings.TrimRight(prefix[:i], " \t\n")
	}
	return false
}

func exTags(text string) map[string]string {
	tags := map[string]string{}
	for _, part := range strings.Split(strings.Trim(text, " []"), ",") {
		part = strings.TrimSpace(part)
		if m := exAtomTag.FindStringSubmatch(part); m != nil {
			tags[m[1]] = "true"
		}
		if m := exKeywordTag.FindStringSubmatch(part); m != nil {
			v := strings.TrimSpace(m[2])
			if v == "not false" || v == "!false" {
				v = "true"
			}
			tags[m[1]] = v
		}
	}
	return tags
}

func exTagsSkipped(tags, excludes map[string]string) bool {
	for k, v := range tags {
		if (k == "skip" || k == "pending") && v != "false" && v != "nil" {
			return true
		}
		if excluded, ok := excludes[k]; ok && (excluded == "*" || excluded == v) {
			return true
		}
	}
	return false
}

// Block scopes prevent module and describe tags from leaking into siblings.
// Unknown skip expressions are conservatively UNRUN; Gt cannot evaluate an
// environment-dependent runner condition as proof that a test will run.
func exSkippedTests(text string, excludes map[string]string) []bool {
	stack := []map[string]string{{}}
	next := map[string]string{}
	type taggedTest struct {
		scopes []map[string]string
		tags   map[string]string
	}
	var tests []taggedTest
	for _, line := range strings.Split(text, "\n") {
		if m := exTagLine.FindStringSubmatch(line); m != nil {
			target := next
			if m[1] != "tag" {
				target = stack[len(stack)-1]
			}
			for k, v := range exTags(m[2]) {
				target[k] = v
			}
			continue
		}
		if exTestLine.MatchString(line) {
			tests = append(tests, taggedTest{scopes: append([]map[string]string(nil), stack...), tags: next})
			next = map[string]string{}
		}
		code := blankQuotedStrings(line)
		for _, m := range exScopeToken.FindAllStringIndex(code, -1) {
			word := code[m[0]:m[1]]
			if word == "do" && m[1] < len(code) && code[m[1]] == ':' {
				continue
			}
			if word == "end" {
				if len(stack) > 1 {
					stack = stack[:len(stack)-1]
				}
			} else {
				stack = append(stack, map[string]string{})
			}
		}
	}
	var flags []bool
	for _, test := range tests {
		tags := map[string]string{}
		for _, scope := range test.scopes {
			for k, v := range scope {
				tags[k] = v
			}
		}
		for k, v := range test.tags {
			tags[k] = v
		}
		flags = append(flags, exTagsSkipped(tags, excludes))
	}
	return flags
}

func blankQuotedStrings(text string) string {
	out := []byte(text)
	quote, escaped := byte(0), false
	for i, c := range out {
		if quote != 0 {
			out[i] = ' '
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
		} else if c == '"' || c == '\'' || c == '`' {
			quote = c
			out[i] = ' '
		}
	}
	return string(out)
}

func exUnitExclusions(files []string, inventory *sourceFileInventory, g *Gate) map[string]string {
	excludes := map[string]string{}
	for _, rel := range files {
		if filepath.Base(rel) != "test_helper.exs" {
			continue
		}
		data, err := inventory.ReadFile(rel)
		if err != nil {
			g.Errs = append(g.Errs, rel+" is unreadable: "+err.Error())
			continue
		}
		text := stripTestComments(stripDocstrings(string(data), ".exs"), ".exs")
		for _, m := range exExclude.FindAllStringSubmatch(text, -1) {
			for _, part := range strings.Split(m[1], ",") {
				if atom := exAtomTag.FindStringSubmatch(strings.TrimSpace(part)); atom != nil {
					excludes[atom[1]] = "*"
				} else {
					for k, v := range exTags(part) {
						excludes[k] = v
					}
				}
			}
		}
	}
	return excludes
}

type testRunRecord struct {
	Type    string            `json:"type"`
	Suite   string            `json:"suite"`
	SHA256  string            `json:"sha256"`
	Sources map[string]string `json:"sources"`
	Result  string            `json:"result"`
}

// Records attest a passed out-of-band run, not execution by this gate. Each
// row must match current bytes under the governed implementation root.
func recordedTestRuns(design string, inventory *sourceFileInventory, g *Gate) map[string]bool {
	valid := map[string]bool{}
	path := filepath.Join(design, "assurance", "test-runs.json")
	data, err := readDesignFile(design, path)
	if os.IsNotExist(err) {
		return valid
	}
	if err != nil {
		g.Errs = append(g.Errs, "test-runs.json: "+err.Error())
		return valid
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var rows []testRunRecord
	err = decoder.Decode(&rows)
	if err == nil {
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			err = fmt.Errorf("expected one JSON array")
		}
	}
	if err == nil && rows == nil {
		err = fmt.Errorf("expected a JSON array")
	}
	if err != nil {
		g.Errs = append(g.Errs, fmt.Sprintf("test-runs.json: invalid record array: %v", err))
		return valid
	}
	seen := map[string]bool{}
	for _, row := range rows {
		problem := ""
		if row.Type != "out-of-band" || row.Result != "passed" || row.Sources == nil {
			problem = "requires type out-of-band, result passed, and sources object"
		}
		if seen[row.Suite] {
			problem = "duplicate suite record"
		}
		seen[row.Suite] = true
		paths := []string{row.Suite}
		for source := range row.Sources {
			if source != row.Suite {
				paths = append(paths, source)
			} else {
				problem = "suite must not also appear in sources"
			}
		}
		sort.Strings(paths)
		for _, rel := range paths {
			if !filepath.IsLocal(rel) || filepath.ToSlash(filepath.Clean(rel)) != rel {
				problem = "paths must be clean implementation-relative paths"
				break
			}
			want := row.SHA256
			if rel != row.Suite {
				want = row.Sources[rel]
			}
			body, readErr := inventory.ReadFile(filepath.FromSlash(rel))
			if readErr != nil {
				problem = "stale record: " + rel + ": " + readErr.Error()
				break
			}
			if want != fmt.Sprintf("%x", sha256.Sum256(body)) {
				problem = "stale record: sha256 mismatch for " + rel
				break
			}
		}
		if problem != "" {
			delete(valid, row.Suite)
			g.Errs = append(g.Errs, fmt.Sprintf("test-runs.json: %s: %s", row.Suite, problem))
			continue
		}
		valid[row.Suite] = true
	}
	return valid
}
